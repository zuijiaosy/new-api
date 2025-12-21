package quota_reset

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

// 北京时间时区 (UTC+8)
var beijingLocation = time.FixedZone("CST", 8*3600)

// QuotaResetLog 额度重置执行日志
type QuotaResetLog struct {
	ExecutedAt     time.Time `json:"executed_at"`
	TotalUsers     int       `json:"total_users"`
	SuccessCount   int       `json:"success_count"`
	FailedCount    int       `json:"failed_count"`
	SkippedDayCard int       `json:"skipped_day_card"`
	Duration       string    `json:"duration"`
	ErrorMessages  []string  `json:"error_messages,omitempty"`
}

// 最近的执行日志（内存存储，最多保留 100 条）
var (
	quotaResetLogs   []QuotaResetLog
	quotaResetLogsMu sync.RWMutex
	maxLogCount      = 100
)

// 是否正在执行
var isRunning int32

// StartQuotaResetScheduler 启动额度重置定时任务
// 每天在指定时间执行（默认 00:01 北京时间）
func StartQuotaResetScheduler() {
	if !IsCodexzhDBConnected() {
		common.SysLog("codexzh 数据库未连接，无法启动额度重置任务")
		return
	}

	common.SysLog("额度重置定时任务已启动")

	for {
		// 从配置中获取执行时间
		resetTime := GetQuotaResetTime()

		// 计算下次执行时间
		nextRun := calculateNextRunTime(resetTime)
		sleepDuration := time.Until(nextRun)

		common.SysLog(fmt.Sprintf("额度重置任务将在 %s (北京时间) 执行，距离执行还有 %v",
			nextRun.In(beijingLocation).Format("2006-01-02 15:04:05"), sleepDuration))

		time.Sleep(sleepDuration)

		// 检查是否启用
		if !IsQuotaResetEnabled() {
			common.SysLog("额度重置功能已禁用，跳过本次执行")
			continue
		}

		// 执行额度重置
		ExecuteQuotaReset()
	}
}

// GetQuotaResetTime 从配置中获取重置时间
func GetQuotaResetTime() string {
	common.OptionMapRWMutex.RLock()
	resetTime, ok := common.OptionMap["QuotaResetTime"]
	common.OptionMapRWMutex.RUnlock()
	if !ok || resetTime == "" {
		return "00:01"
	}
	return resetTime
}

// IsQuotaResetEnabled 检查是否启用额度重置
func IsQuotaResetEnabled() bool {
	common.OptionMapRWMutex.RLock()
	enabled, ok := common.OptionMap["QuotaResetEnabled"]
	common.OptionMapRWMutex.RUnlock()
	if !ok {
		return false
	}
	return enabled == "true"
}

// GetQuotaResetConcurrency 获取并发数配置
func GetQuotaResetConcurrency() int {
	common.OptionMapRWMutex.RLock()
	concurrency, ok := common.OptionMap["QuotaResetConcurrency"]
	common.OptionMapRWMutex.RUnlock()
	if !ok || concurrency == "" {
		return 3
	}
	val, err := strconv.Atoi(concurrency)
	if err != nil || val < 1 {
		return 3
	}
	if val > 10 {
		return 10
	}
	return val
}

// IsWeeklyQuotaLimitEnabled 检查是否启用周额度限制
func IsWeeklyQuotaLimitEnabled() bool {
	common.OptionMapRWMutex.RLock()
	enabled, ok := common.OptionMap["WeeklyQuotaLimitEnabled"]
	common.OptionMapRWMutex.RUnlock()
	if !ok {
		return false
	}
	return enabled == "true"
}

// calculateNextRunTime 计算下次执行时间（北京时间）
func calculateNextRunTime(timeStr string) time.Time {
	now := time.Now().In(beijingLocation)

	// 解析时间字符串（格式：HH:MM）
	hour, minute := 0, 1
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		common.SysLog("解析重置时间失败，使用默认时间 00:01")
		hour, minute = 0, 1
	}

	// 设置目标时间为今天（北京时间）
	nextRun := time.Date(now.Year(), now.Month(), now.Day(),
		hour, minute, 0, 0, beijingLocation)

	// 如果今天的执行时间已过，设置为明天
	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}

