package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServiceName    = "mobile-api"
	testServiceVersion = "1.0.0"
	testJWTSecret      = "test-secret-key-for-testing-only"
)

// setupTestRouter creates a minimal test router with the same middleware chain as main.go
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware chain similar to main.go
	router.Use(middleware.CorrelationID())
	router.Use(middleware.SecurityHeaders())

	return router
}

// setupTestRouterWithAuth creates a test router with authentication middleware
func setupTestRouterWithAuth() (*gin.Engine, jwtkeys.KeyProvider) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware chain similar to main.go
	router.Use(middleware.CorrelationID())
	router.Use(middleware.SecurityHeaders())

	// Create JWT provider
	jwtProvider := jwtkeys.NewStaticProvider(testJWTSecret)

	return router, jwtProvider
}

// setupFullTestRouter creates a complete test router with all routes
func setupFullTestRouter() (*gin.Engine, jwtkeys.KeyProvider) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware chain similar to main.go
	router.Use(middleware.CorrelationID())
	router.Use(middleware.SecurityHeaders())

	// Create JWT provider
	jwtProvider := jwtkeys.NewStaticProvider(testJWTSecret)

	// Health check endpoints
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": testServiceName, "version": testServiceVersion})
	})
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "alive", "service": testServiceName, "version": testServiceVersion})
	})

	// Mock database health check
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		return nil // Mock healthy database
	}

	router.GET("/health/ready", func(c *gin.Context) {
		for name, check := range healthChecks {
			if err := check(); err != nil {
				c.JSON(503, gin.H{"status": "not ready", "service": testServiceName, "failed_check": name, "error": err.Error()})
				return
			}
		}
		c.JSON(200, gin.H{"status": "ready", "service": testServiceName, "version": testServiceVersion})
	})

	router.GET("/metrics", func(c *gin.Context) {
		c.String(200, "# HELP go_goroutines Number of goroutines\n")
	})

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Ride routes
		rides := api.Group("/rides")
		{
			rides.GET("/history", func(c *gin.Context) {
				c.JSON(200, gin.H{"rides": []interface{}{}})
			})
			rides.GET("/:id/receipt", func(c *gin.Context) {
				c.JSON(200, gin.H{"receipt": gin.H{}})
			})
		}

		// Favorites routes
		favs := api.Group("/favorites")
		{
			favs.POST("", func(c *gin.Context) {
				c.JSON(201, gin.H{"id": uuid.New().String()})
			})
			favs.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"favorites": []interface{}{}})
			})
			favs.GET("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"favorite": gin.H{}})
			})
			favs.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"favorite": gin.H{}})
			})
			favs.DELETE("/:id", func(c *gin.Context) {
				c.JSON(204, nil)
			})
		}

		// Profile routes
		api.GET("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"profile": gin.H{}})
		})
		api.PUT("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"profile": gin.H{}})
		})

		// Rate ride
		api.POST("/rides/:id/rate", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})
	}

	return router, jwtProvider
}

// generateTestToken creates a valid JWT token for testing
func generateTestToken(userID uuid.UUID, email string, role models.UserRole) string {
	claims := middleware.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testJWTSecret))
	return tokenString
}

// generateExpiredTestToken creates an expired JWT token for testing
func generateExpiredTestToken(userID uuid.UUID, email string, role models.UserRole) string {
	claims := middleware.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testJWTSecret))
	return tokenString
}

// ====================
// Route Registration Tests
// ====================

func TestHealthEndpoints(t *testing.T) {
	router, _ := setupFullTestRouter()

	tests := []struct {
		name     string
		endpoint string
		wantCode int
		wantBody map[string]interface{}
	}{
		{
			name:     "healthz returns 200",
			endpoint: "/healthz",
			wantCode: http.StatusOK,
			wantBody: map[string]interface{}{"status": "healthy", "service": testServiceName},
		},
		{
			name:     "health/live returns 200",
			endpoint: "/health/live",
			wantCode: http.StatusOK,
			wantBody: map[string]interface{}{"status": "alive", "service": testServiceName},
		},
		{
			name:     "health/ready returns 200 when database is healthy",
			endpoint: "/health/ready",
			wantCode: http.StatusOK,
			wantBody: map[string]interface{}{"status": "ready", "service": testServiceName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			for key, expectedValue := range tt.wantBody {
				assert.Equal(t, expectedValue, response[key])
			}
		})
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "# HELP")
}

