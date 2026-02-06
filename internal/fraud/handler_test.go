package fraud

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// FraudServiceInterface defines the interface for the fraud service used by the handler
type FraudServiceInterface interface {
	GetPendingAlerts(ctx context.Context, limit, offset int) ([]*FraudAlert, int64, error)
	GetAlert(ctx context.Context, alertID uuid.UUID) (*FraudAlert, error)
	CreateAlert(ctx context.Context, alert *FraudAlert) error
	InvestigateAlert(ctx context.Context, alertID, investigatorID uuid.UUID, notes string) error
	ResolveAlert(ctx context.Context, alertID, investigatorID uuid.UUID, confirmed bool, notes, actionTaken string) error
	GetUserAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*FraudAlert, int64, error)
	GetUserRiskProfile(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error)
	AnalyzeUser(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error)
	SuspendUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error
	ReinstateUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error
	DetectPaymentFraud(ctx context.Context, userID uuid.UUID) error
	DetectRideFraud(ctx context.Context, userID uuid.UUID) error
	DetectAccountFraud(ctx context.Context, userID uuid.UUID) error
	GetFraudStatistics(ctx context.Context, startDate, endDate time.Time) (*FraudStatistics, error)
	GetFraudPatterns(ctx context.Context, limit int) ([]*FraudPattern, error)
}

// MockFraudService is a mock implementation of FraudServiceInterface
type MockFraudService struct {
	mock.Mock
}

func (m *MockFraudService) GetPendingAlerts(ctx context.Context, limit, offset int) ([]*FraudAlert, int64, error) {
	args := m.Called(ctx, limit, offset)
	alerts, _ := args.Get(0).([]*FraudAlert)
	return alerts, int64(args.Int(1)), args.Error(2)
}

func (m *MockFraudService) GetAlert(ctx context.Context, alertID uuid.UUID) (*FraudAlert, error) {
	args := m.Called(ctx, alertID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FraudAlert), args.Error(1)
}

func (m *MockFraudService) CreateAlert(ctx context.Context, alert *FraudAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockFraudService) InvestigateAlert(ctx context.Context, alertID, investigatorID uuid.UUID, notes string) error {
	args := m.Called(ctx, alertID, investigatorID, notes)
	return args.Error(0)
}

func (m *MockFraudService) ResolveAlert(ctx context.Context, alertID, investigatorID uuid.UUID, confirmed bool, notes, actionTaken string) error {
	args := m.Called(ctx, alertID, investigatorID, confirmed, notes, actionTaken)
	return args.Error(0)
}

func (m *MockFraudService) GetUserAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*FraudAlert, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	alerts, _ := args.Get(0).([]*FraudAlert)
	return alerts, int64(args.Int(1)), args.Error(2)
}

func (m *MockFraudService) GetUserRiskProfile(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserRiskProfile), args.Error(1)
}

func (m *MockFraudService) AnalyzeUser(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserRiskProfile), args.Error(1)
}

func (m *MockFraudService) SuspendUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error {
	args := m.Called(ctx, userID, adminID, reason)
	return args.Error(0)
}

func (m *MockFraudService) ReinstateUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error {
	args := m.Called(ctx, userID, adminID, reason)
	return args.Error(0)
}

func (m *MockFraudService) DetectPaymentFraud(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockFraudService) DetectRideFraud(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockFraudService) DetectAccountFraud(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockFraudService) GetFraudStatistics(ctx context.Context, startDate, endDate time.Time) (*FraudStatistics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FraudStatistics), args.Error(1)
}

func (m *MockFraudService) GetFraudPatterns(ctx context.Context, limit int) ([]*FraudPattern, error) {
	args := m.Called(ctx, limit)
	patterns, _ := args.Get(0).([]*FraudPattern)
	return patterns, args.Error(1)
}

// MockableHandler provides testable handler methods with mock service injection
type MockableHandler struct {
	service *MockFraudService
}

func NewMockableHandler(mockService *MockFraudService) *MockableHandler {
	return &MockableHandler{service: mockService}
}

// Handler methods that use the mock service
func (h *MockableHandler) GetPendingAlerts(c *gin.Context) {
	limit := 20
	offset := 0

	alerts, total, err := h.service.GetPendingAlerts(c.Request.Context(), limit, offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pending alerts")
		return
	}

	common.SuccessResponseWithMeta(c, alerts, &common.Meta{Limit: limit, Offset: offset, Total: total})
}

