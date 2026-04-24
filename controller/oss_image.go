package controller

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/oss"
	"github.com/QuantumNous/new-api/setting/oss_setting"

	"github.com/gin-gonic/gin"
)

// PingOssImage 用入参构造临时 Storage，Put 一个小对象再 Delete 做健康探测。
// SecretKey 空串视为沿用已保存值（前端脱敏占位的兼容）。
func PingOssImage(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	var req oss_setting.OssImageSetting
	if err := common.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	if req.SecretKey == "" {
		req.SecretKey = oss_setting.GetOssImageSetting().SecretKey
	}

	start := time.Now()
	storage, err := oss.NewTransientStorage(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	probeKey := "images/_ping/" + common.GetRandomString(16)
	probeBody := []byte("new-api-oss-ping")
	if _, err := storage.Put(ctx, probeKey, bytes.NewReader(probeBody), int64(len(probeBody)), "application/octet-stream"); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = storage.Delete(ctx, probeKey)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"latency_ms": time.Since(start).Milliseconds(),
		"message":    "ok",
	})
}

// CleanupOssImages 管理员手动触发；运行中返回 409。
func CleanupOssImages(c *gin.Context) {
	report, err := oss.RunOssImageCleanupOnce(c.Request.Context())
	if err != nil {
		if errors.Is(err, oss.ErrCleanupInProgress) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"scanned":    report.Scanned,
		"deleted":    report.Deleted,
		"failed":     report.Failed,
		"elapsed_ms": report.ElapsedMs,
	})
}
