package experiments

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

// Handler handles HTTP requests for experiments and feature flags
type Handler struct {
	service *Service
}

// NewHandler creates a new experiments handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// FEATURE FLAG ENDPOINTS
// ========================================

// EvaluateFlag evaluates a single feature flag for the authenticated user
// GET /api/v1/flags/:key
func (h *Handler) EvaluateFlag(c *gin.Context) {
	key := c.Param("key")
	userCtx := h.buildUserContext(c)

	result, err := h.service.EvaluateFlag(c.Request.Context(), key, userCtx)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to evaluate flag")
		return
	}

	common.SuccessResponse(c, result)
}

// EvaluateFlags evaluates multiple feature flags for the authenticated user
// POST /api/v1/flags/evaluate
func (h *Handler) EvaluateFlags(c *gin.Context) {
	var req struct {
		Keys []string `json:"keys" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	userCtx := h.buildUserContext(c)

	result, err := h.service.EvaluateFlags(c.Request.Context(), req.Keys, userCtx)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to evaluate flags")
		return
	}

	common.SuccessResponse(c, result)
}

// CreateFlag creates a new feature flag (admin)
// POST /api/v1/admin/flags
func (h *Handler) CreateFlag(c *gin.Context) {
	adminID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	flag, err := h.service.CreateFlag(c.Request.Context(), adminID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create flag")
		return
	}

	common.SuccessResponse(c, flag)
}

// ListFlags lists all feature flags (admin)
// GET /api/v1/admin/flags
func (h *Handler) ListFlags(c *gin.Context) {
	params := pagination.ParseParams(c)

	var status *FlagStatus
	if s := c.Query("status"); s != "" {
		st := FlagStatus(s)
		status = &st
	}

	flags, err := h.service.ListFlags(c.Request.Context(), status, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list flags")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(flags)))
	common.SuccessResponseWithMeta(c, flags, meta)
}

// GetFlag gets a flag by key (admin)
// GET /api/v1/admin/flags/:key
func (h *Handler) GetFlag(c *gin.Context) {
	key := c.Param("key")

	flag, err := h.service.GetFlag(c.Request.Context(), key)
	if err != nil || flag == nil {
		common.ErrorResponse(c, http.StatusNotFound, "flag not found")
		return
	}

	common.SuccessResponse(c, flag)
}

// UpdateFlag updates a feature flag (admin)
// PUT /api/v1/admin/flags/:id
func (h *Handler) UpdateFlag(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	var req UpdateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateFlag(c.Request.Context(), flagID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update flag")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Flag updated successfully"})
}

// ToggleFlag quickly enables or disables a flag (admin)
// POST /api/v1/admin/flags/:id/toggle
func (h *Handler) ToggleFlag(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ToggleFlag(c.Request.Context(), flagID, req.Enabled); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to toggle flag")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Flag toggled"})
}

// ArchiveFlag archives a feature flag (admin)
// DELETE /api/v1/admin/flags/:id
func (h *Handler) ArchiveFlag(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	if err := h.service.ArchiveFlag(c.Request.Context(), flagID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to archive flag")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Flag archived"})
}

// CreateOverride creates a per-user flag override (admin)
// POST /api/v1/admin/flags/:id/overrides
func (h *Handler) CreateOverride(c *gin.Context) {
	adminID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	var req CreateOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CreateOverride(c.Request.Context(), adminID, flagID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create override")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Override created"})
}

// ListOverrides lists overrides for a flag (admin)
// GET /api/v1/admin/flags/:id/overrides
func (h *Handler) ListOverrides(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	params := pagination.ParseParams(c)

	overrides, err := h.service.ListOverrides(c.Request.Context(), flagID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list overrides")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(overrides)))
	common.SuccessResponseWithMeta(c, overrides, meta)
}

// ========================================
// EXPERIMENT ENDPOINTS
// ========================================

// CreateExperiment creates a new A/B experiment (admin)
// POST /api/v1/admin/experiments
func (h *Handler) CreateExperiment(c *gin.Context) {
	adminID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	experiment, err := h.service.CreateExperiment(c.Request.Context(), adminID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create experiment")
		return
	}

	common.SuccessResponse(c, experiment)
}

// ListExperiments lists experiments (admin)
// GET /api/v1/admin/experiments
func (h *Handler) ListExperiments(c *gin.Context) {
	params := pagination.ParseParams(c)

	var status *ExperimentStatus
	if s := c.Query("status"); s != "" {
		st := ExperimentStatus(s)
		status = &st
	}

	experiments, err := h.service.ListExperiments(c.Request.Context(), status, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list experiments")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(experiments)))
	common.SuccessResponseWithMeta(c, experiments, meta)
}

// GetExperiment gets an experiment with variants (admin)
// GET /api/v1/admin/experiments/:id
func (h *Handler) GetExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	experiment, variants, err := h.service.GetExperiment(c.Request.Context(), experimentID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "experiment not found")
		return
	}

	common.SuccessResponse(c, gin.H{
		"experiment": experiment,
		"variants":   variants,
	})
}

// StartExperiment starts a draft experiment (admin)
// POST /api/v1/admin/experiments/:id/start
func (h *Handler) StartExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	if err := h.service.StartExperiment(c.Request.Context(), experimentID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start experiment")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Experiment started"})
}

// PauseExperiment pauses a running experiment (admin)
// POST /api/v1/admin/experiments/:id/pause
func (h *Handler) PauseExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	if err := h.service.PauseExperiment(c.Request.Context(), experimentID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to pause experiment")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Experiment paused"})
}

// ConcludeExperiment ends an experiment (admin)
// POST /api/v1/admin/experiments/:id/conclude
func (h *Handler) ConcludeExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	if err := h.service.ConcludeExperiment(c.Request.Context(), experimentID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to conclude experiment")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Experiment concluded"})
}

// GetResults gets experiment results with statistical analysis (admin)
// GET /api/v1/admin/experiments/:id/results
func (h *Handler) GetResults(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	results, err := h.service.GetResults(c.Request.Context(), experimentID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get results")
		return
	}

	common.SuccessResponse(c, results)
}

// GetVariant gets the user's assigned variant for an experiment
// GET /api/v1/experiments/:key/variant
func (h *Handler) GetVariant(c *gin.Context) {
	key := c.Param("key")
	userCtx := h.buildUserContext(c)

	variant, err := h.service.GetVariantForUser(c.Request.Context(), key, userCtx)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get variant")
		return
	}

	if variant == nil {
		common.SuccessResponse(c, gin.H{
			"enrolled": false,
			"variant":  nil,
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"enrolled": true,
		"variant":  variant,
	})
}

// TrackEvent records an experiment event
// POST /api/v1/experiments/track
func (h *Handler) TrackEvent(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.TrackEvent(c.Request.Context(), userID, &req); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to track event")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "event tracked"})
}

// ========================================
// HELPERS
// ========================================

func (h *Handler) buildUserContext(c *gin.Context) *UserContext {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return nil
	}

	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	return &UserContext{
		UserID:   userID,
		Role:     roleStr,
		Country:  c.GetHeader("X-Country"),
		City:     c.GetHeader("X-City"),
		Platform: c.GetHeader("X-Platform"),
		AppVersion: c.GetHeader("X-App-Version"),
	}
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers experiment and feature flag routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Client-facing flag evaluation
	flags := r.Group("/api/v1/flags")
	flags.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		flags.GET("/:key", h.EvaluateFlag)
		flags.POST("/evaluate", h.EvaluateFlags)
	}

	// Client-facing experiment endpoints
	experiments := r.Group("/api/v1/experiments")
	experiments.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		experiments.GET("/:key/variant", h.GetVariant)
		experiments.POST("/track", h.TrackEvent)
	}

	// Admin flag management
	adminFlags := r.Group("/api/v1/admin/flags")
	adminFlags.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	adminFlags.Use(middleware.RequireRole(models.RoleAdmin))
	{
		adminFlags.POST("", h.CreateFlag)
		adminFlags.GET("", h.ListFlags)
		adminFlags.GET("/:key", h.GetFlag)
		adminFlags.PUT("/:id", h.UpdateFlag)
		adminFlags.POST("/:id/toggle", h.ToggleFlag)
		adminFlags.DELETE("/:id", h.ArchiveFlag)
		adminFlags.POST("/:id/overrides", h.CreateOverride)
		adminFlags.GET("/:id/overrides", h.ListOverrides)
	}

	// Admin experiment management
	adminExperiments := r.Group("/api/v1/admin/experiments")
	adminExperiments.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	adminExperiments.Use(middleware.RequireRole(models.RoleAdmin))
	{
		adminExperiments.POST("", h.CreateExperiment)
		adminExperiments.GET("", h.ListExperiments)
		adminExperiments.GET("/:id", h.GetExperiment)
		adminExperiments.POST("/:id/start", h.StartExperiment)
		adminExperiments.POST("/:id/pause", h.PauseExperiment)
		adminExperiments.POST("/:id/conclude", h.ConcludeExperiment)
		adminExperiments.GET("/:id/results", h.GetResults)
	}
}
