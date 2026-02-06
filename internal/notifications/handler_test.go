package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Service Interface and Implementation
// ============================================================================

// ServiceInterface defines the interface for the notifications service used by the handler
type ServiceInterface interface {
	SendNotification(ctx context.Context, userID uuid.UUID, notifType, channel, title, body string, data map[string]interface{}) (*models.Notification, error)
	ScheduleNotification(ctx context.Context, userID uuid.UUID, notifType, channel, title, body string, data map[string]interface{}, scheduledAt time.Time) (*models.Notification, error)
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID) error
	NotifyRideRequested(ctx context.Context, driverID, rideID uuid.UUID, pickupLocation string) error
	NotifyRideAccepted(ctx context.Context, riderID uuid.UUID, driverName string, eta int) error
	NotifyRideStarted(ctx context.Context, riderID uuid.UUID) error
	NotifyRideCompleted(ctx context.Context, riderID, driverID uuid.UUID, fare float64) error
	NotifyRideCancelled(ctx context.Context, userID uuid.UUID, cancelledBy string) error
	SendBulkNotification(ctx context.Context, userIDs []uuid.UUID, notifType, channel, title, body string, data map[string]interface{}) error
}

// MockService is a mock implementation of ServiceInterface
type MockService struct {
	mock.Mock
}

func (m *MockService) SendNotification(ctx context.Context, userID uuid.UUID, notifType, channel, title, body string, data map[string]interface{}) (*models.Notification, error) {
	args := m.Called(ctx, userID, notifType, channel, title, body, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockService) ScheduleNotification(ctx context.Context, userID uuid.UUID, notifType, channel, title, body string, data map[string]interface{}, scheduledAt time.Time) (*models.Notification, error) {
	args := m.Called(ctx, userID, notifType, channel, title, body, data, scheduledAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockService) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Notification, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*models.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockService) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockService) NotifyRideRequested(ctx context.Context, driverID, rideID uuid.UUID, pickupLocation string) error {
	args := m.Called(ctx, driverID, rideID, pickupLocation)
	return args.Error(0)
}

func (m *MockService) NotifyRideAccepted(ctx context.Context, riderID uuid.UUID, driverName string, eta int) error {
	args := m.Called(ctx, riderID, driverName, eta)
	return args.Error(0)
}

func (m *MockService) NotifyRideStarted(ctx context.Context, riderID uuid.UUID) error {
	args := m.Called(ctx, riderID)
	return args.Error(0)
}

func (m *MockService) NotifyRideCompleted(ctx context.Context, riderID, driverID uuid.UUID, fare float64) error {
	args := m.Called(ctx, riderID, driverID, fare)
	return args.Error(0)
}

func (m *MockService) NotifyRideCancelled(ctx context.Context, userID uuid.UUID, cancelledBy string) error {
	args := m.Called(ctx, userID, cancelledBy)
	return args.Error(0)
}

func (m *MockService) SendBulkNotification(ctx context.Context, userIDs []uuid.UUID, notifType, channel, title, body string, data map[string]interface{}) error {
	args := m.Called(ctx, userIDs, notifType, channel, title, body, data)
	return args.Error(0)
}

// ============================================================================
// Mockable Handler for Testing
// ============================================================================

// MockableHandler provides testable handler methods with mock service injection
type MockableHandler struct {
	service *MockService
}

func NewMockableHandler(mockService *MockService) *MockableHandler {
	return &MockableHandler{service: mockService}
}

// SendNotification sends a notification - testable version
func (h *MockableHandler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	notification, err := h.service.SendNotification(
		c.Request.Context(),
		userID,
		req.Type,
		req.Channel,
		req.Title,
		req.Body,
		req.Data,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, notification, "Notification sent successfully")
}

