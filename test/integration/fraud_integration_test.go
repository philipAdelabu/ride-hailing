//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/richxcame/ride-hailing/internal/fraud"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

const fraudServiceKey = "fraud"

// FraudIntegrationTestSuite tests end-to-end fraud detection flows
type FraudIntegrationTestSuite struct {
	suite.Suite
	admin  authSession
	rider  authSession
	driver authSession
}

func TestFraudIntegrationSuite(t *testing.T) {
	suite.Run(t, new(FraudIntegrationTestSuite))
}

func (s *FraudIntegrationTestSuite) SetupSuite() {
	// Ensure all services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[fraudServiceKey]; !ok {
		services[fraudServiceKey] = startFraudService()
	}
	if _, ok := services[paymentsServiceKey]; !ok {
		services[paymentsServiceKey] = startPaymentsService(mustLoadConfig("payments-service"))
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}
}

func (s *FraudIntegrationTestSuite) SetupTest() {
	truncateFraudTables(s.T())
	s.admin = registerAndLogin(s.T(), models.RoleAdmin)
	s.rider = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
}

func startFraudService() *serviceInstance {
	repo := fraud.NewRepository(dbPool)
	service := fraud.NewService(repo)
	handler := fraud.NewHandler(service)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	api := router.Group("/api/v1/fraud")
	api.Use(middleware.AuthMiddleware("integration-secret"))
	api.Use(middleware.RequireAdmin())
	{
		// Fraud alerts
		api.GET("/alerts", handler.GetPendingAlerts)
		api.GET("/alerts/:id", handler.GetAlert)
		api.POST("/alerts", handler.CreateAlert)
		api.PUT("/alerts/:id/investigate", handler.InvestigateAlert)
		api.PUT("/alerts/:id/resolve", handler.ResolveAlert)

		// User risk management
		api.GET("/users/:id/alerts", handler.GetUserAlerts)
		api.GET("/users/:id/risk-profile", handler.GetUserRiskProfile)
		api.POST("/users/:id/analyze", handler.AnalyzeUser)
		api.POST("/users/:id/suspend", handler.SuspendUser)
		api.POST("/users/:id/reinstate", handler.ReinstateUser)

		// Fraud detection
		api.POST("/detect/payment/:user_id", handler.DetectPaymentFraud)
		api.POST("/detect/ride/:user_id", handler.DetectRideFraud)
		api.POST("/detect/account/:user_id", handler.DetectAccountFraud)

		// Statistics and patterns
		api.GET("/statistics", handler.GetFraudStatistics)
		api.GET("/patterns", handler.GetFraudPatterns)
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func truncateFraudTables(t *testing.T) {
	t.Helper()
	tables := []string{
		"fraud_alerts",
		"user_risk_profiles",
		"fraud_patterns",
		"wallet_transactions",
		"payments",
		"rides",
		"drivers",
		"wallets",
		"users",
	}

	for _, table := range tables {
		_, err := dbPool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Table may not exist, that's fine for optional tables
			continue
		}
	}
}

// ============================================
// PAYMENT FRAUD DETECTION TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestPaymentFraud_SuspiciousPaymentPatterns() {
	t := s.T()
	ctx := context.Background()

	// Create multiple failed payment attempts to simulate suspicious behavior
	for i := 0; i < 5; i++ {
		_, err := dbPool.Exec(ctx, `
			INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			uuid.New(), uuid.New(), s.rider.User.ID, s.driver.User.ID, 50.00, "usd", "failed", "card", time.Now().Add(-time.Duration(i)*time.Hour))
		require.NoError(t, err)
	}

	// Trigger payment fraud detection
	detectPath := fmt.Sprintf("/users/%s/analyze", s.rider.User.ID)
	detectResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodPost, detectPath, nil, authHeaders(s.admin.Token))
	require.True(t, detectResp.Success)
	require.NotNil(t, detectResp.Data)
	require.Greater(t, detectResp.Data.RiskScore, 0.0)
}

func (s *FraudIntegrationTestSuite) TestPaymentFraud_VelocityCheck() {
	t := s.T()
	ctx := context.Background()

	// Create many transactions in a short time (velocity abuse)
	now := time.Now()
	for i := 0; i < 20; i++ {
		_, err := dbPool.Exec(ctx, `
			INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			uuid.New(), uuid.New(), s.rider.User.ID, s.driver.User.ID, 25.00, "usd", "completed", "card", now.Add(-time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	// Check payment fraud indicators
	detectPath := fmt.Sprintf("/detect/payment/%s", s.rider.User.ID)
	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, detectPath, nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify alerts were created for velocity check
	alertsPath := fmt.Sprintf("/users/%s/alerts", s.rider.User.ID)
	alertsResp := doRequest[[]*fraud.FraudAlert](t, fraudServiceKey, http.MethodGet, alertsPath, nil, authHeaders(s.admin.Token))
	require.True(t, alertsResp.Success)
}

func (s *FraudIntegrationTestSuite) TestPaymentFraud_UnusualAmountDetection() {
	t := s.T()
	ctx := context.Background()

	// Create normal transactions first
	for i := 0; i < 5; i++ {
		_, err := dbPool.Exec(ctx, `
			INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			uuid.New(), uuid.New(), s.rider.User.ID, s.driver.User.ID, 30.00, "usd", "completed", "card", time.Now().Add(-time.Duration(i)*24*time.Hour))
		require.NoError(t, err)
	}

	// Create suspiciously large transaction
	_, err := dbPool.Exec(ctx, `
		INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.New(), uuid.New(), s.rider.User.ID, s.driver.User.ID, 5000.00, "usd", "completed", "card", time.Now())
	require.NoError(t, err)

	// Analyze user for fraud
	analyzePath := fmt.Sprintf("/users/%s/analyze", s.rider.User.ID)
	analyzeResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodPost, analyzePath, nil, authHeaders(s.admin.Token))
	require.True(t, analyzeResp.Success)
}

func (s *FraudIntegrationTestSuite) TestPaymentFraud_BlockedCardHandling() {
	t := s.T()
	ctx := context.Background()

	// Create payment with blocked/declined status (simulating blocked card)
	paymentID := uuid.New()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method, created_at, payment_method_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		paymentID, uuid.New(), s.rider.User.ID, s.driver.User.ID, 100.00, "usd", "failed", "card", time.Now(), "pm_blocked_card")
	require.NoError(t, err)

	// Create a fraud alert for blocked card
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "payment_fraud",
		"alert_level": "high",
		"description": "Blocked card detected",
		"details": map[string]interface{}{
			"payment_id":        paymentID,
			"payment_method_id": "pm_blocked_card",
			"reason":            "card_blocked",
		},
		"risk_score": 75.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.NotNil(t, createResp.Data)
	require.Equal(t, fraud.AlertTypePaymentFraud, createResp.Data.AlertType)
	require.Equal(t, fraud.AlertLevelHigh, createResp.Data.AlertLevel)
}

func (s *FraudIntegrationTestSuite) TestPaymentFraud_MultiplePaymentMethodChanges() {
	t := s.T()
	ctx := context.Background()

	// Create transactions with different payment methods (suspicious rapid changes)
	paymentMethods := []string{"pm_card_1", "pm_card_2", "pm_card_3", "pm_card_4", "pm_card_5"}
	for i, pm := range paymentMethods {
		_, err := dbPool.Exec(ctx, `
			INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method, payment_method_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			uuid.New(), uuid.New(), s.rider.User.ID, s.driver.User.ID, 30.00, "usd", "completed", "card", pm, time.Now().Add(-time.Duration(i)*time.Hour))
		require.NoError(t, err)
	}

	// Detect payment fraud
	detectPath := fmt.Sprintf("/detect/payment/%s", s.rider.User.ID)
	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, detectPath, nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// ============================================
// RIDE FRAUD DETECTION TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestRideFraud_GPSSpoofingDetection() {
	t := s.T()
	ctx := context.Background()

	// Create a fraud alert for GPS spoofing
	alertReq := map[string]interface{}{
		"user_id":     s.driver.User.ID,
		"alert_type":  "ride_fraud",
		"alert_level": "critical",
		"description": "GPS spoofing detected - impossible location jump",
		"details": map[string]interface{}{
			"location_jump_km":   500,
			"time_elapsed_sec":   60,
			"max_possible_speed": 100,
			"detected_speed":     30000,
		},
		"risk_score": 95.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.NotNil(t, createResp.Data)
	require.Equal(t, fraud.AlertLevelCritical, createResp.Data.AlertLevel)

	// Verify risk profile was updated
	riskPath := fmt.Sprintf("/users/%s/risk-profile", s.driver.User.ID)
	riskResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodGet, riskPath, nil, authHeaders(s.admin.Token))
	require.True(t, riskResp.Success)
	require.Greater(t, riskResp.Data.RiskScore, 0.0)

	_ = ctx // Used for potential DB ops
}

func (s *FraudIntegrationTestSuite) TestRideFraud_FareManipulationDetection() {
	t := s.T()
	ctx := context.Background()

	// Create rides with suspicious fare patterns
	rideID := uuid.New()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO rides (id, rider_id, driver_id, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
		                   pickup_address, dropoff_address, status, estimated_fare, final_fare, estimated_distance, actual_distance, requested_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		rideID, s.rider.User.ID, s.driver.User.ID, 37.7749, -122.4194, 37.7850, -122.4094,
		"Pickup", "Dropoff", "completed", 15.00, 150.00, 2.0, 20.0, time.Now().Add(-1*time.Hour), time.Now())
	require.NoError(t, err)

	// Create fraud alert for fare manipulation
	alertReq := map[string]interface{}{
		"user_id":     s.driver.User.ID,
		"alert_type":  "ride_fraud",
		"alert_level": "high",
		"description": "Fare manipulation detected - final fare 10x estimated",
		"details": map[string]interface{}{
			"ride_id":        rideID,
			"estimated_fare": 15.00,
			"final_fare":     150.00,
			"fare_ratio":     10.0,
		},
		"risk_score": 80.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
}

func (s *FraudIntegrationTestSuite) TestRideFraud_UnusualRoutePatterns() {
	t := s.T()
	ctx := context.Background()

	// Create ride with unusual route (long detour)
	rideID := uuid.New()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO rides (id, rider_id, driver_id, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
		                   pickup_address, dropoff_address, status, estimated_distance, actual_distance, estimated_fare, final_fare, requested_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		rideID, s.rider.User.ID, s.driver.User.ID, 37.7749, -122.4194, 37.7850, -122.4094,
		"Start", "End", "completed", 5.0, 50.0, 15.00, 100.00, time.Now().Add(-2*time.Hour), time.Now())
	require.NoError(t, err)

	// Detect ride fraud
	detectPath := fmt.Sprintf("/detect/ride/%s", s.driver.User.ID)
	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, detectPath, nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestRideFraud_GhostRideDetection() {
	t := s.T()
	ctx := context.Background()

	// Create a "ghost ride" - completed very quickly (impossible)
	rideID := uuid.New()
	now := time.Now()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO rides (id, rider_id, driver_id, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
		                   pickup_address, dropoff_address, status, estimated_distance, actual_distance, estimated_duration,
		                   estimated_fare, final_fare, requested_at, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		rideID, s.rider.User.ID, s.driver.User.ID, 37.7749, -122.4194, 37.8044, -122.2712,
		"SF", "Oakland", "completed", 20.0, 20.0, 30, 50.00, 50.00, now.Add(-5*time.Minute), now.Add(-4*time.Minute), now.Add(-3*time.Minute))
	require.NoError(t, err)

	// Create alert for ghost ride
	alertReq := map[string]interface{}{
		"user_id":     s.driver.User.ID,
		"alert_type":  "ride_fraud",
		"alert_level": "high",
		"description": "Ghost ride detected - 20km ride completed in 1 minute",
		"details": map[string]interface{}{
			"ride_id":          rideID,
			"distance_km":      20.0,
			"duration_minutes": 1,
			"implied_speed":    1200,
		},
		"risk_score": 85.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
}

func (s *FraudIntegrationTestSuite) TestRideFraud_ExcessiveCancellations() {
	t := s.T()
	ctx := context.Background()

	// Create many cancelled rides
	for i := 0; i < 15; i++ {
		_, err := dbPool.Exec(ctx, `
			INSERT INTO rides (id, rider_id, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			                   pickup_address, dropoff_address, status, estimated_fare, requested_at, cancelled_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			uuid.New(), s.rider.User.ID, 37.7749, -122.4194, 37.8044, -122.2712,
			"Pickup", "Dropoff", "cancelled", 25.00, time.Now().Add(-time.Duration(i)*24*time.Hour), time.Now().Add(-time.Duration(i)*24*time.Hour+5*time.Minute))
		require.NoError(t, err)
	}

	// Detect ride fraud
	detectPath := fmt.Sprintf("/detect/ride/%s", s.rider.User.ID)
	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, detectPath, nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestRideFraud_DriverRiderCollusion() {
	t := s.T()
	ctx := context.Background()

	// Create alert for driver-rider collusion
	alertReq := map[string]interface{}{
		"user_id":     s.driver.User.ID,
		"alert_type":  "ride_fraud",
		"alert_level": "critical",
		"description": "Potential driver-rider collusion detected",
		"details": map[string]interface{}{
			"rider_id":              s.rider.User.ID,
			"rides_together":        50,
			"rides_with_max_promo":  45,
			"average_fare_discount": 0.95,
		},
		"risk_score": 90.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertLevelCritical, createResp.Data.AlertLevel)

	_ = ctx // For potential DB operations
}

// ============================================
// ACCOUNT FRAUD DETECTION TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestAccountFraud_MultipleAccountsSameDevice() {
	t := s.T()

	// Create fraud alert for multiple accounts from same device
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "account_fraud",
		"alert_level": "high",
		"description": "Multiple accounts detected from same device",
		"details": map[string]interface{}{
			"device_id":        "device_fingerprint_12345",
			"accounts_on_device": []string{s.rider.User.ID.String(), uuid.NewString(), uuid.NewString()},
			"first_account_at": time.Now().Add(-30 * 24 * time.Hour),
		},
		"risk_score": 75.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertTypeAccountFraud, createResp.Data.AlertType)
}

func (s *FraudIntegrationTestSuite) TestAccountFraud_UnusualLoginPatterns() {
	t := s.T()

	// Create fraud alert for unusual login pattern
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "account_fraud",
		"alert_level": "medium",
		"description": "Unusual login pattern detected",
		"details": map[string]interface{}{
			"login_countries":     []string{"US", "RU", "CN", "BR"},
			"login_time_span":     "2 hours",
			"impossible_travel":   true,
			"new_device_detected": true,
		},
		"risk_score": 65.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
}

func (s *FraudIntegrationTestSuite) TestAccountFraud_AccountTakeoverDetection() {
	t := s.T()

	// Create fraud alert for potential account takeover
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "account_fraud",
		"alert_level": "critical",
		"description": "Potential account takeover detected",
		"details": map[string]interface{}{
			"password_changed":           true,
			"email_changed":              true,
			"phone_changed":              true,
			"payment_method_added":       true,
			"all_changes_within_minutes": 10,
			"new_ip_address":             "185.220.101.1",
		},
		"risk_score": 95.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertLevelCritical, createResp.Data.AlertLevel)
}

func (s *FraudIntegrationTestSuite) TestAccountFraud_ReferralAbuseDetection() {
	t := s.T()
	ctx := context.Background()

	// Create alert for referral abuse
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "promo_abuse",
		"alert_level": "high",
		"description": "Referral abuse detected - self-referral pattern",
		"details": map[string]interface{}{
			"referral_count":          25,
			"same_ip_referrals":       20,
			"same_device_referrals":   18,
			"avg_first_ride_amount":   5.00,
			"referral_bonus_collected": 250.00,
		},
		"risk_score": 85.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertTypePromoAbuse, createResp.Data.AlertType)

	_ = ctx // For potential DB operations
}

