// ============================================================
// 文件：back/internal/middleware/logger.go
// 作用：Gin 框架的 HTTP 请求日志中间件，记录每条请求的关键信息。
//
// RequestLogger 记录哪些信息：
//   - method：HTTP 方法（GET/POST/PUT/DELETE）
//   - path：请求路径（含查询参数）
//   - status：HTTP 状态码（200/404/500 等）
//   - latency_ms：请求处理耗时（毫秒）
//   - ip：客户端 IP 地址
//   - user_id：当前操作的用户ID（由 JWT 中间件注入 Context）
//
// 日志级别策略：
//   - 状态码 2xx/3xx → INFO（正常）
//   - 状态码 4xx → WARN（客户端错误，如参数错误）
//   - 状态码 5xx → ERROR（服务器错误，需要关注）
//   这样在日志系统里可以快速用级别过滤出需要关注的请求。
//
// 为什么在中间件里记录日志而不是在每个 handler 里？
//   中间件天然处于"每个请求都会经过"的位置，只需写一次就能覆盖所有接口，
//   避免了在每个 handler 里重复写日志代码。
// ============================================================
// middleware.logger.go
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger 记录每条 HTTP 请求的关键信息：
// 方法、路径、状态码、耗时(ms)、客户端 IP、操作用户 ID（由 JWT 中间件注入）。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()

		// 尝试从 JWT 中间件注入的 context 里取 userId
		userId, _ := c.Get("userId")
		userIdStr, _ := userId.(string)

		if raw != "" {
			path = path + "?" + raw
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), level, "http",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", clientIP,
			"user_id", userIdStr,
		)
	}
}
