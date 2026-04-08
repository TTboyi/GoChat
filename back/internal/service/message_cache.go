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
