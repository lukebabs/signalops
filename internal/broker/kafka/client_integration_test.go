package kafka

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/pkg/broker"
)

func TestPublishConsumeCommitAgainstRedpanda(t *testing.T) {
	if os.Getenv("SIGNALOPS_BROKER_INTEGRATION") != "1" {
		t.Skip("set SIGNALOPS_BROKER_INTEGRATION=1 to run Redpanda integration test")
	}

	brokers := strings.Split(envOrDefault("SIGNALOPS_BROKER_BROKERS", "localhost:19092"), ",")
	topic := broker.TopicName(envOrDefault("SIGNALOPS_ENV", broker.DefaultEnvironment), broker.RawTopic)
	key := "g008-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	client, err := NewClient(Config{Brokers: brokers, ClientID: "signalops-g008-test"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() {
		if err := client.Close(context.Background()); err != nil {
			t.Fatalf("client Close() error = %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := client.Publish(ctx, broker.Message{
		Topic:         topic,
		Key:           key,
		Value:         []byte(`{"gate":"G008","status":"integration"}`),
		Headers:       map[string]string{"content_type": "application/json"},
		CorrelationID: "g008-correlation",
		CausationID:   "g008-causation",
		TraceID:       "g008-trace",
		PublishedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if result.Topic != topic {
		t.Fatalf("Publish() topic = %q, want %q", result.Topic, topic)
	}
	if result.Offset < 0 {
		t.Fatalf("Publish() offset = %d, want non-negative", result.Offset)
	}

	consumer, err := client.NewConsumer("signalops-g008-"+key, []string{topic})
	if err != nil {
		t.Fatalf("NewConsumer() error = %v", err)
	}
	defer func() {
		if err := consumer.Close(context.Background()); err != nil {
			t.Fatalf("consumer Close() error = %v", err)
		}
	}()

	for {
		msg, err := consumer.Consume(ctx)
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}
		if msg.Key != key {
			continue
		}

		if string(msg.Value) != `{"gate":"G008","status":"integration"}` {
			t.Fatalf("Value = %q", string(msg.Value))
		}
		if msg.CorrelationID != "g008-correlation" {
			t.Fatalf("CorrelationID = %q", msg.CorrelationID)
		}
		if msg.CausationID != "g008-causation" {
			t.Fatalf("CausationID = %q", msg.CausationID)
		}
		if msg.TraceID != "g008-trace" {
			t.Fatalf("TraceID = %q", msg.TraceID)
		}
		if err := consumer.Commit(ctx, msg); err != nil {
			t.Fatalf("Commit() error = %v", err)
		}
		return
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
