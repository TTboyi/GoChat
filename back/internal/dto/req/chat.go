// ============================================================
// 文件：back/internal/dto/req/chat.go
// 作用：WebSocket 消息请求相关的 DTO（数据传输对象）结构体。
//
// 什么是 DTO（数据传输对象）？
//   DTO 是专门用于"数据传输"的简单结构体，它只有数据，没有业务逻辑。
//   和 Model（数据库模型）的区别：
//   - Model 的字段和数据库表对应，可能包含数据库敏感字段（如密码）
//   - DTO 的字段和接口请求/响应对应，只暴露前端需要的字段
//
// req/ 目录下的 DTO 是"请求入参"DTO，用于接收前端提交的数据。
// resp/ 目录下的 DTO 是"响应出参"DTO，用于向前端返回数据。
// ============================================================
package req

// ChatMessageRequest 聊天消息请求
type ChatMessageRequest struct {
	Type      int8   `json:"type"`      // 消息类型：0=文本，1=文件，2=通话
	Content   string `json:"content"`   // 消息内容
	SendId    string `json:"sendId"`    // 发送者
	ReceiveId string `json:"receiveId"` // 接收者（用户uuid 或 群聊uuid）
}
