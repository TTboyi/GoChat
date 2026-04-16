// ============================================================
// 文件：back/internal/chat/kafka_consumer_cache.go
// 作用：Kafka 消费者之三 —— "缓存更新消费者"。
//       负责把新消息同步到 Redis 缓存，加速会话列表和最近消息的加载。
//
// 缓存更新做了什么（cacheMessage 里实现）：
//   1. 把消息 JSON 压入 Redis 的 List（chat:session:msgs:{sessionId}）的头部
//      最多保留最近 100 条（LTRIM 自动截断）
//   2. 把这个会话的最新时间戳写入双方各自的"会话列表 ZSET"
//      (chat:session:list:{userId}，score = 消息时间戳）
//      ZSET 会自动按时间戳排序，最近有消息的会话总在前面
//
// 为什么要缓存，直接查数据库不行吗？
//   数据库（MySQL）的查询延迟通常是几到几十毫秒，
//   Redis 的读取延迟通常小于 1 毫秒（因为在内存中）。
//   对于"每次打开聊天页面都要加载会话列表"这种高频操作，
//   缓存可以让响应快 10-100 倍，用户感知明显。
//
// 和 Persist 消费者的 Offset 策略不同：
//   Cache 用 OffsetNewest，不回放历史。
//   如果 Redis 缓存丢失（重启/清缓存），下次用户访问会直接从 MySQL 读，
//   缓存会在新消息到来时重建，不需要特意回放历史。
// ============================================================

package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

type CacheConsumer struct{}

func StartCacheConsumer(brokers []string, group, topic string) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	go func() {
		for {
			client, err := sarama.NewConsumerGroup(brokers, group, cfg)
			if err != nil {
				slog.Error("kafka_consumer_create_failed", "group", group, "err", err)
				time.Sleep(time.Second)
				continue
			}

			err = client.Consume(context.Background(), []string{topic}, &CacheConsumer{})
			client.Close()
			if err != nil {
				slog.Error("kafka_consume_error", "group", group, "err", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

func (c *CacheConsumer) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (c *CacheConsumer) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (c *CacheConsumer) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {

	for msg := range claim.Messages() {
		var km KafkaMessage
		if err := json.Unmarshal(msg.Value, &km); err != nil {
			slog.Error("kafka_decode_failed", "consumer", "cache", "err", err)
			continue
		}

		if err := cacheMessage(&km); err != nil {
			slog.Error("kafka_cache_failed", "msg_id", km.MsgId, "err", err)
			continue
		}

		sess.MarkMessage(msg, "")
	}

	return nil
}
