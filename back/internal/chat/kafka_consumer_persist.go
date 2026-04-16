// ============================================================
// 文件：back/internal/chat/kafka_consumer_persist.go
// 作用：Kafka 消费者之二 —— "持久化消费者"。
//       负责把 Kafka 队列里的消息写入 MySQL 数据库，保证历史记录永久保存。
//
// 与 Dispatcher 消费者的关键区别：
//   Dispatcher 用 OffsetNewest（只看新消息），是因为推送只有实时意义。
//   Persist 用 OffsetOldest（从头开始，或从上次提交的位置继续），
//   是因为数据库必须完整、不能有遗漏。如果服务重启，Persist 会从上次
//   成功入库的位置继续，确保"每条消息都被写进数据库至少一次"。
//
// 幂等性设计（persistMessage 里实现）：
//   如果因为重启等原因同一条消息被消费两次，第二次写库前会先查询
//   "uuid 是否已存在"，存在则跳过，不会重复插入。
//   这叫"幂等性"：多次执行和执行一次的结果完全相同。
//
// 不 Mark 就是不确认，Kafka 会重试：
//   如果 persistMessage 失败，代码不调用 sess.MarkMessage，
//   Kafka 下次还会重新投递这条消息，直到成功为止。
//   这是"至少一次投递"（at-least-once delivery）语义。
// ============================================================

package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"chatapp/back/internal/config"

	"github.com/IBM/sarama"
)

type PersistConsumer struct{}

func StartPersistConsumer(brokers []string, group, topic string) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest // ✅ DB 要可回放

	go func() {
		for {
			client, err := sarama.NewConsumerGroup(brokers, group, cfg)
			if err != nil {
				slog.Error("kafka_consumer_create_failed", "group", group, "err", err)
				time.Sleep(3 * time.Second)
				continue
			}
			err = client.Consume(context.Background(), []string{topic}, &PersistConsumer{})
			client.Close()
			if err != nil {
				slog.Error("kafka_consume_error", "group", group, "err", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

func (c *PersistConsumer) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (c *PersistConsumer) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (c *PersistConsumer) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {

	db := config.GetDB()

	for msg := range claim.Messages() {
		var km KafkaMessage
		if err := json.Unmarshal(msg.Value, &km); err != nil {
			slog.Error("kafka_decode_failed", "consumer", "persist", "err", err)
			continue
		}

		if err := persistMessage(db, &km); err != nil {
			slog.Error("kafka_persist_failed", "msg_id", km.MsgId, "err", err)
			// ❗不 Mark，让 Kafka 重试
			continue
		}

		sess.MarkMessage(msg, "")
	}

	return nil
}
