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

func TestGetWeeklyUsedQuota_NoTopUp_EqualsTotalConsume(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{
		Email:      "user@example.com",
		DailyQuota: 100,
	}
	seedQuotaResetUser(t, 1, user.Email)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 10, 0, 0), 40)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 12, 0, 0), 30)

	assert.Equal(t, int64(70), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_ExcludeTopUpPurchasedWindow(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{
		Email:      "user@example.com",
		DailyQuota: 100,
	}
	seedQuotaResetUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 20, 0, 0), 15)
	seedTopUp(t, 1, 1, 0, 30, "creem", bjTime(2026, 4, 16, 12, 0, 0))

	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_MultipleTopUpsSameDay_AccumulateInOrder(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 30, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{
		Email:      "user@example.com",
		DailyQuota: 200,
	}
	seedQuotaResetUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 11, 0, 0), 10)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 16, 0, 0), 15)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 19, 0, 0), 20)
	seedTopUp(t, 1, 1, 0, 25, "creem", bjTime(2026, 4, 16, 12, 0, 0))
	seedTopUp(t, 2, 1, 0, 10, "creem", bjTime(2026, 4, 16, 15, 0, 0))

	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_MultipleTopUpsSameDay_DoesNotDoubleDeductOverlappingConsume(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 30, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{
		Email:      "user@example.com",
		DailyQuota: 200,
	}
	seedQuotaResetUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 15)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 16, 0, 0), 20)
	seedTopUp(t, 1, 1, 0, 30, "creem", bjTime(2026, 4, 16, 12, 0, 0))
	seedTopUp(t, 2, 1, 0, 30, "creem", bjTime(2026, 4, 16, 15, 0, 0))

	assert.Equal(t, int64(15), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_SubscriptionStartLimitsScope(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	subscriptionStart := bjTime(2026, 4, 15, 9, 0, 0)
	user := CodexzhUser{
		Email:             "user@example.com",
		DailyQuota:        200,
		SubscriptionStart: &subscriptionStart,
	}
	seedQuotaResetUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 14, 12, 0, 0), 50)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 15, 18, 0, 0), 40)
	seedTopUp(t, 1, 1, 0, 30, "creem", bjTime(2026, 4, 15, 12, 0, 0))

	assert.Equal(t, int64(30), getWeeklyUsedQuota(&user))
}

func TestGetWeeklyUsedQuota_TopUpsOnDifferentDaysOnlyExcludeSameDayConsume(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 17, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	user := CodexzhUser{
		Email:      "user@example.com",
		DailyQuota: 200,
	}
	seedQuotaResetUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 17, 9, 0, 0), 30)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 17, 16, 0, 0), 15)
	seedTopUp(t, 1, 1, 0, 30, "creem", bjTime(2026, 4, 16, 12, 0, 0))
	seedTopUp(t, 2, 1, 0, 20, "creem", bjTime(2026, 4, 17, 15, 0, 0))

	assert.Equal(t, int64(50), getWeeklyUsedQuota(&user))
}

func TestCalculateTodayQuota_WeeklyLimitDisabled_ReturnsDailyQuota(t *testing.T) {
	user := &CodexzhUser{
		Email:      "user@example.com",
		DailyQuota: 88,
	}

	assert.Equal(t, int64(88), calculateTodayQuota(user, false))
}

func TestCalculateTodayQuota_UsesTopUpExcludedWeeklyUsage(t *testing.T) {
	setupQuotaResetTestDB(t)

	now := bjTime(2026, 4, 16, 23, 0, 0)
	restore := stubQuotaResetNow(now)
	defer restore()

	weeklyQuota := int64(50)
	user := &CodexzhUser{
		Email:       "user@example.com",
		DailyQuota:  100,
		WeeklyQuota: &weeklyQuota,
	}
	seedQuotaResetUser(t, 1, user.Email)

	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 10, 0, 0), 20)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 13, 0, 0), 25)
	seedConsumeLog(t, user.Email, bjTime(2026, 4, 16, 20, 0, 0), 15)
	seedTopUp(t, 1, 1, 0, 30, "creem", bjTime(2026, 4, 16, 12, 0, 0))

	assert.Equal(t, int64(20), calculateTodayQuota(user, true))
}

func setupQuotaResetTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldUsingSQLite := common.UsingSQLite

	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true
	common.LogConsumeEnabled = true

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
	})

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.Log{}))
}

func seedQuotaResetUser(t *testing.T, id int, email string) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.User{
		Id:       id,
		Username: fmt.Sprintf("user_%d", id),
		Email:    email,
		Status:   common.UserStatusEnabled,
		Password: "test-password",
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

func seedTopUp(t *testing.T, id int, userId int, money float64, amount int64, paymentMethod string, completeAt time.Time) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.TopUp{
		Id:            id,
		UserId:        userId,
		Money:         money,
		Amount:        amount,
		PaymentMethod: paymentMethod,
		TradeNo:       fmt.Sprintf("%s-%d-%d", paymentMethod, completeAt.Unix(), id),
		CreateTime:    completeAt.Add(-time.Minute).Unix(),
		CompleteTime:  completeAt.Unix(),
		Status:        common.TopUpStatusSuccess,
	}).Error)
}

func stubQuotaResetNow(now time.Time) func() {
	old := quotaResetNow
	quotaResetNow = func() time.Time { return now }
	return func() { quotaResetNow = old }
}

func bjTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, beijingLocation)
}
