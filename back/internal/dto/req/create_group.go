// ============================================================
// 文件：back/internal/dto/req/create_group.go
// 作用：创建群聊请求的参数结构体。
// ============================================================
package req

type CreateGroupRequest struct {
	Name    string `json:"name" binding:"required"`
	Notice  string `json:"notice"`
	OwnerId string `json:"ownerId" binding:"required"` // 可从 JWT 中提取
	AddMode int    `json:"addMode "`
	Avatar  string `json:"avatar"` // 可选默认值
}
