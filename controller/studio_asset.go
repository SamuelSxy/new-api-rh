package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

func getArkAssetURIByStatus(status, assetID string) string {
	if status == "Active" && assetID != "" {
		return fmt.Sprintf("asset://%s", assetID)
	}
	return ""
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

// downloadAssetFromURL fetches a remote file via HTTP/HTTPS with SSRF protection
// and returns the file bytes, detected MIME type, and any error.
// maxSize is the maximum allowed file size in bytes.
func downloadAssetFromURL(urlStr string, maxSize int64) (data []byte, contentType string, err error) {
	// Validate the initial URL against SSRF — block private IPs, resolve domains to check.
	if err = common.ValidateURLWithFetchSetting(urlStr, true, false, false, false, nil, nil, nil, true); err != nil {
		return nil, "", fmt.Errorf("URL 安全校验失败: %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			// Re-validate each redirect target to prevent SSRF-via-redirect.
			if verr := common.ValidateURLWithFetchSetting(req.URL.String(), true, false, false, false, nil, nil, nil, true); verr != nil {
				return fmt.Errorf("重定向目标 URL 安全校验失败: %w", verr)
			}
			return nil
		},
	}

	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("远端服务器返回 HTTP %d", resp.StatusCode)
	}

	// Read body with size limit (+1 so we can detect exact-limit vs over-limit).
	lr := io.LimitReader(resp.Body, maxSize+1)
	buf, err := io.ReadAll(lr)
	if err != nil {
		return nil, "", fmt.Errorf("读取响应内容失败: %w", err)
	}
	if int64(len(buf)) > maxSize {
		return nil, "", fmt.Errorf("文件大小超过限制 (%d MB)", maxSize>>20)
	}

	// Determine content type: prefer response header, fall back to byte sniffing.
	ct := resp.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	if ct == "" || ct == "application/octet-stream" {
		ct = http.DetectContentType(buf)
		if idx := strings.Index(ct, ";"); idx != -1 {
			ct = strings.TrimSpace(ct[:idx])
		}
	}

	return buf, ct, nil
}

