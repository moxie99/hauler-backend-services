package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// TrackingUpdatedEvent is emitted when order tracking information changes.
type TrackingUpdatedEvent struct {
	EventType           string    `json:"event_type"`
	OrderID             uint      `json:"order_id"`
	GeoCell             string    `json:"geo_cell"`
	Status              string    `json:"status"`
	DriverID            *uint     `json:"driver_id,omitempty"`
	FeeUnits            *int      `json:"fee_units,omitempty"`
	EstimatedTimeMins   *float64  `json:"estimated_time_mins,omitempty"`
	EstimatedDistanceKm *float64  `json:"estimated_distance_km,omitempty"`
	Timestamp           time.Time `json:"timestamp"`
}

// UpdateTrackingRequest represents a tracking update from driver app.
type UpdateTrackingRequest struct {
	Latitude             *float64 `json:"latitude,omitempty"`
	Longitude            *float64 `json:"longitude,omitempty"`
	Status               string   `json:"status" binding:"required,oneof=picked_up=en_route=delivered"`
	EstimatedArrivalMins *int     `json:"estimated_arrival_mins,omitempty"`
	Message              string   `json:"message,omitempty"`
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
		FeeUnits:            statusEvent.FeeUnits,
		EstimatedTimeMins:   statusEvent.EstimatedTimeMins,
		EstimatedDistanceKm: statusEvent.EstimatedDistanceKm,
		Timestamp:           time.Now(),
	}

	PublishEventWithKey(TrackingEventsTopic, []byte(statusEvent.GeoCell), trackingEvent)
	log.Printf("[Tracking] Emitted tracking update for order %d status %s", statusEvent.OrderID, statusEvent.Status)
	return nil
}

// UpdateOrderTracking allows drivers to update order tracking information.
func UpdateOrderTracking(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid order ID", nil)
		return
	}

	// Verify driver owns this order
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		ResponseJSON(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		ResponseJSON(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Check if user is the assigned driver for this order
	var order Order
	if err := DB.Where("id = ? AND driver_id = ?", uint(orderID), userID).First(&order).Error; err != nil {
		ResponseJSON(c, http.StatusForbidden, "You are not assigned to this order", nil)
		return
	}

	var req UpdateTrackingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request: "+err.Error(), nil)
		return
	}

	// Validate status transition
	validStatuses := map[string][]string{
		"driver_assigned": {"picked_up", "en_route"},
		"picked_up":       {"en_route", "delivered"},
		"en_route":        {"delivered"},
	}

	validNextStatuses, exists := validStatuses[string(order.Status)]
	if !exists {
		ResponseJSON(c, http.StatusBadRequest, "Invalid current status", nil)
		return
	}

	validTransition := false
	for _, status := range validNextStatuses {
		if req.Status == status {
			validTransition = true
			break
		}
	}

	if !validTransition {
		ResponseJSON(c, http.StatusBadRequest, "Invalid status transition from "+string(order.Status)+" to "+req.Status, nil)
		return
	}

	// Update order status
	if err := DB.Model(&order).Update("status", req.Status).Error; err != nil {
		log.Printf("[Tracking] Error updating order %d status: %v", orderID, err)
		ResponseJSON(c, http.StatusInternalServerError, "Failed to update order status", nil)
		return
	}

	// Update driver location if provided
	if req.Latitude != nil && req.Longitude != nil {
		var location UserLocation
		result := DB.Where("user_id = ?", userID).First(&location)

		if result.Error == nil {
			// Update existing location
			if err := DB.Model(&location).Updates(map[string]interface{}{
				"latitude":   *req.Latitude,
				"longitude":  *req.Longitude,
				"updated_at": time.Now(),
			}).Error; err != nil {
				log.Printf("[Tracking] Error updating driver location: %v", err)
			}
		} else {
			// Create new location
			location = UserLocation{
				UserID:    userID,
				Latitude:  *req.Latitude,
				Longitude: *req.Longitude,
			}
			if err := DB.Create(&location).Error; err != nil {
				log.Printf("[Tracking] Error creating driver location: %v", err)
			}
		}
	}

	// Emit tracking update event
	trackingEvent := TrackingUpdatedEvent{
		EventType:           OrderEventTypeTrackingUpdated,
		OrderID:             uint(orderID),
		GeoCell:             order.GeoCell,
		Status:              req.Status,
		DriverID:            &userID,
		FeeUnits:            order.FeeUnits,
		EstimatedTimeMins:   &order.EstimatedTimeMins,
		EstimatedDistanceKm: &order.EstimatedDistanceKm,
		Timestamp:           time.Now(),
	}

	PublishEventWithKey(TrackingEventsTopic, []byte(order.GeoCell), trackingEvent)

	// Emit order status update event for notifications
	statusEvent := OrderStatusUpdatedEvent{
		EventType:           OrderEventTypeStatusUpdated,
		OrderID:             uint(orderID),
		GeoCell:             order.GeoCell,
		Status:              OrderStatus(req.Status),
		DriverID:            &userID,
		FeeUnits:            order.FeeUnits,
		EstimatedTimeMins:   &order.EstimatedTimeMins,
		EstimatedDistanceKm: &order.EstimatedDistanceKm,
		UpdatedAt:           time.Now(),
	}

	PublishEventWithKey(OrderEventsTopic, []byte(order.GeoCell), statusEvent)

	// Broadcast real-time update via WebSocket
	var driverLocation *UserLocation
	if req.Latitude != nil && req.Longitude != nil {
		driverLocation = &UserLocation{
			UserID:    userID,
			Latitude:  *req.Latitude,
			Longitude: *req.Longitude,
			UpdatedAt: time.Now(),
		}
	}
	BroadcastOrderUpdate(uint(orderID), "tracking_update", map[string]interface{}{
		"status":          req.Status,
		"driver_location": driverLocation,
		"timestamp":       time.Now(),
	})

	log.Printf("[Tracking] Driver %d updated order %d status to %s", userID, orderID, req.Status)

	ResponseJSON(c, http.StatusOK, "Tracking updated", map[string]interface{}{
		"order_id":   orderID,
		"status":     req.Status,
		"updated_at": time.Now(),
	})
}

