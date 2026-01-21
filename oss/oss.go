package oss

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"EZ-Encrypt-Middleware/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// OssClient is the S3-compatible client for OSS operations
type OssClient struct {
	client *s3.Client
	bucket string
}

var Client *OssClient

// InitOss initializes the OSS client
func InitOss() error {
	if !config.AppConfig.OSSEnabled {
		log.Println("OSS 功能未启用")
		return nil
	}

	if config.AppConfig.OSSEndpoint == "" || config.AppConfig.OSSAccessKey == "" || config.AppConfig.OSSSecretKey == "" {
		log.Println("OSS 配置不完整，跳过初始化")
		return nil
	}

	// Create custom resolver for S3-compatible endpoints (Cloudflare R2, Aliyun OSS, etc.)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               config.AppConfig.OSSEndpoint,
			SigningRegion:     config.AppConfig.OSSRegion,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(config.AppConfig.OSSRegion),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AppConfig.OSSAccessKey,
			config.AppConfig.OSSSecretKey,
			"",
		)),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return fmt.Errorf("无法加载 AWS 配置: %w", err)
	}

	Client = &OssClient{
		client: s3.NewFromConfig(cfg),
		bucket: config.AppConfig.OSSBucket,
	}

	log.Printf("OSS 客户端初始化成功: %s", config.AppConfig.OSSEndpoint)
	return nil
}

// SubscriptionUserInfo represents user subscription info for the header
type SubscriptionUserInfo struct {
	Upload   int64 // bytes uploaded
	Download int64 // bytes downloaded
	Total    int64 // total transfer limit
	Expire   int64 // expiration timestamp
}

// FormatHeader formats the user info as subscription-userinfo header value
func (s *SubscriptionUserInfo) FormatHeader() string {
	return fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
		s.Upload, s.Download, s.Total, s.Expire)
}

// UploadSubscription uploads a user's subscription config to OSS
// Returns the CDN URL of the uploaded file
func UploadSubscription(token string, configContent []byte, userInfo *SubscriptionUserInfo) (string, error) {
	if Client == nil {
		return "", fmt.Errorf("OSS 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := fmt.Sprintf("subs/%s", token)

	// Prepare metadata
	metadata := map[string]string{
		"subscription-userinfo": userInfo.FormatHeader(),
	}

	// Upload to S3/OSS
	_, err := Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(Client.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(configContent),
		ContentType:  aws.String("text/yaml; charset=utf-8"),
		CacheControl: aws.String("max-age=300"), // 5 minutes cache
		Metadata:     metadata,
	})

	if err != nil {
		return "", fmt.Errorf("上传到 OSS 失败: %w", err)
	}

	// Return CDN URL
	cdnURL := fmt.Sprintf("%s/%s", config.AppConfig.OSSCdnDomain, key)
	log.Printf("订阅配置已上传到 OSS: %s", cdnURL)

	return cdnURL, nil
}

// IsEnabled returns whether OSS feature is enabled
func IsEnabled() bool {
	return Client != nil && config.AppConfig.OSSEnabled
}
