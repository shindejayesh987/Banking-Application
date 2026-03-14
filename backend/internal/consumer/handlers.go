package consumer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// TransactionLogger logs all transaction events to stdout.
// In production, this would write to an audit table or external analytics system.
func TransactionLogger() Handler {
	return func(ctx context.Context, msg kafka.Message) error {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[txn-logger] invalid JSON at offset %d: %v", msg.Offset, err)
			return nil // don't retry malformed messages
		}

		log.Printf("[txn-logger] topic=%s partition=%d offset=%d type=%v",
			msg.Topic, msg.Partition, msg.Offset, event["type"])
		return nil
	}
}

// AccountEventHandler processes account lifecycle events.
// Useful for triggering notifications, updating caches, etc.
func AccountEventHandler() Handler {
	return func(ctx context.Context, msg kafka.Message) error {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[account-events] invalid JSON at offset %d: %v", msg.Offset, err)
			return nil
		}

		log.Printf("[account-events] topic=%s partition=%d offset=%d type=%v",
			msg.Topic, msg.Partition, msg.Offset, event["type"])
		return nil
	}
}
