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
