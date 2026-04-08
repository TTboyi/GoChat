package req

// 获取聊天记录（私聊）
type GetMessageListRequest struct {
	TargetId   string `json:"targetId" binding:"required"` // 对方用户 UUID
	Limit      int    `json:"limit"`                       // 限制条数，默认50
	BeforeTime int64  `json:"beforeTime"`                  // Unix时间戳，加载此时间之前的消息（分页用）
}

// 获取群聊消息
type GetGroupMessageListRequest struct {
	GroupId    string `json:"groupId" binding:"required"`
	Limit      int    `json:"limit"`
	BeforeTime int64  `json:"beforeTime"` // Unix时间戳，分页用
}
