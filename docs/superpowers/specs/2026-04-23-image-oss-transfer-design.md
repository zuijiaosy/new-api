# 图片 OSS 转存功能设计稿

- 日期：2026-04-23
- 状态：已通过 brainstorming 审核，待写实施计划
- 作者：brainstorming 会话
- 范围：仅 OpenAI 兼容格式生图接口（`/v1/images/generations` + `/v1/images/edits`）

## 1. 背景与目标

new-api 作为 AI 网关，调用 OpenAI 兼容格式的上游生图渠道时，若上游返回图片 URL，会把该 URL 原样透传给终端用户。这会导致：

1. 上游渠道域名直接暴露给客户，损害 new-api 作为"统一入口"的语义；
2. 上游图片 URL 通常短时效或私有分发，用户缓存/分享后失效；
3. 图片内容无法归入 new-api 的审计、生命周期管理。

目标：在 new-api 内对图片 URL 做一次透明转存，用 MinIO 作为默认后端，返回给用户的 URL 全部走部署方自己的 MinIO 公网域名；同时通过定时任务对过期图片自动清理，避免无限占用空间。

### 非目标（明确排除）

- 不做访问次数统计与基于访问计数的删除。
- 不做图片管理列表页面 / 图片预览 / 按用户限额。
- 不处理 `response_format=b64_json` 的情况（不改变其语义）。
- 不覆盖 Chat Completions 多模态图片输出、独立 Task 型绘图渠道（MidJourney / 即梦 / 阿里万相 / 智谱 4V 等）——但抽象出可扩展的 `ImageURLInterceptor`，后续按需挂入。

## 2. 总体架构

### 2.1 生图请求数据流

```
Client
  │ POST /v1/images/generations
  ▼
Gin Router
  │
  ▼
ImageHelper (relay/image_handler.go)
  │  adaptor.DoRequest → 拿到上游 http.Response
  │
  ▼
adaptor.DoResponse  ──►  OpenaiHandlerWithUsage（现有）
                          1. 读 responseBody
                          2. common.Unmarshal usage
                          3. 【新增】ImageURLInterceptor.Intercept(responseBody, info)
                                 - 若 cfg.Enabled=false / 非 url 响应 / b64 → 原样返回
                                 - 并发：下载上游图片 → MinIO 上传 → 替换 data[].url
                                 - 批量 insert oss_images 记录
                                 - 失败按"严格 / 降级"策略决定
                          4. service.IOCopyBytesGracefully（写回替换后的 body）
                          5. 返回 usage → postConsumeQuota
```

严格模式下拦截失败发生在 `postConsumeQuota` 之前，错误沿 `types.NewError` 路径冒泡，额度自动不扣。

### 2.2 清理任务数据流

```
main.go 启动
  │
  ▼
StartOssImageCleanupTask()
  │  sync.Once 守护
  │  if !common.IsMasterNode → 不启动
  │  gopool.Go(ticker，默认 24h 周期)
  │
  ▼
RunOssImageCleanupOnce(ctx)
  │  atomic.Bool 抢占（失败直接返回）
  │  loop:
  │    expired := model.ListExpiredOssImages(now - retention, batchSize)
  │    if len(expired)==0 → break
  │    keys := collect file_keys
  │    storage.BatchDelete(keys)
  │    model.DeleteOssImagesByIds(ids)
  │  返回 CleanupReport{scanned, deleted, failed, elapsed}
```

### 2.3 手动清理

```
Admin UI "立即执行清理"
  │
  ▼
POST /api/oss/images/cleanup   (Admin 权限)
  │  复用 RunOssImageCleanupOnce
  │  正在运行 → 409 Conflict
  │  完成     → 返回统计
```

### 2.4 失败边界

| 失败点 | 严格模式 | 降级模式 |
|---|---|---|
| 上游图片下载 502/超时 | 整请求 5xx + 额度不扣 | 返回原始 URL，warn 日志 |
| MinIO 上传失败 | 同上 | 同上 |
| DB `oss_images` insert 失败 | 同上 | 同上 |
| 清理任务单批失败 | 记 warn，下次继续 | 同 |

## 3. 后端组件拆分

