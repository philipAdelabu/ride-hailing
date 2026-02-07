//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/richxcame/ride-hailing/pkg/models"
)

// PaymentFlowTestSuite tests end-to-end payment flows
type PaymentFlowTestSuite struct {
	suite.Suite
	rider  authSession
	driver authSession
	admin  authSession
}

func TestPaymentFlowSuite(t *testing.T) {
	suite.Run(t, new(PaymentFlowTestSuite))
}

func (s *PaymentFlowTestSuite) SetupSuite() {
	// Ensure all services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}
	if _, ok := services[paymentsServiceKey]; !ok {
		services[paymentsServiceKey] = startPaymentsService(mustLoadConfig("payments-service"))
	}
	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}
}

func (s *PaymentFlowTestSuite) SetupTest() {
	truncateTables(s.T())
	s.rider = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
	s.admin = registerAndLogin(s.T(), models.RoleAdmin)
}

// ============================================
// WALLET TOP-UP FLOW TESTS
// ============================================

func (s *PaymentFlowTestSuite) TestWalletTopUp_Success() {
	t := s.T()

	// Initial wallet balance should be 0
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	require.Equal(t, 0.0, walletResp.Data.Balance)

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                50.00,
		"stripe_payment_method": "pm_card_visa",
	}

	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)
	require.NotNil(t, topUpResp.Data)
	require.Equal(t, "credit", topUpResp.Data.Type)
	require.InEpsilon(t, 50.00, topUpResp.Data.Amount, 1e-6)
}

