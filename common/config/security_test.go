package config

import "testing"

func TestAllowWebsocketOrigin(t *testing.T) {
	defer func() { WebsocketAllowedOrigins = nil }()

	// 默认（未配置白名单）：保持历史行为，任意来源都允许
	WebsocketAllowedOrigins = nil
	if !AllowWebsocketOrigin("https://evil.example.com") {
		t.Fatal("default must stay compatible: any browser origin should be allowed")
	}
	// 服务端 SDK / curl 没有 Origin 头，任何情况下都必须放行
	WebsocketAllowedOrigins = []string{"https://chat.example.com"}
	if !AllowWebsocketOrigin("") {
		t.Fatal("non-browser clients have no Origin header and must always be allowed")
	}

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://chat.example.com", true},
		{"https://chat.example.com/", true},
		{"HTTPS://CHAT.EXAMPLE.COM", true},
		{"https://evil.example.com", false},
		{"http://chat.example.com", false},
		{"https://chat.example.com.evil.com", false},
	}
	for _, tc := range cases {
		if got := AllowWebsocketOrigin(tc.origin); got != tc.want {
			t.Errorf("origin %q: got %v, want %v", tc.origin, got, tc.want)
		}
	}

	WebsocketAllowedOrigins = []string{"*"}
	if !AllowWebsocketOrigin("https://evil.example.com") {
		t.Fatal("explicit wildcard must allow all origins")
	}
}

func TestSessionVersionStable(t *testing.T) {
	old := SessionSecret
	defer func() { SessionSecret = old }()
	SessionSecret = "test-only-session-secret-0123456789"
	a := SessionVersion("token-a")
	b := SessionVersion("token-a")
	c := SessionVersion("token-b")
	if a != b || a == c || len(a) != 32 {
		t.Fatalf("unexpected session version: %q %q %q", a, b, c)
	}
}
