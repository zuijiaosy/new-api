package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSaveTokenCustomRPMUpsert(t *testing.T) {
	oldDB := DB
	oldLogDB := LOG_DB

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	DB = db
	LOG_DB = db

	defer func() {
		DB = oldDB
		LOG_DB = oldLogDB
	}()

	if err := db.AutoMigrate(&User{}, &Token{}, &TokenRateLimit{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	user, token := createTokenRateLimitTestUserAndToken(t, db, "rpm-upsert-user", "rpm-upsert-token")

	if err := SaveTokenCustomRPM(user.Id, token.Id, 10); err != nil {
		t.Fatalf("failed to save first token rpm: %v", err)
	}
	if err := SaveTokenCustomRPM(user.Id, token.Id, 15); err != nil {
		t.Fatalf("failed to save second token rpm: %v", err)
	}

	var count int64
	if err := db.Model(&TokenRateLimit{}).
		Where("user_id = ? AND token_id = ?", user.Id, token.Id).
		Count(&count).Error; err != nil {
		t.Fatalf("failed to count token rate limit rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one token rate limit row, got %d", count)
	}

	rpm, found, err := GetTokenCustomRPM(user.Id, token.Id)
	if err != nil {
		t.Fatalf("failed to query token rpm: %v", err)
	}
	if !found {
		t.Fatalf("expected token rpm record to exist")
	}
	if rpm != 15 {
		t.Fatalf("expected rpm 15, got %d", rpm)
	}
}

func TestDeleteTokenWithRateLimitsById(t *testing.T) {
	oldDB := DB
	oldLogDB := LOG_DB
	oldRedisEnabled := common.RedisEnabled

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	DB = db
	LOG_DB = db
	common.RedisEnabled = false

	defer func() {
		DB = oldDB
		LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
	}()

	if err := db.AutoMigrate(&User{}, &Token{}, &TokenRateLimit{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	user, token := createTokenRateLimitTestUserAndToken(t, db, "rpm-delete-user", "rpm-delete-token")
	if err := db.Create(&TokenRateLimit{
		UserId:  user.Id,
		TokenId: token.Id,
		RPM:     9,
	}).Error; err != nil {
		t.Fatalf("failed to create token rpm record: %v", err)
	}

	if err := DeleteTokenWithRateLimitsById(token.Id, user.Id); err != nil {
		t.Fatalf("failed to delete token with rate limits: %v", err)
	}

	var tokenCount int64
	if err := db.Model(&Token{}).Where("id = ?", token.Id).Count(&tokenCount).Error; err != nil {
		t.Fatalf("failed to count tokens: %v", err)
	}
	if tokenCount != 0 {
		t.Fatalf("expected token to be deleted, got %d rows", tokenCount)
	}

	var rateLimitCount int64
	if err := db.Model(&TokenRateLimit{}).
		Where("user_id = ? AND token_id = ?", user.Id, token.Id).
		Count(&rateLimitCount).Error; err != nil {
		t.Fatalf("failed to count token rpm rows: %v", err)
	}
	if rateLimitCount != 0 {
		t.Fatalf("expected token rpm to be deleted, got %d rows", rateLimitCount)
	}
}

func createTokenRateLimitTestUserAndToken(t *testing.T, db *gorm.DB, username string, tokenKey string) (*User, *Token) {
	t.Helper()

	user := &User{
		Username: username,
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token := &Token{
		UserId:         user.Id,
		Key:            tokenKey,
		Name:           tokenKey,
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	return user, token
}
