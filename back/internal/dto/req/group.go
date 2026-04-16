// ============================================================
// 文件：back/internal/dto/req/group.go
// 作用：群聊管理相关操作（修改名称、公告、头像、移除成员等）的请求参数。
// ============================================================
package req

// 查询我创建的群聊请求
type LoadMyGroupRequest struct {
	OwnerId string `json:"ownerId" binding:"required"` // 群主的用户uuid
}

// 检查加群方式请求
type CheckGroupAddModeRequest struct {
	GroupId string `json:"groupId" binding:"required"` // 群聊uuid
}
