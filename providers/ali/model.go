package ali

import (
	"errors"
	"fmt"
	"net/http"
	"one-api/providers/openai"
)

const (
	// WebSearchSupportedModels 定义支持web_search插件的模型关键词
	WebSearchSupportedModels = "qwen-plus,qwen-turbo,qwen-max,qwq-plus"
)

func (p *AliProvider) GetModelList() ([]string, error) {
	// 模型列表统一走 compatible-mode，走专有网关时也能拉到
	fullRequestURL := fmt.Sprintf("%s%s", normalizeCompatBaseURL(p.GetBaseURL()), p.Config.ModelList)
	headers := p.GetRequestHeaders()

	req, err := p.Requester.NewRequest(http.MethodGet, fullRequestURL, p.Requester.WithHeader(headers))
	if err != nil {
		return nil, errors.New("new_request_failed")
	}

	response := &openai.ModelListResponse{}
	_, errWithCode := p.Requester.SendRequest(req, response, false)
	if errWithCode != nil {
		return nil, errors.New(errWithCode.Message)
	}

	var modelList []string
	for _, model := range response.Data {
		modelList = append(modelList, model.Id)
	}

	return modelList, nil
}
