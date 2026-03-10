package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const (
	tokenRPMLimitDurationSeconds int64 = 60
	TokenRPMLimitMark                  = "TRPM"
)

func tokenRPMExceededMessage(rpm int) string {
	return fmt.Sprintf("当前令牌已达到 RPM 限制：1分钟内最多请求%d次", rpm)
}

func redisTokenRPMLimitHandler(c *gin.Context, tokenId int, rpm int) bool {
	ctx := context.Background()
	key := fmt.Sprintf("rateLimit:%s:token:%d", TokenRPMLimitMark, tokenId)

	allowed, err := checkRedisRateLimit(ctx, common.RDB, key, rpm, tokenRPMLimitDurationSeconds)
	if err != nil {
		abortWithOpenAiMessage(c, http.StatusInternalServerError, "token_rpm_rate_limit_check_failed")
		return false
	}
	if !allowed {
		abortWithOpenAiMessage(c, http.StatusTooManyRequests, tokenRPMExceededMessage(rpm))
		return false
	}

	recordRedisRequest(ctx, common.RDB, key, rpm)
	return true
}

func memoryTokenRPMLimitHandler(c *gin.Context, tokenId int, rpm int) bool {
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)

	key := fmt.Sprintf("%s:token:%d", TokenRPMLimitMark, tokenId)
	if !inMemoryRateLimiter.Request(key, rpm, tokenRPMLimitDurationSeconds) {
		abortWithOpenAiMessage(c, http.StatusTooManyRequests, tokenRPMExceededMessage(rpm))
		return false
	}
	return true
}

func TokenRPMLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		tokenId := c.GetInt("token_id")
		if userId <= 0 || tokenId <= 0 {
			c.Next()
			return
		}

		rpm, _, found, err := model.GetEffectiveTokenRPM(userId, tokenId)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "token_rpm_rate_limit_lookup_failed")
			return
		}
		if !found || rpm <= 0 {
			c.Next()
			return
		}

		var allowed bool
		if common.RedisEnabled && common.RDB != nil {
			allowed = redisTokenRPMLimitHandler(c, tokenId, rpm)
		} else {
			allowed = memoryTokenRPMLimitHandler(c, tokenId, rpm)
		}
		if !allowed {
			return
		}

		c.Next()
	}
}
