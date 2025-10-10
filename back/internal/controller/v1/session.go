package v1

import (
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 打开会话
func OpenSession(c *gin.Context) {
	userId := c.GetString("userId")
	var form req.OpenSessionRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	session, err := service.OpenSession(userId, &form)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

// 获取用户会话列表
func GetUserSessionList(c *gin.Context) {
	userId := c.GetString("userId")
	list, err := service.GetUserSessionList(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// 获取群聊会话列表
func GetGroupSessionList(c *gin.Context) {
	groupId := c.Query("groupId")
	list, err := service.GetGroupSessionList(groupId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// 删除会话
func DeleteSession(c *gin.Context) {
	userId := c.GetString("userId")
	sessionUuid := c.PostForm("sessionUuid")
	if err := service.DeleteSession(userId, sessionUuid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 检查是否允许打开会话
func CheckOpenSessionAllowed(c *gin.Context) {
	userId := c.GetString("userId")
	targetId := c.Query("targetId")
	allowed, _ := service.CheckOpenSessionAllowed(userId, targetId)
	c.JSON(http.StatusOK, gin.H{"allowed": allowed})
}
