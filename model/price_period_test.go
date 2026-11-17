package model

import (
	"testing"
	"time"

	"gorm.io/datatypes"
)

func mustPriceTimePrices(periods ...PricePeriod) *datatypes.JSONType[PriceTimePrices] {
	value := datatypes.NewJSONType(PriceTimePrices{Enabled: true, Periods: periods})
	return &value
}

// 东八区的某个时刻
func cst(hour, minute int) time.Time {
	return time.Date(2026, 11, 17, hour, minute, 0, 0, BillingZone)
}

func TestPricePeriodContains(t *testing.T) {
	crossMidnight := PricePeriod{Name: "闲时", Start: "22:00", End: "08:00", Input: 0.0714, Output: 0.2857}
	daytime := PricePeriod{Name: "忙时", Start: "08:00", End: "22:00", Input: 0.14286, Output: 0.57143}

	cases := []struct {
		name   string
		period PricePeriod
		now    time.Time
		want   bool
	}{
		{"跨零点-起点", crossMidnight, cst(22, 0), true},
		{"跨零点-深夜", crossMidnight, cst(23, 30), true},
		{"跨零点-凌晨", crossMidnight, cst(3, 0), true},
		{"跨零点-终点不含", crossMidnight, cst(8, 0), false},
		{"跨零点-白日不含", crossMidnight, cst(12, 0), false},
		{"白天-起点", daytime, cst(8, 0), true},
		{"白天-末尾", daytime, cst(21, 59), true},
		{"白天-终点不含", daytime, cst(22, 0), false},
		{"白天-凌晨不含", daytime, cst(7, 59), false},
		{"非法时间", PricePeriod{Start: "25:00", End: "08:00"}, cst(3, 0), false},
		{"空串", PricePeriod{Start: "", End: ""}, cst(3, 0), false},
		{"起止相同", PricePeriod{Start: "08:00", End: "08:00"}, cst(8, 0), false},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if got := item.period.Contains(item.now); got != item.want {
				t.Errorf("Contains(%s) = %v, want %v", item.now.Format("15:04"), got, item.want)
			}
		})
	}
}

func TestPricePeriodUsesCST(t *testing.T) {
	// UTC 14:30 即东八区 22:30，应命中闲时
	period := PricePeriod{Start: "22:00", End: "08:00"}
	if !period.Contains(time.Date(2026, 11, 17, 14, 30, 0, 0, time.UTC)) {
		t.Error("UTC 14:30 应换算成东八区 22:30 并命中闲时")
	}
	// UTC 16:30 即东八区次日 00:30
	if !period.Contains(time.Date(2026, 11, 17, 16, 30, 0, 0, time.UTC)) {
		t.Error("UTC 16:30 应换算成东八区 00:30 并命中闲时")
	}
}

func TestPriceRatiosAt(t *testing.T) {
	price := &Price{
		Type:   TokensPriceType,
		Input:  0.14286,
		Output: 0.57143,
		TimePrices: mustPriceTimePrices(
			PricePeriod{Name: "闲时", Start: "22:00", End: "08:00", Input: 0.07143, Output: 0.28571},
		),
	}

	input, output, period := price.RatiosAt(cst(23, 0))
	if period == nil || period.Name != "闲时" {
		t.Fatalf("23:00 应命中闲时，实际 %+v", period)
	}
	if input != 0.07143 || output != 0.28571 {
		t.Errorf("闲时倍率 = %v/%v，期望 0.07143/0.28571", input, output)
	}

	input, output, period = price.RatiosAt(cst(12, 0))
	if period != nil {
		t.Errorf("12:00 不应命中任何时段，实际 %+v", period)
	}
	if input != 0.14286 || output != 0.57143 {
		t.Errorf("未命中时段应回落基础价，实际 %v/%v", input, output)
	}
}

func TestPriceRatiosAtInherit(t *testing.T) {
	// 时段只覆盖输入价，输出价沿用基础价格
	price := &Price{
		Type:       TokensPriceType,
		Input:      0.2,
		Output:     0.8,
		TimePrices: mustPriceTimePrices(PricePeriod{Start: "00:00", End: "12:00", Input: 0.1}),
	}

	input, output, period := price.RatiosAt(cst(1, 0))
	if period == nil || input != 0.1 || output != 0.8 {
		t.Errorf("期望 input=0.1 output=0.8，实际 %v/%v (period=%+v)", input, output, period)
	}
}

func TestPriceRatiosAtDisabled(t *testing.T) {
	disabled := datatypes.NewJSONType(PriceTimePrices{
		Enabled: false,
		Periods: []PricePeriod{{Start: "00:00", End: "23:59", Input: 0.001, Output: 0.001}},
	})
	price := &Price{Type: TokensPriceType, Input: 0.2, Output: 0.8, TimePrices: &disabled}

	input, output, period := price.RatiosAt(cst(12, 0))
	if period != nil || input != 0.2 || output != 0.8 {
		t.Errorf("关闭分时后应始终使用基础价，实际 %v/%v period=%+v", input, output, period)
	}
}

func TestPricePeriodFirstMatchWins(t *testing.T) {
	price := &Price{
		Type:   TokensPriceType,
		Input:  1,
		Output: 1,
		TimePrices: mustPriceTimePrices(
			PricePeriod{Name: "夜间前半", Start: "22:00", End: "02:00", Input: 0.1},
			PricePeriod{Name: "夜间后半", Start: "02:00", End: "08:00", Input: 0.2},
		),
	}

	_, _, period := price.RatiosAt(cst(1, 0))
	if period == nil || period.Name != "夜间前半" {
		t.Errorf("01:00 应命中第一个时段，实际 %+v", period)
	}

	_, _, period = price.RatiosAt(cst(3, 0))
	if period == nil || period.Name != "夜间后半" {
		t.Errorf("03:00 应命中第二个时段，实际 %+v", period)
	}
}
