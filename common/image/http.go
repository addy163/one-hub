package image

import (
	"encoding/json"
	"errors"
	"net/http"
	"one-api/common/config"
	"one-api/common/utils"
	"time"
)

var ImageHttpClients = &http.Client{
	Transport: &http.Transport{
		DialContext: utils.Socks5ProxyFunc,
		Proxy:       utils.ProxyFunc,
	},
	Timeout: 15 * time.Second,
	// 与 RequestFile 保持同一份运行时配置，后台修改后无需重启。
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return utils.SSRFCheckRedirect(!config.PrivateImageAllowed())(req, via)
	},
}

var maxFileSize int64 = 20 * 1024 * 1024 // 20MB

type CFRequest struct {
	Action string `json:"action"`
	APIKey string `json:"api_key"`
	URL    string `json:"url"`
}

type CFResponse struct {
	Status   bool   `json:"status"`
	Message  string `json:"message,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func RequestFile(url, action string) (*http.Response, error) {
	// image_url 由 API 请求体直接提供，是普通用户级 SSRF 入口。
	// 默认拒绝内网/回环/云元数据地址；确有内网图床需求可设置 ssrf_allow_private_image: true。
	blockPrivate := !config.PrivateImageAllowed()
	if err := utils.ValidatePublicURL(url, blockPrivate); err != nil {
		return nil, err
	}

	reqUrl := url
	method := http.MethodGet
	var requestBody any

	if config.CFWorkerImageUrl != "" {
		requestBody = &CFRequest{
			Action: action,
			APIKey: config.CFWorkerImageKey,
			URL:    url,
		}
		reqUrl = config.CFWorkerImageUrl
		method = http.MethodPost
	}

	res, err := utils.RequestBuilder(utils.SetProxy(config.ChatImageRequestProxy, nil), method, reqUrl, requestBody, nil)

	if err != nil {
		return nil, err
	}

	response, err := ImageHttpClients.Do(res)
	if err != nil {
		return nil, err
	}

	response.Body = http.MaxBytesReader(nil, response.Body, maxFileSize)

	if response.StatusCode != http.StatusOK && config.CFWorkerImageUrl != "" {
		var cfResp *CFResponse
		err = json.NewDecoder(response.Body).Decode(&cfResp)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(cfResp.Message)
	}

	return response, err
}