func TestRidesRouteGroup(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	tests := []struct {
		name     string
		method   string
		endpoint string
		wantCode int
	}{
		{
			name:     "GET /api/v1/rides/history requires auth",
			method:   http.MethodGet,
			endpoint: "/api/v1/rides/history",
			wantCode: http.StatusOK,
		},
		{
			name:     "GET /api/v1/rides/:id/receipt requires auth",
			method:   http.MethodGet,
			endpoint: "/api/v1/rides/123/receipt",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestFavoritesRouteGroup(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	tests := []struct {
		name     string
		method   string
		endpoint string
		body     string
		wantCode int
	}{
		{
			name:     "POST /api/v1/favorites creates favorite",
			method:   http.MethodPost,
			endpoint: "/api/v1/favorites",
			body:     `{"name": "Home", "address": "123 Test St"}`,
			wantCode: http.StatusCreated,
		},
		{
			name:     "GET /api/v1/favorites lists favorites",
			method:   http.MethodGet,
			endpoint: "/api/v1/favorites",
			wantCode: http.StatusOK,
		},
		{
			name:     "GET /api/v1/favorites/:id gets single favorite",
			method:   http.MethodGet,
			endpoint: "/api/v1/favorites/" + uuid.New().String(),
			wantCode: http.StatusOK,
		},
		{
			name:     "PUT /api/v1/favorites/:id updates favorite",
			method:   http.MethodPut,
			endpoint: "/api/v1/favorites/" + uuid.New().String(),
			body:     `{"name": "Office"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "DELETE /api/v1/favorites/:id deletes favorite",
			method:   http.MethodDelete,
			endpoint: "/api/v1/favorites/" + uuid.New().String(),
			wantCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.endpoint, nil)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestProfileRoutes(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	tests := []struct {
		name     string
		method   string
		endpoint string
		body     string
		wantCode int
	}{
		{
			name:     "GET /api/v1/profile returns user profile",
			method:   http.MethodGet,
			endpoint: "/api/v1/profile",
			wantCode: http.StatusOK,
		},
		{
			name:     "PUT /api/v1/profile updates user profile",
			method:   http.MethodPut,
			endpoint: "/api/v1/profile",
			body:     `{"first_name": "John", "last_name": "Doe"}`,
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.endpoint, nil)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestRateRideEndpoint(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+uuid.New().String()+"/rate", strings.NewReader(`{"rating": 5}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ====================
// Authentication Tests
// ====================

func TestAuthMiddlewareRequiresToken(t *testing.T) {
	router, _ := setupFullTestRouter()

	tests := []struct {
		name     string
		endpoint string
		method   string
	}{
		{"GET /api/v1/rides/history", "/api/v1/rides/history", http.MethodGet},
		{"GET /api/v1/favorites", "/api/v1/favorites", http.MethodGet},
		{"GET /api/v1/profile", "/api/v1/profile", http.MethodGet},
		{"PUT /api/v1/profile", "/api/v1/profile", http.MethodPut},
	}

	for _, tt := range tests {
		t.Run(tt.name+" returns 401 without token", func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	router, _ := setupFullTestRouter()

	tests := []struct {
		name   string
		header string
	}{
		{"invalid token format", "Bearer invalid-token"},
		{"missing Bearer prefix", "some-token"},
		{"empty Bearer token", "Bearer "},
		{"wrong prefix", "Basic some-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddlewareRejectsExpiredToken(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateExpiredTestToken(uuid.New(), "test@example.com", models.RoleRider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareSupportsQueryParamToken(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile?token="+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ====================
// Middleware Chain Tests
// ====================

func TestCorrelationIDMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	t.Run("adds X-Request-ID header when not provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		correlationID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, correlationID)
		// Verify it's a valid UUID
		_, err := uuid.Parse(correlationID)
		assert.NoError(t, err)
	})

	t.Run("uses provided X-Request-ID header", func(t *testing.T) {
		providedID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", providedID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, providedID, w.Header().Get("X-Request-ID"))
	})

	t.Run("generates new ID for invalid provided ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "invalid-uuid")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		correlationID := w.Header().Get("X-Request-ID")
		assert.NotEqual(t, "invalid-uuid", correlationID)
		_, err := uuid.Parse(correlationID)
		assert.NoError(t, err)
	})
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		t.Run(tt.header+" is set", func(t *testing.T) {
			assert.Equal(t, tt.expected, w.Header().Get(tt.header))
		})
	}

	t.Run("Content-Security-Policy is set", func(t *testing.T) {
		csp := w.Header().Get("Content-Security-Policy")
		assert.NotEmpty(t, csp)
		assert.Contains(t, csp, "default-src")
	})

	t.Run("Permissions-Policy is set", func(t *testing.T) {
		permissionsPolicy := w.Header().Get("Permissions-Policy")
		assert.NotEmpty(t, permissionsPolicy)
	})
}

func TestMaxBodySizeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.MaxBodySize(100)) // 100 bytes limit
	router.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	t.Run("accepts request within size limit", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"small": "data"}`))
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects request exceeding size limit", func(t *testing.T) {
		// Create a body larger than 100 bytes
		largeBody := bytes.NewReader([]byte(`{"data": "` + strings.Repeat("x", 200) + `"}`))
		req := httptest.NewRequest(http.MethodPost, "/test", largeBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

// ====================
// Health Check Tests
// ====================

func TestHealthzEndpointContent(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, testServiceName, response["service"])
	assert.Equal(t, testServiceVersion, response["version"])
}

func TestLivenessProbeContent(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "alive", response["status"])
	assert.Equal(t, testServiceName, response["service"])
	assert.Equal(t, testServiceVersion, response["version"])
}

func TestReadinessProbeWithHealthyDatabase(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response["status"])
}

func TestReadinessProbeWithUnhealthyDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock unhealthy database
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		return errors.New("connection refused")
	}

	router.GET("/health/ready", func(c *gin.Context) {
		for name, check := range healthChecks {
			if err := check(); err != nil {
				c.JSON(503, gin.H{"status": "not ready", "service": testServiceName, "failed_check": name, "error": err.Error()})
				return
			}
		}
		c.JSON(200, gin.H{"status": "ready", "service": testServiceName})
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "not ready", response["status"])
	assert.Equal(t, "database", response["failed_check"])
	assert.Contains(t, response["error"], "connection refused")
}

// ====================
// Helper Function Tests
// ====================

func TestGetEnvReturnsDefaultWhenNotSet(t *testing.T) {
	// Ensure the env var is not set
	os.Unsetenv("TEST_UNSET_VAR")

	result := getEnv("TEST_UNSET_VAR", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvReturnsValueWhenSet(t *testing.T) {
	t.Setenv("TEST_SET_VAR", "actual_value")

	result := getEnv("TEST_SET_VAR", "default_value")
	assert.Equal(t, "actual_value", result)
}

func TestGetEnvAsIntReturnsDefaultForInvalidInt(t *testing.T) {
	t.Setenv("TEST_INVALID_INT", "not_a_number")

	result := getEnvAsInt("TEST_INVALID_INT", 42)
	assert.Equal(t, 42, result)
}

func TestGetEnvAsIntReturnsDefaultWhenNotSet(t *testing.T) {
	os.Unsetenv("TEST_UNSET_INT")

	result := getEnvAsInt("TEST_UNSET_INT", 99)
	assert.Equal(t, 99, result)
}

func TestGetEnvAsIntReturnsParsedValue(t *testing.T) {
	t.Setenv("TEST_VALID_INT", "123")

	result := getEnvAsInt("TEST_VALID_INT", 0)
	assert.Equal(t, 123, result)
}

func TestGetEnvAsIntHandlesNegativeNumbers(t *testing.T) {
	t.Setenv("TEST_NEGATIVE_INT", "-50")

	result := getEnvAsInt("TEST_NEGATIVE_INT", 0)
	assert.Equal(t, -50, result)
}

func TestGetEnvAsIntHandlesZero(t *testing.T) {
	t.Setenv("TEST_ZERO_INT", "0")

	result := getEnvAsInt("TEST_ZERO_INT", 100)
	assert.Equal(t, 0, result)
}

func TestGetEnvAsIntHandlesEmptyString(t *testing.T) {
	t.Setenv("TEST_EMPTY_INT", "")

	result := getEnvAsInt("TEST_EMPTY_INT", 42)
	assert.Equal(t, 42, result)
}

// ====================
// Route Access Control Tests
// ====================

func TestPublicEndpointsDoNotRequireAuth(t *testing.T) {
	router, _ := setupFullTestRouter()

	publicEndpoints := []string{
		"/healthz",
		"/health/live",
		"/health/ready",
		"/metrics",
	}

	for _, endpoint := range publicEndpoints {
		t.Run(fmt.Sprintf("GET %s is public", endpoint), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not return 401 Unauthorized
			assert.NotEqual(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestProtectedEndpointsRequireAuth(t *testing.T) {
	router, _ := setupFullTestRouter()

	protectedEndpoints := []struct {
		method   string
		endpoint string
	}{
		{http.MethodGet, "/api/v1/rides/history"},
		{http.MethodGet, "/api/v1/favorites"},
		{http.MethodPost, "/api/v1/favorites"},
		{http.MethodGet, "/api/v1/profile"},
		{http.MethodPut, "/api/v1/profile"},
	}

	for _, ep := range protectedEndpoints {
		t.Run(fmt.Sprintf("%s %s requires auth", ep.method, ep.endpoint), func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// ====================
// JWT Provider Tests
// ====================

func TestStaticProviderResolveKey(t *testing.T) {
	provider := jwtkeys.NewStaticProvider(testJWTSecret)

	key, err := provider.ResolveKey("any-kid")
	require.NoError(t, err)
	assert.Equal(t, []byte(testJWTSecret), key)
}

func TestStaticProviderLegacyKey(t *testing.T) {
	provider := jwtkeys.NewStaticProvider(testJWTSecret)

	key := provider.LegacyKey()
	assert.Equal(t, []byte(testJWTSecret), key)
}

func TestStaticProviderWithEmptySecret(t *testing.T) {
	provider := jwtkeys.NewStaticProvider("")

	_, err := provider.ResolveKey("any-kid")
	assert.Error(t, err)
	assert.Equal(t, jwtkeys.ErrKeyNotFound, err)
}

// ====================
// Token Claims Tests
// ====================

func TestTokenClaimsExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jwtProvider := jwtkeys.NewStaticProvider(testJWTSecret)
	router.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))

	expectedUserID := uuid.New()
	expectedEmail := "test@example.com"
	expectedRole := models.RoleDriver

	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists)
		assert.Equal(t, expectedUserID, userID)

		email, exists := c.Get("user_email")
		assert.True(t, exists)
		assert.Equal(t, expectedEmail, email)

		role, exists := c.Get("user_role")
		assert.True(t, exists)
		assert.Equal(t, expectedRole, role)

		c.JSON(200, gin.H{"ok": true})
	})

	token := generateTestToken(expectedUserID, expectedEmail, expectedRole)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ====================
// Response Format Tests
// ====================

func TestHealthCheckResponseFormat(t *testing.T) {
	router, _ := setupFullTestRouter()

	endpoints := []string{"/healthz", "/health/live", "/health/ready"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// All health responses should have these fields
			assert.Contains(t, response, "status")
			assert.Contains(t, response, "service")
		})
	}
}

func TestErrorResponseFormat(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Error responses should have an error field
	assert.Contains(t, response, "error")
}

// ====================
// Edge Case Tests
// ====================

func TestEmptyAuthorizationHeader(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWhitespaceOnlyAuthorizationHeader(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "   ")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBearerWithNoToken(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMultipleSpacesInBearerHeader(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer  "+token) // Double space
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail because of invalid format
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNonExistentRoute(t *testing.T) {
	router, _ := setupFullTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMethodNotAllowed(t *testing.T) {
	router, _ := setupFullTestRouter()

	// DELETE is not supported on /healthz
	req := httptest.NewRequest(http.MethodDelete, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Gin returns 404 for unsupported methods on defined routes
	assert.Contains(t, []int{http.StatusMethodNotAllowed, http.StatusNotFound}, w.Code)
}

// ====================
// Timeout Configuration Tests
// ====================

func TestTimeoutConfigDefaults(t *testing.T) {
	cfg := config.TimeoutConfig{}

	// Test default timeout duration
	assert.Equal(t, 0*time.Second, cfg.DefaultRequestTimeoutDuration())
	assert.Equal(t, 0*time.Second, cfg.HTTPClientTimeoutDuration())
	assert.Equal(t, 0*time.Second, cfg.DatabaseQueryTimeoutDuration())
}

func TestTimeoutConfigWithValues(t *testing.T) {
	cfg := config.TimeoutConfig{
		HTTPClientTimeout:      30,
		DatabaseQueryTimeout:   10,
		DefaultRequestTimeout:  60,
	}

	assert.Equal(t, 30*time.Second, cfg.HTTPClientTimeoutDuration())
	assert.Equal(t, 10*time.Second, cfg.DatabaseQueryTimeoutDuration())
	assert.Equal(t, 60*time.Second, cfg.DefaultRequestTimeoutDuration())
}

func TestTimeoutForRouteWithOverrides(t *testing.T) {
	cfg := config.TimeoutConfig{
		DefaultRequestTimeout: 30,
		RouteOverrides: map[string]int{
			"POST:/api/v1/rides": 60,
		},
	}

	assert.Equal(t, 60*time.Second, cfg.TimeoutForRoute("POST", "/api/v1/rides"))
	assert.Equal(t, 30*time.Second, cfg.TimeoutForRoute("GET", "/api/v1/profile"))
}

func TestTimeoutForRouteWithNilOverrides(t *testing.T) {
	cfg := config.TimeoutConfig{
		DefaultRequestTimeout: 30,
		RouteOverrides:        nil,
	}

	assert.Equal(t, 30*time.Second, cfg.TimeoutForRoute("POST", "/api/v1/rides"))
}

// ====================
// Concurrent Request Tests
// ====================

func TestConcurrentRequests(t *testing.T) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConcurrentHealthChecks(t *testing.T) {
	router, _ := setupFullTestRouter()

	done := make(chan bool, 30)

	endpoints := []string{"/healthz", "/health/live", "/health/ready"}

	for i := 0; i < 10; i++ {
		for _, endpoint := range endpoints {
			go func(ep string) {
				req := httptest.NewRequest(http.MethodGet, ep, nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				done <- true
			}(endpoint)
		}
	}

	// Wait for all requests to complete
	for i := 0; i < 30; i++ {
		<-done
	}
}

// ====================
// Different User Role Tests
// ====================

func TestDifferentUserRolesCanAccessProtectedRoutes(t *testing.T) {
	router, _ := setupFullTestRouter()

	roles := []models.UserRole{
		models.RoleRider,
		models.RoleDriver,
		models.RoleAdmin,
	}

	for _, role := range roles {
		t.Run(string(role)+" can access profile", func(t *testing.T) {
			token := generateTestToken(uuid.New(), "test@example.com", role)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// ====================
// Benchmark Tests
// ====================

func BenchmarkHealthzEndpoint(b *testing.B) {
	router, _ := setupFullTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthenticatedRequest(b *testing.B) {
	router, _ := setupFullTestRouter()
	token := generateTestToken(uuid.New(), "test@example.com", models.RoleRider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkCorrelationIDMiddleware(b *testing.B) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkSecurityHeadersMiddleware(b *testing.B) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
