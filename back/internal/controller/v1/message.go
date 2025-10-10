package v1

import (
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 获取私聊消息
func GetMessageList(c *gin.Context) {
	userId := c.GetString("userId")
	log.Println("userId from JWT:", c.GetString("userId"))

	var form req.GetMessageListRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	messages, err := service.GetMessageList(userId, form.TargetId, form.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}

// 获取群聊消息
func GetGroupMessageList(c *gin.Context) {
	var form req.GetGroupMessageListRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	messages, err := service.GetGroupMessageList(form.GroupId, form.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    messages,
	})

}
