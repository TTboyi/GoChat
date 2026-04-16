// ============================================================
// 文件：back/internal/chat/session_logic.go
// 作用：确保 session（会话）记录在数据库中存在，不存在则自动创建。
//       供 persist_logic.go（消息持久化）调用。
//
// 这里的 "Session" 不是指 HTTP Session（HTTP 会话状态），
// 而是指"聊天会话"——即"两个用户之间（或用户与群之间）的一个持续对话记录"。
// 对应前端"左侧聊天列表"里的每一行。
//
// ensureDirectSession 逻辑：
//   先查（sendId, recvId）是否有记录 → 有则返回已有的 uuid
//                                    → 没有则创建新记录
// 这种"先查后创建"的模式在数据库操作中称为 "upsert"（update or insert）。
// 这里用 First + Create 两步来模拟，而不是直接用 UPSERT 语句，
// 是为了兼容更多数据库方言，也让逻辑更清晰。
// ============================================================

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
