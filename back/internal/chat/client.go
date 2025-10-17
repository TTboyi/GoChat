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

// ChatMessageRequest 前端发来的消息
type ChatMessageRequest struct {
	Type      int8   `json:"type"`      // 消息类型
	Content   string `json:"content"`   // 消息内容
	ReceiveId string `json:"receiveId"` // 接收方 id
	SendId    string `json:"sendId"`    // 发送方 id
}

// ===== WebSocket 加入群聊指令 =====
type JoinGroupRequest struct {
	Action  string `json:"action"`
	GroupId string `json:"groupId"`
}

func (c *Client) Read() {
	defer func() {
		ChatServer.RemoveClient(c.Uuid)
		_ = c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("❌ 用户 %s 读取消息失败: %v", c.Uuid, err)
			break
		}

		// ✅ 检查是否为 join_group 类型
		var join JoinGroupRequest
		if err := json.Unmarshal(msg, &join); err == nil && join.Action == "join_group" {
			log.Printf("✅ 用户 %s 订阅群 %s 的 WebSocket 消息", c.Uuid, join.GroupId)
			ChatServer.AddUserToGroup(c.Uuid, join.GroupId)
			continue
		}

		var chatMsg ChatMessageRequest
		if err := json.Unmarshal(msg, &chatMsg); err != nil {
			log.Println("消息解析失败:", err)
			continue
		}

		// ✅ 广播到其他客户端
		env := ChatEnvelope{
			Type:      chatMsg.Type,
			Content:   chatMsg.Content,
			SendId:    c.Uuid,
			ReceiveId: chatMsg.ReceiveId,
		}
		ChatServer.Transmit <- env
	}
}

// Write 负责向前端发送消息
func (c *Client) Write() {
	defer func() {
		ChatServer.RemoveClient(c.Uuid)
		_ = c.Conn.Close()
	}()

	for msg := range c.SendBack {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("❌ 用户 %s 发送消息失败: %v", c.Uuid, err)
			break
		}
	}
}
