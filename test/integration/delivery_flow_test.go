//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/richxcame/ride-hailing/internal/delivery"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

const deliveryServiceKey = "delivery"

// DeliveryFlowTestSuite tests delivery service flows
type DeliveryFlowTestSuite struct {
	suite.Suite
	sender authSession
	driver authSession
	admin  authSession
}

func TestDeliveryFlowSuite(t *testing.T) {
	suite.Run(t, new(DeliveryFlowTestSuite))
}

func (s *DeliveryFlowTestSuite) SetupSuite() {
	// Ensure all services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[paymentsServiceKey]; !ok {
		services[paymentsServiceKey] = startPaymentsService(mustLoadConfig("payments-service"))
	}
	if _, ok := services[deliveryServiceKey]; !ok {
		services[deliveryServiceKey] = startDeliveryService()
	}
}

func (s *DeliveryFlowTestSuite) SetupTest() {
	truncateDeliveryTables(s.T())
	s.sender = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
	s.admin = registerAndLogin(s.T(), models.RoleAdmin)
}

func startDeliveryService() *serviceInstance {
	repo := delivery.NewRepository(dbPool)
	service := delivery.NewService(repo)
	handler := delivery.NewHandler(service)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	// Public tracking
	router.GET("/api/v1/deliveries/track/:code", handler.TrackDelivery)

	// Sender routes (any authenticated user can send packages)
	deliveries := router.Group("/api/v1/deliveries")
	deliveries.Use(middleware.AuthMiddleware("integration-secret"))
	{
		deliveries.POST("/estimate", handler.GetEstimate)
		deliveries.POST("", handler.CreateDelivery)
		deliveries.GET("", handler.GetMyDeliveries)
		deliveries.GET("/stats", handler.GetStats)
		deliveries.GET("/:id", handler.GetDelivery)
		deliveries.POST("/:id/cancel", handler.CancelDelivery)
		deliveries.POST("/:id/rate", handler.RateDelivery)
	}

	// Driver routes
	driverDeliveries := router.Group("/api/v1/driver/deliveries")
	driverDeliveries.Use(middleware.AuthMiddleware("integration-secret"))
	driverDeliveries.Use(middleware.RequireRole(models.RoleDriver))
	{
		driverDeliveries.GET("/available", handler.GetAvailableDeliveries)
		driverDeliveries.GET("/active", handler.GetActiveDelivery)
		driverDeliveries.GET("", handler.GetDriverDeliveries)
		driverDeliveries.POST("/:id/accept", handler.AcceptDelivery)
		driverDeliveries.POST("/:id/pickup", handler.ConfirmPickup)
		driverDeliveries.POST("/:id/status", handler.UpdateStatus)
		driverDeliveries.POST("/:id/deliver", handler.ConfirmDelivery)
		driverDeliveries.POST("/:id/return", handler.ReturnDelivery)
		driverDeliveries.POST("/:id/stops/:stopId/status", handler.UpdateStopStatus)
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func truncateDeliveryTables(t *testing.T) {
	t.Helper()
	truncateTables(t)

	// Truncate delivery-specific tables if they exist
	deliveryTables := []string{
		"delivery_tracking",
		"delivery_stops",
		"deliveries",
	}

	for _, table := range deliveryTables {
		_, _ = dbPool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}

// ============================================
// CREATING DELIVERY ORDERS TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestCreateDelivery_Success() {
	t := s.T()

	createReq := map[string]interface{}{
		"pickup_latitude":     37.7749,
		"pickup_longitude":    -122.4194,
		"pickup_address":      "123 Sender St, San Francisco, CA",
		"pickup_contact":      "John Sender",
		"pickup_phone":        "+15551234567",
		"dropoff_latitude":    37.8044,
		"dropoff_longitude":   -122.2712,
		"dropoff_address":     "456 Receiver Ave, Oakland, CA",
		"recipient_name":      "Jane Receiver",
		"recipient_phone":     "+15559876543",
		"package_size":        "small",
		"package_description": "Electronics - Handle with care",
		"is_fragile":          true,
		"requires_signature":  true,
		"priority":            "standard",
	}

	type deliveryResponse struct {
		ID                string  `json:"id"`
		TrackingCode      string  `json:"tracking_code"`
		Status            string  `json:"status"`
		EstimatedFare     float64 `json:"estimated_fare"`
		EstimatedDistance float64 `json:"estimated_distance"`
		EstimatedDuration int     `json:"estimated_duration"`
	}

	createResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries", createReq, authHeaders(s.sender.Token))
	require.True(t, createResp.Success)
	require.NotEmpty(t, createResp.Data.ID)
	require.NotEmpty(t, createResp.Data.TrackingCode)
	require.Equal(t, "requested", createResp.Data.Status)
	require.Greater(t, createResp.Data.EstimatedFare, 0.0)
}

func (s *DeliveryFlowTestSuite) TestCreateDelivery_ExpressPriority() {
	t := s.T()

	createReq := map[string]interface{}{
		"pickup_latitude":     37.7749,
		"pickup_longitude":    -122.4194,
		"pickup_address":      "Express Pickup Location",
		"pickup_contact":      "Express Sender",
		"pickup_phone":        "+15551111111",
		"dropoff_latitude":    37.7849,
		"dropoff_longitude":   -122.4094,
		"dropoff_address":     "Express Dropoff Location",
		"recipient_name":      "Express Receiver",
		"recipient_phone":     "+15552222222",
		"package_size":        "envelope",
		"package_description": "Urgent documents",
		"priority":            "express",
	}

	type deliveryResponse struct {
		ID       string `json:"id"`
		Priority string `json:"priority"`
		Status   string `json:"status"`
	}

	createResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries", createReq, authHeaders(s.sender.Token))
	require.True(t, createResp.Success)
	require.Equal(t, "express", createResp.Data.Priority)
}

func (s *DeliveryFlowTestSuite) TestCreateDelivery_ScheduledPickup() {
	t := s.T()

	scheduledTime := time.Now().Add(2 * time.Hour)

	createReq := map[string]interface{}{
		"pickup_latitude":      37.7749,
		"pickup_longitude":     -122.4194,
		"pickup_address":       "Scheduled Pickup",
		"pickup_contact":       "Scheduler",
		"pickup_phone":         "+15553333333",
		"dropoff_latitude":     37.8044,
		"dropoff_longitude":    -122.2712,
		"dropoff_address":      "Scheduled Dropoff",
		"recipient_name":       "Recipient",
		"recipient_phone":      "+15554444444",
		"package_size":         "medium",
		"package_description":  "Scheduled delivery",
		"priority":             "scheduled",
		"scheduled_pickup_at":  scheduledTime.Format(time.RFC3339),
	}

	type deliveryResponse struct {
		ID                 string     `json:"id"`
		Priority           string     `json:"priority"`
		ScheduledPickupAt  *time.Time `json:"scheduled_pickup_at"`
	}

	createResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries", createReq, authHeaders(s.sender.Token))
	require.True(t, createResp.Success)
	require.Equal(t, "scheduled", createResp.Data.Priority)
}

func (s *DeliveryFlowTestSuite) TestCreateDelivery_WithDeclaredValue() {
	t := s.T()

	declaredValue := 500.00

	createReq := map[string]interface{}{
		"pickup_latitude":     37.7749,
		"pickup_longitude":    -122.4194,
		"pickup_address":      "Valuable Pickup",
		"pickup_contact":      "Valuable Sender",
		"pickup_phone":        "+15555555555",
		"dropoff_latitude":    37.8044,
		"dropoff_longitude":   -122.2712,
		"dropoff_address":     "Valuable Dropoff",
		"recipient_name":      "Valuable Receiver",
		"recipient_phone":     "+15556666666",
		"package_size":        "small",
		"package_description": "Jewelry",
		"declared_value":      declaredValue,
		"requires_signature":  true,
		"priority":            "standard",
	}

	type deliveryResponse struct {
		ID            string   `json:"id"`
		DeclaredValue *float64 `json:"declared_value"`
	}

	createResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries", createReq, authHeaders(s.sender.Token))
	require.True(t, createResp.Success)
	require.NotNil(t, createResp.Data.DeclaredValue)
	require.InEpsilon(t, declaredValue, *createResp.Data.DeclaredValue, 1e-6)
}

func (s *DeliveryFlowTestSuite) TestCreateDelivery_MissingRequiredFields() {
	t := s.T()

	// Missing pickup information
	createReq := map[string]interface{}{
		"dropoff_latitude":    37.8044,
		"dropoff_longitude":   -122.2712,
		"dropoff_address":     "Dropoff Only",
		"recipient_name":      "Receiver",
		"recipient_phone":     "+15557777777",
		"package_size":        "small",
		"package_description": "Test",
		"priority":            "standard",
	}

	resp := doRawRequest(t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries", createReq, authHeaders(s.sender.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ============================================
// DRIVER ACCEPTING DELIVERIES TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestDriverAcceptDelivery_Success() {
	t := s.T()

	// Create a delivery
	deliveryID := s.createDelivery(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/accept", deliveryID)

	type deliveryResponse struct {
		ID       string     `json:"id"`
		DriverID *string    `json:"driver_id"`
		Status   string     `json:"status"`
	}

	acceptResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)
	require.Equal(t, "accepted", acceptResp.Data.Status)
	require.NotNil(t, acceptResp.Data.DriverID)
	require.Equal(t, s.driver.User.ID.String(), *acceptResp.Data.DriverID)
}

func (s *DeliveryFlowTestSuite) TestDriverAcceptDelivery_AlreadyAccepted() {
	t := s.T()

	// Create a delivery
	deliveryID := s.createDelivery(t)

	// First driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/accept", deliveryID)
	acceptResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Second driver tries to accept - should fail
	driver2 := registerAndLogin(t, models.RoleDriver)
	secondResp := doRawRequest(t, deliveryServiceKey, http.MethodPost, acceptPath, nil, authHeaders(driver2.Token))
	defer secondResp.Body.Close()
	require.Contains(t, []int{http.StatusBadRequest, http.StatusConflict}, secondResp.StatusCode)
}

func (s *DeliveryFlowTestSuite) TestGetAvailableDeliveries_Success() {
	t := s.T()

	// Create multiple deliveries
	for i := 0; i < 3; i++ {
		s.createDelivery(t)
	}

	// Get available deliveries near location
	availablePath := "/api/v1/driver/deliveries/available?latitude=37.7749&longitude=-122.4194"

	type availableResponse struct {
		Deliveries []map[string]interface{} `json:"deliveries"`
		Count      int                      `json:"count"`
	}

	availableResp := doRequest[availableResponse](t, deliveryServiceKey, http.MethodGet, availablePath, nil, authHeaders(s.driver.Token))
	require.True(t, availableResp.Success)
	require.GreaterOrEqual(t, availableResp.Data.Count, 3)
}

func (s *DeliveryFlowTestSuite) TestGetAvailableDeliveries_MissingLocation() {
	t := s.T()

	availableResp := doRawRequest(t, deliveryServiceKey, http.MethodGet, "/api/v1/driver/deliveries/available", nil, authHeaders(s.driver.Token))
	defer availableResp.Body.Close()
	require.Equal(t, http.StatusBadRequest, availableResp.StatusCode)
}

// ============================================
// PICKUP AND DROPOFF CONFIRMATIONS TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestConfirmPickup_Success() {
	t := s.T()
	ctx := context.Background()

	// Create and accept delivery
	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)

	// Confirm pickup
	pickupPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/pickup", deliveryID)
	pickupReq := map[string]interface{}{
		"photo_url": "https://example.com/pickup-photo.jpg",
		"notes":     "Package collected from reception",
	}

	type pickupResponse struct {
		Message string `json:"message"`
	}

	pickupResp := doRequest[pickupResponse](t, deliveryServiceKey, http.MethodPost, pickupPath, pickupReq, authHeaders(s.driver.Token))
	require.True(t, pickupResp.Success)
	require.Contains(t, pickupResp.Data.Message, "confirmed")

	// Verify status was updated
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "picked_up", status)
}

