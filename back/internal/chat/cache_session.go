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
