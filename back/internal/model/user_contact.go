// ============================================================
// 文件：back/internal/model/user_contact.go
// 作用：用户联系人关系表，记录"用户A 的通讯录里有 B"这种关系。
//
// 数据结构设计：
//   UserId     - 通讯录拥有者（"我"）
//   ContactId  - 通讯录里的人或群的 UUID
//   ContactType - 0=好友，1=群聊（区分联系人类型）
//   Status     - 0=正常，1=拉黑
//
// 为什么好友关系需要两条记录？
//   用户 A 和用户 B 互加好友后，会创建两条记录：
//   · (userId=A, contactId=B, type=0)  → 代表"A的通讯录里有B"
//   · (userId=B, contactId=A, type=0)  → 代表"B的通讯录里有A"
//   查询"A的联系人列表"只需 WHERE user_id = A，不需要 JOIN。
//   代价：删除好友时需要删除两条记录（contact_service.go 里的 DeleteContact 实现了这个）。
//
// 索引（idx_user_contact_lookup）：
//   (user_id, contact_type, status) 三列联合索引，
//   专为 "WHERE user_id=? AND contact_type=0 AND status=0"（查询正常好友列表）优化。
// ============================================================
package model

import (
    "gorm.io/gorm"
    "time"
)

type UserContact struct {
    Id          int64          `gorm:"column:id;primaryKey;comment:自增id"`
    UserId      string         `gorm:"column:user_id;type:char(20);not null;comment:用户唯一id;index:idx_user_contact_lookup"`
    ContactId   string         `gorm:"column:contact_id;type:char(20);not null;comment:对应联系id"`
    ContactType int8           `gorm:"column:contact_type;not null;comment:联系类型，0.用户，1.群聊;index:idx_user_contact_lookup"`
    Status      int8           `gorm:"column:status;not null;comment:联系状态，0.正常，1.拉黑，...;index:idx_user_contact_lookup"`
    CreatedAt   time.Time      `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
    DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index;comment:删除时间"`
}

func (UserContact) TableName() string {
    return "user_contact"
}