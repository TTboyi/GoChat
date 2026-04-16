// ============================================================
// 文件：back/internal/dao/user_dao.go
// 作用：用户数据访问层（DAO = Data Access Object），封装用户相关的数据库查询。
//
// 什么是 DAO（数据访问对象）？
//   DAO 是一个设计模式：把"如何查询数据库"的代码单独放在一层，
//   上层的 service 只管"我要什么数据"，不需要知道 SQL 细节。
//   好处：如果以后换数据库（比如从 MySQL 换成 PostgreSQL），只改 DAO 层就行。
//
// 本项目的 DAO 层很轻薄（只有一个文件），因为大多数查询直接在 service 里写了。
// GetUserByName 是被 RegisterUser/LoginUser 频繁调用的基础查询。
// ============================================================
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
