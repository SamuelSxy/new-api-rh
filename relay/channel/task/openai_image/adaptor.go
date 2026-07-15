package openai_image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TaskAdaptor 将 OpenAI /v1/images/generations 同步图像请求包装成任务流程，
// 用于像 gpt-image-* 这类同步返回但希望在「任务日志」中追踪结果的模型。
//
// 上游真正的请求仍然是同步的：BuildRequestBody 透传客户端原始 body，
// DoResponse 解析返回的 data[].url / b64_json，把图片落到 TOS（启用时）
// 并写入 info.TaskRelayInfo.CompletedResult。controller/relay.go 会据此
// 把 task 标记为 SUCCESS，写入 result_url。
//
// 当客户端在 JSON 形式的 /v1/images/generations 请求里携带参考图（image / images /
// metadata.imageUrls / data URI / 纯 base64）时，网关会自动 fetch URL 拼成
// multipart，并改走 /v1/images/edits，从而支持「URL 图生图」。
type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

// editImageMaxCount 与 OpenAI gpt-image-* 协议一致，image[] 最多 5 张。
const editImageMaxCount = 5

// editImageMetaKey 在 ValidateRequestAndSetAction 解码出参考图后，
// 用此 key 把图片字节挂到 TaskSubmitReq.Metadata 上，
// BuildRequestURL / BuildRequestBody 据此判断是否切到 /v1/images/edits。
const editImageMetaKey = "__openai_edit_images"

// editMaskMetaKey 暂存解析好的 mask（可选，单张）。
const editMaskMetaKey = "__openai_edit_mask"

// editContentTypeCtxKey 暂存 multipart Content-Type，BuildRequestBody 设置后由
// BuildRequestHeader 写到出站 req.Header 上（c.Request.Header 不会传递给上游 req）。
const editContentTypeCtxKey = "__openai_edit_content_type"

// editRequestMetaKey 缓存解析好的客户端原始 JSON，BuildRequestBody 写 multipart 时复用。
const editRequestMetaKey = "__openai_edit_request"

