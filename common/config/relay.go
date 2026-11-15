package config

import "github.com/spf13/viper"

// 中继请求的参数保真策略。
//
// 老版本使用固定 struct 解析并重新序列化请求，任何未定义的字段都会被静默丢弃，
// 导致 service_tier / store / metadata / prompt_cache_* / extra_body / 各家厂商扩展参数
// 以及图片、音频的新参数无法到达上游。new-api 用 json.RawMessage 解决同一问题，
// 这里复用程序里已有的 mergeExtraBodyFromRawRequest：以程序处理后的请求为底，
// 只补回原始 body 中「程序未定义」的字段；已定义字段一律以程序结果为准，
// 否则会把程序主动清理掉的参数（例如 o 系列的 temperature）重新带给上游。
var (
	// PassthroughExtraBody 是全局总闸，默认开启。
	// 设为 false 可全局关闭，仅对显式勾选了 allow_extra_body 的渠道保留透传。
	PassthroughExtraBody = true

	// PassthroughNativeProtocol 控制是否注册 Claude / Gemini 原生 SDK 的标准路径别名
	// （/v1/messages、/v1beta/models...）。关闭后仍可使用原有的
	// /claude/v1/messages 与 /gemini/:version/models/:model 前缀，不影响既有客户端。
	// 原生请求本身通过 ClaudeRequest / GeminiChatRequest 的 Extra 保真，
	// 不会经过 OpenAI→原生的二次转换。
	PassthroughNativeProtocol = true
)

func InitRelay() {
	PassthroughExtraBody = viper.GetBool("passthrough_extra_body")
	PassthroughNativeProtocol = viper.GetBool("passthrough_native_protocol")
}
