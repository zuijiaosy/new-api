package oss

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/oss_setting"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// RelayMeta 是拦截器需要的请求上下文元数据切片，避免引入 relay 包循环依赖。
type RelayMeta struct {
	UserId    int
	ChannelId int
	TokenId   int
	ModelName string
}

// OssImageRepo 抽象 DB 写入，方便测试注入。
type OssImageRepo interface {
	BatchCreate(imgs []model.OssImage) error
}

type defaultRepo struct{}

func (defaultRepo) BatchCreate(imgs []model.OssImage) error {
	return model.BatchCreateOssImages(imgs)
}

// ImageURLInterceptor 负责把 ImageResponse.data[].url 替换为 OSS URL。
type ImageURLInterceptor struct {
	storage Storage
	repo    OssImageRepo
	cfg     oss_setting.OssImageSetting
}

// NewImageURLInterceptor 根据当前全局配置构造拦截器。若未启用返回 nil 表示无须拦截。
func NewImageURLInterceptor() *ImageURLInterceptor {
	cfg := oss_setting.GetOssImageSetting()
	if !cfg.Enabled {
		return nil
	}
	if !cfg.IsConfigured() {
		common.SysLog("oss image interceptor enabled but not configured, skip")
		return nil
	}
	s, err := GetStorage()
	if err != nil {
		common.SysLog(fmt.Sprintf("oss image interceptor: storage unavailable: %v", err))
		return nil
	}
	return &ImageURLInterceptor{storage: s, repo: defaultRepo{}, cfg: cfg}
}

// Intercept 返回（新 body，是否修改过 body，错误）。
// Enabled=false / body 非 ImageResponse / 全是 b64 → 原样返回，changed=false。
// 严格模式任意失败 → 返回 error，上层放弃计费。
// 降级模式：失败项保留原 URL，不写 DB。
func (i *ImageURLInterceptor) Intercept(ctx context.Context, body []byte, meta *RelayMeta) ([]byte, bool, error) {
	if i == nil || !i.cfg.Enabled {
		return body, false, nil
	}
	var resp dto.ImageResponse
	if err := common.Unmarshal(body, &resp); err != nil || len(resp.Data) == 0 {
		return body, false, nil
	}

	type task struct {
		idx int
		url string
	}
	tasks := make([]task, 0, len(resp.Data))
	for idx, d := range resp.Data {
		if d.Url != "" && d.B64Json == "" {
			tasks = append(tasks, task{idx: idx, url: d.Url})
		}
	}
	if len(tasks) == 0 {
		return body, false, nil
	}

	timeout := time.Duration(i.cfg.DownloadTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	httpClient := &http.Client{Timeout: timeout}

	type result struct {
		idx         int
		upstreamUrl string
		publicUrl   string
		fileKey     string
		mime        string
		size        int64
		err         error
	}
	results := make([]result, len(tasks))

	strict := !i.cfg.FallbackToUpstream
	eg, gctx := errgroup.WithContext(ctx)
	for i2, tk := range tasks {
		eg.Go(func() error {
			res := result{idx: tk.idx, upstreamUrl: tk.url}
			payload, mimeType, err := downloadImage(gctx, httpClient, tk.url)
			if err != nil {
				res.err = fmt.Errorf("%w: %v", ErrUpstreamDownload, err)
				results[i2] = res
				if strict {
					return res.err
				}
				return nil
			}
			key := buildObjectKey(mimeType)
			publicUrl, err := i.storage.Put(gctx, key, bytes.NewReader(payload), int64(len(payload)), mimeType)
			if err != nil {
				res.err = fmt.Errorf("%w: %v", ErrStorageUpload, err)
				results[i2] = res
				if strict {
					return res.err
				}
				return nil
			}
			res.fileKey = key
			res.mime = mimeType
			res.size = int64(len(payload))
			res.publicUrl = publicUrl
			results[i2] = res
			return nil
		})
	}
	waitErr := eg.Wait()

	if strict {
		// 严格模式任意失败：把已成功上传的对象回滚删除，避免孤儿。
		var firstErr error
		for _, r := range results {
			if r.err != nil && firstErr == nil {
				firstErr = r.err
			}
		}
		if firstErr == nil {
			firstErr = waitErr
		}
		if firstErr != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			for _, r := range results {
				if r.err == nil && r.fileKey != "" {
					if delErr := i.storage.Delete(cleanupCtx, r.fileKey); delErr != nil {
						common.SysLog(fmt.Sprintf("oss strict rollback delete failed key=%s: %v", r.fileKey, delErr))
					}
				}
			}
			cancel()
			return nil, false, firstErr
		}
	}

	persistBatch := make([]model.OssImage, 0, len(results))
	for _, r := range results {
		if r.err != nil {
			continue
		}
		persistBatch = append(persistBatch, model.OssImage{
			FileKey:     r.fileKey,
			PublicUrl:   r.publicUrl,
			MimeType:    r.mime,
			SizeBytes:   r.size,
			UpstreamUrl: r.upstreamUrl,
			UserId:      meta.UserId,
			ChannelId:   meta.ChannelId,
			TokenId:     meta.TokenId,
			ModelName:   meta.ModelName,
		})
	}
	if len(persistBatch) > 0 {
		if err := i.repo.BatchCreate(persistBatch); err != nil {
			if !i.cfg.FallbackToUpstream {
				return nil, false, fmt.Errorf("%w: %v", ErrStoragePersist, err)
			}
			// 降级模式下 OSS 对象已上传但 DB 记录缺失，列出 fileKey 便于后续人工/巡检清理孤儿对象。
			orphanKeys := make([]string, 0, len(persistBatch))
			for _, p := range persistBatch {
				orphanKeys = append(orphanKeys, p.FileKey)
			}
			common.SysLog(fmt.Sprintf("oss persist failed (fallback mode): %v; orphan keys=%v", err, orphanKeys))
			return body, false, nil
		}
	}

	changed := false
	for _, r := range results {
		if r.err != nil || r.publicUrl == "" {
			continue
		}
		resp.Data[r.idx].Url = r.publicUrl
		changed = true
	}
	if !changed {
		return body, false, nil
	}
	newBody, err := common.Marshal(&resp)
	if err != nil {
		if !i.cfg.FallbackToUpstream {
			return nil, false, err
		}
		return body, false, nil
	}
	return newBody, true, nil
}