type editImage struct {
	Bytes    []byte
	MimeType string
	Filename string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var imgReq dto.ImageRequest
	if err := common.UnmarshalBodyReusable(c, &imgReq); err != nil {
		return service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(imgReq.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest)
	}

	rawImages := collectReferenceImageStrings(&imgReq)
	editImages, taskErr := fetchReferenceImages(rawImages)
	if taskErr != nil {
		return taskErr
	}

	maskImg, taskErr := decodeOptionalMask(imgReq.Mask)
	if taskErr != nil {
		return taskErr
	}

	taskReq := relaycommon.TaskSubmitReq{
		Model:  imgReq.Model,
		Prompt: imgReq.Prompt,
		Size:   imgReq.Size,
	}
	// 不管走 generations 透传还是 edits multipart，都需要 imgReq 来重新 marshal /
	// 平铺字段，以剥掉前端业务侧带过来的 metadata / imageUrls 等 OpenAI 不识别的扩展。
	taskReq.Metadata = map[string]interface{}{
		editRequestMetaKey: &imgReq,
	}
	if len(editImages) > 0 || maskImg != nil {
		// 只有 mask 没有参考图时，上游会报错；这里仍然走 edits 端点把请求转发过去，
		// 让上游返回标准错误，避免我们自己造一套不一致的语义。
		taskReq.Metadata[editImageMetaKey] = editImages
		if maskImg != nil {
			taskReq.Metadata[editMaskMetaKey] = maskImg
		}
		info.Action = constant.TaskActionImageEdit
	} else {
		info.Action = constant.TaskActionImageGenerate
	}

	c.Set("task_request", taskReq)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if hasEditImages(info) {
		// 强制走 edits 端点：客户端发的是 /v1/images/generations，但带了参考图，
		// 必须改写成 OpenAI 唯一支持 image 输入的 /v1/images/edits。
		// 借用 GetFullRequestURL 处理 Cloudflare Gateway 等特殊前缀。
		return relaycommon.GetFullRequestURL(a.baseURL, "/v1/images/edits", info.ChannelType), nil
	}
	return relaycommon.GetFullRequestURL(a.baseURL, info.RequestURLPath, info.ChannelType), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	if hasEditImages(info) {
		// multipart 路径：BuildRequestBody 已把分界 Content-Type 暂存到 ctx，
		// 必须显式写到出站 req 上（c.Request.Header 不会传过去）。
		if ct, ok := c.Get(editContentTypeCtxKey); ok {
			if s, ok2 := ct.(string); ok2 && s != "" {
				req.Header.Set("Content-Type", s)
			}
		}
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	editImages, mask, imgReq := getEditPayload(c)
	if len(editImages) > 0 || mask != nil {
		body, contentType, err := buildEditMultipartBody(imgReq, editImages, mask)
		if err != nil {
			return nil, err
		}
		// 仅暂存到 ctx；c.Request.Header 修改无效，BuildRequestHeader 会读出来写到出站 req。
		c.Set(editContentTypeCtxKey, contentType)
		return body, nil
	}

	// 走 /v1/images/generations 透传：不能直接转发原始 body，因为前端创意工作台
	// 可能带了 metadata / imageUrls 等 OpenAI 不识别的扩展字段。
	// 用 ImageRequest.MarshalJSON 重新序列化（Extra 字段会被自动剥掉）。
	if imgReq != nil {
		data, err := common.Marshal(imgReq)
		if err != nil {
			return nil, fmt.Errorf("marshal image request failed: %w", err)
		}
		return bytes.NewReader(data), nil
	}

	// 兜底：仍透传原始体（理论不会走到，imgReq 在 Validate 阶段必然非空）。
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, fmt.Errorf("read request body failed: %w", err)
	}
	return storage, nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var imageResp dto.ImageResponse
	if err := common.Unmarshal(responseBody, &imageResp); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
	}

	if len(imageResp.Data) == 0 {
		snippet := string(responseBody)
		if len(snippet) > 1024 {
			snippet = snippet[:1024] + "...(truncated)"
		}
		common.SysLog(fmt.Sprintf("openai_image: empty data array, body=%s", snippet))
		return "", nil, service.TaskErrorWrapper(
			fmt.Errorf("no image in response: %s", snippet),
			"no_image_in_response",
			http.StatusBadGateway,
		)
	}

	// 遍历所有图片：b64_json 上传到 TOS，避免大体积 base64 进数据库。
	// 上游已经是 https URL 的就保持不变。TOS 失败则回退到 data URI。
	for i := range imageResp.Data {
		d := &imageResp.Data[i]
		if d.Url != "" {
			continue
		}
		if d.B64Json == "" {
			continue
		}
		if uploaded, tosErr := uploadBase64ToTos(d.B64Json); tosErr == nil {
			d.Url = uploaded
			d.B64Json = ""
		} else {
			if system_setting.TosEnabled {
				common.SysLog(fmt.Sprintf("openai_image: TOS upload failed, fallback to data URI: %v", tosErr))
			}
			// 回退：保留 b64_json，并把 Url 设成 data URI，方便前端直接渲染。
			d.Url = "data:image/png;base64," + d.B64Json
		}
	}

	resultURL := ""
	for _, d := range imageResp.Data {
		if d.Url != "" {
			resultURL = d.Url
			break
		}
	}
	if resultURL == "" {
		return "", nil, service.TaskErrorWrapper(
			fmt.Errorf("no image url after processing"),
			"no_image_in_response",
			http.StatusBadGateway,
		)
	}

	info.TaskRelayInfo.CompletedResult = &relaycommon.SyncTaskResult{
		ResultURL: resultURL,
	}

	if imageResp.Created == 0 {
		imageResp.Created = time.Now().Unix()
	}

	// tiered_expr 计费：用上游返回的真实 token 重算 quota，覆盖预扣值，
	// RelayTaskSubmit 后续 SettleBilling 会基于此做差额结算。
	// 上游 usage 用 input_tokens / output_tokens 风格（Responses API），
	// 需归一化到 PromptTokens / CompletionTokens 以匹配 BuildTieredTokenParams。
	if imageResp.Usage != nil && info.TieredBillingSnapshot != nil {
		usage := normalizeImageUsage(imageResp.Usage)
		usedVars := billingexpr.UsedVars(info.TieredBillingSnapshot.ExprString)
		if ok, actualQuota, _ := service.TryTieredSettle(info, service.BuildTieredTokenParams(usage, false, usedVars)); ok && actualQuota > 0 {
			info.PriceData.Quota = actualQuota
		}
	}

	c.JSON(http.StatusOK, imageResp)
	return info.PublicTaskID, responseBody, nil
}

