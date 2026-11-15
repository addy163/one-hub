package middleware

import (
	"net/http/httptest"
	"one-api/common/config"
	"strconv"
	"strings"
	"testing"
)

func TestDynamicCookieStoreSecureFlag(t *testing.T) {
	original := config.CookieMode()
	defer func() { _ = config.SetRuntimeOption("SessionSecureMode", original) }()
	if err := config.SetRuntimeOption("SessionSecureMode", config.SessionSecureAuto); err != nil {
		t.Fatal(err)
	}

	store := NewDynamicCookieStore(
		[]byte("test-auth-key-must-be-at-least-32-bytes!!"),
		config.SessionBlockKey(),
	)

	cases := []struct {
		name       string
		headers    map[string]string
		wantSecure bool
	}{
		{"internal http direct", nil, false},
		{"public https reverse proxy", map[string]string{"X-Forwarded-Proto": "https"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://127.0.0.1:3010/api/status", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}
			session, err := store.New(req, "session")
			if err != nil {
				t.Fatal(err)
			}
			session.Values["id"] = 1
			recorder := httptest.NewRecorder()
			if err := store.Save(req, recorder, session); err != nil {
				t.Fatal(err)
			}
			cookie := recorder.Header().Get("Set-Cookie")
			if cookie == "" {
				t.Fatal("no Set-Cookie generated")
			}
			gotSecure := strings.Contains(cookie, "Secure")
			if gotSecure != tc.wantSecure {
				t.Errorf("cookie %q: Secure=%v, want %v", cookie, gotSecure, tc.wantSecure)
			}
			if !strings.Contains(cookie, "HttpOnly") {
				t.Errorf("cookie must stay HttpOnly: %q", cookie)
			}
		})
	}
}

// 与浏览器行为一致：Secure Cookie 只在 HTTPS 请求里回传，
// auto 模式用于同时支持公网反代与内网 HTTP 管理入口。
func TestDynamicCookieStoreMaxAgeFromRuntime(t *testing.T) {
	original := config.CookieMaxAge()
	defer func() { _ = config.SetRuntimeOption("SessionMaxAge", strconv.Itoa(original)) }()

	if err := config.SetRuntimeOption("SessionMaxAge", "3600"); err != nil {
		t.Fatal(err)
	}
	store := NewDynamicCookieStore(
		[]byte("test-auth-key-must-be-at-least-32-bytes!!"),
		config.SessionBlockKey(),
	)
	req := httptest.NewRequest("GET", "http://127.0.0.1:3010/api/status", nil)
	session, err := store.New(req, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["id"] = 1
	recorder := httptest.NewRecorder()
	if err := store.Save(req, recorder, session); err != nil {
		t.Fatal(err)
	}
	if cookie := recorder.Header().Get("Set-Cookie"); !strings.Contains(cookie, "Max-Age=3600") {
		t.Errorf("runtime SessionMaxAge not applied: %q", cookie)
	}
}