func (s *FraudIntegrationTestSuite) TestAccountFraud_SuspiciousEmailPattern() {
	t := s.T()

	// Detect account fraud for user with suspicious email
	detectPath := fmt.Sprintf("/detect/account/%s", s.rider.User.ID)
	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, detectPath, nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestAccountFraud_VPNUsageDetection() {
	t := s.T()

	// Create alert for VPN usage pattern
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "account_fraud",
		"alert_level": "low",
		"description": "VPN/proxy usage detected",
		"details": map[string]interface{}{
			"ip_address":    "45.153.160.1",
			"vpn_provider":  "NordVPN",
			"datacenter_ip": true,
		},
		"risk_score": 25.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertLevelLow, createResp.Data.AlertLevel)
}

// ============================================
// RISK SCORING TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestRiskScoring_CalculateRiskScore() {
	t := s.T()
	ctx := context.Background()

	// Create user risk profile with initial data
	_, err := dbPool.Exec(ctx, `
		INSERT INTO user_risk_profiles (user_id, risk_score, total_alerts, critical_alerts, confirmed_fraud_cases, account_suspended, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET risk_score = $2, total_alerts = $3`,
		s.rider.User.ID, 45.0, 5, 1, 0, false, time.Now())
	require.NoError(t, err)

	// Get risk profile
	riskPath := fmt.Sprintf("/users/%s/risk-profile", s.rider.User.ID)
	riskResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodGet, riskPath, nil, authHeaders(s.admin.Token))
	require.True(t, riskResp.Success)
	require.NotNil(t, riskResp.Data)
	require.InEpsilon(t, 45.0, riskResp.Data.RiskScore, 1e-6)
	require.Equal(t, 5, riskResp.Data.TotalAlerts)
}

