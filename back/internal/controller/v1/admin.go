package v1

import (
	"chatapp/back/internal/chat"
	"chatapp/back/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetAllUsers(c *gin.Context) {
	users, err := service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func BanUser(c *gin.Context) {
	userId := c.Param("id")
	status := c.Query("status") // true / false
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 status 参数"})
		return
	}
	ban := status == "true"

	if err := service.BanUser(userId, ban); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msg := "已解封"
	if ban {
		msg = "已封禁"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func GetAllGroups(c *gin.Context) {
	groups, err := service.GetAllGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func AdminDismissGroup(c *gin.Context) {
	groupId := c.Param("id")
	if err := service.AdminDismissGroup(groupId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "群聊已解散"})
}

func GetSystemStats(c *gin.Context) {
	// 从 ChatServer 实时获取在线用户数，避免 service 层直接依赖 chat 包
	chat.ChatServer.Mutex.Lock()
	onlineUsers := int64(len(chat.ChatServer.Clients))
	chat.ChatServer.Mutex.Unlock()

	stats, err := service.GetSystemStats(onlineUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func GetDailyStats(c *gin.Context) {
	days := 7
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil {
			days = n
		}
	}

	data, err := service.GetDailyStats(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func RequireAdmin(c *gin.Context) {
	isAdmin := c.GetInt("isAdmin")
	if isAdmin != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		c.Abort()
		return
	}
}

