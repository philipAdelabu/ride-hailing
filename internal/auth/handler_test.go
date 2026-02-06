package auth

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
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// Mock Service
// ============================================================================

// MockAuthService is a mock implementation of the auth service for handler testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoginResponse), args.Error(1)
}

func (m *MockAuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, updates *models.User) (*models.User, error) {
	args := m.Called(ctx, userID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// ============================================================================
// Testable Handler (wraps mock service)
// ============================================================================

// TestableHandler provides a testable handler with mock service injection
type TestableHandler struct {
	service *MockAuthService
}

func NewTestableHandler(mockService *MockAuthService) *TestableHandler {
	return &TestableHandler{service: mockService}
}

// Register handles user registration - testable version
func (h *TestableHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "registration failed")
		return
	}

	common.CreatedResponse(c, user)
}

// Login handles user login - testable version
func (h *TestableHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "login failed")
		return
	}

	common.SuccessResponse(c, response)
}

// GetProfile handles getting user profile - testable version
func (h *TestableHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetProfile(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get profile")
		return
	}

	common.SuccessResponse(c, user)
}

// UpdateProfile handles updating user profile - testable version
func (h *TestableHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var updates models.User
	if err := c.ShouldBindJSON(&updates); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID.(uuid.UUID), &updates)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update profile")
		return
	}

	common.SuccessResponse(c, user)
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

func setUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func createTestUser() *models.User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	return &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PhoneNumber:  "+1234567890",
		PasswordHash: string(hashedPassword),
		FirstName:    "John",
		LastName:     "Doe",
		Role:         models.RoleRider,
		IsActive:     true,
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestUserWithEmail(email string) *models.User {
	user := createTestUser()
	user.Email = email
	return user
}

func createTestUserWithRole(role models.UserRole) *models.User {
	user := createTestUser()
	user.Role = role
	return user
}

