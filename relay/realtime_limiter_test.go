package relay

import (
	"net/http/httptest"
	"one-api/common/config"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRealtimeContext(tokenID int) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/realtime?model=gpt-4o-realtime", nil)
	c.Set("token_id", tokenID)
	return c
}

func TestRealtimeLimiterLimits(t *testing.T) {
	old := config.CurrentWsLimits()
	defer func() {
		_ = config.SetRuntimeOption("WsMaxConnections", strconv.Itoa(old.MaxConnections))
		_ = config.SetRuntimeOption("WsMaxConnectionsPerToken", strconv.Itoa(old.MaxConnectionsPerToken))
	}()

	saved := wsLimiter
	wsLimiter = &realtimeLimiter{perToken: make(map[string]int)}
	defer func() { wsLimiter = saved }()

	if err := config.SetRuntimeOption("WsMaxConnections", "2"); err != nil {
		t.Fatal(err)
	}
	if err := config.SetRuntimeOption("WsMaxConnectionsPerToken", "1"); err != nil {
		t.Fatal(err)
	}

	c1 := newRealtimeContext(1)
	c2 := newRealtimeContext(2)
	c3 := newRealtimeContext(3)

	if !wsLimiter.acquire(c1) || !wsLimiter.acquire(c2) {
		t.Fatal("first two connections should be accepted")
	}
	if wsLimiter.acquire(c3) {
		t.Fatal("global connection limit must be enforced")
	}
	// 同一个令牌不能突破单令牌上限
	if wsLimiter.acquire(newRealtimeContext(1)) {
		t.Fatal("per-token connection limit must be enforced")
	}

	wsLimiter.release(c1)
	if ActiveRealtimeConnections() != 1 {
		t.Errorf("active = %d, want 1", ActiveRealtimeConnections())
	}
	// 释放后可以再次用该令牌建立连接
	if !wsLimiter.acquire(newRealtimeContext(1)) {
		t.Fatal("slot should be reusable after release")
	}
	wsLimiter.release(newRealtimeContext(1))
	wsLimiter.release(c2)
	if ActiveRealtimeConnections() != 0 {
		t.Errorf("active = %d, want 0", ActiveRealtimeConnections())
	}
	// 重复 release 不能把计数变成负数
	wsLimiter.release(c2)
	if ActiveRealtimeConnections() != 0 {
		t.Errorf("double release corrupted counter: %d", ActiveRealtimeConnections())
	}

	// 放宽上限后立即可用，验证后台配置热生效
	if err := config.SetRuntimeOption("WsMaxConnectionsPerToken", "5"); err != nil {
		t.Fatal(err)
	}
	a := newRealtimeContext(9)
	b := newRealtimeContext(9)
	if !wsLimiter.acquire(a) || !wsLimiter.acquire(b) {
		t.Fatal("updated per-token limit should take effect without restart")
	}
}
