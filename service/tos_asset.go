package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func tosHmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(data))
	return h.Sum(nil)
}

func tosHashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func tosSigningKey(secretKey, shortDate, region string) []byte {
	kDate := tosHmacSHA256([]byte("TOS4"+secretKey), shortDate)
	kRegion := tosHmacSHA256(kDate, region)
	kService := tosHmacSHA256(kRegion, "tos")
	return tosHmacSHA256(kService, "request")
}

func tosNormalizeEndpoint(endpoint string) string {
	ep := strings.TrimSpace(endpoint)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	ep = strings.TrimSuffix(ep, "/")
	if idx := strings.Index(ep, "/"); idx >= 0 {
		ep = ep[:idx]
	}
	return ep
}

func tosBuildHost(bucket, endpoint string) string {
	if strings.HasPrefix(endpoint, bucket+".") {
		return endpoint
	}
	return bucket + "." + endpoint
}

func tosCanonicalURI(objectKey string) string {
	cleanKey := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	return "/" + strings.ReplaceAll(url.PathEscape(cleanKey), "%2F", "/")
}

func tosDoSignedRequest(method string, body []byte, objectKey, contentType string) (string, error) {
	accessKey := strings.TrimSpace(system_setting.TosAccessKey)
	secretKey := strings.TrimSpace(system_setting.TosSecretKey)
	region := strings.TrimSpace(system_setting.TosRegion)
	bucket := strings.TrimSpace(system_setting.TosBucket)
	endpoint := tosNormalizeEndpoint(system_setting.TosEndpoint)

	if accessKey == "" || secretKey == "" || region == "" || bucket == "" || endpoint == "" {
		return "", fmt.Errorf("tos: config not complete")
	}
	if strings.TrimSpace(objectKey) == "" {
		return "", fmt.Errorf("tos: object key is empty")
	}

	host := tosBuildHost(bucket, endpoint)
	canonicalURI := tosCanonicalURI(objectKey)

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")
	payloadHash := tosHashSHA256(body)

	headersToSign := map[string]string{
		"host":                 host,
		"x-tos-content-sha256": payloadHash,
		"x-tos-date":           amzDate,
	}

	sortedHeaderKeys := make([]string, 0, len(headersToSign))
	for k := range headersToSign {
		sortedHeaderKeys = append(sortedHeaderKeys, k)
	}
	sort.Strings(sortedHeaderKeys)

	canonicalHeaders := strings.Builder{}
	signedHeaders := strings.Builder{}
	for i, k := range sortedHeaderKeys {
		canonicalHeaders.WriteString(k)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headersToSign[k]))
		canonicalHeaders.WriteString("\n")
		if i > 0 {
			signedHeaders.WriteString(";")
		}
		signedHeaders.WriteString(k)
	}

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		"",
		canonicalHeaders.String(),
		signedHeaders.String(),
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{shortDate, region, "tos", "request"}, "/")
	stringToSign := strings.Join([]string{
		"TOS4-HMAC-SHA256",
		amzDate,
		scope,
		tosHashSHA256([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(tosHmacSHA256(tosSigningKey(secretKey, shortDate, region), stringToSign))
	authorization := fmt.Sprintf(
		"TOS4-HMAC-SHA256 Credential=%s/%s,SignedHeaders=%s,Signature=%s",
		accessKey,
		scope,
		signedHeaders.String(),
		signature,
	)

	reqURL := "https://" + host + canonicalURI
	req, err := http.NewRequest(method, reqURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("tos: create request: %w", err)
	}
	req.Header.Set("Host", host)
	req.Header.Set("X-Tos-Date", amzDate)
	req.Header.Set("X-Tos-Content-Sha256", payloadHash)
	req.Header.Set("Authorization", authorization)
	if method == http.MethodPut {
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tos: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tos: request failed, method=%s status=%d body=%s", method, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return reqURL, nil
}

func TosUploadFile(file io.Reader, filename string, contentType string) (string, error) {
	body, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("tos: read file: %w", err)
	}
	return tosDoSignedRequest(http.MethodPut, body, filename, contentType)
}

func TosDeleteFile(filename string) error {
	_, err := tosDoSignedRequest(http.MethodDelete, nil, filename, "")
	return err
}
