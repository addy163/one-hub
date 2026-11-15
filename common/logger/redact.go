package logger

import (
	"regexp"

	"go.uber.org/zap/zapcore"
)

// 上一轮加固只按「参数名」脱敏了 Gin 访问日志的 query，第三方通知渠道错误里的
// 凭据并不会经过那里：例如 Telegram 把 bot token 直接拼在 URL 路径里
// (https://api.telegram.org/bot<id>:<secret>/getMe)，Go 的 url.Error 会把整条 URL
// 原样带进错误文本，再由 SysError / LogError 写进日志文件和管理后台的内存日志。
// 因此在 zap core 这一层统一兜底，覆盖所有日志出口（访问日志、系统日志、第三方错误）。
var sensitivePatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// Telegram: bot<id>:<secret>，无论出现在哪个 host 的路径里
	{regexp.MustCompile(`(?i)\bbot\d{6,}:[A-Za-z0-9_-]{20,}`), "bot<redacted>"},
	// 飞书 / 企业微信 / 钉钉 webhook：凭据在路径或 query 里
	{regexp.MustCompile(`(?i)(https?://open\.feishu\.cn/open-apis/bot/v2/hook/)[^\s"'<>]+`), "${1}<redacted>"},
	{regexp.MustCompile(`(?i)(https?://qyapi\.weixin\.qq\.com/cgi-bin/webhook/send\?key=)[^\s"'<>]+`), "${1}<redacted>"},
	{regexp.MustCompile(`(?i)(https?://oapi\.dingtalk\.com/robot/send\?access_token=)[^\s"'<>]+`), "${1}<redacted>"},
	// 通用 query / 表单形态的敏感参数
	{regexp.MustCompile(`(?i)\b(access_token|api_key|apikey|token|key|secret|password|pushkey|sign|signature|code|state)=[^&\s"'<>` + "`" + `]{6,}`), "${1}=<redacted>"},
	// 常见 API 凭据前缀
	{regexp.MustCompile(`\b(?:ghp_|gho_|ghs_|ghu_|github_pat_)[A-Za-z0-9_]{20,}`), "<redacted>"},
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{32,}`), "<redacted>"},
}

// RedactSensitive 去掉日志文本里可能夹带的凭据。可重复调用，结果幂等。
func RedactSensitive(s string) string {
	if s == "" {
		return s
	}
	for _, p := range sensitivePatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	return s
}

// redactCore 在消息真正写入文件之前做最后一次脱敏，
// 这样即使调用方绕过 SysLog / LogError 直接用 Logger.Info 也不会漏。
type redactCore struct {
	zapcore.Core
}

func (c redactCore) With(fields []zapcore.Field) zapcore.Core {
	return redactCore{Core: c.Core.With(fields)}
}

func (c redactCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c redactCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	entry.Message = RedactSensitive(entry.Message)
	for i := range fields {
		if fields[i].Type == zapcore.StringType {
			fields[i].String = RedactSensitive(fields[i].String)
		}
	}
	return c.Core.Write(entry, fields)
}