func (s *DeliveryFlowTestSuite) TestConfirmPickup_NotAccepted() {
	t := s.T()

	// Create delivery but don't accept it
	deliveryID := s.createDelivery(t)

	// Try to confirm pickup without accepting
	pickupPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/pickup", deliveryID)

	pickupResp := doRawRequest(t, deliveryServiceKey, http.MethodPost, pickupPath, nil, authHeaders(s.driver.Token))
	defer pickupResp.Body.Close()
	require.Contains(t, []int{http.StatusBadRequest, http.StatusForbidden}, pickupResp.StatusCode)
}

func (s *DeliveryFlowTestSuite) TestConfirmDelivery_WithSignature() {
	t := s.T()
	ctx := context.Background()

	// Create, accept, and pickup delivery
	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Update status to in_transit then arrived
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'arrived' WHERE id = $1`, deliveryID)

	// Confirm delivery with signature
	deliverPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/deliver", deliveryID)
	deliverReq := map[string]interface{}{
		"proof_type":    "signature",
		"signature_url": "https://example.com/signature.png",
		"notes":         "Delivered to recipient",
	}

	type deliverResponse struct {
		Message string `json:"message"`
	}

	deliverResp := doRequest[deliverResponse](t, deliveryServiceKey, http.MethodPost, deliverPath, deliverReq, authHeaders(s.driver.Token))
	require.True(t, deliverResp.Success)

	// Verify status was updated
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "delivered", status)
}

func (s *DeliveryFlowTestSuite) TestConfirmDelivery_WithPhoto() {
	t := s.T()
	ctx := context.Background()

	// Create full delivery flow
	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Update status to arrived
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'arrived' WHERE id = $1`, deliveryID)

	// Confirm delivery with photo
	deliverPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/deliver", deliveryID)
	deliverReq := map[string]interface{}{
		"proof_type": "photo",
		"photo_url":  "https://example.com/delivery-proof.jpg",
	}

	type deliverResponse struct {
		Message string `json:"message"`
	}

	deliverResp := doRequest[deliverResponse](t, deliveryServiceKey, http.MethodPost, deliverPath, deliverReq, authHeaders(s.driver.Token))
	require.True(t, deliverResp.Success)

	// Verify proof was stored
	var proofType, photoURL *string
	err := dbPool.QueryRow(ctx, `SELECT proof_type, proof_photo_url FROM deliveries WHERE id = $1`, deliveryID).Scan(&proofType, &photoURL)
	require.NoError(t, err)
	require.NotNil(t, proofType)
	require.Equal(t, "photo", *proofType)
}

