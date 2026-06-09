package controller

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	assetBaseDir       = "./data/studio-assets"
	maxImageSize int64 = 30 << 20  // 30 MB (Ark requirement)
	maxVideoSize int64 = 500 << 20 // 500 MB
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"image/bmp":  true,
	"image/tiff": true,
	"image/heic": true,
	"image/heif": true,
}

var allowedVideoTypes = map[string]bool{
	"video/mp4":       true,
	"video/mpeg":      true,
	"video/quicktime": true,
	"video/webm":      true,
	"video/x-msvideo": true,
}

// safeExtForMIME maps each allowed MIME type to a fixed, server-controlled extension.
// This prevents attackers from storing files with dangerous extensions (e.g. .html)
// by supplying a spoofed Content-Type header, which would allow stored-XSS via
// ServeUserAssetFile when the browser renders the file with text/html.
var safeExtForMIME = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/bmp":       ".bmp",
	"image/tiff":      ".tiff",
	"image/heic":      ".heic",
	"image/heif":      ".heif",
	"video/mp4":       ".mp4",
	"video/mpeg":      ".mpeg",
	"video/quicktime": ".mov",
	"video/webm":      ".webm",
	"video/x-msvideo": ".avi",
}

// UploadUserAsset handles multipart file upload, stores the file and optionally
// registers it with the Ark Asset API.
func UploadUserAsset(c *gin.Context) {
	userId := c.GetInt("id")

	assetType := strings.TrimSpace(c.PostForm("asset_type")) // "Image" or "Video"
	if assetType != "Image" && assetType != "Video" {
		common.ApiErrorMsg(c, "asset_type 必须为 Image 或 Video")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))

	var maxSize int64
	if assetType == "Image" {
		maxSize = maxImageSize
	} else {
		maxSize = maxVideoSize
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	if err := c.Request.ParseMultipartForm(maxSize); err != nil {
		common.ApiErrorMsg(c, "文件过大或请求格式错误")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "缺少 file 字段")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
	}

	if assetType == "Image" && !allowedImageTypes[contentType] {
		common.ApiErrorMsg(c, "不支持的图片格式，请上传 JPEG/PNG/GIF/WebP/BMP/TIFF/HEIC/HEIF")
		return
	}
	if assetType == "Video" && !allowedVideoTypes[contentType] {
		common.ApiErrorMsg(c, "不支持的视频格式，请上传 MP4/MOV/WebM/AVI")
		return
	}

	if name == "" {
		name = fileHeader.Filename
	}

	ext := safeExtForMIME[contentType] // use server-controlled extension to prevent stored-XSS
	fileUUID := uuid.New().String()
	storedFileName := fileUUID + ext
	tosObjectKey := fmt.Sprintf("studio-assets/%d/%s", userId, storedFileName)

	var sourceURL string
	var cleanupFunc func()

	if system_setting.TosEnabled {
		src, err := fileHeader.Open()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		defer src.Close()

		tosURL, err := service.TosUploadFile(src, tosObjectKey, contentType)
		if err != nil {
			common.ApiError(c, fmt.Errorf("上传到 TOS 失败: %w", err))
			return
		}
		sourceURL = tosURL
		cleanupFunc = func() {
			_ = service.TosDeleteFile(tosObjectKey)
		}
	} else {
		userDir := filepath.Join(assetBaseDir, strconv.Itoa(userId))
		if err := os.MkdirAll(userDir, 0755); err != nil {
			common.ApiError(c, fmt.Errorf("创建目录失败: %w", err))
			return
		}

		src, err := fileHeader.Open()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		defer src.Close()

		dstPath := filepath.Join(userDir, storedFileName)
		dstFile, err := os.Create(dstPath)
		if err != nil {
			common.ApiError(c, fmt.Errorf("创建文件失败: %w", err))
			return
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, src); err != nil {
			common.ApiError(c, fmt.Errorf("写入文件失败: %w", err))
			return
		}

		sourceURL = fmt.Sprintf("/api/studio/assets/file/%d/%s", userId, storedFileName)
		cleanupFunc = func() {
			_ = os.Remove(dstPath)
		}
	}

	// For Ark API calls and direct external access (e.g. Seedance), an absolute
	// publicly-accessible URL is required. localhost/127.x addresses cannot be
	// reached by external services, so we skip Ark registration in that case but
	// always continue to save the file to the database.
	serverAddr := strings.TrimRight(system_setting.ServerAddress, "/")
	arkAddrInvalid := serverAddr == "" ||
		strings.HasPrefix(serverAddr, "http://localhost") ||
		strings.HasPrefix(serverAddr, "https://localhost") ||
		strings.HasPrefix(serverAddr, "http://127.") ||
		strings.HasPrefix(serverAddr, "https://127.")

	var arkAssetId, arkAssetUri string
	if !arkAddrInvalid {
		arkDownloadURL := sourceURL
		if !strings.HasPrefix(arkDownloadURL, "https://") && !strings.HasPrefix(arkDownloadURL, "http://") {
			arkDownloadURL = serverAddr + sourceURL
		}
		var arkErr error
		arkAssetId, arkAssetUri, arkErr = service.ArkCreateAsset(
			name,
			arkDownloadURL,
			assetType,
			system_setting.ArkAssetProjectName,
			system_setting.ArkAssetGroupId,
		)
		if arkErr != nil {
			// ArkCreateAsset already sets arkAssetUri = arkDownloadURL on error,
			// so external services can still attempt to use the public URL directly.
			common.SysError(fmt.Sprintf("ark asset register failed: user_id=%d file=%s err=%v", userId, storedFileName, arkErr))
		}
	} else if system_setting.ArkAssetAccessKey != "" && system_setting.ArkAssetSecretKey != "" {
		common.SysError(fmt.Sprintf("ark asset registration skipped: ServerAddress is localhost/empty; configure a public ServerAddress to enable Ark asset registration (user_id=%d)", userId))
	}

	arkStatus := ""
	if arkAssetId != "" {
		arkStatus = "Processing"
	} else if arkAddrInvalid {
		// Server address is localhost or not configured — Ark registration skipped.
		arkStatus = "Skipped"
	} else if arkAssetUri != sourceURL {
		// ArkCreateAsset returned the original URL on error, meaning registration failed.
		arkStatus = "Failed"
	}

	asset := &model.UserAsset{
		UserId:      userId,
		Name:        name,
		AssetType:   assetType,
		FileName:    storedFileName,
		FileSize:    fileHeader.Size,
		ContentType: contentType,
		SourceUrl:   sourceURL,
		ArkAssetId:  arkAssetId,
		ArkAssetUri: arkAssetUri,
		ArkStatus:   arkStatus,
	}
	if err := asset.Insert(); err != nil {
		if cleanupFunc != nil {
			cleanupFunc()
		}
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, asset)
}

