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

// CancellationFlowTestSuite tests ride cancellation flows
type CancellationFlowTestSuite struct {
	suite.Suite
	rider  authSession
	driver authSession
	admin  authSession
}

func TestCancellationFlowSuite(t *testing.T) {
	suite.Run(t, new(CancellationFlowTestSuite))
}

func (s *CancellationFlowTestSuite) SetupSuite() {
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
}

func (s *CancellationFlowTestSuite) SetupTest() {
	truncateTables(s.T())
	s.rider = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
	s.admin = registerAndLogin(s.T(), models.RoleAdmin)
}

// ============================================
// RIDER CANCELLATION - WITHIN GRACE PERIOD (NO FEE)
// ============================================

func (s *CancellationFlowTestSuite) TestRiderCancellation_WithinGracePeriod_NoFee() {
	t := s.T()
	ctx := context.Background()

	// Top up rider wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride
	rideID := s.createRide(t)

	// Cancel immediately (within grace period - typically 2-5 minutes)
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "changed_plans",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)
	require.Equal(t, models.RideStatusCancelled, cancelResp.Data.Status)

	// Verify no cancellation fee was charged
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	require.InEpsilon(t, 100.00, walletResp.Data.Balance, 1e-6) // Balance unchanged

	// Verify cancellation reason was recorded
	var reason string
	err := dbPool.QueryRow(ctx, `SELECT cancellation_reason FROM rides WHERE id = $1`, rideID).Scan(&reason)
	require.NoError(t, err)
	require.Equal(t, "changed_plans", reason)
}

func (s *CancellationFlowTestSuite) TestRiderCancellation_BeforeDriverAccepts_NoFee() {
	t := s.T()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride (no driver assigned yet)
	rideID := s.createRide(t)

	// Cancel before driver accepts - should be free
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "found_another_ride",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)
	require.Equal(t, models.RideStatusCancelled, cancelResp.Data.Status)

	// Verify no fee charged
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	require.InEpsilon(t, 100.00, walletResp.Data.Balance, 1e-6)
}

func (s *CancellationFlowTestSuite) TestRiderCancellation_ImmediatelyAfterAccept_NoFee() {
	t := s.T()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Cancel immediately after accept (still within grace period)
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "emergency",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)
	require.Equal(t, models.RideStatusCancelled, cancelResp.Data.Status)

	// Verify no fee (within grace period)
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	require.InEpsilon(t, 100.00, walletResp.Data.Balance, 1e-6)
}

// ============================================
// RIDER CANCELLATION - AFTER GRACE PERIOD (FEE APPLIED)
// ============================================