// normalizeImageUsage 把 OpenAI Responses 风格的 input_tokens / output_tokens
// 归一化到 BuildTieredTokenParams 读取的 PromptTokens / CompletionTokens。
// gpt-image-* 上游返回的就是这种格式。
func normalizeImageUsage(u *dto.Usage) *dto.Usage {
	normalized := *u
	if normalized.PromptTokens == 0 && normalized.InputTokens > 0 {
		normalized.PromptTokens = normalized.InputTokens
	}
	if normalized.CompletionTokens == 0 && normalized.OutputTokens > 0 {
		normalized.CompletionTokens = normalized.OutputTokens
	}
	if normalized.PromptTokensDetails.ImageTokens == 0 && normalized.InputTokensDetails != nil {
		normalized.PromptTokensDetails.ImageTokens = normalized.InputTokensDetails.ImageTokens
		normalized.PromptTokensDetails.TextTokens = normalized.InputTokensDetails.TextTokens
		normalized.PromptTokensDetails.CachedTokens = normalized.InputTokensDetails.CachedTokens
	}
	if normalized.TotalTokens == 0 {
		normalized.TotalTokens = normalized.PromptTokens + normalized.CompletionTokens
	}
	return &normalized
}

// uploadBase64ToTos 将 base64 编码的 PNG 图片上传到 TOS，未启用时返回 error。
// OpenAI gpt-image-* 的 b64_json 字段固定为 PNG。
func uploadBase64ToTos(b64Data string) (string, error) {
	if !system_setting.TosEnabled {
		return "", fmt.Errorf("tos not enabled")
	}
	imgBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		imgBytes, err = base64.RawStdEncoding.DecodeString(b64Data)
		if err != nil {
			return "", fmt.Errorf("base64 decode: %w", err)
		}
	}
	objectKey := fmt.Sprintf("openai-image/%s.png", uuid.New().String())
	return service.TosUploadFile(bytes.NewReader(imgBytes), objectKey, "image/png")
}

func (a *TaskAdaptor) GetModelList() []string { return []string{} }
func (a *TaskAdaptor) GetChannelName() string { return "openai_image" }

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	return nil, fmt.Errorf("FetchTask not supported: openai_image tasks complete synchronously")
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{Status: model.TaskStatusSuccess, Progress: "100%"}, nil
}

// ============================
// 参考图收集与抓取
// ============================

// collectReferenceImageStrings 从客户端请求里收集所有可能的参考图字符串：
//   - image: string 或 []string
//   - images: string 或 []string
//   - metadata.imageUrls: []string（与 gemini_image 对齐，创意控制台常用）
//
// 不在此处 fetch，仅做去重与基础清洗。
func collectReferenceImageStrings(req *dto.ImageRequest) []string {
	var out []string
	seen := make(map[string]struct{})

	appendStr := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	extractFromRaw := func(raw []byte) {
		if len(raw) == 0 {
			return
		}
		var v interface{}
		if err := common.Unmarshal(raw, &v); err != nil {
			return
		}
		switch vv := v.(type) {
		case string:
			appendStr(vv)
		case []interface{}:
			for _, it := range vv {
				if s, ok := it.(string); ok {
					appendStr(s)
				}
			}
		}
	}

	extractFromRaw(req.Image)
	extractFromRaw(req.Images)

	if mRaw, ok := req.Extra["metadata"]; ok {
		var meta map[string]interface{}
		if err := common.Unmarshal(mRaw, &meta); err == nil {
			if v, ok := meta["imageUrls"]; ok {
				if arr, ok := v.([]interface{}); ok {
					for _, it := range arr {
						if s, ok := it.(string); ok {
							appendStr(s)
						}
					}
				}
			}
		}
	}

	return out
}

