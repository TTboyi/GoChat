// ============================================================
// 文件：back/internal/model/message.go
// 作用：定义消息持久化表的数据库模型，对应 message 表。
//
// 字段设计思路（"宽表"设计）：
//   一张表同时承载文本、文件、通话三种类型的消息：
//   - type = 0：文本消息，使用 content 字段
//   - type = 1：文件消息，使用 url / fileName / fileType / fileSize 字段
//   - type = 2：通话记录，使用 avData 字段
//   没有用到的字段留空（NULL），这叫"宽表"设计，用一张表覆盖多种场景，
//   避免了多表联查的复杂性（代价是有些字段会浪费空间）。
//
// ReadAt（已读时间）：
//   指针类型 *time.Time，NULL 表示"未读"，非 NULL 表示"已读，时间是XXX"。
//   用 NULL 而不是 bool(IsRead) 的好处：可以知道消息是什么时候被读的。
//
// 复合索引（idx_session_time）：
//   gorm:"index:idx_session_time,priority:1"（SessionId 在索引里优先级更高）
//   gorm:"index:idx_session_time,priority:2"（CreatedAt 是次级排序）
//   这个复合索引专门为"查某个会话里时间段内的消息"优化：
//   WHERE session_id = ? AND created_at < ? ORDER BY created_at DESC
//   有了这个索引，查询不需要全表扫描，直接定位到对应会话的消息范围。
// ============================================================
package model

import "time"

// Message 是消息持久化表。
// 一条记录既能表示文本消息，也能表示文件消息，甚至能承载通话过程中的附加数据，
// 因此字段设计上偏“宽表”。
type Message struct {
	Id         int64      `gorm:"column:id;primaryKey;comment:自增id" json:"id"`
	Uuid       string     `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:消息uuid" json:"uuid"`
	SessionId  string     `gorm:"column:session_id;type:char(20);not null;comment:会话uuid;index:idx_session_time,priority:1" json:"sessionId"`
	Type       int8       `gorm:"column:type;not null;comment:消息类型，0.文本，1.文件，2.通话" json:"type"`
	Content    string     `gorm:"column:content;type:TEXT;comment:消息内容" json:"content"`
	Url        string     `gorm:"column:url;type:char(255);comment:消息url" json:"url"`
	SendId     string     `gorm:"column:send_id;index;type:char(20);not null;comment:发送者uuid" json:"sendId"`
	SendName   string     `gorm:"column:send_name;type:varchar(20);not null;comment:发送者昵称" json:"sendName"`
	SendAvatar string     `gorm:"column:send_avatar;type:varchar(255);not null;comment:发送者头像" json:"sendAvatar"`
	ReceiveId  string     `gorm:"column:receive_id;type:char(20);not null;comment:接受者uuid;index:idx_receive_time,priority:1" json:"receiveId"`
	FileType   string     `gorm:"column:file_type;type:char(10);comment:文件类型" json:"fileType"`
	FileName   string     `gorm:"column:file_name;type:varchar(255);comment:文件名" json:"fileName"`
	FileSize   string     `gorm:"column:file_size;type:char(20);comment:文件大小" json:"fileSize"`
	Status     int8       `gorm:"column:status;not null;comment:状态，0.未发送，1.已发送" json:"status"`
	IsRecalled int8       `gorm:"column:is_recalled;default:0;comment:是否撤回，0.否，1.是" json:"isRecalled"`
	ReadAt     *time.Time `gorm:"column:read_at;comment:已读时间" json:"readAt"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;comment:创建时间;index:idx_session_time,priority:2;index:idx_receive_time,priority:2" json:"createdAt"`
	AVdata     string     `gorm:"column:av_data;comment:通话传递数据" json:"avData"`
}

// TableName 显式指定数据库表名。
func (Message) TableName() string {
	return "message"
}
