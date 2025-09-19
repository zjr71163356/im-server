package broker

import (
	"context"
	"time"

	"im-server/pkg/config"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer      *kafka.Writer
	topicPrefix string
}

func NewKafkaProducer(cfg config.BrokerConfig) *KafkaProducer {
	brokers := cfg.KafkaBrokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
		},
		topicPrefix: cfg.TopicPrefix,
	}
}

func (p *KafkaProducer) Topic(name string) string {
	if p.topicPrefix == "" {
		return name
	}
	return p.topicPrefix + "." + name
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	})
}

func (p *KafkaProducer) Close() error { return p.writer.Close() }

// KafkaConsumer 简单消费者封装
// 使用 groupID 订阅若干 topic，并把消息交给 handler 处理。
type KafkaConsumer struct {
	reader *kafka.Reader
}

type MessageHandler func(ctx context.Context, m kafka.Message) error

func NewKafkaConsumer(cfg config.BrokerConfig, groupID string, topics ...string) *KafkaConsumer {
	brokers := cfg.KafkaBrokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupID:     groupID,
			GroupTopics: topics,
		}),
	}
}

func (c *KafkaConsumer) Start(ctx context.Context, handler MessageHandler) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		if handler != nil {
			_ = handler(ctx, m)
		}
	}
}

func (c *KafkaConsumer) Close() error { return c.reader.Close() }