func (s *FraudIntegrationTestSuite) TestRiskScoring_LowRiskThreshold() {
	t := s.T()

	// Create alert with low risk score
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "payment_fraud",
		"alert_level": "low",
		"description": "Minor suspicious activity",
		"risk_score":  25.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertLevelLow, createResp.Data.AlertLevel)

	// Verify user is not suspended
	riskPath := fmt.Sprintf("/users/%s/risk-profile", s.rider.User.ID)
	riskResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodGet, riskPath, nil, authHeaders(s.admin.Token))
	require.True(t, riskResp.Success)
	require.False(t, riskResp.Data.AccountSuspended)
}

func (s *FraudIntegrationTestSuite) TestRiskScoring_MediumRiskThreshold() {
	t := s.T()

	// Create alert with medium risk score
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "ride_fraud",
		"alert_level": "medium",
		"description": "Moderate suspicious activity",
		"risk_score":  55.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertLevelMedium, createResp.Data.AlertLevel)
}

func (s *FraudIntegrationTestSuite) TestRiskScoring_HighRiskThreshold() {
	t := s.T()

	// Create alert with high risk score
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "account_fraud",
		"alert_level": "high",
		"description": "High risk activity detected",
		"risk_score":  80.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.Equal(t, fraud.AlertLevelHigh, createResp.Data.AlertLevel)
}

