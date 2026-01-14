package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

// SemVer 版本号正则表达式
// 支持: 1.2.3, 1.2.3-alpha, 1.2.3-alpha.1, 1.2.3-beta, 1.2.3-beta.2
var semverRegex = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta)(\.[0-9]+)?)?$`)

// Codex 客户端 User-Agent 前缀
var codexUserAgentPrefixes = []string{
	"codex_cli_rs/",
	"codex_exec/",
	"codex_vscode/",
}

// CodexClientRestriction 中间件用于限制只有 Codex 客户端才能访问 API
func CodexClientRestriction() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用限制，直接放行
		if !setting.CodexClientRestrictionEnabled {
			c.Next()
			return
		}

		// 命中可信任 IP 白名单：跳过客户端校验
		if setting.IsCodexClientRestrictionTrustedIP(c.ClientIP()) {
			c.Next()
			return
		}

		// 验证是否为有效的 Codex 客户端
		if !isValidCodexClient(c) {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("Codex 客户端限制：拒绝请求，user=%d ip=%s ua=%q path=%s", c.GetInt("id"), c.ClientIP(), c.GetHeader("User-Agent"), c.Request.URL.Path))
			abortWithOpenAiMessage(c, http.StatusForbidden, "Please use in Codex. Third-party clients are not supported")
			return
		}

		c.Next()
	}
}

// isValidCodexClient 验证请求是否来自有效的 Codex 客户端
func isValidCodexClient(c *gin.Context) bool {
	userAgent := c.GetHeader("User-Agent")

	// 1. 检查是否为 opencode 客户端
	// opencode 客户端特征: User-Agent 中包含 ai-sdk/openai/*, ai-sdk/provider-utils/*, runtime/bun/*
	if isOpenCodeClient(userAgent) {
		return true
	}

	// 2. 检查 User-Agent 前缀并提取版本号
	var version string
	for _, prefix := range codexUserAgentPrefixes {
		if strings.HasPrefix(userAgent, prefix) {
			// 提取版本号部分（去除前缀后的第一个空格之前的内容）
			remainder := strings.TrimPrefix(userAgent, prefix)
			version = strings.Split(remainder, " ")[0]
			break
		}
	}

	// 如果没有匹配任何前缀
	if version == "" {
		return false
	}

	// 3. 验证版本号格式
	if !isValidVersionFormat(version) {
		return false
	}

	// 4. 检查必需的头部（session_id 必须存在，conversation_id 可选）
	sessionId := c.GetHeader("session_id")

	if sessionId == "" {
		return false
	}

	return true
}

// isOpenCodeClient 检查是否为 opencode 客户端
// opencode 客户端的 User-Agent 格式: ai-sdk/openai/版本 ai-sdk/provider-utils/版本 runtime/bun/版本
func isOpenCodeClient(userAgent string) bool {
	// 检查是否包含这三个特征字符串
	hasOpenAI := strings.Contains(userAgent, "ai-sdk/openai/")
	hasProviderUtils := strings.Contains(userAgent, "ai-sdk/provider-utils/")
	hasRuntimeBun := strings.Contains(userAgent, "runtime/bun/")

	return hasOpenAI && hasProviderUtils && hasRuntimeBun
}

// isValidVersionFormat 验证版本号格式是否符合 SemVer
func isValidVersionFormat(version string) bool {
	return semverRegex.MatchString(version)
}