// fetchReferenceImages 把字符串形式的图片来源解析成二进制：
//   - http/https URL: 通过 service.GetImageFromUrl 下载
//   - data URI: 解码 base64
//   - 其他: 尝试当作纯 base64（PNG）
//
// 任一图片失败立即返回 502，与 gemini_image 行为对齐。
// 数量超过 editImageMaxCount 时拒绝（避免上游 4xx）。
func fetchReferenceImages(rawImages []string) ([]editImage, *dto.TaskError) {
	if len(rawImages) == 0 {
		return nil, nil
	}
	if len(rawImages) > editImageMaxCount {
		return nil, service.TaskErrorWrapperLocal(
			fmt.Errorf("too many reference images: %d (max %d)", len(rawImages), editImageMaxCount),
			"invalid_request",
			http.StatusBadRequest,
		)
	}

	images := make([]editImage, 0, len(rawImages))
	for i, s := range rawImages {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		img, err := decodeReferenceImage(s)
		if err != nil {
			common.SysLog(fmt.Sprintf("openai_image: fetch reference image failed idx=%d err=%v", i, err))
			return nil, service.TaskErrorWrapper(err, "fetch_reference_image_failed", http.StatusBadGateway)
		}
		img.Filename = fmt.Sprintf("image_%d%s", i, mimeToExt(img.MimeType))
		images = append(images, img)
	}
	return images, nil
}

func decodeReferenceImage(s string) (editImage, error) {
	if strings.HasPrefix(s, "data:") {
		rest := strings.TrimPrefix(s, "data:")
		idx := strings.Index(rest, ",")
		if idx < 0 {
			return editImage{}, fmt.Errorf("invalid data URI")
		}
		meta := rest[:idx]
		b64 := rest[idx+1:]
		mime := "image/png"
		if i := strings.Index(meta, ";"); i >= 0 {
			mime = meta[:i]
		} else if meta != "" {
			mime = meta
		}
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(b64)
			if err != nil {
				return editImage{}, fmt.Errorf("base64 decode failed: %w", err)
			}
		}
		return editImage{Bytes: raw, MimeType: mime}, nil
	}

	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		mime, b64, err := service.GetImageFromUrl(s)
		if err != nil {
			return editImage{}, err
		}
		raw, decErr := base64.StdEncoding.DecodeString(b64)
		if decErr != nil {
			return editImage{}, fmt.Errorf("decode downloaded image: %w", decErr)
		}
		if mime == "" {
			mime = http.DetectContentType(raw)
		}
		return editImage{Bytes: raw, MimeType: mime}, nil
	}

	// 含 :// 但 scheme 不是 http/https：基本是 URL 写错了（如 "ttps://..."），
	// 给出明确错误，避免后续 base64 解码失败造成误导。
	if strings.Contains(s, "://") {
		return editImage{}, fmt.Errorf("unsupported url scheme, only http/https are accepted: %q", truncateForError(s))
	}

	// 视为已 base64 编码的纯数据，按 PNG 处理。
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return editImage{}, fmt.Errorf("base64 decode failed: %w", err)
	}
	mime := http.DetectContentType(raw)
	if !strings.HasPrefix(mime, "image/") {
		mime = "image/png"
	}
	return editImage{Bytes: raw, MimeType: mime}, nil
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	return ".png"
}

// ============================
// 上下文工具与 multipart 构造
// ============================

func hasEditImages(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	// info 上没存图片，BuildRequestURL 阶段也拿不到 c。
	// 用 ValidateRequestAndSetAction 设置好的 Action 作为判定信号。
	return info.Action == constant.TaskActionImageEdit
}

func getEditPayload(c *gin.Context) ([]editImage, *editImage, *dto.ImageRequest) {
	v, ok := c.Get("task_request")
	if !ok {
		return nil, nil, nil
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, nil, nil
	}
	if req.Metadata == nil {
		return nil, nil, nil
	}
	imgs, _ := req.Metadata[editImageMetaKey].([]editImage)
	imgReq, _ := req.Metadata[editRequestMetaKey].(*dto.ImageRequest)
	mask, _ := req.Metadata[editMaskMetaKey].(*editImage)
	return imgs, mask, imgReq
}

