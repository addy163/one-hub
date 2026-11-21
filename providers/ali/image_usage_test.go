package ali

import (
	"testing"

	"one-api/types"
)

func TestImageTierFromType(t *testing.T) {
	cases := map[string]string{
		"qima_output_2k":  "2k",
		"qima_output_1k":  "1k",
		"qima_input_2k":   "2k",
		"qima_output_4k":  "4k",
		"2K":              "2k",
		"":                "",
		"qima_output_12k": "",
	}
	for in, want := range cases {
		if got := imageTier(in); got != want {
			t.Errorf("imageTier(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestImageTierBySize(t *testing.T) {
	cases := []struct {
		w, h int
		want string
	}{
		{1024, 1024, "1k"},
		{1280, 720, "1k"},
		{2048, 2048, "2k"},
		{2688, 1536, "2k"},
		{4096, 4096, "4k"},
		{0, 0, ""},
	}
	for _, c := range cases {
		if got := imageTierBySize(c.w, c.h); got != c.want {
			t.Errorf("imageTierBySize(%d,%d) = %q, want %q", c.w, c.h, got, c.want)
		}
	}
}

func TestApplyAliImageUsage(t *testing.T) {
	usage := &types.Usage{}
	applyAliImageUsage(usage, &AliUsage{
		OutputImageCount: 1,
		OutputImageType:  "qima_output_2k",
		OutputWidth:      2048,
		OutputHeight:     2048,
	})

	if usage.OutputImages["2k"] != 1 {
		t.Errorf("应记 1 张 2k 输出图，实际 %v", usage.OutputImages)
	}
	if !usage.HasImages() {
		t.Error("HasImages 应返回 true")
	}

	// 流式会重复收到累计值，应覆盖而不是累加
	applyAliImageUsage(usage, &AliUsage{OutputImageCount: 2, OutputImageType: "qima_output_1k"})
	if usage.OutputImages["2k"] != 0 || usage.OutputImages["1k"] != 2 {
		t.Errorf("重复回调应覆盖旧值，实际 %v", usage.OutputImages)
	}

	// 没有类型标签时按宽高兜底
	usage2 := &types.Usage{}
	applyAliImageUsage(usage2, &AliUsage{
		OutputImageCount: 1,
		OutputWidth:      1024,
		OutputHeight:     1024,
		InputImageCount:  2,
		InputImageType:   "qima_input_1k",
	})
	if usage2.OutputImages["1k"] != 1 || usage2.InputImages["1k"] != 2 {
		t.Errorf("按宽高兜底失败，实际 out=%v in=%v", usage2.OutputImages, usage2.InputImages)
	}
}
