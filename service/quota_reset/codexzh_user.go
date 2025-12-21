package quota_reset

import (
	"time"
)

// CodexzhUser codexzh 系统的用户模型
// 对应 codexzh 数据库的 users 表
type CodexzhUser struct {
	Id                int64      `gorm:"column:id;primaryKey"`
	Email             string     `gorm:"column:email"`
	ApiKey            *string    `gorm:"column:apiKey"`
	SubscriptionStart *time.Time `gorm:"column:subscriptionStart"`
	SubscriptionEnd   *time.Time `gorm:"column:subscriptionEnd"`
	DailyQuota        int64      `gorm:"column:dailyQuota;default:45000000"`
	WeeklyQuota       *int64     `gorm:"column:weeklyQuota"`
}

// TableName 返回实际的表名
func (CodexzhUser) TableName() string {
	return "users" // 小写复数
}

// GetActiveUsers 获取所有活跃用户
// 条件：apiKey 不为空 且 subscriptionEnd > 当前时间
func GetActiveUsers() ([]CodexzhUser, error) {
	var users []CodexzhUser
	now := time.Now()

	err := CodexzhDB.Where(`"apiKey" IS NOT NULL AND "subscriptionEnd" > ?`, now).
		Find(&users).Error

	return users, err
}

// IsDayPass 判断是否是天卡用户（订阅时长 <= 48小时）
// 天卡用户不参与每日额度重置
func (u *CodexzhUser) IsDayPass() bool {
	if u.SubscriptionStart == nil || u.SubscriptionEnd == nil {
		return false
	}

	startTime := u.SubscriptionStart.Unix()
	endTime := u.SubscriptionEnd.Unix()

	// 异常数据检查
	if endTime < startTime {
		return false
	}

	// 计算订阅时长（小时）
	durationHours := float64(endTime-startTime) / 3600.0

	// 业务规则：48小时以内视为天卡，跳过重置
	return durationHours <= 48
}

// HasWeeklyQuotaLimit 检查用户是否有周额度限制
// 如果 weeklyQuota 为 NULL 或 0，则表示不限制
func (u *CodexzhUser) HasWeeklyQuotaLimit() bool {
	return u.WeeklyQuota != nil && *u.WeeklyQuota > 0
}

// MaskEmail 返回脱敏后的邮箱地址
// 例如：user@example.com -> us***@example.com
func (u *CodexzhUser) MaskEmail() string {
	email := u.Email
	if len(email) < 3 {
		return "***"
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex == -1 || atIndex < 2 {
		return email[:2] + "***"
	}
	return email[:2] + "***" + email[atIndex:]
}
