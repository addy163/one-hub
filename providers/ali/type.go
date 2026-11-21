package ali

import (
	"fmt"
	"strings"

	"one-api/types"
)

type AliError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"request_id"`
}

type AliUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	// 作图模型的 usage 没有 token，只有张数与档位。
	// 形如 {"output_width":2048,"output_height":2048,"output_image_count":1,
	//       "output_image_type":"qima_output_2k","input_image_count":0}
	OutputWidth      int    `json:"output_width"`
	OutputHeight     int    `json:"output_height"`
	OutputImageCount int    `json:"output_image_count"`
	OutputImageType  string `json:"output_image_type"`
	InputImageCount  int    `json:"input_image_count"`
	InputImageType   string `json:"input_image_type"`
}

// imageTier 从上游的 type 字段（如 qima_output_2k）里取出档位名。
func imageTier(imageType string) string {
	lower := strings.ToLower(imageType)
	for _, tier := range []string{"1k", "2k", "4k"} {
		if lower == tier || strings.HasSuffix(lower, "_"+tier) {
			return tier
		}
	}
	return ""
}

// imageTierBySize 在没有 type 字段时按像素面积兜底分档。
// 1024 见方（约 1.05M 像素）算 1K，2048 见方与 2688×1536（约 4.1M）算 2K，再大算 4K。
func imageTierBySize(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	pixels := width * height
	switch {
	case pixels <= 1500*1000:
		return "1k"
	case pixels <= 8*1000*1000:
		return "2k"
	default:
		return "4k"
	}
}

type AliMessage struct {
	Content any    `json:"content"`
	Role    string `json:"role"`
}

type AliMessagePart struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

type AliInput struct {
	// Prompt  string       `json:"prompt"`
	Messages []AliMessage `json:"messages"`
}

type AliParameters struct {
	TopP              float64 `json:"top_p,omitempty"`
	TopK              int     `json:"top_k,omitempty"`
	Seed              uint64  `json:"seed,omitempty"`
	EnableSearch      bool    `json:"enable_search,omitempty"`
	IncrementalOutput bool    `json:"incremental_output,omitempty"`
	ResultFormat      string  `json:"result_format,omitempty"`
	EnableThinking    *bool   `json:"enable_thinking,omitempty"` // qwen3 thinking switch

	// 作图参数，仅在客户端显式传入时下发
	Size           string `json:"size,omitempty"` // 形如 1024*1024，不传则用上游默认（2048*2048）
	N              int    `json:"n,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	PromptExtend   *bool  `json:"prompt_extend,omitempty"`
	Watermark      *bool  `json:"watermark,omitempty"`
}

type AliChatRequest struct {
	Model      string        `json:"model"`
	Input      AliInput      `json:"input"`
	Parameters AliParameters `json:"parameters,omitempty"`
}

type AliChoice struct {
	FinishReason string                      `json:"finish_reason"`
	Message      types.ChatCompletionMessage `json:"message"`
}

type AliOutput struct {
	Choices      []types.ChatCompletionChoice `json:"choices"`
	FinishReason string                       `json:"finish_reason,omitempty"`
}

func (o *AliOutput) ToChatCompletionChoices() []types.ChatCompletionChoice {
	for i := range o.Choices {
		content := o.Choices[i].Message.Content
		if _, ok := content.(string); ok {
			continue
		}

		// 原生接口返回的是 parts 数组：文本在 text 字段，作图模型（qwen-image 等）
		// 把图片放在 image 字段。转成纯文本，图片用 Markdown 保留下来，
		// 否则上层拿到空的 content，图片地址直接丢了。
		list, ok := content.([]any)
		if !ok {
			o.Choices[i].Message.Content = o.Choices[i].Message.ParseContent()
			continue
		}

		parts := make([]string, 0, len(list))
		for _, item := range list {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
			if image, ok := part["image"].(string); ok && image != "" {
				parts = append(parts, fmt.Sprintf("![image](%s)", image))
			}
		}
		o.Choices[i].Message.Content = strings.Join(parts, "\n")
	}
	return o.Choices
}

type AliChatResponse struct {
	Output AliOutput `json:"output"`
	Usage  AliUsage  `json:"usage"`
	AliError
}

type AliEmbeddingRequest struct {
	Model string `json:"model"`
	Input struct {
		Texts []string `json:"texts"`
	} `json:"input"`
	Parameters *struct {
		TextType string `json:"text_type,omitempty"`
	} `json:"parameters,omitempty"`
}

type AliEmbedding struct {
	Embedding []float64 `json:"embedding"`
	TextIndex int       `json:"text_index"`
}

type AliEmbeddingResponse struct {
	Output struct {
		Embeddings []AliEmbedding `json:"embeddings"`
	} `json:"output"`
	Usage AliUsage `json:"usage"`
	AliError
}
