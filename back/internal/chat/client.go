// ============================================================
// 文件：back/internal/chat/client.go
// 作用：定义单个 WebSocket 连接的数据结构（Client），以及
//       该连接的读循环（Read）和写循环（Write）。
//
// 关键概念：WebSocket
//   WebSocket 是一种"全双工"通信协议：服务器和客户端可以在同一条连接上
//   随时互相发送消息，而不需要客户端每次发请求才能收到数据。
//   对比 HTTP：HTTP 是"请求-响应"模式，客户端问一次服务器答一次；
//             WebSocket 建立后，服务器可以主动"推"消息给客户端。
//
// 为什么读写要分两个 goroutine（协程）？
//   网络 I/O 是阻塞操作：等待消息就得卡在那里。
//   如果读和写在同一个 goroutine，写消息时就没法同时读，反之亦然。
//   用两个协程，一个专门等着读，另一个专门等着写，互不阻塞。
//   Go 的 goroutine 非常轻量（几KB栈内存），同时运行成千上万个没问题。
//
// SendBack channel（通道）的作用：
//   Server 要给客户端发消息时，不是直接调用 WebSocket 写函数，
//   而是把消息放进 SendBack 这个 channel。
//   写 goroutine（Write 函数）会持续监听这个 channel，有消息就发。
//   这样的好处：解耦（Server 不需要知道 WebSocket 细节）+ 线程安全（channel 本身是并发安全的）。
//
// 心跳机制（Ping/Pong）：
//   WebSocket 连接建立后，如果长时间没有数据传输，中间的防火墙或负载均衡器
//   可能会把"沉默"的连接认为是僵死的并强制断开。
//   解决方案：服务器每 30 秒主动发一个 Ping 帧，客户端收到后回一个 Pong 帧。
//   如果 60 秒内没收到 Pong，服务器就认为连接已断，主动关闭。
// ============================================================

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
