package chat

import (
	"encoding/json"
	"log"
	"time"

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

const (
	pingInterval = 30 * time.Second  // 每30秒发一次心跳
	pongWait     = 60 * time.Second  // 60秒内没收到 Pong 则认为断线
	writeWait    = 10 * time.Second  // 写超时
)

// Read 循环监听前端消息（含心跳 Ping）
func (c *Client) Read() {
	defer func() {
		ChatServer.RemoveClient(c)
	}()

	// 设置读取截止时间（Pong 处理器会重置）
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Read 非正常断开: %v", err)
			} else {
				log.Printf("ℹ️ Read 连接关闭: %v", err)
			}
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

			// ✅ 1. 发 Kafka（新主链路）
			if ChatKafkaProducer != nil {
				ChatKafkaProducer.Publish(env)
			}

			// ⚠️ 2. 暂时保留旧内存链路（下一阶段删除）
			// ChatServer.Transmit <- env
		}
	}
}

// Write 循环下发服务端消息（含心跳 Ping）
func (c *Client) Write() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.SendBack:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// channel 已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("❌ Write 错误: %v", err)
				return
			}

		case <-ticker.C:
			// 发送 Ping 心跳帧，保持连接
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Ping 发送失败，连接断开: %v", err)
				return
			}
			log.Printf("💓 Ping -> %s", c.Uuid)
		}
	}
}
