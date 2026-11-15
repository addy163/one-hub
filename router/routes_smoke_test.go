package router

import (
	"net/http"
	"net/http/httptest"
	"one-api/common/config"
	"one-api/middleware"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// 路由注册冒烟测试：Gin 在同一层级混合静态段 / 参数段 / catch-all 时
// 会在注册阶段直接 panic，这里覆盖开关的两种取值，确保服务能正常启动。
// 原生协议别名现在始终注册、由 gate 在请求时判定，所以两种取值的路由数相同。
func TestRelayRouterRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := config.GetRuntimeOption("PassthroughNativeProtocol")
	defer func() { _ = config.SetRuntimeOption("PassthroughNativeProtocol", original) }()

	counts := make([]int, 0, 2)
	for _, enabled := range []bool{true, false} {
		if err := config.SetRuntimeOption("PassthroughNativeProtocol", strconv.FormatBool(enabled)); err != nil {
			t.Fatal(err)
		}
		engine := gin.New()
		SetRelayRouter(engine)
		routes := engine.Routes()
		if len(routes) == 0 {
			t.Fatalf("no routes registered when native=%v", enabled)
		}
		counts = append(counts, len(routes))
		t.Logf("native=%v registered %d routes", enabled, len(routes))
	}
	if counts[0] != counts[1] {
		t.Errorf("native protocol switch must not change route count: %v", counts)
	}
}

func TestNativeProtocolGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := config.GetRuntimeOption("PassthroughNativeProtocol")
	defer func() { _ = config.SetRuntimeOption("PassthroughNativeProtocol", original) }()

	engine := gin.New()
	engine.POST("/probe", middleware.NativeProtocolGate(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	probe := func() int {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/probe", nil))
		return w.Code
	}

	if err := config.SetRuntimeOption("PassthroughNativeProtocol", "false"); err != nil {
		t.Fatal(err)
	}
	if code := probe(); code != http.StatusNotFound {
		t.Errorf("disabled native protocol returned %d, want 404", code)
	}

	if err := config.SetRuntimeOption("PassthroughNativeProtocol", "true"); err != nil {
		t.Fatal(err)
	}
	if code := probe(); code != http.StatusNoContent {
		t.Errorf("enabled native protocol returned %d, want 204", code)
	}
}
