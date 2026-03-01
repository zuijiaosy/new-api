package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetUsageCardStats 统一卡片统计接口
// 参数：token_name(必填), today_start, today_end, week_start, week_end
// 用 2 条 SQL 替代原来 5+ 条，消除全表 COUNT
func GetUsageCardStats(c *gin.Context) {
	tokenName := c.Query("token_name")
	if tokenName == "" {
		common.ApiErrorMsg(c, "token_name is required")
		return
	}

	todayStart, err := strconv.ParseInt(c.Query("today_start"), 10, 64)
	if err != nil || todayStart <= 0 {
		common.ApiErrorMsg(c, "invalid today_start")
		return
	}

	todayEnd, err := strconv.ParseInt(c.Query("today_end"), 10, 64)
	if err != nil || todayEnd <= 0 {
		common.ApiErrorMsg(c, "invalid today_end")
		return
	}

	weekStart, err := strconv.ParseInt(c.Query("week_start"), 10, 64)
	if err != nil || weekStart <= 0 {
		common.ApiErrorMsg(c, "invalid week_start")
		return
	}

	weekEnd, err := strconv.ParseInt(c.Query("week_end"), 10, 64)
	if err != nil || weekEnd <= 0 {
		common.ApiErrorMsg(c, "invalid week_end")
		return
	}

	// 校验：时间范围不超过 31 天
	const maxRangeSeconds int64 = 31 * 24 * 60 * 60
	if weekEnd-weekStart > maxRangeSeconds {
		common.ApiErrorMsg(c, "time range exceeds 31 days")
		return
	}

	result, err := model.GetUsageCardStats(tokenName, todayStart, todayEnd, weekStart, weekEnd)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, result)
}
