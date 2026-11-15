package controller

import (
	"net/http"
	"one-api/common/utils"
	"one-api/model"
	"one-api/providers"
	providersBase "one-api/providers/base"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetModelList(c *gin.Context) {
	channel := &model.Channel{}
	err := c.ShouldBindJSON(channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	keys := strings.Split(channel.Key, "\n")
	channel.Key = keys[0]

	if channel.Key == "" {
		if channel.Id == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "key is required",
			})
			return
		}

		channel, err = model.GetChannelById(channel.Id)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	// 该接口的渠道对象直接来自请求体，管理员可在单次请求里填任意 base_url
	// 让服务端发起请求并回显结果。这里至少拦掉回环与云元数据地址；
	// blockPrivate=false 保留 192.168/10.x 内网自建渠道可用。
	if baseURL := channel.GetBaseURL(); baseURL != "" {
		if err := utils.ValidatePublicURL(baseURL, false); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "渠道地址不被允许：" + err.Error(),
			})
			return
		}
	}

	provider := providers.GetProvider(channel, c)
	if provider == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "provider not found",
		})
		return
	}

	modelProvider, ok := provider.(providersBase.ModelListInterface)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "channel not implemented",
		})
		return
	}

	modelList, err := modelProvider.GetModelList()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 去除重复的模型名称
	uniqueModels := removeDuplicates(modelList)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    uniqueModels,
	})
}

// 辅助函数：去除切片中的重复元素
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
