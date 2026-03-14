package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicTransactions  = "transactions"
	TopicAccountEvents = "account-events"
)

type Event struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(writer *kafka.Writer) *Producer {
	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, topic, key string, event Event) error {
	event.Timestamp = time.Now()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	})
	if err != nil {
		log.Printf("kafka publish error (topic=%s): %v", topic, err)
		return fmt.Errorf("publish to %s: %w", topic, err)
	}
	return nil
}
