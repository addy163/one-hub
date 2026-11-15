package middleware

import (
	"net/http"
	"net/http/httptest"
	"one-api/common/config"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSUsesRuntimeConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalOrigins := config.GetRuntimeOption("CorsAllowedOrigins")
	originalCredentials := config.GetRuntimeOption("CorsAllowCredentials")
	defer func() {
		_ = config.SetRuntimeOption("CorsAllowedOrigins", originalOrigins)
		_ = config.SetRuntimeOption("CorsAllowCredentials", originalCredentials)
	}()

	engine := gin.New()
	engine.Use(CORS())
	engine.GET("/api/status", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(origin, method string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/status", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}

	// 默认：通配来源，但绝不回显 Allow-Credentials
	_ = config.SetRuntimeOption("CorsAllowedOrigins", "")
	_ = config.SetRuntimeOption("CorsAllowCredentials", "true")
	w := do("https://evil.example.com", http.MethodGet)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("default allow origin = %q, want *", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("wildcard origin must not allow credentials, got %q", got)
	}

	// 白名单：只回显匹配的 Origin，并按需允许凭据
	_ = config.SetRuntimeOption("CorsAllowedOrigins", "https://api.uuz.ai,https://api.uuai.cc")
	w = do("https://api.uuai.cc", http.MethodGet)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://api.uuai.cc" {
		t.Errorf("allowlisted origin = %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("allowlisted credentials = %q", got)
	}
	w = do("https://evil.example.com", http.MethodGet)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("non-allowlisted origin must get no CORS header, got %q", got)
	}

	// 预检请求
	preflight := httptest.NewRequest(http.MethodOptions, "/api/status", nil)
	preflight.Header.Set("Origin", "https://api.uuz.ai")
	preflight.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, preflight)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "authorization,content-type" {
		t.Errorf("preflight allow headers = %q", got)
	}

	// 后台关闭凭据后应即时生效，不需要重启
	_ = config.SetRuntimeOption("CorsAllowCredentials", "false")
	w = do("https://api.uuz.ai", http.MethodGet)
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("credentials should be gone after runtime update, got %q", got)
	}
}
