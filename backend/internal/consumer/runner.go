package consumer

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// Handler processes a single Kafka message.
type Handler func(ctx context.Context, msg kafka.Message) error

// Runner manages a Kafka consumer group for a single topic.
type Runner struct {
	reader  *kafka.Reader
	handler Handler
	topic   string
}

func NewRunner(brokers []string, topic, groupID string, handler Handler) *Runner {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	return &Runner{
		reader:  reader,
		handler: handler,
		topic:   topic,
	}
}

// Run starts consuming messages. Blocks until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) {
	log.Printf("[consumer:%s] started", r.topic)
	for {
		msg, err := r.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("[consumer:%s] shutting down", r.topic)
				return
			}
			log.Printf("[consumer:%s] fetch error: %v", r.topic, err)
			continue
		}

		if err := r.handler(ctx, msg); err != nil {
			log.Printf("[consumer:%s] handler error (offset=%d): %v", r.topic, msg.Offset, err)
			// Don't commit — message will be retried
			continue
		}

		if err := r.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[consumer:%s] commit error: %v", r.topic, err)
		}
	}
}

// Close shuts down the reader.
func (r *Runner) Close() error {
	return r.reader.Close()
}