// ExecuteQuotaReset 执行额度重置（用于定时任务调用）
// 返回执行日志
func ExecuteQuotaReset() (logEntry *QuotaResetLog) {
	// 防止并发执行
	if !atomic.CompareAndSwapInt32(&isRunning, 0, 1) {
		common.SysLog("额度重置任务正在执行中，跳过本次触发")
		return nil
	}
	defer atomic.StoreInt32(&isRunning, 0)

	return executeQuotaResetInternal()
}

// executeQuotaResetInternal 执行额度重置的内部实现
// 调用方需要自行处理 isRunning 锁
func executeQuotaResetInternal() (logEntry *QuotaResetLog) {
	startTime := time.Now()
	common.SysLog("开始执行额度重置任务...")

	logEntry = &QuotaResetLog{
		ExecutedAt:    startTime.In(beijingLocation),
		ErrorMessages: make([]string, 0),
	}

	// 避免定时任务/手动触发因 panic 直接导致进程崩溃
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("额度重置任务发生 panic: %v", r)
			common.SysLog(errMsg)
			logEntry.ErrorMessages = append(logEntry.ErrorMessages, errMsg)
			logEntry.Duration = time.Since(startTime).String()
			addLog(logEntry)
		}
	}()

	// 如果启用了批量落库，先把历史增量落库，避免"先重置后补写昨日增量"导致额度错乱
	if common.BatchUpdateEnabled {
		model.BatchUpdateNow()
	}

	// 1. 检查数据库连接
	if !IsCodexzhDBConnected() {
		errMsg := "codexzh 数据库未连接"
		common.SysLog(errMsg)
		logEntry.ErrorMessages = append(logEntry.ErrorMessages, errMsg)
		addLog(logEntry)
		return logEntry
	}

	// 2. 获取所有活跃用户
	users, err := GetActiveUsers()
	if err != nil {
		errMsg := "获取活跃用户失败: " + err.Error()
		common.SysLog(errMsg)
		logEntry.ErrorMessages = append(logEntry.ErrorMessages, errMsg)
		addLog(logEntry)
		return logEntry
	}

	logEntry.TotalUsers = len(users)
	common.SysLog(fmt.Sprintf("找到 %d 个活跃用户", len(users)))

	// 3. 获取周额度限制开关状态
	weeklyLimitEnabled := IsWeeklyQuotaLimitEnabled()

	// 4. 获取并发数
	concurrency := GetQuotaResetConcurrency()
	common.SysLog(fmt.Sprintf("使用并发数: %d", concurrency))

	// 5. 使用信号量控制并发
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	var successCount, failedCount, skippedDayCard int32
	var errorMsgsMu sync.Mutex

	for _, user := range users {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(u CodexzhUser) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 重要：子 goroutine 的 panic 不会被上层 recover 捕获，必须在这里兜底，避免进程直接崩溃
			var result processUserResult
			func() {
				defer func() {
					if r := recover(); r != nil {
						result = processUserResult{
							Status: "failed",
							Error:  fmt.Sprintf("用户 %s: 处理发生 panic: %v", u.MaskEmail(), r),
						}
						common.SysLog(fmt.Sprintf("处理用户 %s 发生 panic: %v", u.MaskEmail(), r))
					}
				}()
				result = processUser(&u, weeklyLimitEnabled)
			}()

			switch result.Status {
			case "success":
				atomic.AddInt32(&successCount, 1)
			case "skipped_day_card":
				atomic.AddInt32(&skippedDayCard, 1)
			case "failed":
				atomic.AddInt32(&failedCount, 1)
				if result.Error != "" {
					errorMsgsMu.Lock()
					if len(logEntry.ErrorMessages) < 50 { // 最多记录 50 条错误
						logEntry.ErrorMessages = append(logEntry.ErrorMessages, result.Error)
					}
					errorMsgsMu.Unlock()
				}
			}
		}(user)
	}

	wg.Wait()

	// 6. 记录结果
	logEntry.SuccessCount = int(successCount)
	logEntry.FailedCount = int(failedCount)
	logEntry.SkippedDayCard = int(skippedDayCard)
	logEntry.Duration = time.Since(startTime).String()

	common.SysLog(fmt.Sprintf(
		"额度重置任务完成: 总用户=%d, 成功=%d, 失败=%d, 跳过天卡=%d, 耗时=%s",
		logEntry.TotalUsers, logEntry.SuccessCount, logEntry.FailedCount,
		logEntry.SkippedDayCard, logEntry.Duration))

	addLog(logEntry)
	return logEntry
}

