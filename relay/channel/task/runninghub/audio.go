package runninghub

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	runningHubPollInterval = 2 * time.Second
	runningHubPollTimeout  = 120 * time.Second
)

// Adaptor implements channel.Adaptor for RunningHub, supporting /v1/audio/speech.
// Internally it submits an async task and polls until the audio URL is ready,
// then streams the audio back to the caller — appearing synchronous to the client.
type Adaptor struct {
	baseURL string
	apiKey  string
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	a.apiKey = info.ApiKey
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	path := strings.TrimSpace(info.UpstreamModelName)
	if path == "" {
		return "", errors.New("runninghub: model path is required")
	}
	return buildRunningHubURL(a.baseURL, path), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	req.Set("Authorization", "Bearer "+a.apiKey)
	req.Set("Content-Type", "application/json")
	return nil
}

// ConvertAudioRequest converts an OpenAI-style TTS request into a RunningHub
// workflow request body. Fields are placed at the top level of the body,
// matching how RunningHub's other task endpoints expect parameters.
//
// Standard fields (voice → voice_id, speed, response_format) are mapped first.
// Any additional provider-specific fields can be passed via request.Metadata
// as a flat JSON object — they are merged in and override the defaults.
func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	// Build flat top-level body (mirrors BuildRequestBody in TaskAdaptor)
	body := map[string]any{
		"text":  request.Input,
	}
	if request.Voice != "" {
		body["voice_id"] = request.Voice
	}
	if request.Speed != nil {
		body["speed"] = *request.Speed
	}
	if request.ResponseFormat != "" {
		body["response_format"] = request.ResponseFormat
	}

	// Merge caller-supplied metadata on top (flat, top-level overrides)
	if len(request.Metadata) > 0 {
		var extra map[string]any
		if err := common.Unmarshal(request.Metadata, &extra); err != nil {
			return nil, fmt.Errorf("runninghub: invalid metadata: %w", err)
		}
		for k, v := range extra {
			body[k] = v
		}
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("runninghub: marshal request failed: %w", err)
	}
	return bytes.NewReader(data), nil
}

// DoRequest submits the TTS task to RunningHub, polls until it completes, then
// fetches the resulting audio and returns its HTTP response for DoResponse to
// stream back to the client.
func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	// Step 1: Submit the task
	submitURL, err := a.GetRequestURL(info)
	if err != nil {
		return nil, err
	}

	submitReq, err := http.NewRequest(http.MethodPost, submitURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("runninghub: create submit request failed: %w", err)
	}
	submitReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	submitReq.Header.Set("Content-Type", "application/json")

	submitResp, err := getRunningHubClient().Do(submitReq)
	if err != nil {
		return nil, fmt.Errorf("runninghub: submit request failed: %w", err)
	}
	defer submitResp.Body.Close()

	submitBody, err := io.ReadAll(submitResp.Body)
	if err != nil {
		return nil, fmt.Errorf("runninghub: read submit response failed: %w", err)
	}

	if submitResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runninghub: submit request returned status %d: %s", submitResp.StatusCode, string(submitBody))
	}

	// Parse taskId from submit response
	var submitResult map[string]any
	if err = common.Unmarshal(submitBody, &submitResult); err != nil {
		return nil, fmt.Errorf("runninghub: unmarshal submit response failed: %w", err)
	}

	taskID := extractTaskID(submitResult)
	if taskID == "" {
		if errCode, ok := submitResult["errorCode"].(string); ok && errCode != "" {
			errMsg, _ := submitResult["errorMessage"].(string)
			if errMsg == "" {
				errMsg = errCode
			}
			return nil, fmt.Errorf("runninghub: submit error: %s", errMsg)
		}
		return nil, errors.New("runninghub: no taskId in submit response")
	}

	logger.LogInfo(c, fmt.Sprintf("runninghub: submitted audio task %s", taskID))

	// Step 2: Poll until done
	ta := &TaskAdaptor{}
	deadline := time.Now().Add(runningHubPollTimeout)

	for time.Now().Before(deadline) {
		time.Sleep(runningHubPollInterval)

		pollResp, err := ta.FetchTask(a.baseURL, a.apiKey, map[string]any{"taskId": taskID}, "")
		if err != nil {
			logger.LogWarn(c, fmt.Sprintf("runninghub: poll task %s failed: %v", taskID, err))
			continue
		}

		pollBody, err := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()
		if err != nil {
			logger.LogWarn(c, fmt.Sprintf("runninghub: read poll response failed: %v", err))
			continue
		}

		taskInfo, err := ta.ParseTaskResult(pollBody)
		if err != nil {
			logger.LogWarn(c, fmt.Sprintf("runninghub: parse task result failed: %v", err))
			continue
		}

		switch taskInfo.Status {
		case model.TaskStatusSuccess:
			if taskInfo.Url == "" {
				return nil, errors.New("runninghub: task succeeded but no audio URL returned")
			}
			// Step 3: Fetch the audio file and return its response
			audioResp, err := getRunningHubClient().Get(taskInfo.Url)
			if err != nil {
				return nil, fmt.Errorf("runninghub: fetch audio from %s failed: %w", taskInfo.Url, err)
			}
			return audioResp, nil

		case model.TaskStatusFailure:
			return nil, fmt.Errorf("runninghub: task %s failed: %s", taskID, taskInfo.Reason)

		default:
			// Still in progress, keep polling
			logger.LogInfo(c, fmt.Sprintf("runninghub: task %s status=%s, polling...", taskID, taskInfo.Status))
		}
	}

	return nil, fmt.Errorf("runninghub: audio task %s timed out after %s", taskID, runningHubPollTimeout)
}

// DoResponse streams the audio response body to the client.
func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if resp == nil {
		return &dto.Usage{PromptTokens: info.GetEstimatePromptTokens()}, nil
	}
	defer resp.Body.Close()

	// Forward content-type header
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}
	c.Header("Content-Type", contentType)

	audioData, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("runninghub: read audio response failed: %w", readErr),
			types.ErrorCodeReadResponseBodyFailed,
			http.StatusInternalServerError,
		)
	}

	c.Data(http.StatusOK, contentType, audioData)

	promptTokens := info.GetEstimatePromptTokens()
	return &dto.Usage{
		PromptTokens: promptTokens,
		TotalTokens:  promptTokens,
	}, nil
}

// --- Unsupported methods required by channel.Adaptor interface ---

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertOpenAIRequest not supported")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertRerankRequest not supported")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertEmbeddingRequest not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertImageRequest not supported")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertOpenAIResponsesRequest not supported")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertClaudeRequest not supported")
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("runninghub: ConvertGeminiRequest not supported")
}

func (a *Adaptor) GetModelList() []string {
	return nil
}

func (a *Adaptor) GetChannelName() string {
	return "runninghub"
}
