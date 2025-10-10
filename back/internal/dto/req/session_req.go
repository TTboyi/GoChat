package req

// 打开会话
type OpenSessionRequest struct {
	ReceiveId   string `json:"receiveId" binding:"required"`   // 接收方 UUID（用户或群聊）
	ReceiveName string `json:"receiveName" binding:"required"` // 名称
	Avatar      string `json:"avatar"`                         // 头像
}