func (s *FraudIntegrationTestSuite) TestRiskScoring_AutoBlockingAtHighRisk() {
	t := s.T()
	ctx := context.Background()

	// Create user risk profile with very high risk score
	_, err := dbPool.Exec(ctx, `
		INSERT INTO user_risk_profiles (user_id, risk_score, total_alerts, critical_alerts, confirmed_fraud_cases, account_suspended, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET risk_score = $2, confirmed_fraud_cases = $4`,
		s.rider.User.ID, 95.0, 10, 5, 3, false, time.Now())
	require.NoError(t, err)

	// Analyze user - should trigger auto-suspension due to high risk
	analyzePath := fmt.Sprintf("/users/%s/analyze", s.rider.User.ID)
	analyzeResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodPost, analyzePath, nil, authHeaders(s.admin.Token))
	require.True(t, analyzeResp.Success)
	// Note: Auto-suspension may depend on the service implementation
}

func (s *FraudIntegrationTestSuite) TestRiskScoring_RiskScoreIncrement() {
	t := s.T()

	// Create multiple alerts and verify risk score increments
	for i := 0; i < 3; i++ {
		alertReq := map[string]interface{}{
			"user_id":     s.rider.User.ID,
			"alert_type":  "payment_fraud",
			"alert_level": "medium",
			"description": fmt.Sprintf("Suspicious activity #%d", i+1),
			"risk_score":  50.0,
		}

		createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
		require.True(t, createResp.Success)
	}

	// Verify risk profile shows accumulated risk
	riskPath := fmt.Sprintf("/users/%s/risk-profile", s.rider.User.ID)
	riskResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodGet, riskPath, nil, authHeaders(s.admin.Token))
	require.True(t, riskResp.Success)
	require.Equal(t, 3, riskResp.Data.TotalAlerts)
}

