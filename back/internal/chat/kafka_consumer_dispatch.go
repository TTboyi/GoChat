package chat

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

type DispatcherConsumer struct {
	group string
	topic string
}

func StartDispatcherConsumer(brokers []string, group, topic string) {
	log.Println("🚨 StartDispatcherConsumer CALLED")

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
				log.Printf("❌ create dispatcher consumer failed: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}

			log.Println("🟡 Dispatcher Consumer created")

			err = client.Consume(context.Background(), []string{topic}, consumer)
			client.Close()
			if err != nil {
				log.Printf("❌ dispatcher consume error: %v", err)
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

	log.Printf("🟡 Dispatcher ConsumeClaim START")

	for msg := range claim.Messages() {
		log.Printf("🟢 Dispatcher got kafka msg key=%s valueLen=%d",
			string(msg.Key),
			len(msg.Value),
		)

		var km KafkaMessage
		if err := json.Unmarshal(msg.Value, &km); err != nil {
			log.Printf("❌ Dispatcher decode failed: %v", err)
			continue
		}

		log.Printf("🟢 Dispatcher decoded send=%s recv=%s content=%q",
			km.SendId, km.ReceiveId, km.Content,
		)

		dispatchKafkaMessage(&km)

		sess.MarkMessage(msg, "")
	}

	return nil
}
