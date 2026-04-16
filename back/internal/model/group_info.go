// ============================================================
// 文件：back/internal/model/group_info.go
// 作用：定义群聊主表模型，对应数据库的 group_info 表。
//
// Members 字段（JSON 列）：
//   类型是 json.RawMessage，数据库列类型是 JSON。
//   存储格式：["userId1", "userId2", "userId3"]（用户UUID的 JSON 数组）
//   为什么不用单独的群成员表？
//   - 群成员不需要复杂查询（查成员只需要反序列化这个数组）
//   - JSON 列比额外的关联表省一次 JOIN，读取更快
//   - 缺点：成员数量很多时，更新一个成员就要读写整个 JSON，有并发安全问题
//     （本项目群成员数量有限，这个权衡是合理的）
//   查找"我参与的群"时，使用 MySQL 的 JSON_CONTAINS 函数在数据库层过滤。
//
// Uuid 的设计（6位数字）：
//   群 UUID 是 6 位随机数字（000000~999999），比用户 UUID 短，
//   方便用户输入群号加入群聊（类似 QQ 群号）。
//   generateGroupID 函数用循环确保不重复。
//
// Status 字段的含义：
//   0 = 正常
//   1 = 禁用（群存在但无法发消息）
//   2 = 解散（群已解散）
// ============================================================
package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type GroupInfo struct {
	Id        int64           `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid      string          `gorm:"column:uuid;uniqueIndex;type:char(6);not null;comment:群组唯一id"`
	Name      string          `gorm:"column:name;type:varchar(20);not null;comment:群名称"`
	Notice    string          `gorm:"column:notice;type:varchar(500);comment:群公告"`
	Members   json.RawMessage `gorm:"column:members;type:json;comment:群组成员"`
	MemberCnt int             `gorm:"column:member_cnt;default:1;comment:群人数"`
	OwnerId   string          `gorm:"column:owner_id;type:char(20);not null;comment:群主uuid"`
	AddMode   int8            `gorm:"column:add_mode;default:0;comment:加群方式，0.直接，1.审核"`
	Avatar    string          `gorm:"column:avatar;type:char(255);default:https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png;not null;comment:头像"`
	Status    int8            `gorm:"column:status;default:0;comment:状态，0.正常，1.禁用，2.解散"`
	CreatedAt time.Time       `gorm:"column:created_at;index;type:datetime;not null;comment:创建时间"`
	DeletedAt gorm.DeletedAt  `gorm:"column:deleted_at;index;comment:删除时间"`
}

func (GroupInfo) TableName() string {
	return "group_info"
}
