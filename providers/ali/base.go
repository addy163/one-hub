package ali

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"one-api/common/requester"
	"one-api/model"
	"one-api/providers/base"
	"one-api/providers/openai"
	"one-api/types"
)

const (
	OpenaiBaseURL = "https://dashscope.aliyuncs.com/compatible-mode"
	AliBaseURL    = "https://dashscope.aliyuncs.com"

	// 原生聊天接口有两套，模型挂在哪套上没有规律：
	// 老的 qwen-flash / qwen-turbo 只在 text-generation，
	// qwen3.8-omni、qwen4、deepseek 等新模型只在 multimodal-generation。
	textChatPath       = "/api/v1/services/aigc/text-generation/generation"
	multimodalChatPath = "/api/v1/services/aigc/multimodal-generation/generation"
)

// nativeChatPathCache 记录每个「地址 + 模型」实际可用的原生聊天接口。
//
// DashScope 在模型与接口不匹配时只回一句含糊的 "url error, please check url！"，
// 无法从模型名推断（qwen4-max 走多模态、qwen-flash 走文本）。所以首次请求
// 失败后回退另一个接口，成功的组合缓存下来，后续同模型请求直接命中。
var nativeChatPathCache sync.Map

func nativeCacheKey(baseURL, model string) string {
	return strings.ToLower(strings.TrimSuffix(baseURL, "/")) + "|" + strings.ToLower(model)
}

// nativeChatPath 返回该模型优先尝试的原生聊天接口。
//
// 默认 multimodal-generation：当前新模型（omni / qwen4 / deepseek 等）都在这个
// 统一消息格式接口下，老的文本模型首次失败后会自动回退到 text-generation。
func (p *AliProvider) nativeChatPath(model string) string {
	if path, ok := nativeChatPathCache.Load(nativeCacheKey(p.GetBaseURL(), model)); ok {
		if s, ok := path.(string); ok && s != "" {
			return s
		}
	}
	return multimodalChatPath
}

func (p *AliProvider) rememberNativeChatPath(model, path string) {
	nativeChatPathCache.Store(nativeCacheKey(p.GetBaseURL(), model), path)
}

func alternateChatPath(path string) string {
	if path == multimodalChatPath {
		return textChatPath
	}
	return multimodalChatPath
}

// isNativeEndpointMismatch 判断错误是否为「模型与原生接口不匹配」。
//
// 官方与百炼网关都返回 400 + InvalidParameter + "url error, please check url！"；
// 有的自建网关没挂某个原生接口时直接 404，同样值得换另一个接口再试一次。
func isNativeEndpointMismatch(errWithCode *types.OpenAIErrorWithStatusCode) bool {
	if errWithCode == nil {
		return false
	}
	if errWithCode.Code == "InvalidParameter" {
		msg := strings.ToLower(errWithCode.Message)
		if strings.Contains(msg, "url error") || strings.Contains(msg, "check url") {
			return true
		}
	}
	return errWithCode.StatusCode == http.StatusNotFound
}

// pluginEnabled 读取渠道插件里的 enable 开关
func pluginEnabled(channel *model.Channel, name string) bool {
	if channel == nil || channel.Plugin == nil {
		return false
	}
	plugin := channel.Plugin.Data()
	param, ok := plugin[name]
	if !ok {
		return false
	}
	enable, ok := param["enable"].(bool)
	return ok && enable
}

// useNativeAPI 判断是否走 DashScope 原生协议（/api/v1/services/aigc/...）
//
// 默认走原生：官方文档、百炼专属网关以及 qwen-image 这类作图/多模态模型
// 都以原生协议为准，原生聊天接口又统一在 multimodal-generation 下，
// 覆盖面比兼容模式大（Web 搜索、思考模式、图像输出都只有原生有）。
//
// 需要兼容协议时按渠道勾选「使用 OpenAI 兼容接口」即可；此外原生请求
// 打不通（接口不存在 / 404）时会自动退回兼容模式，专有网关不必额外配置。
func useNativeAPI(channel *model.Channel) bool {
	if pluginEnabled(channel, "use_native_api") {
		return true
	}
	if pluginEnabled(channel, "use_openai_api") {
		return false
	}
	return true
}

// normalizeCompatBaseURL 把渠道地址归一化成 compatible-mode 基址。
//
// 兼容模式下请求地址是 {base}/v1/chat/completions，而用户常见的三种写法
// （只填域名、填 .../compatible-mode、填完整 endpoint）都会拼错，这里统一：
//
//	空                        -> https://dashscope.aliyuncs.com/compatible-mode
//	https://host              -> https://host/compatible-mode
//	https://host/compatible-mode[/v1][/chat/completions] -> https://host/compatible-mode
//
// 带其他路径的地址视为用户已写明，只做尾部清理，不再追加 compatible-mode。
func normalizeCompatBaseURL(raw string) string {
	base := strings.TrimSpace(raw)
	if base == "" {
		return OpenaiBaseURL
	}

	base = strings.TrimRight(base, "/")
	for _, suffix := range []string{"/chat/completions", "/completions", "/v1"} {
		base = strings.TrimSuffix(base, suffix)
	}
	base = strings.TrimRight(base, "/")
	if base == "" {
		return OpenaiBaseURL
	}

	if !strings.HasSuffix(base, "/compatible-mode") {
		if u, err := url.Parse(base); err == nil && (u.Path == "" || u.Path == "/") {
			base += "/compatible-mode"
		}
	}

	return base
}

