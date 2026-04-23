package oss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/oss_setting"
)

// ----- 伪造依赖 -----

type fakeStorage struct {
	mu       sync.Mutex
	puts     int
	putError error
}

func (f *fakeStorage) Put(_ context.Context, key string, _ io.Reader, _ int64, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts++
	if f.putError != nil {
		return "", f.putError
	}
	return "https://cdn.example.com/bucket/" + key, nil
}
func (f *fakeStorage) Delete(context.Context, string) error { return nil }
func (f *fakeStorage) BatchDelete(context.Context, []string) (int, []string, error) {
	return 0, nil, nil
}
func (f *fakeStorage) Ping(context.Context) error { return nil }

type fakeRepo struct {
	created atomic.Int32
	failErr error
}

func (f *fakeRepo) BatchCreate(imgs []model.OssImage) error {
	if f.failErr != nil {
		return f.failErr
	}
	f.created.Add(int32(len(imgs)))
	return nil
}

func newInterceptorFor(t *testing.T, storage Storage, repo OssImageRepo, cfg oss_setting.OssImageSetting) *ImageURLInterceptor {
	t.Helper()
	return &ImageURLInterceptor{storage: storage, repo: repo, cfg: cfg}
}

func TestInterceptDisabled(t *testing.T) {
	in := newInterceptorFor(t, nil, nil, oss_setting.OssImageSetting{Enabled: false})
	body := []byte(`{"data":[{"url":"https://up/x.png"}]}`)
	out, changed, err := in.Intercept(context.Background(), body, &RelayMeta{})
	if err != nil || changed || !bytes.Equal(out, body) {
		t.Fatalf("disabled should passthrough: changed=%v err=%v", changed, err)
	}
}

func TestInterceptB64Passthrough(t *testing.T) {
	in := newInterceptorFor(t, &fakeStorage{}, &fakeRepo{}, enabledCfg())
	body := []byte(`{"data":[{"b64_json":"aGVsbG8="}]}`)
	out, changed, err := in.Intercept(context.Background(), body, &RelayMeta{})
	if err != nil || changed || !bytes.Equal(out, body) {
		t.Fatalf("b64 should passthrough: changed=%v err=%v", changed, err)
	}
}

func TestInterceptSingleSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("pixels"))
	}))
	t.Cleanup(srv.Close)

	stg := &fakeStorage{}
	repo := &fakeRepo{}
	in := newInterceptorFor(t, stg, repo, enabledCfg())
	body := []byte(fmt.Sprintf(`{"data":[{"url":%q}]}`, srv.URL+"/a.png"))
	out, changed, err := in.Intercept(context.Background(), body, &RelayMeta{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !changed {
		t.Fatalf("should be changed")
	}
	if stg.puts != 1 || repo.created.Load() != 1 {
		t.Fatalf("want 1 put/1 create: puts=%d created=%d", stg.puts, repo.created.Load())
	}
	if bytes.Contains(out, []byte(srv.URL)) {
		t.Fatalf("upstream url should be replaced")
	}
	var parsed dto.ImageResponse
	if err := common.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal out: %v", err)
	}
	if len(parsed.Data) != 1 || !strings.HasPrefix(parsed.Data[0].Url, "https://cdn.example.com/") {
		t.Fatalf("replaced url should point to CDN: %+v", parsed.Data)
	}
}

func TestInterceptStrictDownload404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfg := enabledCfg()
	cfg.FallbackToUpstream = false
	in := newInterceptorFor(t, &fakeStorage{}, &fakeRepo{}, cfg)
	body := []byte(fmt.Sprintf(`{"data":[{"url":%q}]}`, srv.URL+"/x.png"))
	_, _, err := in.Intercept(context.Background(), body, &RelayMeta{})
	if err == nil {
		t.Fatalf("strict mode should error")
	}
}

func TestInterceptFallbackDownload404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfg := enabledCfg()
	cfg.FallbackToUpstream = true
	in := newInterceptorFor(t, &fakeStorage{}, &fakeRepo{}, cfg)
	orig := fmt.Sprintf(`{"data":[{"url":%q}]}`, srv.URL+"/x.png")
	out, _, err := in.Intercept(context.Background(), []byte(orig), &RelayMeta{})
	if err != nil {
		t.Fatalf("fallback mode should not error: %v", err)
	}
	if !bytes.Contains(out, []byte(srv.URL)) {
		t.Fatalf("fallback should keep upstream url")
	}
}

func TestInterceptConcurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("pixels"))
	}))
	t.Cleanup(srv.Close)

	stg := &fakeStorage{}
	repo := &fakeRepo{}
	in := newInterceptorFor(t, stg, repo, enabledCfg())
	body := []byte(fmt.Sprintf(
		`{"data":[{"url":%q},{"url":%q},{"url":%q}]}`,
		srv.URL+"/1.png", srv.URL+"/2.png", srv.URL+"/3.png",
	))
	_, changed, err := in.Intercept(context.Background(), body, &RelayMeta{})
	if err != nil || !changed {
		t.Fatalf("unexpected: changed=%v err=%v", changed, err)
	}
	if stg.puts != 3 || repo.created.Load() != 3 {
		t.Fatalf("want 3 put/3 create: puts=%d created=%d", stg.puts, repo.created.Load())
	}
}

func enabledCfg() oss_setting.OssImageSetting {
	return oss_setting.OssImageSetting{
		Enabled:                true,
		FallbackToUpstream:     false,
		Endpoint:               "localhost:9000",
		AccessKey:              "ak",
		SecretKey:              "sk",
		Bucket:                 "b",
		PublicUrlPrefix:        "https://cdn.example.com",
		DownloadTimeoutSeconds: 5,
	}
}
