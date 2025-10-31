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
	Type      int8   `json:"type"`      // 0=文本, 1=文件, 2=通话信令
	Content   string `json:"content"`   // 消息内容 / SDP / ICE
	ReceiveId string `json:"receiveId"` // 接收方ID（用户或群）
	SendId    string `json:"sendId"`    // 发送者ID（兜底client.Uuid）
	Action    string `json:"action"`    // join_group / call_invite / call_answer / call_candidate / call_end / send_message
	GroupId   string `json:"groupId"`   // 用于 join_group 订阅

	// 通话相关字段
	CallType string `json:"callType"` // "audio" | "video"
	CallId   string `json:"callId"`   // 通话唯一ID
	Accept   *bool  `json:"accept"`   // 对于 call_answer：true=接听 false=拒绝
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
		log.Printf("🧩 收到前端 action=%q content len=%d", req.Action, len(req.Content))

		if err := json.Unmarshal(data, &req); err != nil {
			log.Printf("❌ 无法解析前端消息: %v", err)
			continue
		}

		switch req.Action {
		case "join_group":
			if req.GroupId == "" {
				log.Printf("⚠️ join_group 缺少 groupId")
				continue
			}
			ChatServer.AddUserToGroup(c.Uuid, req.GroupId)
			log.Printf("✅ 用户 %s 成功 join_group %s", c.Uuid, req.GroupId)
			continue

		case "call_invite", "call_answer", "call_candidate", "call_end":
			// 音视频信令转发
			log.Printf("✅ 转发通话信令 action=%s to=%s", req.Action, req.ReceiveId)
			ChatServer.ForwardCallSignal(c.Uuid, req)
			continue

		default:
			// 普通消息
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
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("❌ Write 错误: %v", err)
			break
		}
	}
}
