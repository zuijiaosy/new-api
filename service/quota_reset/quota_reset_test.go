package quota_reset

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// getWeeklyUsedQuota 测试
// ──────────────────────────────────────────────

func TestGetWeeklyUsedQuota_NoTopUp_EqualsTotalConsume(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0) // 周四
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 100}
	seedNewAPIUser(t, 1, user.Email)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 40) // 周二
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 12, 0, 0), 30) // 周三

	assert.Equal(t, int64(70), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_ExcludeTopUpPurchasedWindow(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 100}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 20, 0, 0), 15)
	// 12:00 买了加油包 30 creditTokens，窗口 [12:00, Apr 17 00:00]，消耗 25+15=40，排除 min(40,30)=30
	seedTopUpOrder(t, 1, 1, "加油包", "topup", 30, bjTime(2026, 4, 16, 12, 0, 0))

	// 总消耗 60，排除 30 → weeklyUsed = 30
	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_OldTopUpOrder_FallbackByName(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 100}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	// 旧订单：orderType=NULL，name='加油包'，同样应排除
	seedTopUpOrder(t, 1, 1, "加油包", "", 25, bjTime(2026, 4, 16, 12, 0, 0))

	// 窗口内消耗 25，排除 min(25,25)=25 → weeklyUsed = 20
	assert.Equal(t, int64(20), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_MultipleTopUpsSameDay(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 30, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 11, 0, 0), 10)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 16, 0, 0), 15)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 19, 0, 0), 20)
	// 加油包1: 12:00，25 tokens；窗口消耗 20+15+20=55，排除 min(55,25)=25
	seedTopUpOrder(t, 1, 1, "加油包", "topup", 25, bjTime(2026, 4, 16, 12, 0, 0))
	// 加油包2: 15:00，10 tokens；窗口消耗 15+20=35，排除 min(35,10)=10
	seedTopUpOrder(t, 2, 1, "加油包", "topup", 10, bjTime(2026, 4, 16, 15, 0, 0))

	// 总消耗 65，排除 25+10=35 → weeklyUsed = 30
	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_TopUpsOnDifferentDays(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 17, 23, 0, 0) // 周五
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 17, 9, 0, 0), 30)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 17, 16, 0, 0), 15)
	// Apr 16 12:00 买加油包，窗口 [12:00, Apr 17 00:00]，消耗 25，排除 min(25,30)=25
	seedTopUpOrder(t, 1, 1, "加油包", "topup", 30, bjTime(2026, 4, 16, 12, 0, 0))
	// Apr 17 15:00 买加油包，窗口 [15:00, Apr 18 00:00]，消耗 15，排除 min(15,20)=15
	seedTopUpOrder(t, 2, 1, "大加油包", "topup", 20, bjTime(2026, 4, 17, 15, 0, 0))

	// 总消耗 90，排除 25+15=40 → weeklyUsed = 50
	assert.Equal(t, int64(50), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_SubscriptionOrderLimitsScope(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 12, 0, 0), 50) // 周二（续购前）
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 20) // 周三 续购后
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 18, 0, 0), 40) // 周三

	// 周三 09:00 续购，considerStart = Apr 15 09:00
	seedSubscriptionOrder(t, 10, 1, bjTime(2026, 4, 15, 9, 0, 0))
	// 周三 12:00 买加油包，窗口 [12:00, Apr 16 00:00]，消耗 40，排除 min(40,30)=30
	seedTopUpOrder(t, 11, 1, "加油包", "topup", 30, bjTime(2026, 4, 15, 12, 0, 0))

	// 统计从 Apr 15 09:00 开始：消耗 20+40=60，排除 30 → weeklyUsed = 30
	// Apr 14 的 50 不计入
	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_ActivationCodeLimitsScope(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 12, 0, 0), 50) // 激活码兑换前
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 30) // 激活后

	// 激活码于周三 08:00 使用，considerStart = Apr 15 08:00
	seedActivationCode(t, 1, 1, bjTime(2026, 4, 15, 8, 0, 0))

	// Apr 14 的 50 不计入；Apr 15 的 30 计入 → weeklyUsed = 30
	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_UpgradeOrderDoesNotResetScope(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 12, 0, 0), 50) // 周二（本周一之后）
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 30) // 周三

	// upgrade 类型订单不触发 considerStart 调整，考察窗口从本周一开始
	seedOrderWithType(t, 1, 1, "upgrade", bjTime(2026, 4, 15, 9, 0, 0))

	// considerStart = 本周一（Apr 13 00:00），两笔消耗均在本周内 → weeklyUsed = 80
	assert.Equal(t, int64(80), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_SubscriptionPriorityOverActivationCode(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0) // 周四
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	// 周三 08:00 使用激活码（早于订阅续购）
	seedActivationCode(t, 1, 1, bjTime(2026, 4, 15, 8, 0, 0))
	// 周三 12:00 续购订阅（晚于激活码，但优先级更高）
	seedSubscriptionOrder(t, 10, 1, bjTime(2026, 4, 15, 12, 0, 0))

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 9, 0, 0), 20)  // 激活码后、续购前，不计入
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 18, 0, 0), 30) // 续购后，计入

	// considerStart = Apr 15 12:00（订阅优先，不是激活码的 08:00）
	// weeklyUsed = 30
	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_MultipleActivationCodes_TakesEarliest(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0) // 周四
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 200}
	seedNewAPIUser(t, 1, user.Email)

	// 两条激活码，插入顺序：先 14:00，再 09:00（更早）
	seedActivationCode(t, 1, 1, bjTime(2026, 4, 15, 14, 0, 0))
	seedActivationCode(t, 2, 1, bjTime(2026, 4, 15, 9, 0, 0))

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 8, 0, 0), 50)  // 最早激活码前，不计入
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 30) // 09:00 之后，计入

	// considerStart = Apr 15 09:00（取最早的激活码 usedAt）
	// weeklyUsed = 30
	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

