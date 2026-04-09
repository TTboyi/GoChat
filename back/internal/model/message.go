package model

import "time"

type Message struct {
	Id         int64      `gorm:"column:id;primaryKey;comment:自增id" json:"id"`
	Uuid       string     `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:消息uuid" json:"uuid"`
	SessionId  string     `gorm:"column:session_id;index;type:char(20);not null;comment:会话uuid" json:"sessionId"`
	Type       int8       `gorm:"column:type;not null;comment:消息类型，0.文本，1.文件，2.通话" json:"type"`
	Content    string     `gorm:"column:content;type:TEXT;comment:消息内容" json:"content"`
	Url        string     `gorm:"column:url;type:char(255);comment:消息url" json:"url"`
	SendId     string     `gorm:"column:send_id;index;type:char(20);not null;comment:发送者uuid" json:"sendId"`
	SendName   string     `gorm:"column:send_name;type:varchar(20);not null;comment:发送者昵称" json:"sendName"`
	SendAvatar string     `gorm:"column:send_avatar;type:varchar(255);not null;comment:发送者头像" json:"sendAvatar"`
	ReceiveId  string     `gorm:"column:receive_id;index;type:char(20);not null;comment:接受者uuid" json:"receiveId"`
	FileType   string     `gorm:"column:file_type;type:char(10);comment:文件类型" json:"fileType"`
	FileName   string     `gorm:"column:file_name;type:varchar(255);comment:文件名" json:"fileName"`
	FileSize   string     `gorm:"column:file_size;type:char(20);comment:文件大小" json:"fileSize"`
	Status     int8       `gorm:"column:status;not null;comment:状态，0.未发送，1.已发送" json:"status"`
	IsRecalled int8       `gorm:"column:is_recalled;default:0;comment:是否撤回，0.否，1.是" json:"isRecalled"`
	ReadAt     *time.Time `gorm:"column:read_at;comment:已读时间" json:"readAt"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;comment:创建时间" json:"createdAt"`
	AVdata     string     `gorm:"column:av_data;comment:通话传递数据" json:"avData"`
}

func (Message) TableName() string {
	return "message"
}
