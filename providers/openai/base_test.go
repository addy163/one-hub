package openai

import (
	"net/http/httptest"
	"one-api/common/config"
	"one-api/providers/base"
	"one-api/types"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func newProviderWithRawBody(t *testing.T, raw any) *OpenAIProvider {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if raw != nil {
		switch value := raw.(type) {
		case []byte:
			c.Set(config.GinRequestBodyKey, value)
		case string:
			c.Set(config.GinRequestBodyKey, []byte(value))
		default:
			t.Fatalf("unsupported raw body type %T", raw)
		}
	}
	return &OpenAIProvider{BaseProvider: base.BaseProvider{Context: c}}
}

func TestMergeExtraBodyFromRawRequest(t *testing.T) {
	cases := []struct {
		name    string
		raw     any
		initial map[string]interface{}
		want    map[string]interface{}
	}{
		{
			name:    "no cached raw body keeps request untouched",
			raw:     nil,
			initial: map[string]interface{}{"model": "gpt-4o"},
			want:    map[string]interface{}{"model": "gpt-4o"},
		},
		{
			name:    "invalid json keeps request untouched",
			raw:     `{broken`,
			initial: map[string]interface{}{"model": "gpt-4o"},
			want:    map[string]interface{}{"model": "gpt-4o"},
		},
		{
			name: "only unknown fields are added",
			raw: `{
				"model":"attacker-model",
				"temperature":0.9,
				"stream":true,
				"service_tier":"flex",
				"x_vendor_flag":true,
				"x_trace":{"id":"t-1"},
				"future_param":"v1"
			}`,
			// service_tier / metadata / prompt_cache_key 等已在 ChatCompletionRequest
			// 中声明，属于「程序已定义」，由程序序列化负责，不在这里重复补回。
			initial: map[string]interface{}{"model": "gpt-4o", "messages": []any{}},
			want: map[string]interface{}{
				"model":         "gpt-4o",
				"messages":      []any{},
				"x_vendor_flag": true,
				"x_trace":       map[string]interface{}{"id": "t-1"},
				"future_param":  "v1",
			},
		},
		{
			name:    "existing keys are never overwritten",
			raw:     `{"service_tier":"from-client","custom_flag":true}`,
			initial: map[string]interface{}{"service_tier": "from-program"},
			want: map[string]interface{}{
				"service_tier": "from-program",
				"custom_flag":  true,
			},
		},
		{
			name:    "program-cleared known fields stay cleared",
			raw:     `{"model":"gpt-4o","temperature":0.7,"stop":["END"]}`,
			initial: map[string]interface{}{"model": "gpt-4o"},
			want:    map[string]interface{}{"model": "gpt-4o"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := newProviderWithRawBody(t, tc.raw)
			got := provider.mergeExtraBodyFromRawRequest(tc.initial, &types.ChatCompletionRequest{})
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
