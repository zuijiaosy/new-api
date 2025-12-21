package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/service/quota_reset"
	"github.com/gin-gonic/gin"
)

// GetQuotaResetLogs 获取额度重置执行日志
func GetQuotaResetLogs(c *gin.Context) {
	// 获取分页参数
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	logs := quota_reset.GetQuotaResetLogs(limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

// TriggerQuotaReset 手动触发额度重置
func TriggerQuotaReset(c *gin.Context) {
	// 检查数据库连接
	if !quota_reset.IsCodexzhDBConnected() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "codexzh 数据库未连接，无法执行额度重置",
		})
		return
	}

	// 尝试启动任务（原子操作，避免竞态条件）
	if !quota_reset.TryStartQuotaReset() {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "额度重置任务正在执行中，请稍后再试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "额度重置任务已触发，请稍后查看执行日志",
	})
}

// GetQuotaResetStatus 获取额度重置状态
func GetQuotaResetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"is_running":           quota_reset.IsQuotaResetRunning(),
			"db_connected":         quota_reset.IsCodexzhDBConnected(),
			"enabled":              quota_reset.IsQuotaResetEnabled(),
			"weekly_limit_enabled": quota_reset.IsWeeklyQuotaLimitEnabled(),
			"reset_time":           quota_reset.GetQuotaResetTime(),
			"concurrency":          quota_reset.GetQuotaResetConcurrency(),
		},
	})
}
