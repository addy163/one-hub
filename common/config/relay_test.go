package config

import (
	"strconv"
	"testing"
)

// TestShouldPassthroughExtraBody 固化三层透传语义，防止后续改动把
// 「全局总闸 / 渠道否决票 / 全局关闭时的白名单」三者关系弄反。
func TestShouldPassthroughExtraBody(t *testing.T) {
	original := PassthroughExtraBodyEnabled()
	t.Cleanup(func() {
		_ = SetRuntimeOption("PassthroughExtraBody", strconv.FormatBool(original))
	})

	cases := []struct {
		name    string
		global  bool
		disable bool
		allow   bool
		want    bool
	}{
		// 全局开启：默认全透传，这是线上 52 个 allow_extra_body=0 渠道的实际状态
		{name: "global on, legacy allow=0 still passes", global: true, disable: false, allow: false, want: true},
		{name: "global on, allow=1 passes", global: true, disable: false, allow: true, want: true},

		// 全局开启 + 渠道否决：否决票优先级最高
		{name: "global on, channel disable wins", global: true, disable: true, allow: false, want: false},
		{name: "global on, disable beats allow", global: true, disable: true, allow: true, want: false},

		// 全局关闭：默认全不透传
		{name: "global off, default blocked", global: false, disable: false, allow: false, want: false},

		// 全局关闭 + 渠道白名单：单渠道例外
		{name: "global off, channel allow re-enables", global: false, disable: false, allow: true, want: true},
		{name: "global off, disable beats allow", global: false, disable: true, allow: true, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := SetRuntimeOption("PassthroughExtraBody", strconv.FormatBool(tc.global)); err != nil {
				t.Fatalf("set global switch: %v", err)
			}
			got := ShouldPassthroughExtraBody(tc.disable, tc.allow)
			if got != tc.want {
				t.Errorf("global=%v disable=%v allow=%v: got %v, want %v",
					tc.global, tc.disable, tc.allow, got, tc.want)
			}
		})
	}
}
