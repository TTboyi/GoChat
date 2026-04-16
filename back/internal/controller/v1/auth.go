// ============================================================
// 文件：back/internal/controller/v1/auth.go
// 作用：处理 token 刷新和登出请求。
//
// RefreshToken（刷新 token）：
//   当 access token 过期（API 返回 401），前端调用这个接口，
//   用 refresh token 换取一对新的 access + refresh token。
//   前端在 api.ts 的响应拦截器里自动处理了这个流程（无感刷新）。
//
// Logout（登出）：
//   登出并不是真的"删除 token"（JWT 无法被删除），
//   而是把这个 token 放进 Redis 黑名单。
//   之后中间件校验时发现 token 在黑名单里，就拒绝访问。
//   黑名单 key 的 TTL 等于 token 剩余有效期，到期后 Redis 自动删除，不浪费空间。
// ============================================================
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
