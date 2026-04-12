package producer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewProducer(brokers string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers),
		Topic:    "rate-limit-events",
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaProducer{writer: writer}
}

func (kp *KafkaProducer) SendEvent(eventType string, data map[string]string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := map[string]interface{}{
		"event_type": eventType,
		"data":       data,
		"timestamp":  time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	msg := kafka.Message{
		Value: payload,
	}

	if err := kp.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("Failed to write message to Kafka: %v", err)
	}
}

func (kp *KafkaProducer) Close() error {
	return kp.writer.Close()
}
