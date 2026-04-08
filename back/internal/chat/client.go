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
	Type      int8   `json:"type"`
	Content   string `json:"content"`
	ReceiveId string `json:"receiveId"`
	SendId    string `json:"sendId"`
	Action    string `json:"action"` // join_group / call_invite / call_answer / call_candidate / call_end
	GroupId   string `json:"groupId"`
	LocalId   string `json:"localId"`  // 乐观更新用
	Url       string `json:"url"`
	FileName  string `json:"fileName"`
	FileType  string `json:"fileType"`
	FileSize  string `json:"fileSize"`

	// 通话相关
	CallType string `json:"callType"`
	CallId   string `json:"callId"`
	Accept   *bool  `json:"accept"`
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
		log.Printf("🧩 收到前端 action=%q content len=%d", req.Action, len(req.Content))

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
			env := ChatEnvelope{
				Type:      req.Type,
				Content:   req.Content,
				Url:       req.Url,
				FileName:  req.FileName,
				FileType:  req.FileType,
				FileSize:  req.FileSize,
				SendId:    nz(req.SendId, c.Uuid),
				ReceiveId: req.ReceiveId,
				LocalId:   req.LocalId,
			}

			log.Printf("📤 Send Kafka msg send=%s recv=%s type=%d content=%q",
				env.SendId,
				env.ReceiveId,
				env.Type,
				env.Content,
			)

			// // ✅ 1. 发 Kafka（新主链路）
			if ChatKafkaProducer != nil {

				ChatKafkaProducer.Publish(env)
			}

			// // ⚠️ 2. 暂时保留旧内存链路（下一阶段删除）
			// ChatServer.Transmit <- env

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
