package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// UniversalAuth 同时识别 OpenAI / Anthropic / Gemini 三种客户端放置凭据的位置，
// 使 /v1/models 之类的公共端点可以被原生 SDK 直接调用。
// 校验逻辑仍然复用 tokenAuth，不降低任何鉴权强度。
func UniversalAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		key := c.GetHeader("x-api-key")
		if key == "" {
			key = c.GetHeader("x-goog-api-key")
		}
		if key == "" {
			key = c.Query("key")
		}
		if key == "" {
			key = c.Request.Header.Get("Authorization")
		}

		if key == "" && c.GetHeader("Upgrade") == "websocket" {
			protocols := c.Request.Header["Sec-Websocket-Protocol"]
			if len(protocols) > 0 {
				for _, protocol := range strings.Split(protocols[0], ",") {
					protocol = strings.TrimSpace(protocol)
					if strings.HasPrefix(protocol, "openai-insecure-api-key.") {
						key = strings.TrimPrefix(protocol, "openai-insecure-api-key.")
						break
					}
				}
			}
		}

		tokenAuth(c, key)
	}
}