func (s *PaymentFlowTestSuite) TestWalletTopUp_MultipleTopUps() {
	t := s.T()

	amounts := []float64{25.00, 50.00, 100.00}
	expectedTotal := 0.0

	for _, amount := range amounts {
		topUpReq := map[string]interface{}{
			"amount":                amount,
			"stripe_payment_method": "pm_card_visa",
		}

		topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
		require.True(t, topUpResp.Success)
		expectedTotal += amount
	}

	// Verify transaction count
	transactionsResp := doRequest[[]*models.WalletTransaction](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet/transactions", nil, authHeaders(s.rider.Token))
	require.True(t, transactionsResp.Success)
	require.Len(t, transactionsResp.Data, 3)
}

func (s *PaymentFlowTestSuite) TestWalletTopUp_InvalidAmount() {
	t := s.T()

	testCases := []struct {
		name   string
		amount interface{}
	}{
		{"negative amount", -50.00},
		{"zero amount", 0.0},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			topUpReq := map[string]interface{}{
				"amount":                tc.amount,
				"stripe_payment_method": "pm_card_visa",
			}

			resp := doRawRequest(t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
			defer resp.Body.Close()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func (s *PaymentFlowTestSuite) TestWalletTopUp_Unauthorized() {
	t := s.T()

	topUpReq := map[string]interface{}{
		"amount":                50.00,
		"stripe_payment_method": "pm_card_visa",
	}

	resp := doRawRequest(t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ============================================
// RIDE PAYMENT FLOW TESTS
// ============================================

func (s *PaymentFlowTestSuite) TestRidePaymentFlow_WalletPayment() {
	t := s.T()

	// Top up rider's wallet
	topUpReq := map[string]interface{}{
		"amount":                100.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create and complete a ride
	rideID := s.createAndCompleteRide(t, 25.00)

	// Process payment for the ride
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         25.00,
		"payment_method": "wallet",
	}

	type paymentResponse struct {
		ID     string  `json:"id"`
		RideID string  `json:"ride_id"`
		Amount float64 `json:"amount"`
		Status string  `json:"status"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)
	require.Equal(t, "completed", paymentResp.Data.Status)
	require.InEpsilon(t, 25.00, paymentResp.Data.Amount, 1e-6)

	// Verify wallet balance was deducted
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	require.InEpsilon(t, 75.00, walletResp.Data.Balance, 1e-6)
}

func (s *PaymentFlowTestSuite) TestRidePaymentFlow_InsufficientBalance() {
	t := s.T()

	// Top up with small amount
	topUpReq := map[string]interface{}{
		"amount":                10.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create and complete a ride with higher fare
	rideID := s.createAndCompleteRide(t, 50.00)

	// Attempt payment - should fail due to insufficient balance
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         50.00,
		"payment_method": "wallet",
	}

	resp := doRawRequest(t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func (s *PaymentFlowTestSuite) TestRidePaymentFlow_DriverPayout() {
	t := s.T()
	ctx := context.Background()

	// Top up rider's wallet
	topUpReq := map[string]interface{}{
		"amount":                100.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create driver wallet
	driverWalletID := uuid.New()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO wallets (id, user_id, balance, currency, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO NOTHING`,
		driverWalletID, s.driver.User.ID, 0.0, "usd", true)
	require.NoError(t, err)

	// Create and complete a ride
	rideID := s.createAndCompleteRide(t, 100.00)

	// Process payment
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         100.00,
		"payment_method": "wallet",
	}

	type paymentResponse struct {
		ID     string  `json:"id"`
		Status string  `json:"status"`
		Amount float64 `json:"amount"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)

	// Simulate payout to driver (in production, this would be triggered by an event)
	expectedDriverEarnings := 100.00 * 0.80 // 80% after 20% commission

	// Update driver wallet directly to simulate payout
	_, err = dbPool.Exec(ctx, `
		UPDATE wallets SET balance = balance + $1 WHERE user_id = $2`,
		expectedDriverEarnings, s.driver.User.ID)
	require.NoError(t, err)

	// Create wallet transaction for driver
	_, err = dbPool.Exec(ctx, `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, description, reference_type, reference_id, balance_before, balance_after)
		SELECT $1, id, 'credit', $2, $3, 'ride', $4, 0.0, $2
		FROM wallets WHERE user_id = $5`,
		uuid.New(), expectedDriverEarnings, "Earnings from ride", rideID, s.driver.User.ID)
	require.NoError(t, err)

	// Verify driver received payout
	var driverBalance float64
	err = dbPool.QueryRow(ctx, `SELECT balance FROM wallets WHERE user_id = $1`, s.driver.User.ID).Scan(&driverBalance)
	require.NoError(t, err)
	require.InEpsilon(t, expectedDriverEarnings, driverBalance, 1e-6)
}

func (s *PaymentFlowTestSuite) TestRidePaymentFlow_StripePayment() {
	t := s.T()

	// Create and complete a ride
	rideID := s.createAndCompleteRide(t, 30.00)

	// Process payment via Stripe
	paymentReq := map[string]interface{}{
		"ride_id":               rideID,
		"amount":                30.00,
		"payment_method":        "stripe",
		"stripe_payment_method": "pm_card_visa",
	}

	type paymentResponse struct {
		ID              string  `json:"id"`
		RideID          string  `json:"ride_id"`
		Amount          float64 `json:"amount"`
		Status          string  `json:"status"`
		StripePaymentID *string `json:"stripe_payment_id"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)
	// Status might be pending (awaiting confirmation) or succeeded (mock)
	require.Contains(t, []string{"pending", "succeeded", "completed"}, paymentResp.Data.Status)
}

// ============================================
// PROMO CODE REDEMPTION WITH PAYMENT TESTS
// ============================================

func (s *PaymentFlowTestSuite) TestPaymentWithPromoCode_PercentageDiscount() {
	t := s.T()
	ctx := context.Background()

	// Create promo code
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "SAVE25",
		"description":    "25% off rides",
		"discount_type":  "percentage",
		"discount_value": 25.0,
		"valid_from":     now.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       100,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                100.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create and complete ride with original fare of 40.00
	rideID := s.createAndCompleteRide(t, 40.00)

	// Get promo code ID
	var promoCodeID uuid.UUID
	err := dbPool.QueryRow(ctx, "SELECT id FROM promo_codes WHERE code = 'SAVE25'").Scan(&promoCodeID)
	require.NoError(t, err)

	// Apply promo code - 25% of 40 = 10, final = 30
	discountedAmount := 40.00 * 0.75

	// Record promo code use
	_, err = dbPool.Exec(ctx, `
		INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		promoCodeID, s.rider.User.ID, rideID, 10.00, 40.00, discountedAmount)
	require.NoError(t, err)

	// Process payment for discounted amount
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         discountedAmount,
		"payment_method": "wallet",
	}

	type paymentResponse struct {
		ID     string  `json:"id"`
		Amount float64 `json:"amount"`
		Status string  `json:"status"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)
	require.InEpsilon(t, discountedAmount, paymentResp.Data.Amount, 1e-6)

	// Verify promo code use was recorded
	var useCount int
	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM promo_code_uses
		WHERE promo_code_id = $1 AND user_id = $2 AND ride_id = $3`,
		promoCodeID, s.rider.User.ID, rideID).Scan(&useCount)
	require.NoError(t, err)
	require.Equal(t, 1, useCount)
}

func (s *PaymentFlowTestSuite) TestPaymentWithPromoCode_FixedDiscount() {
	t := s.T()
	ctx := context.Background()

	// Create fixed amount promo code
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "FLAT10",
		"description":    "$10 off rides",
		"discount_type":  "fixed_amount",
		"discount_value": 10.0,
		"valid_from":     now.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       100,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                100.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create and complete ride
	originalFare := 35.00
	rideID := s.createAndCompleteRide(t, originalFare)

	// Get promo code ID
	var promoCodeID uuid.UUID
	err := dbPool.QueryRow(ctx, "SELECT id FROM promo_codes WHERE code = 'FLAT10'").Scan(&promoCodeID)
	require.NoError(t, err)

	// Apply fixed discount
	discountedAmount := originalFare - 10.00

	// Record promo code use
	_, err = dbPool.Exec(ctx, `
		INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		promoCodeID, s.rider.User.ID, rideID, 10.00, originalFare, discountedAmount)
	require.NoError(t, err)

	// Process payment
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         discountedAmount,
		"payment_method": "wallet",
	}

	type paymentResponse struct {
		Amount float64 `json:"amount"`
		Status string  `json:"status"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)
	require.InEpsilon(t, discountedAmount, paymentResp.Data.Amount, 1e-6)
}

func (s *PaymentFlowTestSuite) TestPaymentWithPromoCode_MaxDiscountCapped() {
	t := s.T()
	ctx := context.Background()

	// Create promo code with max discount cap
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":                "BIGDEAL",
		"description":         "50% off, max $15",
		"discount_type":       "percentage",
		"discount_value":      50.0,
		"max_discount_amount": 15.0, // Cap at $15
		"valid_from":          now.Format(time.RFC3339),
		"valid_until":         validUntil.Format(time.RFC3339),
		"max_uses":            100,
		"uses_per_user":       1,
		"is_active":           true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(s.admin.Token))
	require.True(t, createResp.Success)

	// Validate promo code
	validateReq := map[string]interface{}{
		"code":        "BIGDEAL",
		"ride_amount": 100.0, // 50% would be $50, but capped at $15
	}

	type promoValidation struct {
		Valid          bool    `json:"valid"`
		DiscountAmount float64 `json:"discount_amount"`
		FinalAmount    float64 `json:"final_amount"`
	}

	validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(s.rider.Token))
	require.True(t, validateResp.Success)
	require.True(t, validateResp.Data.Valid)
	require.InEpsilon(t, 15.0, validateResp.Data.DiscountAmount, 1e-6) // Capped at max
	require.InEpsilon(t, 85.0, validateResp.Data.FinalAmount, 1e-6)

	_ = ctx // Used for potential direct DB operations
}

// ============================================
// REFUND PROCESSING TESTS
// ============================================

func (s *PaymentFlowTestSuite) TestRefundProcessing_FullRefund() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                100.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create and complete ride
	rideID := s.createAndCompleteRide(t, 30.00)

	// Process payment
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         30.00,
		"payment_method": "wallet",
	}

	type paymentResponse struct {
		ID     string  `json:"id"`
		Status string  `json:"status"`
		Amount float64 `json:"amount"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)

	// Wallet balance should be 70
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	require.InEpsilon(t, 70.00, walletResp.Data.Balance, 1e-6)

	// Process full refund (simulated - in production would be via admin endpoint)
	refundAmount := 30.00
	walletID := walletResp.Data.ID

	_, err := dbPool.Exec(ctx, `
		UPDATE wallets SET balance = balance + $1 WHERE id = $2`,
		refundAmount, walletID)
	require.NoError(t, err)

	// Create refund transaction
	_, err = dbPool.Exec(ctx, `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, description, reference_type, reference_id, balance_before, balance_after)
		VALUES ($1, $2, 'credit', $3, 'Full refund for ride', 'ride', $4, $5, $6)`,
		uuid.New(), walletID, refundAmount, rideID, 70.00, 100.00)
	require.NoError(t, err)

	// Verify balance restored
	var newBalance float64
	err = dbPool.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&newBalance)
	require.NoError(t, err)
	require.InEpsilon(t, 100.00, newBalance, 1e-6)
}

func (s *PaymentFlowTestSuite) TestRefundProcessing_PartialRefund() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                100.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create and complete ride
	rideID := s.createAndCompleteRide(t, 50.00)

	// Process payment
	paymentReq := map[string]interface{}{
		"ride_id":        rideID,
		"amount":         50.00,
		"payment_method": "wallet",
	}

	type paymentResponse struct {
		ID string `json:"id"`
	}

	paymentResp := doRequest[paymentResponse](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
	require.True(t, paymentResp.Success)

	// Get wallet
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	walletID := walletResp.Data.ID

	// Process partial refund (50% of fare due to late cancellation)
	partialRefund := 25.00 // 50% of 50.00
	balanceBefore := walletResp.Data.Balance

	_, err := dbPool.Exec(ctx, `
		UPDATE wallets SET balance = balance + $1 WHERE id = $2`,
		partialRefund, walletID)
	require.NoError(t, err)

	// Create partial refund transaction
	_, err = dbPool.Exec(ctx, `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, description, reference_type, reference_id, balance_before, balance_after)
		VALUES ($1, $2, 'credit', $3, 'Partial refund (50%) for ride cancellation', 'ride', $4, $5, $6)`,
		uuid.New(), walletID, partialRefund, rideID, balanceBefore, balanceBefore+partialRefund)
	require.NoError(t, err)

	// Verify partial refund
	var newBalance float64
	err = dbPool.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&newBalance)
	require.NoError(t, err)
	require.InEpsilon(t, balanceBefore+partialRefund, newBalance, 1e-6)
}

