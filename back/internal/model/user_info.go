// ============================================================
// 文件：back/internal/model/user_info.go
// 作用：定义用户主表的数据库模型，对应数据库中的 user_info 表。
//
// GORM 标签解读（以 Uuid 字段为例）：
//   gorm:"column:uuid"         → 数据库列名为 uuid（不是 Go 的 Uuid）
//   gorm:"uniqueIndex"         → 建立唯一索引，保证每个用户的 uuid 不重复
//   gorm:"type:char(20)"       → 数据库字段类型是定长字符 20 位
//   gorm:"not null"            → 数据库不允许这个字段为 NULL
//   gorm:"comment:用户唯一id"   → 数据库列注释
//
// 为什么同时有 Id（自增）和 Uuid（UUID）两个标识字段？
//   - Id（自增整数）：数据库内部主键，查询高效（整数比较比字符串快），
//     用于 JOIN 操作和内部引用。
//   - Uuid（字符串）：对外暴露的用户标识，不暴露自增规律（防止猜测"有多少用户"），
//     格式固定方便前端处理，在 URL、JWT、WebSocket 消息里使用。
//
// gorm.DeletedAt（软删除）：
//   删除用户时，不是真正从数据库删除这行，而是把 deleted_at 字段设置为当前时间。
//   GORM 的 Unscoped() 可以查到软删除的记录，普通查询会自动过滤掉已删除的。
//   好处：数据可以恢复，也可以用于审计（知道什么时候删除的）。
//
// Password 字段（bcrypt 哈希）：
//   密码不能明文存储！注册时用 bcrypt 算法把明文密码哈希成 72 位字符串。
//   bcrypt 是"慢哈希"算法：即使数据库泄露，攻击者也要花费大量时间枚举密码。
//   登录时用 bcrypt.CompareHashAndPassword 比对，不需要也不能"解密"哈希。
// ============================================================
package model

import (
	"time"

	"gorm.io/gorm"
)

// UserInfo 对应系统中的“用户主表”。
// 这个模型同时承担了登录资料、展示资料、权限状态三类信息，
// 因此在阅读 controller/service 时会频繁见到它。
type UserInfo struct {
	Id        int64          `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid      string         `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:用户唯一id"`
	Nickname  string         `gorm:"column:nickname;type:varchar(20);not null;comment:昵称"`
	Telephone string         `gorm:"column:telephone;index;not null;type:char(11);comment:电话"`
	Email     string         `gorm:"column:email;type:char(30);comment:邮箱"`
	Avatar    string         `gorm:"column:avatar;type:char(255);default:https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png;not null;comment:头像"`
	Gender    int8           `gorm:"column:gender;comment:性别，0.男，1.女"`
	Signature string         `gorm:"column:signature;type:varchar(100);comment:个性签名"`
	Password  string         `gorm:"column:password;type:varchar(72);not null;comment:密码"`
	Birthday  string         `gorm:"column:birthday;type:char(8);comment:生日"`
	CreatedAt time.Time      `gorm:"column:created_at;index;type:datetime;not null;comment:创建时间"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;comment:删除时间"`
	IsAdmin   int8           `gorm:"column:is_admin;not null;comment:是否是管理员，0.不是，1.是"`
	Status    int8           `gorm:"column:status;not null;comment:状态，0.正常，1.禁用"`
}

// TableName 显式指定表名，避免 GORM 使用默认复数推断。
func (UserInfo) TableName() string {
	return "user_info"
}
