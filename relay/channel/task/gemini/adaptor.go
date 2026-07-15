package gemini

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction parses body, validates fields and sets default action.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate)
}

// BuildRequestURL constructs the Gemini API predictLongRunning endpoint for Veo.
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	modelName := info.UpstreamModelName
	version := model_setting.GetGeminiVersionSetting(modelName)

	return fmt.Sprintf(
		"%s/%s/models/%s:predictLongRunning",
		a.baseURL,
		version,
		modelName,
	), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", a.apiKey)
	return nil
}

// BuildRequestBody converts request into the Veo predictLongRunning format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, ok := c.Get("task_request")
	if !ok {
		return nil, fmt.Errorf("request not found in context")
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, fmt.Errorf("unexpected task_request type")
	}

	instance := VeoInstance{Prompt: req.Prompt}
	if img := ExtractMultipartImage(c, info); img != nil {
		instance.Image = img
	} else if len(req.Images) > 0 {
		if parsed := ParseImageInput(req.Images[0]); parsed != nil {
			instance.Image = parsed
			info.Action = constant.TaskActionGenerate
		}
	}

	params := &VeoParameters{}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, params); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}
	if params.DurationSeconds == 0 && req.Duration > 0 {
		params.DurationSeconds = req.Duration
	}
	if params.Resolution == "" && req.Size != "" {
		params.Resolution = SizeToVeoResolution(req.Size)
	}
	if params.AspectRatio == "" && req.Size != "" {
		params.AspectRatio = SizeToVeoAspectRatio(req.Size)
	}
	params.Resolution = strings.ToLower(params.Resolution)
	params.SampleCount = 1

	body := VeoRequestPayload{
		Instances:  []VeoInstance{instance},
		Parameters: params,
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var s submitResponse
	if err := common.Unmarshal(responseBody, &s); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
	}
	if strings.TrimSpace(s.Name) == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("missing operation name"), "invalid_response", http.StatusInternalServerError)
	}
	taskID = taskcommon.EncodeLocalTaskID(s.Name)
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return taskID, responseBody, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"veo-3.0-generate-001",
		"veo-3.0-fast-generate-001",
		"veo-3.1-generate-preview",
		"veo-3.1-fast-generate-preview",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "gemini"
}

// EstimateBilling returns OtherRatios based on durationSeconds and resolution.
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	v, ok := c.Get("task_request")
	if !ok {
		return nil
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil
	}

	seconds := ResolveVeoDuration(req.Metadata, req.Duration, req.Seconds)
	resolution := ResolveVeoResolution(req.Metadata, req.Size)
	resRatio := VeoResolutionRatio(info.UpstreamModelName, resolution)

	return map[string]float64{
		"seconds":    float64(seconds),
		"resolution": resRatio,
	}
}

// FetchTask polls task status via the Gemini operations GET endpoint.
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	upstreamName, err := taskcommon.DecodeLocalTaskID(taskID)
	if err != nil {
		return nil, fmt.Errorf("decode task_id failed: %w", err)
	}

	version := model_setting.GetGeminiVersionSetting("default")
	url := fmt.Sprintf("%s/%s/%s", baseUrl, version, upstreamName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var op operationResponse
	if err := common.Unmarshal(respBody, &op); err != nil {
		return nil, fmt.Errorf("unmarshal operation response failed: %w", err)
	}

	ti := &relaycommon.TaskInfo{}

	if op.Error.Message != "" {
		ti.Status = model.TaskStatusFailure
		ti.Reason = op.Error.Message
		ti.Progress = "100%"
		return ti, nil
	}

	if !op.Done {
		ti.Status = model.TaskStatusInProgress
		ti.Progress = "50%"
		return ti, nil
	}

	ti.Status = model.TaskStatusSuccess
	ti.Progress = "100%"

	ti.TaskID = taskcommon.EncodeLocalTaskID(op.Name)

	if len(op.Response.GenerateVideoResponse.GeneratedVideos) > 0 {
		if uri := op.Response.GenerateVideoResponse.GeneratedVideos[0].Video.URI; uri != "" {
			ti.RemoteUrl = uri
		}
	}

	return ti, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	upstreamTaskID := task.GetUpstreamTaskID()
	upstreamName, err := taskcommon.DecodeLocalTaskID(upstreamTaskID)
	if err != nil {
		upstreamName = ""
	}
	modelName := extractModelFromOperationName(upstreamName)
	if strings.TrimSpace(modelName) == "" {
		modelName = "veo-3.0-generate-001"
	}

	video := dto.NewOpenAIVideo()
	video.ID = task.TaskID
	video.Model = modelName
	video.Status = task.Status.ToVideoStatus()
	video.SetProgressStr(task.Progress)
	video.CreatedAt = task.CreatedAt
	if task.FinishTime > 0 {
		video.CompletedAt = task.FinishTime
	} else if task.UpdatedAt > 0 {
		video.CompletedAt = task.UpdatedAt
	}

	return common.Marshal(video)
}