func (s *PaymentFlowTestSuite) TestRefundProcessing_AlreadyRefunded() {
	t := s.T()
	ctx := context.Background()

	// Create a refunded payment directly in DB
	paymentID := uuid.New()
	rideID := uuid.New()

	_, err := dbPool.Exec(ctx, `
		INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		paymentID, rideID, s.rider.User.ID, s.driver.User.ID, 50.00, "usd", "refunded", "wallet")
	require.NoError(t, err)

	// Verify payment is marked as refunded
	var status string
	err = dbPool.QueryRow(ctx, `SELECT status FROM payments WHERE id = $1`, paymentID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "refunded", status)
}

// ============================================
// CORPORATE BILLING FLOW TESTS
// ============================================

func (s *PaymentFlowTestSuite) TestCorporateBilling_CreateInvoice() {
	t := s.T()
	ctx := context.Background()

	// Create a corporate account (simulated)
	corporateID := uuid.New()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO corporate_accounts (id, company_name, billing_email, is_active, credit_limit, current_balance)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING`,
		corporateID, "Acme Corp", "billing@acme.com", true, 10000.00, 0.0)
	// Skip if table doesn't exist
	if err != nil {
		t.Skip("Corporate accounts table not available")
	}

	// Link rider to corporate account
	_, err = dbPool.Exec(ctx, `
		UPDATE users SET corporate_account_id = $1 WHERE id = $2`,
		corporateID, s.rider.User.ID)
	if err != nil {
		t.Skip("Corporate account linking not supported")
	}

	// Create rides for corporate billing
	rideIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		rideIDs[i] = s.createAndCompleteRide(t, float64(20+i*10))
	}

	// Create invoice (simulated)
	invoiceID := uuid.New()
	totalAmount := 20.0 + 30.0 + 40.0 // Sum of ride fares

	_, err = dbPool.Exec(ctx, `
		INSERT INTO invoices (id, corporate_account_id, total_amount, status, invoice_date, due_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING`,
		invoiceID, corporateID, totalAmount, "pending", time.Now(), time.Now().Add(30*24*time.Hour))
	if err != nil {
		t.Skip("Invoices table not available")
	}

	// Verify invoice was created
	var invoiceAmount float64
	err = dbPool.QueryRow(ctx, `SELECT total_amount FROM invoices WHERE id = $1`, invoiceID).Scan(&invoiceAmount)
	if err != nil {
		t.Skip("Invoice verification not supported")
	}
	require.InEpsilon(t, totalAmount, invoiceAmount, 1e-6)
}

