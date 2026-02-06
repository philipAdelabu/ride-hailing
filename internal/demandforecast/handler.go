package demandforecast

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for demand forecasting
type Handler struct {
	service *Service
}

// NewHandler creates a new demand forecast handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// PREDICTION ENDPOINTS
// ========================================

// GetPrediction returns demand prediction for a location
// POST /api/v1/demand/predict
func (h *Handler) GetPrediction(c *gin.Context) {
	var req GetPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	pred, err := h.service.GeneratePrediction(c.Request.Context(), req.Latitude, req.Longitude, req.Timeframe)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate prediction")
		return
	}

	common.SuccessResponse(c, pred)
}

// GetHeatmap returns a demand heatmap for a geographic area
// POST /api/v1/demand/heatmap
func (h *Handler) GetHeatmap(c *gin.Context) {
	var req GetHeatmapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	heatmap, err := h.service.GetDemandHeatmap(c.Request.Context(), &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate heatmap")
		return
	}

	common.SuccessResponse(c, heatmap)
}

// GetHotspots returns the top demand hotspots
// GET /api/v1/demand/hotspots
func (h *Handler) GetHotspots(c *gin.Context) {
	timeframe := PredictionTimeframe(c.DefaultQuery("timeframe", "30min"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	hotspots, err := h.service.GetTopHotspots(c.Request.Context(), timeframe, limit)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get hotspots")
		return
	}

	common.SuccessResponse(c, gin.H{
		"hotspots":  hotspots,
		"count":     len(hotspots),
		"timeframe": timeframe,
	})
}

// ========================================
// DRIVER REPOSITIONING ENDPOINTS
// ========================================

// GetRepositionRecommendations returns driver repositioning suggestions
// POST /api/v1/demand/reposition
func (h *Handler) GetRepositionRecommendations(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req GetRepositionRecommendationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}
	req.DriverID = driverID

	response, err := h.service.GetRepositionRecommendations(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get recommendations")
		return
	}

	common.SuccessResponse(c, response)
}

// ========================================
// EVENT ENDPOINTS (Admin)
// ========================================

// CreateEvent creates a special event
// POST /api/v1/demand/events
func (h *Handler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	event, err := h.service.CreateEvent(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create event")
		return
	}

	common.SuccessResponse(c, event)
}

// ListUpcomingEvents lists upcoming events
// GET /api/v1/demand/events
func (h *Handler) ListUpcomingEvents(c *gin.Context) {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

	events, err := h.service.GetUpcomingEvents(c.Request.Context(), hours)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list events")
		return
	}

	common.SuccessResponse(c, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// ========================================
// MODEL ACCURACY ENDPOINTS (Admin)
// ========================================

// GetModelAccuracy returns model accuracy metrics
// GET /api/v1/demand/accuracy
func (h *Handler) GetModelAccuracy(c *gin.Context) {
	timeframe := PredictionTimeframe(c.DefaultQuery("timeframe", "30min"))
	daysBack, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	metrics, err := h.service.GetModelAccuracy(c.Request.Context(), timeframe, daysBack)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get accuracy metrics")
		return
	}

	common.SuccessResponse(c, metrics)
}

// ========================================
// DATA COLLECTION ENDPOINT
// ========================================

// RecordDemandSnapshot records current demand (called by internal services)
// POST /api/v1/internal/demand/record
func (h *Handler) RecordDemandSnapshot(c *gin.Context) {
	var req struct {
		H3Index          string  `json:"h3_index" binding:"required"`
		RideRequests     int     `json:"ride_requests"`
		CompletedRides   int     `json:"completed_rides"`
		AvailableDrivers int     `json:"available_drivers"`
		AvgWaitTime      float64 `json:"avg_wait_time"`
		SurgeMultiplier  float64 `json:"surge_multiplier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RecordDemandSnapshot(
		c.Request.Context(),
		req.H3Index,
		req.RideRequests,
		req.CompletedRides,
		req.AvailableDrivers,
		req.AvgWaitTime,
		req.SurgeMultiplier,
	); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to record demand")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "demand recorded"})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers demand forecast routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Public prediction endpoints (drivers)
	demand := r.Group("/api/v1/demand")
	demand.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Predictions
		demand.POST("/predict", h.GetPrediction)
		demand.POST("/heatmap", h.GetHeatmap)
		demand.GET("/hotspots", h.GetHotspots)

		// Driver repositioning
		demand.POST("/reposition", h.GetRepositionRecommendations)

		// Events (read)
		demand.GET("/events", h.ListUpcomingEvents)
	}

	// Admin endpoints
	admin := r.Group("/api/v1/admin/demand")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.POST("/events", h.CreateEvent)
		admin.GET("/accuracy", h.GetModelAccuracy)
	}

	// Internal endpoints (service-to-service)
	internal := r.Group("/api/v1/internal/demand")
	{
		internal.POST("/record", h.RecordDemandSnapshot)
	}
}

// Ensure uuid.UUID is used (prevent unused import)
var _ = uuid.New
