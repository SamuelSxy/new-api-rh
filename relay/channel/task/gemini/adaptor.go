package gemini

import (
	"bytes"
	"encoding/base64"
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

	var images []string
	if len(imgReq.Image) > 0 {
		var imageVal interface{}
		if err := common.Unmarshal(imgReq.Image, &imageVal); err == nil {
			switch v := imageVal.(type) {
			case string:
				if v != "" {
					images = append(images, v)
				}
			case []interface{}:
				for _, img := range v {
					if s, ok := img.(string); ok && s != "" {
						images = append(images, s)
					}
				}
			}
		}
	}

	taskReq := relaycommon.TaskSubmitReq{
		Model:  imgReq.Model,
		Prompt: imgReq.Prompt,
		Size:   imgReq.Size,
		Images: images,
	}

	// Pro 模型专属参数：存入 Metadata 供 BuildRequestBody 转发
	meta := map[string]interface{}{}
	if imgReq.Watermark != nil {
		// Watermark 复用为 enhancePrompt 开关（保留原语义）
	}
	if v, ok := imgReq.Extra["seed"]; ok {
		meta["seed"] = v
	}
	if v, ok := imgReq.Extra["enhance_prompt"]; ok {
		meta["enhance_prompt"] = v
	}
	if v, ok := imgReq.Extra["negative_prompt"]; ok {
		meta["negative_prompt"] = v
	}
	if len(meta) > 0 {
		taskReq.Metadata = meta
	}

	info.Action = constant.TaskActionImageGenerate
	c.Set("task_request", taskReq)
	return nil
}

func (a *ImageTaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	modelName := info.UpstreamModelName
	version := model_setting.GetGeminiVersionSetting(modelName)

	// 规范化 baseURL：移除尾部斜杠；若以 /v1 或 /v1beta 结尾则剥离，
	// 以避免与下面拼接的 /{version}/models/... 产生重复版本号导致 404/405。
	base := strings.TrimRight(a.baseURL, "/")
	for _, suffix := range []string{"/v1beta", "/v1"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}

	// 原生 Gemini 渠道（类型 24）直接拼 /{version}/models/...
	// OpenAI 兼容代理通常将 Gemini 原生 API 挂在 /gemini 前缀下，
	// 例如 https://api.asiai.cloud/gemini/v1beta/models/...
	var url string
	if info.ChannelType == constant.ChannelTypeGemini {
		url = fmt.Sprintf("%s/%s/models/%s:generateContent", base, version, modelName)
	} else {
		url = fmt.Sprintf("%s/gemini/%s/models/%s:generateContent", base, version, modelName)
	}
	common.SysLog(fmt.Sprintf("gemini_image: POST %s (model=%s, version=%s, channelType=%d)", url, modelName, version, info.ChannelType))
	return url, nil
}

func (a *ImageTaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", a.apiKey)
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

	body := dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: parts},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ResponseModalities: []string{"IMAGE", "TEXT"},
		},
	}

	// 应用 quality → imageSize 映射（由 task BuildRequestBody 处理，size 已通过 Size 传入）
	// 转发 Pro 模型专属参数
	if req.Metadata != nil {
		if v, ok := req.Metadata["seed"]; ok {
			switch n := v.(type) {
			case float64:
				s := int64(n)
				body.GenerationConfig.Seed = &s
			case int64:
				body.GenerationConfig.Seed = &n
			}
		}
		if v, ok := req.Metadata["enhance_prompt"]; ok {
			if b, ok2 := v.(bool); ok2 {
				body.EnhancePrompt = &b
			}
		}
		if v, ok := req.Metadata["negative_prompt"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				body.NegativePrompt = s
			}
		}
	}
	data, err := common.Marshal(body)
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

	var geminiResp dto.GeminiChatResponse
	if err := common.Unmarshal(responseBody, &geminiResp); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
	}

	imageDataURI := extractGeminiImageDataURI(geminiResp)
	if imageDataURI == "" {
		return "", nil, service.TaskErrorWrapper(
			fmt.Errorf("no image returned from Gemini"),
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
	c.JSON(http.StatusOK, imageResp)
	return info.PublicTaskID, responseBody, nil
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
