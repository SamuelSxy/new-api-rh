package doubao

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type MediaURL struct {
	URL string `json:"url,omitempty"`
}

type requestPayload struct {
	Model                 string         `json:"model"`
	Content               []ContentItem  `json:"content,omitempty"`
	CallbackURL           string         `json:"callback_url,omitempty"`
	ReturnLastFrame       *dto.BoolValue `json:"return_last_frame,omitempty"`
	ServiceTier           string         `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *dto.IntValue  `json:"execution_expires_after,omitempty"`
	GenerateAudio         *dto.BoolValue `json:"generate_audio,omitempty"`
	Draft                 *dto.BoolValue `json:"draft,omitempty"`
	Tools                 []struct {
		Type string `json:"type,omitempty"`
	} `json:"tools,omitempty"`
	Resolution  string         `json:"resolution,omitempty"`
	Ratio       string         `json:"ratio,omitempty"`
	RenderMode  string         `json:"render_mode,omitempty"`
	Duration    *dto.IntValue  `json:"duration,omitempty"`
	Frames      *dto.IntValue  `json:"frames,omitempty"`
	Seed        *dto.IntValue  `json:"seed,omitempty"`
	CameraFixed *dto.BoolValue `json:"camera_fixed,omitempty"`
	Watermark   *dto.BoolValue `json:"watermark,omitempty"`
}

type responsePayload struct {
	ID string `json:"id"` // task_id
}

type responseTask struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL string `json:"video_url"`
	} `json:"content"`
	Seed            int    `json:"seed"`
	Resolution      string `json:"resolution"`
	Duration        int    `json:"duration"`
	Ratio           string `json:"ratio"`
	FramesPerSecond int    `json:"framespersecond"`
	ServiceTier     string `json:"service_tier"`
	Tools           []struct {
		Type string `json:"type"`
	} `json:"tools"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		ToolUsage        struct {
			WebSearch int `json:"web_search"`
		} `json:"tool_usage"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

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
	// Accept only POST /v1/video/generations as "generate" action.
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL constructs the upstream URL.
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/v3/contents/generations/tasks", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// resolveResolutionFromRequest 从 TaskSubmitReq 的 metadata 中提取并归一化分辨率。
func resolveResolutionFromRequest(req relaycommon.TaskSubmitReq) string {
	if req.Metadata == nil {
		return ""
	}
	v, ok := req.Metadata["resolution"]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return normalizeDoubaoResolution(s)
}

// applyResolutionPricing 若管理员为超分模型配置了 @resolution 专属价格
// （如 doubao-seedance-2-0-fall@1080p），则用该价格覆盖 info.PriceData 中的基础定价。
// 这样后续的预扣和 AdjustBillingOnComplete 均使用分辨率匹配的费率。
func applyResolutionPricing(info *relaycommon.RelayInfo, resolution string) {
	if resolution == "" || !mediakitEnhanceModels[info.OriginModelName] {
		return
	}
	qualifiedName := info.OriginModelName + "@" + resolution
	if resRatio, ok, _ := ratio_setting.GetModelRatio(qualifiedName); ok {
		info.PriceData.ModelRatio = resRatio
		info.PriceData.UsePrice = false
		info.PriceData.Quota = int(resRatio * common.QuotaPerUnit * info.PriceData.GroupRatioInfo.GroupRatio)
	} else if resPrice, ok2 := ratio_setting.GetModelPrice(qualifiedName, false); ok2 {
		info.PriceData.ModelPrice = resPrice
		info.PriceData.UsePrice = true
		info.PriceData.Quota = int(resPrice * common.QuotaPerUnit * info.PriceData.GroupRatioInfo.GroupRatio)
	}
}

// getVideoInputRatioForResolution 返回模型在指定分辨率下的视频输入倍率。
// 优先级：model@resolution > model（fallback）。
func getVideoInputRatioForResolution(modelName, resolution string) (float64, bool) {
	if resolution != "" {
		if r, ok := ratio_setting.GetVideoInputRatio(modelName + "@" + resolution); ok {
			return r, true
		}
	}
	return ratio_setting.GetVideoInputRatio(modelName)
}

// EstimateBilling 根据模型计费模式返回 OtherRatios。
// 对于按秒计费模型，返回 video_input 倍率 + seconds（估算秒数）。
// 对于普通视频输入，仅返回 video_input 倍率。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	ratios := make(map[string]float64)
	resolution := resolveResolutionFromRequest(req)

	// 按秒计费
	if defaultSec, ok := isDurationBilling(info.OriginModelName); ok {
		sec := resolveRequestedDuration(req, defaultSec)
		if sec > 0 {
			// 已知时长：按估算秒数预扣
			ratios["seconds"] = float64(sec)
		}
		// sec <= 0 表示时长未知，不预扣时长费，AdjustBillingOnComplete 完成后补收
		// 分辨率专用定价：若管理员配置了 @resolution 专属价格则覆盖基础倍率
		applyResolutionPricing(info, resolution)
		// 也检查是否有视频输入倍率（支持按分辨率区分）
		if hasVideoInMetadata(req.Metadata) {
			if r, ok2 := getVideoInputRatioForResolution(info.OriginModelName, resolution); ok2 {
				ratios["video_input"] = r
			}
		}
		return ratios
	}

	// 普通视频输入倍率（支持按分辨率区分）
	if hasVideoInMetadata(req.Metadata) {
		if r, ok := getVideoInputRatioForResolution(info.OriginModelName, resolution); ok {
			ratios["video_input"] = r
		}
	}
	if len(ratios) == 0 {
		return nil
	}
	return ratios
}

// isDurationBilling 返回模型是否按秒计费及默认秒数。
// 优先级：billing_setting DB 配置 > 硬编码 durationBillingModels（通过 billing_setting fallback）。
func isDurationBilling(modelName string) (defaultSec int, ok bool) {
	if billing_setting.IsDurationBillingModel(modelName) {
		if sec, configured := billing_setting.GetDurationBillingDefault(modelName); configured {
			return sec, true
		}
		return 5, true
	}
	return 0, false
}

// resolveRequestedDuration 从请求中提取视频时长（秒），优先级：
// req.Duration > req.Seconds > metadata["duration"] > defaultSec
func resolveRequestedDuration(req relaycommon.TaskSubmitReq, defaultSec int) int {
	if req.Duration > 0 {
		return req.Duration
	}
	if req.Seconds != "" {
		if i, err := strconv.Atoi(req.Seconds); err == nil && i > 0 {
			return i
		}
	}
	if req.Metadata != nil {
		if v, ok := req.Metadata["duration"]; ok {
			switch n := v.(type) {
			case float64:
				if n > 0 {
					return int(n)
				}
			case int:
				if n > 0 {
					return n
				}
			case string:
				if i, err := strconv.Atoi(n); err == nil && i > 0 {
					return i
				}
			}
		}
	}
	return defaultSec
}

// AdjustBillingOnComplete 用 API 实际返回的视频时长重新结算按秒计费。
// - 已知预估时长：actualQuota = preChargedQuota / estimatedSeconds * actualSeconds
// - 未知预估时长（estimatedSeconds <= 0）：从 modelRatio × groupRatio × actualDuration 从零计算
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, _ *relaycommon.TaskInfo) int {
	if _, ok := isDurationBilling(task.Properties.OriginModelName); !ok {
		return 0
	}

	// 从任务结果数据中解析实际视频时长
	var resTask responseTask
	if err := common.Unmarshal(task.Data, &resTask); err != nil || resTask.Duration <= 0 {
		return 0
	}

	bc := task.PrivateData.BillingContext
	if bc == nil {
		return 0
	}

	estimatedSeconds := bc.OtherRatios["seconds"]

	if estimatedSeconds > 0 {
		// 已知预估时长：按比例折算
		quota := int(float64(task.Quota) / estimatedSeconds * float64(resTask.Duration))
		if quota < 0 {
			quota = 0
		}
		return quota
	}

	// 未知预估时长（提交时未预扣秒数）：从 modelRatio × groupRatio × actualDuration 从零计算
	if bc.ModelRatio <= 0 || bc.GroupRatio <= 0 {
		return 0
	}
	videoInputRatio := 1.0
	if r, ok := bc.OtherRatios["video_input"]; ok && r > 0 {
		videoInputRatio = r
	}
	quota := int(bc.ModelRatio * common.QuotaPerUnit * bc.GroupRatio * float64(resTask.Duration) * videoInputRatio)
	if quota < 0 {
		quota = 0
	}
	return quota
}

// hasVideoInMetadata 直接检查 metadata 的 content 数组是否包含 video_url 条目，
// 避免构建完整的上游 requestPayload。
func hasVideoInMetadata(metadata map[string]interface{}) bool {
	if metadata == nil {
		return false
	}
	contentRaw, ok := metadata["content"]
	if !ok {
		return false
	}
	contentSlice, ok := contentRaw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range contentSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["type"] == "video_url" {
			return true
		}
		if _, has := itemMap["video_url"]; has {
			return true
		}
	}
	return false
}

// needsMediakitEnhancement 判断是否需要 Mediakit 超分：
// 仅当 MediakitEnabled=true 且模型在 mediakitEnhanceModels 中
// 且分辨率为 720p 或 1080p 时返回 true。
// 超分720：先生成 480p 再超分；超分1080：先生成 720p 再超分。
func needsMediakitEnhancement(originModelName, resolution string) bool {
	if !system_setting.MediakitEnabled {
		return false
	}
	if !mediakitEnhanceModels[originModelName] {
		return false
	}
	norm := normalizeDoubaoResolution(resolution)
	return norm == "720p" || norm == "1080p"
}

// BuildRequestBody converts request into Doubao specific format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}
	if info.IsModelMapped {
		body.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = body.Model
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
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Doubao response
	var dResp responsePayload
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if dResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return dResp.ID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/v3/contents/generations/tasks/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}


func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*requestPayload, error) {
	r := requestPayload{
		Model:   req.Model,
		Content: []ContentItem{},
	}

	// Add images if present
	if req.HasImage() {
		for _, imgURL := range req.Images {
			r.Content = append(r.Content, ContentItem{
				Type: "image_url",
				ImageURL: &MediaURL{
					URL: imgURL,
				},
			})
		}
	}

	metadata := req.Metadata
	if err := taskcommon.UnmarshalMetadata(metadata, &r); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	// Studio/schema 历史配置里常见 resolution 为 "720" / "1080" / "1280x720"，
	// doubao i2v 期望 "720p" / "1080p" 这类枚举，这里做兼容归一化。
	r.Resolution = normalizeDoubaoResolution(r.Resolution)

	// 若需要 Mediakit 超分：将实际分辨率存入 TaskRelayInfo，改写请求为中间分辨率。
	// 超分1080：先生成 720p，再超分到 1080p；超分720：先生成 480p，再超分到 720p。
	if info != nil && info.TaskRelayInfo != nil && needsMediakitEnhancement(info.OriginModelName, r.Resolution) {
		info.TaskRelayInfo.MediakitTargetResolution = r.Resolution
		if r.Resolution == "1080p" {
			r.Resolution = "720p"
		} else {
			r.Resolution = "480p"
		}
	}

	// 顶层 duration 优先级最高（与 resolveRequestedDuration 扣费逻辑保持一致），
	// 其次是 req.Seconds，metadata.duration 已由 UnmarshalMetadata 写入 r.Duration 作为兜底。
	if req.Duration > 0 {
		r.Duration = lo.ToPtr(dto.IntValue(req.Duration))
	} else if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
		r.Duration = lo.ToPtr(dto.IntValue(sec))
	}

	r.Content = lo.Reject(r.Content, func(c ContentItem, _ int) bool { return c.Type == "text" })
	r.Content = append(r.Content, ContentItem{
		Type: "text",
		Text: req.Prompt,
	})

	return &r, nil
}

func normalizeDoubaoResolution(value string) string {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return value
	}

	switch v {
	case "480", "480p", "640x480":
		return "480p"
	case "720", "720p", "1280x720":
		return "720p"
	case "1080", "1080p", "1920x1080":
		return "1080p"
	default:
		return value
	}
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	// Map Doubao status to internal status
	switch resTask.Status {
	case "pending", "queued":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "10%"
	case "processing", "running":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case "succeeded":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = resTask.Content.VideoURL
		// 解析 usage 信息用于按倍率计费
		taskResult.CompletionTokens = resTask.Usage.CompletionTokens
		taskResult.TotalTokens = resTask.Usage.TotalTokens
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = resTask.Error.Message
	default:
		// Unknown status, treat as processing
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var dResp responseTask
	if err := common.Unmarshal(originTask.Data, &dResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal doubao task data failed")
	}

	// 优先使用 PrivateData.ResultURL（Mediakit 超分后的 URL），
	// fall 模型走超分路径时 task.Data 里的 video_url 是中间分辨率原始 TOS 地址（短期签名），
	// 超分完成后 PrivateData.ResultURL 才是有效的最终视频地址。
	videoURL := originTask.GetResultURL()
	if videoURL == "" {
		videoURL = dResp.Content.VideoURL
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.SetMetadata("url", videoURL)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	openAIVideo.Model = originTask.Properties.OriginModelName

	if dResp.Status == "failed" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: dResp.Error.Message,
			Code:    dResp.Error.Code,
		}
	}

	return common.Marshal(openAIVideo)
}
