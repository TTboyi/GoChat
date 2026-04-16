// ============================================================
// 文件：back/internal/chat/persist_logic.go
// 作用：把 KafkaMessage 持久化写入 MySQL 数据库，以及确保会话（Session）记录存在。
//
// persistMessage 的两步操作：
//   1. 幂等检查：先查 uuid 是否已存在，存在则直接返回（跳过重复写）
//   2. ensureSession：确保 session 表里有这个会话的记录
//      · 私聊：确保 (sendId, recvId) 这对组合有会话
//      · 群聊：确保 (sendId, groupId) 有会话
//      如果不存在则自动创建，如果已存在则直接返回 sessionId
//   3. 构造 model.Message 并写入数据库
//
// 为什么消息写入时要先"确保会话"？
//   前端的"会话列表"是从 session 表查出来的。
//   如果一个用户第一次给另一个人发消息，session 表里还没有这对用户的记录，
//   所以在写消息的同时，需要自动建立这条会话记录，让双方的会话列表都能看到这个聊天。
//
// ensureSessionForDirect 和 ensureSessionForGroup 的区别：
//   私聊：一对一，(sendId, recvId) 确定唯一会话
//   群聊：每个成员对这个群有自己的会话记录，(sendId, groupId) 确定
// ============================================================

package chat

import (
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/model"

	"gorm.io/gorm"
)

func persistMessage(db *gorm.DB, km *KafkaMessage) error {
	// 幂等：消息已存在直接跳过
	var cnt int64
	db.Model(&model.Message{}).Where("uuid = ?", km.MsgId).Count(&cnt)
	if cnt > 0 {
		return nil
	}

	// 确保会话
	sessId, err := ensureSession(db, km)
	if err != nil {
		return err
	}

	// 若 KafkaMessage 缺少发送者信息，从 DB 补全
	senderName := km.SendName
	senderAvatar := km.SendAvatar
	if senderName == "" || senderAvatar == "" {
		rdb := config.GetRedis()
		_ = rdb // suppress unused
		var u model.UserInfo
		if e := db.Where("uuid = ?", km.SendId).First(&u).Error; e == nil {
			if senderName == "" {
				senderName = u.Nickname
			}
			if senderAvatar == "" {
				senderAvatar = u.Avatar
			}
		}
	}

	msg := model.Message{
		Uuid:       km.MsgId,
		SessionId:  sessId,
		Type:       km.Type,
		Content:    km.Content,
		Url:        km.Url,
		FileName:   km.FileName,
		FileType:   km.FileType,
		FileSize:   km.FileSize,
		SendId:     km.SendId,
		SendName:   senderName,
		SendAvatar: senderAvatar,
		ReceiveId:  km.ReceiveId,
		Status:     1,
		CreatedAt:  time.Unix(km.CreatedAt, 0),
	}

	return db.Create(&msg).Error
}
