package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/dto/req"
	"chatapp/back/internal/model"
	"errors"
	"time"

	"github.com/google/uuid"
)

// 打开会话
func OpenSession(userId string, form *req.OpenSessionRequest) (*model.Session, error) {
	db := config.GetDB()

	// 先检查是否已有会话
	var session model.Session
	if err := db.Where("send_id = ? AND receive_id = ?", userId, form.ReceiveId).First(&session).Error; err == nil {
		return &session, nil
	}

	// 创建新会话
	newSession := model.Session{
		Uuid:        "S" + uuid.NewString()[:19],
		SendId:      userId,
		ReceiveId:   form.ReceiveId,
		ReceiveName: form.ReceiveName,
		Avatar:      form.Avatar,
		CreatedAt:   time.Now(),
	}
	if newSession.Avatar == "" {
		newSession.Avatar = "default_avatar.png"
	}

	if err := db.Create(&newSession).Error; err != nil {
		return nil, errors.New("创建会话失败")
	}

	return &newSession, nil
}

// 获取用户的会话列表（只查 send_id）
func GetUserSessionList(userId string) ([]model.Session, error) {
	db := config.GetDB()
	var sessions []model.Session
	if err := db.Where("send_id = ?", userId).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// 获取群聊的会话列表（查 receive_id 为群聊ID）
func GetGroupSessionList(groupId string) ([]model.Session, error) {
	db := config.GetDB()
	var sessions []model.Session
	if err := db.Where("receive_id = ?", groupId).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// 删除会话
func DeleteSession(userId, sessionUuid string) error {
	db := config.GetDB()
	return db.Where("uuid = ? AND send_id = ?", sessionUuid, userId).Delete(&model.Session{}).Error
}

// 检查是否允许打开会话（例如黑名单逻辑）
func CheckOpenSessionAllowed(userId, targetId string) (bool, error) {
	db := config.GetDB()

	var contact model.UserContact
	if err := db.Where("user_id = ? AND contact_id = ?", userId, targetId).First(&contact).Error; err == nil {
		if contact.Status == 1 { // 1 = 黑名单
			return false, nil
		}
		return true, nil
	}

	// 如果没有关系，暂时允许（可扩展为只允许好友）
	return true, nil
}
