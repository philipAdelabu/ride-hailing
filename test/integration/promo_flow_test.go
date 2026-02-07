//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/pkg/models"
)

// promoValidation is used for promo validation response parsing
type promoValidation struct {
	Valid          bool    `json:"valid"`
	DiscountAmount float64 `json:"discount_amount"`
	FinalAmount    float64 `json:"final_amount"`
	Message        string  `json:"message"`
}

// referralCodeResponse is used for referral code response parsing
type referralCodeResponse struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	Code           string  `json:"code"`
	TotalReferrals int     `json:"total_referrals"`
	TotalEarnings  float64 `json:"total_earnings"`
}

// applyReferralResponse is used for apply referral response parsing
type applyReferralResponse struct {
	Message string  `json:"message"`
	Bonus   float64 `json:"bonus"`
}

// TestPromoFlow_AdminCreatePromoCode tests admin promo code creation with various configurations
func TestPromoFlow_AdminCreatePromoCode(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	t.Run("create percentage promo code", func(t *testing.T) {
		now := time.Now()
		validUntil := now.Add(30 * 24 * time.Hour)
		maxDiscountAmount := 25.0
		maxUses := 500

		createPromoReq := map[string]interface{}{
			"code":                "PERCENT25",
			"description":         "25% off up to $25",
			"discount_type":       "percentage",
			"discount_value":      25.0,
			"max_discount_amount": maxDiscountAmount,
			"valid_from":          now.Format(time.RFC3339),
			"valid_until":         validUntil.Format(time.RFC3339),
			"max_uses":            maxUses,
			"uses_per_user":       2,
			"is_active":           true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)
		require.NotNil(t, createResp.Data["id"])
	})

	t.Run("create fixed amount promo code", func(t *testing.T) {
		now := time.Now()
		validUntil := now.Add(30 * 24 * time.Hour)
		minRideAmount := 30.0
		maxUses := 100

		createPromoReq := map[string]interface{}{
			"code":            "FLAT10OFF",
			"description":     "$10 off rides over $30",
			"discount_type":   "fixed_amount",
			"discount_value":  10.0,
			"min_ride_amount": minRideAmount,
			"valid_from":      now.Format(time.RFC3339),
			"valid_until":     validUntil.Format(time.RFC3339),
			"max_uses":        maxUses,
			"uses_per_user":   1,
			"is_active":       true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)
	})

	t.Run("rider cannot create promo code", func(t *testing.T) {
		now := time.Now()
		validUntil := now.Add(30 * 24 * time.Hour)

		createPromoReq := map[string]interface{}{
			"code":           "HACKPROMO",
			"description":    "Unauthorized promo",
			"discount_type":  "percentage",
			"discount_value": 50.0,
			"valid_from":     now.Format(time.RFC3339),
			"valid_until":    validUntil.Format(time.RFC3339),
			"max_uses":       100,
			"uses_per_user":  1,
			"is_active":      true,
		}

		resp := doRawRequest(t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(rider.Token))
		defer resp.Body.Close()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// TestPromoFlow_ApplyPromoCodeToRide tests applying promo codes to rides
func TestPromoFlow_ApplyPromoCodeToRide(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create a percentage promo code
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":                "RIDE30",
		"description":         "30% off your ride",
		"discount_type":       "percentage",
		"discount_value":      30.0,
		"max_discount_amount": 15.0,
		"valid_from":          now.Format(time.RFC3339),
		"valid_until":         validUntil.Format(time.RFC3339),
		"max_uses":            1000,
		"uses_per_user":       3,
		"is_active":           true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	t.Run("validate promo code with standard ride amount", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "RIDE30",
			"ride_amount": 40.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)
		// 30% of $40 = $12, under max cap of $15
		require.InEpsilon(t, 12.0, validateResp.Data.DiscountAmount, 1e-6)
		require.InEpsilon(t, 28.0, validateResp.Data.FinalAmount, 1e-6)
	})

	t.Run("validate promo code with high ride amount hits max cap", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "RIDE30",
			"ride_amount": 100.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)
		// 30% of $100 = $30, but capped at max $15
		require.InEpsilon(t, 15.0, validateResp.Data.DiscountAmount, 1e-6)
		require.InEpsilon(t, 85.0, validateResp.Data.FinalAmount, 1e-6)
	})

	t.Run("simulate promo code use and verify discount in database", func(t *testing.T) {
		// Get promo code ID from database
		var promoCodeID uuid.UUID
		err := dbPool.QueryRow(context.Background(), "SELECT id FROM promo_codes WHERE code = 'RIDE30'").Scan(&promoCodeID)
		require.NoError(t, err)

		// Simulate a ride with this promo code
		rideID := uuid.New()
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO rides (id, rider_id, status, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, estimated_fare, promo_code_id, discount_amount)
			VALUES ($1, $2, 'completed', 37.7749, -122.4194, 37.8044, -122.2712, 40.0, $3, 12.0)`,
			rideID, rider.User.ID, promoCodeID)
		require.NoError(t, err)

		// Record promo code use
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			promoCodeID, rider.User.ID, rideID, 12.0, 40.0, 28.0)
		require.NoError(t, err)

		// Verify promo code use is recorded
		var useCount int
		err = dbPool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM promo_code_uses WHERE promo_code_id = $1 AND user_id = $2",
			promoCodeID, rider.User.ID).Scan(&useCount)
		require.NoError(t, err)
		require.Equal(t, 1, useCount)
	})
}

// TestPromoFlow_ValidateMinimumAmountRestriction tests minimum ride amount restriction
func TestPromoFlow_ValidateMinimumAmountRestriction(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create promo code with minimum amount requirement
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)
	minRideAmount := 50.0

	createPromoReq := map[string]interface{}{
		"code":            "MIN50",
		"description":     "Requires minimum $50 ride",
		"discount_type":   "fixed_amount",
		"discount_value":  15.0,
		"min_ride_amount": minRideAmount,
		"valid_from":      now.Format(time.RFC3339),
		"valid_until":     validUntil.Format(time.RFC3339),
		"max_uses":        100,
		"uses_per_user":   1,
		"is_active":       true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	t.Run("ride amount below minimum is rejected", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "MIN50",
			"ride_amount": 30.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Contains(t, validateResp.Data.Message, "Minimum ride amount")
	})

	t.Run("ride amount at minimum is accepted", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "MIN50",
			"ride_amount": 50.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)
		require.InEpsilon(t, 15.0, validateResp.Data.DiscountAmount, 1e-6)
		require.InEpsilon(t, 35.0, validateResp.Data.FinalAmount, 1e-6)
	})

	t.Run("ride amount above minimum is accepted", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "MIN50",
			"ride_amount": 75.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)
		require.InEpsilon(t, 15.0, validateResp.Data.DiscountAmount, 1e-6)
		require.InEpsilon(t, 60.0, validateResp.Data.FinalAmount, 1e-6)
	})
}

// TestPromoFlow_FirstRideOnlyRestriction tests first ride only promo codes
func TestPromoFlow_FirstRideOnlyRestriction(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	newRider := registerAndLogin(t, models.RoleRider)

	// Create a first-ride promo code (uses_per_user = 1)
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "FIRSTRIDE",
		"description":    "First ride discount",
		"discount_type":  "percentage",
		"discount_value": 50.0,
		"valid_from":     now.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       10000,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	t.Run("new user can use first ride promo", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "FIRSTRIDE",
			"ride_amount": 30.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(newRider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)
		require.InEpsilon(t, 15.0, validateResp.Data.DiscountAmount, 1e-6)
	})

	t.Run("user cannot use first ride promo twice", func(t *testing.T) {
		// Get promo code ID
		var promoCodeID uuid.UUID
		err := dbPool.QueryRow(context.Background(), "SELECT id FROM promo_codes WHERE code = 'FIRSTRIDE'").Scan(&promoCodeID)
		require.NoError(t, err)

		// Simulate first ride with promo code
		rideID := uuid.New()
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO rides (id, rider_id, status, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, estimated_fare)
			VALUES ($1, $2, 'completed', 37.7749, -122.4194, 37.8044, -122.2712, 30.0)`,
			rideID, newRider.User.ID)
		require.NoError(t, err)

		// Record promo code use
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			promoCodeID, newRider.User.ID, rideID, 15.0, 30.0, 15.0)
		require.NoError(t, err)

		// Try to validate again
		validateReq := map[string]interface{}{
			"code":        "FIRSTRIDE",
			"ride_amount": 30.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(newRider.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Contains(t, validateResp.Data.Message, "already used")
	})
}

