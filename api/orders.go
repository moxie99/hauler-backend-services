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
	BargainedPriceUnits     *int        `json:"bargained_price_units,omitempty"`
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
	FeeUnits            *int        `json:"fee_units,omitempty"`
	EstimatedTimeMins   *float64    `json:"estimated_time_mins,omitempty"`
	EstimatedDistanceKm *float64    `json:"estimated_distance_km,omitempty"`
	UpdatedAt           time.Time   `json:"updated_at"`
}

func publishOrderEvent(topic string, key []byte, event any) {
	PublishEventWithKey(topic, key, event)
}

// CreateOrder handles order creation, calculates pricing synchronously, and returns
// the projected price along with origin/destination place names to the frontend.
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

	// Calculate pricing synchronously before saving
	estimatedDistance := haversineDistance(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	estimatedTime := EstimateTimeMins(estimatedDistance)
	feeUnits := CalculateFeeUnits(estimatedDistance, req.WeightKg, req.RequiresSpecialHandling)

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
		FeeUnits:                &feeUnits,
		EstimatedDistanceKm:     estimatedDistance,
		EstimatedTimeMins:       estimatedTime,
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

	ResponseJSON(c, http.StatusCreated, "Order placed successfully", gin.H{
		"order":                 order,
		"origin":                order.PickupAddress,
		"destination":           order.DropoffAddress,
		"fee_units":             feeUnits,
		"currency":              "NGN",
		"estimated_distance_km": estimatedDistance,
		"estimated_time_mins":   estimatedTime,
	})
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

// GetAllOrders returns all orders for admin with filtering options.
func GetAllOrders(c *gin.Context) {
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}
	offset := (page - 1) * pageSize

	query := DB.Order("created_at desc")

	// Apply filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if customerID := c.Query("customer_id"); customerID != "" {
		if id, err := strconv.ParseUint(customerID, 10, 32); err == nil {
			query = query.Where("user_id = ?", uint(id))
		}
	}
	if driverID := c.Query("driver_id"); driverID != "" {
		if id, err := strconv.ParseUint(driverID, 10, 32); err == nil {
			query = query.Where("driver_id = ?", uint(id))
		}
	}
	if vehicleTypeID := c.Query("vehicle_type_id"); vehicleTypeID != "" {
		if id, err := strconv.ParseUint(vehicleTypeID, 10, 32); err == nil {
			query = query.Where("vehicle_type_id = ?", uint(id))
		}
	}
	if geoCell := c.Query("geo_cell"); geoCell != "" {
		query = query.Where("geo_cell = ?", geoCell)
	}
	if pickupDate := c.Query("pickup_date"); pickupDate != "" {
		// Assuming date format YYYY-MM-DD
		query = query.Where("DATE(preferred_pickup_time) = ?", pickupDate)
	}

	var orders []Order
	if err := query.Preload("VehicleType").Preload("LoadType").Preload("Category").Preload("User").Preload("Driver").Limit(pageSize).Offset(offset).Find(&orders).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to retrieve orders", nil)
		return
	}

	// Get total count for pagination
	var total int64
	if err := query.Model(&Order{}).Count(&total).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to count orders", nil)
		return
	}

	response := gin.H{
		"orders":      orders,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}

	ResponseJSON(c, http.StatusOK, "Orders retrieved successfully", response)
}

// ConfirmOrderRequest is the payload for confirming an order with a bargained price.
type ConfirmOrderRequest struct {
	BargainedPriceUnits int `json:"bargained_price_units" binding:"required,gt=0"`
}

// ConfirmOrder allows the customer to confirm an order with a bargained price.
// The bargained price must be within ±30% of the projected fee.
func ConfirmOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	if UserRole(role.(string)) != RoleCustomer {
		ResponseJSON(c, http.StatusForbidden, "Only customers may confirm orders", nil)
		return
	}

	var req ConfirmOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request payload: "+err.Error(), nil)
		return
	}

	var order Order
	if err := DB.First(&order, orderID).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Order not found", nil)
		return
	}

	if order.UserID != userID.(uint) {
		ResponseJSON(c, http.StatusForbidden, "You may only confirm your own orders", nil)
		return
	}

	if order.Status != OrderStatusCreated {
		ResponseJSON(c, http.StatusBadRequest, "Order has already been confirmed or is no longer pending", nil)
		return
	}

	if order.FeeUnits == nil {
		ResponseJSON(c, http.StatusInternalServerError, "Projected price not available for this order", nil)
		return
	}

	// Validate bargained price: cannot go more than 500 units below projected, no upper limit
	projected := *order.FeeUnits
	lowerBound := projected - 500

	if req.BargainedPriceUnits < lowerBound {
		ResponseJSON(c, http.StatusBadRequest, "Bargained price cannot be more than 500 units below the projected price", gin.H{
			"projected_price_units": projected,
			"min_allowed_units":     lowerBound,
		})
		return
	}

	// Save bargained price and move order to dispatch_requested
	bargained_int := req.BargainedPriceUnits
	order.BargainedPriceUnits = &bargained_int
	order.Status = OrderStatusDispatchRequested

	if err := DB.Save(&order).Error; err != nil {
		ResponseJSON(c, http.StatusInternalServerError, "Failed to confirm order", nil)
		return
	}

	// Emit dispatch event with bargained price so nearby drivers see it
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
		BargainedPriceUnits:     order.BargainedPriceUnits,
		Status:                  order.Status,
		CreatedAt:               order.CreatedAt,
	}

	publishOrderEvent(OrderEventsTopic, []byte(order.GeoCell), event)

	ResponseJSON(c, http.StatusOK, "Order confirmed and sent to nearby drivers", gin.H{
		"order":                 order,
		"origin":                order.PickupAddress,
		"destination":           order.DropoffAddress,
		"projected_price_units": *order.FeeUnits,
		"bargained_price_units": req.BargainedPriceUnits,
		"currency":              order.Currency,
		"estimated_distance_km": order.EstimatedDistanceKm,
		"estimated_time_mins":   order.EstimatedTimeMins,
	})
}
type UpdateOrderStatusRequest struct {
	Status              OrderStatus `json:"status" binding:"required,oneof=pricing_requested on_hold dispatch_requested driver_assigned picked_up delivered cancelled"`
	DriverID            *uint       `json:"driver_id,omitempty"`
	FeeUnits            *int        `json:"fee_units,omitempty"`
	DriverRateUnits     *int        `json:"driver_rate_units,omitempty"`
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
	if req.FeeUnits != nil {
		order.FeeUnits = req.FeeUnits
	}
	if req.DriverRateUnits != nil {
		order.DriverRateUnits = req.DriverRateUnits
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
		FeeUnits:            order.FeeUnits,
		EstimatedTimeMins:   &order.EstimatedTimeMins,
		EstimatedDistanceKm: &order.EstimatedDistanceKm,
		UpdatedAt:           order.UpdatedAt,
	}

	publishOrderEvent(OrderEventsTopic, []byte(order.GeoCell), event)

	ResponseJSON(c, http.StatusOK, "Order status updated successfully", order)
}
