package relay_util

import (
	"testing"

	"gorm.io/datatypes"

	"one-api/common/config"
	"one-api/model"
	"one-api/types"
)

func newImageQuota(priceType string, groupRatio float64, tiers map[string]model.PriceImageTier) *Quota {
	conf := datatypes.NewJSONType(model.PriceImagePrices{Enabled: true, Tiers: tiers})
	q := &Quota{
		price:      model.Price{Type: priceType, ImagePrices: &conf},
		groupRatio: groupRatio,
	}
	return q
}

// 0.5 元/张 对应倍率 0.5/0.014 = 35.714，乘以 1000 得 ≈35714 配额
const quotaPerRMB = 1000 / model.RMBRate // 71428.57 配额/元

func nearly(t *testing.T, got int, wantRMB float64) {
	t.Helper()
	want := int(wantRMB * quotaPerRMB)
	// 每个档位单独向上取整，误差累积几档，给 10 的容差足够
	if got < want-10 || got > want+10 {
		t.Errorf("配额 %d 与 %.2f 元不符（期望约 %d）", got, wantRMB, want)
	}
}

func TestImageQuotaByTier(t *testing.T) {
	tiers := map[string]model.PriceImageTier{
		"1k": {Input: 0.02 / model.RMBRate, Output: 0.25 / model.RMBRate},
		"2k": {Input: 0.02 / model.RMBRate, Output: 0.5 / model.RMBRate},
	}
	q := newImageQuota(model.TokensPriceType, 1, tiers)
	q.imageOutputs = map[string]int{"2k": 1}

	// token 为 0 时不能被当成异常清零，否则作图就是免费的
	quota := q.GetTotalQuota(0, 0, nil)
	nearly(t, quota, 0.5)
}

func TestImageQuotaMixedTiersAndInputs(t *testing.T) {
	tiers := map[string]model.PriceImageTier{
		"1k": {Input: 0.02 / model.RMBRate, Output: 0.25 / model.RMBRate},
		"2k": {Input: 0.02 / model.RMBRate, Output: 0.5 / model.RMBRate},
	}
	q := newImageQuota(model.TokensPriceType, 1, tiers)
	q.imageOutputs = map[string]int{"1k": 2, "2k": 1}
	q.imageInputs = map[string]int{"2k": 3}

	quota := q.GetTotalQuota(0, 0, nil)
	// 2×0.25 + 1×0.5 + 3×0.02 = 1.06 元
	nearly(t, quota, 1.06)
}

func TestImageQuotaGroupRatio(t *testing.T) {
	tiers := map[string]model.PriceImageTier{"2k": {Output: 0.5 / model.RMBRate}}
	q := newImageQuota(model.TokensPriceType, 2, tiers)
	q.imageOutputs = map[string]int{"2k": 1}

	nearly(t, q.GetTotalQuota(0, 0, nil), 1.0)
}

func TestImageQuotaReplacesTimesPrice(t *testing.T) {
	tiers := map[string]model.PriceImageTier{"2k": {Output: 0.5 / model.RMBRate}}
	q := newImageQuota(model.TimesPriceType, 1, tiers)
	q.inputRatio = 5 // 固定价 5 倍率
	q.imageOutputs = map[string]int{"2k": 1}

	// 配了图片档位价时以图片为准，不应再叠加固定单价
	nearly(t, q.GetTotalQuota(0, 0, nil), 0.5)
}

func TestTimesPriceWithoutImagesStillCharged(t *testing.T) {
	q := newImageQuota(model.TimesPriceType, 1, nil)
	q.inputRatio = 5

	if got := q.GetTotalQuota(0, 0, nil); got != 5000 {
		t.Errorf("按次计费应得到 5000 配额，实际 %d", got)
	}
}

func TestImageQuotaWithoutConfigIsZero(t *testing.T) {
	q := &Quota{price: model.Price{Type: model.TokensPriceType}, groupRatio: 1}
	q.imageOutputs = map[string]int{"2k": 1}

	// 没配图片档位价时不应凭空产生费用，也不该给出错误的 1 配额
	if got := q.GetTotalQuota(0, 0, nil); got != 0 {
		t.Errorf("未配置图片价格时应为 0，实际 %d", got)
	}
}

// gpt-image 这类模型上游会回显图像 output token，此时必须按 token 结算，
// 即使误配了档位价也不能再叠加，否则一张图会被收两次。
func TestImageTokenBillingTakesPrecedence(t *testing.T) {
	conf := datatypes.NewJSONType(model.PriceImagePrices{Enabled: true, Tiers: map[string]model.PriceImageTier{
		"2k": {Output: 0.5 / model.RMBRate},
	}})
	extra := datatypes.NewJSONType(map[string]float64{
		config.UsageExtraInputTextTokens: 0.625,
		config.UsageExtraCache:           0.15625,
	})
	q := &Quota{
		price: model.Price{
			Type:        model.TokensPriceType,
			Input:       4,  // $8/M 图像输入
			Output:      15, // $30/M 图像输出
			ImagePrices: &conf,
			ExtraRatios: &extra,
		},
		groupRatio:  1,
		inputRatio:  4,
		outputRatio: 15,
	}
	usage := &types.Usage{PromptTokens: 1100, CompletionTokens: 4160, TotalTokens: 5260}
	usage.PromptTokensDetails.TextTokens = 1000
	q.imageOutputs = map[string]int{"2k": 1}

	quota := q.GetTotalQuotaByUsage(usage)
	// 文本输入按 0.625 修正：1100 + 1000×(0.625-1) = 725
	// 725×4 + 4160×15 = 65300；若再叠加 2k 档位价会多出约 35714
	if quota != 65300 {
		t.Errorf("应只按 token 计费得到 65300，实际 %d（多收说明叠加了档位价）", quota)
	}
}
