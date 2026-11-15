package relay

import (
	"github.com/gin-gonic/gin"
)

// ListModelsAuto 按客户端请求头返回对应协议的模型列表。
// Anthropic SDK 用 x-api-key + anthropic-version，Gemini SDK 用 x-goog-api-key 或 ?key=，
// 其余按 OpenAI 格式返回。这样 Claude Code / Gemini SDK 把 base_url 指向 /v1 时
// 也能正常拉取模型，而不必改用 /claude 或 /gemini 前缀。
func ListModelsAuto(c *gin.Context) {
	if c.GetHeader("x-api-key") != "" || c.GetHeader("anthropic-version") != "" {
		ListClaudeModelsByToken(c)
		return
	}

	if c.GetHeader("x-goog-api-key") != "" || c.Query("key") != "" {
		ListGeminiModelsByToken(c)
		return
	}

	ListModelsByToken(c)
}
