package dao

import (
	"chatapp/back/internal/model"

	"gorm.io/gorm"
)

func GetUserByName(db *gorm.DB, name string) (*model.UserInfo, error) {
	var user model.UserInfo
	err := db.Where("nickname = ?", name).First(&user).Error
	return &user, err
}

func CreateUser(db *gorm.DB, user *model.UserInfo) error {
	return db.Create(user).Error
}
