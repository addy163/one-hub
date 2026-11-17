package controller

import (
	"net/http"
	"one-api/common/config"
	"one-api/common/endpointprobe"

	"github.com/gin-gonic/gin"
)

// GetRelayEndpoints 管理端读取全量线路（含分组与健康状态）。
func GetRelayEndpoints(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"endpoints":       config.RelayEndpointsForAdmin(config.RelayEndpoints),
			"probe_interval":  config.RelayEndpointProbeInterval,
			"probe_timeout":   config.RelayEndpointProbeTimeout,
			"probe_available": true,
		},
	})
}

// ProbeRelayEndpoints 立即触发一轮探测，返回最新结果。
func ProbeRelayEndpoints(c *gin.Context) {
	endpoints := endpointprobe.ProbeAll(config.RelayEndpoints)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"endpoints": endpoints,
		},
	})
}
