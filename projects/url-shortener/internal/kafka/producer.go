package kafka

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

type RedirectEvent struct {
	ShortCode string `json:"short_code"`
	LongURL   string `json:"long_url"`
	Timestamp string `json:"timestamp"`
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
}

type Producer struct {
	writer *kafka.Writer
	topic  string
}

func NewProducer() *Producer {
	broker := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")

	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.Hash{},    // same key always goes to same partition
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,             // fire and forget, never block the redirect
		ErrorLogger:  kafka.LoggerFunc(func(msg string, args ...interface{}) {}),
	}

	return &Producer{
		writer: writer,
		topic:  topic,
	}
}

func (p *Producer) PublishRedirectEvent(ctx context.Context, event RedirectEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ShortCode), // partition key, same short code = same partition
		Value: payload,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}