// processUserResult 处理单个用户的结果
type processUserResult struct {
	Status string // success, skipped_day_card, failed
	Error  string
}

// processUser 处理单个用户的额度重置
// weeklyLimitEnabled: 周额度限制开关
func processUser(user *CodexzhUser, weeklyLimitEnabled bool) processUserResult {
	result := processUserResult{}

	// 1. 跳过天卡用户（不参与每日额度重置）
	if user.IsDayPass() {
		result.Status = "skipped_day_card"
		return result
	}

	// 2. 计算今日可分配额度（根据开关决定是否考虑周额度）
	todayQuota := calculateTodayQuota(user, weeklyLimitEnabled)

	// 3. 更新 new-api 的 token.remain_quota
	err := updateTokenRemainQuota(user.Email, todayQuota)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("用户 %s: %v", user.MaskEmail(), err)
		common.SysLog(fmt.Sprintf("更新用户 %s 的 token 额度失败: %v", user.MaskEmail(), err))
		return result
	}

	result.Status = "success"
	return result
}

// calculateTodayQuota 计算今日可分配额度
// weeklyLimitEnabled: 周额度限制开关
// 逻辑：
// - 如果周额度限制未启用，直接使用 dailyQuota
// - 如果 weeklyQuota 为 NULL 或 0，不限制，直接使用 dailyQuota
// - 否则：min(dailyQuota, weeklyQuota - weeklyUsed)
// - 如果周额度已用尽，返回 0
func calculateTodayQuota(user *CodexzhUser, weeklyLimitEnabled bool) int64 {
	dailyQuota := user.DailyQuota
	if dailyQuota < 0 {
		return 0
	}

	// 如果周额度限制未启用，直接使用日额度
	if !weeklyLimitEnabled {
		return dailyQuota
	}

	// 如果没有设置周额度限制，直接使用日额度
	if !user.HasWeeklyQuotaLimit() {
		return dailyQuota
	}

	// 查询本周实际使用量（通过 token_name = email），每天上限为 dailyQuota
	weeklyUsed := getWeeklyUsedQuota(user.Email, dailyQuota)
	weeklyRemain := *user.WeeklyQuota - weeklyUsed

	// 周额度已用尽
	if weeklyRemain <= 0 {
		return 0
	}

	// 返回 min(dailyQuota, weeklyRemain)
	if dailyQuota < weeklyRemain {
		return dailyQuota
	}
	return weeklyRemain
}

// getWeeklyUsedQuota 查询用户本周计入周额度的已用额度
// 规则：每天消耗最多计入 dailyQuota，超出部分视为加油包消耗，不计入周额度
// 通过 token_name = email 关联
func getWeeklyUsedQuota(email string, dailyQuota int64) int64 {
	now := time.Now().In(beijingLocation)
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	daysToMonday := int(weekday) - 1

	var weeklyUsed int64 = 0

	// 遍历本周已过的每一天（周一到今天）
	for i := 0; i <= daysToMonday; i++ {
		// 计算当天的开始和结束时间
		day := now.AddDate(0, 0, -daysToMonday+i)
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, beijingLocation)
		dayEnd := dayStart.Add(24 * time.Hour)

		// 如果是今天，结束时间用当前时间
		if i == daysToMonday {
			dayEnd = now
		}

		// 查询当天消耗
		stat := model.SumUsedQuota(
			model.LogTypeConsume,
			dayStart.Unix(),
			dayEnd.Unix(),
			"",    // modelName
			"",    // username
			email, // tokenName = email
			0,     // channel
			"",    // group
		)

		dayUsed := int64(stat.Quota)

		// 每天最多计入 dailyQuota，超出部分是加油包
		if dayUsed > dailyQuota {
			dayUsed = dailyQuota
		}

		weeklyUsed += dayUsed
	}

	return weeklyUsed
}

