package chat

import (
	"time"

	"chatapp/back/internal/model"

	"gorm.io/gorm"
)

func ensureSession(db *gorm.DB, km *KafkaMessage) (string, error) {
	// 群
	if isGroup(km.ReceiveId) {
		return ensureGroupSession(db, km.SendId, km.ReceiveId)
	}
	// 单聊
	return ensureDirectSession(db, km.SendId, km.ReceiveId)
}

func ensureDirectSession(db *gorm.DB, sendId, recvId string) (string, error) {
	var sess model.Session
	if err := db.Where("send_id=? AND receive_id=?", sendId, recvId).
		First(&sess).Error; err == nil {
		return sess.Uuid, nil
	}

	sess = model.Session{
		Uuid:      newIDWithPrefix("S"),
		SendId:    sendId,
		ReceiveId: recvId,
		CreatedAt: time.Now(),
	}
	return sess.Uuid, db.Create(&sess).Error
}

func ensureGroupSession(db *gorm.DB, sendId, groupId string) (string, error) {
	var sess model.Session
	if err := db.Where("send_id=? AND receive_id=?", sendId, groupId).
		First(&sess).Error; err == nil {
		return sess.Uuid, nil
	}

	sess = model.Session{
		Uuid:      newIDWithPrefix("S"),
		SendId:    sendId,
		ReceiveId: groupId,
		CreatedAt: time.Now(),
	}
	return sess.Uuid, db.Create(&sess).Error
}
