package ridetypes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// AdminHandler handles admin HTTP requests for ride type management
type AdminHandler struct {
	service *Service
}

// NewAdminHandler creates a new ride types admin handler
func NewAdminHandler(service *Service) *AdminHandler {
	return &AdminHandler{service: service}
}

// RegisterRoutes registers ride type admin routes
func (h *AdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rt := rg.Group("/ride-types")
	{
		rt.GET("", h.ListRideTypes)
		rt.POST("", h.CreateRideType)
		rt.GET("/:id", h.GetRideType)
		rt.PUT("/:id", h.UpdateRideType)
		rt.DELETE("/:id", h.DeleteRideType)
	}
}

// ListRideTypes lists all ride types
func (h *AdminHandler) ListRideTypes(c *gin.Context) {
	params := pagination.ParseParams(c)
	includeInactive := c.Query("include_inactive") == "true"

	items, total, err := h.service.ListRideTypes(c.Request.Context(), params.Limit, params.Offset, includeInactive)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch ride types")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, items, meta)
}

// CreateRideType creates a new ride type
func (h *AdminHandler) CreateRideType(c *gin.Context) {
	var req CreateRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	rt := &RideType{
		Name:          req.Name,
		Description:   req.Description,
		BaseFare:      req.BaseFare,
		PerKmRate:     req.PerKmRate,
		PerMinuteRate: req.PerMinuteRate,
		MinimumFare:   req.MinimumFare,
		Capacity:      req.Capacity,
		IsActive:      req.IsActive,
	}

	if err := h.service.CreateRideType(c.Request.Context(), rt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to create ride type")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusCreated, rt, "Ride type created successfully")
}

// GetRideType retrieves a ride type by ID
func (h *AdminHandler) GetRideType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	rt, err := h.service.GetRideTypeByID(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "Ride type not found")
		return
	}

	common.SuccessResponse(c, rt)
}

// UpdateRideType updates a ride type
func (h *AdminHandler) UpdateRideType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	rt, err := h.service.GetRideTypeByID(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "Ride type not found")
		return
	}

	var req UpdateRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name != nil {
		rt.Name = *req.Name
	}
	if req.Description != nil {
		rt.Description = req.Description
	}
	if req.BaseFare != nil {
		rt.BaseFare = *req.BaseFare
	}
	if req.PerKmRate != nil {
		rt.PerKmRate = *req.PerKmRate
	}
	if req.PerMinuteRate != nil {
		rt.PerMinuteRate = *req.PerMinuteRate
	}
	if req.MinimumFare != nil {
		rt.MinimumFare = *req.MinimumFare
	}
	if req.Capacity != nil {
		rt.Capacity = *req.Capacity
	}
	if req.IsActive != nil {
		rt.IsActive = *req.IsActive
	}

	if err := h.service.UpdateRideType(c.Request.Context(), rt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to update ride type")
		return
	}

	common.SuccessResponse(c, rt)
}

// DeleteRideType soft-deletes a ride type
func (h *AdminHandler) DeleteRideType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	if err := h.service.DeleteRideType(c.Request.Context(), id); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete ride type")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride type deleted successfully")
}
