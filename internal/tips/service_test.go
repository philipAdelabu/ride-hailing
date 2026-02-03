package tips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTipPresets(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name       string
		fareAmount float64
		expected   []float64 // expected amounts for 10%, 15%, 20%, 25%
	}{
		{
			name:       "standard fare $20",
			fareAmount: 20.0,
			expected:   []float64{2.0, 3.0, 4.0, 5.0},
		},
		{
			name:       "high fare $100",
			fareAmount: 100.0,
			expected:   []float64{10.0, 15.0, 20.0, 25.0},
		},
		{
			name:       "small fare $5",
			fareAmount: 5.0,
			expected:   []float64{1.0, 1.0, 1.0, 1.25},
		},
		{
			name:       "very small fare rounds up to min",
			fareAmount: 2.0,
			expected:   []float64{1.0, 1.0, 1.0, 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := svc.GetTipPresets(tt.fareAmount)

			assert.NotNil(t, resp)
			assert.Len(t, resp.Presets, 4)
			assert.Equal(t, maxTipAmount, resp.MaxTip)
			assert.Equal(t, defaultCurrency, resp.Currency)

			for i, preset := range resp.Presets {
				assert.InDelta(t, tt.expected[i], preset.Amount, 0.01, "preset %d amount mismatch", i)
				assert.True(t, preset.IsPercent)
				// Min tip enforcement
				assert.GreaterOrEqual(t, preset.Amount, minTipAmount)
			}
		})
	}
}

func TestGetTipPresets_Labels(t *testing.T) {
	svc := &Service{}
	resp := svc.GetTipPresets(50.0)

	expectedLabels := []string{"10%", "15%", "20%", "25%"}
	for i, preset := range resp.Presets {
		assert.Equal(t, expectedLabels[i], preset.Label)
	}

	// 15% should be the default
	assert.False(t, resp.Presets[0].IsDefault)
	assert.True(t, resp.Presets[1].IsDefault)
	assert.False(t, resp.Presets[2].IsDefault)
	assert.False(t, resp.Presets[3].IsDefault)
}

func TestTipValidation_Constants(t *testing.T) {
	assert.Equal(t, 1.0, minTipAmount)
	assert.Equal(t, 200.0, maxTipAmount)
	assert.Equal(t, 72, tipWindowHours)
	assert.Equal(t, "USD", defaultCurrency)
}

func TestGetTipPresets_ZeroFare(t *testing.T) {
	svc := &Service{}
	resp := svc.GetTipPresets(0.0)

	// All presets should be at minimum tip
	for _, preset := range resp.Presets {
		assert.GreaterOrEqual(t, preset.Amount, minTipAmount)
	}
}

func TestPeriodToTimeRange(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name   string
		period string
	}{
		{"today", "today"},
		{"this_week", "this_week"},
		{"this_month", "this_month"},
		{"all_time", "all_time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := svc.periodToTimeRange(tt.period)
			assert.True(t, from.Before(to), "from should be before to")
		})
	}
}
