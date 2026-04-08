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
	_, ok := groupMembers[id]
	return ok
}

func dispatchToGroup(groupId string, raw []byte) {
	if subs, ok := groupMembers[groupId]; ok {
		for uid := range subs {
			ChatServer.DeliverToUser(uid, raw)
		}
	}
}
