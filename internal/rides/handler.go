package rides

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for rides
type Handler struct {
	service *Service
}

// NewHandler creates a new rides handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RequestRide handles creating a new ride request
func (h *Handler) RequestRide(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.RideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ride, err := h.service.RequestRide(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to request ride")
		return
	}

	common.CreatedResponse(c, ride)
}

// GetRide handles getting a ride by ID
func (h *Handler) GetRide(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	ride, err := h.service.GetRide(c.Request.Context(), rideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// AcceptRide handles a driver accepting a ride
func (h *Handler) AcceptRide(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	ride, err := h.service.AcceptRide(c.Request.Context(), rideID, driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to accept ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// StartRide handles a driver starting a ride
func (h *Handler) StartRide(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	ride, err := h.service.StartRide(c.Request.Context(), rideID, driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// CompleteRide handles completing a ride
func (h *Handler) CompleteRide(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	var req struct {
		ActualDistance float64 `json:"actual_distance" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ride, err := h.service.CompleteRide(c.Request.Context(), rideID, driverID, req.ActualDistance)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to complete ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// CancelRide handles cancelling a ride
func (h *Handler) CancelRide(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ride, err := h.service.CancelRide(c.Request.Context(), rideID, userID, req.Reason)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// RateRide handles rating a ride
func (h *Handler) RateRide(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	var req models.RideRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.RateRide(c.Request.Context(), rideID, riderID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to rate ride")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ride rated successfully"})
}

// GetMyRides handles getting rides for the current user
func (h *Handler) GetMyRides(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, err := middleware.GetUserRole(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	var rides []*models.Ride

	if role == models.RoleRider {
		rides, err = h.service.GetRiderRides(c.Request.Context(), userID, page, perPage)
	} else if role == models.RoleDriver {
		rides, err = h.service.GetDriverRides(c.Request.Context(), userID, page, perPage)
	} else {
		common.ErrorResponse(c, http.StatusForbidden, "invalid role")
		return
	}

	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get rides")
		return
	}

	common.SuccessResponse(c, rides)
}

// GetAvailableRides handles getting available ride requests for drivers
func (h *Handler) GetAvailableRides(c *gin.Context) {
	rides, err := h.service.GetAvailableRides(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get available rides")
		return
	}

	common.SuccessResponse(c, rides)
}

// RegisterRoutes registers ride routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtSecret string) {
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtSecret))

	// Rider routes
	riders := api.Group("/rides")
	riders.Use(middleware.RequireRole(models.RoleRider, models.RoleDriver))
	{
		riders.POST("", h.RequestRide)
		riders.GET("/:id", h.GetRide)
		riders.GET("", h.GetMyRides)
		riders.POST("/:id/cancel", h.CancelRide)
		riders.POST("/:id/rate", h.RateRide)
	}

	// Driver routes
	drivers := api.Group("/driver/rides")
	drivers.Use(middleware.RequireRole(models.RoleDriver))
	{
		drivers.GET("/available", h.GetAvailableRides)
		drivers.POST("/:id/accept", h.AcceptRide)
		drivers.POST("/:id/start", h.StartRide)
		drivers.POST("/:id/complete", h.CompleteRide)
	}
}
