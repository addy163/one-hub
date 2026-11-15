package relay

import (
	"one-api/common/config"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// realtimeLimiter 约束 /v1/realtime 的并发连接数。
// 原来只有「每条消息 2 分钟读超时」，一个客户端可以无限占用连接与上游配额；
// 这里补上全局上限、单令牌上限，配合 WSProxy 的最大连接时长与心跳做完整治理。
type realtimeLimiter struct {
	mu       sync.Mutex
	total    int
	perToken map[string]int
}

var wsLimiter = &realtimeLimiter{perToken: make(map[string]int)}

func realtimeLimitKey(c *gin.Context) string {
	if id := c.GetInt("token_id"); id > 0 {
		return "token:" + strconv.Itoa(id)
	}
	if id := c.GetInt("id"); id > 0 {
		return "user:" + strconv.Itoa(id)
	}
	return "ip:" + c.ClientIP()
}

func (l *realtimeLimiter) acquire(c *gin.Context) bool {
	limits := config.CurrentWsLimits()
	key := realtimeLimitKey(c)

	l.mu.Lock()
	defer l.mu.Unlock()
	if limits.MaxConnections > 0 && l.total >= limits.MaxConnections {
		return false
	}
	if limits.MaxConnectionsPerToken > 0 && l.perToken[key] >= limits.MaxConnectionsPerToken {
		return false
	}
	l.total++
	l.perToken[key]++
	return true
}

func (l *realtimeLimiter) release(c *gin.Context) {
	key := realtimeLimitKey(c)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.total > 0 {
		l.total--
	}
	if l.perToken[key] > 1 {
		l.perToken[key]--
	} else {
		delete(l.perToken, key)
	}
}

// ActiveRealtimeConnections 供日志 / 诊断读取当前活跃连接数。
func ActiveRealtimeConnections() int {
	wsLimiter.mu.Lock()
	defer wsLimiter.mu.Unlock()
	return wsLimiter.total
}
