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
