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
