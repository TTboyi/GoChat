package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"chatapp/back/internal/config"
)

const (
	maxSessionCount = 20
	maxMessageCount = 100
)

func cacheMessage(km *KafkaMessage) error {
	rdb := config.GetRedis()
	ctx := context.Background()

	sessionId := buildSessionId(km)

	// 1️⃣ 缓存最近消息（LIST）
	msgRaw, _ := json.Marshal(km)
	msgKey := fmt.Sprintf("chat:session:msgs:%s", sessionId)

	if err := rdb.LPush(ctx, msgKey, msgRaw).Err(); err != nil {
		return err
	}
	if err := rdb.LTrim(ctx, msgKey, 0, maxMessageCount-1).Err(); err != nil {
		return err
	}

	// 2️⃣ 更新会话列表（ZSET）
	updateSessionList(ctx, rdb, km.SendId, sessionId, km.CreatedAt)
	updateSessionList(ctx, rdb, km.ReceiveId, sessionId, km.CreatedAt)

	return nil
}
