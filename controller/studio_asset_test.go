package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type studioAssetAPIResp struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type studioAssetListResp struct {
	Items    []model.UserAsset `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

func setupStudioAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(&model.User{}, &model.UserAsset{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	user := &model.User{
		Id:          1,
		Username:    "asset_test_user",
		Password:    "asset_test_password",
		DisplayName: "Asset Test User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func decodeStudioAssetResp(t *testing.T, recorder *httptest.ResponseRecorder) studioAssetAPIResp {
	t.Helper()

	var resp studioAssetAPIResp
	if err := common.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v, body=%s", err, recorder.Body.String())
	}
	return resp
}

func TestStudioAssetHandlers_ListUploadDelete(t *testing.T) {
	setupStudioAssetTestDB(t)

	if err := os.MkdirAll(filepath.Join(assetBaseDir, "1"), 0755); err != nil {
		t.Fatalf("failed to create asset test dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(assetBaseDir, "1"))
	})

	router := gin.New()
	router.GET("/api/studio/assets", func(c *gin.Context) {
		c.Set("id", 1)
		ListUserAssets(c)
	})
	router.POST("/api/studio/assets", func(c *gin.Context) {
		c.Set("id", 1)
		UploadUserAsset(c)
	})
	router.DELETE("/api/studio/assets/:id", func(c *gin.Context) {
		c.Set("id", 1)
		DeleteUserAsset(c)
	})
	router.GET("/api/studio/assets/file/:userId/:filename", ServeUserAssetFile)

	// 1) list before upload
	listBeforeReq := httptest.NewRequest(http.MethodGet, "/api/studio/assets?page=1&page_size=20", nil)
	listBeforeRec := httptest.NewRecorder()
	router.ServeHTTP(listBeforeRec, listBeforeReq)
	if listBeforeRec.Code != http.StatusOK {
		t.Fatalf("list before upload status=%d body=%s", listBeforeRec.Code, listBeforeRec.Body.String())
	}
	listBeforeResp := decodeStudioAssetResp(t, listBeforeRec)
	if !listBeforeResp.Success {
		t.Fatalf("list before upload failed: %s", listBeforeResp.Message)
	}
	var listBeforeData studioAssetListResp
	if err := common.Unmarshal(listBeforeResp.Data, &listBeforeData); err != nil {
		t.Fatalf("failed to decode list before data: %v", err)
	}
	if listBeforeData.Total != 0 {
		t.Fatalf("expected total=0 before upload, got %d", listBeforeData.Total)
	}

	// 2) upload image asset
	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)
	if err := writer.WriteField("asset_type", "Image"); err != nil {
		t.Fatalf("failed to write asset_type: %v", err)
	}
	if err := writer.WriteField("name", "studio-asset-test"); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="tiny.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("failed to create file part: %v", err)
	}
	_, _ = io.Copy(part, bytes.NewReader([]byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x04, 0x00, 0x00, 0x00, 0xB5, 0x1C, 0x0C,
		0x02, 0x00, 0x00, 0x00, 0x0B, 0x49, 0x44, 0x41,
		0x54, 0x78, 0xDA, 0x63, 0xFC, 0xFF, 0x1F, 0x00,
		0x03, 0x03, 0x02, 0x00, 0xEE, 0xD9, 0x2B, 0xC7,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}))
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/studio/assets", &formBody)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	router.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploadRec.Code, uploadRec.Body.String())
	}
	uploadResp := decodeStudioAssetResp(t, uploadRec)
	if !uploadResp.Success {
		t.Fatalf("upload failed: %s", uploadResp.Message)
	}

	var uploaded model.UserAsset
	if err := common.Unmarshal(uploadResp.Data, &uploaded); err != nil {
		t.Fatalf("failed to decode upload data: %v", err)
	}
	if uploaded.Id <= 0 {
		t.Fatalf("invalid uploaded id: %d", uploaded.Id)
	}
	if uploaded.SourceUrl == "" || uploaded.FileName == "" {
		t.Fatalf("invalid uploaded payload: source_url=%q file_name=%q", uploaded.SourceUrl, uploaded.FileName)
	}

	// 3) uploaded file should be publicly accessible
	fileBeforeReq := httptest.NewRequest(http.MethodGet, uploaded.SourceUrl, nil)
	fileBeforeRec := httptest.NewRecorder()
	router.ServeHTTP(fileBeforeRec, fileBeforeReq)
	if fileBeforeRec.Code != http.StatusOK {
		t.Fatalf("expected file status 200 before delete, got %d", fileBeforeRec.Code)
	}

	// 4) list after upload should include one item
	listAfterUploadReq := httptest.NewRequest(http.MethodGet, "/api/studio/assets?page=1&page_size=20", nil)
	listAfterUploadRec := httptest.NewRecorder()
	router.ServeHTTP(listAfterUploadRec, listAfterUploadReq)
	if listAfterUploadRec.Code != http.StatusOK {
		t.Fatalf("list after upload status=%d body=%s", listAfterUploadRec.Code, listAfterUploadRec.Body.String())
	}
	listAfterUploadResp := decodeStudioAssetResp(t, listAfterUploadRec)
	if !listAfterUploadResp.Success {
		t.Fatalf("list after upload failed: %s", listAfterUploadResp.Message)
	}
	var listAfterUploadData studioAssetListResp
	if err := common.Unmarshal(listAfterUploadResp.Data, &listAfterUploadData); err != nil {
		t.Fatalf("failed to decode list after upload data: %v", err)
	}
	if listAfterUploadData.Total != 1 || len(listAfterUploadData.Items) != 1 {
		t.Fatalf("expected one item after upload, total=%d items=%d", listAfterUploadData.Total, len(listAfterUploadData.Items))
	}

	// 5) delete uploaded asset
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/studio/assets/"+strconv.Itoa(uploaded.Id), nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	deleteResp := decodeStudioAssetResp(t, deleteRec)
	if !deleteResp.Success {
		t.Fatalf("delete failed: %s", deleteResp.Message)
	}

	// 6) file endpoint should return 404 after delete
	fileAfterReq := httptest.NewRequest(http.MethodGet, uploaded.SourceUrl, nil)
	fileAfterRec := httptest.NewRecorder()
	router.ServeHTTP(fileAfterRec, fileAfterReq)
	if fileAfterRec.Code != http.StatusNotFound {
		t.Fatalf("expected file status 404 after delete, got %d", fileAfterRec.Code)
	}

	// 7) list after delete
	listAfterDeleteReq := httptest.NewRequest(http.MethodGet, "/api/studio/assets?page=1&page_size=20", nil)
	listAfterDeleteRec := httptest.NewRecorder()
	router.ServeHTTP(listAfterDeleteRec, listAfterDeleteReq)
	if listAfterDeleteRec.Code != http.StatusOK {
		t.Fatalf("list after delete status=%d body=%s", listAfterDeleteRec.Code, listAfterDeleteRec.Body.String())
	}
	listAfterDeleteResp := decodeStudioAssetResp(t, listAfterDeleteRec)
	if !listAfterDeleteResp.Success {
		t.Fatalf("list after delete failed: %s", listAfterDeleteResp.Message)
	}
	var listAfterDeleteData studioAssetListResp
	if err := common.Unmarshal(listAfterDeleteResp.Data, &listAfterDeleteData); err != nil {
		t.Fatalf("failed to decode list after delete data: %v", err)
	}
	if listAfterDeleteData.Total != 0 || len(listAfterDeleteData.Items) != 0 {
		t.Fatalf("expected zero items after delete, total=%d items=%d", listAfterDeleteData.Total, len(listAfterDeleteData.Items))
	}
}