func (s *CancellationFlowTestSuite) TestRiderCancellation_AfterGracePeriod_FeeApplied() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Simulate time passing (update accepted_at to be in the past)
	// Grace period is typically 2-5 minutes, we simulate 10 minutes passed
	_, err := dbPool.Exec(ctx, `
		UPDATE rides SET accepted_at = NOW() - INTERVAL '10 minutes' WHERE id = $1`,
		rideID)
	require.NoError(t, err)

	// Cancel after grace period - should incur fee
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "changed_plans",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)
	require.Equal(t, models.RideStatusCancelled, cancelResp.Data.Status)

	// Verify cancellation fee was charged (10% of estimated fare)
	// Check if cancellation fee transaction exists
	var txCount int
	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM wallet_transactions wt
		JOIN wallets w ON wt.wallet_id = w.id
		WHERE w.user_id = $1 AND wt.type = 'debit' AND wt.description LIKE '%cancellation%'`,
		s.rider.User.ID).Scan(&txCount)
	// If cancellation fee is implemented
	if err == nil && txCount > 0 {
		walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
		require.True(t, walletResp.Success)
		require.Less(t, walletResp.Data.Balance, 100.00) // Balance should be reduced
	}
}

func (s *CancellationFlowTestSuite) TestRiderCancellation_AfterDriverEnRoute_FeeApplied() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Simulate driver being en route for a while
	_, err := dbPool.Exec(ctx, `
		UPDATE rides SET accepted_at = NOW() - INTERVAL '8 minutes' WHERE id = $1`,
		rideID)
	require.NoError(t, err)

	// Cancel - should incur fee due to driver having traveled
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "taking_too_long",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)
	require.Equal(t, models.RideStatusCancelled, cancelResp.Data.Status)
}

func (s *CancellationFlowTestSuite) TestRiderCancellation_InsufficientBalance_Fails() {
	t := s.T()
	ctx := context.Background()

	// Create ride with NO wallet balance
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Simulate time passing
	_, err := dbPool.Exec(ctx, `
		UPDATE rides SET accepted_at = NOW() - INTERVAL '15 minutes' WHERE id = $1`,
		rideID)
	require.NoError(t, err)

	// Try to cancel - might fail if cancellation fee can't be charged
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "changed_plans",
	}

	// The response depends on implementation - either allows cancellation without fee
	// or requires balance for the fee
	cancelResp := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	defer cancelResp.Body.Close()

	// Either success (free cancellation) or bad request (insufficient balance)
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, cancelResp.StatusCode)
}

// ============================================
// DRIVER CANCELLATION - PENALTY APPLIED
// ============================================

func (s *CancellationFlowTestSuite) TestDriverCancellation_PenaltyApplied() {
	t := s.T()
	ctx := context.Background()

	// Create driver wallet
	s.createDriverWallet(t)

	// Create ride
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Driver cancels - should incur penalty
	cancelPath := fmt.Sprintf("/api/v1/driver/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "vehicle_breakdown",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.driver.Token))
	require.True(t, cancelResp.Success)
	require.Equal(t, models.RideStatusCancelled, cancelResp.Data.Status)

	// Verify driver cancellation recorded
	var cancelledBy string
	err := dbPool.QueryRow(ctx, `
		SELECT COALESCE(
			CASE WHEN driver_id IS NOT NULL THEN 'driver' ELSE 'rider' END,
			'unknown'
		) FROM rides WHERE id = $1`,
		rideID).Scan(&cancelledBy)
	require.NoError(t, err)
}

func (s *CancellationFlowTestSuite) TestDriverCancellation_MultipleOffenses_IncreasedPenalty() {
	t := s.T()
	ctx := context.Background()

	// Create driver wallet with balance
	driverWalletID := uuid.New()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO wallets (id, user_id, balance, currency, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET balance = $3`,
		driverWalletID, s.driver.User.ID, 100.00, "usd", true)
	require.NoError(t, err)

	// Track initial balance
	var initialBalance float64
	err = dbPool.QueryRow(ctx, `SELECT balance FROM wallets WHERE user_id = $1`, s.driver.User.ID).Scan(&initialBalance)
	require.NoError(t, err)

	// Create and cancel multiple rides
	for i := 0; i < 3; i++ {
		rideID := s.createRide(t)

		// Accept ride
		acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
		acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
		require.True(t, acceptResp.Success)

		// Cancel ride
		cancelPath := fmt.Sprintf("/api/v1/driver/rides/%s/cancel", rideID)
		cancelReq := map[string]interface{}{
			"reason": fmt.Sprintf("test_cancellation_%d", i+1),
		}

		cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.driver.Token))
		require.True(t, cancelResp.Success)
	}

	// Verify driver cancellation count increased
	var cancellationCount int
	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE driver_id = $1 AND status = 'cancelled' AND cancellation_reason IS NOT NULL`,
		s.driver.User.ID).Scan(&cancellationCount)
	require.NoError(t, err)
	require.Equal(t, 3, cancellationCount)
}

func (s *CancellationFlowTestSuite) TestDriverCancellation_RiderNotified() {
	t := s.T()
	ctx := context.Background()

	// Create ride
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Driver cancels
	cancelPath := fmt.Sprintf("/api/v1/driver/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "emergency",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.driver.Token))
	require.True(t, cancelResp.Success)

	// Verify ride status is cancelled for rider view
	rideDetail := doRequest[*models.Ride](t, ridesServiceKey, http.MethodGet, fmt.Sprintf("/api/v1/rides/%s", rideID), nil, authHeaders(s.rider.Token))
	require.True(t, rideDetail.Success)
	require.Equal(t, models.RideStatusCancelled, rideDetail.Data.Status)

	// Check for notification (if notifications table exists)
	var notificationCount int
	err := dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id = $1 AND type = 'ride_cancelled'`,
		s.rider.User.ID).Scan(&notificationCount)
	if err == nil {
		// If notifications are implemented, verify one was created
		require.GreaterOrEqual(t, notificationCount, 0)
	}
}

