package cancellation

import (
	"testing"

	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetCancellationReasons_Rider(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(false)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 8)

	// Verify first and last reason codes
	assert.Equal(t, ReasonRiderChangedMind, resp.Reasons[0].Code)
	assert.Equal(t, ReasonRiderOther, resp.Reasons[7].Code)

	// All reasons must have labels and descriptions
	for _, reason := range resp.Reasons {
		assert.NotEmpty(t, reason.Code)
		assert.NotEmpty(t, reason.Label)
		assert.NotEmpty(t, reason.Description)
	}
}

func TestGetCancellationReasons_Driver(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(true)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 7)

	assert.Equal(t, ReasonDriverRiderNoShow, resp.Reasons[0].Code)
	assert.Equal(t, ReasonDriverOther, resp.Reasons[6].Code)

	for _, reason := range resp.Reasons {
		assert.NotEmpty(t, reason.Code)
		assert.NotEmpty(t, reason.Label)
		assert.NotEmpty(t, reason.Description)
	}
}

func TestCalculateFee_DriverCancellation_AlwaysFree(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	// Driver cancellations should always be free regardless of timing
	tests := []struct {
		name                string
		minutesSinceRequest float64
	}{
		{"immediately", 0},
		{"after 1 minute", 1},
		{"after 5 minutes", 5},
		{"after 30 minutes", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateFee(nil, [16]byte{}, CancelledByDriver, tt.minutesSinceRequest, nil, policy)

			assert.Equal(t, 0.0, result.FeeAmount)
			assert.True(t, result.FeeWaived)
			assert.NotNil(t, result.WaiverReason)
			assert.Equal(t, WaiverDriverFault, *result.WaiverReason)
			assert.Contains(t, result.Explanation, "Driver cancellations")
		})
	}
}

func TestCalculateFee_FreeCancellationWindow(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy // FreeCancelWindowMinutes = 2

	// Ride needed for cases past the free window (accesses ride.Status)
	ride := &models.Ride{Status: models.RideStatusRequested}

	tests := []struct {
		name                string
		minutesSinceRequest float64
		expectFree          bool
	}{
		{"within window - 0 minutes", 0, true},
		{"within window - 1 minute", 1, true},
		{"within window - 1.9 minutes", 1.9, true},
		{"at boundary - 2 minutes (unaccepted ride)", 2, true},
		{"after window - 3 minutes (unaccepted ride)", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateFee(nil, [16]byte{}, CancelledByRider, tt.minutesSinceRequest, ride, policy)

			if tt.expectFree {
				assert.True(t, result.FeeWaived)
				assert.NotNil(t, result.WaiverReason)
				assert.Equal(t, WaiverFreeCancellationWindow, *result.WaiverReason)
			}
		})
	}
}

func TestCalculateFee_UnacceptedRide_AlwaysFree(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	ride := &models.Ride{Status: models.RideStatusRequested}

	// Even after free window, unaccepted rides should be free
	result := svc.calculateFee(nil, [16]byte{}, CancelledByRider, 10.0, ride, policy)
	assert.True(t, result.FeeWaived)
	assert.Contains(t, result.Explanation, "hasn't been accepted")
}

func TestDefaultPolicy_Values(t *testing.T) {
	assert.Equal(t, 2, defaultPolicy.FreeCancelWindowMinutes)
	assert.Equal(t, 3, defaultPolicy.MaxFreeCancelsPerDay)
	assert.Equal(t, 10, defaultPolicy.MaxFreeCancelsPerWeek)
	assert.Equal(t, 5, defaultPolicy.DriverNoShowMinutes)
	assert.Equal(t, 5, defaultPolicy.RiderNoShowMinutes)
	assert.Equal(t, 20, defaultPolicy.DriverPenaltyThreshold)
	assert.Equal(t, 30, defaultPolicy.RiderPenaltyThreshold)
}

func TestWaivePtr(t *testing.T) {
	reason := WaiverDriverFault
	ptr := waivePtr(reason)
	assert.NotNil(t, ptr)
	assert.Equal(t, reason, *ptr)
}