```
service/oss/
  storage.go          ← Storage 接口（Put/Delete/BatchDelete/Ping）
  storage_minio.go    ← MinIO 实现（基于 minio-go v7 SDK）
  image_interceptor.go← ImageURLInterceptor
  cleanup_task.go     ← 每日定时任务 + 手动触发入口
  errors.go           ← 分类错误

model/
  oss_image.go        ← oss_images 表 GORM model + CRUD

setting/oss_setting/
  oss_setting.go      ← 配置注册（走 config.GlobalConfig）

controller/
  oss_image.go        ← 管理接口：ping、手动清理

router/
  api-router.go 内追加 Admin 权限组路由
```

### 3.1 `Storage` 接口

```go
type Storage interface {
    Put(ctx context.Context, key string, body io.Reader, size int64, mime string) (publicURL string, err error)
    Delete(ctx context.Context, key string) error
    BatchDelete(ctx context.Context, keys []string) (deleted int, failed []string, err error)
    Ping(ctx context.Context) error
}
```

- MinIO 实现走 `github.com/minio/minio-go/v7`
- 单例 + 版本号 `atomic.Uint64`：配置保存成功后 `BumpStorageVersion()` 下一次获取时 rebuild，正在进行的请求沿用旧 client

### 3.2 `ImageURLInterceptor`

```go
type ImageURLInterceptor struct {
    storage Storage
    cfg     *OssImageSetting
    repo    OssImageRepo
}

func (i *ImageURLInterceptor) Intercept(ctx context.Context, body []byte, info *relaycommon.RelayInfo) ([]byte, bool, error)
```

行为：

1. `cfg.Enabled=false` → `return body, false, nil`
2. 按 `dto.ImageResponse` 解析；解析失败或无 url 字段 → 原样返回
3. 过滤出 `Url != "" && B64Json == ""` 的条目
4. 无可转存项 → 原样返回
5. `errgroup` 并发下载 + 上传；单图超时 `cfg.DownloadTimeoutSeconds`
6. 全部成功：批量 insert `oss_images`，重新 `common.Marshal` 生成新 body
7. 任一失败：
    - 严格模式 → 直接返回 error（上层回退）
    - 降级模式 → 保留原 URL，该行不 insert

**拦截点**：放在 `relay/channel/openai/relay-openai.go::OpenaiHandlerWithUsage` 内，**仅当 `info.RelayMode` 为 `RelayModeImagesGenerations` 或 `RelayModeImagesEdits` 时生效**，不污染 chat completion 主路径；且插入在 `service.IOCopyBytesGracefully` 之前。

### 3.3 清理任务

```go
func StartOssImageCleanupTask()                               // sync.Once + IsMasterNode + ticker
func RunOssImageCleanupOnce(ctx context.Context) (CleanupReport, error)  // 定时 & 手动共用
```

- `atomic.Bool` 互斥；并发调用直接返回 `ErrCleanupInProgress`
- 单次分批处理直到拉空；MinIO 不存在的 key 视作删除成功继续推进
- `CleanupReport{Scanned, Deleted, Failed, ElapsedMs}`

## 4. 数据模型

```go
// model/oss_image.go
type OssImage struct {
    Id         int64  `json:"id" gorm:"primaryKey;autoIncrement"`
    FileKey    string `json:"file_key" gorm:"type:varchar(512);uniqueIndex;not null"`
    PublicUrl  string `json:"public_url" gorm:"type:varchar(1024);not null"`
    MimeType   string `json:"mime_type" gorm:"type:varchar(64)"`
    SizeBytes  int64  `json:"size_bytes"`

    UserId     int    `json:"user_id" gorm:"index"`
    ChannelId  int    `json:"channel_id" gorm:"index"`
    TokenId    int    `json:"token_id"`
    ModelName  string `json:"model_name" gorm:"type:varchar(128)"`

    UpstreamUrl string `json:"upstream_url" gorm:"type:varchar(2048)"`

    CreatedAt int64 `json:"created_at" gorm:"autoCreateTime;index"`
}
```

### CRUD

```go
func CreateOssImage(img *OssImage) error
func BatchCreateOssImages(imgs []OssImage) error
func ListExpiredOssImages(beforeUnix int64, limit int) ([]OssImage, error)
func DeleteOssImagesByIds(ids []int64) (int64, error)
func GetOssImageById(id int64) (*OssImage, error)
```

### 跨库兼容（Rule 2）

