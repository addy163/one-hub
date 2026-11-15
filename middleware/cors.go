package middleware

import (
	"net/http"
	"one-api/common/config"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 每次请求读取运行时配置，后台修改白名单后无需重启即可生效。
//
// 安全约定保持不变：
//   - 白名单为空：允许任意来源，但一律不发送 Allow-Credentials（浏览器会拒绝凭据请求）；
//   - 白名单非空：仅回显匹配到的 Origin，是否允许携带凭据由 CorsAllowCredentials 决定；
//   - "*" 与 Allow-Credentials 互斥，避免把管理接口暴露给任意站点。
func CORS() gin.HandlerFunc {
	const allowMethods = "GET, POST, PUT, DELETE, OPTIONS, PATCH"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		origins, allowCredentials := config.CorsSettings()
		allowedOrigin := ""
		if len(origins) == 0 {
			allowedOrigin = "*"
		} else {
			normalized := strings.TrimSuffix(origin, "/")
			for _, item := range origins {
				item = strings.TrimSpace(item)
				if item == "*" {
					allowedOrigin = "*"
					break
				}
				if strings.EqualFold(strings.TrimSuffix(item, "/"), normalized) {
					allowedOrigin = origin
					break
				}
			}
			if allowedOrigin == "" {
				// 不在白名单：不回显任何 CORS 头，由浏览器自行拦截。
				if c.Request.Method == http.MethodOptions {
					c.AbortWithStatus(http.StatusNoContent)
					return
				}
				c.Next()
				return
			}
			c.Writer.Header().Add("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		if allowCredentials && allowedOrigin != "*" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			if requested := c.GetHeader("Access-Control-Request-Headers"); requested != "" {
				c.Header("Access-Control-Allow-Headers", requested)
			} else {
				c.Header("Access-Control-Allow-Headers", "*")
			}
			c.Header("Access-Control-Max-Age", "43200")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
