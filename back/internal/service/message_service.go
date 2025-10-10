package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"log"
)

// 获取双方聊天记录
func GetMessageList(userId, targetId string, limit int) ([]model.Message, error) {
	db := config.GetDB()
	var list []model.Message

	if limit <= 0 {
		limit = 50 // 默认取 50 条
	}

	err := db.Where(
		"(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)",
		userId, targetId, targetId, userId,
	).
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error
	log.Printf("查询消息：userId=%s, targetId=%s, limit=%d", userId, targetId, limit)
	log.Printf("查询结果数量: %d", len(list))
	for _, m := range list {
		log.Printf("消息: %+v\n", m)
	}

	return list, err
}

// 获取群聊消息
func GetGroupMessageList(groupId string, limit int) ([]model.Message, error) {
	db := config.GetDB()
	var list []model.Message

	if limit <= 0 {
		limit = 50
	}

	err := db.Where("receive_id = ?", groupId).
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error

	return list, err
}

func SaveMessage(msg *model.Message) error {
	db := config.GetDB()
	return db.Create(msg).Error
}
