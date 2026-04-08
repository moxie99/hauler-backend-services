package api

import (
	"encoding/json"
	"log"
)

// EventPublisher defines the contract for publishing domain events.
type EventPublisher interface {
	Publish(topic string, key []byte, event any) error
}

// EventProducer is the global event publisher used by the application.
// It can be replaced with a Kafka producer or another streaming backend.
var EventProducer EventPublisher = &LogEventPublisher{}

// SetEventProducer configures the global event publisher.
func SetEventProducer(producer EventPublisher) {
	EventProducer = producer
}

// PublishEvent publishes an event without a message key.
func PublishEvent(topic string, event any) {
	PublishEventWithKey(topic, nil, event)
}

// PublishEventWithKey publishes an event with an optional message key.
func PublishEventWithKey(topic string, key []byte, event any) {
	if EventProducer == nil {
		EventProducer = &LogEventPublisher{}
	}

	if err := EventProducer.Publish(topic, key, event); err != nil {
		log.Printf("failed to publish event to topic %s: %v", topic, err)
	}
}

// EventType returns the event_type field from a raw JSON event payload.
func EventType(value []byte) (string, error) {
	var envelope struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return "", err
	}
	return envelope.EventType, nil
}

// LogEventPublisher is a simple fallback publisher that logs event payloads.
type LogEventPublisher struct{}

// Publish logs the event payload to stdout.
func (l *LogEventPublisher) Publish(topic string, key []byte, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if key != nil {
		log.Printf("[EVENT] topic=%s key=%s payload=%s", topic, string(key), string(payload))
	} else {
		log.Printf("[EVENT] topic=%s payload=%s", topic, string(payload))
	}

	return nil
}
