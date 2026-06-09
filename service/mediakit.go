package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
)

const mediakitBaseURL = "https://mediakit.cn-beijing.volces.com"

type mediakitEnhanceRequest struct {
	VideoURL    string `json:"video_url"`
	ToolVersion string `json:"tool_version"`
	Scene       string `json:"scene"`
	Resolution  string `json:"resolution"`
}

type mediakitSubmitResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type mediakitStatusResponse struct {
	TaskID  string          `json:"task_id"`
	Status  string          `json:"status"`
	Result  json.RawMessage `json:"result"`   // 可能是字符串(URL)或对象
	Message string          `json:"message"`  // 失败原因
	Code    int             `json:"code"`
}

// SubmitEnhanceVideoTask 向 Mediakit 提交视频超分任务，返回 taskID。
func SubmitEnhanceVideoTask(videoURL, targetResolution, toolVersion, scene, apiKey string) (string, error) {
	payload := mediakitEnhanceRequest{
		VideoURL:    videoURL,
		ToolVersion: toolVersion,
		Scene:       scene,
		Resolution:  targetResolution,
	}

	bodyBytes, err := common.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal mediakit request failed: %w", err)
	}

	url := mediakitBaseURL + "/api/v1/tools/enhance-video"
	req, err := buildMediakitRequest(http.MethodPost, url, apiKey, bodyBytes)
	if err != nil {
		return "", err
	}

	client, err := GetHttpClientWithProxy("")
	if err != nil {
		return "", fmt.Errorf("get http client failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("mediakit enhance-video request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read mediakit response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mediakit enhance-video returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result mediakitSubmitResponse
	if err := common.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal mediakit submit response failed: %w", err)
	}

	if result.TaskID == "" {
		return "", fmt.Errorf("mediakit enhance-video returned empty task_id, body: %s", string(respBody))
	}

	return result.TaskID, nil
}

// GetEnhanceVideoTaskStatus 查询 Mediakit 超分任务状态。
// 返回 status ("pending"/"running"/"succeeded"/"failed") 和 videoURL（成功时非空）。
func GetEnhanceVideoTaskStatus(taskID, apiKey string) (status string, videoURL string, err error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s", mediakitBaseURL, taskID)
	req, err := buildMediakitRequest(http.MethodGet, url, apiKey, nil)
	if err != nil {
		return "", "", err
	}

	client, err := GetHttpClientWithProxy("")
	if err != nil {
		return "", "", fmt.Errorf("get http client failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("mediakit task status request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read mediakit status response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("mediakit task status returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result mediakitStatusResponse
	if err := common.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("unmarshal mediakit status response failed: %w, body: %s", err, string(respBody))
	}

	// result 字段可能是字符串(URL)或对象 {video_url: "..."}
	if len(result.Result) > 0 && string(result.Result) != "null" {
		raw := string(result.Result)
		if raw[0] == '"' {
			// 字符串形式："https://..."
			if err := json.Unmarshal(result.Result, &videoURL); err != nil {
				videoURL = raw[1 : len(raw)-1] // 去掉引号的 fallback
			}
		} else if raw[0] == '{' {
			// 对象形式：{"video_url": "https://..."}
			var obj struct {
				VideoURL string `json:"video_url"`
				URL      string `json:"url"`
			}
			if err := json.Unmarshal(result.Result, &obj); err == nil {
				if obj.VideoURL != "" {
					videoURL = obj.VideoURL
				} else {
					videoURL = obj.URL
				}
			}
		}
	}

	return result.Status, videoURL, nil
}

func buildMediakitRequest(method, url, apiKey string, bodyBytes []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("new mediakit request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return req, nil
}
