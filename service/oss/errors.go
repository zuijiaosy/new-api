package oss

import "errors"

// 对外暴露的错误哨兵，上层可通过 errors.Is 判定。
var (
	ErrStorageNotConfigured = errors.New("oss storage not configured")
	ErrUpstreamDownload     = errors.New("failed to download upstream image")
	ErrStorageUpload        = errors.New("failed to upload to storage")
	ErrStoragePersist       = errors.New("failed to persist oss image record")
	ErrCleanupInProgress    = errors.New("cleanup already in progress")
)
