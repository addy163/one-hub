package middleware

import (
	"net/http"
	"one-api/common/config"

	"github.com/gin-gonic/gin"
)

// NativeProtocolGate 把原生协议别名路由的开关从「启动时注册」变成「请求时判定」。
// 这样后台修改 passthrough_native_protocol 后立即生效，不必重启服务。
func NativeProtocolGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.NativeProtocolEnabled() {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