func (s *DeliveryFlowTestSuite) TestConfirmDelivery_ContactlessDelivery() {
	t := s.T()
	ctx := context.Background()

	// Create full delivery flow
	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Update status to arrived
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'arrived' WHERE id = $1`, deliveryID)

	// Confirm contactless delivery
	deliverPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/deliver", deliveryID)
	deliverReq := map[string]interface{}{
		"proof_type": "contactless",
		"photo_url":  "https://example.com/left-at-door.jpg",
		"notes":      "Left at front door as requested",
	}

	deliverResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, deliverPath, deliverReq, authHeaders(s.driver.Token))
	require.True(t, deliverResp.Success)

	// Verify status
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "delivered", status)
}

// ============================================
// DELIVERY TRACKING TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestTrackDelivery_ByCode() {
	t := s.T()
	ctx := context.Background()

	// Create a delivery
	deliveryID := s.createDelivery(t)

	// Get tracking code
	var trackingCode string
	err := dbPool.QueryRow(ctx, `SELECT tracking_code FROM deliveries WHERE id = $1`, deliveryID).Scan(&trackingCode)
	require.NoError(t, err)

	// Track delivery (public endpoint)
	trackPath := fmt.Sprintf("/api/v1/deliveries/track/%s", trackingCode)

	type trackResponse struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		PickupAddress    string `json:"pickup_address"`
		DropoffAddress   string `json:"dropoff_address"`
	}

	trackResp := doRequest[trackResponse](t, deliveryServiceKey, http.MethodGet, trackPath, nil, nil)
	require.True(t, trackResp.Success)
	require.Equal(t, deliveryID.String(), trackResp.Data.ID)
}

func (s *DeliveryFlowTestSuite) TestTrackDelivery_InvalidCode() {
	t := s.T()

	trackResp := doRawRequest(t, deliveryServiceKey, http.MethodGet, "/api/v1/deliveries/track/INVALID123", nil, nil)
	defer trackResp.Body.Close()
	require.Equal(t, http.StatusNotFound, trackResp.StatusCode)
}

func (s *DeliveryFlowTestSuite) TestGetDelivery_AsSender() {
	t := s.T()

	deliveryID := s.createDelivery(t)

	getPath := fmt.Sprintf("/api/v1/deliveries/%s", deliveryID)

	type deliveryResponse struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		TrackingCode  string `json:"tracking_code"`
		Priority      string `json:"priority"`
	}

	getResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodGet, getPath, nil, authHeaders(s.sender.Token))
	require.True(t, getResp.Success)
	require.Equal(t, deliveryID.String(), getResp.Data.ID)
}

func (s *DeliveryFlowTestSuite) TestGetDelivery_AsDriver() {
	t := s.T()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)

	getPath := fmt.Sprintf("/api/v1/deliveries/%s", deliveryID)

	type deliveryResponse struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	getResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodGet, getPath, nil, authHeaders(s.driver.Token))
	require.True(t, getResp.Success)
	require.Equal(t, "accepted", getResp.Data.Status)
}

func (s *DeliveryFlowTestSuite) TestUpdateStatus_InTransit() {
	t := s.T()
	ctx := context.Background()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Update status to in_transit
	statusPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/status", deliveryID)
	statusReq := map[string]interface{}{
		"status": "in_transit",
	}

	statusResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, statusPath, statusReq, authHeaders(s.driver.Token))
	require.True(t, statusResp.Success)

	// Verify status
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "in_transit", status)
}

func (s *DeliveryFlowTestSuite) TestUpdateStatus_Arrived() {
	t := s.T()
	ctx := context.Background()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Update to in_transit first
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'in_transit' WHERE id = $1`, deliveryID)

	// Update status to arrived
	statusPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/status", deliveryID)
	statusReq := map[string]interface{}{
		"status": "arrived",
	}

	statusResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, statusPath, statusReq, authHeaders(s.driver.Token))
	require.True(t, statusResp.Success)

	// Verify status
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "arrived", status)
}

