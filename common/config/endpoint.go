package config

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// 线路健康状态取值
const (
	EndpointStatusUp      = "up"
	EndpointStatusDown    = "down"
	EndpointStatusUnknown = "unknown"
)

// RelayEndpoint 是一条可以下发给用户的 API 线路。
// Groups 为空表示所有分组可见；Enabled 为 false 的线路不下发。
// Health 由服务端探测写入，不会从后台配置里读取。
type RelayEndpoint struct {
	Id      string               `json:"id"`
	Name    string               `json:"name"`
	Url     string               `json:"url"`
	Groups  []string             `json:"groups,omitempty"`
	Note    string               `json:"note,omitempty"`
	Enabled bool                 `json:"enabled"`
	Sort    int                  `json:"sort,omitempty"`
	Health  *RelayEndpointHealth `json:"health,omitempty"`
}

// RelayEndpointHealth 是一条线路最近一次探测的结果。
// LatencyMs 为 0 表示没拿到响应时间（例如直接连接失败）。
type RelayEndpointHealth struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	CheckedAt int64  `json:"checked_at"`
	Error     string `json:"error,omitempty"`
}

// healthStore 保存各线路最近一次探测结果，key 为规范化后的地址。
var healthStore = struct {
	sync.RWMutex
	items map[string]RelayEndpointHealth
}{items: make(map[string]RelayEndpointHealth)}

// SetEndpointHealth 记录一条线路的探测结果。
func SetEndpointHealth(rawUrl string, health RelayEndpointHealth) {
	key := strings.ToLower(strings.TrimRight(strings.TrimSpace(rawUrl), "/"))
	if key == "" {
		return
	}
	if health.CheckedAt == 0 {
		health.CheckedAt = time.Now().Unix()
	}
	healthStore.Lock()
	healthStore.items[key] = health
	healthStore.Unlock()
}

// GetEndpointHealth 返回一条线路最近一次的探测结果，没探测过时状态为 unknown。
func GetEndpointHealth(rawUrl string) RelayEndpointHealth {
	key := strings.ToLower(strings.TrimRight(strings.TrimSpace(rawUrl), "/"))
	healthStore.RLock()
	health, ok := healthStore.items[key]
	healthStore.RUnlock()
	if !ok {
		return RelayEndpointHealth{Status: EndpointStatusUnknown}
	}
	return health
}

// withHealth 给线路附加健康状态，返回新值以避免共享指针。
func withHealth(item RelayEndpoint) RelayEndpoint {
	health := GetEndpointHealth(item.Url)
	item.Health = &health
	return item
}

// ParseRelayEndpoints 解析后台配置的线路清单。
// 整体不是合法 JSON 时返回空切片；单条非法只丢弃该条，不影响其它线路。
// 返回值按 Sort 降序（与 ChatLinks 的排序语义保持一致）。
func ParseRelayEndpoints(raw string) []RelayEndpoint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []RelayEndpoint{}
	}

	var items []RelayEndpoint
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []RelayEndpoint{}
	}

	seen := make(map[string]bool, len(items))
	out := make([]RelayEndpoint, 0, len(items))
	for _, item := range items {
		normalized, ok := normalizeEndpointURL(item.Url)
		if !ok {
			continue
		}

		key := strings.ToLower(normalized)
		if seen[key] {
			continue
		}
		seen[key] = true

		item.Url = normalized
		item.Id = strings.TrimSpace(item.Id)
		if item.Id == "" {
			item.Id = key
		}
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			item.Name = normalized
		}
		item.Note = strings.TrimSpace(item.Note)
		item.Groups = normalizeGroups(item.Groups)

		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort != out[j].Sort {
			return out[i].Sort > out[j].Sort
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// RelayEndpointsForGroup 返回指定用户组可见的线路。
// 未登录（group 为空）按 default 处理；groups 字段不会下发给客户端，
// 避免把其它分组的命名暴露出来。
func RelayEndpointsForGroup(raw, group string) []RelayEndpoint {
	group = strings.TrimSpace(group)
	if group == "" {
		group = "default"
	}

	all := ParseRelayEndpoints(raw)
	out := make([]RelayEndpoint, 0, len(all))
	for _, item := range all {
		if !item.Enabled {
			continue
		}
		if len(item.Groups) > 0 && !containsString(item.Groups, group) {
			continue
		}
		item.Groups = nil
		item = withHealth(item)
		// 探测失败的具体原因（如内网地址、证书错误）只给管理员看，避免泄露内部信息
		if item.Health != nil {
			item.Health.Error = ""
		}
		out = append(out, item)
	}
	return out
}

// RelayEndpointsForAdmin 返回全量线路（含分组与健康状态），仅供管理端使用。
func RelayEndpointsForAdmin(raw string) []RelayEndpoint {
	all := ParseRelayEndpoints(raw)
	out := make([]RelayEndpoint, 0, len(all))
	for _, item := range all {
		out = append(out, withHealth(item))
	}
	return out
}

func normalizeEndpointURL(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	s = strings.TrimRight(s, "/")
	parsed, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host == "" {
		return "", false
	}
	return s, true
}

func normalizeGroups(groups []string) []string {
	out := make([]string, 0, len(groups))
	seen := make(map[string]bool, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" || seen[group] {
			continue
		}
		seen[group] = true
		out = append(out, group)
	}
	sort.Strings(out)
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