// UploadUserAsset handles multipart file upload or URL-based download, stores the
// asset in TOS or on local disk, and optionally registers it with the Ark Asset API.
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
	if err := c.Request.ParseMultipartForm(maxSize); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		common.ApiErrorMsg(c, "文件过大或请求格式错误")
		return
	}

	urlStr := strings.TrimSpace(c.PostForm("url"))
	fileHeader, fileErr := c.FormFile("file")

	// Resolve content source: file upload takes priority over URL download.
	var (
		contentType string
		fileSize    int64
		srcReader   io.Reader
		srcClose    func()
	)

	if fileErr == nil {
		// --- File upload path ---
		contentType = fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
		}
		fileSize = fileHeader.Size
		if name == "" {
			name = fileHeader.Filename
		}
		f, err := fileHeader.Open()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		srcReader = f
		srcClose = func() { f.Close() }
	} else if urlStr != "" {
		// --- URL download path ---
		buf, ct, err := downloadAssetFromURL(urlStr, maxSize)
		if err != nil {
			common.ApiErrorMsg(c, fmt.Sprintf("从 URL 下载失败: %v", err))
			return
		}
		contentType = ct
		fileSize = int64(len(buf))
		srcReader = bytes.NewReader(buf)
		if name == "" {
			if parsedURL, parseErr := url.Parse(urlStr); parseErr == nil {
				baseName := path.Base(parsedURL.Path)
				if baseName != "." && baseName != "/" && baseName != "" {
					name = baseName
				}
			}
			if name == "" {
				name = "asset-" + uuid.New().String()
			}
		}
	} else {
		common.ApiErrorMsg(c, "缺少 file 字段或 url 字段")
		return
	}

	if srcClose != nil {
		defer srcClose()
	}

	if assetType == "Image" && !allowedImageTypes[contentType] {
		common.ApiErrorMsg(c, "不支持的图片格式，请上传 JPEG/PNG/GIF/WebP/BMP/TIFF/HEIC/HEIF")
		return
	}
	if assetType == "Video" && !allowedVideoTypes[contentType] {
		common.ApiErrorMsg(c, "不支持的视频格式，请上传 MP4/MOV/WebM/AVI")
		return
	}

	ext := safeExtForMIME[contentType] // use server-controlled extension to prevent stored-XSS
	fileUUID := uuid.New().String()
	storedFileName := fileUUID + ext
	tosObjectKey := fmt.Sprintf("studio-assets/%d/%s", userId, storedFileName)

	var sourceURL string
	var cleanupFunc func()

	if system_setting.TosEnabled {
		tosURL, err := service.TosUploadFile(srcReader, tosObjectKey, contentType)
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

		dstPath := filepath.Join(userDir, storedFileName)
		dstFile, err := os.Create(dstPath)
		if err != nil {
			common.ApiError(c, fmt.Errorf("创建文件失败: %w", err))
			return
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcReader); err != nil {
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
	arkCredsConfigured := system_setting.ArkAssetAccessKey != "" && system_setting.ArkAssetSecretKey != ""

	var arkAssetId string
	var arkCallErr error
	if !arkAddrInvalid && arkCredsConfigured {
		// Build an absolute URL for Ark. sourceURL is usually relative, but can
		// already be absolute when TOS is enabled.
		arkDownloadURL := sourceURL
		if !strings.HasPrefix(sourceURL, "http://") && !strings.HasPrefix(sourceURL, "https://") {
			arkDownloadURL = serverAddr + sourceURL
		}
		arkAssetId, _, arkCallErr = service.ArkCreateAsset(
			name,
			arkDownloadURL,
			assetType,
			system_setting.ArkAssetProjectName,
			system_setting.ArkAssetGroupId,
		)
		if arkCallErr != nil {
			common.SysError(fmt.Sprintf("ark asset register failed: user_id=%d file=%s err=%v", userId, storedFileName, arkCallErr))
		}
	} else if arkAddrInvalid && arkCredsConfigured {
		common.SysError(fmt.Sprintf("ark asset registration skipped: ServerAddress is localhost/empty; configure a public ServerAddress to enable Ark asset registration (user_id=%d)", userId))
	}

	arkStatus := ""
	if arkAssetId != "" {
		arkStatus = "Processing"
	} else if arkAddrInvalid || !arkCredsConfigured {
		// Ark registration skipped: server address is invalid/localhost, or credentials not configured.
		arkStatus = "Skipped"
	} else if arkCallErr != nil {
		// Credentials configured and server address valid, but Ark API call failed.
		arkStatus = "Failed"
	}
	arkAssetUri := getArkAssetURIByStatus(arkStatus, arkAssetId)

	asset := &model.UserAsset{
		UserId:      userId,
		Name:        name,
		AssetType:   assetType,
		FileName:    storedFileName,
		FileSize:    fileSize,
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
	for _, asset := range list {
		asset.ArkAssetUri = getArkAssetURIByStatus(asset.ArkStatus, asset.ArkAssetId)
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
		if asset.ArkAssetUri != "" {
			if err := model.UpdateArkStatusAndURI(asset.Id, asset.ArkStatus, ""); err != nil {
				common.SysError(fmt.Sprintf("clear ark_asset_uri failed: id=%d err=%v", asset.Id, err))
			}
			asset.ArkAssetUri = ""
		}
		common.ApiSuccess(c, asset)
		return
	}

	// Already settled — skip the API call.
	if asset.ArkStatus == "Active" || asset.ArkStatus == "Failed" {
		expectedURI := getArkAssetURIByStatus(asset.ArkStatus, asset.ArkAssetId)
		if asset.ArkAssetUri != expectedURI {
			if err := model.UpdateArkStatusAndURI(asset.Id, asset.ArkStatus, expectedURI); err != nil {
				common.SysError(fmt.Sprintf("normalize ark fields failed: id=%d err=%v", asset.Id, err))
			}
			asset.ArkAssetUri = expectedURI
		}
		common.ApiSuccess(c, asset)
		return
	}

	status, err := service.ArkGetAssetStatus(asset.ArkAssetId, system_setting.ArkAssetProjectName)
	if err != nil {
		common.SysError(fmt.Sprintf("ark get asset status failed: user_id=%d asset_id=%s err=%v", userId, asset.ArkAssetId, err))
		common.ApiSuccess(c, asset)
		return
	}

	expectedURI := getArkAssetURIByStatus(status, asset.ArkAssetId)
	if status != asset.ArkStatus || asset.ArkAssetUri != expectedURI {
		if err := model.UpdateArkStatusAndURI(asset.Id, status, expectedURI); err != nil {
			common.SysError(fmt.Sprintf("update ark fields failed: id=%d err=%v", asset.Id, err))
		}
	}
	asset.ArkStatus = status
	asset.ArkAssetUri = expectedURI

	common.ApiSuccess(c, asset)
}
