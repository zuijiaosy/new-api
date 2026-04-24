package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newOssImageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&OssImage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBatchCreateAndListExpired(t *testing.T) {
	db := newOssImageTestDB(t)
	oldDB := DB
	DB = db
	t.Cleanup(func() { DB = oldDB })

	now := time.Now().Unix()
	imgs := []OssImage{
		{FileKey: "k1", PublicUrl: "u1", MimeType: "image/png", SizeBytes: 1, CreatedAt: now - 7200},
		{FileKey: "k2", PublicUrl: "u2", MimeType: "image/png", SizeBytes: 2, CreatedAt: now - 3600},
		{FileKey: "k3", PublicUrl: "u3", MimeType: "image/png", SizeBytes: 3, CreatedAt: now},
	}
	if err := BatchCreateOssImages(imgs); err != nil {
		t.Fatalf("batch create: %v", err)
	}

	// threshold = now - 3600：严格小于才算过期
	expired, err := ListExpiredOssImages(now-3600, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("want 1 expired (k1), got %d", len(expired))
	}
	if expired[0].FileKey != "k1" {
		t.Fatalf("want k1, got %s", expired[0].FileKey)
	}
}

func TestDeleteOssImagesByIds(t *testing.T) {
	db := newOssImageTestDB(t)
	oldDB := DB
	DB = db
	t.Cleanup(func() { DB = oldDB })

	if err := BatchCreateOssImages([]OssImage{
		{FileKey: "a"}, {FileKey: "b"}, {FileKey: "c"},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 空切片 no-op
	n, err := DeleteOssImagesByIds(nil)
	if err != nil || n != 0 {
		t.Fatalf("nil ids: n=%d err=%v", n, err)
	}

	// 删除前两条（含一个不存在 id）
	var all []OssImage
	DB.Find(&all)
	ids := []int64{all[0].Id, all[1].Id, 99999}
	n, err = DeleteOssImagesByIds(ids)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 deleted, got %d", n)
	}

	var remain []OssImage
	DB.Find(&remain)
	if len(remain) != 1 || remain[0].FileKey != "c" {
		t.Fatalf("unexpected remain: %+v", remain)
	}
}
