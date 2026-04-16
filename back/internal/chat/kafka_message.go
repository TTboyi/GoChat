// ============================================================
// 文件：back/internal/chat/kafka_message.go
// 作用：定义在 Kafka 队列中流转的消息格式 KafkaMessage。
//
// KafkaMessage vs ChatEnvelope vs model.Message vs OutgoingMessage：
//   这个项目里有四个相似但职责不同的消息结构，它们服务于消息流转的不同阶段：
//
//   ChatEnvelope（server.go 定义）：
//     WebSocket 收到前端消息后第一步解析得到的结构，字段最少，
//     只包含前端发来的原始信息（不含发送者昵称、头像等需要后端查询的字段）。
//
//   KafkaMessage（本文件定义）：
//     写入 Kafka 队列的完整消息，在 Envelope 基础上补充了：
//     MsgId（后端生成的唯一ID）、SendName、SendAvatar、CreatedAt（服务器时间）。
//     这是消息在后端内部流转的完整格式。
//
//   model.Message（back/internal/model/message.go 定义）：
//     数据库表结构，用于 GORM 持久化，有 SessionId、Status、IsRecalled 等 DB 专属字段。
//
//   OutgoingMessage（server.go 定义）：
//     发回前端的消息格式，字段名与前端 TypeScript 类型保持一致。
//     把 uuid（数据库自增ID）换成了前端认识的 uuid 字段名，等等。
//
// json tag 的含义：
//   `json:"msgId"` → 序列化/反序列化时使用 "msgId" 作为 JSON key 名
//   `json:"localId,omitempty"` → 字段为空时省略这个 key（不输出 "localId": ""）
// ============================================================

package chat

// KafkaMessage — Kafka 中的统一消息结构（含完整元数据，供 dispatcher/persist 使用）
type KafkaMessage struct {
	MsgId      string            `json:"msgId"`
	LocalId    string            `json:"localId,omitempty"`   // 前端生成，用于乐观更新
	Type       int8              `json:"type"`
	SendId     string            `json:"sendId"`
	SendName   string            `json:"sendName,omitempty"`
	SendAvatar string            `json:"sendAvatar,omitempty"`
	ReceiveId  string            `json:"receiveId"`
	Content    string            `json:"content,omitempty"`
	Url        string            `json:"url,omitempty"`
	FileName   string            `json:"fileName,omitempty"`
	FileType   string            `json:"fileType,omitempty"`
	FileSize   string            `json:"fileSize,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
	CreatedAt  int64             `json:"createdAt"`
}