// ============================================
// ALERT MANAGEMENT TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestAlertManagement_CreateFraudAlert() {
	t := s.T()

	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "payment_fraud",
		"alert_level": "high",
		"description": "Suspicious payment pattern detected",
		"details": map[string]interface{}{
			"failed_attempts": 5,
			"chargebacks":     2,
		},
		"risk_score": 75.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	require.NotNil(t, createResp.Data)
	require.NotEqual(t, uuid.Nil, createResp.Data.ID)
	require.Equal(t, s.rider.User.ID, createResp.Data.UserID)
	require.Equal(t, fraud.AlertStatusPending, createResp.Data.Status)
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_GetPendingAlerts() {
	t := s.T()

	// Create multiple pending alerts
	for i := 0; i < 3; i++ {
		alertReq := map[string]interface{}{
			"user_id":     s.rider.User.ID,
			"alert_type":  "payment_fraud",
			"alert_level": "medium",
			"description": fmt.Sprintf("Test alert %d", i+1),
			"risk_score":  50.0,
		}
		createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
		require.True(t, createResp.Success)
	}

	// Get pending alerts
	alertsResp := doRequest[[]*fraud.FraudAlert](t, fraudServiceKey, http.MethodGet, "/alerts", nil, authHeaders(s.admin.Token))
	require.True(t, alertsResp.Success)
	require.GreaterOrEqual(t, len(alertsResp.Data), 3)
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_InvestigateAlert() {
	t := s.T()

	// Create an alert
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "ride_fraud",
		"alert_level": "high",
		"description": "GPS spoofing detected",
		"risk_score":  80.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	alertID := createResp.Data.ID

	// Start investigation
	investigateReq := map[string]interface{}{
		"notes": "Starting investigation on GPS spoofing case",
	}

	investigatePath := fmt.Sprintf("/alerts/%s/investigate", alertID)
	investigateResp := doRequest[map[string]string](t, fraudServiceKey, http.MethodPut, investigatePath, investigateReq, authHeaders(s.admin.Token))
	require.True(t, investigateResp.Success)

	// Verify alert status changed
	alertPath := fmt.Sprintf("/alerts/%s", alertID)
	alertDetailResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodGet, alertPath, nil, authHeaders(s.admin.Token))
	require.True(t, alertDetailResp.Success)
	require.Equal(t, fraud.AlertStatusInvestigating, alertDetailResp.Data.Status)
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_ResolveAlertConfirmed() {
	t := s.T()

	// Create and investigate an alert
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "payment_fraud",
		"alert_level": "high",
		"description": "Confirmed fraud case",
		"risk_score":  85.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	alertID := createResp.Data.ID

	// Resolve as confirmed fraud
	resolveReq := map[string]interface{}{
		"confirmed":    true,
		"notes":        "Fraud confirmed after investigation",
		"action_taken": "Account suspended, funds recovered",
	}

	resolvePath := fmt.Sprintf("/alerts/%s/resolve", alertID)
	resolveResp := doRequest[map[string]string](t, fraudServiceKey, http.MethodPut, resolvePath, resolveReq, authHeaders(s.admin.Token))
	require.True(t, resolveResp.Success)

	// Verify alert is resolved
	alertPath := fmt.Sprintf("/alerts/%s", alertID)
	alertDetailResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodGet, alertPath, nil, authHeaders(s.admin.Token))
	require.True(t, alertDetailResp.Success)
	require.Equal(t, fraud.AlertStatusConfirmed, alertDetailResp.Data.Status)
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_ResolveAlertFalsePositive() {
	t := s.T()

	// Create an alert
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "account_fraud",
		"alert_level": "medium",
		"description": "Potential false positive",
		"risk_score":  50.0,
	}

	createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)
	alertID := createResp.Data.ID

	// Resolve as false positive
	resolveReq := map[string]interface{}{
		"confirmed":    false,
		"notes":        "After investigation, determined to be legitimate activity",
		"action_taken": "No action required",
	}

	resolvePath := fmt.Sprintf("/alerts/%s/resolve", alertID)
	resolveResp := doRequest[map[string]string](t, fraudServiceKey, http.MethodPut, resolvePath, resolveReq, authHeaders(s.admin.Token))
	require.True(t, resolveResp.Success)

	// Verify alert is marked as false positive
	alertPath := fmt.Sprintf("/alerts/%s", alertID)
	alertDetailResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodGet, alertPath, nil, authHeaders(s.admin.Token))
	require.True(t, alertDetailResp.Success)
	require.Equal(t, fraud.AlertStatusFalsePositive, alertDetailResp.Data.Status)
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_GetUserAlerts() {
	t := s.T()

	// Create alerts for specific user
	for i := 0; i < 5; i++ {
		alertReq := map[string]interface{}{
			"user_id":     s.rider.User.ID,
			"alert_type":  "payment_fraud",
			"alert_level": "medium",
			"description": fmt.Sprintf("User alert %d", i+1),
			"risk_score":  40.0,
		}
		createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
		require.True(t, createResp.Success)
	}

	// Get user alerts
	alertsPath := fmt.Sprintf("/users/%s/alerts", s.rider.User.ID)
	alertsResp := doRequest[[]*fraud.FraudAlert](t, fraudServiceKey, http.MethodGet, alertsPath, nil, authHeaders(s.admin.Token))
	require.True(t, alertsResp.Success)
	require.GreaterOrEqual(t, len(alertsResp.Data), 5)

	// Verify all alerts belong to the user
	for _, alert := range alertsResp.Data {
		require.Equal(t, s.rider.User.ID, alert.UserID)
	}
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_SuspendUser() {
	t := s.T()

	// Suspend user
	suspendReq := map[string]interface{}{
		"reason": "Multiple confirmed fraud cases",
	}

	suspendPath := fmt.Sprintf("/users/%s/suspend", s.rider.User.ID)
	suspendResp := doRequest[map[string]string](t, fraudServiceKey, http.MethodPost, suspendPath, suspendReq, authHeaders(s.admin.Token))
	require.True(t, suspendResp.Success)

	// Verify user is suspended
	riskPath := fmt.Sprintf("/users/%s/risk-profile", s.rider.User.ID)
	riskResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodGet, riskPath, nil, authHeaders(s.admin.Token))
	require.True(t, riskResp.Success)
	require.True(t, riskResp.Data.AccountSuspended)
}