// ============================================
// CANCELLATION TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestCancelDelivery_BeforeAccept() {
	t := s.T()
	ctx := context.Background()

	deliveryID := s.createDelivery(t)

	// Cancel before driver accepts
	cancelPath := fmt.Sprintf("/api/v1/deliveries/%s/cancel", deliveryID)
	cancelReq := map[string]interface{}{
		"reason": "Changed my mind",
	}

	cancelResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.sender.Token))
	require.True(t, cancelResp.Success)

	// Verify status
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "cancelled", status)
}

func (s *DeliveryFlowTestSuite) TestCancelDelivery_AfterPickup_Fails() {
	t := s.T()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Try to cancel after pickup
	cancelPath := fmt.Sprintf("/api/v1/deliveries/%s/cancel", deliveryID)
	cancelReq := map[string]interface{}{
		"reason": "Don't need it anymore",
	}

	cancelResp := doRawRequest(t, deliveryServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.sender.Token))
	defer cancelResp.Body.Close()
	require.Contains(t, []int{http.StatusBadRequest, http.StatusForbidden}, cancelResp.StatusCode)
}

func (s *DeliveryFlowTestSuite) TestReturnDelivery_Success() {
	t := s.T()
	ctx := context.Background()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Update to arrived
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'arrived' WHERE id = $1`, deliveryID)

	// Return delivery (recipient not available)
	returnPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/return", deliveryID)
	returnReq := map[string]interface{}{
		"reason": "Recipient not available after multiple attempts",
	}

	returnResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, returnPath, returnReq, authHeaders(s.driver.Token))
	require.True(t, returnResp.Success)

	// Verify status
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM deliveries WHERE id = $1`, deliveryID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "returned", status)
}

