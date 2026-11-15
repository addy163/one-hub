package config

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// 运行时可管理配置。
//
// 设计目标：所有新加的安全 / 透传 / WebSocket 治理开关都支持两个入口：
//  1. config.yaml —— 部署时的默认值，随镜像 / 编排一起版本化；
//  2. 后台「系统设置 → 安全与透传」—— 立即生效的覆盖值，写入 options 表。
//
// 优先级：后台设置 > config.yaml > 代码默认值。GetOptions 里会标明当前值的来源，
// 避免「改了配置文件却不生效」这类困惑。除可信代理列表外都不需要重启。
const (
	RuntimeTypeBool   = "bool"
	RuntimeTypeInt    = "int"
	RuntimeTypeList   = "list"
	RuntimeTypeEnum   = "enum"
	RuntimeTypeString = "string"

	SessionSecureAuto = "auto"
	SessionSecureOn   = "on"
	SessionSecureOff  = "off"
)

// RuntimeOptionSpec 是后台设置面板的渲染描述，前端按 type 选择控件。
type RuntimeOptionSpec struct {
	Key             string   `json:"key"`
	Type            string   `json:"type"`
	Group           string   `json:"group"`
	Default         string   `json:"default"`
	Options         []string `json:"options,omitempty"`
	Min             int      `json:"min,omitempty"`
	Max             int      `json:"max,omitempty"`
	RestartRequired bool     `json:"restart_required"`
	Value           string   `json:"value"`
	Source          string   `json:"source"`
}

var (
	runtimeMu sync.RWMutex

	// configRuntimeValues 是启动时从 config.yaml / 环境变量读到的值快照。
	// 后台「恢复配置文件默认」用它把被 options 表覆盖的项还原，
	// 否则删除数据库记录后内存里仍留着旧值。
	configRuntimeValues map[string]string

	// 会话 / 跨域 / SSRF / 代理 / WebSocket 治理的运行时状态。
	CorsAllowedOrigins       []string
	CorsAllowCredentials     bool
	SsrfAllowPrivateImage    bool
	TrustedProxies           []string
	WsMaxConnections         = 200
	WsMaxConnectionsPerToken = 10
	WsIdleTimeoutSeconds     = 120
	WsMaxConnectionSeconds   = 3600
	WsPingIntervalSeconds    = 30
)

// runtimeOptionSpecs 顺序即后台面板的展示顺序。
var runtimeOptionSpecs = []RuntimeOptionSpec{
	{Key: "PassthroughExtraBody", Type: RuntimeTypeBool, Group: "passthrough", Default: "true"},
	{Key: "PassthroughNativeProtocol", Type: RuntimeTypeBool, Group: "passthrough", Default: "true"},
	{Key: "SessionSecureMode", Type: RuntimeTypeEnum, Group: "session", Default: SessionSecureAuto,
		Options: []string{SessionSecureAuto, SessionSecureOn, SessionSecureOff}},
	{Key: "SessionMaxAge", Type: RuntimeTypeInt, Group: "session", Default: "604800", Min: 60, Max: 2592000},
	{Key: "TrustedProxies", Type: RuntimeTypeList, Group: "session", Default: "127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16", RestartRequired: true},
	{Key: "CorsAllowedOrigins", Type: RuntimeTypeList, Group: "cors", Default: ""},
	{Key: "CorsAllowCredentials", Type: RuntimeTypeBool, Group: "cors", Default: "false"},
	{Key: "SsrfAllowPrivateImage", Type: RuntimeTypeBool, Group: "ssrf", Default: "false"},
	{Key: "WebsocketAllowedOrigins", Type: RuntimeTypeList, Group: "websocket", Default: ""},
	{Key: "WsMaxConnections", Type: RuntimeTypeInt, Group: "websocket", Default: "200", Min: 1, Max: 100000},
	{Key: "WsMaxConnectionsPerToken", Type: RuntimeTypeInt, Group: "websocket", Default: "10", Min: 1, Max: 10000},
	{Key: "WsIdleTimeoutSeconds", Type: RuntimeTypeInt, Group: "websocket", Default: "120", Min: 5, Max: 3600},
	{Key: "WsMaxConnectionSeconds", Type: RuntimeTypeInt, Group: "websocket", Default: "3600", Min: 10, Max: 86400},
	{Key: "WsPingIntervalSeconds", Type: RuntimeTypeInt, Group: "websocket", Default: "30", Min: 0, Max: 600},
}

