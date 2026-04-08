package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateOrderRequest is the payload for customer order creation.
type CreateOrderRequest struct {
	PickupAddress           string  `json:"pickup_address" binding:"required,min=5,max=500"`
	DropoffAddress          string  `json:"dropoff_address" binding:"required,min=5,max=500"`
	PickupLatitude          float64 `json:"pickup_latitude" binding:"required"`
	PickupLongitude         float64 `json:"pickup_longitude" binding:"required"`
	DropoffLatitude         float64 `json:"dropoff_latitude" binding:"required"`
	DropoffLongitude        float64 `json:"dropoff_longitude" binding:"required"`
	GeoCell                 string  `json:"geo_cell" binding:"required,min=2,max=100"`
	VehicleTypeID           uint    `json:"vehicle_type_id" binding:"required"`
	LoadTypeID              uint    `json:"load_type_id" binding:"required"`
	CategoryID              *uint   `json:"category_id,omitempty"`
	WeightKg                float64 `json:"weight_kg" binding:"required,gt=0"`
	RequiresSpecialHandling bool    `json:"requires_special_handling"`
	PreferredPickupTime     string  `json:"preferred_pickup_time,omitempty" binding:"omitempty,max=50"`
	SpecialInstructions     string  `json:"special_instructions,omitempty" binding:"omitempty,max=1000"`
}

const (
	OrderEventTypePlaced            = "OrderPlaced"
	OrderEventTypeStatusUpdated     = "OrderStatusUpdated"
	OrderEventTypePricingCalculated = "PricingCalculated"
	OrderEventTypeTrackingUpdated   = "TrackingUpdated"
	OrderEventTypeDriverAssigned    = "DriverAssigned"
)

// OrderPlacedEvent is the contract emitted when an order is accepted by the orchestrator.
type OrderPlacedEvent struct {
	EventType               string      `json:"event_type"`
	OrderID                 uint        `json:"order_id"`
	GeoCell                 string      `json:"geo_cell"`
	CustomerID              uint        `json:"customer_id"`
	PickupAddress           string      `json:"pickup_address"`
	DropoffAddress          string      `json:"dropoff_address"`
	PickupLatitude          float64     `json:"pickup_latitude"`
	PickupLongitude         float64     `json:"pickup_longitude"`
	DropoffLatitude         float64     `json:"dropoff_latitude"`
	DropoffLongitude        float64     `json:"dropoff_longitude"`
	VehicleTypeID           uint        `json:"vehicle_type_id"`
	LoadTypeID              uint        `json:"load_type_id"`
	CategoryID              *uint       `json:"category_id,omitempty"`
	WeightKg                float64     `json:"weight_kg"`
	RequiresSpecialHandling bool        `json:"requires_special_handling"`
	Status                  OrderStatus `json:"status"`
	CreatedAt               time.Time   `json:"created_at"`
}

// OrderStatusUpdatedEvent is emitted when the orchestrator changes an order's state.
type OrderStatusUpdatedEvent struct {
	EventType           string      `json:"event_type"`
	OrderID             uint        `json:"order_id"`
	GeoCell             string      `json:"geo_cell"`
	Status              OrderStatus `json:"status"`
	DriverID            *uint       `json:"driver_id,omitempty"`
	FeeCents            *int        `json:"fee_cents,omitempty"`
	EstimatedTimeMins   *float64    `json:"estimated_time_mins,omitempty"`
	EstimatedDistanceKm *float64    `json:"estimated_distance_km,omitempty"`
	UpdatedAt           time.Time   `json:"updated_at"`
}

func publishOrderEvent(topic string, key []byte, event any) {
	PublishEventWithKey(topic, key, event)
}

