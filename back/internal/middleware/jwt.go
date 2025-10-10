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