// ──────────────────────────────────────────────
// calculateTodayQuota 测试
// ──────────────────────────────────────────────

func TestCalculateTodayQuota_WeeklyLimitDisabled_ReturnsDailyQuota(t *testing.T) {
	user := &CodexzhUser{Email: "user@example.com", DailyQuota: 88}
	assert.Equal(t, int64(88), calculateTodayQuota(user, false))
}

func TestCalculateTodayQuota_WeeklyLimitEnabled_ConstrainsToWeeklyRemain(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	restoreMul := stubWeeklyMultiplier(1) // weeklyQuota = dailyQuota × 1 = 100
	defer restoreMul()

	user := &CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 100}
	seedNewAPIUser(t, 1, user.Email)

	// 本周消耗：30（Apr 14）+ 30（Apr 15）+ 60（Apr 16）= 120，减去排除 30 → weeklyUsed = 90
	// weeklyQuota = 100, weeklyRemain = 10 → todayQuota = min(100, 10) = 10
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 30)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 30)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 20, 0, 0), 15)
	// 加油包 12:00，creditTokens=30，窗口内消耗 25+15=40，排除 min(40,30)=30
	seedTopUpOrder(t, 1, 1, "加油包", "topup", 30, bjTime(2026, 4, 16, 12, 0, 0))

	assert.Equal(t, int64(10), calculateTodayQuota(user, true))
}

func TestCalculateTodayQuota_WeeklyExhausted_ReturnsZero(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	restoreMul := stubWeeklyMultiplier(1) // weeklyQuota = 50
	defer restoreMul()

	user := &CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 50}
	seedNewAPIUser(t, 1, user.Email)

	// 消耗 60 > weeklyQuota 50 → weeklyRemain ≤ 0 → todayQuota = 0
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 30)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 30)

	assert.Equal(t, int64(0), calculateTodayQuota(user, true))
}

func TestShouldRunQuotaReset_WhenEitherDailyOrWeeklyEnabled(t *testing.T) {
	assert.False(t, shouldRunQuotaReset(false, false))
	assert.True(t, shouldRunQuotaReset(true, false))
	assert.True(t, shouldRunQuotaReset(false, true))
	assert.True(t, shouldRunQuotaReset(true, true))
}

func TestCalculateQuotaForReset_WeeklyOnlyReturnsWeeklyRemain(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	restoreMul := stubWeeklyMultiplier(3) // weeklyQuota = 60 × 3 = 180
	defer restoreMul()

	user := &CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 60}
	seedNewAPIUser(t, 1, user.Email)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 50)

	assert.Equal(t, int64(130), calculateQuotaForReset(user, false, true))
}

func TestCalculateQuotaForReset_WeeklyOnlyExhaustedReturnsZero(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	restoreMul := stubWeeklyMultiplier(2) // weeklyQuota = 120
	defer restoreMul()

	user := &CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 60}
	seedNewAPIUser(t, 1, user.Email)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 80)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 50)

	assert.Equal(t, int64(0), calculateQuotaForReset(user, false, true))
}

