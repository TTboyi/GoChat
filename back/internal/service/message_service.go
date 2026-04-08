package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"context"
	"errors"
	"time"
)

// 获取双方聊天记录（支持分页：beforeTime为Unix时间戳，0表示最新）
func GetMessageList(userId, targetId string, limit int, beforeTime int64) ([]model.Message, error) {
	// 跳过Redis直接查DB，以支持分页和撤回状态
	db := config.GetDB()
	var list []model.Message

	if limit <= 0 {
		limit = 50
	}

	q := db.Where(
		"(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)",
		userId, targetId, targetId, userId,
	)
	if beforeTime > 0 {
		q = q.Where("UNIX_TIMESTAMP(created_at) < ?", beforeTime)
	}
	err := q.Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// 获取群聊消息（支持分页）
func GetGroupMessageList(groupId string, limit int, beforeTime int64) ([]model.Message, error) {
	db := config.GetDB()
	var list []model.Message

	if limit <= 0 {
		limit = 50
	}

	q := db.Where("receive_id = ?", groupId)
	if beforeTime > 0 {
		q = q.Where("UNIX_TIMESTAMP(created_at) < ?", beforeTime)
	}
	err := q.Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

func SaveMessage(msg *model.Message) error {
	db := config.GetDB()
	return db.Create(msg).Error
}

// RecallMessage 撤回消息（10分钟内）
func RecallMessage(senderId, msgId string) error {
	db := config.GetDB()
	var msg model.Message
	if err := db.Where("uuid = ?", msgId).First(&msg).Error; err != nil {
		return errors.New("消息不存在")
	}
	if msg.SendId != senderId {
		return errors.New("只能撤回自己的消息")
	}
	if time.Since(msg.CreatedAt) > 10*time.Minute {
		return errors.New("超过10分钟，无法撤回")
	}
	return db.Model(&msg).Updates(map[string]interface{}{
		"is_recalled": 1,
		"content":     "",
	}).Error
}

// MarkMessagesRead 标记某会话消息为已读（接收方调用），返回被标记的最早消息发送者ID
func MarkMessagesRead(receiverId, senderId string) error {
	db := config.GetDB()
	now := time.Now()
	return db.Model(&model.Message{}).
		Where("receive_id = ? AND send_id = ? AND read_at IS NULL AND is_recalled = 0", receiverId, senderId).
		Update("read_at", now).Error
}

// GetUnreadCount 获取某会话未读消息数
func GetUnreadCount(ctx context.Context, receiverId, senderId string) (int64, error) {
	db := config.GetDB()
	var count int64
	err := db.Model(&model.Message{}).
		Where("receive_id = ? AND send_id = ? AND read_at IS NULL AND is_recalled = 0", receiverId, senderId).
		Count(&count).Error
	return count, err
}
