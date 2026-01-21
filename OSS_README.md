# EZ-Encrypt-Middleware OSS 订阅分发功能

## 新增功能
本次更新添加了 **OSS 订阅分发** 功能，支持将用户订阅配置同步上传到 S3 兼容的对象存储（如 Cloudflare R2、AWS S3、阿里云 OSS）。

## 配置说明

在 `.env` 文件中添加以下配置（默认关闭）：

```env
# 6. ────────────────────────── OSS 订阅分发配置（可选）───────────────────
OSS_ENABLED=true                           # 是否启用 OSS 订阅分发
OSS_ENDPOINT=https://xxx.r2.cloudflarestorage.com  # S3 兼容端点
OSS_REGION=auto                            # 区域，R2 使用 auto
OSS_ACCESS_KEY=your_access_key             # Access Key ID
OSS_SECRET_KEY=your_secret_key             # Secret Access Key
OSS_BUCKET=your_bucket_name                # Bucket 名称
OSS_CDN_DOMAIN=https://sub-cdn.bnsrf.com   # CDN 域名
```

## 工作原理

1. **触发时机**：当用户请求订阅 (`/client/subscribe`) 时
2. **自动上传**：中间件会将后端返回的配置文件异步上传到 OSS
3. **文件路径**：`/subs/{user_token}`
4. **元数据**：同时写入 `subscription-userinfo` 元数据（流量信息）

## CDN 配置

1. 创建 Cloudflare R2 Bucket
2. 配置公开访问或绑定自定义域名
3. 将域名填入 `OSS_CDN_DOMAIN`

## 依赖安装

```bash
cd /path/to/EZ-Encrypt-Middleware
go mod tidy
```

## 客户端适配

客户端（FlClash）需要：
1. 支持从 OSS 链接获取订阅
2. 读取响应头中的 `subscription-userinfo` 获取流量信息

## 文件变更

- `oss/oss.go` - 新增 OSS 服务
- `config/config.go` - 新增 OSS 配置项
- `proxy/proxy.go` - 订阅请求时上传到 OSS
- `main.go` - 初始化 OSS 客户端
- `.env` - 新增 OSS 配置模板
- `go.mod` - 新增 AWS SDK 依赖
