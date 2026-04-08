package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// NotificationEvent is emitted to the notifications topic for downstream delivery.
type NotificationEvent struct {
	EventType   string         `json:"event_type"`
	OrderID     uint           `json:"order_id"`
	RecipientID uint           `json:"recipient_id"`
	Channel     string         `json:"channel"`
	Subject     string         `json:"subject"`
	Message     string         `json:"message"`
	Payload     map[string]any `json:"payload,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

// NotificationService consumes relevant events and publishes notifications.
type NotificationService struct {
	brokers   []string
	consumers []*EventConsumer
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(brokers []string) *NotificationService {
	return &NotificationService{brokers: brokers}
}

// Start begins consuming events for notifications.
func (ns *NotificationService) Start() error {
	topics := []struct {
		topic   string
		handler func([]byte, []byte) error
	}{
		{OrderEventsTopic, ns.handleOrderEvents},
		{PricingEventsTopic, ns.handlePricingEvents},
		{DispatchEventsTopic, ns.handleDispatchEvents},
		{TrackingEventsTopic, ns.handleTrackingEvents},
	}

	for _, t := range topics {
		consumer := NewEventConsumer(ns.brokers, t.topic, NotificationsConsumerGroup)
		if err := consumer.Start(t.handler); err != nil {
			return err
		}
		ns.consumers = append(ns.consumers, consumer)
	}

	return nil
}

// Stop halts the notification service.
func (ns *NotificationService) Stop() error {
	var lastErr error
	for _, consumer := range ns.consumers {
		if err := consumer.Stop(); err != nil {
			log.Printf("[Notifications] failed to stop consumer for topic %s: %v", consumer.topic, err)
			lastErr = err
		}
	}
	return lastErr
}

func (ns *NotificationService) handleOrderEvents(key []byte, value []byte) error {
	eventType, err := EventType(value)
	if err != nil {
		log.Printf("[Notifications] failed to parse order event envelope: %v", err)
		return nil
	}

	switch eventType {
	case OrderEventTypePlaced:
		return ns.handleOrderPlaced(value)
	case OrderEventTypeStatusUpdated:
		return ns.handleStatusUpdated(value)
	default:
		return nil
	}
}

func (ns *NotificationService) handlePricingEvents(key []byte, value []byte) error {
	var pricingEvent PricingCalculatedEvent
	if err := json.Unmarshal(value, &pricingEvent); err != nil {
		log.Printf("[Notifications] failed to unmarshal PricingCalculated event: %v", err)
		return nil
	}

	recipientID, err := ns.findOrderCustomer(pricingEvent.OrderID)
	if err != nil {
		log.Printf("[Notifications] failed to load customer for order %d: %v", pricingEvent.OrderID, err)
		return nil
	}

	notification := NotificationEvent{
		EventType:   "PricingNotification",
		OrderID:     pricingEvent.OrderID,
		RecipientID: recipientID,
		Channel:     "email",
		Subject:     "Pricing calculated for your order",
		Message:     fmt.Sprintf("Your order pricing is ready. Fee: %d %s. ETA: %.0f mins.", pricingEvent.FeeCents, pricingEvent.Currency, pricingEvent.EstimatedTimeMins),
		Payload: map[string]any{
			"fee_cents":             pricingEvent.FeeCents,
			"currency":              pricingEvent.Currency,
			"estimated_time_mins":   pricingEvent.EstimatedTimeMins,
			"estimated_distance_km": pricingEvent.EstimatedDistanceKm,
		},
		Timestamp: time.Now(),
	}

	PublishEventWithKey(NotificationsTopic, []byte(pricingEvent.GeoCell), notification)
	log.Printf("[Notifications] published pricing notification for order %d", pricingEvent.OrderID)
	return nil
}

func (ns *NotificationService) handleDispatchEvents(key []byte, value []byte) error {
	var dispatchEvent DriverAssignedEvent
	if err := json.Unmarshal(value, &dispatchEvent); err != nil {
		log.Printf("[Notifications] failed to unmarshal DriverAssigned event: %v", err)
		return nil
	}

	recipientID, err := ns.findOrderCustomer(dispatchEvent.OrderID)
	if err != nil {
		log.Printf("[Notifications] failed to load customer for order %d: %v", dispatchEvent.OrderID, err)
		return nil
	}

	notification := NotificationEvent{
		EventType:   "DispatchNotification",
		OrderID:     dispatchEvent.OrderID,
		RecipientID: recipientID,
		Channel:     "sms",
		Subject:     "Driver assigned",
		Message:     fmt.Sprintf("Driver %d has been assigned to your order.", dispatchEvent.DriverID),
		Payload: map[string]any{
			"driver_id":        dispatchEvent.DriverID,
			"assignment_score": dispatchEvent.AssignmentScore,
		},
		Timestamp: time.Now(),
	}

	PublishEventWithKey(NotificationsTopic, []byte(dispatchEvent.GeoCell), notification)
	log.Printf("[Notifications] published driver assignment notification for order %d", dispatchEvent.OrderID)
	return nil
}

func (ns *NotificationService) handleTrackingEvents(key []byte, value []byte) error {
	var trackingEvent TrackingUpdatedEvent
	if err := json.Unmarshal(value, &trackingEvent); err != nil {
		log.Printf("[Notifications] failed to unmarshal TrackingUpdated event: %v", err)
		return nil
	}

	notification := NotificationEvent{
		EventType:   "TrackingNotification",
		OrderID:     trackingEvent.OrderID,
		RecipientID: 0,
		Channel:     "push",
		Subject:     "Order tracking updated",
		Message:     fmt.Sprintf("Order %d status updated to %s.", trackingEvent.OrderID, trackingEvent.Status),
		Payload: map[string]any{
			"status":                trackingEvent.Status,
			"estimated_time_mins":   trackingEvent.EstimatedTimeMins,
			"estimated_distance_km": trackingEvent.EstimatedDistanceKm,
		},
		Timestamp: time.Now(),
	}

	PublishEventWithKey(NotificationsTopic, []byte(trackingEvent.GeoCell), notification)
	log.Printf("[Notifications] published tracking notification for order %d", trackingEvent.OrderID)
	return nil
}

func (ns *NotificationService) handleOrderPlaced(value []byte) error {
	var orderPlaced OrderPlacedEvent
	if err := json.Unmarshal(value, &orderPlaced); err != nil {
		log.Printf("[Notifications] failed to unmarshal OrderPlaced event: %v", err)
		return nil
	}

	notification := NotificationEvent{
		EventType:   "OrderPlacedNotification",
		OrderID:     orderPlaced.OrderID,
		RecipientID: orderPlaced.CustomerID,
		Channel:     "email",
		Subject:     "Order placed successfully",
		Message:     "Your order has been placed and is now being processed.",
		Payload: map[string]any{
			"pickup_address":  orderPlaced.PickupAddress,
			"dropoff_address": orderPlaced.DropoffAddress,
		},
		Timestamp: time.Now(),
	}

	PublishEventWithKey(NotificationsTopic, []byte(orderPlaced.GeoCell), notification)
	log.Printf("[Notifications] published order placed notification for order %d", orderPlaced.OrderID)
	return nil
}

func (ns *NotificationService) handleStatusUpdated(value []byte) error {
	var statusEvent OrderStatusUpdatedEvent
	if err := json.Unmarshal(value, &statusEvent); err != nil {
		log.Printf("[Notifications] failed to unmarshal OrderStatusUpdated event: %v", err)
		return nil
	}

	recipientID, err := ns.findOrderCustomer(statusEvent.OrderID)
	if err != nil {
		log.Printf("[Notifications] failed to load customer for order %d: %v", statusEvent.OrderID, err)
		return nil
	}

	notification := NotificationEvent{
		EventType:   "StatusUpdateNotification",
		OrderID:     statusEvent.OrderID,
		RecipientID: recipientID,
		Channel:     "push",
		Subject:     "Order status updated",
		Message:     fmt.Sprintf("Your order status is now %s.", statusEvent.Status),
		Payload: map[string]any{
			"status": statusEvent.Status,
		},
		Timestamp: time.Now(),
	}

	PublishEventWithKey(NotificationsTopic, []byte(statusEvent.GeoCell), notification)
	log.Printf("[Notifications] published status update notification for order %d", statusEvent.OrderID)
	return nil
}

func (ns *NotificationService) findOrderCustomer(orderID uint) (uint, error) {
	var order Order
	if err := DB.Select("user_id").First(&order, orderID).Error; err != nil {
		return 0, err
	}
	return order.UserID, nil
}
