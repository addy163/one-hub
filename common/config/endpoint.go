package config

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

// RelayEndpoint 是一条可以下发给用户的 API 线路。
// Groups 为空表示所有分组可见；Enabled 为 false 的线路不下发。
type RelayEndpoint struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Url     string   `json:"url"`
	Groups  []string `json:"groups,omitempty"`
	Note    string   `json:"note,omitempty"`
	Enabled bool     `json:"enabled"`
	Sort    int      `json:"sort,omitempty"`
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
		out = append(out, item)
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
