package cache

// ChatMessage = Redis / Kafka 用的消息结构
// 注意：这是“缓存模型”，不是 DB 模型
type ChatMessage struct {
	Id        int64  `json:"id"`
	Uuid      string `json:"uuid"`
	SendId    string `json:"send_id"`
	ReceiveId string `json:"receive_id"`
	Type      int8   `json:"type"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"` // Unix 时间戳（秒）
}
