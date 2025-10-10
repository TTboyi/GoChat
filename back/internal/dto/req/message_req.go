package req

// 获取聊天记录（私聊）
type GetMessageListRequest struct {
	TargetId string `json:"targetId" binding:"required"` // 对方用户 UUID
	Limit    int    `json:"limit"`                       // 限制条数
}

// 获取群聊消息
type GetGroupMessageListRequest struct {
	GroupId string `json:"groupId" binding:"required"`
	Limit   int    `json:"limit"`
}
