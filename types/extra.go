package types

import (
	"encoding/json"
	"reflect"
	"strings"
)

// 固定 struct 反序列化会静默丢弃未知字段，这是老版本相对 new-api 的主要差距。
// 下面两个工具给请求 DTO 提供类似 json.RawMessage 的无损保真能力：
// CollectExtra 收下所有未定义字段，MarshalWithExtra 在序列化时原样写回。
// 程序已经处理过的字段（model、stream 等）始终以 struct 的值为准。

// CollectExtra 返回 data 中不属于 v 已定义 json 字段的键值对。
func CollectExtra(data []byte, v any) map[string]json.RawMessage {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	known := KnownJSONFields(v)
	extra := make(map[string]json.RawMessage)
	for k, value := range raw {
		if !known[k] {
			extra[k] = value
		}
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

// MarshalWithExtra 序列化 v，并把 extra 中未被 v 定义的字段合并回去。
func MarshalWithExtra(v any, extra map[string]json.RawMessage) ([]byte, error) {
	base, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(extra) == 0 {
		return base, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(base, &fields); err != nil {
		return nil, err
	}
	for k, value := range extra {
		if _, exists := fields[k]; !exists {
			fields[k] = value
		}
	}
	return json.Marshal(fields)
}

// KnownJSONFields 返回 v 上所有显式声明的 json 字段名（含 omitempty 字段）。
// 供转发层区分「程序已定义、由程序负责」与「程序未定义、需原样保留」的字段。
func KnownJSONFields(v any) map[string]bool {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fields := make(map[string]bool)
	if t == nil || t.Kind() != reflect.Struct {
		return fields
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = true
		}
	}
	return fields
}
