package oss

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/oss_setting"
)

// TestMinIOE2E 需要真实 MinIO。未配置环境变量时跳过。
func TestMinIOE2E(t *testing.T) {
	endpoint := os.Getenv("TEST_MINIO_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_MINIO_ENDPOINT not set; skipping MinIO e2e")
	}

	cfg := oss_setting.OssImageSetting{
		Enabled:         true,
		Endpoint:        endpoint,
		AccessKey:       os.Getenv("TEST_MINIO_AK"),
		SecretKey:       os.Getenv("TEST_MINIO_SK"),
		Bucket:          os.Getenv("TEST_MINIO_BUCKET"),
		Region:          "us-east-1",
		UseSSL:          os.Getenv("TEST_MINIO_SSL") == "true",
		UsePathStyle:    true,
		PublicUrlPrefix: os.Getenv("TEST_MINIO_PUBLIC_PREFIX"),
	}
	s, err := NewTransientStorage(cfg)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	key := "images/test/plan-verify.png"
	body := bytes.NewReader([]byte("hello-oss"))
	url, err := s.Put(ctx, key, body, int64(body.Len()), "image/png")
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if !strings.HasPrefix(url, cfg.PublicUrlPrefix) {
		t.Fatalf("url should start with prefix: %s", url)
	}

	n, failed, err := s.BatchDelete(ctx, []string{key, "images/test/not-exist.png"})
	if err != nil {
		t.Fatalf("batch delete: %v", err)
	}
	if n < 1 || len(failed) > 1 {
		t.Fatalf("unexpected delete result: n=%d failed=%v", n, failed)
	}
}