// ============================================
// RATING TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestRateDelivery_AsSender() {
	t := s.T()
	ctx := context.Background()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Mark as delivered
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'delivered', delivered_at = NOW() WHERE id = $1`, deliveryID)

	// Rate delivery
	ratePath := fmt.Sprintf("/api/v1/deliveries/%s/rate", deliveryID)
	rateReq := map[string]interface{}{
		"rating":   5,
		"feedback": "Excellent service, very careful with the package!",
	}

	rateResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, ratePath, rateReq, authHeaders(s.sender.Token))
	require.True(t, rateResp.Success)

	// Verify rating was stored
	var senderRating *int
	err := dbPool.QueryRow(ctx, `SELECT sender_rating FROM deliveries WHERE id = $1`, deliveryID).Scan(&senderRating)
	require.NoError(t, err)
	require.NotNil(t, senderRating)
	require.Equal(t, 5, *senderRating)
}

func (s *DeliveryFlowTestSuite) TestRateDelivery_InvalidRating() {
	t := s.T()
	ctx := context.Background()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)
	s.confirmPickup(t, deliveryID)

	// Mark as delivered
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'delivered', delivered_at = NOW() WHERE id = $1`, deliveryID)

	// Try invalid rating
	ratePath := fmt.Sprintf("/api/v1/deliveries/%s/rate", deliveryID)
	rateReq := map[string]interface{}{
		"rating": 6, // Invalid - max is 5
	}

	rateResp := doRawRequest(t, deliveryServiceKey, http.MethodPost, ratePath, rateReq, authHeaders(s.sender.Token))
	defer rateResp.Body.Close()
	require.Equal(t, http.StatusBadRequest, rateResp.StatusCode)
}

