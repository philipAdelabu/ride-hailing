package earnings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommissionCalculation(t *testing.T) {
	tests := []struct {
		name               string
		grossAmount        float64
		commissionRate     float64
		expectedCommission float64
		expectedNet        float64
	}{
		{
			name:               "standard $100 ride with default 20% commission",
			grossAmount:        100.0,
			commissionRate:     defaultCommissionRate,
			expectedCommission: 20.0,
			expectedNet:        80.0,
		},
		{
			name:               "standard $50 ride",
			grossAmount:        50.0,
			commissionRate:     defaultCommissionRate,
			expectedCommission: 10.0,
			expectedNet:        40.0,
		},
		{
			name:               "$25.50 ride",
			grossAmount:        25.50,
			commissionRate:     defaultCommissionRate,
			expectedCommission: 5.10,
			expectedNet:        20.40,
		},
		{
			name:               "custom 15% commission",
			grossAmount:        100.0,
			commissionRate:     0.15,
			expectedCommission: 15.0,
			expectedNet:        85.0,
		},
		{
			name:               "zero fare",
			grossAmount:        0.0,
			commissionRate:     defaultCommissionRate,
			expectedCommission: 0.0,
			expectedNet:        0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commission := tt.grossAmount * tt.commissionRate
			net := tt.grossAmount - commission

			assert.InDelta(t, tt.expectedCommission, commission, 0.01)
			assert.InDelta(t, tt.expectedNet, net, 0.01)
		})
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 0.20, defaultCommissionRate)
	assert.Equal(t, 5.0, minPayoutAmount)
	assert.Equal(t, "USD", defaultCurrency)
}

func TestPeriodToTimeRange(t *testing.T) {
	svc := &Service{}

	validPeriods := []string{
		"today", "yesterday", "this_week", "last_week", "this_month", "last_month",
	}

	for _, period := range validPeriods {
		t.Run(period, func(t *testing.T) {
			from, to, err := svc.periodToTimeRange(period)
			assert.NoError(t, err)
			assert.True(t, from.Before(to), "from (%v) should be before to (%v)", from, to)
		})
	}
}

func TestPeriodToTimeRange_InvalidPeriod(t *testing.T) {
	svc := &Service{}

	_, _, err := svc.periodToTimeRange("invalid_period")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "period must be")
}

func TestExpectedProgressFraction(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name   string
		period string
	}{
		{"daily", "daily"},
		{"weekly", "weekly"},
		{"monthly", "monthly"},
		{"unknown defaults to 0.5", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fraction := svc.expectedProgressFraction(tt.period)
			assert.GreaterOrEqual(t, fraction, 0.0)
			assert.LessOrEqual(t, fraction, 1.0)

			if tt.period == "unknown" {
				assert.Equal(t, 0.5, fraction)
			}
		})
	}
}

func TestGeneratePayoutReference(t *testing.T) {
	ref := generatePayoutReference()

	assert.NotEmpty(t, ref)
	assert.True(t, len(ref) > 4, "reference should be longer than prefix")
	assert.Equal(t, "PAY-", ref[:4], "reference should start with PAY-")

	// Verify uniqueness
	ref2 := generatePayoutReference()
	assert.NotEqual(t, ref, ref2, "references should be unique")
}

func TestEarningTypes(t *testing.T) {
	types := []EarningType{
		EarningTypeRideFare, EarningTypeTip, EarningTypeBonus,
		EarningTypeSurge, EarningTypePromo, EarningTypeReferral,
		EarningTypeDelivery, EarningTypeWaitTime, EarningTypeAdjustment,
		EarningTypeCancellation,
	}

	assert.Len(t, types, 10, "should have 10 earning types")

	seen := make(map[EarningType]bool)
	for _, et := range types {
		assert.NotEmpty(t, string(et))
		assert.False(t, seen[et], "duplicate earning type: %s", et)
		seen[et] = true
	}
}
