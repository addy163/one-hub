package model

import (
	"embed"
	"encoding/json"
	"one-api/common/config"
	"one-api/common/logger"
	"strconv"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

//go:embed seed/model_info.json
var seedFS embed.FS

// 二次开发引入的默认配置项。只在该 key 不存在时写入，
// 因此不会覆盖管理员已经在后台改过的值。
var forkDefaultOptions = []struct {
	Key   string
	Value string
}{
	{Key: "RelayEndpoints", Value: "[]"},
	{Key: "RelayEndpointProbeInterval", Value: "60"},
	{Key: "RelayEndpointProbeTimeout", Value: "3000"},
}

type seedModelInfo struct {
	Model string   `json:"model"`
	Name  string   `json:"name"`
	Ctx   int      `json:"ctx"`
	Max   int      `json:"max"`
	In    []string `json:"in"`
	Out   []string `json:"out"`
	Desc  string   `json:"desc"`
}

// SeedForkDefaults 补齐二次开发引入的数据：线路配置项、缺失的价格占位行、模型规格表。
//
// 设计原则：
//   - 只补缺失，不覆盖已有数据，可以反复执行；
//   - 价格只是占位默认值，管理员可按自己服务商的实际成本在「模型价格」中调整；
//   - 全新安装时 prices 表为空，价格交给内置价格表（GetDefaultPrice）同步，
//     这里跳过，避免「只有一行价格」导致内置价格不再同步。
func SeedForkDefaults(tx *gorm.DB) error {
	if err := seedOptionDefaults(tx); err != nil {
		return err
	}
	if err := seedPricePlaceholder(tx); err != nil {
		return err
	}
	return seedModelInfoRows(tx)
}

func seedOptionDefaults(tx *gorm.DB) error {
	for _, item := range forkDefaultOptions {
		// 用结构体条件，让 GORM 按方言处理 key 这个保留字
		var count int64
		if err := tx.Model(&Option{}).Where(&Option{Key: item.Key}).Count(&count).Error; err != nil {
			logger.SysError("检查配置项 " + item.Key + " 失败: " + err.Error())
			return err
		}
		if count > 0 {
			continue
		}
		if err := tx.Create(&Option{Key: item.Key, Value: item.Value}).Error; err != nil {
			logger.SysError("写入默认配置项 " + item.Key + " 失败: " + err.Error())
			return err
		}
		logger.SysLog("已补充默认配置项: " + item.Key)
	}
	return nil
}

// seedPricePlaceholder 为缺失的模型补一条价格占位行，已有配置一律不动。
func seedPricePlaceholder(tx *gorm.DB) error {
	const modelName = "qwen4-max"

	var exist int64
	if err := tx.Model(&Price{}).Where("model = ?", modelName).Count(&exist).Error; err != nil {
		return err
	}
	if exist > 0 {
		// 管理员已经配置过，保持原样
		return nil
	}

	// 价格表为空说明是全新安装，交给内置价格表同步，避免阻断其它模型的初始化
	var total int64
	if err := tx.Model(&Price{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return nil
	}

	ratios := datatypes.NewJSONType(map[string]float64{
		config.UsageExtraCache:       0.25,
		config.UsageExtraCachedRead:  0.25,
		config.UsageExtraCachedWrite: 2.0,
	})
	price := &Price{
		Model:       modelName,
		Type:        TokensPriceType,
		ChannelType: config.ChannelTypeAli,
		Input:       0.8,
		Output:      2.4,
		Locked:      true,
		ExtraRatios: &ratios,
	}
	if err := tx.Create(price).Error; err != nil {
		logger.SysError("补充内置价格行失败: " + err.Error())
		return err
	}
	logger.SysLog("已补充内置价格行: " + modelName)
	return nil
}

func seedModelInfoRows(tx *gorm.DB) error {
	raw, err := seedFS.ReadFile("seed/model_info.json")
	if err != nil {
		return err
	}

	var seeds []seedModelInfo
	if err := json.Unmarshal(raw, &seeds); err != nil {
		return err
	}
	if len(seeds) == 0 {
		return nil
	}

	var existing []string
	if err := tx.Model(&ModelInfo{}).Pluck("model", &existing).Error; err != nil {
		return err
	}
	known := make(map[string]bool, len(existing))
	for _, name := range existing {
		known[name] = true
	}

	rows := make([]ModelInfo, 0, len(seeds))
	for _, item := range seeds {
		if item.Model == "" || known[item.Model] {
			continue
		}
		known[item.Model] = true
		rows = append(rows, ModelInfo{
			Model:            item.Model,
			Name:             fallback(item.Name, item.Model),
			Description:      item.Desc,
			ContextLength:    item.Ctx,
			MaxTokens:        item.Max,
			InputModalities:  jsonList(item.In),
			OutputModalities: jsonList(item.Out),
		})
	}
	if len(rows) == 0 {
		return nil
	}

	// 分批插入，避免 SQLite 单条语句变量数量超限
	const batchSize = 50
	for start := 0; start < len(rows); start += batchSize {
		end := start + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := tx.Create(rows[start:end]).Error; err != nil {
			logger.SysError("写入模型规格失败: " + err.Error())
			return err
		}
	}
	logger.SysLog("已补充模型规格 " + strconv.Itoa(len(rows)) + " 条")
	return nil
}

func jsonList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
