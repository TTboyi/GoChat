package req

// ChatMessageRequest 聊天消息请求
type ChatMessageRequest struct {
	Type      int8   `json:"type"`      // 消息类型：0=文本，1=文件，2=通话
	Content   string `json:"content"`   // 消息内容
	SendId    string `json:"sendId"`    // 发送者
	ReceiveId string `json:"receiveId"` // 接收者（用户uuid 或 群聊uuid）
}
