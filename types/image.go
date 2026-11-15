package types

import (
	"encoding/json"
	"mime/multipart"
)

type ImageRequest struct {
	Prompt           string  `json:"prompt,omitempty" binding:"required"`
	Model            string  `json:"model,omitempty"`
	N                int     `json:"n,omitempty"`
	Quality          string  `json:"quality,omitempty"`
	Size             string  `json:"size,omitempty"`
	Style            string  `json:"style,omitempty"`
	ResponseFormat   string  `json:"response_format,omitempty"`
	User             string  `json:"user,omitempty"`
	AspectRatio      *string `json:"aspect_ratio,omitempty"`
	OutputQuality    *int    `json:"output_quality,omitempty"`
	SafetyTolerance  *string `json:"safety_tolerance,omitempty"`
	PromptUpsampling *string `json:"prompt_upsampling,omitempty"`

	Background        *string `json:"background,omitempty"`
	Moderation        *string `json:"moderation,omitempty"`
	OutputCompression *int    `json:"output_compression,omitempty"`
	OutputFormat      *string `json:"output_format,omitempty"`

	// 新模型常见参数（gpt-image-1/1.5、flux、seedream、imagen 等）
	Seed          *int  `json:"seed,omitempty"`
	PartialImages *int  `json:"partial_images,omitempty"`
	Stream        *bool `json:"stream,omitempty"`
	Watermark     *bool `json:"watermark,omitempty"`

	InputFidelity any `json:"input_fidelity,omitempty"`
	Image         any `json:"image,omitempty"`
	Images        any `json:"images,omitempty"`
	Mask          any `json:"mask,omitempty"`
	Metadata      any `json:"metadata,omitempty"`

	// Extra 保存所有未在上述结构体中定义的字段，序列化时原样写回，
	// 避免上游新增参数时被调用方静默丢弃。
	Extra map[string]json.RawMessage `json:"-"`
}

func (r *ImageRequest) UnmarshalJSON(data []byte) error {
	type alias ImageRequest
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = ImageRequest(decoded)
	r.Extra = CollectExtra(data, decoded)
	return nil
}

func (r ImageRequest) MarshalJSON() ([]byte, error) {
	type alias ImageRequest
	return MarshalWithExtra(alias(r), r.Extra)
}

type ImageResponse struct {
	Created  any                      `json:"created,omitempty"`
	Data     []ImageResponseDataInner `json:"data,omitempty"`
	Usage    *ResponsesUsage          `json:"usage,omitempty"`
	Metadata any                      `json:"metadata,omitempty"`
}

type ImageResponseDataInner struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageEditRequest struct {
	Image          *multipart.FileHeader   `form:"image"`
	Images         []*multipart.FileHeader `form:"image[]"`
	Mask           *multipart.FileHeader   `form:"mask"`
	Model          string                  `form:"model"`
	Prompt         string                  `form:"prompt"`
	N              int                     `form:"n"`
	Size           string                  `form:"size"`
	ResponseFormat string                  `form:"response_format"`
	User           string                  `form:"user"`

	Quality           string `form:"quality"`
	Background        string `form:"background"`
	Moderation        string `form:"moderation"`
	OutputFormat      string `form:"output_format"`
	OutputCompression string `form:"output_compression"`
	InputFidelity     string `form:"input_fidelity"`
	Seed              string `form:"seed"`
	Watermark         string `form:"watermark"`
}
