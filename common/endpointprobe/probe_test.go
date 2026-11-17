package endpointprobe

import (
	"net/http"
	"net/http/httptest"
	"one-api/common/config"
	"testing"
	"time"
)

func TestProbeOne(t *testing.T) {
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus string
	}{
		{
			name: "healthy endpoint",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/status" {
					t.Errorf("unexpected probe path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"data":{"version":"test"}}`))
			},
			wantStatus: config.EndpointStatusUp,
		},
		{
			name: "http 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: config.EndpointStatusDown,
		},
		{
			name: "success false",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"success":false,"message":"boom"}`))
			},
			wantStatus: config.EndpointStatusDown,
		},
		{
			name: "html error page",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`<html><body>502 Bad Gateway</body></html>`))
			},
			wantStatus: config.EndpointStatusDown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			health := ProbeOne(server.URL, 800*time.Millisecond)
			if health.Status != tc.wantStatus {
				t.Fatalf("want %s, got %s (%s)", tc.wantStatus, health.Status, health.Error)
			}
			if health.CheckedAt == 0 {
				t.Error("CheckedAt should be filled")
			}
		})
	}
}

func TestProbeOneRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // 端口关闭后必然连接失败

	health := ProbeOne(url, 500*time.Millisecond)
	if health.Status != config.EndpointStatusDown {
		t.Fatalf("want down, got %s", health.Status)
	}
	if health.Error == "" {
		t.Error("error message should be recorded for admin")
	}
	if len(health.Error) > maxErrorLength {
		t.Errorf("error message should be truncated, got %d bytes", len(health.Error))
	}
}

func TestProbeAllSkipsDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	raw := `[
		{"id":"on","name":"on","url":"` + server.URL + `","enabled":true},
		{"id":"off","name":"off","url":"` + server.URL + `","enabled":false}
	]`
	got := ProbeAll(raw)
	// 同 URL 会去重，只保留第一条
	if len(got) != 1 {
		t.Fatalf("want 1 endpoint after dedupe, got %d", len(got))
	}
	if got[0].Health == nil || got[0].Health.Status != config.EndpointStatusUp {
		t.Fatalf("unexpected health: %+v", got[0].Health)
	}
}