func normalizeList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func listToString(items []string) string {
	return strings.Join(items, ",")
}

func clampInt(value, min, max, fallback int) int {
	if value < min || value > max {
		return fallback
	}
	return value
}

// InitRuntime 从 config.yaml / 环境变量读取默认值。只在启动时调用一次。
func InitRuntime() {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	CorsAllowedOrigins = normalizeList(viper.GetStringSlice("cors_allowed_origins"))
	CorsAllowCredentials = viper.GetBool("cors_allow_credentials")
	SsrfAllowPrivateImage = viper.GetBool("ssrf_allow_private_image")
	TrustedProxies = normalizeList(viper.GetStringSlice("trusted_proxies"))
	if len(TrustedProxies) == 0 {
		TrustedProxies = normalizeList([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	}

	WsMaxConnections = clampInt(viper.GetInt("ws_max_connections"), 1, 100000, 200)
	WsMaxConnectionsPerToken = clampInt(viper.GetInt("ws_max_connections_per_token"), 1, 10000, 10)
	WsIdleTimeoutSeconds = clampInt(viper.GetInt("ws_idle_timeout_seconds"), 5, 3600, 120)
	WsMaxConnectionSeconds = clampInt(viper.GetInt("ws_max_connection_seconds"), 10, 86400, 3600)
	WsPingIntervalSeconds = clampInt(viper.GetInt("ws_ping_interval_seconds"), 0, 600, 30)
}

// CaptureRuntimeConfigDefaults 必须在 InitConf 里、InitOptionMap 之前调用，
// 此时全局变量还是 config.yaml 的值，尚未被后台设置覆盖。
func CaptureRuntimeConfigDefaults() {
	values := make(map[string]string, len(runtimeOptionSpecs))
	for _, spec := range runtimeOptionSpecs {
		values[spec.Key] = GetRuntimeOption(spec.Key)
	}
	runtimeMu.Lock()
	configRuntimeValues = values
	runtimeMu.Unlock()
}

// ResetRuntimeOption 把某个运行配置项恢复为 config.yaml 的默认值。
func ResetRuntimeOption(key string) error {
	runtimeMu.RLock()
	value, ok := configRuntimeValues[key]
	runtimeMu.RUnlock()
	if !ok {
		return fmt.Errorf("未找到 %s 的配置文件默认值", key)
	}
	return SetRuntimeOption(key, value)
}

// InitSessionMode 读取 session_secure 的三态值。
// true/false 是历史布尔写法，auto 按请求的 X-Forwarded-Proto 动态决定。
func InitSessionMode() {
	raw := strings.ToLower(strings.TrimSpace(viper.GetString("session_secure")))
	mode := SessionSecureOff
	switch raw {
	case "true", "1", "on", "yes":
		mode = SessionSecureOn
	case SessionSecureAuto:
		mode = SessionSecureAuto
	}

	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	SessionSecureMode = mode
	SessionSecure = mode == SessionSecureOn
}

// IsRuntimeOption 判断一个后台配置 key 是否属于运行时配置。
func IsRuntimeOption(key string) bool {
	for i := range runtimeOptionSpecs {
		if runtimeOptionSpecs[i].Key == key {
			return true
		}
	}
	return false
}

// RuntimeOptions 返回带当前生效值的配置描述（source 由 controller 填充）。
func RuntimeOptions() []RuntimeOptionSpec {
	result := make([]RuntimeOptionSpec, 0, len(runtimeOptionSpecs))
	for _, spec := range runtimeOptionSpecs {
		spec.Value = GetRuntimeOption(spec.Key)
		result = append(result, spec)
	}
	return result
}

// RegisterRuntimeOptions 把运行时配置接入后台设置。
// 不传 defaultValue，避免注册动作把 config.yaml 的默认值覆盖掉；
// 只有 options 表里确实存在该 key 时，loadOptionsFromDatabase 才会覆盖。
func RegisterRuntimeOptions() {
	for _, spec := range runtimeOptionSpecs {
		key := spec.Key
		GlobalOption.RegisterCustom(key, func() string {
			return GetRuntimeOption(key)
		}, func(value string) error {
			return SetRuntimeOption(key, value)
		}, "")
	}
}

// GetRuntimeOption 返回当前生效值（字符串形式）。
func GetRuntimeOption(key string) string {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()

	switch key {
	case "PassthroughExtraBody":
		return strconv.FormatBool(PassthroughExtraBody)
	case "PassthroughNativeProtocol":
		return strconv.FormatBool(PassthroughNativeProtocol)
	case "SessionSecureMode":
		return SessionSecureMode
	case "SessionMaxAge":
		return strconv.Itoa(SessionMaxAge)
	case "TrustedProxies":
		return listToString(TrustedProxies)
	case "CorsAllowedOrigins":
		return listToString(CorsAllowedOrigins)
	case "CorsAllowCredentials":
		return strconv.FormatBool(CorsAllowCredentials)
	case "SsrfAllowPrivateImage":
		return strconv.FormatBool(SsrfAllowPrivateImage)
	case "WebsocketAllowedOrigins":
		return listToString(WebsocketAllowedOrigins)
	case "WsMaxConnections":
		return strconv.Itoa(WsMaxConnections)
	case "WsMaxConnectionsPerToken":
		return strconv.Itoa(WsMaxConnectionsPerToken)
	case "WsIdleTimeoutSeconds":
		return strconv.Itoa(WsIdleTimeoutSeconds)
	case "WsMaxConnectionSeconds":
		return strconv.Itoa(WsMaxConnectionSeconds)
	case "WsPingIntervalSeconds":
		return strconv.Itoa(WsPingIntervalSeconds)
	default:
		return ""
	}
}

func parseRuntimeBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "on", "yes":
		return true, nil
	case "false", "0", "off", "no":
		return false, nil
	}
	return false, fmt.Errorf("只能是 true 或 false")
}

