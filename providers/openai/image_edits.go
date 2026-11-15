package openai

import (
	"bytes"
	"fmt"
	"net/http"
	"one-api/common"
	"one-api/common/config"
	"one-api/common/requester"
	"one-api/types"
)

func (p *OpenAIProvider) CreateImageEdits(request *types.ImageEditRequest) (*types.ImageResponse, *types.OpenAIErrorWithStatusCode) {
	req, errWithCode := p.getRequestImageBody(config.RelayModeImagesEdits, request.Model, request)
	if errWithCode != nil {
		return nil, errWithCode
	}
	defer req.Body.Close()

	response := &OpenAIProviderImageResponse{}
	// 发送请求
	_, errWithCode = p.Requester.SendRequest(req, response, false)
	if errWithCode != nil {
		return nil, errWithCode
	}

	openaiErr := ErrorHandle(&response.OpenAIErrorResponse)
	if openaiErr != nil {
		errWithCode = &types.OpenAIErrorWithStatusCode{
			OpenAIError: *openaiErr,
			StatusCode:  http.StatusBadRequest,
		}
		return nil, errWithCode
	}

	if response.Usage != nil && response.Usage.TotalTokens > 0 {
		*p.Usage = *response.Usage.ToOpenAIUsage()
	} else {
		p.Usage.TotalTokens = p.Usage.PromptTokens
	}

	return &response.ImageResponse, nil
}

func (p *OpenAIProvider) getRequestImageBody(relayMode int, ModelName string, request *types.ImageEditRequest) (*http.Request, *types.OpenAIErrorWithStatusCode) {
	url, errWithCode := p.GetSupportedAPIUri(relayMode)
	if errWithCode != nil {
		return nil, errWithCode
	}
	// 获取请求地址
	fullRequestURL := p.GetFullRequestURL(url, ModelName)

	// 获取请求头
	headers := p.GetRequestHeaders()
	// 创建请求
	var req *http.Request
	var err error
	if p.OriginalModel != request.Model {
		var formBody bytes.Buffer
		builder := p.Requester.CreateFormBuilder(&formBody)
		if err := imagesEditsMultipartForm(p, request, builder); err != nil {
			return nil, common.ErrorWrapper(err, "create_form_builder_failed", http.StatusInternalServerError)
		}
		req, err = p.Requester.NewRequest(
			http.MethodPost,
			fullRequestURL,
			p.Requester.WithBody(&formBody),
			p.Requester.WithHeader(headers),
			p.Requester.WithContentType(builder.FormDataContentType()))
		req.ContentLength = int64(formBody.Len())
	} else {
		body, exists := p.GetRawBody()
		if !exists {
			return nil, common.StringErrorWrapperLocal("request body not found", "request_body_not_found", http.StatusInternalServerError)
		}
		req, err = p.Requester.NewRequest(
			http.MethodPost,
			fullRequestURL,
			p.Requester.WithBody(body),
			p.Requester.WithHeader(headers),
			p.Requester.WithContentType(p.Context.Request.Header.Get("Content-Type")))
		req.ContentLength = p.Context.Request.ContentLength
	}

	if err != nil {
		return nil, common.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}

	return req, nil
}

func imagesEditsMultipartForm(p *OpenAIProvider, request *types.ImageEditRequest, b requester.FormBuilder) (err error) {
	if request.Image != nil {
		err = b.CreateFormFile("image", request.Image)
		if err != nil {
			return fmt.Errorf("creating form image: %w", err)
		}
	}

	if request.Images != nil {
		for _, image := range request.Images {
			err = b.CreateFormFile("image[]", image)
			if err != nil {
				return fmt.Errorf("creating form images: %w", err)
			}
		}
	}

	err = b.WriteField("prompt", request.Prompt)
	if err != nil {
		return fmt.Errorf("writing prompt: %w", err)
	}

	err = b.WriteField("model", request.Model)
	if err != nil {
		return fmt.Errorf("writing model name: %w", err)
	}

	if request.Mask != nil {
		err = b.CreateFormFile("mask", request.Mask)
		if err != nil {
			return fmt.Errorf("writing mask: %w", err)
		}
	}

	if request.ResponseFormat != "" {
		err = b.WriteField("response_format", request.ResponseFormat)
		if err != nil {
			return fmt.Errorf("writing format: %w", err)
		}
	}

	if request.N != 0 {
		err = b.WriteField("n", fmt.Sprintf("%d", request.N))
		if err != nil {
			return fmt.Errorf("writing n: %w", err)
		}
	}

	if request.Size != "" {
		err = b.WriteField("size", request.Size)
		if err != nil {
			return fmt.Errorf("writing size: %w", err)
		}
	}

	if request.User != "" {
		err = b.WriteField("user", request.User)
		if err != nil {
			return fmt.Errorf("writing user: %w", err)
		}
	}

	// 新增图片模型参数：模型发生映射时必须显式写出，否则会被重建的 multipart 丢掉
	extraFields := []struct {
		name  string
		value string
	}{
		{"quality", request.Quality},
		{"background", request.Background},
		{"moderation", request.Moderation},
		{"output_format", request.OutputFormat},
		{"output_compression", request.OutputCompression},
		{"input_fidelity", request.InputFidelity},
		{"seed", request.Seed},
		{"watermark", request.Watermark},
	}
	for _, f := range extraFields {
		if f.value == "" {
			continue
		}
		if err = b.WriteField(f.name, f.value); err != nil {
			return fmt.Errorf("writing %s: %w", f.name, err)
		}
	}

	// 兜底：把客户端原始 multipart 中程序未定义的所有非文件字段原样带过去，
	// 对标 new-api adaptor 遍历 mf.Value 的做法，避免上游新增参数被吞。
	if p.Context != nil && p.Context.Request != nil && p.Context.Request.MultipartForm != nil {
		written := make(map[string]bool)
		for _, name := range []string{
			"image", "image[]", "mask", "prompt", "model", "response_format",
			"n", "size", "user", "quality", "background", "moderation",
			"output_format", "output_compression", "input_fidelity", "seed", "watermark",
		} {
			written[name] = true
		}
		for name, values := range p.Context.Request.MultipartForm.Value {
			if written[name] {
				continue
			}
			for _, value := range values {
				if err = b.WriteField(name, value); err != nil {
					return fmt.Errorf("writing extra field %s: %w", name, err)
				}
			}
		}
	}

	return b.Close()
}
