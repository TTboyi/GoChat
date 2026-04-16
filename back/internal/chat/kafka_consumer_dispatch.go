// ============================================================
// 文件：back/internal/chat/kafka_consumer_dispatch.go
// 作用：Kafka 消费者之一 —— "实时分发消费者"。
//       负责从 Kafka 队列读取消息，并立即推送给在线用户的 WebSocket 连接。
//
// 工作流程：
//   1. 从 Kafka 拿到一条消息（JSON 字节数组）
//   2. 反序列化为 KafkaMessage 结构体
//   3. 调用 dispatchKafkaMessage：
//      · 判断是私聊还是群聊（通过 receiveId 是否在 groupMembers 里判断）
//      · 私聊：把消息推给接收方 + 发送方（发送方需要看到"发送成功"确认）
//      · 群聊：遍历所有在线订阅了该群的用户，逐一推送
//   4. 调用 sess.MarkMessage(msg, "")，告诉 Kafka "这条消息我处理完了"
//
// 消费者组（ConsumerGroup）的作用：
//   消费者组名 "chat-dispatcher-debug-1" 是这个消费者组的唯一标识。
//   Kafka 会记录这个组消费到哪条消息了（叫做 Offset/偏移量）。
//   如果服务重启，消费者会从上次停止的位置继续，不会重复或遗漏。
//
// OffsetNewest 的含义：
//   这个消费者只处理"启动后新来的消息"，不回放历史消息。
//   理由：分发给在线用户只有实时意义，历史消息由另一个消费者（Persist）负责入库。
//
// 为什么要加重连循环（for { ... time.Sleep(3s) ... }）？
//   Kafka 服务偶尔会因为网络或重启而短暂不可用。
//   加上重连循环，即使临时断线，消费者也会在 3 秒后自动重连，不需要手动重启服务。
// ============================================================

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