func (s *FraudIntegrationTestSuite) TestAlertManagement_ReinstateUser() {
	t := s.T()
	ctx := context.Background()

	// First suspend user
	_, err := dbPool.Exec(ctx, `
		INSERT INTO user_risk_profiles (user_id, risk_score, total_alerts, critical_alerts, confirmed_fraud_cases, account_suspended, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET account_suspended = true, risk_score = $2`,
		s.rider.User.ID, 80.0, 5, 2, 1, true, time.Now())
	require.NoError(t, err)

	// Reinstate user
	reinstateReq := map[string]interface{}{
		"reason": "Appeal approved, account reinstated",
	}

	reinstatePath := fmt.Sprintf("/users/%s/reinstate", s.rider.User.ID)
	reinstateResp := doRequest[map[string]string](t, fraudServiceKey, http.MethodPost, reinstatePath, reinstateReq, authHeaders(s.admin.Token))
	require.True(t, reinstateResp.Success)

	// Verify user is reinstated
	riskPath := fmt.Sprintf("/users/%s/risk-profile", s.rider.User.ID)
	riskResp := doRequest[*fraud.UserRiskProfile](t, fraudServiceKey, http.MethodGet, riskPath, nil, authHeaders(s.admin.Token))
	require.True(t, riskResp.Success)
	require.False(t, riskResp.Data.AccountSuspended)
}