func downloadImage(ctx context.Context, cli *http.Client, rawUrl string) ([]byte, string, error) {
	if err := validateUpstreamURL(rawUrl); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawUrl, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", errors.New("upstream http " + resp.Status)
	}
	// constant.MaxFileDownloadMB 为包级 var，若未初始化则兜底 32MB，避免无限读取。
	maxBytes := int64(constant.MaxFileDownloadMB) * 1024 * 1024
	if maxBytes <= 0 {
		maxBytes = 32 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", fmt.Errorf("image exceeds max %d bytes", maxBytes)
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/png"
	}
	if mt, _, err := mime.ParseMediaType(mimeType); err == nil {
		mimeType = mt
	}
	return data, mimeType, nil
}

// validateUpstreamURL 是包级可替换钩子，便于测试注入裸 httptest.Server。
var validateUpstreamURL = defaultValidateUpstreamURL

// defaultValidateUpstreamURL 防 SSRF：限制 scheme 为 http/https，拒绝指向内网/回环/链路本地 IP 的地址。
func defaultValidateUpstreamURL(rawUrl string) error {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return fmt.Errorf("invalid url: %v", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported url scheme: %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("upstream url missing host")
	}
	// 若 host 直接是 IP 字面量，直接校验；否则解析 DNS。
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		addrs, resolveErr := net.LookupIP(host)
		if resolveErr != nil {
			return fmt.Errorf("dns resolve failed: %v", resolveErr)
		}
		ips = addrs
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("upstream url resolves to disallowed ip: %s", ip.String())
		}
	}
	return nil
}

func buildObjectKey(mimeType string) string {
	ext := extForMime(mimeType)
	t := time.Now().UTC()
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	return path.Join("images",
		fmt.Sprintf("%04d", t.Year()),
		fmt.Sprintf("%02d", t.Month()),
		fmt.Sprintf("%02d", t.Day()),
		id+ext)
}

func extForMime(m string) string {
	switch strings.ToLower(m) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".bin"
	}
}
