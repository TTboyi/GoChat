// ============================================================
// 文件：back/internal/chat/cache_logic.go
// 作用：实现把消息写入 Redis 缓存的具体逻辑。
//
// Redis 数据结构选择理由：
//
//   消息列表用 LIST（链表）：
//     Redis KEY = "chat:session:msgs:{sessionId}"
//     操作：LPUSH（头插）+ LTRIM（截断，只保留最新 100 条）
//     选 LIST 而不是 ZSET：消息天然有序（按写入顺序），无需额外排序字段
//     LPUSH 把最新消息放在最前（index 0），取最近 N 条只需 LRANGE key 0 N-1
//
//   会话列表用 ZSET（有序集合）：
//     Redis KEY = "chat:session:list:{userId}"
//     ZSET 的 Member = sessionId，Score = 消息时间戳（Unix 毫秒）
//     自动按 Score 排序，Score 越大（越新）排名越靠后，用 ZREVRANGE 可以取"最近的前N个"
//     每次收到新消息，用 ZADD 更新这个会话的 Score，会话列表自动重排
//     ZREMRANGEBYRANK 删除最早的条目，最多保留 20 个会话
//
// sessionId 的构造逻辑（buildSessionId 函数，在 session_id.go 里）：
//   群聊：sessionId = "G:" + groupUuid
//   私聊：把两个用户 UUID 按字母顺序排序后拼接（小的在前）
//         这样 A→B 和 B→A 的消息映射到同一个 sessionId，不会出现两份缓存
// ============================================================

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
