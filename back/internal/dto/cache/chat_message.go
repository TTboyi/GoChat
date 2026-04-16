// ============================================================
// 文件：back/internal/dto/cache/chat_message.go
// 作用：Redis 缓存中存储消息的数据结构。
//
// ChatMessage 和 model.Message 的区别：
//   model.Message 有很多数据库专属字段（SessionId、DeletedAt 等），
//   直接把 model.Message 序列化进 Redis 会浪费空间（存了很多不需要的字段）。
//   ChatMessage 只保留"展示一条消息气泡所需的最少字段"，
//   序列化后体积更小，Redis 内存占用更低。
// ============================================================
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