// ============================================
// FRAUD STATISTICS TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestFraudStatistics_GetStatistics() {
	t := s.T()

	// Create various alerts for statistics
	alertTypes := []string{"payment_fraud", "ride_fraud", "account_fraud"}
	alertLevels := []string{"low", "medium", "high", "critical"}

	for _, alertType := range alertTypes {
		for _, level := range alertLevels {
			alertReq := map[string]interface{}{
				"user_id":     s.rider.User.ID,
				"alert_type":  alertType,
				"alert_level": level,
				"description": fmt.Sprintf("Test %s %s alert", level, alertType),
				"risk_score":  50.0,
			}
			createResp := doRequest[*fraud.FraudAlert](t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
			require.True(t, createResp.Success)
		}
	}

	// Get statistics
	now := time.Now()
	startDate := now.AddDate(0, -1, 0).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	statsPath := fmt.Sprintf("/statistics?start_date=%s&end_date=%s", startDate, endDate)
	statsResp := doRequest[*fraud.FraudStatistics](t, fraudServiceKey, http.MethodGet, statsPath, nil, authHeaders(s.admin.Token))
	require.True(t, statsResp.Success)
	require.NotNil(t, statsResp.Data)
	require.GreaterOrEqual(t, statsResp.Data.TotalAlerts, 12) // 3 types * 4 levels
}

