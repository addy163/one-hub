// Package endpointprobe 负责定期探测「线路清单」里每个地址是否可用。
//
// 探测方式是请求 `${地址}/api/status`：只要返回 200 且 success 为 true 即视为在线，
// 同时记录耗时。结果写入 config.SetEndpointHealth，由 /api/status 与后台接口读取。
package endpointprobe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"one-api/common/config"
	"one-api/common/logger"
	"strings"
	"sync"
	"time"
)

const (
	probePath        = "/api/status"
	maxConcurrency   = 8
	maxErrorLength   = 120
	minProbeTimeout  = 500 * time.Millisecond
	minProbeInterval = 10 * time.Second
	initialDelay     = 10 * time.Second
	bodyLimit        = 1 << 20
)

var client = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

// ProbeAll 探测全部启用的线路并写入健康状态，返回带状态的结果。
func ProbeAll(raw string) []config.RelayEndpoint {
	endpoints := config.ParseRelayEndpoints(raw)
	if len(endpoints) == 0 {
		return endpoints
	}

	timeout := probeTimeout()
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i := range endpoints {
		if !endpoints[i].Enabled {
			continue
		}
		wg.Add(1)
		go func(item *config.RelayEndpoint) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			health := ProbeOne(item.Url, timeout)
			config.SetEndpointHealth(item.Url, health)
		}(&endpoints[i])
	}
	wg.Wait()

	for i := range endpoints {
		if !endpoints[i].Enabled {
			continue
		}
		health := config.GetEndpointHealth(endpoints[i].Url)
		endpoints[i].Health = &health
	}
	return endpoints
}

// ProbeOne 探测单个地址。
func ProbeOne(rawUrl string, timeout time.Duration) config.RelayEndpointHealth {
	if timeout < minProbeTimeout {
		timeout = minProbeTimeout
	}
	base := strings.TrimRight(strings.TrimSpace(rawUrl), "/")
	if base == "" {
		return config.RelayEndpointHealth{Status: config.EndpointStatusDown, Error: "empty url", CheckedAt: time.Now().Unix()}
	}

	health := doProbe(base+probePath, timeout)
	// 网络抖动时补一次，避免偶发失败把线路标成离线
	if health.Status != config.EndpointStatusUp {
		retry := doProbe(base+probePath, timeout)
		if retry.Status == config.EndpointStatusUp {
			return retry
		}
		return health
	}
	return health
}

func doProbe(target string, timeout time.Duration) config.RelayEndpointHealth {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return downStatus(err.Error())
	}
	req.Header.Set("User-Agent", "one-hub-endpoint-probe")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return downStatus(err.Error())
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	latency := time.Since(start).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		return config.RelayEndpointHealth{
			Status:    config.EndpointStatusDown,
			LatencyMs: latency,
			CheckedAt: time.Now().Unix(),
			Error:     "http status " + resp.Status,
		}
	}

	// 只校验 JSON 结构，不解析业务字段，避免被反代的 HTML 错误页骗过
	if readErr != nil || len(body) == 0 {
		return config.RelayEndpointHealth{
			Status:    config.EndpointStatusDown,
			LatencyMs: latency,
			CheckedAt: time.Now().Unix(),
			Error:     "empty response body",
		}
	}
	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return config.RelayEndpointHealth{
			Status:    config.EndpointStatusDown,
			LatencyMs: latency,
			CheckedAt: time.Now().Unix(),
			Error:     "invalid json response",
		}
	}
	if !payload.Success {
		return config.RelayEndpointHealth{
			Status:    config.EndpointStatusDown,
			LatencyMs: latency,
			CheckedAt: time.Now().Unix(),
			Error:     "success=false",
		}
	}

	return config.RelayEndpointHealth{
		Status:    config.EndpointStatusUp,
		LatencyMs: latency,
		CheckedAt: time.Now().Unix(),
	}
}

func downStatus(msg string) config.RelayEndpointHealth {
	msg = strings.TrimSpace(msg)
	if len(msg) > maxErrorLength {
		msg = msg[:maxErrorLength]
	}
	return config.RelayEndpointHealth{
		Status:    config.EndpointStatusDown,
		CheckedAt: time.Now().Unix(),
		Error:     msg,
	}
}

func probeTimeout() time.Duration {
	timeout := time.Duration(config.RelayEndpointProbeTimeout) * time.Millisecond
	if timeout < minProbeTimeout {
		timeout = 3 * time.Second
	}
	return timeout
}

// Start 启动后台探测循环。间隔为 0 或负数时等同于关闭，只做空转检查，
// 因此后台把间隔改成 0 后不需要重启即可停止探测。
func Start() {
	go func() {
		// 等服务监听起来再探第一轮，否则自身域名必然失败
		time.Sleep(initialDelay)
		for {
			interval := time.Duration(config.RelayEndpointProbeInterval) * time.Second
			if interval <= 0 {
				time.Sleep(30 * time.Second)
				continue
			}
			if interval < minProbeInterval {
				interval = minProbeInterval
			}

			raw := config.RelayEndpoints
			if strings.TrimSpace(raw) != "" {
				ProbeAll(raw)
			}
			time.Sleep(interval)
		}
	}()
	logger.SysLog("relay endpoint health probe started")
}