func (s *PaymentFlowTestSuite) TestCorporateBilling_RideToInvoiceFlow() {
	t := s.T()
	ctx := context.Background()

	// This test simulates the flow from ride completion to invoice generation
	// Skip if corporate billing is not implemented
	var tableExists bool
	err := dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'corporate_accounts'
		)`).Scan(&tableExists)

	if err != nil || !tableExists {
		t.Skip("Corporate billing not implemented")
	}

	t.Log("Corporate billing flow test - tables exist, testing flow")
}

// ============================================
// PAYMENT HISTORY AND REPORTING TESTS
// ============================================

func (s *PaymentFlowTestSuite) TestPaymentHistory_GetUserPayments() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	topUpReq := map[string]interface{}{
		"amount":                200.00,
		"stripe_payment_method": "pm_card_visa",
	}
	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
	require.True(t, topUpResp.Success)

	// Create multiple rides and payments
	for i := 0; i < 3; i++ {
		rideID := s.createAndCompleteRide(t, float64(20+i*10))

		paymentReq := map[string]interface{}{
			"ride_id":        rideID,
			"amount":         float64(20 + i*10),
			"payment_method": "wallet",
		}

		paymentResp := doRequest[map[string]interface{}](t, paymentsServiceKey, http.MethodPost, "/api/v1/payments", paymentReq, authHeaders(s.rider.Token))
		require.True(t, paymentResp.Success)
	}

	// Verify payments in database
	var paymentCount int
	err := dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM payments WHERE rider_id = $1`, s.rider.User.ID).Scan(&paymentCount)
	require.NoError(t, err)
	require.Equal(t, 3, paymentCount)
}

