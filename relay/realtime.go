package relay

import (
	"fmt"
	"net/http"
	"one-api/common"
	"one-api/common/config"
	"one-api/common/logger"
	"one-api/common/requester"
	"one-api/common/utils"
	"one-api/metrics"
	providersBase "one-api/providers/base"
	"one-api/relay/relay_util"
	"one-api/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type RelayModeChatRealtime struct {
	relayBase
	userConn       *websocket.Conn
	messageHandler requester.MessageHandler
	providerConn   *websocket.Conn
	quota          *relay_util.Quota
	usage          *types.UsageEvent
}

var upgrader = websocket.Upgrader{
	// 默认不限制 Origin（websocket_allowed_origins 为空时保持历史行为）。
	// 这里不能无条件改成「默认白名单」：one-hub 是 API 代理，浏览器 Realtime SDK
	// 来自任意站点；而本端点的鉴权只认令牌、不读 Cookie，因此不存在
	// Cross-Site WebSocket Hijacking。仅当管理员显式配置白名单时才收紧。
	CheckOrigin: func(r *http.Request) bool {
		return config.AllowWebsocketOrigin(r.Header.Get("Origin"))
	},
	Subprotocols: []string{"realtime"},
}

func ChatRealtime(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		common.AbortWithMessage(c, http.StatusBadRequest, "model_name_required")
		return
	}

	// 在 Upgrade 之前拒绝：超限握手返回 429 JSON，客户端不会误以为连接已建立。
	if !wsLimiter.acquire(c) {
		common.AbortWithMessage(c, http.StatusTooManyRequests, "too_many_realtime_connections")
		return
	}
	defer wsLimiter.release(c)

	userConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade 失败时 gorilla 已经写入 HTTP 响应，不能再写一次 header。
		logger.LogError(c.Request.Context(), fmt.Sprintf("websocket upgrade failed: %s", err.Error()))
		return
	}

	relay := &RelayModeChatRealtime{
		relayBase: relayBase{
			c: c,
		},
		userConn: userConn,
	}
	relay.setOriginalModel(modelName)

	if !relay.getProvider() {
		return
	}

	relay.quota = relay_util.NewQuota(relay.getContext(), relay.getModelName(), 0)

	relay.usage = &types.UsageEvent{}

	limits := config.CurrentWsLimits()
	wsProxy := requester.NewWSProxy(
		relay.userConn,
		relay.providerConn,
		time.Duration(limits.IdleTimeoutSeconds)*time.Second,
		relay.messageHandler,
		relay.usageHandler,
		requester.WithWsPingInterval(time.Duration(limits.PingIntervalSeconds)*time.Second),
		requester.WithWsMaxDuration(time.Duration(limits.MaxConnectionSeconds)*time.Second),
	)

	wsProxy.Start()
	wsProxy.Wait()

	closedBy := "timeout_or_error"
	select {
	case <-wsProxy.UserClosed():
		closedBy = "user"
	case <-wsProxy.SupplierClosed():
		closedBy = "provider"
	default:
	}
	logger.LogInfo(relay.c.Request.Context(), fmt.Sprintf("连接由%s关闭，当前活跃连接数 %d", closedBy, ActiveRealtimeConnections()))

	// 先关掉另一侧还在读的 goroutine，再同步结算配额。
	// 原来把结算放在独立 goroutine 里，handler 返回后 Gin context 已回收，
	// 存在生命周期竞态；这里改成同步执行，也不用担心重复扣费。
	wsProxy.Close()
	relay.quota.Consume(relay.c, relay.usage.ToChatUsage(), false)
}

func (r *RelayModeChatRealtime) abortWithMessage(message string) {
	eventErr := types.NewErrorEvent("", "system_error", "system_error", message)

	r.userConn.WriteMessage(websocket.TextMessage, []byte(eventErr.Error()))
	r.userConn.Close()
	// 首帧校验失败 / 重试全部失败时，已建立的上游连接也要关掉，否则会一直挂着。
	if r.providerConn != nil {
		r.providerConn.Close()
	}
}

func (r *RelayModeChatRealtime) getProvider() bool {
	retryTimes := config.RetryTimes
	if retryTimes == 0 {
		retryTimes = 1
	}

	for i := retryTimes; i > 0; i-- {
		// 找不到直接返回
		if err := r.setProvider(r.getOriginalModel()); err != nil {
			r.abortWithMessage(err.Error())
			return false
		}

		realtimeProvider, ok := r.provider.(providersBase.RealtimeInterface)
		if !ok {
			r.abortWithMessage("channel not implemented")
			return false
		}
		channel := r.provider.GetChannel()

		providerConn, messageHandler, apiErr := realtimeProvider.CreateChatRealtime(r.modelName)
		if apiErr != nil {
			r.skipChannelIds(channel.Id)
			logger.LogError(r.c.Request.Context(), fmt.Sprintf("using channel #%d(%s) Error: %s to retry (remain times %d)", channel.Id, channel.Name, apiErr.Error(), i))
			metrics.RecordProvider(r.c, apiErr.StatusCode)

			continue
		}

		r.messageHandler = messageHandler
		r.providerConn = providerConn

		if r.getRealtimeFirstMessage() {
			metrics.RecordProvider(r.c, 200)
			return true
		}

		r.skipChannelIds(channel.Id)
	}

	r.abortWithMessage("get provider failed")
	return false
}

func (r *RelayModeChatRealtime) skipChannelIds(channelId int) {
	skipChannelIds, ok := utils.GetGinValue[[]int](r.c, "skip_channel_ids")
	if !ok {
		skipChannelIds = make([]int, 0)
	}

	skipChannelIds = append(skipChannelIds, channelId)

	r.c.Set("skip_channel_ids", skipChannelIds)
}

func (r *RelayModeChatRealtime) getRealtimeFirstMessage() bool {
	messageType, firstMessage, err := r.providerConn.ReadMessage()
	if err != nil {
		return false
	}

	if messageType != websocket.TextMessage {
		return false
	}

	shouldContinue, _, newMessage, err := r.messageHandler(requester.SupplierMessage, messageType, firstMessage)

	if !shouldContinue || err != nil {
		return false
	}

	if newMessage != nil {
		r.userConn.WriteMessage(websocket.TextMessage, newMessage)
	} else {
		r.userConn.WriteMessage(websocket.TextMessage, firstMessage)
	}

	return true
}

func (r *RelayModeChatRealtime) usageHandler(usage *types.UsageEvent) error {
	err := r.quota.UpdateUserRealtimeQuota(r.usage, usage)
	if err != nil {
		return types.NewErrorEvent("", "system_error", "system_error", err.Error())
	}

	return nil
}