func (h *MockableHandler) GetAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	alert, err := h.service.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	common.SuccessResponse(c, alert)
}

func (h *MockableHandler) CreateAlert(c *gin.Context) {
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	alert := &FraudAlert{
		UserID:      req.UserID,
		AlertType:   req.AlertType,
		AlertLevel:  req.AlertLevel,
		Description: req.Description,
		Details:     req.Details,
		RiskScore:   req.RiskScore,
	}

	if err := h.service.CreateAlert(c.Request.Context(), alert); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.CreatedResponse(c, alert)
}

func (h *MockableHandler) InvestigateAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	var req InvestigateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	adminID := c.GetString("user_id")
	investigatorID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid user ID")
		return
	}

	if err := h.service.InvestigateAlert(c.Request.Context(), alertID, investigatorID, req.Notes); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "alert marked as investigating"})
}

func (h *MockableHandler) ResolveAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	var req ResolveAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	adminID := c.GetString("user_id")
	investigatorID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid user ID")
		return
	}

	if err := h.service.ResolveAlert(c.Request.Context(), alertID, investigatorID, req.Confirmed, req.Notes, req.ActionTaken); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "alert resolved successfully"})
}

func (h *MockableHandler) GetUserAlerts(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	limit := 20
	offset := 0

	alerts, total, err := h.service.GetUserAlerts(c.Request.Context(), userID, limit, offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get user alerts")
		return
	}

	common.SuccessResponseWithMeta(c, alerts, &common.Meta{Limit: limit, Offset: offset, Total: total})
}

func (h *MockableHandler) GetUserRiskProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	profile, err := h.service.GetUserRiskProfile(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, profile)
}

func (h *MockableHandler) AnalyzeUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	profile, err := h.service.AnalyzeUser(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, profile)
}

func (h *MockableHandler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req SuspendUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	adminID := c.GetString("user_id")
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid admin ID")
		return
	}

	if err := h.service.SuspendUser(c.Request.Context(), userID, adminUUID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "user account suspended"})
}

func (h *MockableHandler) ReinstateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req ReinstateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	adminID := c.GetString("user_id")
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid admin ID")
		return
	}

	if err := h.service.ReinstateUser(c.Request.Context(), userID, adminUUID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "user account reinstated"})
}

func (h *MockableHandler) DetectPaymentFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DetectPaymentFraud(c.Request.Context(), userID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "payment fraud detection completed"})
}

func (h *MockableHandler) DetectRideFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DetectRideFraud(c.Request.Context(), userID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ride fraud detection completed"})
}

func (h *MockableHandler) DetectAccountFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DetectAccountFraud(c.Request.Context(), userID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "account fraud detection completed"})
}

func (h *MockableHandler) GetFraudStatistics(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid start_date format (use YYYY-MM-DD)")
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid end_date format (use YYYY-MM-DD)")
		return
	}

	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	stats, err := h.service.GetFraudStatistics(c.Request.Context(), start, end)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, stats)
}

func (h *MockableHandler) GetFraudPatterns(c *gin.Context) {
	limit := 50

	patterns, err := h.service.GetFraudPatterns(c.Request.Context(), limit)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if patterns == nil {
		patterns = []*FraudPattern{}
	}

	common.SuccessResponse(c, patterns)
}

// Helper functions for setting up test context
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

func setAdminContext(c *gin.Context, adminID uuid.UUID) {
	c.Set("user_id", adminID.String())
	c.Set("user_role", "admin")
	c.Set("user_email", "admin@example.com")
}

func createTestAlert(userID uuid.UUID, status FraudAlertStatus, level FraudAlertLevel) *FraudAlert {
	return &FraudAlert{
		ID:          uuid.New(),
		UserID:      userID,
		AlertType:   AlertTypePaymentFraud,
		AlertLevel:  level,
		Status:      status,
		Description: "Test fraud alert",
		Details:     map[string]interface{}{"test": true},
		RiskScore:   75.0,
		DetectedAt:  time.Now(),
	}
}

func createTestRiskProfile(userID uuid.UUID) *UserRiskProfile {
	return &UserRiskProfile{
		UserID:              userID,
		RiskScore:           45.0,
		TotalAlerts:         3,
		CriticalAlerts:      1,
		ConfirmedFraudCases: 0,
		AccountSuspended:    false,
		LastUpdated:         time.Now(),
	}
}

