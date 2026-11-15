package config

import (
	"strings"
	"time"

	"one-api/common/utils"

	"github.com/spf13/viper"
)

func InitConf() {
	defaultConfig()
	setEnv()
	Language = viper.GetString("language")
	IsMasterNode = viper.GetString("node_type") != "slave"
	RequestInterval = time.Duration(viper.GetInt("polling_interval")) * time.Second
	// 拒绝部署模板里的占位密钥和过短密钥，避免攻击者伪造任意 role 的 Cookie
	SessionSecret, WeakSessionSecretReplaced = NormalizeSessionSecret(utils.GetOrDefault("session_secret", SessionSecret))
	InitSecurity()
	InitRelay()
	UserInvoiceMonth = viper.GetBool("user_invoice_month")
	GitHubProxy = viper.GetString("github_proxy")
	MCP_ENABLE = viper.GetBool("mcp.enable") != false
	UPTIMEKUMA_ENABLE = viper.GetBool("uptime_kuma.enable") != false
	UPTIMEKUMA_DOMAIN = viper.GetString("uptime_kuma.domain")
	UPTIMEKUMA_STATUS_PAGE_NAME = viper.GetString("uptime_kuma.status_page_name")
}

func setEnv() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func defaultConfig() {
	viper.SetDefault("port", "3000")
	viper.SetDefault("gin_mode", "release")
	viper.SetDefault("log_dir", "./logs")
	viper.SetDefault("sqlite_path", "one-api.db")
	viper.SetDefault("sqlite_busy_timeout", 3000)
	viper.SetDefault("sync_frequency", 600)
	viper.SetDefault("batch_update_interval", 5)
	viper.SetDefault("global.api_rate_limit", 300)
	viper.SetDefault("global.web_rate_limit", 180)
	viper.SetDefault("connect_timeout", 5)
	viper.SetDefault("auto_price_updates", false)
	viper.SetDefault("auto_price_updates_mode", "system")
	viper.SetDefault("auto_price_updates_interval", 1440)
	viper.SetDefault("update_price_service", "https://raw.githubusercontent.com/MartialBE/one-api/prices/prices.json")
	viper.SetDefault("language", "zh_CN")
	viper.SetDefault("favicon", "")
	viper.SetDefault("user_invoice_month", false)
	viper.SetDefault("mcp.enable", false)
	viper.SetDefault("uptime_kuma.enable", false)
	viper.SetDefault("uptime_kuma.domain", "")
	viper.SetDefault("uptime_kuma.status_page_name", "")
	// 请求参数无损透传默认开启，解决固定 struct 重序列化丢弃新字段的问题
	viper.SetDefault("passthrough_extra_body", true)
	viper.SetDefault("passthrough_native_protocol", true)
}
