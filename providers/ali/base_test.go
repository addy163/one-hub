package ali

import (
	"net/http"
	"testing"

	"gorm.io/datatypes"

	"one-api/model"
	"one-api/types"
)

func newChannelWithPlugin(name string, enable bool) *model.Channel {
	plugin := datatypes.NewJSONType(model.PluginType{
		name: {"enable": enable},
	})
	return &model.Channel{Plugin: &plugin}
}

func TestNormalizeCompatBaseURL(t *testing.T) {
	cases := map[string]string{
		"":                                "https://dashscope.aliyuncs.com/compatible-mode",
		"https://dashscope.aliyuncs.com":  "https://dashscope.aliyuncs.com/compatible-mode",
		"https://dashscope.aliyuncs.com/": "https://dashscope.aliyuncs.com/compatible-mode",
		"https://dashscope.aliyuncs.com/compatible-mode":                     "https://dashscope.aliyuncs.com/compatible-mode",
		"https://dashscope.aliyuncs.com/compatible-mode/":                    "https://dashscope.aliyuncs.com/compatible-mode",
		"https://dashscope.aliyuncs.com/compatible-mode/v1":                  "https://dashscope.aliyuncs.com/compatible-mode",
		"https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions": "https://dashscope.aliyuncs.com/compatible-mode",
		"https://cn-beijing.maas.qianwenaiapi.com/compatible-mode":           "https://cn-beijing.maas.qianwenaiapi.com/compatible-mode",
		"https://cn-beijing.maas.qianwenaiapi.com":                           "https://cn-beijing.maas.qianwenaiapi.com/compatible-mode",
		"https://dashscope-intl.aliyuncs.com":                                "https://dashscope-intl.aliyuncs.com/compatible-mode",
		// 用户已写明路径时不追加
		"https://proxy.example.com/ali": "https://proxy.example.com/ali",
	}

	for in, want := range cases {
		if got := normalizeCompatBaseURL(in); got != want {
			t.Errorf("normalizeCompatBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUseNativeAPI(t *testing.T) {
	if !useNativeAPI(&model.Channel{}) {
		t.Error("默认应走原生 DashScope 接口")
	}
	if useNativeAPI(newChannelWithPlugin("use_openai_api", true)) {
		t.Error("勾选 OpenAI 兼容接口时应走兼容模式")
	}
	if !useNativeAPI(newChannelWithPlugin("use_openai_api", false)) {
		t.Error("兼容接口未勾选时应保持原生")
	}
	// 兼容模式是后来才变成可选项的，老渠道里存的 use_native_api 仍然尊重
	if !useNativeAPI(newChannelWithPlugin("use_native_api", true)) {
		t.Error("显式开启原生接口时应走原生")
	}
	if !useNativeAPI(newChannelWithPlugin("web_search", true)) {
		t.Error("web_search 只有原生支持，应强制原生")
	}
}

func TestNormalizeNativeBaseURL(t *testing.T) {
	cases := map[string]string{
		"":                                "https://dashscope.aliyuncs.com",
		"https://dashscope.aliyuncs.com":  "https://dashscope.aliyuncs.com",
		"https://dashscope.aliyuncs.com/": "https://dashscope.aliyuncs.com",
		"https://dashscope.aliyuncs.com/compatible-mode":            "https://dashscope.aliyuncs.com",
		"https://cn-beijing.maas.qianwenaiapi.com/compatible-mode/": "https://cn-beijing.maas.qianwenaiapi.com",
		"https://cn-beijing.maas.qianwenaiapi.com/v1":               "https://cn-beijing.maas.qianwenaiapi.com",
		"https://cn-beijing.maas.qianwenaiapi.com":                  "https://cn-beijing.maas.qianwenaiapi.com",
	}

	for in, want := range cases {
		if got := normalizeNativeBaseURL(in); got != want {
			t.Errorf("normalizeNativeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func newAliProviderWithBase(base string) *AliProvider {
	provider := &AliProvider{}
	provider.Channel = &model.Channel{}
	if base != "" {
		b := base
		provider.Channel.BaseURL = &b
	}
	return provider
}

func TestNativeChatPathCache(t *testing.T) {
	provider := newAliProviderWithBase("")
	// 没什么配置时，新模型默认走统一消息格式接口
	if got := provider.nativeChatPath("qwen4-max"); got != multimodalChatPath {
		t.Errorf("未探测的模型应默认多模态接口，实际 %q", got)
	}
	if got := alternateChatPath(multimodalChatPath); got != textChatPath {
		t.Errorf("多模态接口的备选应是文本接口，实际 %q", got)
	}
	if got := alternateChatPath(textChatPath); got != multimodalChatPath {
		t.Errorf("文本接口的备选应是多模态接口，实际 %q", got)
	}

	// 探测成功后按「地址 + 模型」缓存，不同地址互不影响
	provider.rememberNativeChatPath("qwen-flash", textChatPath)
	if got := provider.nativeChatPath("qwen-flash"); got != textChatPath {
		t.Errorf("命中缓存时应返回 %q，实际 %q", textChatPath, got)
	}
	other := newAliProviderWithBase("https://other.example.com")
	if got := other.nativeChatPath("qwen-flash"); got != multimodalChatPath {
		t.Errorf("换地址后不应复用其他地址的缓存，实际 %q", got)
	}
}

func TestIsNativeEndpointMismatch(t *testing.T) {
	mismatch := &types.OpenAIErrorWithStatusCode{
		StatusCode: http.StatusBadRequest,
		OpenAIError: types.OpenAIError{
			Code:    "InvalidParameter",
			Message: "url error, please check url！",
		},
	}
	if !isNativeEndpointMismatch(mismatch) {
		t.Error("官方 url error 应判定为接口不匹配")
	}

	notFound := &types.OpenAIErrorWithStatusCode{StatusCode: http.StatusNotFound}
	if !isNativeEndpointMismatch(notFound) {
		t.Error("网关 404 应判定为接口不匹配")
	}

	other := &types.OpenAIErrorWithStatusCode{
		StatusCode:  http.StatusBadRequest,
		OpenAIError: types.OpenAIError{Code: "InvalidParameter", Message: "Role must be user or assistant"},
	}
	if isNativeEndpointMismatch(other) {
		t.Error("普通参数错误不应触发接口切换")
	}
	if isNativeEndpointMismatch(nil) {
		t.Error("nil 错误不应触发接口切换")
	}
}
