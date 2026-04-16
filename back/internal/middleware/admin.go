// ============================================================
// 文件：back/internal/middleware/admin.go
// 作用：管理员权限检查中间件，在 JWT 鉴权通过后，再额外校验用户是否是管理员。
//
// 为什么需要两层验证？
//   第一层（AuthMiddleware）：验证"你是谁"（身份验证，authentication）
//   第二层（AdminOnly）：验证"你有权限做这件事吗"（权限验证，authorization）
//   分两层的好处：普通用户通过第一层但会被第二层挡住，管理员两层都能通过。
//
// 为什么不把管理员状态存在 JWT 里而是每次查数据库？
//   JWT 存了 isAdmin 字段（在 GenerateToken 时写入），但这里仍然查了数据库。
//   原因：如果管理员权限被撤销（is_admin 改为 0），只要 JWT 还未过期，
//   JWT 里的 isAdmin=1 仍然有效。实时查数据库可以确保权限变更立即生效，
//   不需要等 token 过期。
//   代价：每个管理员请求多一次 DB 查询，但管理员接口调用频率极低，可以接受。
// ============================================================
package middleware

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetString("userId")
		if userId == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			c.Abort()
			return
		}
		var u model.UserInfo
		if err := config.GetDB().Select("is_admin").Where("uuid = ?", userId).First(&u).Error; err != nil || u.IsAdmin != 1 {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}