// ============================
// helpers
// ============================

var modelRe = regexp.MustCompile(`models/([^/]+)/operations/`)

func extractModelFromOperationName(name string) string {
	if name == "" {
		return ""
	}
	if m := modelRe.FindStringSubmatch(name); len(m) == 2 {
		return m[1]
	}
	if idx := strings.Index(name, "models/"); idx >= 0 {
		s := name[idx+len("models/"):]
		if p := strings.Index(s, "/operations/"); p > 0 {
			return s[:p]
		}
	}
	return ""
}

// ============================
// ImageTaskAdaptor — Gemini 图片生成
// 将 /v1/images/generations 转换为 generateContent 调用，
// 同步返回图片 data URI，立即写入任务日志。
// ============================

type ImageTaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

func (a *ImageTaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *ImageTaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var imgReq dto.ImageRequest
	if err := common.UnmarshalBodyReusable(c, &imgReq); err != nil {
		return service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(imgReq.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest)
	}

	var rawImages []string
	if len(imgReq.Image) > 0 {
		var imageVal interface{}
		if err := common.Unmarshal(imgReq.Image, &imageVal); err == nil {
			switch v := imageVal.(type) {
			case string:
				if v != "" {
					rawImages = append(rawImages, v)
				}
			case []interface{}:
				for _, img := range v {
					if s, ok := img.(string); ok && s != "" {
						rawImages = append(rawImages, s)
					}
				}
			}
		}
	}

	// 创意控制台等前端把参考图放在 metadata.imageUrls 里（通常是 TOS HTTP URL）。
	if mRaw, ok := imgReq.Extra["metadata"]; ok && len(mRaw) > 0 {
		var meta map[string]interface{}
		if err := common.Unmarshal(mRaw, &meta); err == nil {
			if v, ok2 := meta["imageUrls"]; ok2 {
				if arr, ok3 := v.([]interface{}); ok3 {
					for _, item := range arr {
						if s, ok4 := item.(string); ok4 && s != "" {
							rawImages = append(rawImages, s)
						}
					}
				}
			}
		}
	}

	images := make([]string, 0, len(rawImages))
	for _, s := range rawImages {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "data:") {
			images = append(images, s)
			continue
		}
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			mimeType, b64, err := service.GetImageFromUrl(s)
			if err != nil {
				common.SysLog(fmt.Sprintf("gemini_image: fetch reference image failed url=%s err=%v", s, err))
				return service.TaskErrorWrapper(err, "fetch_reference_image_failed", http.StatusBadGateway)
			}
			images = append(images, fmt.Sprintf("data:%s;base64,%s", mimeType, b64))
			continue
		}
		// 视为已 base64 编码的纯数据，按 PNG 处理。
		images = append(images, "data:image/png;base64,"+s)
	}

	taskReq := relaycommon.TaskSubmitReq{
		Model:  imgReq.Model,
		Prompt: imgReq.Prompt,
		Size:   imgReq.Size,
		Images: images,
	}

	// Pro 模型专属参数：存入 Metadata 供 BuildRequestBody 转发
	meta := map[string]interface{}{}
	if v, ok := imgReq.Extra["seed"]; ok {
		var seedVal float64
		if err := common.Unmarshal(v, &seedVal); err == nil {
			meta["seed"] = seedVal
		}
	}
	if len(meta) > 0 {
		taskReq.Metadata = meta
	}

	info.Action = constant.TaskActionImageGenerate
	c.Set("task_request", taskReq)
	return nil
}

func (a *ImageTaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	base := strings.TrimRight(a.baseURL, "/")

	// 原生 Gemini 渠道（类型 24）：剥离版本后缀后拼 /:version/models/:model:generateContent
	if info.ChannelType == constant.ChannelTypeGemini {
		modelName := info.UpstreamModelName
		version := model_setting.GetGeminiVersionSetting(modelName)
		for _, suffix := range []string{"/v1beta", "/v1"} {
			if strings.HasSuffix(base, suffix) {
				base = strings.TrimSuffix(base, suffix)
				break
			}
		}
		url := fmt.Sprintf("%s/%s/models/%s:generateContent", base, version, modelName)
		common.SysLog(fmt.Sprintf("gemini_image: POST %s (native, model=%s)", url, modelName))
		return url, nil
	}

	// OpenAI 兼容代理：代理不做格式转换，直接透传到 Google Gemini。
	// 需要使用 Gemini 原生格式，但代理在 /gemini 前缀下路由。
	modelName := info.UpstreamModelName
	version := model_setting.GetGeminiVersionSetting(modelName)
	for _, suffix := range []string{"/v1beta", "/v1"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}
	url := fmt.Sprintf("%s/gemini/%s/models/%s:generateContent", base, version, modelName)
	common.SysLog(fmt.Sprintf("gemini_image: POST %s (proxy-passthrough, model=%s)", url, modelName))
	return url, nil
}

