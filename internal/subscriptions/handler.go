package subscriptions

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// Handler handles HTTP requests for subscriptions
type Handler struct {
	service *Service
}

// NewHandler creates a new subscription handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// PLAN ENDPOINTS (Public)
// ========================================

// ListPlans lists available subscription plans
// GET /api/v1/subscriptions/plans
func (h *Handler) ListPlans(c *gin.Context) {
	params := pagination.ParseParams(c)

	plans, err := h.service.ListPlans(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list plans")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(plans)))
	common.SuccessResponseWithMeta(c, plans, meta)
}

// ComparePlans returns plans with personalized savings estimates
// GET /api/v1/subscriptions/compare
func (h *Handler) ComparePlans(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	comparison, err := h.service.ComparePlans(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to compare plans")
		return
	}

	common.SuccessResponse(c, comparison)
}

// ========================================
// SUBSCRIPTION ENDPOINTS
// ========================================

// Subscribe subscribes the user to a plan
// POST /api/v1/subscriptions
func (h *Handler) Subscribe(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.service.Subscribe(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to subscribe")
		return
	}

	common.SuccessResponse(c, response)
}

// GetSubscription gets the user's active subscription
// GET /api/v1/subscriptions/me
func (h *Handler) GetSubscription(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	response, err := h.service.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.SuccessResponse(c, gin.H{
			"has_subscription": false,
			"message":          "No active subscription",
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"has_subscription": true,
		"subscription":     response,
	})
}

// PauseSubscription pauses the user's subscription
// POST /api/v1/subscriptions/me/pause
func (h *Handler) PauseSubscription(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.PauseSubscription(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to pause subscription")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Subscription paused"})
}

// ResumeSubscription resumes the user's paused subscription
// POST /api/v1/subscriptions/me/resume
func (h *Handler) ResumeSubscription(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.ResumeSubscription(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to resume subscription")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Subscription resumed"})
}

// CancelSubscription cancels the user's subscription
// DELETE /api/v1/subscriptions/me
func (h *Handler) CancelSubscription(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.service.CancelSubscription(c.Request.Context(), userID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel subscription")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Subscription cancelled"})
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// CreatePlan creates a new subscription plan (admin)
// POST /api/v1/admin/subscriptions/plans
func (h *Handler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	plan, err := h.service.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create plan")
		return
	}

	common.SuccessResponse(c, plan)
}

// ListAllPlans lists all plans including inactive (admin)
// GET /api/v1/admin/subscriptions/plans
func (h *Handler) ListAllPlans(c *gin.Context) {
	params := pagination.ParseParams(c)

	plans, err := h.service.ListAllPlans(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list plans")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(plans)))
	common.SuccessResponseWithMeta(c, plans, meta)
}

// DeactivatePlan deactivates a plan (admin)
// POST /api/v1/admin/subscriptions/plans/:id/deactivate
func (h *Handler) DeactivatePlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid plan ID")
		return
	}

	if err := h.service.DeactivatePlan(c.Request.Context(), planID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to deactivate plan")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Plan deactivated"})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers subscription routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Public plan listing
	subs := r.Group("/api/v1/subscriptions")
	subs.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Plans
		subs.GET("/plans", h.ListPlans)
		subs.GET("/compare", h.ComparePlans)

		// Subscription management
		subs.POST("", h.Subscribe)
		subs.GET("/me", h.GetSubscription)
		subs.POST("/me/pause", h.PauseSubscription)
		subs.POST("/me/resume", h.ResumeSubscription)
		subs.DELETE("/me", h.CancelSubscription)
	}

	// Admin plan management
	admin := r.Group("/api/v1/admin/subscriptions")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.POST("/plans", h.CreatePlan)
		admin.GET("/plans", h.ListAllPlans)
		admin.POST("/plans/:id/deactivate", h.DeactivatePlan)
	}
}
