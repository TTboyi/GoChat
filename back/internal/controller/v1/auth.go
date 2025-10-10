package v1

import (
	"chatapp/back/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 刷新 token
func RefreshToken(c *gin.Context) {
	var form struct {
		AccessToken  string `json:"access"  binding:"required"`
		RefreshToken string `json:"refresh" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	access, refresh, err := service.RefreshToken(form.AccessToken, form.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "刷新成功",
		"token":   access,
		"refresh": refresh,
	})
}

// 退出登录：把 access 放黑名单
func Logout(c *gin.Context) {
	var form struct {
		AccessToken string `json:"access" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := service.Logout(form.AccessToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "退出失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}