func (a *ImageTaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if info.ChannelType == constant.ChannelTypeGemini {
		// 原生 Gemini API 使用 API Key header
		req.Header.Set("x-goog-api-key", a.apiKey)
	} else {
		// 代理兼容：同时发送 Bearer 和 x-goog-api-key，避免网关鉴权分支不一致。
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
		req.Header.Set("x-goog-api-key", a.apiKey)
	}
	return nil
}

func (a *ImageTaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, ok := c.Get("task_request")
	if !ok {
		return nil, fmt.Errorf("task_request not found in context")
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, fmt.Errorf("unexpected task_request type")
	}

	// 原生渠道和代理渠道均使用 Gemini 原生格式。
	// 代理（如 api.asiai.cloud）不做格式转换，直接透传到 Google Gemini API。
	parts := []dto.GeminiPart{{Text: req.Prompt}}
	for _, imgStr := range req.Images {
		imgStr = strings.TrimSpace(imgStr)
		if !strings.HasPrefix(imgStr, "data:") {
			continue
		}
		rest := strings.TrimPrefix(imgStr, "data:")
		idx := strings.Index(rest, ",")
		if idx < 0 {
			continue
		}
		meta := rest[:idx]
		b64 := rest[idx+1:]
		mimeType := "image/png"
		if i := strings.Index(meta, ";"); i >= 0 {
			mimeType = meta[:i]
		} else if meta != "" {
			mimeType = meta
		}
		parts = append(parts, dto.GeminiPart{
			InlineData: &dto.GeminiInlineData{MimeType: mimeType, Data: b64},
		})
	}

	inner := dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: parts},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ResponseModalities: []string{"IMAGE", "TEXT"},
			ImageConfig:        json.RawMessage(`{}`),
		},
	}

	if req.Metadata != nil {
		if sv, ok2 := req.Metadata["seed"]; ok2 {
			switch n := sv.(type) {
			case float64:
				s := int64(n)
				inner.GenerationConfig.Seed = &s
			case int64:
				inner.GenerationConfig.Seed = &n
			}
		}
	}

	// 原生渠道与代理渠道一律使用 Gemini generateContent 原生 body。
	// 之前曾在代理路径里把 model/prompt 加到顶层，结果被透传到 Google 后
	// 会被 "Unknown name \"prompt\"" 拒绝。
	data, err := common.Marshal(inner)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *ImageTaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *ImageTaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	// 原生渠道和代理渠道均返回 Gemini generateContent 响应格式。
	var geminiResp dto.GeminiChatResponse
	if err := common.Unmarshal(responseBody, &geminiResp); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
	}

	imageDataURI := extractGeminiImageDataURI(geminiResp)
	if imageDataURI == "" {
		snippet := string(responseBody)
		if len(snippet) > 1024 {
			snippet = snippet[:1024] + "...(truncated)"
		}
		common.SysLog(fmt.Sprintf("gemini_image: no inline image, body=%s", snippet))
		return "", nil, service.TaskErrorWrapper(
			fmt.Errorf("no image in response: %s", snippet),
			"no_image_in_response",
			http.StatusBadGateway,
		)
	}

	info.TaskRelayInfo.CompletedResult = &relaycommon.SyncTaskResult{
		ResultURL: imageDataURI,
	}

	// 尝试上传到 TOS，成功则用 https:// URL 替换 data URI，
	// 避免大体积 base64 存入数据库。
	if finalURL, tosErr := uploadDataURIToTos(imageDataURI); tosErr == nil {
		imageDataURI = finalURL
		info.TaskRelayInfo.CompletedResult.ResultURL = finalURL
	} else if system_setting.TosEnabled {
		common.SysLog(fmt.Sprintf("gemini_image: TOS upload failed, fallback to data URI: %v", tosErr))
	}

	// 以 OpenAI images/generations 标准格式返回，前端可直接渲染，无需轮询。
	// imageDataURI 是 TOS URL 或 data URI，放在 url 字段。
	imageResp := dto.ImageResponse{
		Created: time.Now().Unix(),
		Data: []dto.ImageData{
			{Url: imageDataURI},
		},
	}
	// tiered_expr 计费：用上游 usageMetadata 真实 token 重算 quota，覆盖预扣值。
	// RelayTaskSubmit 后续 SettleBilling 会基于此做差额结算。
	if info.TieredBillingSnapshot != nil && geminiResp.UsageMetadata.TotalTokenCount > 0 {
		usage := normalizeGeminiUsage(&geminiResp.UsageMetadata)
		usedVars := billingexpr.UsedVars(info.TieredBillingSnapshot.ExprString)
		if ok, actualQuota, _ := service.TryTieredSettle(info, service.BuildTieredTokenParams(usage, false, usedVars)); ok && actualQuota > 0 {
			info.PriceData.Quota = actualQuota
		}
	}

	c.JSON(http.StatusOK, imageResp)
	return info.PublicTaskID, responseBody, nil
}