// updateTokenRemainQuota 更新 new-api 中对应 token 的 remain_quota
// 通过 token.name = user.email 进行关联
func updateTokenRemainQuota(email string, quota int64) error {
	// 防御性处理：避免负数/溢出写入
	if quota < 0 {
		quota = 0
	}
	maxInt := int64(int(^uint(0) >> 1))
	if quota > maxInt {
		quota = maxInt
	}

	// 通过 name 查找 token
	var token model.Token
	err := model.DB.Where("name = ?", email).Order("id desc").First(&token).Error
	if err != nil {
		// 如果找不到对应的 token，可能用户还没创建 token，跳过
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("未找到对应的 token（按 name=email 关联）")
		}
		return fmt.Errorf("查询 token 失败: %v", err)
	}

	// 准备更新的数据
	updates := map[string]interface{}{
		"remain_quota": int(quota),
	}

	// 仅对非无限额度 token 调整状态，避免把 UnlimitedQuota 的 token 错误标记为 Exhausted
	if !token.UnlimitedQuota {
		// 如果额度为 0，设置状态为 TokenStatusExhausted
		if quota == 0 {
			updates["status"] = common.TokenStatusExhausted
		} else if token.Status == common.TokenStatusExhausted {
			// 如果之前是用尽状态，现在有额度了，恢复为启用状态
			updates["status"] = common.TokenStatusEnabled
		}
	}

	// 执行更新
	err = model.DB.Model(&token).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("更新 token 失败: %v", err)
	}

	// 同步 Redis 缓存（如果存在缓存且有 TTL），否则会出现“DB 已重置但 Redis 仍是旧额度”的不一致
	if common.RedisEnabled && token.Key != "" {
		hmacKey := common.GenerateHMAC(token.Key)
		redisKey := fmt.Sprintf("token:%s", hmacKey)
		_ = common.RedisHSetField(redisKey, constant.TokenFiledRemainQuota, int(quota))
		if status, ok := updates["status"]; ok {
			_ = common.RedisHSetField(redisKey, "Status", status)
		}
	}

	return nil
}

// addLog 添加执行日志到内存
func addLog(log *QuotaResetLog) {
	quotaResetLogsMu.Lock()
	defer quotaResetLogsMu.Unlock()

	// 添加到开头
	quotaResetLogs = append([]QuotaResetLog{*log}, quotaResetLogs...)

	// 保持最多 maxLogCount 条
	if len(quotaResetLogs) > maxLogCount {
		quotaResetLogs = quotaResetLogs[:maxLogCount]
	}
}

// GetQuotaResetLogs 获取最近的执行日志
func GetQuotaResetLogs(limit int) []QuotaResetLog {
	quotaResetLogsMu.RLock()
	defer quotaResetLogsMu.RUnlock()

	if limit <= 0 || limit > len(quotaResetLogs) {
		limit = len(quotaResetLogs)
	}

	result := make([]QuotaResetLog, limit)
	copy(result, quotaResetLogs[:limit])
	return result
}

// IsQuotaResetRunning 检查是否正在执行
func IsQuotaResetRunning() bool {
	return atomic.LoadInt32(&isRunning) == 1
}

// TryStartQuotaReset 尝试启动额度重置任务（用于手动触发）
// 使用原子操作获取锁，成功后异步执行任务
// 返回 true 表示成功启动，false 表示任务已在运行
func TryStartQuotaReset() bool {
	// 原子操作尝试获取锁
	if !atomic.CompareAndSwapInt32(&isRunning, 0, 1) {
		return false
	}

	// 成功获取锁，异步执行任务
	go func() {
		defer atomic.StoreInt32(&isRunning, 0)
		defer func() {
			if r := recover(); r != nil {
				common.SysLog(fmt.Sprintf("手动触发额度重置发生 panic: %v", r))
			}
		}()
		executeQuotaResetInternal()
	}()

	return true
}
