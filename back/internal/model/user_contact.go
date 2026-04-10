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