package controller

import (
	"encoding/json"
	"net/http"
	"one-api/common/config"
	"one-api/common/utils"
	"one-api/model"
	"one-api/safty"
	"strings"

	"github.com/gin-gonic/gin"
)

// 以 Key / Password 结尾的敏感配置，后缀黑名单原本漏掉了
var sensitiveOptionKeys = map[string]bool{
	"TurnstileSecretKey": true,
	"CFWorkerImageKey":   true,
}

func isSensitiveOptionKey(k string) bool {
	if strings.HasSuffix(k, "Token") || strings.HasSuffix(k, "Secret") ||
		strings.HasSuffix(k, "SecretKey") || strings.HasSuffix(k, "Password") {
		return true
	}
	return sensitiveOptionKeys[k]
}

func GetOptions(c *gin.Context) {
	var options []*model.Option
	for k, v := range config.GlobalOption.GetAll() {
		if isSensitiveOptionKey(k) {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: utils.Interface2String(v),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
	return
}

// GetRuntimeOptions 返回后台「安全与透传」面板需要的配置描述与当前生效值。
// source = database 表示该值来自后台设置（优先），config 表示来自 config.yaml / 代码默认值。
func GetRuntimeOptions(c *gin.Context) {
	specs := config.RuntimeOptions()
	sources := make(map[string]bool)
	if options, err := model.AllOption(); err == nil {
		for _, option := range options {
			sources[option.Key] = true
		}
	}
	for i := range specs {
		specs[i].Source = "config"
		if sources[specs[i].Key] {
			specs[i].Source = "database"
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    specs,
	})
}

// DeleteRuntimeOption 删除后台覆盖值，恢复 config.yaml / 代码默认值。
func DeleteRuntimeOption(c *gin.Context) {
	key := c.Param("key")
	if !config.IsRuntimeOption(key) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不是可恢复的运行时配置项",
		})
		return
	}
	if err := model.DeleteOption(key); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := config.ResetRuntimeOption(key); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func GetSafeTools(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    safty.GetAllSafeToolsName(),
	})
	return
}

func UpdateOption(c *gin.Context) {
	var option model.Option
	err := json.NewDecoder(c.Request.Body).Decode(&option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	// 运行时配置先校验再落库：model.UpdateOption 会先写数据库再更新内存，
	// 若不预校验，非法值会留在 options 表并在下次启动时覆盖默认配置。
	if config.IsRuntimeOption(option.Key) {
		if err := config.ValidateRuntimeOption(option.Key, option.Value); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "配置值无效：" + err.Error(),
			})
			return
		}
	}

	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && config.GitHubClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 GitHub OAuth，请先填入 GitHub Client Id 以及 GitHub Client Secret！",
			})
			return
		}
	case "OIDCAuthEnabled":
		if option.Value == "true" && (config.OIDCClientId == "" || config.OIDCClientSecret == "" || config.OIDCIssuer == "" || config.OIDCScopes == "" || config.OIDCUsernameClaims == "") {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 OIDC，请先填入OIDC信息！",
			})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(config.EmailDomainWhitelist) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用邮箱域名限制，请先填入限制的邮箱域名！",
			})
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && config.WeChatServerAddress == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用微信登录，请先填入微信登录相关配置信息！",
			})
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && config.TurnstileSiteKey == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！",
			})
			return
		}
	}
	err = model.UpdateOption(option.Key, option.Value)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}
