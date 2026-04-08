package v1

import (
	"chatapp/back/internal/chat"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/service"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取私聊消息
func GetMessageList(c *gin.Context) {
	userId := c.GetString("userId")
	log.Println("userId from JWT:", userId)

	var form req.GetMessageListRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	messages, err := service.GetMessageList(userId, form.TargetId, form.Limit, form.BeforeTime)
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

	messages, err := service.GetGroupMessageList(form.GroupId, form.Limit, form.BeforeTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data":    messages,
	})
}

// RecallMessage 撤回消息
func RecallMessage(c *gin.Context) {
	userId := c.GetString("userId")
	var form struct {
		MsgId string `json:"msgId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := service.RecallMessage(userId, form.MsgId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 通过WS通知相关方消息已撤回
	// 从DB查消息的receiveId来推送通知
	// 构建撤回通知
	payload := map[string]any{
		"action": "msg_recall",
		"msgId":  form.MsgId,
		"sendId": userId,
		"time":   time.Now().Unix(),
	}
	raw, _ := json.Marshal(payload)
	// 推给自己（其他客户端）和接收方（由调用方知道receiveId，这里简化处理）
	// 实际上需要查DB拿receiveId再推
	chat.ChatServer.DeliverToUser(userId, raw)

	c.JSON(http.StatusOK, gin.H{"message": "撤回成功"})
}

// RecallMessageFull 撤回消息并推送给对方（完整版）
func RecallMessageFull(c *gin.Context) {
	userId := c.GetString("userId")
	var form struct {
		MsgId     string `json:"msgId" binding:"required"`
		ReceiveId string `json:"receiveId"` // 接收方（用于推送）
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := service.RecallMessage(userId, form.MsgId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 推送撤回通知给发送方和接收方
	payload := map[string]any{
		"action":    "msg_recall",
		"msgId":     form.MsgId,
		"sendId":    userId,
		"receiveId": form.ReceiveId,
		"time":      time.Now().Unix(),
	}
	raw, _ := json.Marshal(payload)
	chat.ChatServer.DeliverToUser(userId, raw)
	if form.ReceiveId != "" {
		chat.ChatServer.DeliverToUser(form.ReceiveId, raw)
	}

	c.JSON(http.StatusOK, gin.H{"message": "撤回成功"})
}

// MarkMessagesRead 标记消息为已读
func MarkMessagesRead(c *gin.Context) {
	userId := c.GetString("userId")
	var form struct {
		SenderId string `json:"senderId" binding:"required"` // 消息发送方ID
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := service.MarkMessagesRead(userId, form.SenderId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "标记已读失败"})
		return
	}

	// 通知发送方：消息已读
	payload := map[string]any{
		"action":     "msg_read",
		"senderId":   form.SenderId,
		"receiverId": userId,
		"time":       time.Now().Unix(),
	}
	raw, _ := json.Marshal(payload)
	chat.ChatServer.DeliverToUser(form.SenderId, raw)

	c.JSON(http.StatusOK, gin.H{"message": "已读"})
}