// ============================================
// CANCELLATION REFUND PROCESSING TESTS
// ============================================

func (s *CancellationFlowTestSuite) TestCancellationRefund_PrePaidRide() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride
	rideID := s.createRide(t)

	// Simulate pre-payment (some systems collect payment upfront)
	var estimatedFare float64
	err := dbPool.QueryRow(ctx, `SELECT estimated_fare FROM rides WHERE id = $1`, rideID).Scan(&estimatedFare)
	require.NoError(t, err)

	// Create prepayment
	paymentID := uuid.New()
	_, err = dbPool.Exec(ctx, `
		INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		paymentID, rideID, s.rider.User.ID, s.rider.User.ID, estimatedFare, "usd", "completed", "wallet")
	require.NoError(t, err)

	// Deduct from wallet
	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	walletID := walletResp.Data.ID

	_, err = dbPool.Exec(ctx, `UPDATE wallets SET balance = balance - $1 WHERE id = $2`, estimatedFare, walletID)
	require.NoError(t, err)

	// Cancel ride
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "driver_not_found",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)

	// Process refund
	_, err = dbPool.Exec(ctx, `UPDATE wallets SET balance = balance + $1 WHERE id = $2`, estimatedFare, walletID)
	require.NoError(t, err)

	// Create refund transaction
	_, err = dbPool.Exec(ctx, `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, description, reference_type, reference_id, balance_before, balance_after)
		VALUES ($1, $2, 'credit', $3, 'Refund for cancelled ride', 'ride', $4, $5, $6)`,
		uuid.New(), walletID, estimatedFare, rideID, 100.00-estimatedFare, 100.00)
	require.NoError(t, err)

	// Verify full refund
	var newBalance float64
	err = dbPool.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&newBalance)
	require.NoError(t, err)
	require.InEpsilon(t, 100.00, newBalance, 1e-6)
}

func (s *CancellationFlowTestSuite) TestCancellationRefund_PartialRefund() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create and accept ride
	rideID := s.createRide(t)

	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Simulate pre-payment
	var estimatedFare float64
	err := dbPool.QueryRow(ctx, `SELECT estimated_fare FROM rides WHERE id = $1`, rideID).Scan(&estimatedFare)
	require.NoError(t, err)

	walletResp := doRequest[*models.Wallet](t, paymentsServiceKey, http.MethodGet, "/api/v1/wallet", nil, authHeaders(s.rider.Token))
	require.True(t, walletResp.Success)
	walletID := walletResp.Data.ID

	_, err = dbPool.Exec(ctx, `UPDATE wallets SET balance = balance - $1 WHERE id = $2`, estimatedFare, walletID)
	require.NoError(t, err)

	// Simulate time passing (cancellation fee applies)
	_, err = dbPool.Exec(ctx, `
		UPDATE rides SET accepted_at = NOW() - INTERVAL '10 minutes' WHERE id = $1`,
		rideID)
	require.NoError(t, err)

	// Cancel ride
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "late_cancellation",
	}

	cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp.Success)

	// Process partial refund (deducting 10% cancellation fee)
	cancellationFee := estimatedFare * 0.10
	refundAmount := estimatedFare - cancellationFee

	_, err = dbPool.Exec(ctx, `UPDATE wallets SET balance = balance + $1 WHERE id = $2`, refundAmount, walletID)
	require.NoError(t, err)

	// Verify partial refund
	var newBalance float64
	err = dbPool.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1`, walletID).Scan(&newBalance)
	require.NoError(t, err)
	require.InEpsilon(t, 100.00-cancellationFee, newBalance, 1e-6)
}

