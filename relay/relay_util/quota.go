package relay_util

import (
	"context"
	"errors"
	"math"
	"net/http"
	"one-api/common"
	"one-api/common/config"
	"one-api/common/logger"
	"one-api/model"
	"one-api/types"
	"time"

	"github.com/gin-gonic/gin"
)

type Quota struct {
	modelName        string
	promptTokens     int
	price            model.Price
	groupName        string
	isBackupGroup    bool // 新增字段记录是否使用备用分组
	backupGroupName  string
	groupRatio       float64
	inputRatio       float64
	outputRatio      float64
	baseInputRatio   float64  // 不含分组倍率的输入倍率（分时命中后为时段价）
	baseOutputRatio  float64  // 不含分组倍率的输出倍率
	pricePeriod      string   // 命中的时段名称，未命中为空
	cachedRatio      *float64 // 时段内指定的缓存命中倍率，未指定为 nil
	preConsumedQuota int
	cacheQuota       int
	userId           int
	channelId        int
	tokenId          int
	unlimitedQuota   bool
	HandelStatus     bool

	startTime         time.Time
	firstResponseTime time.Time
	extraBillingData  map[string]ExtraBillingData

	// 本次调用的图片张数（按档位），由 usage 填充，用于按张计费
	imageInputs  map[string]int
	imageOutputs map[string]int
}

func NewQuota(c *gin.Context, modelName string, promptTokens int) *Quota {
	isBackupGroup := c.GetBool("is_backupGroup")

	quota := &Quota{
		modelName:      modelName,
		promptTokens:   promptTokens,
		userId:         c.GetInt("id"),
		channelId:      c.GetInt("channel_id"),
		tokenId:        c.GetInt("token_id"),
		unlimitedQuota: c.GetBool("token_unlimited_quota"),
		HandelStatus:   false,
		isBackupGroup:  isBackupGroup, // 记录是否使用备用分组
	}

	quota.price = *model.PricingInstance.GetPrice(quota.modelName)
	quota.groupName = c.GetString("token_group")
	quota.backupGroupName = c.GetString("token_backup_group")
	quota.groupRatio = c.GetFloat64("group_ratio") // 这里的倍率已经在 common.go 中正确设置了

	// 分时倍率：按请求到达时间选时段，未配置时段时等同于基础价格
	inputRatio, outputRatio, period := quota.price.RatiosAt(time.Now())
	quota.baseInputRatio = inputRatio
	quota.baseOutputRatio = outputRatio
	quota.inputRatio = inputRatio * quota.groupRatio
	quota.outputRatio = outputRatio * quota.groupRatio
	if period != nil {
		quota.pricePeriod = period.Name
		if quota.pricePeriod == "" {
			quota.pricePeriod = period.Start + "-" + period.End
		}
		quota.cachedRatio = period.Cached
	}

	return quota

}

// getExtraRatio 返回额外的 token 倍率。处于分时时段且该时段指定了缓存倍率时，
// 缓存类的倍率以时段配置为准。
func (q *Quota) getExtraRatio(key string) float64 {
	if q.cachedRatio != nil && (key == config.UsageExtraCache || key == config.UsageExtraCachedRead) {
		return *q.cachedRatio
	}
	return q.price.GetExtraRatio(key)
}

func (q *Quota) PreQuotaConsumption() *types.OpenAIErrorWithStatusCode {
	if q.price.Type == model.TimesPriceType {
		q.preConsumedQuota = int(1000 * q.inputRatio)
	} else if q.baseInputRatio != 0 || q.baseOutputRatio != 0 {
		q.preConsumedQuota = int(float64(q.promptTokens)*q.inputRatio) + config.PreConsumedQuota
	}

	if q.preConsumedQuota == 0 {
		return nil
	}

	userQuota, err := model.CacheGetUserQuota(q.userId)
	if err != nil {
		return common.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}

	if userQuota < q.preConsumedQuota {
		return common.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusPaymentRequired)
	}

	err = model.CacheDecreaseUserQuota(q.userId, q.preConsumedQuota)
	if err != nil {
		return common.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}

	if userQuota > 100*q.preConsumedQuota {
		// in this case, we do not pre-consume quota
		// because the user has enough quota
		q.preConsumedQuota = 0
		// common.LogInfo(c.Request.Context(), fmt.Sprintf("user %d has enough quota %d, trusted and no need to pre-consume", userId, userQuota))
	}

	if q.preConsumedQuota > 0 {
		err := model.PreConsumeTokenQuota(q.tokenId, q.preConsumedQuota)
		if err != nil {
			return common.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
		q.HandelStatus = true
	}

	return nil
}