// decodeOptionalMask 解析可选的 mask 字段。
// 支持 string（URL / data URI / base64），或 {"data": "..."} 形式。空值返回 nil, nil。
func decodeOptionalMask(raw []byte) (*editImage, *dto.TaskError) {
	if len(raw) == 0 {
		return nil, nil
	}
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == `""` {
		return nil, nil
	}

	var maskStr string
	if err := common.Unmarshal(raw, &maskStr); err != nil {
		// 非 string 形式不支持（OpenAI mask 本来就是单个文件，不会是数组）
		return nil, service.TaskErrorWrapperLocal(
			fmt.Errorf("mask must be a string (url / data URI / base64)"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}
	maskStr = strings.TrimSpace(maskStr)
	if maskStr == "" {
		return nil, nil
	}

	img, err := decodeReferenceImage(maskStr)
	if err != nil {
		common.SysLog(fmt.Sprintf("openai_image: fetch mask failed err=%v", err))
		return nil, service.TaskErrorWrapper(err, "fetch_mask_failed", http.StatusBadGateway)
	}
	img.Filename = "mask" + mimeToExt(img.MimeType)
	return &img, nil
}

// buildEditMultipartBody 把 ImageRequest 的 JSON 字段平铺成 multipart 表单，
// 并把参考图按 image[] 形式写入；mask 不为 nil 时以单字段 mask 写入。
// 返回 body 和 Content-Type。
func buildEditMultipartBody(imgReq *dto.ImageRequest, images []editImage, mask *editImage) (io.Reader, string, error) {
	if imgReq == nil {
		return nil, "", fmt.Errorf("image request payload missing")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writeField := func(name, value string) {
		if value == "" {
			return
		}
		_ = writer.WriteField(name, value)
	}
	writeJSONField := func(name string, raw []byte) {
		// raw 是 json.RawMessage。string / number / bool 去掉引号；
		// 其它复杂结构按 JSON 字符串原样写入。
		if len(raw) == 0 {
			return
		}
		s := strings.TrimSpace(string(raw))
		if s == "" || s == "null" {
			return
		}
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			var unquoted string
			if err := common.UnmarshalJsonStr(s, &unquoted); err == nil {
				writeField(name, unquoted)
				return
			}
		}
		writeField(name, s)
	}

	writeField("model", imgReq.Model)
	writeField("prompt", imgReq.Prompt)
	writeField("size", imgReq.Size)
	writeField("quality", imgReq.Quality)
	if imgReq.N != nil {
		writeField("n", strconv.FormatUint(uint64(*imgReq.N), 10))
	}
	writeField("response_format", imgReq.ResponseFormat)
	writeJSONField("background", imgReq.Background)
	writeJSONField("moderation", imgReq.Moderation)
	writeJSONField("output_format", imgReq.OutputFormat)
	writeJSONField("output_compression", imgReq.OutputCompression)
	writeJSONField("partial_images", imgReq.PartialImages)
	writeJSONField("input_fidelity", imgReq.InputFidelity)
	writeJSONField("user", imgReq.User)
	writeJSONField("style", imgReq.Style)
	if imgReq.Watermark != nil {
		writeField("watermark", strconv.FormatBool(*imgReq.Watermark))
	}

	for i, img := range images {
		part, err := createImagePart(writer, "image[]", img.Filename, img.MimeType)
		if err != nil {
			return nil, "", fmt.Errorf("create form file failed (idx=%d): %w", i, err)
		}
		if _, err := part.Write(img.Bytes); err != nil {
			return nil, "", fmt.Errorf("write image bytes failed (idx=%d): %w", i, err)
		}
	}

	if mask != nil {
		part, err := createImagePart(writer, "mask", mask.Filename, mask.MimeType)
		if err != nil {
			return nil, "", fmt.Errorf("create mask form file failed: %w", err)
		}
		if _, err := part.Write(mask.Bytes); err != nil {
			return nil, "", fmt.Errorf("write mask bytes failed: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}
	return &buf, writer.FormDataContentType(), nil
}

// createImagePart 与 writer.CreateFormFile 类似，但允许显式指定 Content-Type，
// 避免默认 application/octet-stream 导致部分上游拒绝。
func createImagePart(writer *multipart.Writer, fieldName, filename, mimeType string) (io.Writer, error) {
	if mimeType == "" {
		mimeType = "image/png"
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(fieldName), escapeQuotes(filename)))
	h.Set("Content-Type", mimeType)
	return writer.CreatePart(h)
}

func escapeQuotes(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

// truncateForError 截断过长的字符串以免错误信息里塞进大段 base64。
func truncateForError(s string) string {
	const max = 80
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
