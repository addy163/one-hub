package middleware

import (
	"one-api/common/logger"
	"one-api/metrics"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gin-gonic/gin"
)

func SetUpLogger(server *gin.Engine) {

	server.Use(GinzapWithConfig())
}

// 旧实现只匹配 "key=sk-" 前缀且只替换第一处，导致新版 token key、
// 密码重置 token、OAuth code/state、Turnstile 响应等明文进入访问日志。
var sensitiveQueryKeys = map[string]bool{
	"key": true, "token": true, "access_token": true, "code": true,
	"state": true, "secret": true, "signature": true, "sign": true,
	"password": true, "turnstile": true, "aff": true, "api_key": true,
	"accesskey": true, "session": true, "email_verification_code": true,
}

func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "&")
	for i, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || kv[1] == "" {
			continue
		}
		if sensitiveQueryKeys[strings.ToLower(kv[0])] {
			parts[i] = kv[0] + "=***"
		}
	}
	return strings.Join(parts, "&")
}

func GinzapWithConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := redactQuery(c.Request.URL.RawQuery)
		c.Next()
		end := time.Now()
		latency := end.Sub(start)
		requestID := c.GetString(logger.RequestIdKey)
		userID := c.GetInt("id")

		fields := []zapcore.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.Int("user_id", userID),
			zap.String("original_model", c.GetString("original_model")),
			zap.String("new_model", c.GetString("new_model")),
			zap.Int("token_id", c.GetInt("token_id")),
			zap.String("token_name", c.GetString("token_name")),
			zap.Int("channel_id", c.GetInt("channel_id")),
		}

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			for _, e := range c.Errors.Errors() {
				logger.Logger.Error(e, fields...)
			}
		} else {
			logger.Logger.Info("GIN request", fields...)
		}
		metrics.RecordHttp(c, latency)
	}
}