// 更新用户实时配额
func (q *Quota) UpdateUserRealtimeQuota(usage *types.UsageEvent, nowUsage *types.UsageEvent) error {
	usage.Merge(nowUsage)

	// 不开启Redis，则不更新实时配额
	if !config.RedisEnabled {
		return nil
	}

	promptTokens, completionTokens := q.getComputeTokensByUsageEvent(nowUsage)
	increaseQuota := q.GetTotalQuota(promptTokens, completionTokens, nil)

	cacheQuota, err := model.CacheIncreaseUserRealtimeQuota(q.userId, increaseQuota)
	if err != nil {
		return errors.New("error update user realtime quota cache: " + err.Error())
	}

	q.cacheQuota += increaseQuota
	userQuota, err := model.CacheGetUserQuota(q.userId)
	if err != nil {
		return errors.New("error get user quota cache: " + err.Error())
	}

	if cacheQuota >= int64(userQuota) {
		return errors.New("user quota is not enough")
	}

	return nil
}

func (q *Quota) completedQuotaConsumption(usage *types.Usage, tokenName string, isStream bool, sourceIp string, ctx context.Context) error {
	defer func() {
		if q.cacheQuota > 0 {
			model.CacheDecreaseUserRealtimeQuota(q.userId, q.cacheQuota)
		}
	}()

	quota := q.GetTotalQuotaByUsage(usage)

	if quota > 0 {
		quotaDelta := quota - q.preConsumedQuota
		err := model.PostConsumeTokenQuotaWithInfo(q.tokenId, q.userId, q.unlimitedQuota, quotaDelta)
		if err != nil {
			return errors.New("error consuming token remain quota: " + err.Error())
		}
		err = model.CacheUpdateUserQuota(q.userId)
		if err != nil {
			return errors.New("error consuming token remain quota: " + err.Error())
		}
		model.UpdateChannelUsedQuota(q.channelId, quota)
	}

	model.RecordConsumeLog(
		ctx,
		q.userId,
		q.channelId,
		usage.PromptTokens,
		usage.CompletionTokens,
		q.modelName,
		tokenName,
		quota,
		"",
		q.getRequestTime(),
		isStream,
		q.GetLogMeta(usage),
		sourceIp,
	)
	model.UpdateUserUsedQuotaAndRequestCount(q.userId, quota)

	return nil
}

func (q *Quota) Undo(c *gin.Context) {
	if q.HandelStatus {
		go func(ctx context.Context) {
			// return pre-consumed quota
			err := model.PostConsumeTokenQuotaWithInfo(q.tokenId, q.userId, q.unlimitedQuota, -q.preConsumedQuota)
			if err != nil {
				logger.LogError(ctx, "error return pre-consumed quota: "+err.Error())
			}
		}(c.Request.Context())
	}
}

func (q *Quota) Consume(c *gin.Context, usage *types.Usage, isStream bool) {
	tokenName := c.GetString("token_name")
	q.startTime = c.GetTime("requestStartTime")
	// 如果没有报错，则消费配额
	go func(ctx context.Context) {
		err := q.completedQuotaConsumption(usage, tokenName, isStream, c.ClientIP(), ctx)
		if err != nil {
			logger.LogError(ctx, err.Error())
		}
	}(c.Request.Context())
}

func (q *Quota) GetInputRatio() float64 {
	return q.inputRatio
}

func (q *Quota) GetLogMeta(usage *types.Usage) map[string]any {
	meta := map[string]any{
		"group_name":        q.groupName,
		"backup_group_name": q.backupGroupName,
		"is_backup_group":   q.isBackupGroup, // 添加是否使用备用分组的标识
		"price_type":        q.price.Type,
		"group_ratio":       q.groupRatio,
		"input_ratio":       q.baseInputRatio,
		"output_ratio":      q.baseOutputRatio,
	}

	if q.pricePeriod != "" {
		meta["price_period"] = q.pricePeriod
	}

	if q.cachedRatio != nil {
		meta["price_period_cached_ratio"] = *q.cachedRatio
	}

	firstResponseTime := q.GetFirstResponseTime()
	if firstResponseTime > 0 {
		meta["first_response"] = firstResponseTime
	}

	if usage != nil {
		extraTokens := usage.GetExtraTokens()

		for key, value := range extraTokens {
			meta[key] = value
			extraRatio := q.getExtraRatio(key)
			meta[key+"_ratio"] = extraRatio
		}

		// 图片按张计费时把张数与档位记进日志，方便核对上游账单
		if len(usage.OutputImages) > 0 {
			meta["output_images"] = usage.OutputImages
		}
		if len(usage.InputImages) > 0 {
			meta["input_images"] = usage.InputImages
		}
	}

	if q.extraBillingData != nil {
		meta["extra_billing"] = q.extraBillingData
	}

	return meta
}

func (q *Quota) getRequestTime() int {
	return int(time.Since(q.startTime).Milliseconds())
}

