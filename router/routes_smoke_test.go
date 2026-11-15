package router

import (
	"testing"

	"one-api/common/config"

	"github.com/gin-gonic/gin"
)

// 路由注册冒烟测试：Gin 在同一层级混合静态段 / 参数段 / catch-all 时
// 会在注册阶段直接 panic，这里覆盖开关的两种取值，确保服务能正常启动。
func TestRelayRouterRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := config.PassthroughNativeProtocol
	defer func() { config.PassthroughNativeProtocol = original }()

	for _, enabled := range []bool{true, false} {
		config.PassthroughNativeProtocol = enabled
		engine := gin.New()
		SetRelayRouter(engine)
		routes := engine.Routes()
		if len(routes) == 0 {
			t.Fatalf("no routes registered when native=%v", enabled)
		}
		t.Logf("native=%v registered %d routes", enabled, len(routes))
	}
}
