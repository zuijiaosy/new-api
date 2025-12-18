package middleware

import (
	"net/http"
	"regexp"
	"strings"

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

		// 验证是否为有效的 Codex 客户端
		if !isValidCodexClient(c) {
			abortWithOpenAiMessage(c, http.StatusForbidden, "Please use in Codex. Third-party clients are not supported")
			return
		}

		c.Next()
	}
}

// isValidCodexClient 验证请求是否来自有效的 Codex 客户端
func isValidCodexClient(c *gin.Context) bool {
	userAgent := c.GetHeader("User-Agent")

	// 1. 检查 User-Agent 前缀并提取版本号
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

	// 2. 验证版本号格式
	if !isValidVersionFormat(version) {
		return false
	}

	// 3. 检查必需的头部
	conversationId := c.GetHeader("conversation_id")
	sessionId := c.GetHeader("session_id")

	if conversationId == "" || sessionId == "" {
		return false
	}

	return true
}

// isValidVersionFormat 验证版本号格式是否符合 SemVer
func isValidVersionFormat(version string) bool {
	return semverRegex.MatchString(version)
}
