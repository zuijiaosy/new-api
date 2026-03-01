package model

import (
	"time"
)

// UsageCardResult 卡片统计结果
type UsageCardResult struct {
	TodayCalls int   `json:"today_calls"`
	TodayQuota int64 `json:"today_quota"`
	WeekCalls  int   `json:"week_calls"`
	WeekQuota  int64 `json:"week_quota"`
	Rpm        int   `json:"rpm"`
	Tpm        int64 `json:"tpm"`
}

// cardAggRow 本周范围条件聚合行
type cardAggRow struct {
	TodayCalls int   `gorm:"column:today_calls"`
	TodayQuota int64 `gorm:"column:today_quota"`
	WeekCalls  int   `gorm:"column:week_calls"`
	WeekQuota  int64 `gorm:"column:week_quota"`
}

// cardRpmRow RPM/TPM 聚合行
type cardRpmRow struct {
	Rpm int   `gorm:"column:rpm"`
	Tpm int64 `gorm:"column:tpm"`
}

// GetUsageCardStats 用 2 条 SQL 获取卡片统计
// SQL-1: 本周范围条件聚合（今日⊆本周），利用 idx_token_created_type 索引
// SQL-2: 最近 60 秒 RPM/TPM
func GetUsageCardStats(tokenName string, todayStart, todayEnd, weekStart, weekEnd int64) (*UsageCardResult, error) {
	var agg cardAggRow

	// SQL-1: 在本周范围内用条件聚合同时算出今日和本周的 calls/quota
	err := LOG_DB.Table("logs").
		Select(`
			COUNT(CASE WHEN created_at >= ? AND created_at <= ? THEN 1 END) AS today_calls,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at <= ? THEN quota END), 0) AS today_quota,
			COUNT(*) AS week_calls,
			COALESCE(SUM(quota), 0) AS week_quota
		`, todayStart, todayEnd, todayStart, todayEnd).
		Where("token_name = ? AND type = ? AND created_at >= ? AND created_at <= ?",
			tokenName, LogTypeConsume, weekStart, weekEnd).
		Scan(&agg).Error

	if err != nil {
		return nil, err
	}

	// SQL-2: 最近 60 秒的 RPM/TPM
	var rpm cardRpmRow
	sixtySecondsAgo := time.Now().Unix() - 60
	err = LOG_DB.Table("logs").
		Select("COUNT(*) AS rpm, COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS tpm").
		Where("token_name = ? AND type = ? AND created_at >= ?",
			tokenName, LogTypeConsume, sixtySecondsAgo).
		Scan(&rpm).Error

	if err != nil {
		return nil, err
	}

	return &UsageCardResult{
		TodayCalls: agg.TodayCalls,
		TodayQuota: agg.TodayQuota,
		WeekCalls:  agg.WeekCalls,
		WeekQuota:  agg.WeekQuota,
		Rpm:        rpm.Rpm,
		Tpm:        rpm.Tpm,
	}, nil
}
