// ============================================================
// 文件：back/internal/dto/req/session_req.go
// 作用：会话操作（打开会话、删除会话）的请求参数结构体。
// ============================================================
package req

// 打开会话
type OpenSessionRequest struct {
	ReceiveId   string `json:"receiveId" binding:"required"`   // 接收方 UUID（用户或群聊）
	ReceiveName string `json:"receiveName" binding:"required"` // 名称
	Avatar      string `json:"avatar"`                         // 头像
}
