package proxy

import (
	"EZ-Encrypt-Middleware/config"
	"EZ-Encrypt-Middleware/oss"
	"EZ-Encrypt-Middleware/utils"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyHandler handles incoming requests and forwards them to the backend API
func ProxyHandler(c *gin.Context) {
	fullPath := c.Request.URL.Path

	encodedPath := strings.TrimPrefix(fullPath, "/")

	iv := c.GetHeader("X-IV")
	if iv == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少X-IV请求头"})
		return
	}

	encryptedPath, err := base64.URLEncoding.DecodeString(encodedPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的Base64编码路径"})
		return
	}

	decryptedPath := utils.Decryption(string(encryptedPath), iv)

	backendURL := config.AppConfig.BackendAPIURL

	targetURL := backendURL + "/api/v1" + decryptedPath

	target, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无效的目标URL"})
		return
	}

	// 检测是否是订阅请求 (需要特殊处理)
	isSubscribeRequest := strings.Contains(decryptedPath, "/client/subscribe")

	proxy := httputil.NewSingleHostReverseProxy(target)

	timeout := 30 * time.Second
	if config.AppConfig.RequestTimeout != "" {
		if t, err := time.ParseDuration(config.AppConfig.RequestTimeout + "ms"); err == nil {
			timeout = t
		}
	}

	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// 修改请求以转发到解密后的路径
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		req.URL.RawQuery = target.RawQuery
		req.Host = target.Host

		req.Header.Del("X-IV")

		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		req = req.WithContext(ctx)
		defer cancel()
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Max-Age")

		// 处理订阅请求：添加 subscription-userinfo header 并上传到 OSS
		if isSubscribeRequest && resp.StatusCode == http.StatusOK {
			// 读取原始响应头中的用户信息
			subscriptionUserInfo := resp.Header.Get("subscription-userinfo")
			
			// 如果后端返回了 subscription-userinfo，直接透传
			// 如果没有，尝试从其他 header 构建
			if subscriptionUserInfo == "" {
				// 尝试从 V2Board 标准 header 构建
				upload := resp.Header.Get("profile-update-interval")
				if upload == "" {
					// 尝试使用 content-disposition 中的信息或其他方式
					// 这里保持简单，如果没有就不添加
				}
			}

			// 如果 OSS 启用，上传配置到 OSS
			if oss.IsEnabled() && resp.StatusCode == http.StatusOK {
				// 读取响应体
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("读取响应体失败: %v", err)
					return nil
				}
				// 重新设置响应体供后续使用
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// 从 URL 中提取 token
				token := c.Query("token")
				if token == "" {
					// 尝试从解密路径中提取
					if strings.Contains(decryptedPath, "token=") {
						parts := strings.Split(decryptedPath, "token=")
						if len(parts) > 1 {
							token = strings.Split(parts[1], "&")[0]
						}
					}
				}

				if token != "" && len(bodyBytes) > 0 {
					// 解析 subscription-userinfo header
					userInfo := parseSubscriptionUserInfo(subscriptionUserInfo)

					go func() {
						_, uploadErr := oss.UploadSubscription(token, bodyBytes, userInfo)
						if uploadErr != nil {
							log.Printf("OSS 上传失败: %v", uploadErr)
						}
					}()
				}
			}
		}

		return nil
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// parseSubscriptionUserInfo parses the subscription-userinfo header
// Format: upload=111; download=222; total=333; expire=444
func parseSubscriptionUserInfo(headerValue string) *oss.SubscriptionUserInfo {
	info := &oss.SubscriptionUserInfo{}

	if headerValue == "" {
		return info
	}

	parts := strings.Split(headerValue, ";")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value, _ := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)

		switch key {
		case "upload":
			info.Upload = value
		case "download":
			info.Download = value
		case "total":
			info.Total = value
		case "expire":
			info.Expire = value
		}
	}

	return info
}

