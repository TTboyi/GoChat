package v1

import (
	"chatapp/back/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SendEmailCaptcha(c *gin.Context) {
	var form struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
		return
	}
	if err := service.SendEmailCaptcha(form.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送"})
}

func EmailCaptchaLogin(c *gin.Context) {
	var form struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code"  binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	access, refresh, err := service.EmailCaptchaLogin(form.Email, form.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"token":   access,
		"refresh": refresh,
	})
}
