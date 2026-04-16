// ============================================================
// 文件：back/internal/model/session.go
// 作用：定义会话表模型，对应数据库的 session 表。
//       每条记录代表"某用户 A 和某人/群 B 之间的聊天列表条目"。
//
// Session 和 Message 的关系：
//   Session（会话）= 聊天列表里的一项，包含"对方是谁"、"头像名称"
//   Message（消息）= 聊天窗口里的每条消息，通过 session_id 关联到会话
//   一个 Session 对应多条 Message（一对多）。
//
// 为什么 Session 要单独存一张表？
//   聊天列表的查询需求是"给我这个用户所有会话，按最近消息排序"。
//   如果没有 session 表，每次打开聊天列表都要 GROUP BY 消息表再排序，
//   数据量大时会很慢。Session 表专门存最新状态，查询更高效。
//
// 唯一索引（idx_session_pair）：
//   (send_id, receive_id) 组合唯一索引，
//   保证同一用户和同一目标之间只有一条会话记录，防止重复创建。
// ============================================================
package model

import (
    "gorm.io/gorm"
    "time"
)

type Session struct {
    Id          int64          `gorm:"column:id;primaryKey;comment:自增id"`
    Uuid        string         `gorm:"column:uuid;uniqueIndex;type:char(20);comment:会话uuid"`
    SendId      string         `gorm:"column:send_id;type:char(20);not null;comment:创建会话人id;uniqueIndex:idx_session_pair"`
    ReceiveId   string         `gorm:"column:receive_id;type:char(20);not null;comment:接受会话人id;uniqueIndex:idx_session_pair"`
    ReceiveName string         `gorm:"column:receive_name;type:varchar(20);not null;comment:名称"`
    Avatar      string         `gorm:"column:avatar;type:char(255);default:default_avatar.png;not null;comment:头像"`
    CreatedAt   time.Time      `gorm:"column:created_at;index;type:datetime;comment:创建时间"`
    DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index;type:datetime;comment:删除时间"`
}

func (Session) TableName() string {
    return "session"
}