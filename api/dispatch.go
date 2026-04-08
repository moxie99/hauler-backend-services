package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

// DriverAssignedEvent is emitted when a driver is assigned to an order.
type DriverAssignedEvent struct {
	EventType       string    `json:"event_type"`
	OrderID         uint      `json:"order_id"`
	DriverID        uint      `json:"driver_id"`
	GeoCell         string    `json:"geo_cell"`
	AssignmentScore float64   `json:"assignment_score"`
	Timestamp       time.Time `json:"timestamp"`
}

// MatchResult holds the result of a driver matching operation.
type MatchResult struct {
	DriverID       uint
	Score          float64
	Distance       float64
	AvailableUntil time.Time
}

// DispatchMatcher handles order-to-driver matching logic.
type DispatchMatcher struct {
	brokers  []string
	consumer *EventConsumer
}

// NewDispatchMatcher creates a new dispatch matcher.
func NewDispatchMatcher(brokers []string) *DispatchMatcher {
	return &DispatchMatcher{
		brokers: brokers,
	}
}

// Start begins consuming and matching orders to drivers.
func (dm *DispatchMatcher) Start() error {
	dm.consumer = NewEventConsumer(dm.brokers, OrderEventsTopic, DispatchConsumerGroup)
	return dm.consumer.Start(dm.handleOrderPlaced)
}

// Stop halts the dispatch matcher.
func (dm *DispatchMatcher) Stop() error {
	if dm.consumer != nil {
		return dm.consumer.Stop()
	}
	return nil
}

// handleOrderPlaced processes OrderPlaced events and attempts to match drivers.
func (dm *DispatchMatcher) handleOrderPlaced(key []byte, value []byte) error {
	var event OrderPlacedEvent
	if err := json.Unmarshal(value, &event); err != nil {
		log.Printf("[Dispatch] Failed to unmarshal OrderPlaced event: %v", err)
		return nil // Don't retry malformed events
	}

	// Check idempotency
	eventKey := generateEventKey(event.OrderID, "OrderPlaced", event.CreatedAt)
	if IsEventProcessed(eventKey) {
		log.Printf("[Dispatch] Event already processed: %s", eventKey)
		return nil
	}

	log.Printf("[Dispatch] Processing order %d in geo-cell %s", event.OrderID, event.GeoCell)

	// Find the best matching driver
	bestMatch, err := dm.findBestDriver(&event)
	if err != nil {
		log.Printf("[Dispatch] Error finding driver for order %d: %v", event.OrderID, err)
		return nil // Log the error but don't retry; dispatch will retry later
	}

	if bestMatch == nil {
		log.Printf("[Dispatch] No suitable driver found for order %d", event.OrderID)
		return nil // No drivers available; will retry on next run
	}

	// Emit DriverAssigned event
	assignedEvent := DriverAssignedEvent{
		EventType:       OrderEventTypeDriverAssigned,
		OrderID:         event.OrderID,
		DriverID:        bestMatch.DriverID,
		GeoCell:         event.GeoCell,
		AssignmentScore: bestMatch.Score,
		Timestamp:       time.Now(),
	}

	PublishEventWithKey(DispatchEventsTopic, key, assignedEvent)

	// Mark event as processed
	offset, partition := extractOffsetFromKey(key)
	MarkEventProcessed(event.OrderID, "OrderPlaced", event.GeoCell, eventKey, offset, partition)

	log.Printf("[Dispatch] Assigned driver %d (score: %.2f) to order %d", bestMatch.DriverID, bestMatch.Score, event.OrderID)
	return nil
}

// findBestDriver queries available drivers and runs the matching algorithm.
func (dm *DispatchMatcher) findBestDriver(order *OrderPlacedEvent) (*MatchResult, error) {
	// Query drivers with matching vehicle and load types, sorted by availability
	var drivers []struct {
		ID        uint
		Latitude  *float64
		Longitude *float64
	}

	// This is a simplified query; in production, you'd join with driver availability services
	result := DB.
		Table("users").
		Select("users.id, user_locations.latitude, user_locations.longitude").
		Joins("LEFT JOIN user_locations ON users.id = user_locations.user_id").
		Joins("LEFT JOIN driver_vehicle_types ON users.id = driver_vehicle_types.driver_id").
		Joins("LEFT JOIN driver_load_types ON users.id = driver_load_types.driver_id").
		Where("users.role = ? AND users.is_active = ? AND users.kyc_status = ?", RoleDriver, true, KYCStatusApproved).
		Where("driver_vehicle_types.vehicle_type_id = ?", order.VehicleTypeID).
		Where("driver_load_types.load_type_id = ?", order.LoadTypeID).
		Limit(50).
		Scan(&drivers)

	if result.Error != nil {
		return nil, result.Error
	}

	if len(drivers) == 0 {
		return nil, nil
	}

	// Score each driver and return the best match
	var bestMatch *MatchResult
	for _, driver := range drivers {
		score := dm.scoreDriver(driver.ID, order, driver.Latitude, driver.Longitude)
		if bestMatch == nil || score > bestMatch.Score {
			bestMatch = &MatchResult{
				DriverID:       driver.ID,
				Score:          score,
				AvailableUntil: time.Now().Add(30 * time.Minute),
			}
		}
	}

	return bestMatch, nil
}

// scoreDriver calculates a matching score for a driver based on proximity and other factors.
func (dm *DispatchMatcher) scoreDriver(_ uint, order *OrderPlacedEvent, driverLat, driverLng *float64) float64 {
	if driverLat == nil || driverLng == nil {
		return 0 // Driver has no location
	}

	// Calculate distance to pickup location using Haversine formula
	distance := haversineDistance(
		*driverLat, *driverLng,
		order.PickupLatitude, order.PickupLongitude,
	)

	// Base score: closer drivers have higher scores
	// Normalize distance: 0-5km = 100 points, 5-10km = 80 points, 10-20km = 60 points, etc.
	var baseScore float64
	if distance <= 5 {
		baseScore = 100
	} else if distance <= 10 {
		baseScore = 80
	} else if distance <= 20 {
		baseScore = 60
	} else if distance <= 30 {
		baseScore = 40
	} else {
		baseScore = math.Max(10, 100-distance)
	}

	// In production, you'd also factor in:
	// - Driver rating/reviews
	// - Recent acceptance rate
	// - Load specialization
	// - Preferred load types
	// - Current vehicle utilization
	// - Historical order completion time

	return baseScore
}

// haversineDistance calculates the great-circle distance between two points on Earth (in km).
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth radius in kilometers
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Asin(math.Sqrt(a))
	return R * c
}

func toRad(deg float64) float64 {
	return deg * (math.Pi / 180)
}

// generateEventKey creates a unique key for event idempotency.
func generateEventKey(orderID uint, eventType string, timestamp time.Time) string {
	data := fmt.Sprintf("%d-%s-%d", orderID, eventType, timestamp.Unix())
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// extractOffsetFromKey attempts to extract Kafka offset/partition from the event key.
// For now, returns 0, 0 as placeholder; in production, get from reader metadata.
func extractOffsetFromKey(_ []byte) (int64, int32) {
	return 0, 0
}
