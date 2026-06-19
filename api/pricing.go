package api

import (
	"encoding/json"
	"log"
	"math"
	"time"
)

// PricingCalculatedEvent is emitted when estimated pricing is available for an order.
type PricingCalculatedEvent struct {
	EventType           string    `json:"event_type"`
	OrderID             uint      `json:"order_id"`
	GeoCell             string    `json:"geo_cell"`
	FeeUnits            int       `json:"fee_units"`
	Currency            string    `json:"currency"`
	EstimatedDistanceKm float64   `json:"estimated_distance_km"`
	EstimatedTimeMins   float64   `json:"estimated_time_mins"`
	Timestamp           time.Time `json:"timestamp"`
}

// PricingService consumes OrderPlaced events and emits pricing events.
type PricingService struct {
	brokers  []string
	consumer *EventConsumer
}

// NewPricingService creates a new PricingService.
func NewPricingService(brokers []string) *PricingService {
	return &PricingService{brokers: brokers}
}

// Start begins consuming order placed events.
func (ps *PricingService) Start() error {
	ps.consumer = NewEventConsumer(ps.brokers, OrderEventsTopic, PricingConsumerGroup)
	return ps.consumer.Start(ps.handleOrderEvent)
}

// Stop halts the pricing service.
func (ps *PricingService) Stop() error {
	if ps.consumer != nil {
		return ps.consumer.Stop()
	}
	return nil
}

func (ps *PricingService) handleOrderEvent(key []byte, value []byte) error {
	eventType, err := EventType(value)
	if err != nil {
		log.Printf("[Pricing] failed to parse event envelope: %v", err)
		return nil
	}

	if eventType != OrderEventTypePlaced {
		return nil
	}

	var orderPlaced OrderPlacedEvent
	if err := json.Unmarshal(value, &orderPlaced); err != nil {
		log.Printf("[Pricing] failed to unmarshal OrderPlaced event: %v", err)
		return nil
	}

	log.Printf("[Pricing] Calculating price for order %d", orderPlaced.OrderID)

	estimatedDistance := haversineDistance(orderPlaced.PickupLatitude, orderPlaced.PickupLongitude, orderPlaced.DropoffLatitude, orderPlaced.DropoffLongitude)
	estimatedTime := EstimateTimeMins(estimatedDistance)
	feeUnits := CalculateFeeUnits(estimatedDistance, orderPlaced.WeightKg, orderPlaced.RequiresSpecialHandling)

	log.Printf("[Pricing] DEBUG - Pickup: %.6f,%.6f | Dropoff: %.6f,%.6f | Distance: %.2f km | Fee: %d units",
		orderPlaced.PickupLatitude, orderPlaced.PickupLongitude,
		orderPlaced.DropoffLatitude, orderPlaced.DropoffLongitude,
		estimatedDistance, feeUnits)

	pricingEvent := PricingCalculatedEvent{
		EventType:           OrderEventTypePricingCalculated,
		OrderID:             orderPlaced.OrderID,
		GeoCell:             orderPlaced.GeoCell,
		FeeUnits:            feeUnits,
		Currency:            "NGN",
		EstimatedDistanceKm: estimatedDistance,
		EstimatedTimeMins:   estimatedTime,
		Timestamp:           time.Now(),
	}

	PublishEventWithKey(PricingEventsTopic, []byte(orderPlaced.GeoCell), pricingEvent)

	if err := ps.updateOrderEstimates(orderPlaced.OrderID, feeUnits, estimatedDistance, estimatedTime); err != nil {
		log.Printf("[Pricing] failed to update order %d estimates: %v", orderPlaced.OrderID, err)
	}

	return nil
}

func (ps *PricingService) calculateFeeCents(distanceKm, weightKg float64, specialHandling bool) int {
	return CalculateFeeUnits(distanceKm, weightKg, specialHandling)
}

func (ps *PricingService) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	return haversineDistance(lat1, lng1, lat2, lng2)
}

func (ps *PricingService) estimateTimeMins(distanceKm float64) float64 {
	return EstimateTimeMins(distanceKm)
}

// CalculateFeeUnits computes the order fee in the smallest currency unit (e.g. kobo for NGN, cents for USD).
func CalculateFeeUnits(distanceKm, weightKg float64, specialHandling bool) int {
	baseUnits := 1500
	distanceUnits := int(math.Ceil(distanceKm * 120))
	weightUnits := int(math.Ceil(weightKg * 40))
	specialUnits := 0
	if specialHandling {
		specialUnits = 2000
	}
	return baseUnits + distanceUnits + weightUnits + specialUnits
}

// EstimateTimeMins estimates travel time in minutes given a distance in km.
func EstimateTimeMins(distanceKm float64) float64 {
	if distanceKm <= 0 {
		return 0
	}
	return math.Max(15, (distanceKm/35.0)*60.0)
}

func (ps *PricingService) updateOrderEstimates(orderID uint, feeUnits int, distanceKm, timeMins float64) error {
	return DB.Model(&Order{}).
		Where("id = ?", orderID).
		Updates(map[string]any{
			"fee_units":             feeUnits,
			"estimated_distance_km": distanceKm,
			"estimated_time_mins":   timeMins,
			"currency":              "NGN",
		}).Error
}
