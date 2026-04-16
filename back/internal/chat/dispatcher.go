// ============================================================
// 文件：back/internal/chat/dispatcher.go
// 作用：把 Kafka 消费到的消息转换为前端格式，并推送给对应的 WebSocket 连接。
//       私聊 → 推给接收方 + 发送方（回显）
//       群聊 → 推给所有在线订阅该群的用户
//
// dispatchKafkaMessage 的两步判断：
//   第一步：isGroup(km.ReceiveId)
//     检查 groupMembers 内存表里是否有这个 receiveId 作为群ID。
//     如果有，说明这是群聊消息，走群聊分发逻辑。
//   第二步（私聊）：
//     ChatServer.DeliverToUser(km.ReceiveId, raw)  → 推给接收方
//     ChatServer.DeliverToUser(km.SendId, raw)     → 推给发送方（消息回显，让发送方看到"发送成功"）
//
// 为什么发送方也需要收到自己的消息？
//   前端用"乐观更新"：发消息后立刻在界面上显示一个"发送中"气泡，
//   等服务器推回同一条消息（带有服务器生成的 uuid 和时间戳），
//   用 LocalId 字段把"发送中"气泡替换成"已发送"。
//   这样用户看到的界面是即时的，而不是等服务器确认才显示。
// ============================================================

package chat

import (
	"encoding/json"
	"log"
)

// dispatchKafkaMessage 将 KafkaMessage 转换为前端所需格式并推送
func dispatchKafkaMessage(km *KafkaMessage) {
	log.Printf("🔵 dispatch send=%s recv=%s", km.SendId, km.ReceiveId)

	// 构造前端 OutgoingMessage（字段名与前端一致）
	out := OutgoingMessage{
		Uuid:       km.MsgId,    // 前端需要 uuid 字段
		LocalId:    km.LocalId,  // 乐观更新匹配
		Type:       km.Type,
		Content:    km.Content,
		Url:        km.Url,
		FileName:   km.FileName,
		FileType:   km.FileType,
		FileSize:   km.FileSize,
		SendId:     km.SendId,
		SendName:   km.SendName,
		SendAvatar: km.SendAvatar,
		ReceiveId:  km.ReceiveId,
		CreatedAt:  km.CreatedAt,
	}

	raw, err := json.Marshal(out)
	if err != nil {
		log.Printf("❌ dispatch marshal error: %v", err)
		return
	}

	// 群聊
	if isGroup(km.ReceiveId) {
		dispatchToGroup(km.ReceiveId, raw)
		return
	}

	// 点对点：推给接收方和发送方（回显）
	ChatServer.DeliverToUser(km.ReceiveId, raw)
	ChatServer.DeliverToUser(km.SendId, raw)
}

func isGroup(id string) bool {
	groupMemsMu.RLock()
	_, ok := groupMembers[id]
	groupMemsMu.RUnlock()
	return ok
}

func dispatchToGroup(groupId string, raw []byte) {
	groupMemsMu.RLock()
	subs := make(map[string]bool, len(groupMembers[groupId]))
	for k, v := range groupMembers[groupId] {
		subs[k] = v
	}
	groupMemsMu.RUnlock()
	for uid := range subs {
		ChatServer.DeliverToUser(uid, raw)
	}
}
