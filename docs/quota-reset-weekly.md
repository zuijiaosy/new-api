# 额度重置 & 周额度限制功能文档

> 最后更新：2026-04-15
> 涉及文件：`service/quota_reset/`、`model/option.go`、`controller/quota_reset.go`

---

## 一、系统架构概述

本项目由两套数据库协同工作：

| 数据库 | 类型 | 用途 |
|--------|------|------|
| `new-api`（MySQL） | 主库 | token 额度、消费日志（logs）、用户信息 |
| `codexzh`（PostgreSQL） | 业务库 | 订阅用户、支付订单、激活码、套餐定义 |

**关联方式**：`codexzh.users.apiKey` = `new-api.tokens.name`（即 token 的 name 字段存的是用户邮箱）

---

## 二、每日额度重置功能

### 触发方式

- **自动**：`StartQuotaResetScheduler()` 在进程启动时后台运行，每天北京时间 `QuotaResetTime`（默认 `00:01`）触发
- **手动**：管理后台 `/console/setting?tab=quotareset` 页面点击触发，调用 `TryStartQuotaReset()`

### 核心流程

```
StartQuotaResetScheduler()
  └─ ExecuteQuotaReset()
       └─ executeQuotaResetInternal()
            ├─ 1. BatchUpdateNow()（若启用批量落库，先把增量写入DB）
            ├─ 2. GetActiveUsers()（codexzh DB：apiKey IS NOT NULL AND subscriptionEnd > now）
            ├─ 3. 并发处理每个用户（信号量控制，默认3，最大10）
            │    └─ processUser(user, weeklyLimitEnabled)
            │         ├─ IsDayPass() → 天卡用户（订阅时长≤48h）跳过
            │         ├─ calculateTodayQuota() → 计算今日额度
            │         └─ updateTokenRemainQuota() → 写 MySQL token + Redis
            └─ 4. 记录执行日志（内存，最多100条）
```

### 相关配置（存于 new-api options 表）

| 配置Key | 默认值 | 说明 |
|---------|--------|------|
| `QuotaResetEnabled` | `false` | 日额度重置总开关 |
| `QuotaResetTime` | `00:01` | 每日执行时间（北京时间 HH:MM） |
| `QuotaResetConcurrency` | `3` | 并发处理用户数（1~10） |
| `WeeklyQuotaLimitEnabled` | `false` | 周额度限制开关 |
| `WeeklyQuotaMultiplier` | `3` | 周额度倍数（weeklyQuota = dailyQuota × N） |

---

## 三、周额度限制功能

### 核心公式

```
weeklyQuota  = user.dailyQuota × WeeklyQuotaMultiplier（配置项，默认3）
weeklyUsed   = 窗口内总消耗 - Σ加油包排除量
todayQuota   = min(dailyQuota, weeklyQuota - weeklyUsed)
               weeklyUsed >= weeklyQuota → todayQuota = 0
```

> **重要**：不再使用 `codexzh.users.weeklyQuota` 字段，该字段保留但废弃。

---

### 统计窗口起点（considerStart）确定逻辑

三级优先级，全部基于北京时间：

```
1. 查 codexzh.payment_orders：
   WHERE userId=? AND orderType='subscription' AND status='PAID'
   AND paidAt >= 本周一00:00 AND paidAt <= now
   → 取最早的 paidAt（精确时刻）

2. 找不到 → 查 codexzh.activation_codes：
   WHERE userId=? AND status='used'
   AND usedAt >= 本周一00:00 AND usedAt <= now
   → 取最早的 usedAt（精确时刻）

3. 两个都没有 → considerStart = 本周一 00:00（北京时间）
```

**关键说明**：
- 只有 `orderType='subscription'`（到期重购）才触发 considerStart 调整
- `orderType='upgrade'`（套餐升级）**不**触发，因为是同一订阅周期内的升级
- 激活码使用视为"新购套餐"，触发 considerStart 调整
- 旧订单（orderType IS NULL）无法识别为续购，降级处理：不调整 considerStart

---

### 加油包排除逻辑

**识别方式**（两种，均视为加油包）：
- 新订单：`orderType = 'topup'`
- 旧订单降级：`orderType IS NULL AND name LIKE '%加油包%'`（旧订单 name 包含"加油包"字样）

**排除计算（逐笔）**：
```
对每笔加油包订单：
  windowStart = order.paidAt
  windowEnd   = 次日北京时间 00:00（若超过 now 则截断至 now）
  consumed    = SumUsedQuota(email, [windowStart, windowEnd])
  excluded   += min(consumed, creditTokens)  ← creditTokens 来自 param JSON
```

**param JSON 格式**：
```json
{"productType": "topup", "creditUsd": 50, "creditTokens": 25000000}
```
- `creditTokens`：加油包对应的 new-api 额度单位（1 USD ≈ 500,000 quota units）

**降级原则**（保护平台权益）**：
> 旧订单 orderType=NULL 且 name 不含"加油包"的，无法识别 → 不给用户任何排除，直接按全量消耗计入周额度。

---

## 四、codexzh PostgreSQL 关键表结构

