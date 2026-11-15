package types

import "encoding/json"

// 嵌套对象的无损保真。顶层未知字段由转发层的 mergeExtraBodyFromRawRequest 负责，
// 但 reasoning / audio / response_format / content part 这类嵌套对象里的新字段
// 仍会被固定 struct 吃掉，因此这里给它们各自加上 Extra 兜底。

func (v *ChatReasoning) UnmarshalJSON(data []byte) error {
	type alias ChatReasoning
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = ChatReasoning(decoded)
	v.Extra = CollectExtra(data, decoded)
	return nil
}

func (v ChatReasoning) MarshalJSON() ([]byte, error) {
	type alias ChatReasoning
	return MarshalWithExtra(alias(v), v.Extra)
}

func (v *ChatAudio) UnmarshalJSON(data []byte) error {
	type alias ChatAudio
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = ChatAudio(decoded)
	v.Extra = CollectExtra(data, decoded)
	return nil
}

func (v ChatAudio) MarshalJSON() ([]byte, error) {
	type alias ChatAudio
	return MarshalWithExtra(alias(v), v.Extra)
}

func (v *ChatCompletionResponseFormat) UnmarshalJSON(data []byte) error {
	type alias ChatCompletionResponseFormat
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = ChatCompletionResponseFormat(decoded)
	v.Extra = CollectExtra(data, decoded)
	return nil
}

func (v ChatCompletionResponseFormat) MarshalJSON() ([]byte, error) {
	type alias ChatCompletionResponseFormat
	return MarshalWithExtra(alias(v), v.Extra)
}

func (v *FormatJsonSchema) UnmarshalJSON(data []byte) error {
	type alias FormatJsonSchema
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = FormatJsonSchema(decoded)
	v.Extra = CollectExtra(data, decoded)
	return nil
}

func (v FormatJsonSchema) MarshalJSON() ([]byte, error) {
	type alias FormatJsonSchema
	return MarshalWithExtra(alias(v), v.Extra)
}

func (v *ChatMessagePart) UnmarshalJSON(data []byte) error {
	type alias ChatMessagePart
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = ChatMessagePart(decoded)
	v.Extra = CollectExtra(data, decoded)
	return nil
}

func (v ChatMessagePart) MarshalJSON() ([]byte, error) {
	type alias ChatMessagePart
	return MarshalWithExtra(alias(v), v.Extra)
}

func (v *ChatMessageImageURL) UnmarshalJSON(data []byte) error {
	type alias ChatMessageImageURL
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*v = ChatMessageImageURL(decoded)
	v.Extra = CollectExtra(data, decoded)
	return nil
}

func (v ChatMessageImageURL) MarshalJSON() ([]byte, error) {
	type alias ChatMessageImageURL
	return MarshalWithExtra(alias(v), v.Extra)
}
