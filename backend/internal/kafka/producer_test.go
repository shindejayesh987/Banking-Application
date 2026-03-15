package kafka

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvent_JSONSerialization(t *testing.T) {
	e := Event{
		Type:      "account.created",
		Payload:   map[string]interface{}{"account_id": "acc-1", "balance": 0},
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal Event: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}

	if decoded["type"] != "account.created" {
		t.Errorf("type: got %v, want account.created", decoded["type"])
	}
	if decoded["timestamp"] == nil {
		t.Error("timestamp should be present in JSON")
	}
	payload, ok := decoded["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("payload should be a map")
	}
	if payload["account_id"] != "acc-1" {
		t.Errorf("payload account_id: got %v, want acc-1", payload["account_id"])
	}
}

func TestEvent_NilPayload(t *testing.T) {
	e := Event{
		Type:    "test.event",
		Payload: nil,
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal nil payload: %v", err)
	}
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Type != "test.event" {
		t.Errorf("type: got %q, want test.event", decoded.Type)
	}
}

func TestTopicConstants(t *testing.T) {
	if TopicTransactions != "transactions" {
		t.Errorf("TopicTransactions: got %q, want transactions", TopicTransactions)
	}
	if TopicAccountEvents != "account-events" {
		t.Errorf("TopicAccountEvents: got %q, want account-events", TopicAccountEvents)
	}
}
