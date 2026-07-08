package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/pkg/broker"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	defaultClientID = "signalops"

	headerCorrelationID = "correlation_id"
	headerCausationID   = "causation_id"
	headerTraceID       = "trace_id"
)

var (
	ErrMissingBrokers = errors.New("kafka broker list is required")
	ErrMissingTopic   = errors.New("broker message topic is required")
)

// Config contains Kafka-compatible client settings.
type Config struct {
	Brokers  []string
	ClientID string
}

// Client implements the SignalOps broker boundary with a Kafka-compatible
// franz-go client.
type Client struct {
	client   *kgo.Client
	brokers  []string
	clientID string
}

// NewClient creates a Kafka-compatible SignalOps broker client.
func NewClient(cfg Config) (*Client, error) {
	brokers := cleanBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, ErrMissingBrokers
	}

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = defaultClientID
	}

	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka client: %w", err)
	}

	return &Client{
		client:   cl,
		brokers:  brokers,
		clientID: clientID,
	}, nil
}

// Publish synchronously publishes a durable broker message.
func (c *Client) Publish(ctx context.Context, msg broker.Message) (broker.PublishResult, error) {
	if strings.TrimSpace(msg.Topic) == "" {
		return broker.PublishResult{}, ErrMissingTopic
	}

	record := &kgo.Record{
		Topic:     msg.Topic,
		Key:       []byte(msg.Key),
		Value:     msg.Value,
		Headers:   toRecordHeaders(msg),
		Timestamp: publishedAtOrNow(msg.PublishedAt),
	}

	results := c.client.ProduceSync(ctx, record)
	produced, err := results.First()
	if err != nil {
		return broker.PublishResult{}, fmt.Errorf("publish kafka record: %w", err)
	}

	return broker.PublishResult{
		Topic:     produced.Topic,
		Partition: produced.Partition,
		Offset:    produced.Offset,
	}, nil
}

// NewConsumer creates a manual-commit durable consumer.
func (c *Client) NewConsumer(groupID string, topics []string) (broker.Consumer, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("consumer group id is required")
	}

	cleanTopics := cleanTopics(topics)
	if len(cleanTopics) == 0 {
		return nil, errors.New("consumer topic list is required")
	}

	cl, err := kgo.NewClient(
		kgo.SeedBrokers(c.brokers...),
		kgo.ClientID(c.clientID+"-consumer"),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(cleanTopics...),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	return &Consumer{client: cl}, nil
}

// Close closes the producer client.
func (c *Client) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.client.Close()
	return nil
}

// Consumer implements manual-commit consumption.
type Consumer struct {
	client *kgo.Client
	buffer []*kgo.Record
}

// Consume returns the next available durable message.
func (c *Consumer) Consume(ctx context.Context) (broker.ConsumedMessage, error) {
	for {
		if len(c.buffer) > 0 {
			record := c.buffer[0]
			c.buffer = c.buffer[1:]
			return fromRecord(record), nil
		}

		fetches := c.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			// The reset is already applied; continue polling from the broker-selected recovery offset.
			var dataLoss *kgo.ErrDataLoss
			if errors.As(err, &dataLoss) {
				continue
			}
			return broker.ConsumedMessage{}, fmt.Errorf("poll kafka fetches: %w", err)
		}

		c.buffer = fetches.Records()
	}
}

// Commit manually commits the consumed message offset.
func (c *Consumer) Commit(ctx context.Context, msg broker.ConsumedMessage) error {
	record := &kgo.Record{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	}
	if err := c.client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit kafka record: %w", err)
	}
	return nil
}

// Close leaves the consumer group and closes the client.
func (c *Consumer) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.client.Close()
	return nil
}

func cleanBrokers(brokers []string) []string {
	cleaned := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			cleaned = append(cleaned, broker)
		}
	}
	return cleaned
}

func cleanTopics(topics []string) []string {
	cleaned := make([]string, 0, len(topics))
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic != "" {
			cleaned = append(cleaned, topic)
		}
	}
	return cleaned
}

func publishedAtOrNow(publishedAt time.Time) time.Time {
	if publishedAt.IsZero() {
		return time.Now().UTC()
	}
	return publishedAt
}

func toRecordHeaders(msg broker.Message) []kgo.RecordHeader {
	headers := make([]kgo.RecordHeader, 0, len(msg.Headers)+3)
	for key, value := range msg.Headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		headers = append(headers, kgo.RecordHeader{Key: key, Value: []byte(value)})
	}

	headers = appendSignalHeader(headers, headerCorrelationID, msg.CorrelationID)
	headers = appendSignalHeader(headers, headerCausationID, msg.CausationID)
	headers = appendSignalHeader(headers, headerTraceID, msg.TraceID)
	return headers
}

func appendSignalHeader(headers []kgo.RecordHeader, key, value string) []kgo.RecordHeader {
	if value == "" {
		return headers
	}
	return append(headers, kgo.RecordHeader{Key: key, Value: []byte(value)})
}

func fromRecord(record *kgo.Record) broker.ConsumedMessage {
	headers := make(map[string]string, len(record.Headers))
	for _, header := range record.Headers {
		headers[header.Key] = string(header.Value)
	}

	return broker.ConsumedMessage{
		Message: broker.Message{
			Topic:         record.Topic,
			Key:           string(record.Key),
			Value:         record.Value,
			Headers:       headers,
			CorrelationID: headers[headerCorrelationID],
			CausationID:   headers[headerCausationID],
			TraceID:       headers[headerTraceID],
			PublishedAt:   record.Timestamp,
		},
		Partition: record.Partition,
		Offset:    record.Offset,
	}
}
