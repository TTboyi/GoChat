package req

// 查询我创建的群聊请求
type LoadMyGroupRequest struct {
	OwnerId string `json:"ownerId" binding:"required"` // 群主的用户uuid
}

// 检查加群方式请求
type CheckGroupAddModeRequest struct {
	GroupId string `json:"groupId" binding:"required"` // 群聊uuid
}
