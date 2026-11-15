package xAI

import (
	"fmt"
	"io"
	"net/http"
	"one-api/common/requester"
	"one-api/model"
	"one-api/providers/base"
	"one-api/providers/openai"
	"one-api/types"
	"strings"
)

// 定义供应商工厂
type XAIProviderFactory struct{}

// 创建 XAIProvider
func (f XAIProviderFactory) Create(channel *model.Channel) base.ProviderInterface {
	fmt.Println("Creating XAIProvider for channel:")
	return &XAIProvider{
		OpenAIProvider: openai.OpenAIProvider{
			BaseProvider: base.BaseProvider{
				Config:    getConfig(),
				Channel:   channel,
				Requester: requester.NewHTTPRequester(*channel.Proxy, RequestErrorHandle),
			},
			UsageHandler:        usageHandler,
			RequestHandleBefore: requestHandler,
		},
	}
}

func getConfig() base.ProviderConfig {
	return base.ProviderConfig{
		BaseURL:           "https://api.x.ai",
		ChatCompletions:   "/v1/chat/completions",
		ImagesGenerations: "/v1/images/generations",
		ModelList:         "/v1/models",
	}
}

type XAIProvider struct {
	openai.OpenAIProvider
}

func RequestErrorHandle(resp *http.Response) *types.OpenAIError {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	msg := string(bodyBytes)
	if msg == "" {
		return nil
	}

	return openai.ErrorHandle(&types.OpenAIErrorResponse{
		Error: types.OpenAIError{
			Message: msg,
			Code:    "xAI_error",
		},
	})
}

func usageHandler(usage *types.Usage) (ForcedFormatting bool) {
	usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens

	return true
}

func requestHandler(request *types.ChatCompletionRequest) (errWithCode *types.OpenAIErrorWithStatusCode) {

	// grok-3-mini 系列上游确实不接受 stop / penalties，保留原有清理。
	if strings.HasPrefix(request.Model, "grok-3-mini") || strings.HasPrefix(request.Model, "grok-3-mini-fast") {
		request.Stop = nil
		request.FrequencyPenalty = nil
		request.PresencePenalty = nil
	}

	// grok-4 不再无条件清除 reasoning_effort / stop / penalties：
	// 新版 API 已支持推理强度参数，且无损透传会把客户端原始值送到上游，
	// 主动清掉反而让新特性静默失效。上游若仍不支持会自行返回 400，
	// 此时可在渠道上单独关闭透传。
	return nil
}