// ============================================
// DELIVERY LISTING AND STATS TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestGetMyDeliveries_Success() {
	t := s.T()

	// Create multiple deliveries
	for i := 0; i < 5; i++ {
		s.createDelivery(t)
	}

	type listResponse struct {
		Deliveries []map[string]interface{} `json:"deliveries"`
	}

	listResp := doRequest[listResponse](t, deliveryServiceKey, http.MethodGet, "/api/v1/deliveries", nil, authHeaders(s.sender.Token))
	require.True(t, listResp.Success)
	require.GreaterOrEqual(t, len(listResp.Data.Deliveries), 5)
}

func (s *DeliveryFlowTestSuite) TestGetMyDeliveries_FilterByStatus() {
	t := s.T()
	ctx := context.Background()

	// Create deliveries with different statuses
	delivery1 := s.createDelivery(t)
	delivery2 := s.createDelivery(t)
	s.acceptDelivery(t, delivery2)

	// Update one to be in_transit
	_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'in_transit' WHERE id = $1`, delivery2)

	// Filter by requested status
	type listResponse struct {
		Deliveries []map[string]interface{} `json:"deliveries"`
	}

	listResp := doRequest[listResponse](t, deliveryServiceKey, http.MethodGet, "/api/v1/deliveries?status=requested", nil, authHeaders(s.sender.Token))
	require.True(t, listResp.Success)

	// Should only include the requested delivery
	for _, d := range listResp.Data.Deliveries {
		require.Equal(t, "requested", d["status"])
	}

	_ = delivery1 // Use variable
}

func (s *DeliveryFlowTestSuite) TestGetDriverDeliveries_Success() {
	t := s.T()
	ctx := context.Background()

	// Create and accept multiple deliveries
	for i := 0; i < 3; i++ {
		deliveryID := s.createDelivery(t)
		s.acceptDelivery(t, deliveryID)
		s.confirmPickup(t, deliveryID)
		// Complete delivery
		_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'delivered', delivered_at = NOW() WHERE id = $1`, deliveryID)
	}

	type listResponse struct {
		Deliveries []map[string]interface{} `json:"deliveries"`
	}

	listResp := doRequest[listResponse](t, deliveryServiceKey, http.MethodGet, "/api/v1/driver/deliveries", nil, authHeaders(s.driver.Token))
	require.True(t, listResp.Success)
	require.GreaterOrEqual(t, len(listResp.Data.Deliveries), 3)
}

