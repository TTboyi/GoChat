package service

import (
	"chatapp/back/internal/config"
	"chatapp/back/internal/model"
	"errors"
)

func GetAllUsers() ([]model.UserInfo, error) {
	db := config.GetDB()
	var users []model.UserInfo
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func BanUser(userId string, status bool) error {
	db := config.GetDB()
	var user model.UserInfo
	if err := db.Where("uuid = ?", userId).First(&user).Error; err != nil {
		return errors.New("用户不存在")
	}
	if status {
		user.Status = 1 // 1 = 封禁
	} else {
		user.Status = 0 // 0 = 正常
	}
	return db.Save(&user).Error
}

func GetAllGroups() ([]model.GroupInfo, error) {
	db := config.GetDB()
	var groups []model.GroupInfo
	if err := db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func AdminDismissGroup(groupId string) error {
	db := config.GetDB()
	var group model.GroupInfo
	if err := db.Where("uuid = ?", groupId).First(&group).Error; err != nil {
		return errors.New("群聊不存在")
	}
	group.Status = 2 // 解散
	group.Members = []byte("[]")
	group.MemberCnt = 0
	return db.Save(&group).Error
}

func GetSystemStats() (map[string]int64, error) {
	db := config.GetDB()
	var userCnt, groupCnt, messageCnt int64

	db.Model(&model.UserInfo{}).Count(&userCnt)
	db.Model(&model.GroupInfo{}).Count(&groupCnt)
	db.Model(&model.Message{}).Count(&messageCnt)

	return map[string]int64{
		"user_count":    userCnt,
		"group_count":   groupCnt,
		"message_count": messageCnt,
	}, nil
}
