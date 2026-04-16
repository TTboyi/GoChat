// ============================================================
// 文件：back/database/migration.go
// 作用：数据库迁移脚本（如果存在），用于管理数据库表结构的版本化变更。
//
// 本项目主要使用 GORM 的 AutoMigrate（在 config.go 里调用），
// 这个文件可能包含手动迁移逻辑（如添加索引、修改列类型等 AutoMigrate 不能自动处理的操作）。
// ============================================================
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
	dsn := "root:password@tcp(127.0.0.1:3307)/chat_app?charset=utf8mb4&parseTime=True&loc=Local"
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
