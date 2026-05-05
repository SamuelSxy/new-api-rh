package doubao

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	ratio_setting "github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/QuantumNous/new-api/common"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

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

// EstimateBilling implements parameter-based pricing for Doubao video tasks.
// Reads video_size from request metadata and looks up a model price entry
// named "{upstreamModel}@{video_size}" in the admin price table.
//
// Example admin configuration:
//
//	doubao-seedance-2-0-260128        -> 1.0  (base price)
//	doubao-seedance-2-0-260128@720p   -> 0.8
//	doubao-seedance-2-0-260128@1080p  -> 1.5
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil || taskReq.Metadata == nil {
		return nil
	}
	videoSize, _ := taskReq.Metadata["video_size"].(string)
	videoSize = strings.TrimSpace(videoSize)
	if videoSize == "" {
		return nil
	}
	paramModelName := info.UpstreamModelName + "@" + videoSize
	paramPrice, found := ratio_setting.GetModelPrice(paramModelName, false)
	if !found {
		return nil
	}
	basePrice := info.PriceData.ModelPrice
	if basePrice <= 0 {
		return nil
	}
	return map[string]float64{"video_size": paramPrice / basePrice}
}

// BuildRequestBody converts request into Doubao specific format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body, err := a.convertToRequestPayload(&req)
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

// AdjustBillingOnComplete overrides the default token recalculation when a
// resolution-specific model ratio is configured (e.g. "doubao-seedance-2-0-260128@1080p").
// It reads the actual resolution from the completed task response, looks up the
// "@resolution" ratio, and returns the recalculated quota so that
// settleTaskBillingOnComplete uses it instead of the plain model ratio.
// Returns 0 to fall back to the default RecalculateTaskQuotaByTokens behaviour.
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) (int, string) {
	if taskResult.TotalTokens <= 0 {
		common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling skip: TotalTokens=%d, task=%s", taskResult.TotalTokens, task.TaskID))
		return 0, ""
	}

	// Parse completed task data to get actual resolution returned by Doubao.
	var resTask responseTask
	if err := common.Unmarshal(task.Data, &resTask); err != nil {
		common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling skip: unmarshal err=%v, task=%s", err, task.TaskID))
		return 0, ""
	}
	if resTask.Resolution == "" {
		common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling skip: resolution empty, status=%s, task=%s, dataLen=%d", resTask.Status, task.TaskID, len(task.Data)))
		return 0, ""
	}

	// Build "model@resolution" name and look up its ratio.
	modelName := task.Properties.OriginModelName
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		modelName = bc.OriginModelName
	}
	paramModelName := modelName + "@" + strings.ToLower(strings.TrimSpace(resTask.Resolution))
	modelRatio, hasRatio, _ := ratio_setting.GetModelRatio(paramModelName)
	// GetModelRatio returns (37.5, SelfUseModeEnabled, ...) when not found; we must
	// treat "not explicitly configured" as absent to avoid using a wrong default.
	if !hasRatio {
		common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling skip: GetModelRatio hasRatio=false for %s, task=%s", paramModelName, task.TaskID))
		return 0, ""
	}
	// Verify the key actually exists in the configured ratio map.
	if _, exists := ratio_setting.GetModelRatioCopy()[paramModelName]; !exists {
		common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling skip: key not in ratio map: %s, task=%s", paramModelName, task.TaskID))
		return 0, ""
	}

	group := task.Group
	if group == "" {
		common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling skip: group empty, task=%s", task.TaskID))
		return 0, ""
	}
	groupRatio := ratio_setting.GetGroupRatio(group)
	if userRatio, has := ratio_setting.GetGroupGroupRatio(group, group); has {
		groupRatio = userRatio
	}

	actualQuota := int(float64(taskResult.TotalTokens) * modelRatio * groupRatio)
	common.SysLog(fmt.Sprintf("[Doubao] AdjustBilling: task=%s, model=%s, resolution=%s, totalTokens=%d, modelRatio=%.4f, groupRatio=%.4f, preCharge=%d, actualQuota=%d",
		task.TaskID, paramModelName, resTask.Resolution, taskResult.TotalTokens, modelRatio, groupRatio, task.Quota, actualQuota))
	reason := fmt.Sprintf("token重算：tokens=%d, modelRatio=%.2f, groupRatio=%.2f", taskResult.TotalTokens, modelRatio, groupRatio)
	return actualQuota, reason
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
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

	if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
		r.Duration = lo.ToPtr(dto.IntValue(sec))
	}

	// Map video_size → resolution field (the @suffix is only for billing lookup, not sent upstream)
	if r.Resolution == "" {
		if videoSize, _ := metadata["video_size"].(string); strings.TrimSpace(videoSize) != "" {
			r.Resolution = strings.TrimSpace(videoSize)
		}
	}

	r.Content = lo.Reject(r.Content, func(c ContentItem, _ int) bool { return c.Type == "text" })
	r.Content = append(r.Content, ContentItem{
		Type: "text",
		Text: req.Prompt,
	})

	return &r, nil
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

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.SetMetadata("url", dResp.Content.VideoURL)
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
