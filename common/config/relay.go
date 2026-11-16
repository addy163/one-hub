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

// ShouldPassthroughExtraBody 判定单个渠道是否透传客户端未定义的字段。
//
// 判定顺序（三层，不要调换）：
//  1. disable_extra_body 是渠道级否决票，优先级最高，勾选后该渠道一律不透传。
//     仅在上游对未知字段返回 400 时才需要勾选。
//  2. 全局 passthrough_extra_body 默认开启。开启时，所有未被否决的渠道都透传。
//  3. 全局关闭时，allow_extra_body 可作为单渠道例外重新开启。
//
// 注意：全局开启时 allow_extra_body 不参与判定。它是「全局关闭时的白名单」，
// 而不是「必须勾选才透传」的开关；历史数据里 allow_extra_body=0 不代表用户
// 主动关闭过透传，因此不能把它当成否决票使用。
func ShouldPassthroughExtraBody(disableForChannel, allowForChannel bool) bool {
	if disableForChannel {
		return false
	}
	return PassthroughExtraBodyEnabled() || allowForChannel
}
