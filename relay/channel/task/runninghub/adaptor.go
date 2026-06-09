package runninghub

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	ratio_setting "github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

// RunningHub 服务器位于 Tencent EdgeOne CDN 后，CDN 接受 HTTP/2 TLS 协商但在转发时会
// RST 连接。使用强制 HTTP/1.1 的专用客户端绕过此问题。
var (
	runningHubHTTP1Client     *http.Client
	runningHubHTTP1ClientOnce sync.Once
)

func getRunningHubClient() *http.Client {
	runningHubHTTP1ClientOnce.Do(func() {
		runningHubHTTP1Client = &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2: false,
				TLSClientConfig:   &tls.Config{NextProtos: []string{"http/1.1"}},
			},
		}
	})
	return runningHubHTTP1Client
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	a.apiKey = info.ApiKey
}

// EstimateBilling implements parameter-based pricing for RunningHub workflows.
// Reads a pricing parameter (resolution / quality / billing_tier) from request
// metadata and looks up a model price entry named "{upstreamModel}@{value}".
//
// Example admin configuration in the model price table:
//
//	rhart-image-n-pro-official/edit       -> 1.0  (base / no-param price)
//	rhart-image-n-pro-official/edit@1k    -> 0.8
//	rhart-image-n-pro-official/edit@2k    -> 1.0
//	rhart-image-n-pro-official/edit@4k    -> 1.5
//
// Returns ratio = paramPrice / basePrice so the final charge equals the
// parameterized price. Falls back to base price when no entry is found.
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	// Try common pricing parameters in priority order.
	// Add any custom parameter keys here to support new model-specific pricing dimensions.
	pricingKeys := []string{"resolution", "quality", "billing_tier", "size", "style", "duration"}
	var paramValue, matchedKey string
	for _, key := range pricingKeys {
		if v, ok := taskReq.Metadata[key]; ok {
			if s := common.Interface2String(v); strings.TrimSpace(s) != "" {
				paramValue = strings.TrimSpace(s)
				matchedKey = key
				break
			}
		}
	}
	if paramValue == "" {
		return nil
	}
	// Look up "{upstreamModelName}@{paramValue}" in the model price table.
	paramModelName := info.UpstreamModelName + "@" + paramValue
	paramPrice, found := ratio_setting.GetModelPrice(paramModelName, false)
	if !found {
		return nil
	}
	basePrice := info.PriceData.ModelPrice
	if basePrice <= 0 {
		return nil
	}
	return map[string]float64{matchedKey: paramPrice / basePrice}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	path := c.Request.URL.Path

	// /v1/audio/speech: parse as OpenAI AudioRequest and convert to TaskSubmitReq.
	// This path is reached when RelayAudioOrTask routes a RunningHub channel here.
	if strings.HasSuffix(path, "/audio/speech") {
		var audioReq dto.AudioRequest
		if err := common.UnmarshalBodyReusable(c, &audioReq); err != nil {
			return service.TaskErrorWrapper(errors.Wrap(err, "parse_audio_request_failed"), "invalid_request", http.StatusBadRequest)
		}
		if strings.TrimSpace(audioReq.Input) == "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("input is required"), "invalid_request", http.StatusBadRequest)
		}

		metadata := map[string]interface{}{}
		if audioReq.Voice != "" {
			metadata["voice_id"] = audioReq.Voice
		}
		if audioReq.Speed != nil {
			metadata["speed"] = *audioReq.Speed
		}
		if audioReq.ResponseFormat != "" {
			metadata["response_format"] = audioReq.ResponseFormat
		}
		// Merge any extra fields from AudioRequest.Metadata (provider-specific params)
		if len(audioReq.Metadata) > 0 {
			var extra map[string]interface{}
			if err := common.Unmarshal(audioReq.Metadata, &extra); err == nil {
				for k, v := range extra {
					metadata[k] = v
				}
			}
		}

		taskReq := relaycommon.TaskSubmitReq{
			Model:    audioReq.Model,
			Prompt:   audioReq.Input,
			Metadata: metadata,
		}
		// api_path in metadata overrides the upstream model name (workflow path)
		if apiPath, ok := metadata["api_path"].(string); ok && strings.TrimSpace(apiPath) != "" {
			info.UpstreamModelName = strings.TrimSpace(apiPath)
		}
		info.Action = constant.TaskActionAudioGenerate
		c.Set("task_request", taskReq)
		return nil
	}

	// /v1/images/generations or /v1/images/edits: parse as standard OpenAI ImageRequest.
	// Users send the standard OpenAI format; we convert to TaskSubmitReq for RunningHub.
	if strings.Contains(path, "/images/generations") || strings.Contains(path, "/images/edits") {
		return a.validateImageRequest(c, info)
	}

	if taskErr := relaycommon.ValidateMultipartDirect(c, info); taskErr != nil {
		return taskErr
	}

	// Determine action based on the request path for the dedicated RunningHub endpoints.
	// For legacy entries (/v1/video/generations etc.) default to imageGenerate.
	switch {
	case strings.HasPrefix(path, "/runninghub/") && strings.Contains(path, "/video"):
		// Keep the action set by ValidateMultipartDirect:
		// has image input → "generate" (image-to-video)
		// no image input  → "textGenerate" (text-to-video)
	case strings.HasPrefix(path, "/runninghub/") && strings.Contains(path, "/text"):
		info.Action = constant.TaskActionTextOutput
	case strings.HasPrefix(path, "/runninghub/") && strings.Contains(path, "/audio"):
		info.Action = constant.TaskActionAudioGenerate
	default:
		// /runninghub/.../image  OR  legacy endpoints → image generation
		info.Action = constant.TaskActionImageGenerate
	}

	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapper(errors.Wrap(err, "get_task_request_failed"), "invalid_request", http.StatusBadRequest)
	}
	if taskReq.Metadata != nil {
		if apiPath, ok := taskReq.Metadata["api_path"].(string); ok && strings.TrimSpace(apiPath) != "" {
			info.UpstreamModelName = strings.TrimSpace(apiPath)
		}
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	path := strings.TrimSpace(info.UpstreamModelName)
	if path == "" {
		return "", errors.New("runninghub model path is required")
	}
	return buildRunningHubURL(a.baseURL, path), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	contentType := c.Request.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_task_request_failed")
	}

	body := make(map[string]any)
	for k, v := range taskReq.Metadata {
		if isInternalRuntimeKey(k) {
			continue
		}
		body[k] = v
	}
	if taskReq.Prompt != "" {
		if info.Action == constant.TaskActionAudioGenerate {
			// Audio/TTS workflows use "text" as the input field
			body["text"] = taskReq.Prompt
			// RunningHub audio requires enable_base64_output; default to false if not set
			if _, exists := body["enable_base64_output"]; !exists {
				body["enable_base64_output"] = false
			}
		} else {
			body["prompt"] = taskReq.Prompt
		}
	}
	if taskReq.Image != "" {
		body["image"] = taskReq.Image
	}
	if len(taskReq.Images) > 0 {
		body["images"] = taskReq.Images
	}
	if taskReq.Size != "" {
		body["size"] = taskReq.Size
	}
	if taskReq.Duration > 0 {
		body["duration"] = taskReq.Duration
	}
	if taskReq.Seconds != "" {
		body["seconds"] = taskReq.Seconds
	}
	if taskReq.InputReference != "" {
		body["input_reference"] = taskReq.InputReference
	}

	bodyBytes, err := common.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(err, "marshal_runninghub_request_failed")
	}
	return bytes.NewReader(bodyBytes), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	fullRequestURL, err := a.BuildRequestURL(info)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fullRequestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	if err = a.BuildRequestHeader(c, req, info); err != nil {
		return nil, fmt.Errorf("setup request header failed: %w", err)
	}
	resp, err := getRunningHubClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	return resp, nil
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var body map[string]any
	if err = common.Unmarshal(responseBody, &body); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrap(err, "unmarshal_response_body_failed"), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamTaskID := extractTaskID(body)
	if upstreamTaskID == "" && !hasInlineResult(body) {
		// RunningHub 通过 HTTP 200 + JSON 传递业务错误（如缺少必填字段）
		if errCode, ok := body["errorCode"].(string); ok && errCode != "" {
			errMsg, _ := body["errorMessage"].(string)
			if errMsg == "" {
				errMsg = errCode
			}
			taskErr = service.TaskErrorWrapper(errors.New(errMsg), "upstream_error", http.StatusBadRequest)
			return
		}
		taskErr = service.TaskErrorWrapper(errors.New("taskId is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}
	if upstreamTaskID == "" {
		upstreamTaskID = info.PublicTaskID
	}

	rewrittenBody, err := sjson.SetBytes(responseBody, "taskId", info.PublicTaskID)
	if err == nil {
		responseBody = rewrittenBody
	}
	c.Data(http.StatusOK, "application/json", responseBody)
	return upstreamTaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID := common.Interface2String(body["task_id"])
	if taskID == "" {
		taskID = common.Interface2String(body["taskId"])
	}
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	reqBody, err := common.Marshal(map[string]string{
		"taskId": taskID,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, buildRunningHubURL(baseURL, "/openapi/v2/query"), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var taskResp struct {
		TaskID       string `json:"taskId"`
		Status       string `json:"status"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		Results      []struct {
			URL     string `json:"url"`
			FileURL string `json:"fileUrl"`
			Text    string `json:"text"`
		} `json:"results"`
	}
	if err := common.Unmarshal(respBody, &taskResp); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal runninghub task response")
	}
	if taskResp.Status == "" {
		var wrapped struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				TaskID       string `json:"taskId"`
				Status       string `json:"status"`
				ErrorCode    string `json:"errorCode"`
				ErrorMessage string `json:"errorMessage"`
				Results      []struct {
					URL     string `json:"url"`
					FileURL string `json:"fileUrl"`
					Text    string `json:"text"`
				} `json:"results"`
			} `json:"data"`
		}
		if err := common.Unmarshal(respBody, &wrapped); err == nil {
			taskResp.TaskID = wrapped.Data.TaskID
			taskResp.Status = wrapped.Data.Status
			taskResp.ErrorCode = wrapped.Data.ErrorCode
			taskResp.ErrorMessage = wrapped.Data.ErrorMessage
			taskResp.Results = wrapped.Data.Results
			if taskResp.Status == "" && wrapped.Code != 0 {
				return &relaycommon.TaskInfo{
					Status: model.TaskStatusFailure,
					Reason: wrapped.Msg,
				}, nil
			}
		}
	}

	taskInfo := &relaycommon.TaskInfo{}
	switch strings.ToUpper(taskResp.Status) {
	case "PENDING":
		taskInfo.Status = model.TaskStatusSubmitted
		taskInfo.Progress = taskcommon.ProgressSubmitted
	case "RUNNING":
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
	case "SUCCESS":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = taskcommon.ProgressComplete
		if len(taskResp.Results) > 0 {
			r := taskResp.Results[0]
			// Prefer media URL; fall back to text content for text-output workflows
			taskInfo.Url = r.URL
			if taskInfo.Url == "" {
				taskInfo.Url = r.FileURL
			}
			if taskInfo.Url == "" && r.Text != "" {
				taskInfo.Url = r.Text
			}
		}
	case "FAIL", "FAILED", "ERROR":
		taskInfo.Status = model.TaskStatusFailure
		if taskResp.ErrorMessage != "" {
			taskInfo.Reason = taskResp.ErrorMessage
		} else {
			taskInfo.Reason = taskResp.ErrorCode
		}
	default:
		return nil, fmt.Errorf("unknown task status: %s", taskResp.Status)
	}
	return taskInfo, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return nil
}

func (a *TaskAdaptor) GetChannelName() string {
	return "runninghub"
}

func extractTaskID(body map[string]any) string {
	if taskID, ok := body["taskId"].(string); ok {
		return taskID
	}
	if data, ok := body["data"].(map[string]any); ok {
		if taskID, ok := data["taskId"].(string); ok {
			return taskID
		}
	}
	return ""
}

func hasInlineResult(body map[string]any) bool {
	status, _ := body["status"].(string)
	if strings.EqualFold(status, "SUCCESS") {
		return true
	}
	if data, ok := body["data"].(map[string]any); ok {
		status, _ = data["status"].(string)
		return strings.EqualFold(status, "SUCCESS")
	}
	return false
}

func isInternalRuntimeKey(key string) bool {
	switch key {
	case "api_path":
		return true
	default:
		return false
	}
}


// validateImageRequest parses a standard OpenAI /v1/images/generations or /v1/images/edits
// request body and converts it to a RunningHub TaskSubmitReq.
// Standard OpenAI fields are mapped as follows:
//   size     -> TaskSubmitReq.Size and metadata["size"]
//   quality  -> metadata["quality"]
//   style    -> metadata["style"]
//   image(s) -> TaskSubmitReq.Images  (for image editing workflows)
//   Extra    -> metadata (any unknown fields pass through as-is)
func (a *TaskAdaptor) validateImageRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
var imgReq dto.ImageRequest
if err := common.UnmarshalBodyReusable(c, &imgReq); err != nil {
return service.TaskErrorWrapper(errors.Wrap(err, "parse_image_request_failed"), "invalid_request", http.StatusBadRequest)
}
if strings.TrimSpace(imgReq.Prompt) == "" {
return service.TaskErrorWrapperLocal(fmt.Errorf("prompt is required"), "invalid_request", http.StatusBadRequest)
}

metadata := map[string]interface{}{}

if imgReq.Size != "" {
metadata["size"] = imgReq.Size
}
if imgReq.Quality != "" {
metadata["quality"] = imgReq.Quality
}
if len(imgReq.Style) > 0 {
var style interface{}
if err := common.Unmarshal(imgReq.Style, &style); err == nil {
metadata["style"] = style
}
}

// Forward any extra / unknown fields directly into metadata so users can pass
// workflow-specific parameters (e.g., resolution, api_path, steps) without
// knowing about RunningHub metadata structure.
for k, v := range imgReq.Extra {
var val interface{}
if err := common.Unmarshal(v, &val); err == nil {
metadata[k] = val
}
}

taskReq := relaycommon.TaskSubmitReq{
Model:    imgReq.Model,
Prompt:   imgReq.Prompt,
Size:     imgReq.Size,
Metadata: metadata,
}

// Parse image field for image-edit workflows.
if len(imgReq.Image) > 0 {
var imageVal interface{}
if err := common.Unmarshal(imgReq.Image, &imageVal); err == nil {
switch v := imageVal.(type) {
case string:
if v != "" {
taskReq.Images = []string{v}
}
case []interface{}:
for _, img := range v {
if s, ok := img.(string); ok && s != "" {
taskReq.Images = append(taskReq.Images, s)
}
}
}
}
}

// api_path in metadata overrides the upstream model name (workflow path).
if apiPath, ok := metadata["api_path"].(string); ok && strings.TrimSpace(apiPath) != "" {
info.UpstreamModelName = strings.TrimSpace(apiPath)
}

hasImage := len(taskReq.Images) > 0 || taskReq.Image != ""
if !strings.HasSuffix(info.UpstreamModelName, "/text-to-image") &&
	!strings.HasSuffix(info.UpstreamModelName, "/image-to-image") {
	if hasImage {
		info.UpstreamModelName = strings.TrimRight(info.UpstreamModelName, "/") + "/image-to-image"
	} else {
		info.UpstreamModelName = strings.TrimRight(info.UpstreamModelName, "/") + "/text-to-image"
	}
}

info.Action = constant.TaskActionImageGenerate
c.Set("task_request", taskReq)
return nil
}

func buildRunningHubURL(baseURL, path string) string {
	base := strings.TrimRight(baseURL, "/")
	p := strings.TrimPrefix(strings.TrimSpace(path), "/")
	if !strings.HasPrefix(p, "openapi/") {
		p = "openapi/v2/" + p
	}
	return base + "/" + p
}
