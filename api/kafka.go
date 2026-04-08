package api

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	KafkaBrokersEnv      = "KAFKA_BROKERS"
	MaxRetries           = 5
	InitialBackoffMillis = 100
	MaxBackoffMillis     = 5000
)

// KafkaEventPublisher publishes events to Kafka topics with retry and backoff support.
type KafkaEventPublisher struct {
	brokers []string
	writers map[string]*kafka.Writer
	mu      sync.Mutex
}

// NewKafkaEventPublisher creates a KafkaEventPublisher with the provided broker list.
func NewKafkaEventPublisher(brokers []string) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		brokers: brokers,
		writers: make(map[string]*kafka.Writer),
	}
}

// InitEventProducer initializes a Kafka event producer from environment variables.
// If KAFKA_BROKERS is not set, the default log-based publisher remains in use.
func InitEventProducer() {
	brokers := strings.Split(strings.TrimSpace(os.Getenv(KafkaBrokersEnv)), ",")
	if len(brokers) == 0 || brokers[0] == "" {
		return
	}

	SetEventProducer(NewKafkaEventPublisher(brokers))
}

// Publish writes the event to the configured Kafka topic with exponential backoff retry.
func (k *KafkaEventPublisher) Publish(topic string, key []byte, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   key,
		Value: payload,
		Time:  time.Now(),
	}

	return k.publishWithRetry(topic, msg)
}

func (k *KafkaEventPublisher) publishWithRetry(topic string, msg kafka.Message) error {
	writer := k.writer(topic)

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		err := writer.WriteMessages(context.Background(), msg)
		if err == nil {
			return nil
		}

		if attempt < MaxRetries {
			backoff := exponentialBackoff(attempt)
			time.Sleep(backoff)
		} else {
			return err
		}
	}

	return nil
}

func exponentialBackoff(attempt int) time.Duration {
	backoffMs := math.Min(
		float64(InitialBackoffMillis)*math.Pow(2, float64(attempt)),
		float64(MaxBackoffMillis),
	)
	return time.Duration(backoffMs) * time.Millisecond
}

func (k *KafkaEventPublisher) writer(topic string) *kafka.Writer {
	k.mu.Lock()
	defer k.mu.Unlock()

	if writer, ok := k.writers[topic]; ok {
		return writer
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:       k.brokers,
		Topic:         topic,
		Balancer:      &kafka.Hash{},
		MaxAttempts:   MaxRetries + 1,
		QueueCapacity: 100,
	})
	k.writers[topic] = writer
	return writer
}
