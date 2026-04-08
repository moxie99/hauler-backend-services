package api

import (
	"encoding/json"
	"log"
	"time"
)

// TrackingUpdatedEvent is emitted when order tracking information changes.
type TrackingUpdatedEvent struct {
	EventType           string    `json:"event_type"`
	OrderID             uint      `json:"order_id"`
	GeoCell             string    `json:"geo_cell"`
	Status              string    `json:"status"`
	DriverID            *uint     `json:"driver_id,omitempty"`
	FeeCents            *int      `json:"fee_cents,omitempty"`
	EstimatedTimeMins   *float64  `json:"estimated_time_mins,omitempty"`
	EstimatedDistanceKm *float64  `json:"estimated_distance_km,omitempty"`
	Timestamp           time.Time `json:"timestamp"`
}

// TrackingService consumes order status updates and emits tracking events.
type TrackingService struct {
	brokers  []string
	consumer *EventConsumer
}

// NewTrackingService creates a new TrackingService.
func NewTrackingService(brokers []string) *TrackingService {
	return &TrackingService{brokers: brokers}
}

// Start begins consuming status updates from the order topic.
func (ts *TrackingService) Start() error {
	ts.consumer = NewEventConsumer(ts.brokers, OrderEventsTopic, TrackingConsumerGroup)
	return ts.consumer.Start(ts.handleOrderEvent)
}

// Stop halts the tracking service.
func (ts *TrackingService) Stop() error {
	if ts.consumer != nil {
		return ts.consumer.Stop()
	}
	return nil
}

func (ts *TrackingService) handleOrderEvent(key []byte, value []byte) error {
	eventType, err := EventType(value)
	if err != nil {
		log.Printf("[Tracking] failed to parse event envelope: %v", err)
		return nil
	}

	if eventType != OrderEventTypeStatusUpdated {
		return nil
	}

	var statusEvent OrderStatusUpdatedEvent
	if err := json.Unmarshal(value, &statusEvent); err != nil {
		log.Printf("[Tracking] failed to unmarshal OrderStatusUpdated event: %v", err)
		return nil
	}

	trackingEvent := TrackingUpdatedEvent{
		EventType:           OrderEventTypeTrackingUpdated,
		OrderID:             statusEvent.OrderID,
		GeoCell:             statusEvent.GeoCell,
		Status:              string(statusEvent.Status),
		DriverID:            statusEvent.DriverID,
		FeeCents:            statusEvent.FeeCents,
		EstimatedTimeMins:   statusEvent.EstimatedTimeMins,
		EstimatedDistanceKm: statusEvent.EstimatedDistanceKm,
		Timestamp:           time.Now(),
	}

	PublishEventWithKey(TrackingEventsTopic, []byte(statusEvent.GeoCell), trackingEvent)
	log.Printf("[Tracking] Emitted tracking update for order %d status %s", statusEvent.OrderID, statusEvent.Status)
	return nil
}
