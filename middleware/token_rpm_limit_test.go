package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTokenRPMLimitUsesExplicitAndDefaultRPM(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	oldLimiter := inMemoryRateLimiter

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	defer func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		inMemoryRateLimiter = oldLimiter
	}()

	if err := db.AutoMigrate(&model.User{}, &model.Token{}, &model.TokenRateLimit{}); err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	user := &model.User{
		Username: "tester",
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&model.TokenRateLimit{UserId: user.Id, TokenId: 0, RPM: 2}).Error; err != nil {
		t.Fatalf("failed to create default token rpm: %v", err)
	}

	tokenA := &model.Token{UserId: user.Id, Key: "tokena", Name: "token-a", Status: common.TokenStatusEnabled, CreatedTime: 1, AccessedTime: 1, ExpiredTime: -1, UnlimitedQuota: true}
	tokenB := &model.Token{UserId: user.Id, Key: "tokenb", Name: "token-b", Status: common.TokenStatusEnabled, CreatedTime: 1, AccessedTime: 1, ExpiredTime: -1, UnlimitedQuota: true}
	if err := db.Create(tokenA).Error; err != nil {
		t.Fatalf("failed to create tokenA: %v", err)
	}
	if err := db.Create(tokenB).Error; err != nil {
		t.Fatalf("failed to create tokenB: %v", err)
	}

	if err := db.Create(&model.TokenRateLimit{UserId: user.Id, TokenId: tokenA.Id, RPM: 1}).Error; err != nil {
		t.Fatalf("failed to create token rate limit: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		currentToken := tokenA
		if c.GetHeader("X-Token-Id") == "b" {
			currentToken = tokenB
		}
		if err := SetupContextForToken(c, currentToken); err != nil {
			t.Fatalf("failed to setup token context: %v", err)
		}
		c.Next()
	})
	router.Use(TokenRPMLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	requestA1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	responseA1 := httptest.NewRecorder()
	router.ServeHTTP(responseA1, requestA1)
	if responseA1.Code != http.StatusOK {
		t.Fatalf("expected tokenA first request 200, got %d", responseA1.Code)
	}

	requestA2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	responseA2 := httptest.NewRecorder()
	router.ServeHTTP(responseA2, requestA2)
	if responseA2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected tokenA second request 429, got %d", responseA2.Code)
	}
	assertTokenRPMErrorMessage(t, responseA2, 1)

	requestB1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	requestB1.Header.Set("X-Token-Id", "b")
	responseB1 := httptest.NewRecorder()
	router.ServeHTTP(responseB1, requestB1)
	if responseB1.Code != http.StatusOK {
		t.Fatalf("expected tokenB first request 200, got %d", responseB1.Code)
	}

	requestB2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	requestB2.Header.Set("X-Token-Id", "b")
	responseB2 := httptest.NewRecorder()
	router.ServeHTTP(responseB2, requestB2)
	if responseB2.Code != http.StatusOK {
		t.Fatalf("expected tokenB second request 200, got %d", responseB2.Code)
	}

	requestB3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	requestB3.Header.Set("X-Token-Id", "b")
	responseB3 := httptest.NewRecorder()
	router.ServeHTTP(responseB3, requestB3)
	if responseB3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected tokenB third request 429, got %d", responseB3.Code)
	}
	assertTokenRPMErrorMessage(t, responseB3, 2)
}

func TestTokenRPMLimitDoesNotSkipWhitelistedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldRedisEnabled := common.RedisEnabled
	oldLimiter := inMemoryRateLimiter
	oldWhitelist := common.RateLimitWhitelistIPs

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.RateLimitWhitelistIPs = []string{"127.0.0.1"}
	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	defer func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.RedisEnabled = oldRedisEnabled
		common.RateLimitWhitelistIPs = oldWhitelist
		inMemoryRateLimiter = oldLimiter
	}()

	if err := db.AutoMigrate(&model.User{}, &model.Token{}, &model.TokenRateLimit{}); err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	user := &model.User{
		Username: "whitelist-tester",
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token := &model.Token{
		UserId:         user.Id,
		Key:            "whitelist-token",
		Name:           "whitelist-token",
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if err := db.Create(&model.TokenRateLimit{UserId: user.Id, TokenId: token.Id, RPM: 1}).Error; err != nil {
		t.Fatalf("failed to create token rate limit: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		if err := SetupContextForToken(c, token); err != nil {
			t.Fatalf("failed to setup token context: %v", err)
		}
		c.Next()
	})
	router.Use(TokenRPMLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request1.RemoteAddr = "127.0.0.1:12345"
	response1 := httptest.NewRecorder()
	router.ServeHTTP(response1, request1)
	if response1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", response1.Code)
	}

	request2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request2.RemoteAddr = "127.0.0.1:12345"
	response2 := httptest.NewRecorder()
	router.ServeHTTP(response2, request2)
	if response2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", response2.Code)
	}
	assertTokenRPMErrorMessage(t, response2, 1)
}

func assertTokenRPMErrorMessage(t *testing.T, recorder *httptest.ResponseRecorder, rpm int) {
	t.Helper()

	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if payload.Error.Message == "" {
		t.Fatalf("expected non-empty error message")
	}
	if want := tokenRPMExceededMessage(rpm); payload.Error.Message[:len(want)] != want {
		t.Fatalf("expected error message prefix %q, got %q", want, payload.Error.Message)
	}
}