// ScheduleNotification schedules a notification - testable version
func (h *MockableHandler) ScheduleNotification(c *gin.Context) {
	var req ScheduleNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid scheduled_at format, use RFC3339")
		return
	}

	notification, err := h.service.ScheduleNotification(
		c.Request.Context(),
		userID,
		req.Type,
		req.Channel,
		req.Title,
		req.Body,
		req.Data,
		scheduledAt,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to schedule notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, notification, "Notification scheduled successfully")
}

// GetNotifications retrieves user's notifications - testable version
func (h *MockableHandler) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Default pagination
	limit := 20
	offset := 0

	notifications, total, err := h.service.GetUserNotifications(
		c.Request.Context(),
		userID.(uuid.UUID),
		limit,
		offset,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get notifications")
		return
	}

	common.SuccessResponseWithMeta(c, notifications, &common.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  total,
	})
}

// GetUnreadCount gets count of unread notifications - testable version
func (h *MockableHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get unread count")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, gin.H{"count": count}, "Unread count retrieved")
}

// MarkAsRead marks notification as read - testable version
func (h *MockableHandler) MarkAsRead(c *gin.Context) {
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid notification ID")
		return
	}

	err = h.service.MarkAsRead(c.Request.Context(), notificationID)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to mark as read")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Notification marked as read")
}

// NotifyRideRequested handles ride requested notification - testable version
func (h *MockableHandler) NotifyRideRequested(c *gin.Context) {
	var req RideNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	driverID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	rideID, err := uuid.Parse(req.RideID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	pickupLocation := ""
	if loc, ok := req.Data["pickup_location"].(string); ok {
		pickupLocation = loc
	}

	err = h.service.NotifyRideRequested(c.Request.Context(), driverID, rideID, pickupLocation)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride requested notification sent")
}

// NotifyRideAccepted handles ride accepted notification - testable version
func (h *MockableHandler) NotifyRideAccepted(c *gin.Context) {
	var req RideNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	riderID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rider ID")
		return
	}

	driverName := ""
	if name, ok := req.Data["driver_name"].(string); ok {
		driverName = name
	}

	eta := 5
	if e, ok := req.Data["eta"].(float64); ok {
		eta = int(e)
	}

	err = h.service.NotifyRideAccepted(c.Request.Context(), riderID, driverName, eta)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride accepted notification sent")
}

// NotifyRideStarted handles ride started notification - testable version
func (h *MockableHandler) NotifyRideStarted(c *gin.Context) {
	var req RideNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	riderID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rider ID")
		return
	}

	err = h.service.NotifyRideStarted(c.Request.Context(), riderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride started notification sent")
}

// NotifyRideCompleted handles ride completed notification - testable version
func (h *MockableHandler) NotifyRideCompleted(c *gin.Context) {
	var req struct {
		RiderID  string  `json:"rider_id" binding:"required"`
		DriverID string  `json:"driver_id" binding:"required"`
		Fare     float64 `json:"fare" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	riderID, err := uuid.Parse(req.RiderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rider ID")
		return
	}

	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	err = h.service.NotifyRideCompleted(c.Request.Context(), riderID, driverID, req.Fare)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride completed notification sent")
}

// NotifyRideCancelled handles ride cancelled notification - testable version
func (h *MockableHandler) NotifyRideCancelled(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		CancelledBy string `json:"cancelled_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	err = h.service.NotifyRideCancelled(c.Request.Context(), userID, req.CancelledBy)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride cancelled notification sent")
}

