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
