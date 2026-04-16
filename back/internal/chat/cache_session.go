// ============================================================
// 文件：back/internal/chat/cache_session.go
// 作用：把"这个会话有新消息了"这件事更新到 Redis 的会话列表中。
//
// updateSessionList 的具体操作：
//   Redis KEY = "chat:session:list:{userId}"（每个用户有自己的会话列表）
//   ZADD key score member：把 sessionId 加入有序集合，score = 消息时间戳
//     如果 sessionId 已存在，更新它的 score（变为最新时间戳）
//     这样"最近有消息的会话"的 score 最大，排在最前面
//   ZREMRANGEBYRANK key 0 -21：只保留最近 20 个会话（超出的最旧的会被删除）
//
// 为什么 sendId 和 receiveId 都要更新？
//   私聊时，发消息后：
//   - 发送方的会话列表（chat:session:list:sendId）需要更新
//   - 接收方的会话列表（chat:session:list:recvId）也需要更新
//   这样双方打开聊天列表，都能看到这个会话排在最前面。
// ============================================================

package chat

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func updateSessionList(
	ctx context.Context,
	rdb *redis.Client,
	userId, sessionId string,
	ts int64,
) {
	if userId == "" {
		return
	}

	key := fmt.Sprintf("chat:session:list:%s", userId)

	if err := rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(ts),
		Member: sessionId,
	}).Err(); err != nil {
		return
	}

	// 只保留最近 20 个
	rdb.ZRemRangeByRank(ctx, key, 0, -maxSessionCount-1)
}