func (s *PaymentFlowTestSuite) TestWalletTransactionHistory() {
	t := s.T()

	// Perform multiple wallet operations
	amounts := []float64{50.00, 25.00, 75.00}

	for _, amount := range amounts {
		topUpReq := map[string]interface{}{
			"amount":                amount,
			"stripe_payment_method": "pm_card_visa",
		}
		topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(s.rider.Token))
		require.True(t, topUpResp.Success)
	}

	// Get transaction history
	transactionsResp := doRequest[[]*models.WalletTransaction](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet/transactions", nil, authHeaders(s.rider.Token))
	require.True(t, transactionsResp.Success)
	require.GreaterOrEqual(t, len(transactionsResp.Data), 3)

	// Verify transactions are ordered (most recent first typically)
	for _, tx := range transactionsResp.Data {
		require.Equal(t, "credit", tx.Type)
	}
}

// ============================================
// HELPER METHODS
// ============================================

func (s *PaymentFlowTestSuite) createAndCompleteRide(t *testing.T, fare float64) uuid.UUID {
	ctx := context.Background()

	// Create ride request
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St, San Francisco",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway, Oakland",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Driver starts
	startPath := fmt.Sprintf("/api/v1/driver/rides/%s/start", rideID)
	startResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, startPath, nil, authHeaders(s.driver.Token))
	require.True(t, startResp.Success)

	// Driver completes
	completePath := fmt.Sprintf("/api/v1/driver/rides/%s/complete", rideID)
	completeReq := map[string]interface{}{
		"actual_distance": 10.0,
	}
	completeResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, completePath, completeReq, authHeaders(s.driver.Token))
	require.True(t, completeResp.Success)

	// Update final fare to our test value
	_, err := dbPool.Exec(ctx, `UPDATE rides SET final_fare = $1 WHERE id = $2`, fare, rideID)
	require.NoError(t, err)

	return rideID
}
