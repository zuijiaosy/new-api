package quota_reset

import (
	"time"

	"github.com/QuantumNous/new-api/common"
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

// ──────────────────────────────────────────────
// 支付订单 & 激活码（周额度计算专用）
// ──────────────────────────────────────────────

// CodexzhPaymentOrder 支付订单（对应 codexzh PostgreSQL payment_orders 表）
type CodexzhPaymentOrder struct {
	Id        int64      `gorm:"column:id;primaryKey"`
	UserId    int64      `gorm:"column:userId"`
	Name      string     `gorm:"column:name"`
	Status    string     `gorm:"column:status"`
	OrderType *string    `gorm:"column:orderType"`
	PlanId    *int64     `gorm:"column:planId"`
	PaidAt    *time.Time `gorm:"column:paidAt"`
	Param     *string    `gorm:"column:param"`
}

func (CodexzhPaymentOrder) TableName() string { return "payment_orders" }

// CodexzhActivationCode 激活码（对应 codexzh PostgreSQL activation_codes 表）
type CodexzhActivationCode struct {
	Id     int64      `gorm:"column:id;primaryKey"`
	UserId *int64     `gorm:"column:userId"`
	Status string     `gorm:"column:status"`
	UsedAt *time.Time `gorm:"column:usedAt"`
}

func (CodexzhActivationCode) TableName() string { return "activation_codes" }

// GetSubscriptionOrdersThisWeek 查询本周内的套餐续购订单（orderType='subscription'，到期重购）
// start/end 均为北京时间，GORM 会自动转换为数据库期望的格式
func GetSubscriptionOrdersThisWeek(userId int64, start, end time.Time) ([]CodexzhPaymentOrder, error) {
	var orders []CodexzhPaymentOrder
	err := CodexzhDB.
		Where(`"userId" = ? AND "orderType" = 'subscription' AND status = 'PAID' AND "paidAt" >= ? AND "paidAt" <= ?`,
			userId, start, end).
		Order(`"paidAt" ASC`).
		Find(&orders).Error
	return orders, err
}

// GetTopUpOrdersInWindow 查询指定时间窗口内的加油包订单
// 新订单：orderType = 'topup'；旧订单降级：orderType IS NULL AND name LIKE '%加油包%'
func GetTopUpOrdersInWindow(userId int64, start, end time.Time) ([]CodexzhPaymentOrder, error) {
	var orders []CodexzhPaymentOrder
	err := CodexzhDB.
		Where(`"userId" = ? AND status = 'PAID' AND "paidAt" >= ? AND "paidAt" <= ? AND ("orderType" = 'topup' OR ("orderType" IS NULL AND name LIKE ?))`,
			userId, start, end, "%加油包%").
		Order(`"paidAt" ASC`).
		Find(&orders).Error
	return orders, err
}

// GetActivationCodesThisWeek 查询本周内已使用的激活码
func GetActivationCodesThisWeek(userId int64, start, end time.Time) ([]CodexzhActivationCode, error) {
	var codes []CodexzhActivationCode
	err := CodexzhDB.
		Where(`"userId" = ? AND status = 'used' AND "usedAt" >= ? AND "usedAt" <= ?`,
			userId, start, end).
		Order(`"usedAt" ASC`).
		Find(&codes).Error
	return codes, err
}

// ParseParamCreditTokens 从支付订单的 param JSON 中解析加油包额度（creditTokens 字段）
// param 格式示例：{"productType":"topup","creditUsd":50,"creditTokens":25000000}
// 解析失败或字段缺失时返回 0
func ParseParamCreditTokens(param *string) int64 {
	if param == nil || *param == "" {
		return 0
	}
	var data struct {
		CreditTokens int64 `json:"creditTokens"`
	}
	if err := common.UnmarshalJsonStr(*param, &data); err != nil {
		return 0
	}
	return data.CreditTokens
}
