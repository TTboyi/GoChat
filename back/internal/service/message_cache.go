// ============================================================
// 文件：back/internal/service/message_cache.go
// 作用：从 Redis 缓存中读取历史消息（辅助函数）。
//
// 函数 getMessageListFromRedis 直接从 Redis List 里读取指定数量的消息。
// key 格式：im:chat:messages:{userId}:{targetId}
// 使用 LRANGE 命令从尾部取最近 N 条（负索引表示从尾部算起）。
//
// 注意：这个 key 格式与 chat 包里 cacheMessage 使用的
//       chat:session:msgs:{sessionId} 格式不同，是另一套缓存键，
//       可能是早期版本遗留或用于特定接口的辅助查询。
// ============================================================
// internal/service/message_cache.go
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"chatapp/back/internal/config"
	"chatapp/back/internal/dto/cache"
)

func getMessageListFromRedis(
	ctx context.Context,
	userId, targetId string,
	limit int,
) ([]cache.ChatMessage, error) {

	rdb := config.GetRedis()
	key := fmt.Sprintf("im:chat:messages:%s:%s", userId, targetId)

	if limit <= 0 || limit > 100 {
		limit = 100
	}

	values, err := rdb.LRange(ctx, key, int64(-limit), -1).Result()
	if err != nil {
		return nil, err
	}

	list := make([]cache.ChatMessage, 0, len(values))
	for _, v := range values {
		var msg cache.ChatMessage
		if err := json.Unmarshal([]byte(v), &msg); err == nil {
			list = append(list, msg)
		}
	}

	return list, nil
}
