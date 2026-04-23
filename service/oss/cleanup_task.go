package oss

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/oss_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

// CleanupReport 单次清理统计。
type CleanupReport struct {
	Scanned   int   `json:"scanned"`
	Deleted   int   `json:"deleted"`
	Failed    int   `json:"failed"`
	ElapsedMs int64 `json:"elapsed_ms"`
}

// cleanupGuard 用 atomic.Bool 实现单实例互斥。
type cleanupGuard struct{ running atomic.Bool }

func (g *cleanupGuard) tryAcquire() bool { return g.running.CompareAndSwap(false, true) }
func (g *cleanupGuard) release()         { g.running.Store(false) }

var (
	cleanupOnce   sync.Once
	globalCleanup = &cleanupGuard{}
)

// computeCleanupInterval 非正数回落到 24h。
func computeCleanupInterval(hours int) time.Duration {
	if hours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

// StartOssImageCleanupTask 进程级定时任务。
// 仅 master 节点启动；sync.Once 保证只启动一次。
// 周期由 cfg.CleanupIntervalHours 决定，运行中修改仅在下次进程启动生效。
func StartOssImageCleanupTask() {
	cleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		cfg := oss_setting.GetOssImageSetting()
		interval := computeCleanupInterval(cfg.CleanupIntervalHours)

		gopool.Go(func() {
			common.SysLog(fmt.Sprintf("oss cleanup task started: interval=%s", interval))
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				if _, err := RunOssImageCleanupOnce(context.Background()); err != nil && !errors.Is(err, ErrCleanupInProgress) {
					common.SysError(fmt.Sprintf("oss cleanup tick failed: %v", err))
				}
			}
		})
	})
}

// RunOssImageCleanupOnce 单次清理入口，定时 & 手动共用。
// 并发触发返回 ErrCleanupInProgress；未启用 / 未配置直接返回空 report。
func RunOssImageCleanupOnce(ctx context.Context) (CleanupReport, error) {
	if !globalCleanup.tryAcquire() {
		return CleanupReport{}, ErrCleanupInProgress
	}
	defer globalCleanup.release()

	start := time.Now()
	cfg := oss_setting.GetOssImageSetting()
	if !cfg.Enabled || !cfg.IsConfigured() {
		return CleanupReport{ElapsedMs: time.Since(start).Milliseconds()}, nil
	}

	storage, err := GetStorage()
	if err != nil {
		return CleanupReport{ElapsedMs: time.Since(start).Milliseconds()}, err
	}

	retention := cfg.RetentionHours
	if retention <= 0 {
		retention = 24
	}
	threshold := time.Now().Unix() - int64(retention)*3600
	batch := cfg.CleanupBatchSize
	if batch <= 0 {
		batch = 500
	}

	report := CleanupReport{}
	for {
		imgs, err := model.ListExpiredOssImages(threshold, batch)
		if err != nil {
			report.ElapsedMs = time.Since(start).Milliseconds()
			return report, err
		}
		if len(imgs) == 0 {
			break
		}
		report.Scanned += len(imgs)

		keys := make([]string, 0, len(imgs))
		ids := make([]int64, 0, len(imgs))
		for _, im := range imgs {
			keys = append(keys, im.FileKey)
			ids = append(ids, im.Id)
		}
		_, failed, delErr := storage.BatchDelete(ctx, keys)
		if delErr != nil {
			common.SysError(fmt.Sprintf("oss batch delete failed: %v", delErr))
			// 整批失败：不删 DB，避免孤立记录；下次再来
			report.Failed += len(keys)
			break
		}
		report.Failed += len(failed)

		if _, err := model.DeleteOssImagesByIds(ids); err != nil {
			common.SysError(fmt.Sprintf("oss db delete failed: %v", err))
			break
		}
		report.Deleted += len(imgs) - len(failed)

		if len(imgs) < batch {
			break
		}
	}
	report.ElapsedMs = time.Since(start).Milliseconds()
	common.SysLog(fmt.Sprintf("oss cleanup done: scanned=%d deleted=%d failed=%d elapsed=%dms",
		report.Scanned, report.Deleted, report.Failed, report.ElapsedMs))
	return report, nil
}
