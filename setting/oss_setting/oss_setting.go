package oss_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// OssImageSetting 图片 OSS 转存配置
type OssImageSetting struct {
	Enabled            bool `json:"enabled"`
	FallbackToUpstream bool `json:"fallback_to_upstream"`

	Endpoint        string `json:"endpoint"`
	AccessKey       string `json:"access_key"`
	SecretKey       string `json:"secret_key"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	UseSSL          bool   `json:"use_ssl"`
	UsePathStyle    bool   `json:"use_path_style"`
	PublicUrlPrefix string `json:"public_url_prefix"`

	RetentionHours         int `json:"retention_hours"`
	DownloadTimeoutSeconds int `json:"download_timeout_seconds"`
	CleanupIntervalHours   int `json:"cleanup_interval_hours"`
	CleanupBatchSize       int `json:"cleanup_batch_size"`
}

var ossImageSetting = OssImageSetting{
	Enabled:                false,
	FallbackToUpstream:     false,
	Bucket:                 "new-api-images",
	Region:                 "us-east-1",
	UseSSL:                 false,
	UsePathStyle:           true,
	RetentionHours:         24,
	DownloadTimeoutSeconds: 30,
	CleanupIntervalHours:   24,
	CleanupBatchSize:       500,
}

func init() {
	config.GlobalConfig.Register("oss_image_setting", &ossImageSetting)
}

// GetOssImageSetting 返回当前配置的快照（值拷贝），调用方修改不影响全局配置。
// 写入路径由 config.GlobalConfig 统一加锁，与 performance_setting 等包保持一致。
func GetOssImageSetting() OssImageSetting {
	return ossImageSetting
}

// MaskedCopy 返回用于前端展示的副本，SecretKey 脱敏为 ****<后4位>。
func (s OssImageSetting) MaskedCopy() OssImageSetting {
	c := s
	c.SecretKey = MaskSecret(c.SecretKey)
	return c
}

// MaskSecret 返回 `****<后4位>`，长度不足 4 位则全返回 `****`。
func MaskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}

// IsConfigured 判断必填项是否都填了。
func (s OssImageSetting) IsConfigured() bool {
	return s.Endpoint != "" && s.AccessKey != "" && s.SecretKey != "" &&
		s.Bucket != "" && s.PublicUrlPrefix != ""
}