// normalizeNativeBaseURL 去掉地址里的 compatible-mode / v1 尾巴，得到原生接口基址。
//
// 用户常常「先按兼容模式填了 .../compatible-mode，再勾选原生接口」，
// 直接拼原生路径会变成 .../compatible-mode/api/v1/... 而 404，这里统一剥掉。
func normalizeNativeBaseURL(raw string) string {
	base := strings.TrimSpace(raw)
	if base == "" {
		return AliBaseURL
	}

	base = strings.TrimRight(base, "/")
	for _, suffix := range []string{"/compatible-mode", "/v1"} {
		base = strings.TrimSuffix(base, suffix)
	}
	base = strings.TrimRight(base, "/")
	if base == "" {
		return AliBaseURL
	}

	return base
}

// 定义供应商工厂
type AliProviderFactory struct{}

type AliProvider struct {
	openai.OpenAIProvider

	UseOpenaiAPI bool
}

// 创建 AliProvider
// 原生模式：https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation
// 兼容模式：https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions
func (f AliProviderFactory) Create(channel *model.Channel) base.ProviderInterface {
	useOpenaiAPI := !useNativeAPI(channel)

	// 两种模式对渠道地址的写法要求不同：兼容模式要有 /compatible-mode，
	// 原生模式不能带。不归一化的话，两个开关一混就 404。
	rawBase := channel.GetBaseURL()
	normalized := normalizeCompatBaseURL(rawBase)
	if !useOpenaiAPI {
		normalized = normalizeNativeBaseURL(rawBase)
	}
	if normalized != rawBase {
		copied := *channel
		copied.BaseURL = &normalized
		channel = &copied
	}

	provider := &AliProvider{
		OpenAIProvider: openai.OpenAIProvider{
			BaseProvider: base.BaseProvider{
				Config:  getConfig(useOpenaiAPI),
				Channel: channel,
			},
			StreamEscapeJSON:     true,
			SupportStreamOptions: true,
		},
		UseOpenaiAPI: useOpenaiAPI,
	}

	if useOpenaiAPI {
		provider.Requester = requester.NewHTTPRequester(*channel.Proxy, openai.RequestErrorHandle)
	} else {
		provider.Requester = requester.NewHTTPRequester(*channel.Proxy, requestErrorHandle)
	}

	return provider
}

func getConfig(useOpenaiAPI bool) base.ProviderConfig {
	if useOpenaiAPI {
		return base.ProviderConfig{
			BaseURL:           OpenaiBaseURL,
			ChatCompletions:   "/v1/chat/completions",
			Embeddings:        "/v1/embeddings",
			ImagesGenerations: "/v1/images/generations",
			ModelList:         "/v1/models",
		}
	}

	return base.ProviderConfig{
		BaseURL:         AliBaseURL,
		ChatCompletions: textChatPath,
		Embeddings:      "/api/v1/services/embeddings/text-embedding/text-embedding",
		ModelList:       "/v1/models",
	}
}

// compatProvider 返回一个等价的、走 OpenAI 兼容协议的 provider。
//
// 用于原生接口整条打不通时的兜底：有些自建网关只挂 /compatible-mode，
// 默认走原生会连续 404，这里自动退回兼容协议，用户不必再去勾开关。
func (p *AliProvider) compatProvider() *AliProvider {
	channel := p.Channel
	normalized := normalizeCompatBaseURL(channel.GetBaseURL())
	copiedChannel := *channel
	copiedChannel.BaseURL = &normalized

	provider := &AliProvider{
		OpenAIProvider: p.OpenAIProvider,
		UseOpenaiAPI:   true,
	}
	provider.Config = getConfig(true)
	provider.Channel = &copiedChannel
	provider.Requester = requester.NewHTTPRequester(*copiedChannel.Proxy, openai.RequestErrorHandle)

	return provider
}

// 请求错误处理
func requestErrorHandle(resp *http.Response) *types.OpenAIError {
	aliError := &AliError{}
	err := json.NewDecoder(resp.Body).Decode(aliError)
	if err != nil {
		return nil
	}

	return errorHandle(aliError)
}

// 错误处理
func errorHandle(aliError *AliError) *types.OpenAIError {
	if aliError.Code == "" {
		return nil
	}
	return &types.OpenAIError{
		Message: aliError.Message,
		Type:    aliError.Code,
		Param:   aliError.RequestId,
		Code:    aliError.Code,
	}
}

// 获取请求头
func (p *AliProvider) GetRequestHeaders() (headers map[string]string) {
	headers = make(map[string]string)
	p.CommonRequestHeaders(headers)
	headers["Authorization"] = fmt.Sprintf("Bearer %s", p.Channel.Key)
	if p.Channel.Other != "" {
		headers["X-DashScope-Plugin"] = p.Channel.Other
	}

	return headers
}
