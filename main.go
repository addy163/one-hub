package main

import (
	"embed"
	"fmt"
	"net/http"
	"one-api/cli"
	"one-api/common"
	"one-api/common/cache"
	"one-api/common/config"
	"one-api/common/endpointprobe"
	"one-api/common/logger"
	"one-api/common/notify"
	"one-api/common/oidc"
	"one-api/common/redis"
	"one-api/common/requester"
	"one-api/common/search"
	"one-api/common/storage"
	"one-api/common/telegram"
	"one-api/common/webauthn"
	"one-api/controller"
	"one-api/cron"
	"one-api/middleware"
	"one-api/model"
	"one-api/relay/task"
	"one-api/router"
	"one-api/safty"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

//go:embed web/build
var buildFS embed.FS

//go:embed web/build/index.html
var indexPage []byte

func main() {
	cli.InitCli()
	config.InitConf()
	if viper.GetString("log_level") == "debug" {
		config.Debug = true
	}

	logger.SetupLogger()
	logger.SysLog("One Hub " + config.Version + " started")

	// Initialize user token
	err := common.InitUserToken()
	if err != nil {
		logger.FatalLog("failed to initialize user token: " + err.Error())
	}

	// Initialize SQL Database
	model.SetupDB()
	defer model.CloseDB()
	// Initialize Redis
	redis.InitRedisClient()
	cache.InitCacheManager()
	// Initialize options
	model.InitOptionMap()
	// Initialize oidc
	oidc.InitOIDCConfig()
	// Initialize wenauthn
	webauthn.InitWebAuthn()
	model.NewPricing()
	model.HandleOldTokenMaxId()

	initMemoryCache()
	initSync()

	// 线路健康检查：按配置间隔探测每条线路的 /api/status
	endpointprobe.Start()

	common.InitTokenEncoders()
	requester.InitHttpClient()
	// Initialize Telegram bot
	telegram.InitTelegramBot()

	controller.InitMidjourneyTask()
	task.InitTask()
	notify.InitNotifier()
	cron.InitCron()
	storage.InitStorage()
	search.InitSearcher()
	// 初始化安全检查器
	safty.InitSaftyTools()
	// 初始化账单数据
	if config.UserInvoiceMonth {
		logger.SysLog("Enable User Invoice Monthly Data")
		go model.InsertStatisticsMonth()
	}
	initHttpServer()
}

func initMemoryCache() {
	if viper.GetBool("memory_cache_enabled") {
		config.MemoryCacheEnabled = true
	}

	if !config.MemoryCacheEnabled {
		return
	}

	syncFrequency := viper.GetInt("sync_frequency")
	model.TokenCacheSeconds = syncFrequency

	logger.SysLog("memory cache enabled")
	logger.SysLog(fmt.Sprintf("sync frequency: %d seconds", syncFrequency))
	go model.SyncOptions(syncFrequency)
	go SyncChannelCache(syncFrequency)
}

func initSync() {
	// go controller.AutomaticallyUpdateChannels(viper.GetInt("channel.update_frequency"))
	go controller.AutomaticallyTestChannels(viper.GetInt("channel.test_frequency"))
}

func initHttpServer() {
	if viper.GetString("gin_mode") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	server := gin.New()
	server.Use(gin.Recovery())
	server.Use(middleware.RequestId())
	middleware.SetUpLogger(server)

	trustedHeader := viper.GetString("trusted_header")
	if trustedHeader != "" {
		server.TrustedPlatform = trustedHeader
	}

	// 公网反代场景下，只信任来自可信代理的 X-Forwarded-For，避免客户端伪造 IP
	// 绕过全局速率限制。可在 config.yaml 的 trusted_proxies 或后台设置里调整。
	if err := server.SetTrustedProxies(config.TrustedProxyList()); err != nil {
		logger.SysError("SECURITY: trusted_proxies 配置无效，已退回 gin 默认信任策略: " + err.Error())
	}

	if config.WeakSessionSecretReplaced {
		logger.SysError("SECURITY: session_secret 为空或为弱占位值，已自动替换为随机密钥；所有旧会话需要重新登录。生产环境请在 config.yaml 中配置 32 位以上固定随机串。")
	}
	switch config.CookieMode() {
	case config.SessionSecureOff:
		logger.SysError("SECURITY: 会话 Cookie 当前完全禁用 Secure 标记；公网 HTTPS 部署建议设置 session_secure: auto 或 on。")
	case config.SessionSecureAuto:
		logger.SysLog("SECURITY: 会话 Cookie 使用 auto 模式（HTTPS / 反代请求加 Secure，内网 HTTP 不加）。")
	}

	// 第二个参数为加密密钥：原实现只签名不加密，Cookie 内容可被直接解出 id/role。
	// 注意：启用加密后所有旧版 Cookie 会失效，用户需重新登录一次。
	store := middleware.NewDynamicCookieStore([]byte(config.SessionSecret), config.SessionBlockKey())
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   config.CookieMaxAge(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	server.Use(sessions.Sessions("session", store))

	router.SetRouter(server, buildFS, indexPage)
	port := viper.GetString("port")

	err := server.Run(":" + port)
	if err != nil {
		logger.FatalLog("failed to start HTTP server: " + err.Error())
	}
}

func SyncChannelCache(frequency int) {
	// 只有 从 服务器端获取数据的时候才会用到
	if config.IsMasterNode {
		logger.SysLog("master node does't synchronize the channel")
		return
	}
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		logger.SysLog("syncing channels from database")
		model.ChannelGroup.Load()
		model.GlobalUserGroupRatio.Load()
		model.PricingInstance.Init()
		model.ModelOwnedBysInstance.Load()
	}
}
