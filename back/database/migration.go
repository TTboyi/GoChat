package database

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"chatapp/back/internal/model"
)

var DB *gorm.DB

func InitDB() {
	dsn := "root:password@tcp(127.0.0.1:3306)/chat_app?charset=utf8mb4&parseTime=True&loc=Local"
	var err error

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("✅ 成功连接数据库")

	err = DB.AutoMigrate(
		&model.UserInfo{},
		&model.GroupInfo{},
		&model.UserContact{},
		&model.ContactApply{},
		&model.Session{},
		&model.Message{},
	)

	if err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}

	fmt.Println("✅ 数据表自动迁移成功")
}
