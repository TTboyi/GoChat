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