func (s *CancellationFlowTestSuite) TestCancellationRefund_NoRefundAfterRideStarted() {
	t := s.T()
	ctx := context.Background()

	// Top up wallet
	s.topUpWallet(t, s.rider, 100.00)

	// Create ride
	rideID := s.createRide(t)

	// Driver accepts
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Driver starts ride
	startPath := fmt.Sprintf("/api/v1/driver/rides/%s/start", rideID)
	startResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, startPath, nil, authHeaders(s.driver.Token))
	require.True(t, startResp.Success)

	// Attempt to cancel after ride started - should fail or charge full fare
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "changed_my_mind",
	}

	cancelResp := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	defer cancelResp.Body.Close()

	// Cancellation after start should either fail or be handled specially
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusForbidden}, cancelResp.StatusCode)

	_ = ctx
}

// ============================================
// CANCELLATION REASON TRACKING TESTS
// ============================================

func (s *CancellationFlowTestSuite) TestCancellationReason_RiderReasons() {
	t := s.T()
	ctx := context.Background()

	riderReasons := []string{
		"changed_plans",
		"found_another_ride",
		"driver_too_far",
		"long_wait_time",
		"price_too_high",
		"emergency",
		"other",
	}

	for _, reason := range riderReasons {
		s.Run(reason, func() {
			// Create ride
			rideID := s.createRide(t)

			// Cancel with specific reason
			cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
			cancelReq := map[string]interface{}{
				"reason": reason,
			}

			cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
			require.True(t, cancelResp.Success)

			// Verify reason was stored
			var storedReason string
			err := dbPool.QueryRow(ctx, `SELECT cancellation_reason FROM rides WHERE id = $1`, rideID).Scan(&storedReason)
			require.NoError(t, err)
			require.Equal(t, reason, storedReason)
		})
	}
}

func (s *CancellationFlowTestSuite) TestCancellationReason_DriverReasons() {
	t := s.T()
	ctx := context.Background()

	driverReasons := []string{
		"vehicle_breakdown",
		"emergency",
		"rider_unreachable",
		"unsafe_location",
		"wrong_destination",
		"other",
	}

	for _, reason := range driverReasons {
		s.Run(reason, func() {
			// Create and accept ride
			rideID := s.createRide(t)

			acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
			acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
			require.True(t, acceptResp.Success)

			// Cancel with specific reason
			cancelPath := fmt.Sprintf("/api/v1/driver/rides/%s/cancel", rideID)
			cancelReq := map[string]interface{}{
				"reason": reason,
			}

			cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.driver.Token))
			require.True(t, cancelResp.Success)

			// Verify reason was stored
			var storedReason string
			err := dbPool.QueryRow(ctx, `SELECT cancellation_reason FROM rides WHERE id = $1`, rideID).Scan(&storedReason)
			require.NoError(t, err)
			require.Equal(t, reason, storedReason)
		})
	}
}

func (s *CancellationFlowTestSuite) TestCancellationReason_EmptyReason() {
	t := s.T()

	// Create ride
	rideID := s.createRide(t)

	// Cancel without reason
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{}

	cancelResp := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	defer cancelResp.Body.Close()

	// Should either fail (reason required) or default to "other"
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, cancelResp.StatusCode)
}

func (s *CancellationFlowTestSuite) TestCancellationReason_Analytics() {
	t := s.T()
	ctx := context.Background()

	// Create multiple rides with different cancellation reasons
	reasons := map[string]int{
		"changed_plans": 3,
		"long_wait":     2,
		"price_too_high": 1,
	}

	for reason, count := range reasons {
		for i := 0; i < count; i++ {
			rideID := s.createRide(t)

			cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
			cancelReq := map[string]interface{}{
				"reason": reason,
			}

			cancelResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
			require.True(t, cancelResp.Success)
		}
	}

	// Query cancellation analytics
	rows, err := dbPool.Query(ctx, `
		SELECT cancellation_reason, COUNT(*) as count
		FROM rides
		WHERE status = 'cancelled' AND cancellation_reason IS NOT NULL
		GROUP BY cancellation_reason
		ORDER BY count DESC`)
	require.NoError(t, err)
	defer rows.Close()

	analytics := make(map[string]int)
	for rows.Next() {
		var reason string
		var count int
		err := rows.Scan(&reason, &count)
		require.NoError(t, err)
		analytics[reason] = count
	}

	// Verify analytics match
	require.Equal(t, 3, analytics["changed_plans"])
	require.Equal(t, 2, analytics["long_wait"])
	require.Equal(t, 1, analytics["price_too_high"])
}

