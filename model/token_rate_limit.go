package model

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultTokenRateLimitTokenID = 0
	tokenRateLimitCacheNamespace = "token_rate_limit:v1"
)

type TokenRateLimit struct {
	Id      int `json:"id"`
	UserId  int `json:"user_id" gorm:"uniqueIndex:idx_user_token_rpm"`
	TokenId int `json:"token_id" gorm:"uniqueIndex:idx_user_token_rpm;default:0"`
	RPM     int `json:"rpm" gorm:"not null"`
}

type tokenRateLimitCacheValue struct {
	Found bool `json:"found"`
	RPM   int  `json:"rpm"`
}

var (
	tokenRateLimitCache     *cachex.HybridCache[tokenRateLimitCacheValue]
	tokenRateLimitCacheOnce sync.Once
)

func getTokenRateLimitCache() *cachex.HybridCache[tokenRateLimitCacheValue] {
	tokenRateLimitCacheOnce.Do(func() {
		ttl := 60 * time.Second
		tokenRateLimitCache = cachex.NewHybridCache[tokenRateLimitCacheValue](cachex.HybridCacheConfig[tokenRateLimitCacheValue]{
			Namespace: cachex.Namespace(tokenRateLimitCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[tokenRateLimitCacheValue]{},
			Memory: func() *hot.HotCache[string, tokenRateLimitCacheValue] {
				return hot.NewHotCache[string, tokenRateLimitCacheValue](hot.LRU, 10000).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return tokenRateLimitCache
}

func tokenRateLimitCacheKey(userId int, tokenId int) string {
	if userId <= 0 || tokenId < 0 {
		return ""
	}
	return strconv.Itoa(userId) + ":" + strconv.Itoa(tokenId)
}

func invalidateTokenRateLimitCaches(userId int, tokenIds []int) {
	for _, tokenId := range tokenIds {
		invalidateTokenRateLimitCache(userId, tokenId)
	}
}

func invalidateTokenRateLimitCache(userId int, tokenId int) {
	cache := getTokenRateLimitCache()
	keys := []string{tokenRateLimitCacheKey(userId, tokenId)}
	if tokenId != defaultTokenRateLimitTokenID {
		keys = append(keys, tokenRateLimitCacheKey(userId, defaultTokenRateLimitTokenID))
	}
	_, _ = cache.DeleteMany(keys)
}

func getTokenRateLimitRecord(userId int, tokenId int) (int, bool, error) {
	if userId <= 0 || tokenId < 0 {
		return 0, false, errors.New("userId 或 tokenId 无效")
	}

	cacheKey := tokenRateLimitCacheKey(userId, tokenId)
	cache := getTokenRateLimitCache()
	if cacheKey != "" {
		if cached, found, err := cache.Get(cacheKey); err == nil && found {
			return cached.RPM, cached.Found, nil
		} else if err != nil {
			common.SysLog("failed to get token rate limit cache: " + err.Error())
		}
	}

	record := TokenRateLimit{}
	err := DB.Where("user_id = ? AND token_id = ?", userId, tokenId).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = cache.SetWithTTL(cacheKey, tokenRateLimitCacheValue{
				Found: false,
				RPM:   0,
			}, 60*time.Second)
			return 0, false, nil
		}
		return 0, false, err
	}

	_ = cache.SetWithTTL(cacheKey, tokenRateLimitCacheValue{
		Found: true,
		RPM:   record.RPM,
	}, 60*time.Second)
	return record.RPM, true, nil
}

func saveTokenRateLimit(userId int, tokenId int, rpm int) error {
	if userId <= 0 || tokenId < 0 {
		return errors.New("userId 或 tokenId 无效")
	}
	if rpm < 1 {
		return errors.New("rpm 必须大于等于 1")
	}

	record := TokenRateLimit{
		UserId:  userId,
		TokenId: tokenId,
		RPM:     rpm,
	}
	err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "token_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"rpm"}),
	}).Create(&record).Error
	if err != nil {
		return err
	}
	invalidateTokenRateLimitCache(userId, tokenId)
	return nil
}

func deleteTokenRateLimit(userId int, tokenId int) error {
	if userId <= 0 || tokenId < 0 {
		return errors.New("userId 或 tokenId 无效")
	}
	if err := DB.Where("user_id = ? AND token_id = ?", userId, tokenId).Delete(&TokenRateLimit{}).Error; err != nil {
		return err
	}
	invalidateTokenRateLimitCache(userId, tokenId)
	return nil
}

func GetUserDefaultTokenRPM(userId int) (int, bool, error) {
	return getTokenRateLimitRecord(userId, defaultTokenRateLimitTokenID)
}

func SaveUserDefaultTokenRPM(userId int, rpm int) error {
	return saveTokenRateLimit(userId, defaultTokenRateLimitTokenID, rpm)
}

func DeleteUserDefaultTokenRPM(userId int) error {
	return deleteTokenRateLimit(userId, defaultTokenRateLimitTokenID)
}

func GetTokenCustomRPM(userId int, tokenId int) (int, bool, error) {
	return getTokenRateLimitRecord(userId, tokenId)
}

func GetTokenCustomRPMMap(userId int, tokenIds []int) (map[int]int, error) {
	result := make(map[int]int)
	if userId <= 0 {
		return result, errors.New("userId 无效")
	}
	query := DB.Model(&TokenRateLimit{}).
		Where("user_id = ?", userId).
		Where("token_id > ?", defaultTokenRateLimitTokenID)

	if len(tokenIds) > 0 {
		query = query.Where("token_id IN ?", tokenIds)
	}

	records := make([]TokenRateLimit, 0)
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	for _, record := range records {
		result[record.TokenId] = record.RPM
		_ = getTokenRateLimitCache().SetWithTTL(
			tokenRateLimitCacheKey(userId, record.TokenId),
			tokenRateLimitCacheValue{
				Found: true,
				RPM:   record.RPM,
			},
			60*time.Second,
		)
	}
	return result, nil
}

func SaveTokenCustomRPM(userId int, tokenId int, rpm int) error {
	return saveTokenRateLimit(userId, tokenId, rpm)
}

func DeleteTokenCustomRPM(userId int, tokenId int) error {
	return deleteTokenRateLimit(userId, tokenId)
}

func deleteTokenRateLimitRows(db *gorm.DB, userId int, tokenIds []int) error {
	if userId <= 0 || len(tokenIds) == 0 {
		return nil
	}
	if err := db.Where("user_id = ? AND token_id IN ?", userId, tokenIds).Delete(&TokenRateLimit{}).Error; err != nil {
		return err
	}
	return nil
}

func DeleteTokenRateLimitsByTokenIds(userId int, tokenIds []int) error {
	if err := deleteTokenRateLimitRows(DB, userId, tokenIds); err != nil {
		return err
	}
	invalidateTokenRateLimitCaches(userId, tokenIds)
	return nil
}

func GetEffectiveTokenRPM(userId int, tokenId int) (rpm int, usesDefault bool, found bool, err error) {
	rpm, found, err = GetTokenCustomRPM(userId, tokenId)
	if err != nil {
		return 0, false, false, err
	}
	if found {
		return rpm, false, true, nil
	}

	rpm, found, err = GetUserDefaultTokenRPM(userId)
	if err != nil {
		return 0, false, false, err
	}
	if found {
		return rpm, true, true, nil
	}
	return 0, false, false, nil
}

func (TokenRateLimit) TableName() string {
	return "token_rate_limits"
}

func ValidateRPMValue(rpm int) error {
	if rpm < 1 {
		return fmt.Errorf("rpm 必须大于等于 1")
	}
	if rpm > math.MaxInt32 {
		return fmt.Errorf("rpm 最大值为 2147483647")
	}
	return nil
}
