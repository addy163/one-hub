package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

type sampleRelayRequest struct {
	Model   string `json:"model,omitempty"`
	Stream  bool   `json:"stream,omitempty"`
	Ignored string `json:"-"`
	hidden  string
}

func TestKnownJSONFields(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want map[string]bool
	}{
		{"struct", sampleRelayRequest{}, map[string]bool{"model": true, "stream": true}},
		{"pointer", &sampleRelayRequest{}, map[string]bool{"model": true, "stream": true}},
		{"image request", ImageRequest{}, map[string]bool{
			"prompt": true, "model": true, "aspect_ratio": true, "output_format": true,
			"partial_images": true, "image": true,
		}},
		{"non struct", 42, map[string]bool{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := KnownJSONFields(tc.in)
			for key := range tc.want {
				if !got[key] {
					t.Errorf("expected known field %q, got %v", key, got)
				}
			}
			if got["Ignored"] || got["hidden"] || got["-"] {
				t.Errorf("private / ignored fields must not be known: %v", got)
			}
			if tc.name == "struct" && len(got) != 2 {
				t.Errorf("unexpected field count: %v", got)
			}
		})
	}
}

func TestCollectExtra(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantKeys   []string
		wantAbsent []string
		wantNil    bool
	}{
		{
			name:       "unknown fields are collected",
			body:       `{"model":"gpt-4o","stream":true,"service_tier":"flex","metadata":{"tenant":"a"},"safety_identifier":"u-1"}`,
			wantKeys:   []string{"service_tier", "metadata", "safety_identifier"},
			wantAbsent: []string{"model", "stream"},
		},
		{
			name:    "all fields known returns nil",
			body:    `{"model":"gpt-4o","stream":false}`,
			wantNil: true,
		},
		{
			name:    "invalid json returns nil",
			body:    `{not-json`,
			wantNil: true,
		},
		{
			name:    "json array returns nil",
			body:    `[1,2,3]`,
			wantNil: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req sampleRelayRequest
			extra := CollectExtra([]byte(tc.body), req)
			if tc.wantNil {
				if extra != nil {
					t.Fatalf("expected nil extra, got %v", extra)
				}
				return
			}
			for _, key := range tc.wantKeys {
				if _, ok := extra[key]; !ok {
					t.Errorf("missing preserved field %q in %v", key, extra)
				}
			}
			for _, key := range tc.wantAbsent {
				if _, ok := extra[key]; ok {
					t.Errorf("known field %q must not be collected", key)
				}
			}
		})
	}
}

func TestMarshalWithExtra(t *testing.T) {
	req := sampleRelayRequest{Model: "gpt-4o", Stream: true}

	cases := []struct {
		name  string
		extra map[string]json.RawMessage
		check func(t *testing.T, fields map[string]json.RawMessage)
	}{
		{
			name:  "nil extra behaves like json.Marshal",
			extra: nil,
			check: func(t *testing.T, fields map[string]json.RawMessage) {
				if len(fields) != 2 {
					t.Fatalf("unexpected fields: %v", fields)
				}
			},
		},
		{
			name: "unknown fields are written back",
			extra: map[string]json.RawMessage{
				"service_tier": json.RawMessage(`"flex"`),
				"metadata":     json.RawMessage(`{"tenant":"a"}`),
			},
			check: func(t *testing.T, fields map[string]json.RawMessage) {
				if string(fields["service_tier"]) != `"flex"` {
					t.Errorf("service_tier lost: %v", fields)
				}
				if string(fields["metadata"]) != `{"tenant":"a"}` {
					t.Errorf("nested metadata lost: %v", fields)
				}
			},
		},
		{
			name: "program-owned fields always win",
			extra: map[string]json.RawMessage{
				"model":        json.RawMessage(`"attacker-model"`),
				"service_tier": json.RawMessage(`"flex"`),
			},
			check: func(t *testing.T, fields map[string]json.RawMessage) {
				if string(fields["model"]) != `"gpt-4o"` {
					t.Errorf("program result must win: %v", fields)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := MarshalWithExtra(req, tc.extra)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			tc.check(t, fields)
		})
	}
}

func TestImageRequestPreservesUnknownFields(t *testing.T) {
	body := []byte(`{
		"prompt":"a cat",
		"model":"gpt-image-1",
		"size":"1024x1024",
		"person_generation":"dont_allow",
		"safety_setting":{"category":"HARM_CATEGORY_HARASSMENT","threshold":"BLOCK_LOW_AND_ABOVE"},
		"vendor_note":123
	}`)

	var req ImageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if req.Prompt != "a cat" || req.Model != "gpt-image-1" {
		t.Fatalf("known fields decoded incorrectly: %+v", req)
	}
	for _, key := range []string{"person_generation", "safety_setting", "vendor_note"} {
		if _, ok := req.Extra[key]; !ok {
			t.Errorf("unknown field %q not collected", key)
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip map[string]json.RawMessage
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if string(roundTrip["person_generation"]) != `"dont_allow"` {
		t.Errorf("person_generation lost after marshal: %s", data)
	}
	if !reflect.DeepEqual(
		mustUnmarshal(t, body, "safety_setting"),
		mustUnmarshal(t, data, "safety_setting"),
	) {
		t.Errorf("nested safety_setting changed: %s", data)
	}
	if string(roundTrip["vendor_note"]) != "123" {
		t.Errorf("numeric unknown field lost: %s", data)
	}

	// 程序已定义的字段仍以结构体结果为准
	req.Model = "flux-1.1-pro"
	data, _ = json.Marshal(req)
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if string(roundTrip["model"]) != `"flux-1.1-pro"` {
		t.Errorf("program-owned model must win: %s", data)
	}
}

func mustUnmarshal(t *testing.T, data []byte, key string) any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	return fields[key]
}
