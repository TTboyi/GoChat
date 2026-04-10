package chat

import (
	"context"
	"encoding/json"
	"log"
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
				log.Printf("❌ Cache consumer create failed: %v", err)
				time.Sleep(time.Second)
				continue
			}

			err = client.Consume(context.Background(), []string{topic}, &CacheConsumer{})
			client.Close()
			if err != nil {
				log.Printf("❌ Cache consume error: %v", err)
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
			log.Printf("❌ Cache decode failed: %v", err)
			continue
		}

		if err := cacheMessage(&km); err != nil {
			log.Printf("❌ Cache failed msgId=%s err=%v", km.MsgId, err)
			continue
		}

		sess.MarkMessage(msg, "")
	}

	return nil
}
