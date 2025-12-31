package middleware

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// RequestLogger 请求日志中间件，用于打印请求头和请求体
// 仅在 DEBUG=true 且 REQUEST_LOG_ENABLED=true 时生效
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只在调试模式且启用请求日志时打印
		if !common.DebugEnabled || !common.RequestLogEnabled {
			c.Next()
			return
		}

		// 获取请求ID
		requestID := c.GetString(common.RequestIdKey)

		// 打印请求基本信息
		common.SysLog(fmt.Sprintf("\n========== [REQUEST] %s ==========", requestID))
		common.SysLog(fmt.Sprintf("Method: %s", c.Request.Method))
		common.SysLog(fmt.Sprintf("Path: %s", c.Request.URL.Path))
		common.SysLog(fmt.Sprintf("Query: %s", c.Request.URL.RawQuery))
		common.SysLog(fmt.Sprintf("RemoteAddr: %s", c.Request.RemoteAddr))
		common.SysLog(fmt.Sprintf("ClientIP: %s", c.ClientIP()))

		// 打印请求头
		common.SysLog("Headers:")
		for name, values := range c.Request.Header {
			// 过滤敏感信息
			if strings.ToLower(name) == "authorization" && len(values) > 0 {
				// 只显示前几个字符
				if len(values[0]) > 20 {
					common.SysLog(fmt.Sprintf("  %s: %s...(masked)", name, values[0][:20]))
				} else {
					common.SysLog(fmt.Sprintf("  %s: ***masked***", name))
				}
			} else {
				common.SysLog(fmt.Sprintf("  %s: %s", name, strings.Join(values, ", ")))
			}
		}

		// 打印请求体
		// if c.Request.Body != nil && c.Request.ContentLength > 0 {
		// 	// 读取请求体
		// 	bodyBytes, err := io.ReadAll(c.Request.Body)
		// 	if err != nil {
		// 		common.SysLog(fmt.Sprintf("Error reading request body: %v", err))
		// 	} else {
		// 		// 恢复请求体，以便后续处理
		// 		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 		// 打印请求体（限制长度避免日志过长）
		// 		bodyStr := string(bodyBytes)
		// 		maxBodyLen := 10000 // 最多打印 10000 字符
		// 		if len(bodyStr) > maxBodyLen {
		// 			common.SysLog(fmt.Sprintf("Body (truncated, total %d bytes):\n%s\n...(truncated)", len(bodyBytes), bodyStr[:maxBodyLen]))
		// 		} else {
		// 			common.SysLog(fmt.Sprintf("Body (%d bytes):\n%s", len(bodyBytes), bodyStr))
		// 		}
		// 	}
		// } else {
		// 	common.SysLog("Body: (empty)")
		// }

		common.SysLog(fmt.Sprintf("========================================\n"))

		c.Next()
	}
}
