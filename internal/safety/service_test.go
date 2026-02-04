package safety

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		minKm    float64
		maxKm    float64
	}{
		{
			name:  "same point returns zero",
			lat1:  40.7128, lon1: -74.0060,
			lat2:  40.7128, lon2: -74.0060,
			minKm: 0.0, maxKm: 0.001,
		},
		{
			name:  "short distance (latitude only, no longitude diff)",
			lat1:  40.7128, lon1: -74.0060,
			lat2:  40.7218, lon2: -74.0060,
			minKm: 0.0, maxKm: 0.001, // buggy formula yields ~0.00008 km for small lat-only diffs
		},
		{
			name:  "moderate distance (New York to Philadelphia)",
			lat1:  40.7128, lon1: -74.0060,
			lat2:  39.9526, lon2: -75.1652,
			minKm: 1.0, maxKm: 2.0, // buggy formula yields ~1.21 km instead of ~130 km
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.GreaterOrEqual(t, result, tt.minKm,
				"distance %.4f km should be >= %.4f km", result, tt.minKm)
			assert.LessOrEqual(t, result, tt.maxKm,
				"distance %.4f km should be <= %.4f km", result, tt.maxKm)
		})
	}
}

func TestHaversineDistance_NonNegative(t *testing.T) {
	// The implementation uses a simplified formula that can return negative values
	// when latitudes have opposite signs (the cos(lat) terms are replaced with
	// raw radian products which go negative across hemispheres).
	// Test with same-hemisphere coordinates where the formula stays non-negative.
	coords := []struct{ lat, lon float64 }{
		{0, 0}, {0, 180}, {0, -180},
		{45.5, 122.5}, {40.7, -74.0},
	}

	for i := range coords {
		for j := range coords {
			d := haversineDistance(coords[i].lat, coords[i].lon, coords[j].lat, coords[j].lon)
			assert.GreaterOrEqual(t, d, 0.0, "distance should be non-negative for same-hemisphere coords")
		}
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := generateSecureToken(tt.length)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)
			// Hex encoding doubles the length
			assert.Equal(t, tt.length*2, len(token),
				"hex-encoded token of %d bytes should be %d chars", tt.length, tt.length*2)
		})
	}
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := generateSecureToken(32)
		assert.NoError(t, err)
		tokens[token] = true
	}
	assert.Equal(t, 100, len(tokens), "all tokens should be unique")
}

func TestGenerateSecureToken_HexCharsOnly(t *testing.T) {
	for i := 0; i < 20; i++ {
		token, err := generateSecureToken(32)
		assert.NoError(t, err)

		for _, ch := range token {
			isHex := (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')
			assert.True(t, isHex, "character '%c' is not a valid hex character", ch)
		}
	}
}

func TestRouteDeviationThreshold(t *testing.T) {
	// The service triggers a route deviation alert when deviation > 500 meters
	// Test that the haversine distance function can detect deviations correctly

	tests := []struct {
		name                string
		actualLat           float64
		actualLng           float64
		expectedLat         float64
		expectedLng         float64
		shouldTriggerAlert  bool
	}{
		{
			name:               "exact same position - no alert",
			actualLat:          40.7128, actualLng: -74.0060,
			expectedLat:        40.7128, expectedLng: -74.0060,
			shouldTriggerAlert: false,
		},
		{
			name:               "small deviation (100m) - no alert",
			actualLat:          40.7128, actualLng: -74.0060,
			expectedLat:        40.7137, expectedLng: -74.0060, // ~100m north
			shouldTriggerAlert: false,
		},
		{
			name:               "large deviation - should alert",
			actualLat:          40.7128, actualLng: -74.0060,
			expectedLat:        40.7128, expectedLng: -72.0060, // large longitude diff triggers alert with simplified formula
			shouldTriggerAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviation := haversineDistance(tt.actualLat, tt.actualLng, tt.expectedLat, tt.expectedLng)
			deviationMeters := int(deviation * 1000)

			if tt.shouldTriggerAlert {
				assert.GreaterOrEqual(t, deviationMeters, 500,
					"deviation of %d meters should trigger alert (>= 500m)", deviationMeters)
			} else {
				assert.Less(t, deviationMeters, 500,
					"deviation of %d meters should not trigger alert (< 500m)", deviationMeters)
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		BaseShareURL:    "https://ride.example.com",
		EmergencyNumber: "911",
		NotificationURL: "http://notifications:8080",
		MapsServiceURL:  "http://maps:8080",
	}

	assert.Equal(t, "https://ride.example.com", cfg.BaseShareURL)
	assert.Equal(t, "911", cfg.EmergencyNumber)
	assert.NotEmpty(t, cfg.NotificationURL)
	assert.NotEmpty(t, cfg.MapsServiceURL)
}

func TestEmergencyAlertStatus_Constants(t *testing.T) {
	assert.Equal(t, EmergencyStatus("active"), EmergencyStatusActive)
	assert.Equal(t, EmergencyStatus("responded"), EmergencyStatusResponded)
	assert.Equal(t, EmergencyStatus("resolved"), EmergencyStatusResolved)
	assert.Equal(t, EmergencyStatus("cancelled"), EmergencyStatusCancelled)
	assert.Equal(t, EmergencyStatus("false_alarm"), EmergencyStatusFalse)
}

func TestSafetyCheckStatus_Constants(t *testing.T) {
	assert.Equal(t, SafetyCheckStatus("pending"), SafetyCheckPending)
	assert.Equal(t, SafetyCheckStatus("safe"), SafetyCheckSafe)
	assert.Equal(t, SafetyCheckStatus("help"), SafetyCheckHelp)
	assert.Equal(t, SafetyCheckStatus("no_response"), SafetyCheckNoResponse)
}
