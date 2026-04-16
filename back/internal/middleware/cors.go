// ============================================================
// 文件：back/internal/middleware/cors.go
// 作用：处理跨域请求（CORS），允许前端（不同来源的浏览器页面）调用后端 API。
//
// 什么是跨域问题（CORS）？
//   浏览器的"同源策略"：一个网页只能请求与自己"同源"（协议+域名+端口完全相同）的接口。
//   本项目开发时：
//   - 前端在 http://localhost:5173（Vite 开发服务器）
//   - 后端在 http://localhost:8000（Go 服务）
//   端口不同就是"跨域"，浏览器默认会拦截这样的请求。
//   通过 CORS 响应头告诉浏览器"我允许这个来源访问"，浏览器才会放行。
//
// 预检请求（OPTIONS）：
//   浏览器在发起"非简单请求"（比如带 Authorization 头的 POST）前，
//   会先发一个 OPTIONS 请求询问服务器"是否允许这个跨域请求"，
//   服务器返回 200 + CORS 头后，浏览器才会发真正的请求。
//   这里直接 AbortWithStatus(200) 快速回应预检，避免浪费资源走到业务逻辑。
//
// Vary: Origin 头的作用：
//   告诉 CDN/缓存"这个响应因 Origin 不同而不同"，
//   防止缓存把 A 域名的 CORS 响应错误地返回给 B 域名。
// ============================================================
package middleware

import (
	"chatapp/back/internal/config"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := config.GetConfig().SecurityConfig.AllowedOrigins

		// 判断请求来源是否在白名单内
		allowed := false
		if len(allowedOrigins) == 0 {
			// 未配置时开发友好：允许所有来源
			allowed = true
		} else {
			for _, o := range allowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
		}

		if allowed && origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(allowedOrigins) == 0 {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Authorization, X-Requested-With, Accept")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Authorization")
		c.Writer.Header().Set("Vary", "Origin")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		// 非白名单来源只拦截跨域预检，普通请求交给业务层的 JWT 鉴权处理
		c.Next()
	}
}
