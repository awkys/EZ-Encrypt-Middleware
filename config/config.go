package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                      string
	BackendAPIURL             string
	AESKey                    string
	CORSOrigin                string
	AllowedOrigins            string
	RequestTimeout            string
	EnableLogging             string
	DebugMode                 string
	AllowedPaymentNotifyPaths string
	PathPrefix                string

	// OSS 配置 (可选，支持 S3/R2/Aliyun OSS)
	OSSEnabled   bool   // 是否启用 OSS 订阅分发
	OSSEndpoint  string // S3 兼容端点 (如 https://xxx.r2.cloudflarestorage.com)
	OSSRegion    string // 区域 (如 auto, us-east-1)
	OSSAccessKey string // Access Key ID
	OSSSecretKey string // Secret Access Key
	OSSBucket    string // Bucket 名称
	OSSCdnDomain string // CDN 域名 (如 https://sub-cdn.bnsrf.com)
}

var AppConfig *Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("警告: 未找到 .env 文件，将使用环境变量")
	}

	AppConfig = &Config{
		Port:                      getEnv("PORT", "3000"),
		BackendAPIURL:             getEnv("BACKEND_API_URL", ""),
		AESKey:                    getEnv("AES_KEY", ""),
		CORSOrigin:                getEnv("CORS_ORIGIN", "*"),
		AllowedOrigins:            getEnv("ALLOWED_ORIGINS", "*"),
		RequestTimeout:            getEnv("REQUEST_TIMEOUT", "30000"),
		EnableLogging:             getEnv("ENABLE_LOGGING", "false"),
		DebugMode:                 getEnv("DEBUG_MODE", "false"),
		AllowedPaymentNotifyPaths: getEnv("ALLOWED_PAYMENT_NOTIFY_PATHS", ""),
		PathPrefix:                getEnv("PATH_PREFIX", ""),

		// OSS 配置
		OSSEnabled:   getEnv("OSS_ENABLED", "false") == "true",
		OSSEndpoint:  getEnv("OSS_ENDPOINT", ""),
		OSSRegion:    getEnv("OSS_REGION", "auto"),
		OSSAccessKey: getEnv("OSS_ACCESS_KEY", ""),
		OSSSecretKey: getEnv("OSS_SECRET_KEY", ""),
		OSSBucket:    getEnv("OSS_BUCKET", ""),
		OSSCdnDomain: getEnv("OSS_CDN_DOMAIN", ""),
	}

	if AppConfig.BackendAPIURL == "" {
		log.Fatal("错误: BACKEND_API_URL 未在 .env 文件中设置")
	}

	if AppConfig.AESKey == "" {
		log.Fatal("错误: AES_KEY 未在 .env 文件中设置")
	}

	// OSS 配置日志
	if AppConfig.OSSEnabled {
		log.Printf("OSS 订阅分发已启用: %s", AppConfig.OSSCdnDomain)
	}

	log.Println("配置加载成功")
}


func (c *Config) GetAllowedOrigins() []string {
	if c.AllowedOrigins == "*" {
		return []string{"*"}
	}

	origins := strings.Split(c.AllowedOrigins, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}
	return origins
}

func (c *Config) IsOriginAllowed(origin string) bool {
	allowedOrigins := c.GetAllowedOrigins()

	// If wildcard is set, allow all origins
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true
	}

	// Check if origin is in allowed list
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return true
		}
	}

	return false
}

func (c *Config) GetAllowedPaymentNotifyPaths() []string {
	if c.AllowedPaymentNotifyPaths == "" {
		return []string{}
	}

	paths := strings.Split(c.AllowedPaymentNotifyPaths, ",")
	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
	}
	return paths
}

func (c *Config) IsPaymentNotifyPath(path string) bool {
	allowedPaths := c.GetAllowedPaymentNotifyPaths()

	// If no paths are configured, return false
	if len(allowedPaths) == 0 {
		return false
	}

	// Check if path is in allowed list
	for _, allowed := range allowedPaths {
		if allowed == path {
			return true
		}
	}

	return false
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
