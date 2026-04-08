package api

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	// Event topics
	OrderEventsTopic    = "order-events"
	DispatchEventsTopic = "dispatch-events"
	TrackingEventsTopic = "tracking-events"
	PricingEventsTopic  = "pricing-events"
	NotificationsTopic  = "notifications"

	// Consumer group IDs
	DispatchConsumerGroup      = "dispatch-service"
	TrackingConsumerGroup      = "tracking-service"
	PricingConsumerGroup       = "pricing-service"
	NotificationsConsumerGroup = "notifications-service"
)

// TopicConfig defines the Kafka topics used by the application.
type TopicConfig struct {
	Name              string
	Partitions        int
	ReplicationFactor int
}

// DefaultTopics returns the default topic configuration.
func DefaultTopics() []TopicConfig {
	return []TopicConfig{
		{Name: OrderEventsTopic, Partitions: 10, ReplicationFactor: 3},
		{Name: DispatchEventsTopic, Partitions: 10, ReplicationFactor: 3},
		{Name: TrackingEventsTopic, Partitions: 10, ReplicationFactor: 3},
		{Name: PricingEventsTopic, Partitions: 10, ReplicationFactor: 3},
		{Name: NotificationsTopic, Partitions: 5, ReplicationFactor: 3},
	}
}

// EventConsumer consumes events from a Kafka topic.
type EventConsumer struct {
	brokers       []string
	topic         string
	consumerGroup string
	reader        *kafka.Reader
	mu            sync.Mutex
	running       bool
	cancelFunc    context.CancelFunc
}

// NewEventConsumer creates a new EventConsumer.
func NewEventConsumer(brokers []string, topic, consumerGroup string) *EventConsumer {
	return &EventConsumer{
		brokers:       brokers,
		topic:         topic,
		consumerGroup: consumerGroup,
	}
}

// Start begins consuming events from the topic.
func (c *EventConsumer) Start(handler func([]byte, []byte) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	c.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		Topic:          c.topic,
		GroupID:        c.consumerGroup,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Duration(0),
	})

	c.running = true

	go c.consume(ctx, handler)

	return nil
}

// Stop halts the event consumer.
func (c *EventConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	c.running = false

	if c.reader != nil {
		return c.reader.Close()
	}

	return nil
}

func (c *EventConsumer) consume(ctx context.Context, handler func([]byte, []byte) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Printf("error fetching message from %s: %v", c.topic, err)
			continue
		}

		if err := handler(msg.Key, msg.Value); err != nil {
			log.Printf("error processing message from %s: %v", c.topic, err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("error committing message for %s: %v", c.topic, err)
		}
	}
}
