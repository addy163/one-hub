package requester

import (
	"net/http"
	"one-api/common/utils"
	"time"
)

var HTTPClient *http.Client

func InitHttpClient() {
	trans := &http.Transport{
		DialContext: utils.Socks5ProxyFunc,
		Proxy:       utils.ProxyFunc,
		// 只限制等待响应头的时间，不限制流式响应体的读取，避免截断 SSE
		ResponseHeaderTimeout: 120 * time.Second,
	}

	HTTPClient = &http.Client{
		Transport: trans,
		// 逐跳复查重定向目标，防止公网 302 跳到本机/云元数据服务
		CheckRedirect: utils.SSRFCheckRedirect(false),
	}

	relayTimeout := utils.GetOrDefault("relay_timeout", 0)
	if relayTimeout > 0 {
		HTTPClient.Timeout = time.Duration(relayTimeout) * time.Second
	}
}
