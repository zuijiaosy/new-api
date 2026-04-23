package oss

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/oss_setting"
)

// Storage 是图片 OSS 后端抽象。配置变更后通过 BumpStorageVersion 触发重建。
type Storage interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, mime string) (publicURL string, err error)
	Delete(ctx context.Context, key string) error
	BatchDelete(ctx context.Context, keys []string) (deleted int, failed []string, err error)
	Ping(ctx context.Context) error
}

// StorageBuilder 由具体实现通过 init() 注册；只挂一个全局构造器，便于测试替换。
type StorageBuilder func(cfg oss_setting.OssImageSetting) (Storage, error)

var (
	builderMu  sync.RWMutex
	builder    StorageBuilder
	cachedMu   sync.Mutex
	cached     Storage
	cachedVer  uint64
	versionCnt atomic.Uint64
)

// RegisterStorageBuilder 由 MinIO 实现在 init() 中注册。
func RegisterStorageBuilder(b StorageBuilder) {
	builderMu.Lock()
	defer builderMu.Unlock()
	builder = b
}

// BumpStorageVersion 标记需要重建（配置变更后调用）。
func BumpStorageVersion() {
	versionCnt.Add(1)
}

func init() {
	// 订阅配置变更：option 落盘后 Bump 缓存版本，下一次 GetStorage 重建客户端。
	oss_setting.RegisterConfigChangeHandler(BumpStorageVersion)
}

// GetStorage 返回当前配置对应的 Storage 实例。
// 若配置不足或未注册 builder，返回 ErrStorageNotConfigured。
// 实例按 versionCnt 缓存；版本号变更后下一次调用重建，正在使用旧实例的调用者不受影响。
func GetStorage() (Storage, error) {
	cfg := oss_setting.GetOssImageSetting()
	if !cfg.IsConfigured() {
		return nil, ErrStorageNotConfigured
	}

	builderMu.RLock()
	b := builder
	builderMu.RUnlock()
	if b == nil {
		return nil, ErrStorageNotConfigured
	}

	wantVer := versionCnt.Load()
	cachedMu.Lock()
	defer cachedMu.Unlock()
	if cached != nil && cachedVer == wantVer {
		return cached, nil
	}

	s, err := b(cfg)
	if err != nil {
		return nil, err
	}
	cached = s
	cachedVer = wantVer
	return cached, nil
}

// NewTransientStorage 根据入参构造一次性 Storage（用于 Ping 按当前表单值测试，不入缓存）。
func NewTransientStorage(cfg oss_setting.OssImageSetting) (Storage, error) {
	builderMu.RLock()
	b := builder
	builderMu.RUnlock()
	if b == nil {
		return nil, ErrStorageNotConfigured
	}
	return b(cfg)
}
