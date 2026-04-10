package chat

import (
	"context"
	"encoding/json"
	"log"
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
				log.Printf("❌ Persist consumer create failed: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
			err = client.Consume(context.Background(), []string{topic}, &PersistConsumer{})
			client.Close()
			if err != nil {
				log.Printf("❌ Persist consume error: %v", err)
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
			log.Printf("❌ Persist decode failed: %v", err)
			continue
		}

		if err := persistMessage(db, &km); err != nil {
			log.Printf("❌ Persist failed msgId=%s err=%v", km.MsgId, err)
			// ❗不 Mark，让 Kafka 重试
			continue
		}

		sess.MarkMessage(msg, "")
	}

	return nil
}
