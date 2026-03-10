package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type tokenRPMOverviewResponse struct {
	DefaultRPM      int            `json:"default_rpm"`
	Overrides       map[string]int `json:"overrides"`
	EffectiveRPMs   map[string]int `json:"effective_rpms"`
	TokenRPMMap     map[string]int `json:"token_rpm_map"`
	EffectiveRPMMap map[string]int `json:"effective_rpm_map"`
}

type tokenRPMOverviewAPIResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    tokenRPMOverviewResponse `json:"data"`
}

func TestGetTokenRPMOverviewContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := model.DB
	oldLogDB := model.LOG_DB

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	defer func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	}()

	if err := db.AutoMigrate(&model.User{}, &model.Token{}, &model.TokenRateLimit{}); err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	user := &model.User{
		Username: "rpm-contract-user",
		Password: "password123",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	tokenA := &model.Token{
		UserId:         user.Id,
		Key:            "token-contract-a",
		Name:           "token-contract-a",
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}
	tokenB := &model.Token{
		UserId:         user.Id,
		Key:            "token-contract-b",
		Name:           "token-contract-b",
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
	}
	if err := db.Create(tokenA).Error; err != nil {
		t.Fatalf("failed to create tokenA: %v", err)
	}
	if err := db.Create(tokenB).Error; err != nil {
		t.Fatalf("failed to create tokenB: %v", err)
	}

	records := []*model.TokenRateLimit{
		{
			UserId:  user.Id,
			TokenId: 0,
			RPM:     20,
		},
		{
			UserId:  user.Id,
			TokenId: tokenA.Id,
			RPM:     7,
		},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("failed to create token rpm record: %v", err)
		}
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", user.Id)
		c.Next()
	})
	router.GET("/api/token/rpm", GetTokenRPMOverview)

	t.Run("ids 参数返回稳定字段", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodGet,
			"/api/token/rpm?ids="+strconv.Itoa(tokenA.Id),
			nil,
		)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", response.Code)
		}

		var payload tokenRPMOverviewAPIResponse
		if err := common.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if !payload.Success {
			t.Fatalf("expected success response, got message: %s", payload.Message)
		}
		if payload.Data.DefaultRPM != 20 {
			t.Fatalf("expected default rpm 20, got %d", payload.Data.DefaultRPM)
		}
		if payload.Data.Overrides[itoa(tokenA.Id)] != 7 {
			t.Fatalf("expected override rpm 7, got %d", payload.Data.Overrides[itoa(tokenA.Id)])
		}
		if payload.Data.EffectiveRPMs[itoa(tokenA.Id)] != 7 {
			t.Fatalf("expected effective rpm 7, got %d", payload.Data.EffectiveRPMs[itoa(tokenA.Id)])
		}
		if payload.Data.TokenRPMMap != nil {
			t.Fatalf("expected token_rpm_map to be absent")
		}
		if payload.Data.EffectiveRPMMap != nil {
			t.Fatalf("expected effective_rpm_map to be absent")
		}
	})

	t.Run("兼容 token_ids 参数", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodGet,
			"/api/token/rpm?token_ids="+itoa(tokenB.Id),
			nil,
		)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", response.Code)
		}

		var payload tokenRPMOverviewAPIResponse
		if err := common.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if payload.Data.Overrides[itoa(tokenB.Id)] != 0 {
			t.Fatalf("expected override rpm 0, got %d", payload.Data.Overrides[itoa(tokenB.Id)])
		}
		if payload.Data.EffectiveRPMs[itoa(tokenB.Id)] != 20 {
			t.Fatalf("expected effective rpm 20, got %d", payload.Data.EffectiveRPMs[itoa(tokenB.Id)])
		}
	})
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
