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

	ext := filepath.Ext(fileHeader.Filename)
	fileUUID := uuid.New().String()
	storedFileName := fileUUID + ext

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

	// sourceURL is a relative path stored in DB and used for browser display.
	// It works regardless of how the server is accessed externally.
	sourceURL := fmt.Sprintf("/api/studio/assets/file/%d/%s", userId, storedFileName)

	// For Ark API calls, an absolute publicly-accessible URL is required.
	// The default ServerAddress is "http://localhost:3000" which Ark's external
	// servers cannot reach, so we treat localhost/127.x as "not configured".
	serverAddr := strings.TrimRight(system_setting.ServerAddress, "/")
	arkAddrInvalid := serverAddr == "" ||
		strings.HasPrefix(serverAddr, "http://localhost") ||
		strings.HasPrefix(serverAddr, "https://localhost") ||
		strings.HasPrefix(serverAddr, "http://127.") ||
		strings.HasPrefix(serverAddr, "https://127.")
	if system_setting.ArkAssetAccessKey != "" && system_setting.ArkAssetSecretKey != "" && arkAddrInvalid {
		common.ApiErrorMsg(c, "Ark 资源上传需要在系统设置中配置真实的公网 ServerAddress（当前为 localhost，Ark 外部服务器无法访问）")
		return
	}
	arkDownloadURL := serverAddr + sourceURL

	// Register with Ark Asset API (skipped silently if no credentials configured)
	arkAssetId, arkAssetUri, arkErr := service.ArkCreateAsset(
		name,
		arkDownloadURL,
		assetType,
		system_setting.ArkAssetProjectName,
		system_setting.ArkAssetGroupId,
	)
	if arkErr != nil {
		common.SysError(fmt.Sprintf("ark asset register failed: user_id=%d file=%s err=%v", userId, storedFileName, arkErr))
	}

	arkStatus := ""
	if arkAssetId != "" {
		arkStatus = "Processing"
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

	// Remove file from disk
	if asset.FileName != "" {
		filePath := filepath.Join(assetBaseDir, strconv.Itoa(userId), asset.FileName)
		_ = os.Remove(filePath)
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

	// Prevent path traversal
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
