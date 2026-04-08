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
	FeeCents            int       `json:"fee_cents"`
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

	estimatedDistance := ps.calculateDistance(orderPlaced.PickupLatitude, orderPlaced.PickupLongitude, orderPlaced.DropoffLatitude, orderPlaced.DropoffLongitude)
	estimatedTime := ps.estimateTimeMins(estimatedDistance)
	feeCents := ps.calculateFeeCents(estimatedDistance, orderPlaced.WeightKg, orderPlaced.RequiresSpecialHandling)

	pricingEvent := PricingCalculatedEvent{
		EventType:           OrderEventTypePricingCalculated,
		OrderID:             orderPlaced.OrderID,
		GeoCell:             orderPlaced.GeoCell,
		FeeCents:            feeCents,
		Currency:            "NGN",
		EstimatedDistanceKm: estimatedDistance,
		EstimatedTimeMins:   estimatedTime,
		Timestamp:           time.Now(),
	}

	PublishEventWithKey(PricingEventsTopic, []byte(orderPlaced.GeoCell), pricingEvent)

	if err := ps.updateOrderEstimates(orderPlaced.OrderID, feeCents, estimatedDistance, estimatedTime); err != nil {
		log.Printf("[Pricing] failed to update order %d estimates: %v", orderPlaced.OrderID, err)
	}

	return nil
}

func (ps *PricingService) calculateFeeCents(distanceKm, weightKg float64, specialHandling bool) int {
	baseCents := 1500
	distanceCents := int(math.Ceil(distanceKm * 120))
	weightCents := int(math.Ceil(weightKg * 40))
	specialCents := 0
	if specialHandling {
		specialCents = 2000
	}
	return baseCents + distanceCents + weightCents + specialCents
}

func (ps *PricingService) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	return haversineDistance(lat1, lng1, lat2, lng2)
}

func (ps *PricingService) estimateTimeMins(distanceKm float64) float64 {
	if distanceKm <= 0 {
		return 0
	}
	// Estimate with a conservative 35 km/h average speed.
	return math.Max(15, (distanceKm/35.0)*60.0)
}

func (ps *PricingService) updateOrderEstimates(orderID uint, feeCents int, distanceKm, timeMins float64) error {
	return DB.Model(&Order{}).
		Where("id = ?", orderID).
		Updates(map[string]any{
			"fee_cents":             feeCents,
			"estimated_distance_km": distanceKm,
			"estimated_time_mins":   timeMins,
			"currency":              "NGN",
		}).Error
}