// GetOrderTracking returns current tracking information for an order.
func GetOrderTracking(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid order ID", nil)
		return
	}

	// Get user ID for authorization
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		ResponseJSON(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		ResponseJSON(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Get order details
	var order Order
	if err := DB.Preload("Driver").Where("id = ?", uint(orderID)).First(&order).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Order not found", nil)
		return
	}

	// Check if user is the customer or assigned driver
	if order.UserID != userID && (order.DriverID == nil || *order.DriverID != userID) {
		ResponseJSON(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Get driver location if available
	var driverLocation *UserLocation
	if order.DriverID != nil {
		var loc UserLocation
		if err := DB.Where("user_id = ?", *order.DriverID).First(&loc).Error; err == nil {
			driverLocation = &loc
		}
	}

	trackingInfo := map[string]interface{}{
		"order_id":              order.ID,
		"status":                order.Status,
		"pickup_address":        order.PickupAddress,
		"dropoff_address":       order.DropoffAddress,
		"pickup_latitude":       order.PickupLatitude,
		"pickup_longitude":      order.PickupLongitude,
		"dropoff_latitude":      order.DropoffLatitude,
		"dropoff_longitude":     order.DropoffLongitude,
		"estimated_distance_km": order.EstimatedDistanceKm,
		"estimated_time_mins":   order.EstimatedTimeMins,
		"fee_units":             order.FeeUnits,
		"currency":              order.Currency,
		"created_at":            order.CreatedAt,
		"updated_at":            order.UpdatedAt,
	}

	if order.Driver != nil {
		trackingInfo["driver"] = map[string]interface{}{
			"id":         order.Driver.ID,
			"first_name": order.Driver.FirstName,
			"last_name":  order.Driver.LastName,
			"phone":      order.Driver.Phone,
		}
	}

	if driverLocation != nil {
		trackingInfo["driver_location"] = map[string]interface{}{
			"latitude":   driverLocation.Latitude,
			"longitude":  driverLocation.Longitude,
			"updated_at": driverLocation.UpdatedAt,
		}
	}

	ResponseJSON(c, http.StatusOK, "Tracking information retrieved", trackingInfo)
}