// 通过 token 数获取消费配额
func (q *Quota) GetTotalQuota(promptTokens, completionTokens int, extraBilling map[string]types.ExtraBilling) (quota int) {
	// 作图类模型按「张数 × 档位单价」结算，与 token 无关。
	// 但 gpt-image 这类模型上游会回显图像 output token，此时按 token 结算更准确，
	// 档位价退化为「拿不到 token 用量」时的兜底（如 DashScope 只回尺寸和张数），
	// 避免同一张图既按 token 又按档位收两次。
	imageQuota := 0
	if completionTokens == 0 {
		imageQuota = q.getImageQuota()
	}

	if q.price.Type == model.TimesPriceType {
		// 配了图片档位价时以图片计费为准，避免与固定单价重复收费
		if imageQuota > 0 {
			quota = imageQuota
		} else {
			quota = int(1000 * q.inputRatio)
		}
	} else {
		quota = int(math.Ceil((float64(promptTokens) * q.inputRatio) + (float64(completionTokens) * q.outputRatio)))
		quota += imageQuota
	}

	q.GetExtraBillingData(extraBilling)
	extraBillingQuota := 0
	if q.extraBillingData != nil {
		for _, value := range q.extraBillingData {
			extraBillingQuota += int(math.Ceil(
				float64(value.Price)*float64(config.QuotaPerUnit),
			)) * value.CallCount
		}
	}

	if extraBillingQuota > 0 {
		quota += int(math.Ceil(
			float64(extraBillingQuota) * q.groupRatio,
		))
	}

	if q.inputRatio != 0 && quota <= 0 {
		quota = 1
	}
	totalTokens := promptTokens + completionTokens
	if totalTokens == 0 && q.price.Type != model.TimesPriceType && imageQuota == 0 && extraBillingQuota == 0 {
		// in this case, must be some error happened
		// we cannot just return, because we may have to return the pre-consumed quota
		quota = 0
	}

	return quota
}

// getImageQuota 按上游回显的图片张数与档位计算配额。
// 单价存的是倍率（1 倍率 = 0.014 元/张），1 倍率对应 1000 配额。
func (q *Quota) getImageQuota() int {
	conf := q.price.GetImagePrices()
	if conf == nil {
		return 0
	}

	total := 0
	for tier, count := range q.imageOutputs {
		if count <= 0 {
			continue
		}
		if t, ok := conf.GetImageTier(tier); ok && t.Output > 0 {
			total += int(math.Ceil(float64(count) * t.Output * 1000))
		}
	}
	for tier, count := range q.imageInputs {
		if count <= 0 {
			continue
		}
		if t, ok := conf.GetImageTier(tier); ok && t.Input > 0 {
			total += int(math.Ceil(float64(count) * t.Input * 1000))
		}
	}

	if total <= 0 {
		return 0
	}

	return int(math.Ceil(float64(total) * q.groupRatio))
}

// 获取计算的 token 数
func (q *Quota) getComputeTokensByUsage(usage *types.Usage) (promptTokens, completionTokens int) {
	promptTokens = usage.PromptTokens
	completionTokens = usage.CompletionTokens

	extraTokens := usage.GetExtraTokens()

	for key, value := range extraTokens {
		extraRatio := q.getExtraRatio(key)
		if model.GetExtraPriceIsPrompt(key) {
			promptTokens += model.GetIncreaseTokens(value, extraRatio)
		} else {
			completionTokens += model.GetIncreaseTokens(value, extraRatio)
		}
	}

	return
}

func (q *Quota) getComputeTokensByUsageEvent(usage *types.UsageEvent) (promptTokens, completionTokens int) {
	promptTokens = usage.InputTokens
	completionTokens = usage.OutputTokens
	extraTokens := usage.GetExtraTokens()

	for key, value := range extraTokens {
		extraRatio := q.getExtraRatio(key)
		if model.GetExtraPriceIsPrompt(key) {
			promptTokens += model.GetIncreaseTokens(value, extraRatio)
		} else {
			completionTokens += model.GetIncreaseTokens(value, extraRatio)
		}
	}

	return
}

// 通过 usage 获取消费配额
func (q *Quota) GetTotalQuotaByUsage(usage *types.Usage) (quota int) {
	promptTokens, completionTokens := q.getComputeTokensByUsage(usage)
	q.imageInputs = usage.InputImages
	q.imageOutputs = usage.OutputImages
	return q.GetTotalQuota(promptTokens, completionTokens, usage.ExtraBilling)
}

func (q *Quota) GetFirstResponseTime() int64 {
	// 先判断 firstResponseTime 是否为0
	if q.firstResponseTime.IsZero() {
		return 0
	}

	return q.firstResponseTime.Sub(q.startTime).Milliseconds()
}

func (q *Quota) SetFirstResponseTime(firstResponseTime time.Time) {
	q.firstResponseTime = firstResponseTime
}

type ExtraBillingData struct {
	Type      string  `json:"type"`
	CallCount int     `json:"call_count"`
	Price     float64 `json:"price"`
}

func (q *Quota) GetExtraBillingData(extraBilling map[string]types.ExtraBilling) {
	if extraBilling == nil {
		return
	}

	extraBillingData := make(map[string]ExtraBillingData)
	for serviceType, value := range extraBilling {
		extraBillingData[serviceType] = ExtraBillingData{
			Type:      value.Type,
			CallCount: value.CallCount,
			Price:     getDefaultExtraServicePrice(serviceType, q.modelName, value.Type),
		}

	}

	if len(extraBillingData) == 0 {
		return
	}

	q.extraBillingData = extraBillingData
}
