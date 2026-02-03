package recording

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for recordings
type Handler struct {
	service *Service
}

// NewHandler creates a new recording handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// USER ENDPOINTS
// ========================================

// StartRecording starts a new recording
// POST /api/v1/recordings/start
func (h *Handler) StartRecording(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Determine user type from context
	userType := "rider"
	if role, exists := c.Get("role"); exists {
		if role == models.RoleDriver {
			userType = "driver"
		}
	}

	var req StartRecordingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.StartRecording(c.Request.Context(), userID, userType, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start recording")
		return
	}

	common.SuccessResponse(c, result)
}

// StopRecording stops an active recording
// POST /api/v1/recordings/stop
func (h *Handler) StopRecording(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req StopRecordingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.StopRecording(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to stop recording")
		return
	}

	common.SuccessResponse(c, result)
}

// CompleteUpload completes a recording upload
// POST /api/v1/recordings/complete
func (h *Handler) CompleteUpload(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CompleteUpload(c.Request.Context(), userID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to complete upload")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Upload completed successfully",
	})
}

// GetRecording gets a recording
// GET /api/v1/recordings/:id
func (h *Handler) GetRecording(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	recordingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid recording ID")
		return
	}

	result, err := h.service.GetRecording(c.Request.Context(), userID, recordingID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "recording not found")
		return
	}

	common.SuccessResponse(c, result)
}

// GetRecordingsForRide gets all recordings for a ride
// GET /api/v1/recordings/ride/:ride_id
func (h *Handler) GetRecordingsForRide(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("ride_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	recordings, err := h.service.GetRecordingsForRide(c.Request.Context(), userID, rideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get recordings")
		return
	}

	common.SuccessResponse(c, gin.H{
		"recordings": recordings,
	})
}

// DeleteRecording deletes a recording
// DELETE /api/v1/recordings/:id
func (h *Handler) DeleteRecording(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	recordingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid recording ID")
		return
	}

	if err := h.service.DeleteRecording(c.Request.Context(), userID, recordingID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to delete recording")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Recording deleted successfully",
	})
}

// RecordConsent records user's consent for recording
// POST /api/v1/recordings/consent
func (h *Handler) RecordConsent(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	userType := "rider"
	if role, exists := c.Get("role"); exists {
		if role == models.RoleDriver {
			userType = "driver"
		}
	}

	var req RecordingConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	if err := h.service.RecordConsent(c.Request.Context(), userID, userType, &req, &ipAddress, &userAgent); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to record consent")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Consent recorded successfully",
	})
}

// GetSettings gets recording settings
// GET /api/v1/recordings/settings
func (h *Handler) GetSettings(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	settings, err := h.service.GetSettings(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get settings")
		return
	}

	common.SuccessResponse(c, settings)
}

// UpdateSettings updates recording settings
// PUT /api/v1/recordings/settings
func (h *Handler) UpdateSettings(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var settings RecordingSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateSettings(c.Request.Context(), userID, &settings); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update settings")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Settings updated successfully",
	})
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// AdminGetRecording gets a recording for admin review
// GET /api/v1/admin/recordings/:id
func (h *Handler) AdminGetRecording(c *gin.Context) {
	adminID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	recordingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid recording ID")
		return
	}

	reason := c.DefaultQuery("reason", "admin review")

	result, err := h.service.AdminGetRecording(c.Request.Context(), adminID, recordingID, reason)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "recording not found")
		return
	}

	common.SuccessResponse(c, result)
}

// GetRecordingStats gets recording statistics
// GET /api/v1/admin/recordings/stats
func (h *Handler) GetRecordingStats(c *gin.Context) {
	stats, err := h.service.GetRecordingStats(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get statistics")
		return
	}

	common.SuccessResponse(c, stats)
}

// ExtendRetention extends recording retention
// POST /api/v1/admin/recordings/:id/extend
func (h *Handler) ExtendRetention(c *gin.Context) {
	recordingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid recording ID")
		return
	}

	var req struct {
		Policy RetentionPolicy `json:"policy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ExtendRetention(c.Request.Context(), recordingID, req.Policy); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to extend retention")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Retention extended successfully",
	})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers recording routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// User recording routes
	recordings := r.Group("/api/v1/recordings")
	recordings.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		recordings.POST("/start", h.StartRecording)
		recordings.POST("/stop", h.StopRecording)
		recordings.POST("/complete", h.CompleteUpload)
		recordings.GET("/:id", h.GetRecording)
		recordings.GET("/ride/:ride_id", h.GetRecordingsForRide)
		recordings.DELETE("/:id", h.DeleteRecording)
		recordings.POST("/consent", h.RecordConsent)
		recordings.GET("/settings", h.GetSettings)
		recordings.PUT("/settings", h.UpdateSettings)
	}

	// Admin routes
	admin := r.Group("/api/v1/admin/recordings")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.GET("/:id", h.AdminGetRecording)
		admin.GET("/stats", h.GetRecordingStats)
		admin.POST("/:id/extend", h.ExtendRetention)
	}
}

// RegisterRoutesOnGroup registers recording routes on an existing router group
func (h *Handler) RegisterRoutesOnGroup(rg *gin.RouterGroup) {
	recordings := rg.Group("/recordings")
	{
		recordings.POST("/start", h.StartRecording)
		recordings.POST("/stop", h.StopRecording)
		recordings.POST("/complete", h.CompleteUpload)
		recordings.GET("/:id", h.GetRecording)
		recordings.GET("/ride/:ride_id", h.GetRecordingsForRide)
		recordings.DELETE("/:id", h.DeleteRecording)
		recordings.POST("/consent", h.RecordConsent)
		recordings.GET("/settings", h.GetSettings)
		recordings.PUT("/settings", h.UpdateSettings)
	}
}
