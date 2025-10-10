package chat

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"

	"chatapp/back/internal/model"
	"chatapp/back/internal/service"
	"chatapp/back/utils"
	"time"
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

		var chatMsg ChatMessageRequest
		if err := json.Unmarshal(msg, &chatMsg); err != nil {
			log.Println("消息解析失败:", err)
			continue
		}

		fmt.Printf("收到消息: %+v\n", chatMsg)

		// ✅ 保存消息到数据库
		message := &model.Message{
			Uuid:      utils.GenerateUUID(20), // 生成唯一消息ID
			SendId:    c.Uuid,                 // 发送者ID（不信任前端）
			ReceiveId: chatMsg.ReceiveId,
			Type:      chatMsg.Type,
			Content:   chatMsg.Content,
			CreatedAt: time.Now(),
		}

		if err := service.SaveMessage(message); err != nil {
			log.Printf("❌ 保存消息失败: %v", err)
		} else {
			log.Printf("💾 消息已保存: %+v", message)
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
