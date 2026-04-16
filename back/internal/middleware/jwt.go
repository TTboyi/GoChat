// ============================================================
// 文件：back/internal/middleware/jwt.go
// 作用：Gin 框架的 JWT 鉴权中间件，保护需要登录才能访问的接口。
//
// 什么是中间件（Middleware）？
//   就像安检站：在请求到达真正的业务处理函数之前，中间件先拦截检查，
//   通过了才放行，不通过直接返回错误。
//   多个中间件可以串联成"洋葱模型"：请求从外层中间件依次进入内层业务处理，
//   响应再从内层依次经过外层中间件返回给客户端。
//
// AuthMiddleware 的三步验证：
//   1. 检查 Authorization header 是否存在且格式正确（"Bearer xxxxx"）
//   2. 解析并验证 JWT 的签名和过期时间
//   3. 检查 Redis 黑名单（是否已经登出）
//   全部通过后，把 userId 和 isAdmin 注入到 Gin Context，
//   后续的 handler 可以直接用 c.GetString("userId") 拿到当前用户ID。
//
// c.Abort() 的作用：
//   Gin 的中间件链条上，Abort() 会停止继续往下执行，
//   后面的中间件和 handler 都不再运行。
// ============================================================
// middleware.jwt.go
package middleware

import (
	"net/http"
	"strings"

	"chatapp/back/internal/config"
	"chatapp/back/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 返回一个 Gin 中间件，用于保护需要鉴权的路由
func AuthMiddleware(jwt *utils.ARJWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供有效令牌"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwt.ParseAccessToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		rdb := config.GetRedis()
		if exists, _ := rdb.Exists(c, "jwt:blacklist:"+tokenStr).Result(); exists > 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌已失效"})
			c.Abort()
			return
		}

		// 将 userId 注入到上下文中，后续 handler 可以使用
		c.Set("userId", claims.UserID)
		c.Set("isAdmin", claims.IsAdmin)
		c.Set("claims", claims) // 可选，方便后续直接取整个 claims

		// 继续执行后续 handler
		c.Next()
	}
}