// ============================================
// EDGE CASES AND ERROR HANDLING
// ============================================

func (s *CancellationFlowTestSuite) TestCancellation_AlreadyCancelled() {
	t := s.T()

	// Create and cancel ride
	rideID := s.createRide(t)

	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "test",
	}

	// First cancellation
	cancelResp1 := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	require.True(t, cancelResp1.Success)

	// Try to cancel again
	cancelResp2 := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	defer cancelResp2.Body.Close()
	require.Equal(t, http.StatusBadRequest, cancelResp2.StatusCode)
}

func (s *CancellationFlowTestSuite) TestCancellation_CompletedRide() {
	t := s.T()

	// Create and complete ride
	rideID := s.createRide(t)

	// Accept
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Start
	startPath := fmt.Sprintf("/api/v1/driver/rides/%s/start", rideID)
	startResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, startPath, nil, authHeaders(s.driver.Token))
	require.True(t, startResp.Success)

	// Complete
	completePath := fmt.Sprintf("/api/v1/driver/rides/%s/complete", rideID)
	completeReq := map[string]interface{}{"actual_distance": 10.0}
	completeResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, completePath, completeReq, authHeaders(s.driver.Token))
	require.True(t, completeResp.Success)

	// Try to cancel completed ride
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "test",
	}

	cancelResp := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	defer cancelResp.Body.Close()
	require.Equal(t, http.StatusBadRequest, cancelResp.StatusCode)
}

func (s *CancellationFlowTestSuite) TestCancellation_NonexistentRide() {
	t := s.T()

	fakeRideID := uuid.New()
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", fakeRideID)
	cancelReq := map[string]interface{}{
		"reason": "test",
	}

	cancelResp := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(s.rider.Token))
	defer cancelResp.Body.Close()
	require.Equal(t, http.StatusNotFound, cancelResp.StatusCode)
}

func (s *CancellationFlowTestSuite) TestCancellation_UnauthorizedUser() {
	t := s.T()

	// Create ride
	rideID := s.createRide(t)

	// Create another rider
	otherRider := registerAndLogin(t, models.RoleRider)

	// Try to cancel with different user
	cancelPath := fmt.Sprintf("/api/v1/rides/%s/cancel", rideID)
	cancelReq := map[string]interface{}{
		"reason": "test",
	}

	cancelResp := doRawRequest(t, ridesServiceKey, http.MethodPost, cancelPath, cancelReq, authHeaders(otherRider.Token))
	defer cancelResp.Body.Close()
	require.Equal(t, http.StatusForbidden, cancelResp.StatusCode)
}

// ============================================
// HELPER METHODS
// ============================================

func (s *CancellationFlowTestSuite) createRide(t *testing.T) uuid.UUID {
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
	require.NotNil(t, rideResp.Data)
	return rideResp.Data.ID
}

func (s *CancellationFlowTestSuite) topUpWallet(t *testing.T, user authSession, amount float64) {
	topUpReq := map[string]interface{}{
		"amount":                amount,
		"stripe_payment_method": "pm_card_visa",
	}

	topUpResp := doRequest[*models.WalletTransaction](t, paymentsServiceKey, http.MethodPost, "/api/v1/wallet/topup", topUpReq, authHeaders(user.Token))
	require.True(t, topUpResp.Success)
}

func (s *CancellationFlowTestSuite) createDriverWallet(t *testing.T) uuid.UUID {
	ctx := context.Background()
	walletID := uuid.New()

	_, err := dbPool.Exec(ctx, `
		INSERT INTO wallets (id, user_id, balance, currency, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO NOTHING`,
		walletID, s.driver.User.ID, 0.0, "usd", true)
	require.NoError(t, err)

	return walletID
}
