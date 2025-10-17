package chat

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

// Client 表示一个 WebSocket 客户端
type Client struct {
	Conn     *websocket.Conn
	Uuid     string
	SendBack chan []byte // Server → 客户端
}

// ChatMessageRequest 前端发来的消息结构
type ChatMessageRequest struct {
	Type      int8   `json:"type"`      // 消息类型：0=文本
	Content   string `json:"content"`   // 消息内容
	ReceiveId string `json:"receiveId"` // 接收方ID（用户或群）
	SendId    string `json:"sendId"`    // 发送者ID（兜底client.Uuid）
	Action    string `json:"action"`    // join_group / send_message
	GroupId   string `json:"groupId"`   // 用于 join_group 订阅
}

// Read 循环监听前端消息
func (c *Client) Read() {
	defer func() {
		ChatServer.Logout <- c
		_ = c.Conn.Close()
	}()

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("❌ Read 错误: %v", err)
			break
		}

		var req ChatMessageRequest
		if err := json.Unmarshal(data, &req); err != nil {
			log.Printf("❌ 无法解析前端消息: %v", err)
			continue
		}

		// ✅ 区分操作类型
		switch req.Action {
		case "join_group":
			if req.GroupId == "" {
				log.Printf("⚠️ join_group 缺少 groupId")
				continue
			}
			ChatServer.AddUserToGroup(c.Uuid, req.GroupId)
			log.Printf("✅ 用户 %s 成功 join_group %s", c.Uuid, req.GroupId) // ← 加这行
			continue

		default:
			// ✅ 普通聊天消息，推送到主消息通道
			env := ChatEnvelope{
				Type:      req.Type,
				Content:   req.Content,
				SendId:    nz(req.SendId, c.Uuid),
				ReceiveId: req.ReceiveId,
			}
			ChatServer.Transmit <- env
		}
	}
}

// Write 循环下发服务端消息
func (c *Client) Write() {
	defer func() {
		_ = c.Conn.Close()
	}()

	for msg := range c.SendBack {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("❌ Write 错误: %v", err)
			break
		}
	}
}
