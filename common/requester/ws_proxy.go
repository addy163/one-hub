package requester

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"one-api/common/logger"
	"one-api/types"
)

type WSProxy struct {
	userConn       *websocket.Conn
	supplierConn   *websocket.Conn
	timeout        time.Duration
	pingInterval   time.Duration
	maxDuration    time.Duration
	handler        MessageHandler
	usageHandler   UsageHandler
	done           chan struct{}
	stop           chan struct{}
	userClosed     chan struct{}
	supplierClosed chan struct{}
	startOnce      sync.Once
	closeOnce      sync.Once
	deadlineTimer  *time.Timer
}

type MessageSource int

const (
	UserMessage MessageSource = iota
	SupplierMessage
)

type MessageHandler func(source MessageSource, messageType int, message []byte) (bool, *types.UsageEvent, []byte, error)
type UsageHandler func(usage *types.UsageEvent) error

// WSOption 是 WebSocket 连接治理的可选参数。
type WSOption func(*WSProxy)

// WithWsPingInterval 设置心跳间隔；<= 0 表示不主动发 ping。
func WithWsPingInterval(d time.Duration) WSOption {
	return func(p *WSProxy) { p.pingInterval = d }
}

// WithWsMaxDuration 设置单条连接的最大存活时长；<= 0 表示不限制。
func WithWsMaxDuration(d time.Duration) WSOption {
	return func(p *WSProxy) { p.maxDuration = d }
}

func NewWSProxy(userConn, supplierConn *websocket.Conn, timeout time.Duration, handler MessageHandler, usageHandler UsageHandler, opts ...WSOption) *WSProxy {
	p := &WSProxy{
		userConn:     userConn,
		supplierConn: supplierConn,
		timeout:      timeout,
		handler:      handler,
		usageHandler: usageHandler,
		// 两个 transfer 各写一次 done，但 Wait 只读一次；原来用无缓冲 channel，
		// 后结束的那个 goroutine 会永久阻塞在发送上，每条连接泄漏一个 goroutine。
		done:           make(chan struct{}, 2),
		stop:           make(chan struct{}),
		userClosed:     make(chan struct{}),
		supplierClosed: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *WSProxy) idleTimeout() time.Duration {
	if p.timeout <= 0 {
		return 2 * time.Minute
	}
	return p.timeout
}

func (p *WSProxy) connections() []*websocket.Conn {
	return []*websocket.Conn{p.userConn, p.supplierConn}
}

func (p *WSProxy) Start() {
	p.startOnce.Do(func() {
		// 对端回应 pong 时延长读超时，这样即使业务消息稀疏，
		// 心跳也能保住连接；完全没有流量的连接仍会被 idleTimeout 关掉。
		for _, conn := range p.connections() {
			if conn == nil {
				continue
			}
			c := conn
			c.SetPongHandler(func(string) error {
				return c.SetReadDeadline(time.Now().Add(p.idleTimeout()))
			})
		}

		go p.transfer(p.userConn, p.supplierConn, UserMessage, p.userClosed)
		go p.transfer(p.supplierConn, p.userConn, SupplierMessage, p.supplierClosed)

		if p.pingInterval > 0 {
			go p.heartbeat()
		}
		if p.maxDuration > 0 {
			p.deadlineTimer = time.AfterFunc(p.maxDuration, func() {
				logger.SysLog("websocket connection reached max duration, closing")
				p.Close()
			})
		}
	})
}

func (p *WSProxy) heartbeat() {
	ticker := time.NewTicker(p.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			deadline := time.Now().Add(5 * time.Second)
			for _, conn := range p.connections() {
				if conn == nil {
					continue
				}
				// WriteControl 允许与其他写操作并发，不需要额外加锁。
				if err := conn.WriteControl(websocket.PingMessage, []byte("one-hub-ping"), deadline); err != nil {
					p.Close()
					return
				}
			}
		}
	}
}

func (p *WSProxy) Wait() {
	<-p.done
}

// Close 幂等：心跳、最大时长、任一侧断开都可能触发，重复关闭不应 panic。
func (p *WSProxy) Close() {
	p.closeOnce.Do(func() {
		close(p.stop)
		if p.deadlineTimer != nil {
			p.deadlineTimer.Stop()
		}
		for _, conn := range p.connections() {
			if conn != nil {
				_ = conn.Close()
			}
		}
	})
}

func (p *WSProxy) UserClosed() <-chan struct{} {
	return p.userClosed
}

func (p *WSProxy) SupplierClosed() <-chan struct{} {
	return p.supplierClosed
}

func (p *WSProxy) transfer(src, dst *websocket.Conn, source MessageSource, closed chan<- struct{}) {
	defer func() {
		close(closed)
		p.done <- struct{}{}
	}()

	for {
		src.SetReadDeadline(time.Now().Add(p.idleTimeout()))

		messageType, message, err := src.ReadMessage()
		if err != nil {
			logger.SysError(fmt.Sprintf("source: %d, ReadMessage error: %s", source, err.Error()))
			return
		}

		if p.handler != nil {
			shouldContinue, usage, newMessage, err := p.handler(source, messageType, message)
			if err != nil {
				errMsg := []byte(err.Error())
				dst.WriteMessage(websocket.TextMessage, errMsg)
				logger.SysError(fmt.Sprintf("source: %d, handler error: %s", source, err.Error()))
				return
			}

			if !shouldContinue {
				return
			}

			if newMessage != nil {
				message = newMessage
			}

			if usage != nil && p.usageHandler != nil {
				err := p.usageHandler(usage)
				if err != nil {
					dst.WriteMessage(websocket.TextMessage, message)
					errMsg := []byte(err.Error())
					dst.WriteMessage(websocket.TextMessage, errMsg)
					logger.SysError(fmt.Sprintf("source: %d, usageHandler error: %s", source, err.Error()))
					return
				}
			}
		}

		err = dst.WriteMessage(messageType, message)
		if err != nil {
			logger.SysError(fmt.Sprintf("source: %d, WriteMessage error: %s", source, err.Error()))
			return
		}
	}
}
