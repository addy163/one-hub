package model

import (
	"testing"

	"one-api/common/config"
)

// gpt-image 系列按 token 计费，默认价必须是「图像输入为基准 + 文本输入用 extra 修正」，
// 否则文本输入会按图像价多收 60%。
func TestDefaultPriceForGPTImage(t *testing.T) {
	want := map[string]struct {
		input  float64
		output float64
		text   float64
	}{
		"gpt-image-2.6-sunburst": {4, 15, 0.625},
		"gpt-image-2.6-flare":    {4, 15, 0.625},
		"gpt-image-2.5-sunburst": {4, 15, 0.625},
		"gpt-image-2.5-flare":    {4, 15, 0.625},
		"gpt-image-2":            {4, 15, 0.625},
	}

	found := make(map[string]bool)
	for _, price := range GetDefaultPrice() {
		cfg, ok := want[price.Model]
		if !ok {
			continue
		}
		found[price.Model] = true

		if price.Type != TokensPriceType {
			t.Errorf("%s 应按 token 计费，实际 %s", price.Model, price.Type)
		}
		if price.Input != cfg.input || price.Output != cfg.output {
			t.Errorf("%s 倍率应为 input=%v output=%v，实际 %v/%v",
				price.Model, cfg.input, cfg.output, price.Input, price.Output)
		}
		if price.ExtraRatios == nil {
			t.Errorf("%s 缺少 extra_ratios，文本输入会按图像价计费", price.Model)
			continue
		}
		if got := price.ExtraRatios.Data()[config.UsageExtraInputTextTokens]; got != cfg.text {
			t.Errorf("%s 文本输入倍率应为 %v，实际 %v", price.Model, cfg.text, got)
		}
	}

	for model := range want {
		if !found[model] {
			t.Errorf("默认价格表缺少 %s", model)
		}
	}
}

// 已弃用的模型不应再出现在默认价里，否则开启自动同步会被重新加回在售价格表。
func TestDeprecatedImageModelsNotInDefaultPrice(t *testing.T) {
	deprecated := []string{"gpt-image-1", "gpt-image-1-mini", "gpt-image-1.5"}
	for _, price := range GetDefaultPrice() {
		for _, m := range deprecated {
			if price.Model == m {
				t.Errorf("%s 已弃用，不应出现在默认价格表中", m)
			}
		}
	}
}
