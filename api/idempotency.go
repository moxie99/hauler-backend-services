package api

import (
	"time"
)

// OrderEventLog tracks processed events to prevent duplicate handling.
type OrderEventLog struct {
	ID          uint   `gorm:"primaryKey"`
	OrderID     uint   `gorm:"index:idx_order_event;"` // Composite index for order-level idempotency
	EventType   string `gorm:"index:idx_order_event;"` // OrderPlaced, OrderStatusUpdated, etc.
	GeoCell     string `gorm:"index"`                  // For easy lookup by region
	EventKey    string `gorm:"uniqueIndex"`            // Unique key: hash(order_id + event_type + offset)
	Offset      int64  `gorm:"index"`                  // Kafka message offset for tracking
	Partition   int32  // Kafka partition number
	ProcessedAt time.Time
	CreatedAt   time.Time
}

// MarkEventProcessed records that an event has been successfully handled.
func MarkEventProcessed(orderID uint, eventType, geoCell, eventKey string, offset int64, partition int32) error {
	log := OrderEventLog{
		OrderID:     orderID,
		EventType:   eventType,
		GeoCell:     geoCell,
		EventKey:    eventKey,
		Offset:      offset,
		Partition:   partition,
		ProcessedAt: time.Now(),
	}
	return DB.Create(&log).Error
}

// IsEventProcessed checks if an event has already been handled.
func IsEventProcessed(eventKey string) bool {
	var count int64
	DB.Model(&OrderEventLog{}).Where("event_key = ?", eventKey).Count(&count)
	return count > 0
}

// GetLastProcessedOffset returns the highest offset processed for a geo-cell partition.
func GetLastProcessedOffset(geoCell string, partition int32) (int64, error) {
	var maxOffset int64
	result := DB.Model(&OrderEventLog{}).
		Where("geo_cell = ? AND partition = ?", geoCell, partition).
		Select("COALESCE(MAX(offset), -1)").
		Scan(&maxOffset)
	return maxOffset, result.Error
}