- GORM 自动处理主键 / 建表差异
- 时间戳用 `int64` 秒级 + `autoCreateTime`，与项目 `logs` 等表保持一致
- 不使用 JSONB / `ALTER COLUMN` / MySQL 或 PostgreSQL 专有语法
- 迁移：`model/main.go` 的 `AutoMigrate` 列表追加 `&OssImage{}`

### 文件命名规则

- Object key：`images/YYYY/MM/DD/{uuid}.{ext}`
- 扩展名推断：`image/png → .png`，`image/webp → .webp`，`image/jpeg → .jpg`，其它走 `.bin`
- Public URL 拼接：`{PublicUrlPrefix}/{Bucket}/{Key}`

## 5. 配置系统

### 5.1 配置结构

```go
// setting/oss_setting/oss_setting.go
type OssImageSetting struct {
    Enabled              bool   `json:"enabled"`
    FallbackToUpstream   bool   `json:"fallback_to_upstream"`

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
```

### 5.2 默认值

| 字段 | 默认值 |
|---|---|
| Enabled | false |
| FallbackToUpstream | false |
| Bucket | `new-api-images` |
| Region | `us-east-1` |
| UseSSL | false |
| UsePathStyle | true |
| RetentionHours | 24 |
| DownloadTimeoutSeconds | 30 |
| CleanupIntervalHours | 24 |
| CleanupBatchSize | 500 |

### 5.3 注册与持久化

- `init()` 内 `config.GlobalConfig.Register("oss_image_setting", &ossImageSetting)`
- 前端走项目现有 `PUT /api/setting/` 整包保存路径（body `{key, value}`）
- 保存 handler 回调内调用 `oss.BumpStorageVersion()` 触发下次请求重建 client

### 5.4 敏感字段处理

- `SecretKey` 在 GET 返回时脱敏为 `****<后4位>`
- 前端若未修改（保留脱敏占位），提交时 `SecretKey` 置空串；后端"空串 = 不修改"

### 5.5 启用自检

`ImageURLInterceptor.Intercept` 入口快速校验：

- `Enabled=false` → 直通
- `Endpoint / AccessKey / SecretKey / Bucket / PublicUrlPrefix` 任一为空 → 记一次 warn，直通（不阻塞生图主流程）

## 6. 前端页面

### 6.1 独立菜单 "图片 OSS"

- 菜单挂入 `web/src/pages/Setting/index.jsx` 侧边栏（与 Performance、Rate Limit 同级）
- 新建 `web/src/pages/Setting/ImageOss/SettingsImageOss.jsx`
- i18n key 使用中文原文，提交后经 `bun run i18n:sync` 同步其它语种

### 6.2 表单区块

- **基础设置**：启用开关、失败回退开关
- **MinIO 连接**：Endpoint、AccessKey、SecretKey（password 类型）、Bucket、Region、UseSSL、UsePathStyle、PublicUrlPrefix、"连接测试"按钮
- **生命周期**：RetentionHours、DownloadTimeoutSeconds、CleanupIntervalHours、CleanupBatchSize、"立即执行清理"按钮
- **保存配置**按钮

### 6.3 按钮行为

- 连接测试：`POST /api/oss/images/ping`，传入当前表单值（临时构造 Storage 实例测试，不依赖已保存配置）
- 立即执行清理：`POST /api/oss/images/cleanup`，按钮 loading 态，完成后 toast 显示统计

### 6.4 UX 细节

- 功能未启用时页面顶部一条信息条：`当前功能未启用，开启开关后配置才会生效`
- `SecretKey` 输入框默认展示 `****<后4位>`（若后端已有配置）

## 7. 管理接口

| 方法 | 路径 | 权限 | 行为 | 返回 |
|---|---|---|---|---|
| POST | `/api/oss/images/ping` | Admin | 用入参构造临时 Storage → Put 一个 16B 小文件 → Delete | `{success, latency_ms, message}` |
| POST | `/api/oss/images/cleanup` | Admin | 复用 `RunOssImageCleanupOnce` | `{scanned, deleted, failed, elapsed_ms}` 或 409 |

路由挂在 `router/api-router.go` 的 Admin 分组下，与项目现有 controller 命名风格对齐。

## 8. 错误处理与可观测性

### 8.1 错误类型

