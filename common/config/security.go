package config

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/spf13/viper"
)

// 会话安全配置。
// SessionMaxAge 默认 7 天（原来是 30 天），可通过 session_max_age 覆盖。
// SessionSecure 默认 false：内网 HTTP 部署若强制 Secure 会导致无法登录，
// 公网 HTTPS 部署请在 config.yaml 中显式设置 session_secure: true。
var (
	SessionMaxAge = 7 * 24 * 60 * 60
	SessionSecure = false

	// WeakSessionSecretReplaced 表示启动配置里的 session_secret 是弱值/占位值，
	// 已被自动替换为随机密钥。此时所有旧会话会失效，属于预期的安全行为。
	WeakSessionSecretReplaced = false
)

const (
	sessionVersionContext = "one-hub/session-version/v1"
	sessionBlockKeyContext = "one-hub/session-encryption/v1"
)

func InitSecurity() {
	if v := viper.GetInt("session_max_age"); v > 0 {
		SessionMaxAge = v
	}
	SessionSecure = viper.GetBool("session_secure")
}

// NormalizeSessionSecret 拒绝空值、过短值和部署模板里的占位值。
// 返回规范化后的密钥，以及是否发生了替换。
func NormalizeSessionSecret(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	lower := strings.ToLower(s)

	weakValues := map[string]bool{
		"":                    true,
		"random_string":       true,
		"changeme":            true,
		"change_me":           true,
		"secret":              true,
		"session_secret":      true,
		"your_session_secret": true,
		"123456":              true,
	}

	if len(s) >= 32 && !weakValues[lower] {
		return s, false
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败时退化为基于 HMAC 的派生值，仍强于任何占位串
		mac := hmac.New(sha256.New, []byte(s))
		mac.Write([]byte(sessionVersionContext))
		return hex.EncodeToString(mac.Sum(nil)), true
	}
	return hex.EncodeToString(buf), true
}

// SessionBlockKey 从 SessionSecret 派生 32 字节密钥，用于加密（而非仅签名）会话内容。
// 单独派生，避免与签名密钥复用同一个字节串。
func SessionBlockKey() []byte {
	mac := hmac.New(sha256.New, []byte(SessionSecret))
	mac.Write([]byte(sessionBlockKeyContext))
	return mac.Sum(nil)
}

// SessionVersion 由用户当前的 access_token 派生，作为服务端会话版本号。
// 轮换 access_token（改密 / 重置密码 / 重新生成个人令牌）会让所有已签发 Cookie 立即失效，
// 这是 Cookie 泄露后最关键的应急止损手段：用户改一次密码即可踢掉所有会话。
func SessionVersion(accessToken string) string {
	mac := hmac.New(sha256.New, []byte(SessionSecret))
	mac.Write([]byte(sessionVersionContext))
	mac.Write([]byte(accessToken))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}
