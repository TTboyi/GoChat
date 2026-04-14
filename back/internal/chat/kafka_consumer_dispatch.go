package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

type DispatcherConsumer struct {
	group string
	topic string
}

func StartDispatcherConsumer(brokers []string, group, topic string) {
	slog.Info("kafka_dispatcher_start", "group", group, "topic", topic)

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer := &DispatcherConsumer{
		group: group,
		topic: topic,
	}

	go func() {
		for {
			client, err := sarama.NewConsumerGroup(brokers, group, cfg)
			if err != nil {
				slog.Error("kafka_consumer_create_failed", "group", group, "err", err)
				time.Sleep(3 * time.Second)
				continue
			}

			slog.Info("kafka_consumer_ready", "group", group)

			err = client.Consume(context.Background(), []string{topic}, consumer)
			client.Close()
			if err != nil {
				slog.Error("kafka_consume_error", "group", group, "err", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()

}

func (c *DispatcherConsumer) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (c *DispatcherConsumer) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (c *DispatcherConsumer) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {

	slog.Info("kafka_claim_start", "group", c.group, "topic", c.topic)

	for msg := range claim.Messages() {
		var km KafkaMessage
		if err := json.Unmarshal(msg.Value, &km); err != nil {
			slog.Error("kafka_decode_failed", "group", c.group, "err", err)
			continue
		}

		slog.Info("kafka_dispatch", "send_id", km.SendId, "recv_id", km.ReceiveId, "type", km.Type)

		dispatchKafkaMessage(&km)

		sess.MarkMessage(msg, "")
	}

	return nil
}