```go
var (
    ErrStorageNotConfigured = errors.New("oss storage not configured")
    ErrUpstreamDownload     = errors.New("failed to download upstream image")
    ErrStorageUpload        = errors.New("failed to upload to storage")
    ErrStoragePersist       = errors.New("failed to persist oss image record")
    ErrCleanupInProgress    = errors.New("cleanup already in progress")
)
```

### 8.2 日志

- 拦截器每次运行打结构化 info：`user_id / channel_id / model / url_count / upload_ms / total_ms`
- 降级或严格失败打 warn：失败类型 + 原始 URL
- 清理任务每次结束打 info：`oss cleanup done: scanned=X deleted=Y failed=Z elapsed=Nms`

### 8.3 并发与资源

- 单请求多图并发：`errgroup`，不限并发（dalle 最大 10 张）
- 下载 `http.Client{ Timeout: cfg.DownloadTimeoutSeconds }` 独立实例
- 下载字节上限沿用 `constant.MaxFileDownloadMB`
- 清理任务：串行分批
- 集群门控：`IsMasterNode`
- 定时 / 手动互斥：`atomic.Bool`
- `CleanupIntervalHours` 的变更在进程重启后生效；运行中修改只影响下一次启动，这是可接受的简化（不需要为此设计可取消 ticker 的运行时切换）。如需立即生效可通过手动触发按钮补位。

## 9. 测试策略

### 单元测试

1. `service/oss/image_interceptor_test.go`
    - 伪造 `Storage` 和 `OssImageRepo`
    - `httptest.Server` mock 上游图片
    - 场景：单 URL 成功、多 URL 并发成功、下载 404 + 严格、下载 404 + 降级、b64 直通、关闭直通
2. `service/oss/storage_minio_test.go`
    - `TEST_MINIO_ENDPOINT` 为空时 `t.Skip`
    - 非 skip 时做真实 Put/Get/Delete/BatchDelete
3. `model/oss_image_test.go`
    - `ListExpiredOssImages` 边界（恰好等于 threshold 不删）
    - `DeleteOssImagesByIds` 空数组、部分 id 不存在
4. `service/oss/cleanup_task_test.go`
    - 分批逻辑、`atomic.Bool` 互斥

### 集成测试（可选）

- `docker-compose.test.yml` 起 MinIO；跑一次完整"生图 → 替换 URL → 清理"链路

### 手工 QA 清单

- [ ] 启用 / 关闭两态下 `/v1/images/generations` 正常返回
- [ ] 启用后响应 URL 已是 `PublicUrlPrefix` 域名
- [ ] MinIO 离线：严格模式报错；降级模式返回原 URL
- [ ] 配置修改后不用重启服务
- [ ] 手动清理按钮返回统计
- [ ] 24h 后定时任务自动清理（可设 `RetentionHours=1` + 手动触发验证）
- [ ] SQLite / MySQL / PostgreSQL 三库建表、清理正常

## 10. 项目规则自查（CLAUDE.md）

- ✅ Rule 1：所有 JSON 走 `common.Marshal/Unmarshal`
- ✅ Rule 2：纯 GORM；`int64` 时间戳；无 JSONB / `ALTER COLUMN` / 专有语法；三库兼容
- ✅ Rule 3：前端用 `bun`
- ✅ Rule 4：不涉及 StreamOptions
- ✅ Rule 5：不触碰 new-api / QuantumNous 品牌标识
- ✅ Rule 6：不修改请求 DTO
- ✅ Rule 7：**纯 additive**，唯一对既有代码的改动是在 `OpenaiHandlerWithUsage` 中插入一行可选拦截调用，最小化合并冲突

## 11. 遗留与未决

- 暂不提供图片管理列表页面
- 暂不覆盖 Task 型绘图、Chat Completions 多模态图片输出（抽象已预留扩展点）
- 连接测试接口的临时 Storage 是否需要跳过部分 TLS 校验，留到实现时对齐
- SecretKey 脱敏格式 `****<后4位>` 需在实现时与项目中类似字段（如 Stripe/Paypal 相关）保持一致

## 12. 实施路线概览（详细计划另文）

1. 后端基础：`service/oss` + `model/oss_image` + `setting/oss_setting`
2. 拦截点接入 `OpenaiHandlerWithUsage`
3. 管理接口 + 路由
4. 定时任务接入 `main.go`
5. 前端设置页
6. 单元测试 + 手工 QA

每步独立可验证，可拆成若干 PR。
