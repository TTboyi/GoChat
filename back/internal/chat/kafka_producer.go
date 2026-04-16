// ============================================================
// 文件：back/internal/chat/kafka_producer.go
// 作用：封装 Kafka 生产者，负责把用户发出的消息"写入"消息队列。
//
// 什么是 Kafka 生产者？
//   在 Kafka 里，生产者（Producer）负责把消息发布到某个"主题"（Topic）。
//   就像一个邮局的投递窗口：你把信（消息）交给窗口，窗口负责放进对应的邮箱（Topic）。
//
// KafkaProducer 结构体：
//   producer - sarama 库的 SyncProducer，sarama 是 Kafka 的 Go 客户端库
//   topic    - 消息要发布到哪个主题（频道名）
//
// InitKafkaProducer 的三项配置解读：
//   Return.Successes = true    → 每次 SendMessage 都等待 Kafka 确认收到，才返回成功。
//                                这叫"同步发送"，可靠但比异步稍慢。
//   RequiredAcks = WaitForAll  → 要求所有副本（备份节点）都写入后才确认。
//                                最高可靠性级别，防止 leader 宕机丢消息。
//   Retry.Max = 3              → 发送失败最多重试 3 次，应对网络抖动。
//
// Publish 方法的核心逻辑：
//   1. 从 DB 查出发送者的昵称和头像（因为前端需要展示这些信息）
//   2. 构造 KafkaMessage（包含消息ID、内容、时间戳等完整元数据）
//   3. 用 JSON 序列化成字节数组
//   4. 把 receiveId 作为 Kafka 消息的 Key：这样保证"同一个会话"的消息
//      总是被同一个消费者分区处理，从而保证消息的顺序性
// ============================================================

package chat

import (
	"encoding/json"
	"log"
	"time"

	"chatapp/back/internal/config"
	"chatapp/back/internal/model"

	"github.com/IBM/sarama"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
}

var ChatKafkaProducer *KafkaProducer

func InitKafkaProducer(brokers []string, topic string) error {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 3

	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return err
	}

	ChatKafkaProducer = &KafkaProducer{
		producer: p,
		topic:    topic,
	}
	log.Printf("✅ Kafka Producer 已启动 topic=%s", topic)
	return nil
}

func (kp *KafkaProducer) Publish(env ChatEnvelope) {
	if kp == nil {
		return
	}

	// 查询发送者信息
	var senderName, senderAvatar string
	db := config.GetDB()
	var u model.UserInfo
	if err := db.Where("uuid = ?", env.SendId).First(&u).Error; err == nil {
		senderName = u.Nickname
		senderAvatar = u.Avatar
	}

	km := KafkaMessage{
		MsgId:      newIDWithPrefix("M"),
		LocalId:    env.LocalId,
		Type:       env.Type,
		SendId:     env.SendId,
		SendName:   nz(senderName, "用户"),
		SendAvatar: senderAvatar,
		ReceiveId:  env.ReceiveId,
		Content:    env.Content,
		Url:        env.Url,
		FileName:   env.FileName,
		FileType:   env.FileType,
		FileSize:   env.FileSize,
		CreatedAt:  time.Now().Unix(),
	}

	raw, err := json.Marshal(km)
	if err != nil {
		log.Printf("❌ Kafka marshal error: %v", err)
		return
	}

	_, _, err = kp.producer.SendMessage(&sarama.ProducerMessage{
		Topic: kp.topic,
		Key:   sarama.StringEncoder(env.ReceiveId), // 保证同会话有序
		Value: sarama.ByteEncoder(raw),
	})

	if err != nil {
		log.Printf("❌ Kafka send error: %v", err)
	}
}
