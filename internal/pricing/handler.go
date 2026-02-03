package pricing

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Handler handles HTTP requests for pricing
type Handler struct {
	service *Service
}

// NewHandler creates a new pricing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetEstimate returns a fare estimate
func (h *Handler) GetEstimate(c *gin.Context) {
	var req EstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	estimate, err := h.service.GetEstimate(c.Request.Context(), req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to calculate estimate")
		return
	}

	common.SuccessResponse(c, estimate)
}

// GetSurge returns current surge information
func (h *Handler) GetSurge(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "lat and lng are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lat")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lng")
		return
	}

	surgeInfo, err := h.service.GetSurgeInfo(c.Request.Context(), lat, lng)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get surge info")
		return
	}

	common.SuccessResponse(c, surgeInfo)
}

// ValidatePrice validates a negotiated price
func (h *Handler) ValidatePrice(c *gin.Context) {
	var req struct {
		EstimateRequest
		NegotiatedPrice float64 `json:"negotiated_price" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err := h.service.ValidateNegotiatedPrice(c.Request.Context(), req.EstimateRequest, req.NegotiatedPrice)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get the estimate for reference
	estimate, _ := h.service.GetEstimate(c.Request.Context(), req.EstimateRequest)

	common.SuccessResponse(c, gin.H{
		"valid":           true,
		"negotiated_price": req.NegotiatedPrice,
		"estimated_price": estimate.EstimatedFare,
		"variance_pct":    ((req.NegotiatedPrice - estimate.EstimatedFare) / estimate.EstimatedFare) * 100,
	})
}

// GetPricing returns resolved pricing for a location
func (h *Handler) GetPricing(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	rideTypeIDStr := c.Query("ride_type_id")

	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "lat and lng are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lat")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lng")
		return
	}

	var rideTypeID *uuid.UUID
	if rideTypeIDStr != "" {
		id, err := uuid.Parse(rideTypeIDStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "invalid ride_type_id")
			return
		}
		rideTypeID = &id
	}

	pricing, err := h.service.GetPricing(c.Request.Context(), lat, lng, rideTypeID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pricing")
		return
	}

	common.SuccessResponse(c, pricing)
}

// GetCancellationFee returns the cancellation fee
func (h *Handler) GetCancellationFee(c *gin.Context) {
	var req struct {
		Latitude            float64 `json:"latitude" binding:"required"`
		Longitude           float64 `json:"longitude" binding:"required"`
		MinutesSinceRequest float64 `json:"minutes_since_request" binding:"required,gte=0"`
		EstimatedFare       float64 `json:"estimated_fare" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	fee, err := h.service.GetCancellationFee(
		c.Request.Context(),
		req.Latitude, req.Longitude,
		req.MinutesSinceRequest,
		req.EstimatedFare,
	)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to calculate cancellation fee")
		return
	}

	common.SuccessResponse(c, gin.H{
		"cancellation_fee":     fee,
		"minutes_since_request": req.MinutesSinceRequest,
	})
}

// RegisterRoutes registers pricing routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	pricing := rg.Group("/pricing")
	{
		pricing.POST("/estimate", h.GetEstimate)
		pricing.GET("/surge", h.GetSurge)
		pricing.POST("/validate", h.ValidatePrice)
		pricing.GET("/config", h.GetPricing)
		pricing.POST("/cancellation-fee", h.GetCancellationFee)
	}
}
