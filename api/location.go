package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UpdateLocationRequest represents a request to update user location
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// UpdateLocation updates the current location of the authenticated user (typically a driver)
func UpdateLocation(c *gin.Context) {
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

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid request: "+err.Error(), nil)
		return
	}

	// Validate coordinates
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		ResponseJSON(c, http.StatusBadRequest, "Invalid coordinates", nil)
		return
	}

	// Upsert location: find or create
	var location UserLocation
	result := DB.Where("user_id = ?", userID).First(&location)

	if result.Error == nil {
		// Update existing location
		if err := DB.Model(&location).Updates(map[string]interface{}{
			"latitude":   req.Latitude,
			"longitude":  req.Longitude,
			"updated_at": nil, // GORM will set current time
		}).Error; err != nil {
			log.Printf("[Location] Error updating location for user %d: %v", userID, err)
			ResponseJSON(c, http.StatusInternalServerError, "Failed to update location", nil)
			return
		}
		log.Printf("[Location] Updated location for user %d: %.6f,%.6f", userID, req.Latitude, req.Longitude)
	} else {
		// Create new location record
		location = UserLocation{
			UserID:    userID,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		}
		if err := DB.Create(&location).Error; err != nil {
			log.Printf("[Location] Error creating location for user %d: %v", userID, err)
			ResponseJSON(c, http.StatusInternalServerError, "Failed to create location", nil)
			return
		}
		log.Printf("[Location] Created location for user %d: %.6f,%.6f", userID, req.Latitude, req.Longitude)
	}

	ResponseJSON(c, http.StatusOK, "Location updated", location)
}

// GetLocation returns the current location of a specific user (admin/super-admin only)
func GetLocation(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		ResponseJSON(c, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var location UserLocation
	if err := DB.Where("user_id = ?", uint(userID)).First(&location).Error; err != nil {
		ResponseJSON(c, http.StatusNotFound, "Location not found", nil)
		return
	}

	ResponseJSON(c, http.StatusOK, "Location retrieved", location)
}