### `users`
| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | bigint | 主键 |
| `email` | text | 用户邮箱，关联 new-api token.name |
| `apiKey` | text | API密钥，不为空表示已激活 |
| `subscriptionStart` | timestamp | 当前订阅开始时间 |
| `subscriptionEnd` | timestamp | 订阅到期时间 |
| `dailyQuota` | bigint | 日额度（new-api quota 单位），默认 25000000 |
| `weeklyQuota` | bigint | **已废弃**，不再用于计算，默认 90000000 |
| `currentPlanName` | text | 当前套餐名称 |

### `payment_orders`
| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | bigint | 主键 |
| `userId` | bigint | 关联 users.id |
| `name` | text | 订单名称（如"标准月套餐"、"加油包"） |
| `status` | text | `PAID` / `FAILED` / `EXPIRED` / `REFUNDED` |
| `orderType` | text | **重要**，见下方枚举 |
| `planId` | bigint | 关联 pricing_plans.id，加油包为 NULL |
| `paidAt` | timestamp | 支付完成时间 |
| `param` | text | JSON，加油包含 `creditTokens` 字段 |
| `commissionUsed` | numeric | 本单使用的佣金抵扣金额 |

**orderType 枚举**：
| 值 | 含义 | 备注 |
|----|------|------|
| `subscription` | 到期重购/新购套餐 | 触发 considerStart 调整 |
| `upgrade` | 套餐升级（差价支付） | **不**触发 considerStart 调整 |
| `topup` | 加油包购买 | 触发加油包排除 |
| `invoice_fee` | 发票手续费 | 忽略 |
| `trial` | 管理员创建试用 | 触发 considerStart 调整 |
| `NULL` | 旧订单（无类型） | 降级：仅 name LIKE '%加油包%' 时才排除 |

> **注意**：旧订单 orderType 多为 NULL，新订单（2026-04 之后创建）才有正确的 orderType。

### `activation_codes`
| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | bigint | 主键 |
| `code` | text | 激活码 |
| `status` | text | `unused` / `used` |
| `userId` | bigint | 使用者 ID |
| `usedAt` | timestamp | 使用时间 |
| `planId` | bigint | 关联套餐 |
| `dailyQuota` | bigint | 激活码对应日额度 |
| `duration` | int | 有效天数 |

### `pricing_plans`（套餐定义）
| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | bigint | 主键 |
| `name` | text | 套餐标识符（如 `monthly_standard`） |
| `dailyQuota` | bigint | 日额度 |
| `monthlyPrice` | numeric | 月价格（人民币） |
| `duration` | int | 有效天数，默认 30 |

> `pricing_plans` 没有 `weeklyQuota` 字段，周额度完全由 `WeeklyQuotaMultiplier` 动态计算。

---

## 五、代码文件索引

```
service/quota_reset/
  codexzh_db.go       — CodexzhDB 初始化（PostgreSQL 连接）
  codexzh_user.go     — CodexzhUser 结构体 + 新增：
                          CodexzhPaymentOrder（payment_orders 映射）
                          CodexzhActivationCode（activation_codes 映射）
                          GetSubscriptionOrdersThisWeek()
                          GetTopUpOrdersInWindow()
                          GetActivationCodesThisWeek()
                          ParseParamCreditTokens()
  quota_reset.go      — 主逻辑：
                          StartQuotaResetScheduler() — 定时调度
                          ExecuteQuotaReset() / TryStartQuotaReset() — 执行入口
                          calculateTodayQuota() — 今日额度（含周限制逻辑）
                          getWeeklyUsedQuota() — 周已用额度
                          getWeeklyConsiderStart() — 统计起点
                          getWeeklyTopUpExcludedQuota() — 加油包排除
                          GetWeeklyQuotaMultiplier() — 读配置
                          updateTokenRemainQuota() — 写 MySQL + Redis
  quota_reset_test.go — 单元测试（内存 SQLite）
controller/quota_reset.go — HTTP 接口（手动触发、查看日志、查看状态）
model/option.go           — 配置默认值（QuotaReset* 系列）
```

---

## 六、天卡用户特殊处理

订阅时长 ≤ 48 小时的用户视为"天卡"，**跳过每日额度重置**（`IsDayPass()` 返回 true）。

原因：天卡是按天购买的日套餐，其 `remain_quota` 在购买时已一次性写入，不应被日重置覆盖。

---

## 七、token 更新与缓存同步

`updateTokenRemainQuota(email, quota)`：
1. 按 `token.name = email` 查 MySQL 最新 token
2. 更新 `remain_quota`
3. 状态联动：`quota=0` → `status=Exhausted`；原来是 Exhausted 且新 quota>0 → `status=Enabled`
4. 若 Redis 开启，同步更新 `token:{hmac_key}` 的 `remain_quota` 和 `Status` 字段

---

## 八、已知遗留问题

1. **`users.weeklyQuota` 字段废弃但未删除**：字段仍在 DB，但计算逻辑已不使用它，可在合适时机用 migration 删除。
2. **旧订单降级**：orderType=NULL 的旧订单无法区分升级/续购，统一不调整 considerStart，对历史数据可能略有偏差，但优先保障平台利益。
3. **`codexzh.users.weeklyQuota` 默认值 90M**：历史遗留，与当前 `dailyQuota×3` 计算结果有时不符，但已不影响逻辑。