func findRuntimeSpec(key string) *RuntimeOptionSpec {
	for i := range runtimeOptionSpecs {
		if runtimeOptionSpecs[i].Key == key {
			return &runtimeOptionSpecs[i]
		}
	}
	return nil
}

func parseRuntimeInt(value string, spec *RuntimeOptionSpec) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("请输入整数")
	}
	if n < spec.Min || n > spec.Max {
		return 0, fmt.Errorf("取值范围 %d ~ %d", spec.Min, spec.Max)
	}
	return n, nil
}

// ValidateRuntimeOption 只做校验、不改状态。后台保存前先调用它，
// 避免非法值被写进 options 表后，重启时又覆盖掉 config.yaml 的默认值。
func ValidateRuntimeOption(key, value string) error {
	spec := findRuntimeSpec(key)
	if spec == nil {
		return fmt.Errorf("未知的运行时配置项: %s", key)
	}
	switch spec.Type {
	case RuntimeTypeBool:
		_, err := parseRuntimeBool(value)
		return err
	case RuntimeTypeInt:
		_, err := parseRuntimeInt(value, spec)
		return err
	case RuntimeTypeEnum:
		if key == "SessionSecureMode" {
			v := strings.ToLower(strings.TrimSpace(value))
			if v == "true" {
				v = SessionSecureOn
			} else if v == "false" {
				v = SessionSecureOff
			}
			if v != SessionSecureAuto && v != SessionSecureOn && v != SessionSecureOff {
				return fmt.Errorf("只能是 auto / on / off")
			}
		}
	}
	return nil
}

