package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	arkAssetHost    = "ark.cn-beijing.volcengineapi.com"
	arkAssetVersion = "2024-01-01"
	arkAssetService = "ark"
	arkAssetPath    = "/"
)

func arkHostForRegion(region string) string {
	if region == "" || region == "cn-beijing" {
		return arkAssetHost
	}
	return fmt.Sprintf("ark.%s.volcengineapi.com", region)
}

func arkHmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func arkHashSHA256(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func arkNormQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(params[k]))
	}
	return strings.Join(parts, "&")
}

func arkBuildHeaders(accessKey, secretKey, region, host, action, bodyStr string) (http.Header, string) {
	now := time.Now().UTC()
	xDate := now.Format("20060102T150405Z")
	shortDate := xDate[:8]

	queryParams := map[string]string{
		"Action":  action,
		"Version": arkAssetVersion,
	}
	queryStr := arkNormQuery(queryParams)

	xContentSHA256 := arkHashSHA256(bodyStr)
	signedHeaders := "content-type;host;x-content-sha256;x-date"

	canonicalHeaders := strings.Join([]string{
		"content-type:application/json",
		"host:" + host,
		"x-content-sha256:" + xContentSHA256,
		"x-date:" + xDate,
	}, "\n")

	canonicalRequest := strings.Join([]string{
		"POST",
		arkAssetPath,
		queryStr,
		canonicalHeaders,
		"",
		signedHeaders,
		xContentSHA256,
	}, "\n")

	credentialScope := strings.Join([]string{shortDate, region, arkAssetService, "request"}, "/")
	stringToSign := strings.Join([]string{
		"HMAC-SHA256",
		xDate,
		credentialScope,
		arkHashSHA256(canonicalRequest),
	}, "\n")

	kDate := arkHmacSHA256([]byte(secretKey), shortDate)
	kRegion := arkHmacSHA256(kDate, region)
	kService := arkHmacSHA256(kRegion, arkAssetService)
	kSigning := arkHmacSHA256(kService, "request")
	signature := hex.EncodeToString(arkHmacSHA256(kSigning, stringToSign))

	authorization := fmt.Sprintf(
		"HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature,
	)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Host", host)
	headers.Set("X-Content-Sha256", xContentSHA256)
	headers.Set("X-Date", xDate)
	headers.Set("Authorization", authorization)

	return headers, queryStr
}

type arkCreateAssetRequest struct {
	Name        string `json:"Name"`
	GroupId     string `json:"GroupId"`
	URL         string `json:"URL"`
	AssetType   string `json:"AssetType"` // Image | Video
	ProjectName string `json:"ProjectName"`
}

// arkCreateAssetResult is the Result payload returned by the CreateAsset API.
// Ref: https://www.volcengine.com/docs/82379/2333565
type arkCreateAssetResult struct {
	Id string `json:"Id"` // e.g. "asset-20260318071009-xxxxx"
}

