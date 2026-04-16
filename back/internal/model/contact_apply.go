// ============================================================
// 文件：back/internal/model/contact_apply.go
// 作用：好友申请/入群申请表，记录"A 想加 B 为好友"或"A 申请加入群G"的请求。
//
// 申请状态（Status）：
//   0 = 申请中（等待审核）
//   1 = 已通过
//   2 = 已拒绝
//   3 = 已拉黑（被拉黑的申请人之后发起申请会直接静默处理）
//
// ContactType（申请类型）：
//   0 = 申请添加好友（ContactId 是目标用户的 UUID）
//   1 = 申请加入群聊（ContactId 是目标群的 UUID）
//
// LastApplyAt 字段：
//   记录最近一次申请时间，用于防止"频繁骚扰"：
//   如果同一个用户短时间内反复申请，可以在业务层加时间间隔限制。
// ============================================================
package model

import (
    "gorm.io/gorm"
    "time"
)

type ContactApply struct {
    Id          int64          `gorm:"column:id;primaryKey;comment:自增id"`
    Uuid        string         `gorm:"column:uuid;uniqueIndex;type:char(20);comment:申请id"`
    UserId      string         `gorm:"column:user_id;index;type:char(20);not null;comment:申请人id"`
    ContactId   string         `gorm:"column:contact_id;index;type:char(20);not null;comment:被申请id"`
    ContactType int8           `gorm:"column:contact_type;not null;comment:被申请类型，0.用户，1.群聊"`
    Status      int8           `gorm:"column:status;not null;comment:申请状态，0.申请中，1.通过，2.拒绝，3.拉黑"`
    Message     string         `gorm:"column:message;type:varchar(100);comment:申请信息"`
    LastApplyAt time.Time      `gorm:"column:last_apply_at;type:datetime;not null;comment:最后申请时间"`
    DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index;type:datetime;comment:删除时间"`
}

func (ContactApply) TableName() string {
    return "contact_apply"
}