// SetRuntimeOption 校验并即时更新运行时配置。后台保存失败时不会污染内存状态。
func SetRuntimeOption(key, value string) error {
	if err := ValidateRuntimeOption(key, value); err != nil {
		return err
	}
	spec := findRuntimeSpec(key)

	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	switch key {
	case "PassthroughExtraBody":
		v, err := parseRuntimeBool(value)
		if err != nil {
			return err
		}
		PassthroughExtraBody = v
	case "PassthroughNativeProtocol":
		v, err := parseRuntimeBool(value)
		if err != nil {
			return err
		}
		PassthroughNativeProtocol = v
	case "SessionSecureMode":
		v := strings.ToLower(strings.TrimSpace(value))
		if v == "true" {
			v = SessionSecureOn
		} else if v == "false" {
			v = SessionSecureOff
		}
		if v != SessionSecureAuto && v != SessionSecureOn && v != SessionSecureOff {
			return fmt.Errorf("只能是 auto / on / off")
		}
		SessionSecureMode = v
		SessionSecure = v == SessionSecureOn
	case "SessionMaxAge":
		v, err := parseRuntimeInt(value, spec)
		if err != nil {
			return err
		}
		SessionMaxAge = v
	case "TrustedProxies":
		TrustedProxies = normalizeList([]string{value})
	case "CorsAllowedOrigins":
		CorsAllowedOrigins = normalizeList([]string{value})
	case "CorsAllowCredentials":
		v, err := parseRuntimeBool(value)
		if err != nil {
			return err
		}
		CorsAllowCredentials = v
	case "SsrfAllowPrivateImage":
		v, err := parseRuntimeBool(value)
		if err != nil {
			return err
		}
		SsrfAllowPrivateImage = v
	case "WebsocketAllowedOrigins":
		WebsocketAllowedOrigins = normalizeList([]string{value})
	case "WsMaxConnections":
		v, err := parseRuntimeInt(value, spec)
		if err != nil {
			return err
		}
		WsMaxConnections = v
	case "WsMaxConnectionsPerToken":
		v, err := parseRuntimeInt(value, spec)
		if err != nil {
			return err
		}
		WsMaxConnectionsPerToken = v
	case "WsIdleTimeoutSeconds":
		v, err := parseRuntimeInt(value, spec)
		if err != nil {
			return err
		}
		WsIdleTimeoutSeconds = v
	case "WsMaxConnectionSeconds":
		v, err := parseRuntimeInt(value, spec)
		if err != nil {
			return err
		}
		WsMaxConnectionSeconds = v
	case "WsPingIntervalSeconds":
		v, err := parseRuntimeInt(value, spec)
		if err != nil {
			return err
		}
		WsPingIntervalSeconds = v
	}
	return nil
}

// ---- 供各模块读取的带锁 getter（避免后台修改时与请求并发产生数据竞争）----

func PassthroughExtraBodyEnabled() bool {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return PassthroughExtraBody
}

func NativeProtocolEnabled() bool {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return PassthroughNativeProtocol
}

func CookieMode() string {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return SessionSecureMode
}

func CookieMaxAge() int {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return SessionMaxAge
}

// CookieSecureForRequest 决定本次响应是否为会话 Cookie 加 Secure 标记。
// auto 模式下：直连 HTTPS 或经可信反代（X-Forwarded-Proto: https）时加 Secure，
// 内网 HTTP 直连时不加，因此公网反代与内网管理可以共存。
func CookieSecureForRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch CookieMode() {
	case SessionSecureOn:
		return true
	case SessionSecureAuto:
		if r.TLS != nil {
			return true
		}
		proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
		if strings.EqualFold(proto, "https") {
			return true
		}
		return strings.Contains(strings.ToLower(r.Header.Get("Forwarded")), "proto=https")
	default:
		return false
	}
}

func CorsSettings() ([]string, bool) {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return append([]string(nil), CorsAllowedOrigins...), CorsAllowCredentials
}

func PrivateImageAllowed() bool {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return SsrfAllowPrivateImage
}

func TrustedProxyList() []string {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return append([]string(nil), TrustedProxies...)
}

func WebsocketOrigins() []string {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return append([]string(nil), WebsocketAllowedOrigins...)
}

type WsLimits struct {
	MaxConnections         int
	MaxConnectionsPerToken int
	IdleTimeoutSeconds     int
	MaxConnectionSeconds   int
	PingIntervalSeconds    int
}

func CurrentWsLimits() WsLimits {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return WsLimits{
		MaxConnections:         WsMaxConnections,
		MaxConnectionsPerToken: WsMaxConnectionsPerToken,
		IdleTimeoutSeconds:     WsIdleTimeoutSeconds,
		MaxConnectionSeconds:   WsMaxConnectionSeconds,
		PingIntervalSeconds:    WsPingIntervalSeconds,
	}
}