// SendBulkNotification sends notification to multiple users - testable version
func (h *MockableHandler) SendBulkNotification(c *gin.Context) {
	var req struct {
		UserIDs []string               `json:"user_ids" binding:"required"`
		Type    string                 `json:"type" binding:"required"`
		Channel string                 `json:"channel" binding:"required,oneof=push sms email"`
		Title   string                 `json:"title" binding:"required"`
		Body    string                 `json:"body" binding:"required"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	var userIDs []uuid.UUID
	for _, idStr := range req.UserIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID in list")
			return
		}
		userIDs = append(userIDs, id)
	}

	err := h.service.SendBulkNotification(
		c.Request.Context(),
		userIDs,
		req.Type,
		req.Channel,
		req.Title,
		req.Body,
		req.Data,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send bulk notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, gin.H{"sent": len(userIDs)}, "Bulk notifications sent")
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
	c.Set("user_email", "test@example.com")
	c.Set("user_role", "rider")
}

func setAdminContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
	c.Set("user_email", "admin@example.com")
	c.Set("user_role", "admin")
}

func createTestNotification(userID uuid.UUID, status string) *models.Notification {
	now := time.Now()
	return &models.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      "test_type",
		Channel:   "push",
		Title:     "Test Title",
		Body:      "Test Body",
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// ============================================================================
// SendNotification Handler Tests
// ============================================================================

func TestHandler_SendNotification_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedNotification := createTestNotification(userID, "pending")

	reqBody := SendNotificationRequest{
		UserID:  userID.String(),
		Type:    "test_type",
		Channel: "push",
		Title:   "Test Title",
		Body:    "Test Body",
		Data:    map[string]interface{}{"key": "value"},
	}

	mockService.On("SendNotification", mock.Anything, userID, "test_type", "push", "Test Title", "Test Body", reqBody.Data).
		Return(expectedNotification, nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

	handler.SendNotification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SendNotification_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := SendNotificationRequest{
		UserID:  "invalid-uuid",
		Type:    "test_type",
		Channel: "push",
		Title:   "Test Title",
		Body:    "Test Body",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

	handler.SendNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid user ID")
}

func TestHandler_SendNotification_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing user_id",
			body: map[string]interface{}{
				"type":    "test_type",
				"channel": "push",
				"title":   "Test Title",
				"body":    "Test Body",
			},
		},
		{
			name: "missing type",
			body: map[string]interface{}{
				"user_id": uuid.New().String(),
				"channel": "push",
				"title":   "Test Title",
				"body":    "Test Body",
			},
		},
		{
			name: "missing channel",
			body: map[string]interface{}{
				"user_id": uuid.New().String(),
				"type":    "test_type",
				"title":   "Test Title",
				"body":    "Test Body",
			},
		},
		{
			name: "missing title",
			body: map[string]interface{}{
				"user_id": uuid.New().String(),
				"type":    "test_type",
				"channel": "push",
				"body":    "Test Body",
			},
		},
		{
			name: "missing body",
			body: map[string]interface{}{
				"user_id": uuid.New().String(),
				"type":    "test_type",
				"channel": "push",
				"title":   "Test Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			c, w := setupTestContext("POST", "/api/v1/notifications/send", tt.body)

			handler.SendNotification(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_SendNotification_InvalidChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"user_id": uuid.New().String(),
		"type":    "test_type",
		"channel": "invalid_channel",
		"title":   "Test Title",
		"body":    "Test Body",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

	handler.SendNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendNotification_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := SendNotificationRequest{
		UserID:  userID.String(),
		Type:    "test_type",
		Channel: "push",
		Title:   "Test Title",
		Body:    "Test Body",
	}

	mockService.On("SendNotification", mock.Anything, userID, "test_type", "push", "Test Title", "Test Body", mock.Anything).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

	handler.SendNotification(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendNotification_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := SendNotificationRequest{
		UserID:  userID.String(),
		Type:    "test_type",
		Channel: "push",
		Title:   "Test Title",
		Body:    "Test Body",
	}

	mockService.On("SendNotification", mock.Anything, userID, "test_type", "push", "Test Title", "Test Body", mock.Anything).
		Return(nil, common.NewNotFoundError("user not found", nil))

	c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

	handler.SendNotification(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendNotification_AllChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	channels := []string{"push", "sms", "email"}

	for _, channel := range channels {
		t.Run("channel_"+channel, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			expectedNotification := createTestNotification(userID, "pending")
			expectedNotification.Channel = channel

			reqBody := SendNotificationRequest{
				UserID:  userID.String(),
				Type:    "test_type",
				Channel: channel,
				Title:   "Test Title",
				Body:    "Test Body",
			}

			mockService.On("SendNotification", mock.Anything, userID, "test_type", channel, "Test Title", "Test Body", mock.Anything).
				Return(expectedNotification, nil)

			c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

			handler.SendNotification(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_SendNotification_WithData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedNotification := createTestNotification(userID, "pending")
	data := map[string]interface{}{
		"ride_id":  "123",
		"action":   "accept",
		"priority": 1.0,
	}
	expectedNotification.Data = data

	reqBody := SendNotificationRequest{
		UserID:  userID.String(),
		Type:    "ride_request",
		Channel: "push",
		Title:   "New Ride Request",
		Body:    "You have a new ride request",
		Data:    data,
	}

	mockService.On("SendNotification", mock.Anything, userID, "ride_request", "push", "New Ride Request", "You have a new ride request", data).
		Return(expectedNotification, nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/send", reqBody)

	handler.SendNotification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendNotification_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/notifications/send", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/notifications/send", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ScheduleNotification Handler Tests
// ============================================================================

func TestHandler_ScheduleNotification_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	scheduledTime := time.Now().Add(1 * time.Hour)
	expectedNotification := createTestNotification(userID, "pending")
	expectedNotification.ScheduledAt = &scheduledTime

	reqBody := ScheduleNotificationRequest{
		UserID:      userID.String(),
		Type:        "promo",
		Channel:     "push",
		Title:       "Special Offer",
		Body:        "Get 20% off your next ride",
		ScheduledAt: scheduledTime.Format(time.RFC3339),
	}

	mockService.On("ScheduleNotification", mock.Anything, userID, "promo", "push", "Special Offer", "Get 20% off your next ride", mock.Anything, mock.AnythingOfType("time.Time")).
		Return(expectedNotification, nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/schedule", reqBody)

	handler.ScheduleNotification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ScheduleNotification_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := ScheduleNotificationRequest{
		UserID:      "invalid-uuid",
		Type:        "promo",
		Channel:     "push",
		Title:       "Special Offer",
		Body:        "Get 20% off",
		ScheduledAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/schedule", reqBody)

	handler.ScheduleNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid user ID")
}

func TestHandler_ScheduleNotification_InvalidDateFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := ScheduleNotificationRequest{
		UserID:      userID.String(),
		Type:        "promo",
		Channel:     "push",
		Title:       "Special Offer",
		Body:        "Get 20% off",
		ScheduledAt: "invalid-date",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/schedule", reqBody)

	handler.ScheduleNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "RFC3339")
}

func TestHandler_ScheduleNotification_MissingScheduledAt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"user_id": userID.String(),
		"type":    "promo",
		"channel": "push",
		"title":   "Special Offer",
		"body":    "Get 20% off",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/schedule", reqBody)

	handler.ScheduleNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ScheduleNotification_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	scheduledTime := time.Now().Add(1 * time.Hour)

	reqBody := ScheduleNotificationRequest{
		UserID:      userID.String(),
		Type:        "promo",
		Channel:     "push",
		Title:       "Special Offer",
		Body:        "Get 20% off",
		ScheduledAt: scheduledTime.Format(time.RFC3339),
	}

	mockService.On("ScheduleNotification", mock.Anything, userID, "promo", "push", "Special Offer", "Get 20% off", mock.Anything, mock.AnythingOfType("time.Time")).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/notifications/schedule", reqBody)

	handler.ScheduleNotification(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ScheduleNotification_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	scheduledTime := time.Now().Add(1 * time.Hour)

	reqBody := ScheduleNotificationRequest{
		UserID:      userID.String(),
		Type:        "promo",
		Channel:     "push",
		Title:       "Special Offer",
		Body:        "Get 20% off",
		ScheduledAt: scheduledTime.Format(time.RFC3339),
	}

	mockService.On("ScheduleNotification", mock.Anything, userID, "promo", "push", "Special Offer", "Get 20% off", mock.Anything, mock.AnythingOfType("time.Time")).
		Return(nil, common.NewBadRequestError("invalid schedule time", nil))

	c, w := setupTestContext("POST", "/api/v1/notifications/schedule", reqBody)

	handler.ScheduleNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetNotifications Handler Tests
// ============================================================================

func TestHandler_GetNotifications_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	notifications := []*models.Notification{
		createTestNotification(userID, "sent"),
		createTestNotification(userID, "sent"),
	}

	mockService.On("GetUserNotifications", mock.Anything, userID, 20, 0).
		Return(notifications, int64(2), nil)

	c, w := setupTestContext("GET", "/api/v1/notifications", nil)
	setUserContext(c, userID)

	handler.GetNotifications(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["meta"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetNotifications_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/notifications", nil)
	// Don't set user context

	handler.GetNotifications(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetNotifications_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	notifications := []*models.Notification{}

	mockService.On("GetUserNotifications", mock.Anything, userID, 20, 0).
		Return(notifications, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/notifications", nil)
	setUserContext(c, userID)

	handler.GetNotifications(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetNotifications_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUserNotifications", mock.Anything, userID, 20, 0).
		Return(nil, int64(0), errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/notifications", nil)
	setUserContext(c, userID)

	handler.GetNotifications(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetNotifications_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUserNotifications", mock.Anything, userID, 20, 0).
		Return(nil, int64(0), common.NewNotFoundError("user not found", nil))

	c, w := setupTestContext("GET", "/api/v1/notifications", nil)
	setUserContext(c, userID)

	handler.GetNotifications(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetUnreadCount Handler Tests
// ============================================================================

func TestHandler_GetUnreadCount_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUnreadCount", mock.Anything, userID).Return(5, nil)

	c, w := setupTestContext("GET", "/api/v1/notifications/unread/count", nil)
	setUserContext(c, userID)

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(5), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetUnreadCount_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/notifications/unread/count", nil)
	// Don't set user context

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetUnreadCount_ZeroCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUnreadCount", mock.Anything, userID).Return(0, nil)

	c, w := setupTestContext("GET", "/api/v1/notifications/unread/count", nil)
	setUserContext(c, userID)

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetUnreadCount_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUnreadCount", mock.Anything, userID).Return(0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/notifications/unread/count", nil)
	setUserContext(c, userID)

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// MarkAsRead Handler Tests
// ============================================================================

func TestHandler_MarkAsRead_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	notificationID := uuid.New()

	mockService.On("MarkAsRead", mock.Anything, notificationID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/"+notificationID.String()+"/read", nil)
	c.Params = gin.Params{{Key: "id", Value: notificationID.String()}}

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_MarkAsRead_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/notifications/invalid-uuid/read", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid notification ID")
}

func TestHandler_MarkAsRead_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/notifications//read", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MarkAsRead_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	notificationID := uuid.New()

	mockService.On("MarkAsRead", mock.Anything, notificationID).
		Return(common.NewNotFoundError("notification not found", nil))

	c, w := setupTestContext("POST", "/api/v1/notifications/"+notificationID.String()+"/read", nil)
	c.Params = gin.Params{{Key: "id", Value: notificationID.String()}}

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_MarkAsRead_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	notificationID := uuid.New()

	mockService.On("MarkAsRead", mock.Anything, notificationID).
		Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/notifications/"+notificationID.String()+"/read", nil)
	c.Params = gin.Params{{Key: "id", Value: notificationID.String()}}

	handler.MarkAsRead(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// NotifyRideRequested Handler Tests
// ============================================================================

func TestHandler_NotifyRideRequested_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: driverID.String(),
		RideID: rideID.String(),
		Data: map[string]interface{}{
			"pickup_location": "123 Main St",
		},
	}

	mockService.On("NotifyRideRequested", mock.Anything, driverID, rideID, "123 Main St").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/requested", reqBody)

	handler.NotifyRideRequested(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideRequested_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := RideNotificationRequest{
		UserID: "invalid-uuid",
		RideID: uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/requested", reqBody)

	handler.NotifyRideRequested(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid driver ID")
}

func TestHandler_NotifyRideRequested_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := RideNotificationRequest{
		UserID: uuid.New().String(),
		RideID: "invalid-uuid",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/requested", reqBody)

	handler.NotifyRideRequested(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

func TestHandler_NotifyRideRequested_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"ride_id": uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/requested", reqBody)

	handler.NotifyRideRequested(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NotifyRideRequested_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: driverID.String(),
		RideID: rideID.String(),
	}

	mockService.On("NotifyRideRequested", mock.Anything, driverID, rideID, "").
		Return(errors.New("notification failed"))

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/requested", reqBody)

	handler.NotifyRideRequested(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideRequested_EmptyPickupLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: driverID.String(),
		RideID: rideID.String(),
		Data:   map[string]interface{}{},
	}

	mockService.On("NotifyRideRequested", mock.Anything, driverID, rideID, "").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/requested", reqBody)

	handler.NotifyRideRequested(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// NotifyRideAccepted Handler Tests
// ============================================================================

func TestHandler_NotifyRideAccepted_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: riderID.String(),
		Data: map[string]interface{}{
			"driver_name": "John Doe",
			"eta":         float64(10),
		},
	}

	mockService.On("NotifyRideAccepted", mock.Anything, riderID, "John Doe", 10).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/accepted", reqBody)

	handler.NotifyRideAccepted(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideAccepted_InvalidRiderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := RideNotificationRequest{
		UserID: "invalid-uuid",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/accepted", reqBody)

	handler.NotifyRideAccepted(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NotifyRideAccepted_DefaultETA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: riderID.String(),
		Data: map[string]interface{}{
			"driver_name": "Jane Smith",
		},
	}

	// Default ETA is 5
	mockService.On("NotifyRideAccepted", mock.Anything, riderID, "Jane Smith", 5).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/accepted", reqBody)

	handler.NotifyRideAccepted(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideAccepted_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: riderID.String(),
	}

	mockService.On("NotifyRideAccepted", mock.Anything, riderID, "", 5).
		Return(errors.New("notification failed"))

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/accepted", reqBody)

	handler.NotifyRideAccepted(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// NotifyRideStarted Handler Tests
// ============================================================================

func TestHandler_NotifyRideStarted_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: riderID.String(),
	}

	mockService.On("NotifyRideStarted", mock.Anything, riderID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/started", reqBody)

	handler.NotifyRideStarted(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideStarted_InvalidRiderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := RideNotificationRequest{
		UserID: "invalid-uuid",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/started", reqBody)

	handler.NotifyRideStarted(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NotifyRideStarted_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/started", reqBody)

	handler.NotifyRideStarted(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NotifyRideStarted_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()

	reqBody := RideNotificationRequest{
		UserID: riderID.String(),
	}

	mockService.On("NotifyRideStarted", mock.Anything, riderID).
		Return(errors.New("notification failed"))

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/started", reqBody)

	handler.NotifyRideStarted(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// NotifyRideCompleted Handler Tests
// ============================================================================

func TestHandler_NotifyRideCompleted_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"rider_id":  riderID.String(),
		"driver_id": driverID.String(),
		"fare":      25.50,
	}

	mockService.On("NotifyRideCompleted", mock.Anything, riderID, driverID, 25.50).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/completed", reqBody)

	handler.NotifyRideCompleted(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideCompleted_InvalidRiderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"rider_id":  "invalid-uuid",
		"driver_id": uuid.New().String(),
		"fare":      25.50,
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/completed", reqBody)

	handler.NotifyRideCompleted(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid rider ID")
}

func TestHandler_NotifyRideCompleted_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"rider_id":  uuid.New().String(),
		"driver_id": "invalid-uuid",
		"fare":      25.50,
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/completed", reqBody)

	handler.NotifyRideCompleted(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid driver ID")
}

func TestHandler_NotifyRideCompleted_MissingFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"rider_id":  uuid.New().String(),
		"driver_id": uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/completed", reqBody)

	handler.NotifyRideCompleted(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NotifyRideCompleted_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"rider_id":  riderID.String(),
		"driver_id": driverID.String(),
		"fare":      25.50,
	}

	mockService.On("NotifyRideCompleted", mock.Anything, riderID, driverID, 25.50).
		Return(errors.New("notification failed"))

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/completed", reqBody)

	handler.NotifyRideCompleted(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideCompleted_ZeroFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	driverID := uuid.New()

	// Fare of 0 should fail validation (binding:"required")
	reqBody := map[string]interface{}{
		"rider_id":  riderID.String(),
		"driver_id": driverID.String(),
		"fare":      0,
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/completed", reqBody)

	handler.NotifyRideCompleted(c)

	// 0 fails the required validation
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// NotifyRideCancelled Handler Tests
// ============================================================================

func TestHandler_NotifyRideCancelled_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"user_id":      userID.String(),
		"cancelled_by": "rider",
	}

	mockService.On("NotifyRideCancelled", mock.Anything, userID, "rider").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/cancelled", reqBody)

	handler.NotifyRideCancelled(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideCancelled_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"user_id":      "invalid-uuid",
		"cancelled_by": "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/cancelled", reqBody)

	handler.NotifyRideCancelled(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid user ID")
}

func TestHandler_NotifyRideCancelled_MissingCancelledBy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"user_id": uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/cancelled", reqBody)

	handler.NotifyRideCancelled(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_NotifyRideCancelled_CancelledByDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"user_id":      userID.String(),
		"cancelled_by": "driver",
	}

	mockService.On("NotifyRideCancelled", mock.Anything, userID, "driver").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/cancelled", reqBody)

	handler.NotifyRideCancelled(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_NotifyRideCancelled_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"user_id":      userID.String(),
		"cancelled_by": "system",
	}

	mockService.On("NotifyRideCancelled", mock.Anything, userID, "system").
		Return(errors.New("notification failed"))

	c, w := setupTestContext("POST", "/api/v1/notifications/ride/cancelled", reqBody)

	handler.NotifyRideCancelled(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SendBulkNotification Handler Tests
// ============================================================================

func TestHandler_SendBulkNotification_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID1 := uuid.New()
	userID2 := uuid.New()
	userID3 := uuid.New()

	reqBody := map[string]interface{}{
		"user_ids": []string{userID1.String(), userID2.String(), userID3.String()},
		"type":     "promo",
		"channel":  "push",
		"title":    "Special Offer",
		"body":     "Get 20% off your next ride",
	}

	mockService.On("SendBulkNotification", mock.Anything, []uuid.UUID{userID1, userID2, userID3}, "promo", "push", "Special Offer", "Get 20% off your next ride", mock.Anything).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["sent"])
	mockService.AssertExpectations(t)
}

func TestHandler_SendBulkNotification_EmptyUserIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"user_ids": []string{},
		"type":     "promo",
		"channel":  "push",
		"title":    "Special Offer",
		"body":     "Get 20% off",
	}

	// Mock expects empty slice
	mockService.On("SendBulkNotification", mock.Anything, []uuid.UUID(nil), "promo", "push", "Special Offer", "Get 20% off", mock.Anything).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	// Empty array passes validation but sends to 0 users
	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendBulkNotification_InvalidUserIDInList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"user_ids": []string{uuid.New().String(), "invalid-uuid", uuid.New().String()},
		"type":     "promo",
		"channel":  "push",
		"title":    "Special Offer",
		"body":     "Get 20% off",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid user ID in list")
}

func TestHandler_SendBulkNotification_InvalidChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"user_ids": []string{uuid.New().String()},
		"type":     "promo",
		"channel":  "invalid_channel",
		"title":    "Special Offer",
		"body":     "Get 20% off",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendBulkNotification_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing user_ids",
			body: map[string]interface{}{
				"type":    "promo",
				"channel": "push",
				"title":   "Special Offer",
				"body":    "Get 20% off",
			},
		},
		{
			name: "missing type",
			body: map[string]interface{}{
				"user_ids": []string{uuid.New().String()},
				"channel":  "push",
				"title":    "Special Offer",
				"body":     "Get 20% off",
			},
		},
		{
			name: "missing channel",
			body: map[string]interface{}{
				"user_ids": []string{uuid.New().String()},
				"type":     "promo",
				"title":    "Special Offer",
				"body":     "Get 20% off",
			},
		},
		{
			name: "missing title",
			body: map[string]interface{}{
				"user_ids": []string{uuid.New().String()},
				"type":     "promo",
				"channel":  "push",
				"body":     "Get 20% off",
			},
		},
		{
			name: "missing body",
			body: map[string]interface{}{
				"user_ids": []string{uuid.New().String()},
				"type":     "promo",
				"channel":  "push",
				"title":    "Special Offer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", tt.body)
			setAdminContext(c, uuid.New())

			handler.SendBulkNotification(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_SendBulkNotification_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"user_ids": []string{userID.String()},
		"type":     "promo",
		"channel":  "push",
		"title":    "Special Offer",
		"body":     "Get 20% off",
	}

	mockService.On("SendBulkNotification", mock.Anything, []uuid.UUID{userID}, "promo", "push", "Special Offer", "Get 20% off", mock.Anything).
		Return(errors.New("service error"))

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendBulkNotification_AllChannels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	channels := []string{"push", "sms", "email"}

	for _, channel := range channels {
		t.Run("channel_"+channel, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()

			reqBody := map[string]interface{}{
				"user_ids": []string{userID.String()},
				"type":     "promo",
				"channel":  channel,
				"title":    "Special Offer",
				"body":     "Get 20% off",
			}

			mockService.On("SendBulkNotification", mock.Anything, []uuid.UUID{userID}, "promo", channel, "Special Offer", "Get 20% off", mock.Anything).
				Return(nil)

			c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
			setAdminContext(c, uuid.New())

			handler.SendBulkNotification(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_SendBulkNotification_WithData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	data := map[string]interface{}{
		"promo_code": "SAVE20",
		"expires":    "2024-12-31",
	}

	reqBody := map[string]interface{}{
		"user_ids": []string{userID.String()},
		"type":     "promo",
		"channel":  "push",
		"title":    "Special Offer",
		"body":     "Get 20% off",
		"data":     data,
	}

	mockService.On("SendBulkNotification", mock.Anything, []uuid.UUID{userID}, "promo", "push", "Special Offer", "Get 20% off", data).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendBulkNotification_LargeUserList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	var userIDStrings []string
	var userIDs []uuid.UUID
	for i := 0; i < 100; i++ {
		id := uuid.New()
		userIDStrings = append(userIDStrings, id.String())
		userIDs = append(userIDs, id)
	}

	reqBody := map[string]interface{}{
		"user_ids": userIDStrings,
		"type":     "system",
		"channel":  "push",
		"title":    "System Update",
		"body":     "App update available",
	}

	mockService.On("SendBulkNotification", mock.Anything, userIDs, "system", "push", "System Update", "App update available", mock.Anything).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/notifications/bulk", reqBody)
	setAdminContext(c, uuid.New())

	handler.SendBulkNotification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(100), data["sent"])
	mockService.AssertExpectations(t)
}
