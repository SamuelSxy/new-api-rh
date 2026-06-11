package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	tos "github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func newTosClient() (*tos.ClientV2, error) {
	accessKey := strings.TrimSpace(system_setting.TosAccessKey)
	secretKey := strings.TrimSpace(system_setting.TosSecretKey)
	region := strings.TrimSpace(system_setting.TosRegion)
	endpoint := strings.TrimSpace(system_setting.TosEndpoint)

	if accessKey == "" || secretKey == "" || region == "" || endpoint == "" {
		return nil, fmt.Errorf("tos: config not complete (accessKey/secretKey/region/endpoint required)")
	}
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		endpoint = "https://" + endpoint
	}

	client, err := tos.NewClientV2(endpoint,
		tos.WithRegion(region),
		tos.WithCredentials(tos.NewStaticCredentials(accessKey, secretKey)),
	)
	if err != nil {
		return nil, fmt.Errorf("tos: create client: %w", err)
	}
	return client, nil
}

func tosBucketName() string {
	bucket := strings.TrimSpace(system_setting.TosBucket)
	endpoint := strings.TrimSpace(system_setting.TosEndpoint)
	ep := strings.TrimPrefix(endpoint, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	// If bucket was stored as "ai-gc.tos-cn-beijing.volces.com", extract just the bucket name
	if strings.HasSuffix(bucket, "."+ep) {
		return strings.TrimSuffix(bucket, "."+ep)
	}
	return bucket
}

// TosUploadFile uploads a file to Volcengine TOS and returns the accessible URL.
// If TosPublicRead is enabled, the object is uploaded with public-read ACL so
// upstream model services can fetch it directly. If TosCustomDomain is set,
// that domain is used as the URL prefix instead of the default TOS endpoint.
func TosUploadFile(file io.Reader, objectKey string, contentType string) (string, error) {
	client, err := newTosClient()
	if err != nil {
		return "", err
	}

	bucket := tosBucketName()
	if bucket == "" {
		return "", fmt.Errorf("tos: bucket name is empty")
	}

	basicInput := tos.PutObjectBasicInput{
		Bucket:      bucket,
		Key:         objectKey,
		ContentType: contentType,
	}
	if system_setting.TosPublicRead {
		basicInput.ACL = enum.ACLPublicRead
	}

	input := &tos.PutObjectV2Input{
		PutObjectBasicInput: basicInput,
		Content:             file,
	}

	_, err = client.PutObjectV2(context.Background(), input)
	if err != nil {
		return "", fmt.Errorf("tos: upload failed: %w", err)
	}

	return tosObjectURL(bucket, objectKey), nil
}

// tosObjectURL builds the accessible URL for an object.
// Uses TosCustomDomain when configured, otherwise falls back to the default
// bucket endpoint URL.
func tosObjectURL(bucket, objectKey string) string {
	customDomain := strings.TrimSpace(system_setting.TosCustomDomain)
	if customDomain != "" {
		customDomain = strings.TrimSuffix(customDomain, "/")
		return fmt.Sprintf("%s/%s", customDomain, objectKey)
	}

	endpoint := strings.TrimSpace(system_setting.TosEndpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimSuffix(endpoint, "/")
	return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, objectKey)
}

// TosGetPresignedURL generates a time-limited pre-signed URL for a private object.
// expires specifies how long the URL is valid (max 7 days = 604800 seconds).
func TosGetPresignedURL(objectKey string, expires int64) (string, error) {
	client, err := newTosClient()
	if err != nil {
		return "", err
	}

	bucket := tosBucketName()
	if bucket == "" {
		return "", fmt.Errorf("tos: bucket name is empty")
	}

	output, err := client.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     bucket,
		Key:        objectKey,
		Expires:    expires,
	})
	if err != nil {
		return "", fmt.Errorf("tos: presign failed: %w", err)
	}
	return output.SignedUrl, nil
}

// TosObjectURLForKey returns the URL for an already-uploaded object.
// Useful for reconstructing the URL without re-uploading.
func TosObjectURLForKey(objectKey string) string {
	bucket := tosBucketName()
	return tosObjectURL(bucket, objectKey)
}



// TosDeleteFile deletes a file from Volcengine TOS.
func TosDeleteFile(objectKey string) error {
	client, err := newTosClient()
	if err != nil {
		return err
	}

	bucket := tosBucketName()
	if bucket == "" {
		return fmt.Errorf("tos: bucket name is empty")
	}

	_, err = client.DeleteObjectV2(context.Background(), &tos.DeleteObjectV2Input{
		Bucket: bucket,
		Key:    objectKey,
	})
	if err != nil {
		return fmt.Errorf("tos: delete failed: %w", err)
	}
	return nil
}
