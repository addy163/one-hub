package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func CORS() gin.HandlerFunc {
	cfg := cors.DefaultConfig()

	origins := viper.GetStringSlice("cors_allowed_origins")
	if len(origins) > 0 {
		// 显式白名单模式：仅这些来源可跨域，是否允许携带凭据由配置决定
		cfg.AllowAllOrigins = false
		cfg.AllowOrigins = origins
		cfg.AllowCredentials = viper.GetBool("cors_allow_credentials")
	} else {
		// 通配来源模式：必须禁止携带凭据。
		// 原实现 AllowAllOrigins + AllowCredentials 同时为 true，浏览器会回显
		// 任意 Origin 并允许带上会话 Cookie，等于把管理接口暴露给任意站点。
		cfg.AllowAllOrigins = true
		cfg.AllowCredentials = false
	}

	cfg.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	cfg.AllowHeaders = []string{"*"}
	return cors.New(cfg)
}
