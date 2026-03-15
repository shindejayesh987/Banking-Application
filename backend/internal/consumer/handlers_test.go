package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/segmentio/kafka-go"
)

func makeMessage(topic string, offset int64, value []byte) kafka.Message {
	return kafka.Message{
		Topic:     topic,
		Partition: 0,
		Offset:    offset,
		Value:     value,
	}
}

func TestTransactionLogger_ValidMessage(t *testing.T) {
	handler := TransactionLogger()
	payload, _ := json.Marshal(map[string]interface{}{
		"type":    "transaction.deposit",
		"payload": map[string]interface{}{"account_id": "acc-1", "amount": 1000},
	})
	msg := makeMessage("transactions", 42, payload)

	err := handler(context.Background(), msg)
	if err != nil {
		t.Errorf("TransactionLogger: unexpected error: %v", err)
	}
}

func TestTransactionLogger_InvalidJSON(t *testing.T) {
	handler := TransactionLogger()
	msg := makeMessage("transactions", 1, []byte("not json"))

	// Should return nil (don't retry malformed messages)
	err := handler(context.Background(), msg)
	if err != nil {
		t.Errorf("TransactionLogger with bad JSON: want nil, got %v", err)
	}
}

func TestTransactionLogger_EmptyPayload(t *testing.T) {
	handler := TransactionLogger()
	msg := makeMessage("transactions", 2, []byte("{}"))

	err := handler(context.Background(), msg)
	if err != nil {
		t.Errorf("TransactionLogger with empty JSON: unexpected error: %v", err)
	}
}

func TestAccountEventHandler_ValidMessage(t *testing.T) {
	handler := AccountEventHandler()
	payload, _ := json.Marshal(map[string]interface{}{
		"type":    "account.created",
		"payload": map[string]interface{}{"account_id": "acc-2"},
	})
	msg := makeMessage("account-events", 10, payload)

	err := handler(context.Background(), msg)
	if err != nil {
		t.Errorf("AccountEventHandler: unexpected error: %v", err)
	}
}

func TestAccountEventHandler_InvalidJSON(t *testing.T) {
	handler := AccountEventHandler()
	msg := makeMessage("account-events", 5, []byte("{bad json"))

	err := handler(context.Background(), msg)
	if err != nil {
		t.Errorf("AccountEventHandler with bad JSON: want nil, got %v", err)
	}
}

func TestHandlers_ReturnFunctionsNotNil(t *testing.T) {
	if TransactionLogger() == nil {
		t.Error("TransactionLogger() returned nil")
	}
	if AccountEventHandler() == nil {
		t.Error("AccountEventHandler() returned nil")
	}
}
