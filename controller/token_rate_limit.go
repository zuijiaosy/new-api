package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type tokenRPMRequest struct {
	TokenId int `json:"token_id"`
	RPM     int `json:"rpm"`
}

func GetDefaultTokenRPM(c *gin.Context) {
	userId := c.GetInt("id")
	rpm, found, err := model.GetUserDefaultTokenRPM(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if !found {
		rpm = 0
	}
	common.ApiSuccess(c, gin.H{"rpm": rpm})
}

func UpdateDefaultTokenRPM(c *gin.Context) {
	userId := c.GetInt("id")
	req := tokenRPMRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.RPM == 0 {
		if err := model.DeleteUserDefaultTokenRPM(userId); err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, gin.H{"rpm": 0})
		return
	}
	if err := model.ValidateRPMValue(req.RPM); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveUserDefaultTokenRPM(userId, req.RPM); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"rpm": req.RPM})
}

func DeleteDefaultTokenRPM(c *gin.Context) {
	userId := c.GetInt("id")
	if err := model.DeleteUserDefaultTokenRPM(userId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{})
}

func GetTokenRPMOverview(c *gin.Context) {
	userId := c.GetInt("id")
	defaultRPM, defaultFound, err := model.GetUserDefaultTokenRPM(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !defaultFound {
		defaultRPM = 0
	}

	tokenIDsRaw := c.Query("ids")
	if tokenIDsRaw == "" {
		tokenIDsRaw = c.Query("token_ids")
	}

	tokenIDs, err := parseTokenIDs(tokenIDsRaw)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if len(tokenIDs) == 0 {
		common.ApiSuccess(c, gin.H{
			"default_rpm":    defaultRPM,
			"overrides":      map[int]int{},
			"effective_rpms": map[int]int{},
		})
		return
	}

	overrides, err := model.GetTokenCustomRPMMap(userId, tokenIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	effectiveRPMMap := make(map[int]int, len(tokenIDs))
	for _, tokenId := range tokenIDs {
		rpm := defaultRPM
		if overrides[tokenId] > 0 {
			rpm = overrides[tokenId]
		}
		effectiveRPMMap[tokenId] = rpm
	}
	common.ApiSuccess(c, gin.H{
		"default_rpm":    defaultRPM,
		"overrides":      overrides,
		"effective_rpms": effectiveRPMMap,
	})
}

func UpdateTokenRPM(c *gin.Context) {
	req := tokenRPMRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.TokenId <= 0 {
		common.ApiErrorMsg(c, "token_id 无效")
		return
	}
	userId := c.GetInt("id")
	if _, err := model.GetTokenByIds(req.TokenId, userId); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.RPM == 0 {
		if err := model.DeleteTokenCustomRPM(userId, req.TokenId); err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, gin.H{
			"token_id": req.TokenId,
			"rpm":      0,
		})
		return
	}
	if err := model.ValidateRPMValue(req.RPM); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveTokenCustomRPM(userId, req.TokenId, req.RPM); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"token_id": req.TokenId,
		"rpm":      req.RPM,
	})
}

func DeleteTokenRPM(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Query("token_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userId := c.GetInt("id")
	if _, err = model.GetTokenByIds(tokenId, userId); err != nil {
		common.ApiError(c, err)
		return
	}

	if err = model.DeleteTokenCustomRPM(userId, tokenId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{})
}

func parseTokenIDs(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tokenId, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		if tokenId > 0 {
			result = append(result, tokenId)
		}
	}
	return result, nil
}