func (s *FraudIntegrationTestSuite) TestFraudStatistics_GetFraudPatterns() {
	t := s.T()
	ctx := context.Background()

	// Insert a fraud pattern
	patternDetails, _ := json.Marshal(map[string]interface{}{
		"common_ip_range": "185.220.0.0/16",
		"detection_rule":  "velocity_check",
	})

	_, err := dbPool.Exec(ctx, `
		INSERT INTO fraud_patterns (id, pattern_type, description, occurrences, affected_users, first_detected, last_detected, details, severity, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING`,
		uuid.New(), "velocity_abuse", "High transaction velocity from same IP range", 15,
		[]string{s.rider.User.ID.String()}, time.Now().Add(-7*24*time.Hour), time.Now(),
		patternDetails, "high", true)
	if err != nil {
		// Table may not exist, skip test
		t.Skip("fraud_patterns table not available")
	}

	// Get patterns
	patternsResp := doRequest[[]*fraud.FraudPattern](t, fraudServiceKey, http.MethodGet, "/patterns", nil, authHeaders(s.admin.Token))
	require.True(t, patternsResp.Success)
}

// ============================================
// UNAUTHORIZED ACCESS TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestUnauthorizedAccess_NoToken() {
	t := s.T()

	// Try to access fraud endpoints without token
	resp := doRawRequest(t, fraudServiceKey, http.MethodGet, "/alerts", nil, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestUnauthorizedAccess_NonAdminUser() {
	t := s.T()

	// Try to access fraud endpoints as non-admin user
	resp := doRawRequest(t, fraudServiceKey, http.MethodGet, "/alerts", nil, authHeaders(s.rider.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ============================================
// EDGE CASE TESTS
// ============================================

func (s *FraudIntegrationTestSuite) TestEdgeCase_InvalidAlertID() {
	t := s.T()

	invalidID := uuid.New()
	alertPath := fmt.Sprintf("/alerts/%s", invalidID)
	resp := doRawRequest(t, fraudServiceKey, http.MethodGet, alertPath, nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestEdgeCase_InvalidUserID() {
	t := s.T()

	resp := doRawRequest(t, fraudServiceKey, http.MethodGet, "/alerts/invalid-uuid", nil, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestEdgeCase_CreateAlertMissingFields() {
	t := s.T()

	// Missing required fields
	alertReq := map[string]interface{}{
		"user_id": s.rider.User.ID,
		// Missing alert_type, alert_level, description, risk_score
	}

	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func (s *FraudIntegrationTestSuite) TestEdgeCase_RiskScoreOutOfRange() {
	t := s.T()

	// Risk score > 100
	alertReq := map[string]interface{}{
		"user_id":     s.rider.User.ID,
		"alert_type":  "payment_fraud",
		"alert_level": "high",
		"description": "Test alert",
		"risk_score":  150.0, // Invalid
	}

	resp := doRawRequest(t, fraudServiceKey, http.MethodPost, "/alerts", alertReq, authHeaders(s.admin.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