func (s *DeliveryFlowTestSuite) TestGetActiveDelivery_HasActive() {
	t := s.T()

	deliveryID := s.createDelivery(t)
	s.acceptDelivery(t, deliveryID)

	type activeResponse struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	activeResp := doRequest[activeResponse](t, deliveryServiceKey, http.MethodGet, "/api/v1/driver/deliveries/active", nil, authHeaders(s.driver.Token))
	require.True(t, activeResp.Success)
	require.Equal(t, deliveryID.String(), activeResp.Data.ID)
}

func (s *DeliveryFlowTestSuite) TestGetStats_Success() {
	t := s.T()
	ctx := context.Background()

	// Create completed deliveries
	for i := 0; i < 3; i++ {
		deliveryID := s.createDelivery(t)
		s.acceptDelivery(t, deliveryID)
		s.confirmPickup(t, deliveryID)
		_, _ = dbPool.Exec(ctx, `UPDATE deliveries SET status = 'delivered', delivered_at = NOW(), final_fare = 25.00 WHERE id = $1`, deliveryID)
	}

	type statsResponse struct {
		TotalDeliveries int     `json:"total_deliveries"`
		CompletedCount  int     `json:"completed_count"`
		TotalSpent      float64 `json:"total_spent"`
	}

	statsResp := doRequest[statsResponse](t, deliveryServiceKey, http.MethodGet, "/api/v1/deliveries/stats", nil, authHeaders(s.sender.Token))
	require.True(t, statsResp.Success)
	require.GreaterOrEqual(t, statsResp.Data.TotalDeliveries, 3)
	require.GreaterOrEqual(t, statsResp.Data.CompletedCount, 3)
}

// ============================================
// FARE ESTIMATION TESTS
// ============================================