func createTestLoginResponse(user *models.User) *models.LoginResponse {
	return &models.LoginResponse{
		User:  user,
		Token: "test-jwt-token-abc123",
	}
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// ============================================================================
// Register Handler Tests
// ============================================================================

func TestHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	expectedUser := createTestUser()
	expectedUser.PasswordHash = "" // Should be cleared in response

	reqBody := models.RegisterRequest{
		Email:       "newuser@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(expectedUser, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/auth/register", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_Register_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"password":     "SecurePassword123!",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"last_name":    "Smith",
		"role":         "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_MissingPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "newuser@example.com",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"last_name":    "Smith",
		"role":         "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_MissingPhoneNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":      "newuser@example.com",
		"password":   "SecurePassword123!",
		"first_name": "Jane",
		"last_name":  "Smith",
		"role":       "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_MissingFirstName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "newuser@example.com",
		"password":     "SecurePassword123!",
		"phone_number": "+1234567890",
		"last_name":    "Smith",
		"role":         "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_MissingLastName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "newuser@example.com",
		"password":     "SecurePassword123!",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"role":         "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "newuser@example.com",
		"password":     "SecurePassword123!",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"last_name":    "Smith",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "not-a-valid-email",
		"password":     "SecurePassword123!",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"last_name":    "Smith",
		"role":         "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_PasswordTooShort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "newuser@example.com",
		"password":     "short",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"last_name":    "Smith",
		"role":         "rider",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_InvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        "newuser@example.com",
		"password":     "SecurePassword123!",
		"phone_number": "+1234567890",
		"first_name":   "Jane",
		"last_name":    "Smith",
		"role":         "invalid_role",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_UserAlreadyExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.RegisterRequest{
		Email:       "existing@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(nil, common.NewConflictError("user with this email already exists"))

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Register_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.RegisterRequest{
		Email:       "newuser@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(nil, errors.New("database connection error"))

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Register_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.RegisterRequest{
		Email:       "newuser@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(nil, common.NewInternalServerError("failed to hash password"))

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Register_RiderRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	expectedUser := createTestUserWithRole(models.RoleRider)
	expectedUser.PasswordHash = ""

	reqBody := models.RegisterRequest{
		Email:       "rider@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Rider",
		LastName:    "User",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(expectedUser, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Register_DriverRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	expectedUser := createTestUserWithRole(models.RoleDriver)
	expectedUser.PasswordHash = ""

	reqBody := models.RegisterRequest{
		Email:       "driver@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Driver",
		LastName:    "User",
		Role:        models.RoleDriver,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(expectedUser, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Register_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/auth/register", map[string]interface{}{})

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_NullValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":        nil,
		"password":     nil,
		"phone_number": nil,
		"first_name":   nil,
		"last_name":    nil,
		"role":         nil,
	}

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Login Handler Tests
// ============================================================================

func TestHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	testUser := createTestUser()
	testUser.PasswordHash = ""
	expectedResponse := createTestLoginResponse(testUser)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/auth/login", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"password": "password123",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_MissingPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email": "test@example.com",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":    "not-a-valid-email",
		"password": "password123",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, common.NewUnauthorizedError("invalid credentials"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Login_InvalidPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, common.NewUnauthorizedError("invalid credentials"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Login_InactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "inactive@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, common.NewUnauthorizedError("account is inactive"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Login_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Login_TokenGenerationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, common.NewInternalServerError("failed to generate token"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Login_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/auth/login", map[string]interface{}{})

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_EmptyEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":    "",
		"password": "password123",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_EmptyPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_WhitespaceEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"email":    "   ",
		"password": "password123",
	}

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_SuccessfulResponseContainsToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	testUser := createTestUser()
	testUser.PasswordHash = ""
	expectedResponse := &models.LoginResponse{
		User:  testUser,
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token",
	}

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["token"])
	mockService.AssertExpectations(t)
}

func TestHandler_Login_SuccessfulResponseContainsUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	testUser := createTestUser()
	testUser.PasswordHash = ""
	expectedResponse := createTestLoginResponse(testUser)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["user"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetProfile Handler Tests
// ============================================================================

func TestHandler_GetProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	expectedUser := createTestUser()
	expectedUser.ID = userID
	expectedUser.PasswordHash = ""

	mockService.On("GetProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	// Don't set user context to simulate unauthorized

	handler.GetProfile(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetProfile_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetProfile", mock.Anything, userID).Return(nil, common.NewNotFoundError("user not found", nil))

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetProfile", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_PasswordNotInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	expectedUser := createTestUser()
	expectedUser.ID = userID
	expectedUser.PasswordHash = "" // Should be empty

	mockService.On("GetProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	// Password hash should not be in the JSON response (tagged with json:"-")
	_, hasPasswordHash := data["password_hash"]
	assert.False(t, hasPasswordHash)
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_ReturnsUserDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	expectedUser := &models.User{
		ID:          userID,
		Email:       "testuser@example.com",
		PhoneNumber: "+1234567890",
		FirstName:   "Test",
		LastName:    "User",
		Role:        models.RoleRider,
		IsActive:    true,
		IsVerified:  true,
	}

	mockService.On("GetProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "testuser@example.com", data["email"])
	assert.Equal(t, "+1234567890", data["phone_number"])
	assert.Equal(t, "Test", data["first_name"])
	assert.Equal(t, "User", data["last_name"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_RiderRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	expectedUser := createTestUserWithRole(models.RoleRider)
	expectedUser.ID = userID
	expectedUser.PasswordHash = ""

	mockService.On("GetProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_DriverRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	expectedUser := createTestUserWithRole(models.RoleDriver)
	expectedUser.ID = userID
	expectedUser.PasswordHash = ""

	mockService.On("GetProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_AdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	expectedUser := createTestUserWithRole(models.RoleAdmin)
	expectedUser.ID = userID
	expectedUser.PasswordHash = ""

	mockService.On("GetProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleAdmin)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// UpdateProfile Handler Tests
// ============================================================================

func TestHandler_UpdateProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName:   "UpdatedFirst",
		LastName:    "UpdatedLast",
		PhoneNumber: "+9876543210",
	}

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.FirstName = updates.FirstName
	updatedUser.LastName = updates.LastName
	updatedUser.PhoneNumber = updates.PhoneNumber
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	updates := &models.User{
		FirstName: "UpdatedFirst",
	}

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	// Don't set user context

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateProfile_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateProfile_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName: "UpdatedFirst",
	}

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(nil, common.NewNotFoundError("user not found", nil))

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName: "UpdatedFirst",
	}

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_OnlyFirstName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName: "NewFirstName",
	}

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.FirstName = "NewFirstName"
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_OnlyLastName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		LastName: "NewLastName",
	}

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.LastName = "NewLastName"
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_OnlyPhoneNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		PhoneNumber: "+9876543210",
	}

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.PhoneNumber = "+9876543210"
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_WithProfileImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	profileImage := "https://example.com/image.jpg"
	updates := &models.User{
		ProfileImage: &profileImage,
	}

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.ProfileImage = &profileImage
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", map[string]interface{}{})
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_PasswordNotUpdatable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	// Attempt to update password hash (should be ignored by service)
	reqBody := map[string]interface{}{
		"first_name":    "Updated",
		"password_hash": "attempt-to-change-hash",
	}

	updatedUser := createTestUser()
	updatedUser.ID = userID
	updatedUser.FirstName = "Updated"
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_DriverRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName: "DriverFirst",
		LastName:  "DriverLast",
	}

	updatedUser := createTestUserWithRole(models.RoleDriver)
	updatedUser.ID = userID
	updatedUser.FirstName = "DriverFirst"
	updatedUser.LastName = "DriverLast"
	updatedUser.PasswordHash = ""

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(updatedUser, nil)

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleDriver)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName: "Updated",
	}

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(nil, common.NewInternalServerError("failed to update profile"))

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_Register_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
	}{
		{
			name: "valid registration",
			body: map[string]interface{}{
				"email":        "valid@example.com",
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"last_name":    "Doe",
				"role":         "rider",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing email",
			body: map[string]interface{}{
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"last_name":    "Doe",
				"role":         "rider",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			body: map[string]interface{}{
				"email":        "invalid-email",
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"last_name":    "Doe",
				"role":         "rider",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			body: map[string]interface{}{
				"email":        "valid@example.com",
				"password":     "short",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"last_name":    "Doe",
				"role":         "rider",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing phone number",
			body: map[string]interface{}{
				"email":      "valid@example.com",
				"password":   "ValidPass123!",
				"first_name": "John",
				"last_name":  "Doe",
				"role":       "rider",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing first name",
			body: map[string]interface{}{
				"email":        "valid@example.com",
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"last_name":    "Doe",
				"role":         "rider",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing last name",
			body: map[string]interface{}{
				"email":        "valid@example.com",
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"role":         "rider",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid role",
			body: map[string]interface{}{
				"email":        "valid@example.com",
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"last_name":    "Doe",
				"role":         "invalid_role",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "admin role not allowed",
			body: map[string]interface{}{
				"email":        "valid@example.com",
				"password":     "ValidPass123!",
				"phone_number": "+1234567890",
				"first_name":   "John",
				"last_name":    "Doe",
				"role":         "admin",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			handler := NewTestableHandler(mockService)

			// Only set up mock for valid case
			if tt.expectedStatus == http.StatusCreated {
				expectedUser := createTestUser()
				expectedUser.PasswordHash = ""
				mockService.On("Register", mock.Anything, mock.AnythingOfType("*models.RegisterRequest")).Return(expectedUser, nil)
			}

			c, w := setupTestContext("POST", "/api/v1/auth/register", tt.body)

			handler.Register(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_Login_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
	}{
		{
			name: "valid login",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			setupMock: func(m *MockAuthService) {
				user := createTestUser()
				user.PasswordHash = ""
				m.On("Login", mock.Anything, mock.AnythingOfType("*models.LoginRequest")).Return(createTestLoginResponse(user), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing email",
			body: map[string]interface{}{
				"password": "password123",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			body: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			body: map[string]interface{}{
				"email":    "not-an-email",
				"password": "password123",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty email",
			body: map[string]interface{}{
				"email":    "",
				"password": "password123",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			handler := NewTestableHandler(mockService)

			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			c, w := setupTestContext("POST", "/api/v1/auth/login", tt.body)

			handler.Login(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetProfile_AuthorizationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupMock      func(*MockAuthService, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "authorized user",
			setupContext: func(c *gin.Context) {
				setUserContext(c, uuid.New(), models.RoleRider)
			},
			setupMock: func(m *MockAuthService, userID uuid.UUID) {
				user := createTestUser()
				user.ID = userID
				user.PasswordHash = ""
				m.On("GetProfile", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no user context",
			setupContext:   func(c *gin.Context) {},
			setupMock:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			handler := NewTestableHandler(mockService)

			userID := uuid.New()
			if tt.setupMock != nil {
				tt.setupMock(mockService, userID)
			}

			c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
			tt.setupContext(c)

			handler.GetProfile(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_UpdateProfile_Cases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupContext   func(*gin.Context) uuid.UUID
		setupMock      func(*MockAuthService, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "update first name only",
			body: map[string]interface{}{
				"first_name": "NewFirst",
			},
			setupContext: func(c *gin.Context) uuid.UUID {
				userID := uuid.New()
				setUserContext(c, userID, models.RoleRider)
				return userID
			},
			setupMock: func(m *MockAuthService, userID uuid.UUID) {
				user := createTestUser()
				user.ID = userID
				user.FirstName = "NewFirst"
				user.PasswordHash = ""
				m.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update last name only",
			body: map[string]interface{}{
				"last_name": "NewLast",
			},
			setupContext: func(c *gin.Context) uuid.UUID {
				userID := uuid.New()
				setUserContext(c, userID, models.RoleRider)
				return userID
			},
			setupMock: func(m *MockAuthService, userID uuid.UUID) {
				user := createTestUser()
				user.ID = userID
				user.LastName = "NewLast"
				user.PasswordHash = ""
				m.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update phone number only",
			body: map[string]interface{}{
				"phone_number": "+9999999999",
			},
			setupContext: func(c *gin.Context) uuid.UUID {
				userID := uuid.New()
				setUserContext(c, userID, models.RoleRider)
				return userID
			},
			setupMock: func(m *MockAuthService, userID uuid.UUID) {
				user := createTestUser()
				user.ID = userID
				user.PhoneNumber = "+9999999999"
				user.PasswordHash = ""
				m.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update all fields",
			body: map[string]interface{}{
				"first_name":   "NewFirst",
				"last_name":    "NewLast",
				"phone_number": "+9999999999",
			},
			setupContext: func(c *gin.Context) uuid.UUID {
				userID := uuid.New()
				setUserContext(c, userID, models.RoleRider)
				return userID
			},
			setupMock: func(m *MockAuthService, userID uuid.UUID) {
				user := createTestUser()
				user.ID = userID
				user.FirstName = "NewFirst"
				user.LastName = "NewLast"
				user.PhoneNumber = "+9999999999"
				user.PasswordHash = ""
				m.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorized update",
			body: map[string]interface{}{
				"first_name": "NewFirst",
			},
			setupContext: func(c *gin.Context) uuid.UUID {
				// Don't set user context
				return uuid.Nil
			},
			setupMock:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			handler := NewTestableHandler(mockService)

			c, w := setupTestContext("PUT", "/api/v1/auth/profile", tt.body)
			userID := tt.setupContext(c)

			if tt.setupMock != nil {
				tt.setupMock(mockService, userID)
			}

			handler.UpdateProfile(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases and Error Handling Tests
// ============================================================================

func TestHandler_Register_ConcurrencyConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.RegisterRequest{
		Email:       "concurrent@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	// Simulate race condition where user was created between check and insert
	mockService.On("Register", mock.Anything, &reqBody).Return(nil, common.NewConflictError("user already exists"))

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Login_AccountLocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "locked@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, common.NewForbiddenError("account is locked"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateProfile_ConcurrentModification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	updates := &models.User{
		FirstName: "Updated",
	}

	mockService.On("UpdateProfile", mock.Anything, userID, mock.AnythingOfType("*models.User")).Return(nil, common.NewConflictError("concurrent modification detected"))

	c, w := setupTestContext("PUT", "/api/v1/auth/profile", updates)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateProfile(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetProfile_DatabaseTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetProfile", mock.Anything, userID).Return(nil, common.NewServiceUnavailableError("database timeout"))

	c, w := setupTestContext("GET", "/api/v1/auth/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProfile(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Register_DatabaseConnectionError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.RegisterRequest{
		Email:       "newuser@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(nil, common.NewServiceUnavailableError("database connection failed"))

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_Register_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	expectedUser := createTestUser()
	expectedUser.PasswordHash = ""

	reqBody := models.RegisterRequest{
		Email:       "newuser@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        models.RoleRider,
	}

	mockService.On("Register", mock.Anything, &reqBody).Return(expectedUser, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/register", reqBody)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["id"])
	assert.NotNil(t, data["email"])
	assert.NotNil(t, data["first_name"])
	assert.NotNil(t, data["last_name"])
	assert.NotNil(t, data["role"])

	mockService.AssertExpectations(t)
}

func TestHandler_Login_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	testUser := createTestUser()
	testUser.PasswordHash = ""
	expectedResponse := createTestLoginResponse(testUser)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["user"])
	assert.NotNil(t, data["token"])

	user := data["user"].(map[string]interface{})
	assert.NotNil(t, user["id"])
	assert.NotNil(t, user["email"])

	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuthService)
	handler := NewTestableHandler(mockService)

	reqBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	mockService.On("Login", mock.Anything, &reqBody).Return(nil, common.NewUnauthorizedError("invalid credentials"))

	c, w := setupTestContext("POST", "/api/v1/auth/login", reqBody)

	handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])

	mockService.AssertExpectations(t)
}