// ListUserAssets returns the current user's asset list, optionally filtered by asset_type.
func ListUserAssets(c *gin.Context) {
	userId := c.GetInt("id")
	assetType := c.Query("asset_type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total, err := model.ListUserAssets(userId, assetType, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items":     list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// DeleteUserAsset deletes a user's asset by ID (both DB record and file on disk).
func DeleteUserAsset(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "无效的 asset ID")
		return
	}

	asset, err := model.GetUserAssetById(userId, id)
	if err != nil {
		common.ApiErrorMsg(c, "资源不存在")
		return
	}

	if asset.FileName != "" {
		if system_setting.TosEnabled {
			tosObjectKey := fmt.Sprintf("studio-assets/%d/%s", userId, asset.FileName)
			if err := service.TosDeleteFile(tosObjectKey); err != nil {
				common.SysError(fmt.Sprintf("tos delete file failed: user_id=%d file=%s err=%v", userId, asset.FileName, err))
			}
		} else {
			filePath := filepath.Join(assetBaseDir, strconv.Itoa(userId), asset.FileName)
			_ = os.Remove(filePath)
		}
	}

	if err := model.DeleteUserAsset(userId, id); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

// ServeUserAssetFile serves a stored asset file (public, no auth required).
// UUID-based filenames are non-guessable, providing sufficient security.
// Path: /api/studio/assets/file/:userId/:filename
func ServeUserAssetFile(c *gin.Context) {
	requestedUserId := c.Param("userId")
	filename := c.Param("filename")

	// Reject non-integer user IDs to prevent path traversal via the userId segment.
	if _, err := strconv.Atoi(requestedUserId); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// Prevent path traversal in the filename segment.
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		c.Status(http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(assetBaseDir, requestedUserId, filename)
	if _, err := os.Stat(filePath); err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	ext := filepath.Ext(filename)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "private, max-age=3600")
	c.File(filePath)
}

// SyncUserAssetArkStatus calls the Ark GetAsset API to refresh the asset's review
// status in the database and returns the latest status.
// GET /api/studio/assets/:id/ark-status
func SyncUserAssetArkStatus(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "无效的 asset ID")
		return
	}

	asset, err := model.GetUserAssetById(userId, id)
	if err != nil {
		common.ApiErrorMsg(c, "资源不存在")
		return
	}

	// If no Ark credentials or asset not registered, return current status as-is.
	if asset.ArkAssetId == "" {
		common.ApiSuccess(c, gin.H{"ark_status": asset.ArkStatus})
		return
	}

	// Already settled — skip the API call.
	if asset.ArkStatus == "Active" || asset.ArkStatus == "Failed" {
		common.ApiSuccess(c, gin.H{"ark_status": asset.ArkStatus})
		return
	}

	status, err := service.ArkGetAssetStatus(asset.ArkAssetId, system_setting.ArkAssetProjectName)
	if err != nil {
		common.SysError(fmt.Sprintf("ark get asset status failed: user_id=%d asset_id=%s err=%v", userId, asset.ArkAssetId, err))
		common.ApiSuccess(c, gin.H{"ark_status": asset.ArkStatus})
		return
	}

	if status != asset.ArkStatus {
		if err := model.UpdateArkStatus(asset.Id, status); err != nil {
			common.SysError(fmt.Sprintf("update ark_status failed: id=%d err=%v", asset.Id, err))
		}
	}

	common.ApiSuccess(c, gin.H{"ark_status": status})
}
