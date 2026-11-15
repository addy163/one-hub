package requester

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSProxyMaxDurationAndIdempotentClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	connA, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	connB, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	proxy := NewWSProxy(connA, connB, 2*time.Second, nil, nil,
		WithWsMaxDuration(150*time.Millisecond),
		WithWsPingInterval(40*time.Millisecond),
	)
	proxy.Start()

	done := make(chan struct{})
	go func() {
		proxy.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("max duration did not close the websocket connection")
	}

	// 心跳、最大时长、任一侧断开都会调用 Close，重复调用必须安全
	proxy.Close()
	proxy.Close()
	if _, _, err := connA.ReadMessage(); err == nil {
		t.Error("user connection should be closed")
	}
}

func TestWSProxyIdleTimeoutFallback(t *testing.T) {
	proxy := NewWSProxy(nil, nil, 0, nil, nil)
	if got := proxy.idleTimeout(); got != 2*time.Minute {
		t.Errorf("idleTimeout fallback = %v, want 2m", got)
	}
	proxy = NewWSProxy(nil, nil, 30*time.Second, nil, nil)
	if got := proxy.idleTimeout(); got != 30*time.Second {
		t.Errorf("idleTimeout = %v, want 30s", got)
	}
}
