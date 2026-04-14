package chat

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

// Client 表示一个具体的 WebSocket 连接。
// 注意这里是一条连接，不是一个用户；同一用户可能同时拥有多个 Client。
type Client struct {
	Conn     *websocket.Conn
	Uuid     string
	SendBack chan []byte // Server → 客户端
}

// ChatMessageRequest 对应前端通过 WebSocket 发来的 JSON。
// 它把“聊天消息、群订阅、通话信令”三类行为统一装进一个入口结构里，
// 然后在 Read 中根据 Action 再细分处理。
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

// Read 持续读取前端发来的消息。
// 它的职责不是直接执行业务，而是做协议层分流：
// - join_group：更新内存订阅；
// - call_*：转发音视频信令；
// - 默认：组装成 ChatEnvelope，交给 Kafka 主链路。
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
				slog.Warn("ws_read_error", "user_id", c.Uuid, "err", err)
			} else {
				slog.Info("ws_closed", "user_id", c.Uuid, "err", err)
			}
			break
		}

		var req ChatMessageRequest

		if err := json.Unmarshal(data, &req); err != nil {
			slog.Warn("ws_parse_error", "user_id", c.Uuid, "err", err)
			continue
		}

		switch req.Action {
		case "join_group":
			if req.GroupId == "" {
				slog.Warn("join_group_missing_id", "user_id", c.Uuid)
				continue
			}
			ChatServer.AddUserToGroup(c.Uuid, req.GroupId)
			continue

		case "call_invite", "call_answer", "call_candidate", "call_end":
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

			slog.Info("msg_send", "send_id", env.SendId, "recv_id", env.ReceiveId, "type", env.Type)

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
				slog.Warn("ws_write_error", "user_id", c.Uuid, "err", err)
				return
			}

		case <-ticker.C:
		// 发送 Ping 心跳帧，保持连接；
		// 前面的 PongHandler 会在收到客户端 Pong 时延长读超时。
		c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Warn("ws_ping_error", "user_id", c.Uuid, "err", err)
				return
			}
		}
	}
}
