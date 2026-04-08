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