// CreateOrder handles order creation and emits the initial OrderPlaced event.
func CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid order request payload: "+err.Error(), nil)
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if UserRole(role.(string)) != RoleCustomer {
		ResponseJSON(c, http.StatusForbidden, "Only customers may place orders", nil)
		return
	}

	var vehicleType VehicleType
	if err := DB.First(&vehicleType, req.VehicleTypeID).Error; err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid vehicle type", nil)
		return
	}
	if !vehicleType.IsActive {
		ResponseJSON(c, http.StatusBadRequest, "Selected vehicle type is not active", nil)
		return
	}

	var loadType LoadType
	if err := DB.First(&loadType, req.LoadTypeID).Error; err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid load type", nil)
		return
	}
	if !loadType.IsActive {
		ResponseJSON(c, http.StatusBadRequest, "Selected load type is not active", nil)
		return
	}

	var categoryID *uint
	if req.CategoryID != nil {
		var category Category
		if err := DB.First(&category, *req.CategoryID).Error; err != nil {
			ResponseJSON(c, http.StatusBadRequest, "Invalid category", nil)
			return
		}
		categoryID = req.CategoryID
	} else {
		categoryID = &vehicleType.CategoryID
	}

	order := Order{
		UserID:                  userID.(uint),
		GeoCell:                 req.GeoCell,
		PickupAddress:           req.PickupAddress,
		DropoffAddress:          req.DropoffAddress,
		PickupLatitude:          req.PickupLatitude,
		PickupLongitude:         req.PickupLongitude,
		DropoffLatitude:         req.DropoffLatitude,
		DropoffLongitude:        req.DropoffLongitude,
		VehicleTypeID:           req.VehicleTypeID,
		LoadTypeID:              req.LoadTypeID,
		CategoryID:              categoryID,
		WeightKg:                req.WeightKg,
		RequiresSpecialHandling: req.RequiresSpecialHandling,
		PreferredPickupTime:     req.PreferredPickupTime,
		SpecialInstructions:     req.SpecialInstructions,
		Status:                  OrderStatusCreated,
		Currency:                "NGN",
	}

	if err := DB.Create(&order).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to create order", nil)
		return
	}

	event := OrderPlacedEvent{
		EventType:               OrderEventTypePlaced,
		OrderID:                 order.ID,
		GeoCell:                 order.GeoCell,
		CustomerID:              order.UserID,
		PickupAddress:           order.PickupAddress,
		DropoffAddress:          order.DropoffAddress,
		PickupLatitude:          order.PickupLatitude,
		PickupLongitude:         order.PickupLongitude,
		DropoffLatitude:         order.DropoffLatitude,
		DropoffLongitude:        order.DropoffLongitude,
		VehicleTypeID:           order.VehicleTypeID,
		LoadTypeID:              order.LoadTypeID,
		CategoryID:              order.CategoryID,
		WeightKg:                order.WeightKg,
		RequiresSpecialHandling: order.RequiresSpecialHandling,
		Status:                  order.Status,
		CreatedAt:               order.CreatedAt,
	}

	publishOrderEvent(OrderEventsTopic, []byte(order.GeoCell), event)

	ResponseJSON(c, http.StatusCreated, "Order placed successfully", order)
}

// GetOrders returns all orders for the authenticated customer.
func GetOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if UserRole(role.(string)) != RoleCustomer {
		ResponseJSON(c, http.StatusForbidden, "Only customers may view orders", nil)
		return
	}

	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}
	offset := (page - 1) * pageSize

	query := DB.Where("user_id = ?", userID.(uint)).Order("created_at desc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var orders []Order
	if err := query.Preload("VehicleType").Preload("LoadType").Preload("Category").Limit(pageSize).Offset(offset).Find(&orders).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to retrieve orders", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Orders retrieved successfully", orders)
}

// GetOrder returns a single order by ID for the authenticated customer.
func GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if UserRole(role.(string)) != RoleCustomer {
		ResponseJSON(c, http.StatusForbidden, "Only customers may view orders", nil)
		return
	}

	var order Order
	if err := DB.Preload("VehicleType").Preload("LoadType").Preload("Category").First(&order, orderID).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Order not found", nil)
		return
	}

	if order.UserID != userID.(uint) {
		ResponseJSON(c, http.StatusForbidden, "You may only view your own orders", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Order retrieved successfully", order)
}

// UpdateOrderStatusRequest is the payload for status transition requests.
type UpdateOrderStatusRequest struct {
	Status              OrderStatus `json:"status" binding:"required,oneof=pricing_requested on_hold dispatch_requested driver_assigned picked_up delivered cancelled"`
	DriverID            *uint       `json:"driver_id,omitempty"`
	FeeCents            *int        `json:"fee_cents,omitempty"`
	DriverRateCents     *int        `json:"driver_rate_cents,omitempty"`
	EstimatedTimeMins   *float64    `json:"estimated_time_mins,omitempty"`
	EstimatedDistanceKm *float64    `json:"estimated_distance_km,omitempty"`
}

// UpdateOrderStatus updates order status and emits a status update event.
func UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid status update payload: "+err.Error(), nil)
		return
	}

	var order Order
	if err := DB.First(&order, orderID).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Order not found", nil)
		return
	}

	if order.Status == OrderStatusDelivered || order.Status == OrderStatusCancelled {
		if order.Status != req.Status {
			ResponseJSON(c, http.StatusBadRequest, "Cannot change status of a final order state", nil)
			return
		}
		ResponseJSON(c, http.StatusOK, "Order status already set", order)
		return
	}

	order.Status = req.Status
	if req.DriverID != nil {
		order.DriverID = req.DriverID
	}
	if req.FeeCents != nil {
		order.FeeCents = req.FeeCents
	}
	if req.DriverRateCents != nil {
		order.DriverRateCents = req.DriverRateCents
	}
	if req.EstimatedTimeMins != nil {
		order.EstimatedTimeMins = *req.EstimatedTimeMins
	}
	if req.EstimatedDistanceKm != nil {
		order.EstimatedDistanceKm = *req.EstimatedDistanceKm
	}

	if err := DB.Save(&order).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to update order status", nil)
		return
	}

	event := OrderStatusUpdatedEvent{
		EventType:           OrderEventTypeStatusUpdated,
		OrderID:             order.ID,
		GeoCell:             order.GeoCell,
		Status:              order.Status,
		DriverID:            order.DriverID,
		FeeCents:            order.FeeCents,
		EstimatedTimeMins:   &order.EstimatedTimeMins,
		EstimatedDistanceKm: &order.EstimatedDistanceKm,
		UpdatedAt:           order.UpdatedAt,
	}

	publishOrderEvent(OrderEventsTopic, []byte(order.GeoCell), event)

	ResponseJSON(c, http.StatusOK, "Order status updated successfully", order)
}