func TestGetWeeklyUsedQuota_ManualWeeklyResetBaselineExcludesEarlierLogs(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	restoreWeeklyResetAt := stubWeeklyResetAt(bjTime(2026, 4, 16, 12, 0, 0).Unix())
	defer restoreWeeklyResetAt()

	user := CodexzhUser{Id: 1, Email: "user@example.com", DailyQuota: 100}
	seedNewAPIUser(t, 1, user.Email)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 70)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 30)

	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestExecuteManualWeeklyQuotaReset_UpdatesBaselineAndTokenQuotaImmediately(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	restoreMul := stubWeeklyMultiplier(3) // weeklyQuota = 180
	defer restoreMul()
	common.OptionMap["WeeklyQuotaLimitEnabled"] = "true"

	user := seedActiveCodexzhUser(t, 1, "user@example.com", 60, bjTime(2026, 4, 1, 0, 0, 0), bjTime(2026, 5, 1, 0, 0, 0))
	seedToken(t, 1, user.Email, 5)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 100)

	logEntry := ExecuteManualWeeklyQuotaReset()
	require.NotNil(t, logEntry)
	assert.Equal(t, 1, logEntry.SuccessCount)
	assert.Equal(t, "manual_weekly_reset", logEntry.ResetType)
	assert.Equal(t, now.Unix(), GetWeeklyQuotaResetAt())

	var token model.Token
	require.NoError(t, model.DB.Where("name = ?", user.Email).First(&token).Error)
	assert.Equal(t, 180, token.RemainQuota)
}

func TestExecuteManualWeeklyQuotaReset_SkipsWhenWeeklyDisabled(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restoreNow := stubQuotaResetNow(now)
	defer restoreNow()
	// 明确保持 WeeklyQuotaLimitEnabled=false

	logEntry := ExecuteManualWeeklyQuotaReset()
	require.NotNil(t, logEntry)
	assert.Equal(t, "manual_weekly_reset", logEntry.ResetType)
	assert.Equal(t, 1, logEntry.FailedCount)
	assert.Equal(t, 0, logEntry.SuccessCount)
	assert.Equal(t, int64(0), GetWeeklyQuotaResetAt(), "周开关关闭时不应更新 WeeklyQuotaResetAt")

	// 失败日志需被 addLog 持久化到内存
	logs := GetQuotaResetLogs(10)
	require.NotEmpty(t, logs)
	assert.Equal(t, "manual_weekly_reset", logs[0].ResetType)
	assert.Equal(t, 1, logs[0].FailedCount)
}

// ──────────────────────────────────────────────
// 测试基础设施
// ──────────────────────────────────────────────

func setupQuotaResetTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldCodexzhDB := CodexzhDB
	oldUsingSQLite := common.UsingSQLite
	oldOptionMap := common.OptionMap
	oldRedisEnabled := common.RedisEnabled

	model.DB = db
	model.LOG_DB = db
	CodexzhDB = db
	common.UsingSQLite = true
	common.LogConsumeEnabled = true
	common.RedisEnabled = false
	common.OptionMap = map[string]string{
		"QuotaResetConcurrency":   "3",
		"WeeklyQuotaMultiplier":   "3",
		"WeeklyQuotaResetAt":      "0",
		"QuotaResetEnabled":       "false",
		"WeeklyQuotaLimitEnabled": "false",
	}

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		CodexzhDB = oldCodexzhDB
		common.UsingSQLite = oldUsingSQLite
		common.OptionMap = oldOptionMap
		common.RedisEnabled = oldRedisEnabled
	})

	require.NoError(t, db.AutoMigrate(
		&model.Option{},
		&model.User{},
		&model.Token{},
		&model.Log{},
		&CodexzhPaymentOrder{},
		&CodexzhActivationCode{},
	))
	require.NoError(t, db.Exec(`ALTER TABLE users ADD COLUMN apiKey text`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE users ADD COLUMN subscriptionStart datetime`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE users ADD COLUMN subscriptionEnd datetime`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE users ADD COLUMN dailyQuota integer`).Error)
}

func seedNewAPIUser(t *testing.T, id int, email string) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.User{
		Id:       id,
		Username: fmt.Sprintf("user_%d", id),
		Email:    email,
		Status:   common.UserStatusEnabled,
		Password: "test-password",
	}).Error)
}

func seedActiveCodexzhUser(t *testing.T, id int64, email string, dailyQuota int64, start time.Time, end time.Time) CodexzhUser {
	t.Helper()
	apiKey := "active-api-key"
	require.NoError(t, model.DB.Create(&model.User{
		Id:       int(id),
		Username: fmt.Sprintf("user_%d", id),
		Email:    email,
		Status:   common.UserStatusEnabled,
		Password: "test-password",
	}).Error)
	require.NoError(t, model.DB.Table("users").Where("id = ?", id).Updates(map[string]interface{}{
		"apiKey":            apiKey,
		"subscriptionStart": start,
		"subscriptionEnd":   end,
		"dailyQuota":        dailyQuota,
	}).Error)
	user := CodexzhUser{
		Id:                id,
		Email:             email,
		ApiKey:            &apiKey,
		SubscriptionStart: &start,
		SubscriptionEnd:   &end,
		DailyQuota:        dailyQuota,
	}
	return user
}

func seedToken(t *testing.T, id int, name string, remainQuota int) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Token{
		Id:                 id,
		UserId:             id,
		Key:                fmt.Sprintf("sk-test-%d", id),
		Status:             common.TokenStatusEnabled,
		Name:               name,
		RemainQuota:        remainQuota,
		UnlimitedQuota:     false,
		CreatedTime:        time.Now().Unix(),
		AccessedTime:       time.Now().Unix(),
		ExpiredTime:        -1,
		ModelLimits:        "",
		ModelLimitsEnabled: false,
	}).Error)
}

func seedConsumeLog(t *testing.T, tokenName string, createdAt time.Time, quota int) {
	t.Helper()
	require.NoError(t, model.LOG_DB.Create(&model.Log{
		UserId:    1,
		CreatedAt: createdAt.Unix(),
		Type:      model.LogTypeConsume,
		TokenName: tokenName,
		Quota:     quota,
	}).Error)
}

// seedTopUpOrder 插入一笔加油包支付订单
// orderTypeStr 为空时模拟旧订单（orderType IS NULL）
func seedTopUpOrder(t *testing.T, id int64, userId int64, name string, orderTypeStr string, creditTokens int64, paidAt time.Time) {
	t.Helper()
	paramJSON := fmt.Sprintf(`{"creditTokens":%d}`, creditTokens)
	order := CodexzhPaymentOrder{
		Id:     id,
		UserId: userId,
		Name:   name,
		Status: "PAID",
		Param:  &paramJSON,
		PaidAt: &paidAt,
	}
	if orderTypeStr != "" {
		order.OrderType = &orderTypeStr
	}
	require.NoError(t, CodexzhDB.Create(&order).Error)
}

// seedSubscriptionOrder 插入一笔续购套餐订单（orderType='subscription'）
func seedSubscriptionOrder(t *testing.T, id int64, userId int64, paidAt time.Time) {
	t.Helper()
	orderType := "subscription"
	require.NoError(t, CodexzhDB.Create(&CodexzhPaymentOrder{
		Id:        id,
		UserId:    userId,
		Name:      "标准月套餐",
		Status:    "PAID",
		OrderType: &orderType,
		PaidAt:    &paidAt,
	}).Error)
}

// seedOrderWithType 插入一笔指定 orderType 的套餐订单（用于测试不触发排除的类型）
func seedOrderWithType(t *testing.T, id int64, userId int64, orderType string, paidAt time.Time) {
	t.Helper()
	require.NoError(t, CodexzhDB.Create(&CodexzhPaymentOrder{
		Id:        id,
		UserId:    userId,
		Name:      "套餐升级",
		Status:    "PAID",
		OrderType: &orderType,
		PaidAt:    &paidAt,
	}).Error)
}

// seedActivationCode 插入一条已使用的激活码记录
func seedActivationCode(t *testing.T, id int64, userId int64, usedAt time.Time) {
	t.Helper()
	uid := userId
	require.NoError(t, CodexzhDB.Create(&CodexzhActivationCode{
		Id:     id,
		UserId: &uid,
		Status: "used",
		UsedAt: &usedAt,
	}).Error)
}

func stubQuotaResetNow(now time.Time) func() {
	old := quotaResetNow
	quotaResetNow = func() time.Time { return now }
	return func() { quotaResetNow = old }
}

func stubWeeklyMultiplier(m int) func() {
	old := weeklyQuotaMultiplierFn
	weeklyQuotaMultiplierFn = func() int { return m }
	return func() { weeklyQuotaMultiplierFn = old }
}

func stubWeeklyResetAt(ts int64) func() {
	old := weeklyQuotaResetAtFn
	weeklyQuotaResetAtFn = func() int64 { return ts }
	return func() { weeklyQuotaResetAtFn = old }
}

func bjTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, beijingLocation)
}