// normalizeGeminiUsage 把 Gemini UsageMetadata 适配到 dto.Usage，
// 供 BuildTieredTokenParams 读取。Gemini 的 PromptTokensDetails / CandidatesTokensDetails
// 是 [{modality, tokenCount}] 数组，按 modality 分到 image / text token 桶。
// thoughtsTokenCount 是模型的思考 token，按 output text 计费（Gemini 定价把 thinking 当 output 收费）。
func normalizeGeminiUsage(m *dto.GeminiUsageMetadata) *dto.Usage {
	u := &dto.Usage{
		PromptTokens:     m.PromptTokenCount,
		CompletionTokens: m.CandidatesTokenCount + m.ThoughtsTokenCount,
		TotalTokens:      m.TotalTokenCount,
	}
	u.CompletionTokenDetails.TextTokens = m.ThoughtsTokenCount
	for _, d := range m.PromptTokensDetails {
		switch strings.ToUpper(d.Modality) {
		case "IMAGE":
			u.PromptTokensDetails.ImageTokens += d.TokenCount
		case "AUDIO":
			u.PromptTokensDetails.AudioTokens += d.TokenCount
		case "TEXT":
			u.PromptTokensDetails.TextTokens += d.TokenCount
		}
	}
	for _, d := range m.CandidatesTokensDetails {
		switch strings.ToUpper(d.Modality) {
		case "IMAGE":
			u.CompletionTokenDetails.ImageTokens += d.TokenCount
		case "AUDIO":
			u.CompletionTokenDetails.AudioTokens += d.TokenCount
		case "TEXT":
			u.CompletionTokenDetails.TextTokens += d.TokenCount
		}
	}
	return u
}

func extractGeminiImageDataURI(resp dto.GeminiChatResponse) string {
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil || part.InlineData.Data == "" {
				continue
			}
			mime := part.InlineData.MimeType
			if mime == "" {
				mime = "image/png"
			}
			if strings.HasPrefix(mime, "image/") {
				return fmt.Sprintf("data:%s;base64,%s", mime, part.InlineData.Data)
			}
		}
	}
	return ""
}

// uploadDataURIToTos 将 data URI 上传到 TOS。
// TOS 未启用时返回 error，调用方应回退到 data URI。
func uploadDataURIToTos(dataURI string) (string, error) {
	if !system_setting.TosEnabled {
		return "", fmt.Errorf("tos not enabled")
	}

	// 解析 data URI: data:<mime>;base64,<data>
	if !strings.HasPrefix(dataURI, "data:") {
		return "", fmt.Errorf("not a data URI")
	}
	rest := strings.TrimPrefix(dataURI, "data:")
	idx := strings.Index(rest, ",")
	if idx < 0 {
		return "", fmt.Errorf("invalid data URI format")
	}
	meta := rest[:idx]
	b64Data := rest[idx+1:]

	var mimeType string
	if i := strings.Index(meta, ";"); i >= 0 {
		mimeType = meta[:i]
	} else {
		mimeType = meta
	}
	if mimeType == "" {
		mimeType = "image/png"
	}

	// 确定文件扩展名
	extMap := map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
	ext := extMap[mimeType]
	if ext == "" {
		ext = ".png"
	}

	// 解码 base64
	imgBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		// 尝试 RawStdEncoding（无 padding）
		imgBytes, err = base64.RawStdEncoding.DecodeString(b64Data)
		if err != nil {
			return "", fmt.Errorf("base64 decode: %w", err)
		}
	}

	objectKey := fmt.Sprintf("gemini-image/%s%s", uuid.New().String(), ext)
	return service.TosUploadFile(bytes.NewReader(imgBytes), objectKey, mimeType)
}

func (a *ImageTaskAdaptor) GetModelList() []string { return []string{} }
func (a *ImageTaskAdaptor) GetChannelName() string  { return "gemini_image" }

func (a *ImageTaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	return nil, fmt.Errorf("FetchTask not supported: gemini_image tasks complete synchronously")
}

func (a *ImageTaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{Status: model.TaskStatusSuccess, Progress: "100%"}, nil
}