func createTestStatistics() *FraudStatistics {
	return &FraudStatistics{
		Period:                 "2024-01-01 to 2024-01-31",
		TotalAlerts:            150,
		CriticalAlerts:         10,
		HighAlerts:             35,
		MediumAlerts:           60,
		LowAlerts:              45,
		ConfirmedFraudCases:    25,
		FalsePositives:         30,
		PendingInvestigation:   20,
		EstimatedLossPrevented: 50000.00,
		AverageResponseTime:    45,
	}
}

func createTestPattern() *FraudPattern {
	return &FraudPattern{
		PatternType:   "multiple_chargebacks",
		Description:   "Users with multiple chargebacks in short period",
		Occurrences:   15,
		AffectedUsers: []uuid.UUID{uuid.New(), uuid.New()},
		FirstDetected: time.Now().AddDate(0, -1, 0),
		LastDetected:  time.Now(),
		Details:       map[string]interface{}{"threshold": 3},
		Severity:      AlertLevelHigh,
	}
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// ============================================================================
// GetPendingAlerts Handler Tests
// ============================================================================

func TestHandler_GetPendingAlerts_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	alerts := []*FraudAlert{
		createTestAlert(userID, AlertStatusPending, AlertLevelHigh),
		createTestAlert(userID, AlertStatusPending, AlertLevelMedium),
	}

	mockService.On("GetPendingAlerts", mock.Anything, 20, 0).Return(alerts, 2, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts", nil)

	handler.GetPendingAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	assert.NotNil(t, response["meta"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetPendingAlerts_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetPendingAlerts", mock.Anything, 20, 0).Return([]*FraudAlert{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts", nil)

	handler.GetPendingAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetPendingAlerts_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetPendingAlerts", mock.Anything, 20, 0).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts", nil)

	handler.GetPendingAlerts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetPendingAlerts_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetPendingAlerts", mock.Anything, 20, 0).Return(nil, 0, common.NewInternalServerError("service unavailable"))

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts", nil)

	handler.GetPendingAlerts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetAlert Handler Tests
// ============================================================================

func TestHandler_GetAlert_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	alertID := uuid.New()
	alert := createTestAlert(userID, AlertStatusPending, AlertLevelHigh)
	alert.ID = alertID

	mockService.On("GetAlert", mock.Anything, alertID).Return(alert, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}

	handler.GetAlert(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetAlert_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid alert ID")
}

func TestHandler_GetAlert_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()

	mockService.On("GetAlert", mock.Anything, alertID).Return(nil, errors.New("alert not found"))

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}

	handler.GetAlert(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetAlert_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/fraud/alerts/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CreateAlert Handler Tests
// ============================================================================

func TestHandler_CreateAlert_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := CreateAlertRequest{
		UserID:      userID,
		AlertType:   AlertTypePaymentFraud,
		AlertLevel:  AlertLevelHigh,
		Description: "Suspicious payment activity detected",
		Details:     map[string]interface{}{"chargebacks": 3},
		RiskScore:   85.0,
	}

	mockService.On("CreateAlert", mock.Anything, mock.AnythingOfType("*fraud.FraudAlert")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

	handler.CreateAlert(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreateAlert_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/fraud/alerts", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/fraud/alerts", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateAlert_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing user_id",
			body: map[string]interface{}{
				"alert_type":  "payment_fraud",
				"alert_level": "high",
				"description": "Test",
				"risk_score":  75.0,
			},
		},
		{
			name: "missing alert_type",
			body: map[string]interface{}{
				"user_id":     uuid.New().String(),
				"alert_level": "high",
				"description": "Test",
				"risk_score":  75.0,
			},
		},
		{
			name: "missing description",
			body: map[string]interface{}{
				"user_id":     uuid.New().String(),
				"alert_type":  "payment_fraud",
				"alert_level": "high",
				"risk_score":  75.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestContext("POST", "/api/v1/fraud/alerts", tt.body)

			handler.CreateAlert(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_CreateAlert_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := CreateAlertRequest{
		UserID:      userID,
		AlertType:   AlertTypePaymentFraud,
		AlertLevel:  AlertLevelHigh,
		Description: "Test alert",
		RiskScore:   75.0,
	}

	mockService.On("CreateAlert", mock.Anything, mock.AnythingOfType("*fraud.FraudAlert")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

	handler.CreateAlert(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateAlert_InvalidRiskScore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	tests := []struct {
		name      string
		riskScore float64
	}{
		{name: "negative risk score", riskScore: -10.0},
		{name: "risk score over 100", riskScore: 150.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"user_id":     uuid.New().String(),
				"alert_type":  "payment_fraud",
				"alert_level": "high",
				"description": "Test",
				"risk_score":  tt.riskScore,
			}

			c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

			handler.CreateAlert(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// ============================================================================
// InvestigateAlert Handler Tests
// ============================================================================

func TestHandler_InvestigateAlert_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	adminID := uuid.New()
	reqBody := InvestigateAlertRequest{
		Notes: "Starting investigation",
	}

	mockService.On("InvestigateAlert", mock.Anything, alertID, adminID, "Starting investigation").Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/investigate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.InvestigateAlert(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_InvestigateAlert_InvalidAlertID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	reqBody := InvestigateAlertRequest{Notes: "Test"}

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/invalid/investigate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setAdminContext(c, adminID)

	handler.InvestigateAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid alert ID")
}

func TestHandler_InvestigateAlert_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	reqBody := InvestigateAlertRequest{Notes: "Test"}

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/investigate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	c.Set("user_id", "invalid-uuid")

	handler.InvestigateAlert(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_InvestigateAlert_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	adminID := uuid.New()
	reqBody := InvestigateAlertRequest{Notes: "Test"}

	mockService.On("InvestigateAlert", mock.Anything, alertID, adminID, "Test").Return(errors.New("service error"))

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/investigate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.InvestigateAlert(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_InvestigateAlert_EmptyNotes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	adminID := uuid.New()
	reqBody := InvestigateAlertRequest{Notes: ""}

	mockService.On("InvestigateAlert", mock.Anything, alertID, adminID, "").Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/investigate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.InvestigateAlert(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ResolveAlert Handler Tests
// ============================================================================

func TestHandler_ResolveAlert_Success_Confirmed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	adminID := uuid.New()
	reqBody := ResolveAlertRequest{
		Confirmed:   true,
		Notes:       "Confirmed fraud case",
		ActionTaken: "User account suspended",
	}

	mockService.On("ResolveAlert", mock.Anything, alertID, adminID, true, "Confirmed fraud case", "User account suspended").Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.ResolveAlert(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ResolveAlert_Success_FalsePositive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	adminID := uuid.New()
	reqBody := ResolveAlertRequest{
		Confirmed:   false,
		Notes:       "False positive - legitimate activity",
		ActionTaken: "No action needed",
	}

	mockService.On("ResolveAlert", mock.Anything, alertID, adminID, false, "False positive - legitimate activity", "No action needed").Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.ResolveAlert(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ResolveAlert_InvalidAlertID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	reqBody := ResolveAlertRequest{Confirmed: true, Notes: "Test"}

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/invalid/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setAdminContext(c, adminID)

	handler.ResolveAlert(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ResolveAlert_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	reqBody := ResolveAlertRequest{Confirmed: true, Notes: "Test"}

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	c.Set("user_id", "invalid-uuid")

	handler.ResolveAlert(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ResolveAlert_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	alertID := uuid.New()
	adminID := uuid.New()
	reqBody := ResolveAlertRequest{Confirmed: true, Notes: "Test", ActionTaken: "Test"}

	mockService.On("ResolveAlert", mock.Anything, alertID, adminID, true, "Test", "Test").Return(errors.New("service error"))

	c, w := setupTestContext("PUT", "/api/v1/fraud/alerts/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.ResolveAlert(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetUserAlerts Handler Tests
// ============================================================================

func TestHandler_GetUserAlerts_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	alerts := []*FraudAlert{
		createTestAlert(userID, AlertStatusPending, AlertLevelHigh),
		createTestAlert(userID, AlertStatusConfirmed, AlertLevelCritical),
	}

	mockService.On("GetUserAlerts", mock.Anything, userID, 20, 0).Return(alerts, 2, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/users/"+userID.String()+"/alerts", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.GetUserAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetUserAlerts_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/fraud/users/invalid/alerts", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.GetUserAlerts(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetUserAlerts_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUserAlerts", mock.Anything, userID, 20, 0).Return([]*FraudAlert{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/users/"+userID.String()+"/alerts", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.GetUserAlerts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetUserAlerts_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUserAlerts", mock.Anything, userID, 20, 0).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/fraud/users/"+userID.String()+"/alerts", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.GetUserAlerts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetUserRiskProfile Handler Tests
// ============================================================================

func TestHandler_GetUserRiskProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	profile := createTestRiskProfile(userID)

	mockService.On("GetUserRiskProfile", mock.Anything, userID).Return(profile, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/users/"+userID.String()+"/risk-profile", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.GetUserRiskProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetUserRiskProfile_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/fraud/users/invalid/risk-profile", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.GetUserRiskProfile(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetUserRiskProfile_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUserRiskProfile", mock.Anything, userID).Return(nil, errors.New("profile not found"))

	c, w := setupTestContext("GET", "/api/v1/fraud/users/"+userID.String()+"/risk-profile", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.GetUserRiskProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// AnalyzeUser Handler Tests
// ============================================================================

func TestHandler_AnalyzeUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	profile := createTestRiskProfile(userID)
	profile.RiskScore = 75.0

	mockService.On("AnalyzeUser", mock.Anything, userID).Return(profile, nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/analyze", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.AnalyzeUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_AnalyzeUser_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/fraud/users/invalid/analyze", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.AnalyzeUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AnalyzeUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("AnalyzeUser", mock.Anything, userID).Return(nil, errors.New("analysis failed"))

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/analyze", nil)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}

	handler.AnalyzeUser(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SuspendUser Handler Tests
// ============================================================================

func TestHandler_SuspendUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	adminID := uuid.New()
	reqBody := SuspendUserRequest{Reason: "Confirmed fraud activity"}

	mockService.On("SuspendUser", mock.Anything, userID, adminID, "Confirmed fraud activity").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	setAdminContext(c, adminID)

	handler.SuspendUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SuspendUser_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	reqBody := SuspendUserRequest{Reason: "Test"}

	c, w := setupTestContext("POST", "/api/v1/fraud/users/invalid/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setAdminContext(c, adminID)

	handler.SuspendUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SuspendUser_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	adminID := uuid.New()
	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	setAdminContext(c, adminID)

	handler.SuspendUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SuspendUser_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := SuspendUserRequest{Reason: "Test"}

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	c.Set("user_id", "invalid-uuid")

	handler.SuspendUser(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SuspendUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	adminID := uuid.New()
	reqBody := SuspendUserRequest{Reason: "Test"}

	mockService.On("SuspendUser", mock.Anything, userID, adminID, "Test").Return(errors.New("suspension failed"))

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	setAdminContext(c, adminID)

	handler.SuspendUser(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ReinstateUser Handler Tests
// ============================================================================

func TestHandler_ReinstateUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	adminID := uuid.New()
	reqBody := ReinstateUserRequest{Reason: "User verified identity"}

	mockService.On("ReinstateUser", mock.Anything, userID, adminID, "User verified identity").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/reinstate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	setAdminContext(c, adminID)

	handler.ReinstateUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ReinstateUser_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	reqBody := ReinstateUserRequest{Reason: "Test"}

	c, w := setupTestContext("POST", "/api/v1/fraud/users/invalid/reinstate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setAdminContext(c, adminID)

	handler.ReinstateUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReinstateUser_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	adminID := uuid.New()
	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/reinstate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	setAdminContext(c, adminID)

	handler.ReinstateUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReinstateUser_InvalidAdminID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := ReinstateUserRequest{Reason: "Test"}

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/reinstate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	c.Set("user_id", "invalid-uuid")

	handler.ReinstateUser(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ReinstateUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	adminID := uuid.New()
	reqBody := ReinstateUserRequest{Reason: "Test"}

	mockService.On("ReinstateUser", mock.Anything, userID, adminID, "Test").Return(errors.New("reinstatement failed"))

	c, w := setupTestContext("POST", "/api/v1/fraud/users/"+userID.String()+"/reinstate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: userID.String()}}
	setAdminContext(c, adminID)

	handler.ReinstateUser(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// DetectPaymentFraud Handler Tests
// ============================================================================

func TestHandler_DetectPaymentFraud_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("DetectPaymentFraud", mock.Anything, userID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/payment/"+userID.String(), nil)
	c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

	handler.DetectPaymentFraud(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_DetectPaymentFraud_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/payment/invalid", nil)
	c.Params = gin.Params{{Key: "user_id", Value: "invalid"}}

	handler.DetectPaymentFraud(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DetectPaymentFraud_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("DetectPaymentFraud", mock.Anything, userID).Return(errors.New("detection failed"))

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/payment/"+userID.String(), nil)
	c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

	handler.DetectPaymentFraud(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// DetectRideFraud Handler Tests
// ============================================================================

func TestHandler_DetectRideFraud_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("DetectRideFraud", mock.Anything, userID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/ride/"+userID.String(), nil)
	c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

	handler.DetectRideFraud(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_DetectRideFraud_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/ride/invalid", nil)
	c.Params = gin.Params{{Key: "user_id", Value: "invalid"}}

	handler.DetectRideFraud(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DetectRideFraud_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("DetectRideFraud", mock.Anything, userID).Return(errors.New("detection failed"))

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/ride/"+userID.String(), nil)
	c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

	handler.DetectRideFraud(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// DetectAccountFraud Handler Tests
// ============================================================================

func TestHandler_DetectAccountFraud_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("DetectAccountFraud", mock.Anything, userID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/account/"+userID.String(), nil)
	c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

	handler.DetectAccountFraud(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_DetectAccountFraud_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/account/invalid", nil)
	c.Params = gin.Params{{Key: "user_id", Value: "invalid"}}

	handler.DetectAccountFraud(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DetectAccountFraud_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("DetectAccountFraud", mock.Anything, userID).Return(errors.New("detection failed"))

	c, w := setupTestContext("POST", "/api/v1/fraud/detect/account/"+userID.String(), nil)
	c.Params = gin.Params{{Key: "user_id", Value: userID.String()}}

	handler.DetectAccountFraud(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetFraudStatistics Handler Tests
// ============================================================================

func TestHandler_GetFraudStatistics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	stats := createTestStatistics()

	mockService.On("GetFraudStatistics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/statistics?start_date=2024-01-01&end_date=2024-01-31", nil)
	c.Request.URL.RawQuery = "start_date=2024-01-01&end_date=2024-01-31"

	handler.GetFraudStatistics(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetFraudStatistics_DefaultDates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	stats := createTestStatistics()

	mockService.On("GetFraudStatistics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/statistics", nil)

	handler.GetFraudStatistics(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetFraudStatistics_InvalidStartDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/fraud/statistics?start_date=invalid&end_date=2024-01-31", nil)
	c.Request.URL.RawQuery = "start_date=invalid&end_date=2024-01-31"

	handler.GetFraudStatistics(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid start_date format")
}

func TestHandler_GetFraudStatistics_InvalidEndDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/fraud/statistics?start_date=2024-01-01&end_date=invalid", nil)
	c.Request.URL.RawQuery = "start_date=2024-01-01&end_date=invalid"

	handler.GetFraudStatistics(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid end_date format")
}

func TestHandler_GetFraudStatistics_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetFraudStatistics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/fraud/statistics?start_date=2024-01-01&end_date=2024-01-31", nil)
	c.Request.URL.RawQuery = "start_date=2024-01-01&end_date=2024-01-31"

	handler.GetFraudStatistics(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetFraudPatterns Handler Tests
// ============================================================================

func TestHandler_GetFraudPatterns_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	patterns := []*FraudPattern{
		createTestPattern(),
		createTestPattern(),
	}

	mockService.On("GetFraudPatterns", mock.Anything, 50).Return(patterns, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/patterns", nil)

	handler.GetFraudPatterns(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetFraudPatterns_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetFraudPatterns", mock.Anything, 50).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/fraud/patterns", nil)

	handler.GetFraudPatterns(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetFraudPatterns_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetFraudPatterns", mock.Anything, 50).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/fraud/patterns", nil)

	handler.GetFraudPatterns(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Edge Cases and Integration Tests
// ============================================================================

func TestHandler_CreateAlert_AllAlertTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	alertTypes := []FraudAlertType{
		AlertTypePaymentFraud,
		AlertTypeAccountFraud,
		AlertTypeLocationFraud,
		AlertTypeRideFraud,
		AlertTypeRatingManipulation,
		AlertTypePromoAbuse,
	}

	for _, alertType := range alertTypes {
		t.Run(string(alertType), func(t *testing.T) {
			mockService := new(MockFraudService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			reqBody := CreateAlertRequest{
				UserID:      userID,
				AlertType:   alertType,
				AlertLevel:  AlertLevelHigh,
				Description: "Test alert for " + string(alertType),
				RiskScore:   75.0,
			}

			mockService.On("CreateAlert", mock.Anything, mock.AnythingOfType("*fraud.FraudAlert")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

			handler.CreateAlert(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_CreateAlert_AllAlertLevels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	alertLevels := []FraudAlertLevel{
		AlertLevelLow,
		AlertLevelMedium,
		AlertLevelHigh,
		AlertLevelCritical,
	}

	for _, alertLevel := range alertLevels {
		t.Run(string(alertLevel), func(t *testing.T) {
			mockService := new(MockFraudService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			reqBody := CreateAlertRequest{
				UserID:      userID,
				AlertType:   AlertTypePaymentFraud,
				AlertLevel:  alertLevel,
				Description: "Test alert with level " + string(alertLevel),
				RiskScore:   75.0,
			}

			mockService.On("CreateAlert", mock.Anything, mock.AnythingOfType("*fraud.FraudAlert")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

			handler.CreateAlert(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_UUIDFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		uuid           string
		expectedStatus int
	}{
		{
			name:           "invalid format - too short",
			uuid:           "123",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid format - wrong characters",
			uuid:           "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty uuid",
			uuid:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid uuid - special characters",
			uuid:           "!@#$%^&*()_+-={}|[]",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFraudService)
			handler := NewMockableHandler(mockService)

			c, w := setupTestContext("GET", "/api/v1/fraud/alerts/test", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.uuid}}

			handler.GetAlert(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_RiskScoreBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		riskScore      float64
		expectedStatus int
		setupMock      bool
	}{
		{
			name:           "minimum non-zero score",
			riskScore:      0.1,
			expectedStatus: http.StatusCreated,
			setupMock:      true,
		},
		{
			name:           "maximum valid score",
			riskScore:      100.0,
			expectedStatus: http.StatusCreated,
			setupMock:      true,
		},
		{
			name:           "mid-range score",
			riskScore:      50.0,
			expectedStatus: http.StatusCreated,
			setupMock:      true,
		},
		{
			name:           "zero score treated as missing required field",
			riskScore:      0.0,
			expectedStatus: http.StatusBadRequest,
			setupMock:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockFraudService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			reqBody := CreateAlertRequest{
				UserID:      userID,
				AlertType:   AlertTypePaymentFraud,
				AlertLevel:  AlertLevelHigh,
				Description: "Test alert",
				RiskScore:   tt.riskScore,
			}

			if tt.setupMock {
				mockService.On("CreateAlert", mock.Anything, mock.AnythingOfType("*fraud.FraudAlert")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

			handler.CreateAlert(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_ConcurrentAlertRetrieval(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	alerts := []*FraudAlert{
		createTestAlert(userID, AlertStatusPending, AlertLevelHigh),
	}

	mockService.On("GetPendingAlerts", mock.Anything, 20, 0).Return(alerts, 1, nil).Times(5)

	for i := 0; i < 5; i++ {
		c, w := setupTestContext("GET", "/api/v1/fraud/alerts", nil)

		handler.GetPendingAlerts(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
}

func TestHandler_AlertWithDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockFraudService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	details := map[string]interface{}{
		"chargebacks":              3,
		"failed_payment_attempts":  5,
		"multiple_payment_methods": true,
		"suspicious_ip":            "192.168.1.1",
		"device_fingerprint":       "abc123",
	}

	reqBody := CreateAlertRequest{
		UserID:      userID,
		AlertType:   AlertTypePaymentFraud,
		AlertLevel:  AlertLevelCritical,
		Description: "Complex fraud pattern detected",
		Details:     details,
		RiskScore:   95.0,
	}

	mockService.On("CreateAlert", mock.Anything, mock.MatchedBy(func(alert *FraudAlert) bool {
		// JSON unmarshaling converts numbers to float64
		chargebacks, _ := alert.Details["chargebacks"].(float64)
		failedAttempts, _ := alert.Details["failed_payment_attempts"].(float64)
		return chargebacks == 3 &&
			failedAttempts == 5 &&
			alert.RiskScore == 95.0
	})).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/fraud/alerts", reqBody)

	handler.CreateAlert(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}