type arkCreateAssetResponse struct {
	ResponseMetadata struct {
		RequestId string `json:"RequestId"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"ResponseMetadata"`
	Result *arkCreateAssetResult `json:"Result"`
}

// ArkCreateAsset registers an asset URL in the Volcengine Ark asset system.
// If no credentials are configured it returns the original url as-is.
func ArkCreateAsset(name, assetUrl, assetType, projectName, groupId string) (assetId, assetUri string, err error) {
	accessKey := system_setting.ArkAssetAccessKey
	secretKey := system_setting.ArkAssetSecretKey
	region := system_setting.ArkAssetRegion
	if region == "" {
		region = "cn-beijing"
	}

	// Skip Ark registration if credentials not configured
	if accessKey == "" || secretKey == "" {
		return "", assetUrl, nil
	}

	// GroupId is required by Ark API — skip with error if not configured
	if groupId == "" {
		return "", assetUrl, fmt.Errorf("ark_asset: ArkAssetGroupId is not configured in system settings")
	}

	reqBody := arkCreateAssetRequest{
		Name:        name,
		GroupId:     groupId,
		URL:         assetUrl,
		AssetType:   assetType,
		ProjectName: projectName,
	}
	bodyBytes, err := common.Marshal(reqBody)
	if err != nil {
		return "", assetUrl, fmt.Errorf("ark_asset: marshal request: %w", err)
	}
	bodyStr := string(bodyBytes)

	host := arkHostForRegion(region)

	headers, queryStr := arkBuildHeaders(accessKey, secretKey, region, host, "CreateAsset", bodyStr)

	reqURL := fmt.Sprintf("https://%s%s?%s", host, arkAssetPath, queryStr)
	httpReq, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(bodyStr))
	if err != nil {
		return "", assetUrl, fmt.Errorf("ark_asset: create request: %w", err)
	}
	for k, vs := range headers {
		for _, v := range vs {
			httpReq.Header.Set(k, v)
		}
	}

	resp, err := GetHttpClient().Do(httpReq)
	if err != nil {
		return "", assetUrl, fmt.Errorf("ark_asset: do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", assetUrl, fmt.Errorf("ark_asset: read response: %w", err)
	}

	var arkResp arkCreateAssetResponse
	if err := common.Unmarshal(respBytes, &arkResp); err != nil {
		return "", assetUrl, fmt.Errorf("ark_asset: unmarshal response: %w", err)
	}

	if arkResp.ResponseMetadata.Error != nil {
		return "", assetUrl, fmt.Errorf("ark_asset: api error %s: %s",
			arkResp.ResponseMetadata.Error.Code,
			arkResp.ResponseMetadata.Error.Message,
		)
	}

	if arkResp.Result == nil {
		return "", assetUrl, fmt.Errorf("ark_asset: empty result")
	}

	// URI format per docs: asset://<asset_ID>
	return arkResp.Result.Id, "asset://" + arkResp.Result.Id, nil
}

// --- GetAsset ---

type arkGetAssetRequest struct {
	Id          string `json:"Id"`
	ProjectName string `json:"ProjectName,omitempty"`
}

// arkGetAssetResult mirrors the Result block returned by the GetAsset API.
type arkGetAssetResult struct {
	Id     string `json:"Id"`
	Status string `json:"Status"` // Processing | Active | Failed
}

type arkGetAssetResponse struct {
	ResponseMetadata struct {
		RequestId string `json:"RequestId"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"ResponseMetadata"`
	Result *arkGetAssetResult `json:"Result"`
}

// ArkGetAssetStatus queries the Ark GetAsset API and returns the current Status
// string ("Processing", "Active", or "Failed").
func ArkGetAssetStatus(assetId, projectName string) (status string, err error) {
	accessKey := system_setting.ArkAssetAccessKey
	secretKey := system_setting.ArkAssetSecretKey
	region := system_setting.ArkAssetRegion
	if region == "" {
		region = "cn-beijing"
	}

	if accessKey == "" || secretKey == "" {
		return "", fmt.Errorf("ark_asset: credentials not configured")
	}

	reqBody := arkGetAssetRequest{Id: assetId, ProjectName: projectName}
	bodyBytes, err := common.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ark_asset: marshal get request: %w", err)
	}
	bodyStr := string(bodyBytes)

	host := arkHostForRegion(region)
	headers, queryStr := arkBuildHeaders(accessKey, secretKey, region, host, "GetAsset", bodyStr)

	reqURL := fmt.Sprintf("https://%s%s?%s", host, arkAssetPath, queryStr)
	httpReq, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(bodyStr))
	if err != nil {
		return "", fmt.Errorf("ark_asset: create get request: %w", err)
	}
	for k, vs := range headers {
		for _, v := range vs {
			httpReq.Header.Set(k, v)
		}
	}

	resp, err := GetHttpClient().Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ark_asset: do get request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ark_asset: read get response: %w", err)
	}

	var arkResp arkGetAssetResponse
	if err := common.Unmarshal(respBytes, &arkResp); err != nil {
		return "", fmt.Errorf("ark_asset: unmarshal get response: %w", err)
	}

	if arkResp.ResponseMetadata.Error != nil {
		return "", fmt.Errorf("ark_asset: get api error %s: %s",
			arkResp.ResponseMetadata.Error.Code,
			arkResp.ResponseMetadata.Error.Message,
		)
	}

	if arkResp.Result == nil {
		return "", fmt.Errorf("ark_asset: empty get result")
	}

	return arkResp.Result.Status, nil
}
