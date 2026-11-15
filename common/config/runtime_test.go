package config

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"
)

func TestNormalizeList(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"a,b", " c ", "", "d"}, []string{"a", "b", "c", "d"}},
		{nil, []string{}},
		{[]string{"https://api.uuz.ai,https://api.uuai.cc"}, []string{"https://api.uuz.ai", "https://api.uuai.cc"}},
	}
	for _, tc := range cases {
		if got := normalizeList(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("normalizeList(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestValidateRuntimeOption(t *testing.T) {
	cases := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"PassthroughExtraBody", "true", false},
		{"PassthroughExtraBody", "FALSE", false},
		{"PassthroughExtraBody", "maybe", true},
		{"PassthroughNativeProtocol", "0", false},
		{"SessionSecureMode", "auto", false},
		{"SessionSecureMode", "on", false},
		{"SessionSecureMode", "off", false},
		{"SessionSecureMode", "true", false},
		{"SessionSecureMode", "sometimes", true},
		{"SessionMaxAge", "3600", false},
		{"SessionMaxAge", "1", true},
		{"SessionMaxAge", "abc", true},
		{"WsMaxConnections", "500", false},
		{"WsMaxConnections", "0", true},
		{"WsPingIntervalSeconds", "0", false},
		{"CorsAllowedOrigins", "https://api.uuz.ai,https://api.uuai.cc", false},
		{"WebsocketAllowedOrigins", "", false},
		{"UnknownKey", "x", true},
	}
	for _, tc := range cases {
		err := ValidateRuntimeOption(tc.key, tc.value)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateRuntimeOption(%q,%q) error=%v, wantErr=%v", tc.key, tc.value, err, tc.wantErr)
		}
	}
	// 非法值不能污染内存状态
	before := PassthroughExtraBodyEnabled()
	if err := SetRuntimeOption("PassthroughExtraBody", "not-a-bool"); err == nil {
		t.Fatal("invalid bool must be rejected")
	}
	if PassthroughExtraBodyEnabled() != before {
		t.Fatal("invalid value changed runtime state")
	}
}

func TestRuntimeOptionRoundTrip(t *testing.T) {
	RegisterRuntimeOptions()
	original := map[string]string{
		"PassthroughExtraBody":      GetRuntimeOption("PassthroughExtraBody"),
		"SessionSecureMode":         GetRuntimeOption("SessionSecureMode"),
		"SessionMaxAge":             GetRuntimeOption("SessionMaxAge"),
		"WsMaxConnections":          GetRuntimeOption("WsMaxConnections"),
		"CorsAllowedOrigins":        GetRuntimeOption("CorsAllowedOrigins"),
		"CorsAllowCredentials":      GetRuntimeOption("CorsAllowCredentials"),
		"WebsocketAllowedOrigins":   GetRuntimeOption("WebsocketAllowedOrigins"),
		"SsrfAllowPrivateImage":     GetRuntimeOption("SsrfAllowPrivateImage"),
		"WsPingIntervalSeconds":     GetRuntimeOption("WsPingIntervalSeconds"),
		"WsMaxConnectionSeconds":    GetRuntimeOption("WsMaxConnectionSeconds"),
		"WsIdleTimeoutSeconds":      GetRuntimeOption("WsIdleTimeoutSeconds"),
		"WsMaxConnectionsPerToken":  GetRuntimeOption("WsMaxConnectionsPerToken"),
		"TrustedProxies":            GetRuntimeOption("TrustedProxies"),
		"PassthroughNativeProtocol": GetRuntimeOption("PassthroughNativeProtocol"),
	}
	defer func() {
		for key, value := range original {
			_ = SetRuntimeOption(key, value)
		}
	}()

	if err := GlobalOption.Set("PassthroughExtraBody", "false"); err != nil {
		t.Fatal(err)
	}
	if PassthroughExtraBodyEnabled() {
		t.Fatal("GlobalOption.Set must update runtime state")
	}
	if got := GlobalOption.Get("PassthroughExtraBody"); got != "false" {
		t.Errorf("GlobalOption.Get = %q, want false", got)
	}

	if err := SetRuntimeOption("WsMaxConnections", "42"); err != nil {
		t.Fatal(err)
	}
	if got := CurrentWsLimits().MaxConnections; got != 42 {
		t.Errorf("WsMaxConnections = %d, want 42", got)
	}

	if err := SetRuntimeOption("CorsAllowedOrigins", "https://api.uuz.ai, https://api.uuai.cc"); err != nil {
		t.Fatal(err)
	}
	origins, _ := CorsSettings()
	if !reflect.DeepEqual(origins, []string{"https://api.uuz.ai", "https://api.uuai.cc"}) {
		t.Errorf("CorsAllowedOrigins = %v", origins)
	}

	if err := SetRuntimeOption("WebsocketAllowedOrigins", "https://api.uuz.ai"); err != nil {
		t.Fatal(err)
	}
	if !AllowWebsocketOrigin("https://api.uuz.ai") || AllowWebsocketOrigin("https://evil.example.com") {
		t.Error("WebsocketAllowedOrigins did not take effect")
	}

	specs := RuntimeOptions()
	if len(specs) != len(runtimeOptionSpecs) {
		t.Fatalf("runtime specs length mismatch: %d", len(specs))
	}
	for _, spec := range specs {
		if spec.Key == "" || spec.Group == "" || spec.Type == "" {
			t.Errorf("incomplete spec: %+v", spec)
		}
	}
}

func TestCookieSecureForRequest(t *testing.T) {
	original := CookieMode()
	defer func() { _ = SetRuntimeOption("SessionSecureMode", original) }()

	httpsProxy := httptestRequestWithHeaders(map[string]string{"X-Forwarded-Proto": "https, http"})
	httpDirect := httptestRequestWithHeaders(nil)
	forwarded := httptestRequestWithHeaders(map[string]string{"Forwarded": "for=1.2.3.4;proto=https"})
	tlsRequest := httptestRequestWithHeaders(nil)
	tlsRequest.TLS = &tls.ConnectionState{}

	cases := []struct {
		name string
		mode string
		req  *http.Request
		want bool
	}{
		{"auto http direct", SessionSecureAuto, httpDirect, false},
		{"auto https proxy", SessionSecureAuto, httpsProxy, true},
		{"auto forwarded proto", SessionSecureAuto, forwarded, true},
		{"auto direct tls", SessionSecureAuto, tlsRequest, true},
		{"off ignores proxy", SessionSecureOff, httpsProxy, false},
		{"on ignores http", SessionSecureOn, httpDirect, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := SetRuntimeOption("SessionSecureMode", tc.mode); err != nil {
				t.Fatal(err)
			}
			if got := CookieSecureForRequest(tc.req); got != tc.want {
				t.Errorf("CookieSecureForRequest = %v, want %v", got, tc.want)
			}
		})
	}
	if CookieSecureForRequest(nil) {
		t.Error("nil request must not panic and must not be secure")
	}
}

func httptestRequestWithHeaders(headers map[string]string) *http.Request {
	req := &http.Request{Header: make(http.Header)}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req
}