func (s *DeliveryFlowTestSuite) TestGetEstimate_Success() {
	t := s.T()

	estimateReq := map[string]interface{}{
		"pickup_latitude":   37.7749,
		"pickup_longitude":  -122.4194,
		"dropoff_latitude":  37.8044,
		"dropoff_longitude": -122.2712,
		"package_size":      "small",
		"priority":          "standard",
	}

	type estimateResponse struct {
		EstimatedDistance float64 `json:"estimated_distance"`
		EstimatedDuration int     `json:"estimated_duration"`
		TotalEstimate     float64 `json:"total_estimate"`
		Currency          string  `json:"currency"`
	}

	estimateResp := doRequest[estimateResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries/estimate", estimateReq, authHeaders(s.sender.Token))
	require.True(t, estimateResp.Success)
	require.Greater(t, estimateResp.Data.TotalEstimate, 0.0)
	require.Greater(t, estimateResp.Data.EstimatedDistance, 0.0)
	require.Greater(t, estimateResp.Data.EstimatedDuration, 0)
}

func (s *DeliveryFlowTestSuite) TestGetEstimate_DifferentSizes() {
	t := s.T()

	baseReq := map[string]interface{}{
		"pickup_latitude":   37.7749,
		"pickup_longitude":  -122.4194,
		"dropoff_latitude":  37.8044,
		"dropoff_longitude": -122.2712,
		"priority":          "standard",
	}

	sizes := []string{"envelope", "small", "medium", "large", "xlarge"}
	estimates := make(map[string]float64)

	for _, size := range sizes {
		req := map[string]interface{}{
			"pickup_latitude":   baseReq["pickup_latitude"],
			"pickup_longitude":  baseReq["pickup_longitude"],
			"dropoff_latitude":  baseReq["dropoff_latitude"],
			"dropoff_longitude": baseReq["dropoff_longitude"],
			"priority":          baseReq["priority"],
			"package_size":      size,
		}

		type estimateResponse struct {
			TotalEstimate float64 `json:"total_estimate"`
		}

		estimateResp := doRequest[estimateResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries/estimate", req, authHeaders(s.sender.Token))
		require.True(t, estimateResp.Success)
		estimates[size] = estimateResp.Data.TotalEstimate
	}

	// Larger packages should cost more
	require.LessOrEqual(t, estimates["envelope"], estimates["small"])
	require.LessOrEqual(t, estimates["small"], estimates["medium"])
	require.LessOrEqual(t, estimates["medium"], estimates["large"])
}

func (s *DeliveryFlowTestSuite) TestGetEstimate_ExpressPriority() {
	t := s.T()

	standardReq := map[string]interface{}{
		"pickup_latitude":   37.7749,
		"pickup_longitude":  -122.4194,
		"dropoff_latitude":  37.8044,
		"dropoff_longitude": -122.2712,
		"package_size":      "small",
		"priority":          "standard",
	}

	expressReq := map[string]interface{}{
		"pickup_latitude":   37.7749,
		"pickup_longitude":  -122.4194,
		"dropoff_latitude":  37.8044,
		"dropoff_longitude": -122.2712,
		"package_size":      "small",
		"priority":          "express",
	}

	type estimateResponse struct {
		TotalEstimate float64 `json:"total_estimate"`
	}

	standardResp := doRequest[estimateResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries/estimate", standardReq, authHeaders(s.sender.Token))
	require.True(t, standardResp.Success)

	expressResp := doRequest[estimateResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries/estimate", expressReq, authHeaders(s.sender.Token))
	require.True(t, expressResp.Success)

	// Express should cost more
	require.Greater(t, expressResp.Data.TotalEstimate, standardResp.Data.TotalEstimate)
}

// ============================================
// HELPER METHODS
// ============================================

func (s *DeliveryFlowTestSuite) createDelivery(t *testing.T) uuid.UUID {
	createReq := map[string]interface{}{
		"pickup_latitude":     37.7749,
		"pickup_longitude":    -122.4194,
		"pickup_address":      "Test Pickup Address",
		"pickup_contact":      "Test Sender",
		"pickup_phone":        "+15551234567",
		"dropoff_latitude":    37.8044,
		"dropoff_longitude":   -122.2712,
		"dropoff_address":     "Test Dropoff Address",
		"recipient_name":      "Test Recipient",
		"recipient_phone":     "+15559876543",
		"package_size":        "small",
		"package_description": "Test package",
		"priority":            "standard",
	}

	type deliveryResponse struct {
		ID string `json:"id"`
	}

	createResp := doRequest[deliveryResponse](t, deliveryServiceKey, http.MethodPost, "/api/v1/deliveries", createReq, authHeaders(s.sender.Token))
	require.True(t, createResp.Success)

	deliveryID, err := uuid.Parse(createResp.Data.ID)
	require.NoError(t, err)

	return deliveryID
}

func (s *DeliveryFlowTestSuite) acceptDelivery(t *testing.T, deliveryID uuid.UUID) {
	acceptPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/accept", deliveryID)
	acceptResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)
}

func (s *DeliveryFlowTestSuite) confirmPickup(t *testing.T, deliveryID uuid.UUID) {
	pickupPath := fmt.Sprintf("/api/v1/driver/deliveries/%s/pickup", deliveryID)
	pickupResp := doRequest[map[string]interface{}](t, deliveryServiceKey, http.MethodPost, pickupPath, nil, authHeaders(s.driver.Token))
	require.True(t, pickupResp.Success)
}
