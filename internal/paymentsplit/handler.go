package paymentsplit

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// Handler handles HTTP requests for payment splitting
type Handler struct {
	service *Service
}

// NewHandler creates a new payment split handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// SPLIT ENDPOINTS
// ========================================

// CreateSplit creates a new payment split
// POST /api/v1/splits
func (h *Handler) CreateSplit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSplitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.service.CreateSplit(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create split")
		return
	}

	common.SuccessResponse(c, response)
}

// GetSplit gets a payment split with details
// GET /api/v1/splits/:id
func (h *Handler) GetSplit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	splitID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid split ID")
		return
	}

	response, err := h.service.GetSplit(c.Request.Context(), userID, splitID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "split not found")
		return
	}

	common.SuccessResponse(c, response)
}

// GetSplitByRide gets the split for a specific ride
// GET /api/v1/rides/:id/split
func (h *Handler) GetSplitByRide(c *gin.Context) {
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

	response, err := h.service.GetSplitByRide(c.Request.Context(), userID, rideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "no split found for this ride")
		return
	}

	common.SuccessResponse(c, response)
}

// RespondToSplit handles a participant's response to a split invitation
// POST /api/v1/splits/:id/respond
func (h *Handler) RespondToSplit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	splitID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid split ID")
		return
	}

	var req RespondToSplitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RespondToSplit(c.Request.Context(), userID, splitID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to respond to split")
		return
	}

	action := "accepted"
	if !req.Accept {
		action = "declined"
	}

	common.SuccessResponse(c, gin.H{
		"message": "Split invitation " + action,
	})
}

// PaySplit processes payment for a participant's share
// POST /api/v1/splits/:id/pay
func (h *Handler) PaySplit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	splitID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid split ID")
		return
	}

	var req struct {
		PaymentMethod string `json:"payment_method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.PaySplit(c.Request.Context(), userID, splitID, req.PaymentMethod); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to process payment")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Payment processed successfully",
	})
}

// CancelSplit cancels a payment split
// DELETE /api/v1/splits/:id
func (h *Handler) CancelSplit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	splitID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid split ID")
		return
	}

	if err := h.service.CancelSplit(c.Request.Context(), userID, splitID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel split")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Split cancelled successfully",
	})
}

// ========================================
// GROUP ENDPOINTS
// ========================================

// CreateGroup creates a saved split group
// POST /api/v1/splits/groups
func (h *Handler) CreateGroup(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	group, err := h.service.CreateGroup(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create group")
		return
	}

	common.SuccessResponse(c, group)
}

// ListGroups lists saved split groups
// GET /api/v1/splits/groups
func (h *Handler) ListGroups(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	groups, err := h.service.ListGroups(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list groups")
		return
	}

	common.SuccessResponse(c, gin.H{
		"groups": groups,
		"count":  len(groups),
	})
}

// DeleteGroup deletes a saved split group
// DELETE /api/v1/splits/groups/:id
func (h *Handler) DeleteGroup(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid group ID")
		return
	}

	if err := h.service.DeleteGroup(c.Request.Context(), userID, groupID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to delete group")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Group deleted successfully",
	})
}

// ========================================
// HISTORY ENDPOINT
// ========================================

// GetHistory gets payment split history
// GET /api/v1/splits/history
func (h *Handler) GetHistory(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := pagination.ParseParams(c)

	history, err := h.service.GetHistory(c.Request.Context(), userID, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get history")
		return
	}

	common.SuccessResponse(c, history)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers payment split routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	splits := r.Group("/api/v1/splits")
	splits.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Split management
		splits.POST("", h.CreateSplit)
		splits.GET("/:id", h.GetSplit)
		splits.DELETE("/:id", h.CancelSplit)
		splits.POST("/:id/respond", h.RespondToSplit)
		splits.POST("/:id/pay", h.PaySplit)

		// History
		splits.GET("/history", h.GetHistory)

		// Groups
		splits.POST("/groups", h.CreateGroup)
		splits.GET("/groups", h.ListGroups)
		splits.DELETE("/groups/:id", h.DeleteGroup)
	}

	// Ride-scoped split lookup
	rides := r.Group("/api/v1/rides")
	rides.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		rides.GET("/:id/split", h.GetSplitByRide)
	}
}
