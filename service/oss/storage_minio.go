package oss

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/setting/oss_setting"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioStorage struct {
	client          *minio.Client
	bucket          string
	publicUrlPrefix string
}

func newMinioStorage(cfg oss_setting.OssImageSetting) (Storage, error) {
	endpoint := cfg.Endpoint
	// 允许用户填 http://host:port 或 host:port，minio-go 需要裸的 host:port。
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		endpoint = u.Host
	}
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageNotConfigured, err)
	}
	return &minioStorage{
		client:          cli,
		bucket:          cfg.Bucket,
		publicUrlPrefix: strings.TrimRight(cfg.PublicUrlPrefix, "/"),
	}, nil
}

func (s *minioStorage) Put(ctx context.Context, key string, body io.Reader, size int64, mime string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: mime,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageUpload, err)
	}
	// 约定：{PublicUrlPrefix}/{bucket}/{key}
	return fmt.Sprintf("%s/%s/%s", s.publicUrlPrefix, s.bucket, key), nil
}

func (s *minioStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *minioStorage) BatchDelete(ctx context.Context, keys []string) (int, []string, error) {
	if len(keys) == 0 {
		return 0, nil, nil
	}
	objCh := make(chan minio.ObjectInfo, len(keys))
	for _, k := range keys {
		objCh <- minio.ObjectInfo{Key: k}
	}
	close(objCh)

	errCh := s.client.RemoveObjects(ctx, s.bucket, objCh, minio.RemoveObjectsOptions{})
	failed := make([]string, 0)
	for e := range errCh {
		if e.Err != nil {
			// NoSuchKey 视作已删除，不计失败
			resp := minio.ToErrorResponse(e.Err)
			if resp.Code == "NoSuchKey" {
				continue
			}
			failed = append(failed, e.ObjectName)
		}
	}
	return len(keys) - len(failed), failed, nil
}

func (s *minioStorage) Ping(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", s.bucket)
	}
	return nil
}

func init() {
	RegisterStorageBuilder(newMinioStorage)
}