// TestPromoFlow_UsageLimits tests promo code global and per-user usage limits
func TestPromoFlow_UsageLimits(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider1 := registerAndLogin(t, models.RoleRider)
	rider2 := registerAndLogin(t, models.RoleRider)
	rider3 := registerAndLogin(t, models.RoleRider)

	t.Run("test global max uses limit", func(t *testing.T) {
		now := time.Now()
		validUntil := now.Add(30 * 24 * time.Hour)

		// Create promo with only 2 total uses allowed
		createPromoReq := map[string]interface{}{
			"code":           "LIMITED2",
			"description":    "Only 2 uses total",
			"discount_type":  "fixed_amount",
			"discount_value": 5.0,
			"valid_from":     now.Format(time.RFC3339),
			"valid_until":    validUntil.Format(time.RFC3339),
			"max_uses":       2,
			"uses_per_user":  1,
			"is_active":      true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)

		// Get promo code ID
		var promoCodeID uuid.UUID
		err := dbPool.QueryRow(context.Background(), "SELECT id FROM promo_codes WHERE code = 'LIMITED2'").Scan(&promoCodeID)
		require.NoError(t, err)

		// First user uses promo
		rideID1 := uuid.New()
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO rides (id, rider_id, status, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, estimated_fare)
			VALUES ($1, $2, 'completed', 37.7749, -122.4194, 37.8044, -122.2712, 20.0)`,
			rideID1, rider1.User.ID)
		require.NoError(t, err)

		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			promoCodeID, rider1.User.ID, rideID1, 5.0, 20.0, 15.0)
		require.NoError(t, err)

		// Update total uses
		_, err = dbPool.Exec(context.Background(), `UPDATE promo_codes SET total_uses = total_uses + 1 WHERE id = $1`, promoCodeID)
		require.NoError(t, err)

		// Second user uses promo
		rideID2 := uuid.New()
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO rides (id, rider_id, status, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, estimated_fare)
			VALUES ($1, $2, 'completed', 37.7749, -122.4194, 37.8044, -122.2712, 20.0)`,
			rideID2, rider2.User.ID)
		require.NoError(t, err)

		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			promoCodeID, rider2.User.ID, rideID2, 5.0, 20.0, 15.0)
		require.NoError(t, err)

		// Update total uses again
		_, err = dbPool.Exec(context.Background(), `UPDATE promo_codes SET total_uses = total_uses + 1 WHERE id = $1`, promoCodeID)
		require.NoError(t, err)

		// Third user should be rejected (max uses reached)
		validateReq := map[string]interface{}{
			"code":        "LIMITED2",
			"ride_amount": 20.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider3.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Contains(t, validateResp.Data.Message, "maximum usage limit")
	})

	t.Run("test per-user uses limit", func(t *testing.T) {
		now := time.Now()
		validUntil := now.Add(30 * 24 * time.Hour)

		// Create promo with 2 uses per user
		createPromoReq := map[string]interface{}{
			"code":           "TWOPERUSER",
			"description":    "2 uses per user",
			"discount_type":  "fixed_amount",
			"discount_value": 3.0,
			"valid_from":     now.Format(time.RFC3339),
			"valid_until":    validUntil.Format(time.RFC3339),
			"max_uses":       1000,
			"uses_per_user":  2,
			"is_active":      true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)

		// Get promo code ID
		var promoCodeID uuid.UUID
		err := dbPool.QueryRow(context.Background(), "SELECT id FROM promo_codes WHERE code = 'TWOPERUSER'").Scan(&promoCodeID)
		require.NoError(t, err)

		// Create a new rider for this test
		testRider := registerAndLogin(t, models.RoleRider)

		// First validation should succeed
		validateReq := map[string]interface{}{
			"code":        "TWOPERUSER",
			"ride_amount": 15.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(testRider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)

		// Simulate first use
		rideID1 := uuid.New()
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO rides (id, rider_id, status, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, estimated_fare)
			VALUES ($1, $2, 'completed', 37.7749, -122.4194, 37.8044, -122.2712, 15.0)`,
			rideID1, testRider.User.ID)
		require.NoError(t, err)

		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			promoCodeID, testRider.User.ID, rideID1, 3.0, 15.0, 12.0)
		require.NoError(t, err)

		// Second validation should still succeed
		validateResp2 := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(testRider.Token))
		require.True(t, validateResp2.Success)
		require.True(t, validateResp2.Data.Valid)

		// Simulate second use
		rideID2 := uuid.New()
		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO rides (id, rider_id, status, pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, estimated_fare)
			VALUES ($1, $2, 'completed', 37.7749, -122.4194, 37.8044, -122.2712, 15.0)`,
			rideID2, testRider.User.ID)
		require.NoError(t, err)

		_, err = dbPool.Exec(context.Background(), `
			INSERT INTO promo_code_uses (promo_code_id, user_id, ride_id, discount_amount, original_amount, final_amount)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			promoCodeID, testRider.User.ID, rideID2, 3.0, 15.0, 12.0)
		require.NoError(t, err)

		// Third validation should fail (per-user limit reached)
		validateResp3 := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(testRider.Token))
		require.True(t, validateResp3.Success)
		require.False(t, validateResp3.Data.Valid)
		require.Contains(t, validateResp3.Data.Message, "already used")
	})
}

// TestPromoFlow_ExpiredPromoCodes tests expired promo code handling
func TestPromoFlow_ExpiredPromoCodes(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	t.Run("expired promo code is rejected", func(t *testing.T) {
		// Create already expired promo code
		validFrom := time.Now().Add(-14 * 24 * time.Hour)
		validUntil := time.Now().Add(-1 * 24 * time.Hour)

		createPromoReq := map[string]interface{}{
			"code":           "PASTPROMO",
			"description":    "Already expired",
			"discount_type":  "percentage",
			"discount_value": 20.0,
			"valid_from":     validFrom.Format(time.RFC3339),
			"valid_until":    validUntil.Format(time.RFC3339),
			"max_uses":       100,
			"uses_per_user":  1,
			"is_active":      true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)

		validateReq := map[string]interface{}{
			"code":        "PASTPROMO",
			"ride_amount": 50.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Equal(t, "This promo code has expired", validateResp.Data.Message)
	})

	t.Run("not yet valid promo code is rejected", func(t *testing.T) {
		// Create promo code that starts in the future
		validFrom := time.Now().Add(7 * 24 * time.Hour)
		validUntil := time.Now().Add(30 * 24 * time.Hour)

		createPromoReq := map[string]interface{}{
			"code":           "FUTUREPROMO",
			"description":    "Not valid yet",
			"discount_type":  "percentage",
			"discount_value": 30.0,
			"valid_from":     validFrom.Format(time.RFC3339),
			"valid_until":    validUntil.Format(time.RFC3339),
			"max_uses":       100,
			"uses_per_user":  1,
			"is_active":      true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)

		validateReq := map[string]interface{}{
			"code":        "FUTUREPROMO",
			"ride_amount": 50.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Equal(t, "This promo code is not yet valid", validateResp.Data.Message)
	})

	t.Run("deactivated promo code is rejected", func(t *testing.T) {
		now := time.Now()
		validUntil := now.Add(30 * 24 * time.Hour)

		createPromoReq := map[string]interface{}{
			"code":           "DEACTIVATED",
			"description":    "Will be deactivated",
			"discount_type":  "percentage",
			"discount_value": 15.0,
			"valid_from":     now.Format(time.RFC3339),
			"valid_until":    validUntil.Format(time.RFC3339),
			"max_uses":       100,
			"uses_per_user":  1,
			"is_active":      true,
		}

		createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
		require.True(t, createResp.Success)

		// Deactivate the promo code directly in database
		_, err := dbPool.Exec(context.Background(), `UPDATE promo_codes SET is_active = false WHERE code = 'DEACTIVATED'`)
		require.NoError(t, err)

		validateReq := map[string]interface{}{
			"code":        "DEACTIVATED",
			"ride_amount": 50.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Contains(t, validateResp.Data.Message, "no longer active")
	})
}

// TestPromoFlow_ReferralCodeFlow tests the complete referral code flow
func TestPromoFlow_ReferralCodeFlow(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	t.Run("generate and apply referral code", func(t *testing.T) {
		referrer := registerAndLogin(t, models.RoleRider)
		newUser := registerAndLogin(t, models.RoleRider)

		// Step 1: Referrer gets their referral code
		referralResp := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(referrer.Token))
		require.True(t, referralResp.Success)
		require.NotEmpty(t, referralResp.Data.Code)
		require.Equal(t, referrer.User.ID.String(), referralResp.Data.UserID)
		require.Equal(t, 0, referralResp.Data.TotalReferrals)

		referralCode := referralResp.Data.Code

		// Step 2: New user applies referral code
		applyReq := map[string]interface{}{
			"referral_code": referralCode,
		}

		applyResp := doRequest[applyReferralResponse](t, promosServiceKey, http.MethodPost, "/api/v1/referrals/apply", applyReq, authHeaders(newUser.Token))
		require.True(t, applyResp.Success)
		require.Contains(t, applyResp.Data.Message, "successfully")
		require.Equal(t, 10.0, applyResp.Data.Bonus)

		// Step 3: Verify referral relationship in database
		var referralCount int
		err := dbPool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM referrals WHERE referrer_id = $1 AND referred_id = $2",
			referrer.User.ID, newUser.User.ID).Scan(&referralCount)
		require.NoError(t, err)
		require.Equal(t, 1, referralCount)

		// Step 4: Verify referral bonus amounts
		var referrerBonus, referredBonus float64
		err = dbPool.QueryRow(context.Background(),
			"SELECT referrer_bonus, referred_bonus FROM referrals WHERE referrer_id = $1 AND referred_id = $2",
			referrer.User.ID, newUser.User.ID).Scan(&referrerBonus, &referredBonus)
		require.NoError(t, err)
		require.Equal(t, 10.0, referrerBonus)
		require.Equal(t, 10.0, referredBonus)
	})

	t.Run("referral code is idempotent", func(t *testing.T) {
		rider := registerAndLogin(t, models.RoleRider)

		// Request referral code twice
		resp1 := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(rider.Token))
		require.True(t, resp1.Success)

		resp2 := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(rider.Token))
		require.True(t, resp2.Success)

		// Should return same code
		require.Equal(t, resp1.Data.Code, resp2.Data.Code)
		require.Equal(t, resp1.Data.ID, resp2.Data.ID)
	})

	t.Run("cannot use own referral code", func(t *testing.T) {
		rider := registerAndLogin(t, models.RoleRider)

		// Get own referral code
		referralResp := doRequest[referralCodeResponse](t, promosServiceKey, http.MethodGet, "/api/v1/referrals/my-code", nil, authHeaders(rider.Token))
		require.True(t, referralResp.Success)

		// Try to apply own referral code
		applyReq := map[string]interface{}{
			"referral_code": referralResp.Data.Code,
		}

		resp := doRawRequest(t, promosServiceKey, http.MethodPost, "/api/v1/referrals/apply", applyReq, authHeaders(rider.Token))
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid referral code is rejected", func(t *testing.T) {
		rider := registerAndLogin(t, models.RoleRider)

		applyReq := map[string]interface{}{
			"referral_code": "INVALIDCODE123",
		}

		resp := doRawRequest(t, promosServiceKey, http.MethodPost, "/api/v1/referrals/apply", applyReq, authHeaders(rider.Token))
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestPromoFlow_InvalidPromoCode tests invalid promo code handling
func TestPromoFlow_InvalidPromoCode(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	t.Run("non-existent promo code is rejected", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "DOESNOTEXIST",
			"ride_amount": 50.0,
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.False(t, validateResp.Data.Valid)
		require.Equal(t, "Invalid promo code", validateResp.Data.Message)
	})
}

// TestPromoFlow_FixedAmountPromoExceedsRideAmount tests when fixed discount exceeds ride amount
func TestPromoFlow_FixedAmountPromoExceedsRideAmount(t *testing.T) {
	truncatePromoTables(t)

	if _, ok := services[promosServiceKey]; !ok {
		services[promosServiceKey] = startPromosService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Create fixed amount promo that could exceed ride amount
	now := time.Now()
	validUntil := now.Add(30 * 24 * time.Hour)

	createPromoReq := map[string]interface{}{
		"code":           "BIGFLAT",
		"description":    "$50 off",
		"discount_type":  "fixed_amount",
		"discount_value": 50.0,
		"valid_from":     now.Format(time.RFC3339),
		"valid_until":    validUntil.Format(time.RFC3339),
		"max_uses":       100,
		"uses_per_user":  1,
		"is_active":      true,
	}

	createResp := doRequest[map[string]interface{}](t, promosServiceKey, http.MethodPost, "/api/v1/admin/promos", createPromoReq, authHeaders(admin.Token))
	require.True(t, createResp.Success)

	t.Run("discount is capped at ride amount", func(t *testing.T) {
		validateReq := map[string]interface{}{
			"code":        "BIGFLAT",
			"ride_amount": 30.0, // Less than $50 discount
		}

		validateResp := doRequest[promoValidation](t, promosServiceKey, http.MethodPost, "/api/v1/promos/validate", validateReq, authHeaders(rider.Token))
		require.True(t, validateResp.Success)
		require.True(t, validateResp.Data.Valid)
		// Discount should be capped at ride amount ($30), not $50
		require.InEpsilon(t, 30.0, validateResp.Data.DiscountAmount, 1e-6)
		require.InEpsilon(t, 0.0, validateResp.Data.FinalAmount, 1e-6)
	})
}

// truncatePromoTables truncates promo-related tables along with standard tables
func truncatePromoTables(t *testing.T) {
	t.Helper()
	tables := []string{
		"referrals",
		"referral_codes",
		"promo_code_uses",
		"promo_codes",
		"wallet_transactions",
		"payments",
		"rides",
		"drivers",
		"wallets",
		"users",
	}

	for _, table := range tables {
		_, err := dbPool.Exec(context.Background(), "TRUNCATE TABLE "+table+" CASCADE")
		require.NoError(t, err, "failed to truncate %s", table)
	}
}
