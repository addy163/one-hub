package ali_test

import (
	"net/http"
	"one-api/common/config"
	"one-api/common/requester"
	"one-api/common/test"
	"one-api/model"
	"os"
	"testing"

	"gorm.io/datatypes"
)

// TestMain 初始化全局 HTTP 客户端，否则用例里发请求会解引用空指针
func TestMain(m *testing.M) {
	requester.InitHttpClient()
	os.Exit(m.Run())
}

func setupAliTestServer() (baseUrl string, server *test.ServerTest, teardown func()) {
	server = test.NewTestServer()
	ts := server.TestServer(func(w http.ResponseWriter, r *http.Request) bool {
		return test.OpenAICheck(w, r)
	})
	ts.Start()
	teardown = ts.Close

	baseUrl = ts.URL
	return
}

// getAliChannel 构造走原生协议的渠道（默认已是兼容协议，这里显式开启原生）
func getAliChannel(baseUrl string) model.Channel {
	channel := test.GetChannel(config.ChannelTypeAli, baseUrl, "", "", "")
	plugin := datatypes.NewJSONType(model.PluginType{
		"use_native_api": {"enable": true},
	})
	channel.Plugin = &plugin
	return channel
}

// getAliCompatChannel 构造勾选了 OpenAI 兼容协议的渠道（默认已改为原生）
func getAliCompatChannel(baseUrl string) model.Channel {
	channel := test.GetChannel(config.ChannelTypeAli, baseUrl, "", "", "")
	plugin := datatypes.NewJSONType(model.PluginType{
		"use_openai_api": {"enable": true},
	})
	channel.Plugin = &plugin
	return channel
}
