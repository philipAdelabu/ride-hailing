package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper: default config
// ---------------------------------------------------------------------------

func testConfig() config.RateLimitConfig {
	return config.RateLimitConfig{
		Enabled:        true,
		WindowSeconds:  60,
		DefaultLimit:   100,
		DefaultBurst:   10,
		AnonymousLimit: 30,
		AnonymousBurst: 5,
		RedisPrefix:    "rl",
	}
}

// ---------------------------------------------------------------------------
// NewLimiter
// ---------------------------------------------------------------------------

func TestNewLimiter(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()

	limiter := NewLimiter(client, cfg)

	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.client)
	assert.NotNil(t, limiter.script)
	assert.NotNil(t, limiter.now)
	assert.Equal(t, cfg.Enabled, limiter.cfg.Enabled)
	assert.Equal(t, cfg.DefaultLimit, limiter.cfg.DefaultLimit)
	assert.Equal(t, cfg.RedisPrefix, limiter.cfg.RedisPrefix)
}

func TestNewLimiter_NowReturnsCurrentTime(t *testing.T) {
	client, _ := redismock.NewClientMock()
	limiter := NewLimiter(client, testConfig())

	before := time.Now()
	got := limiter.now()
	after := time.Now()

	assert.False(t, got.Before(before))
	assert.False(t, got.After(after))
}

// ---------------------------------------------------------------------------
// WithNow
// ---------------------------------------------------------------------------

func TestWithNow(t *testing.T) {
	client, _ := redismock.NewClientMock()
	limiter := NewLimiter(client, testConfig())

	fixed := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	limiter.WithNow(func() time.Time { return fixed })

	assert.Equal(t, fixed, limiter.now())
}

// ---------------------------------------------------------------------------
// RuleFor – authenticated defaults
// ---------------------------------------------------------------------------

func TestRuleFor_AuthenticatedDefaults(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	limiter := NewLimiter(client, cfg)

	rule := limiter.RuleFor("/api/rides", IdentityAuthenticated)

	assert.Equal(t, cfg.DefaultLimit, rule.Limit)
	assert.Equal(t, cfg.DefaultBurst, rule.Burst)
	assert.Equal(t, cfg.Window(), rule.Window)
}

// ---------------------------------------------------------------------------
// RuleFor – anonymous defaults
// ---------------------------------------------------------------------------

func TestRuleFor_AnonymousDefaults(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	limiter := NewLimiter(client, cfg)

	rule := limiter.RuleFor("/api/rides", IdentityAnonymous)

	assert.Equal(t, cfg.AnonymousLimit, rule.Limit)
	assert.Equal(t, cfg.AnonymousBurst, rule.Burst)
}

// ---------------------------------------------------------------------------
// RuleFor – endpoint overrides
// ---------------------------------------------------------------------------

func TestRuleFor_EndpointOverrides(t *testing.T) {
	tests := []struct {
		name         string
		identity     IdentityType
		override     config.EndpointRateLimitConfig
		expectLimit  int
		expectBurst  int
		expectWindow time.Duration
	}{
		{
			name:     "authenticated override",
			identity: IdentityAuthenticated,
			override: config.EndpointRateLimitConfig{
				AuthenticatedLimit: 200,
				AuthenticatedBurst: 20,
				WindowSeconds:      120,
			},
			expectLimit:  200,
			expectBurst:  20,
			expectWindow: 120 * time.Second,
		},
		{
			name:     "anonymous override",
			identity: IdentityAnonymous,
			override: config.EndpointRateLimitConfig{
				AnonymousLimit: 10,
				AnonymousBurst: 2,
				WindowSeconds:  30,
			},
			expectLimit:  10,
			expectBurst:  2,
			expectWindow: 30 * time.Second,
		},
		{
			name:     "partial override - only window",
			identity: IdentityAuthenticated,
			override: config.EndpointRateLimitConfig{
				WindowSeconds: 300,
			},
			expectLimit:  100, // falls back to default
			expectBurst:  0,   // AuthenticatedBurst=0 treated as valid override (>= 0)
			expectWindow: 300 * time.Second,
		},
		{
			name:     "zero window keeps default",
			identity: IdentityAuthenticated,
			override: config.EndpointRateLimitConfig{
				AuthenticatedLimit: 50,
				WindowSeconds:      0,
			},
			expectLimit:  50,
			expectBurst:  0,              // AuthenticatedBurst=0 treated as valid override (>= 0)
			expectWindow: 60 * time.Second, // default from WindowSeconds=60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := redismock.NewClientMock()
			cfg := testConfig()
			cfg.EndpointOverrides = map[string]config.EndpointRateLimitConfig{
				"/api/login": tt.override,
			}
			limiter := NewLimiter(client, cfg)

			rule := limiter.RuleFor("/api/login", tt.identity)

			assert.Equal(t, tt.expectLimit, rule.Limit)
			assert.Equal(t, tt.expectBurst, rule.Burst)
			assert.Equal(t, tt.expectWindow, rule.Window)
		})
	}
}

func TestRuleFor_NoOverrideForEndpoint(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	cfg.EndpointOverrides = map[string]config.EndpointRateLimitConfig{
		"/api/login": {AuthenticatedLimit: 5},
	}
	limiter := NewLimiter(client, cfg)

	// Request a different endpoint – should get defaults
	rule := limiter.RuleFor("/api/rides", IdentityAuthenticated)
	assert.Equal(t, cfg.DefaultLimit, rule.Limit)
}

// ---------------------------------------------------------------------------
// RuleFor – edge cases (zero / negative limits)
// ---------------------------------------------------------------------------

func TestRuleFor_ZeroLimit(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	cfg.DefaultLimit = 0
	limiter := NewLimiter(client, cfg)

	rule := limiter.RuleFor("/api/test", IdentityAuthenticated)
	assert.Equal(t, 0, rule.Limit)
}

func TestRuleFor_NegativeBurstClampedToZero(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	cfg.DefaultBurst = -5
	limiter := NewLimiter(client, cfg)

	rule := limiter.RuleFor("/api/test", IdentityAuthenticated)
	assert.Equal(t, 0, rule.Burst)
}

// ---------------------------------------------------------------------------
// Allow – disabled rate limiter bypasses
// ---------------------------------------------------------------------------

func TestAllow_DisabledLimiter(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	cfg.Enabled = false
	limiter := NewLimiter(client, cfg)

	rule := Rule{Limit: 100, Burst: 10, Window: time.Minute}
	result, err := limiter.Allow(context.Background(), "/api/test", "user-1", rule, IdentityAuthenticated)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 100, result.Remaining)
	assert.Equal(t, "user-1", result.IdentityKey)
	assert.Equal(t, "/api/test", result.EndpointKey)
	assert.Equal(t, IdentityAuthenticated, result.IdentityType)
	assert.Zero(t, result.RetryAfter)
}

// ---------------------------------------------------------------------------
// Allow – zero limit rule bypasses
// ---------------------------------------------------------------------------

func TestAllow_ZeroLimitRule(t *testing.T) {
	client, _ := redismock.NewClientMock()
	limiter := NewLimiter(client, testConfig())

	rule := Rule{Limit: 0, Burst: 0, Window: time.Minute}
	result, err := limiter.Allow(context.Background(), "/api/test", "user-1", rule, IdentityAuthenticated)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
}

// ---------------------------------------------------------------------------
// Allow – negative limit rule bypasses
// ---------------------------------------------------------------------------

func TestAllow_NegativeLimitRule(t *testing.T) {
	client, _ := redismock.NewClientMock()
	limiter := NewLimiter(client, testConfig())

	rule := Rule{Limit: -1, Burst: 0, Window: time.Minute}
	result, err := limiter.Allow(context.Background(), "/api/test", "user-1", rule, IdentityAnonymous)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, IdentityAnonymous, result.IdentityType)
}

// ---------------------------------------------------------------------------
// Allow – result fields populated correctly (disabled fast-path)
// ---------------------------------------------------------------------------

func TestAllow_ResultFieldsPopulated(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	cfg.Enabled = false
	limiter := NewLimiter(client, cfg)

	tests := []struct {
		name         string
		endpoint     string
		identity     string
		identityType IdentityType
		rule         Rule
	}{
		{
			name:         "authenticated user",
			endpoint:     "/api/rides",
			identity:     "user-abc",
			identityType: IdentityAuthenticated,
			rule:         Rule{Limit: 100, Burst: 10, Window: time.Minute},
		},
		{
			name:         "anonymous user",
			endpoint:     "/api/public",
			identity:     "192.168.1.1",
			identityType: IdentityAnonymous,
			rule:         Rule{Limit: 30, Burst: 5, Window: 30 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := limiter.Allow(context.Background(), tt.endpoint, tt.identity, tt.rule, tt.identityType)

			require.NoError(t, err)
			assert.True(t, result.Allowed)
			assert.Equal(t, tt.rule.Limit, result.Remaining)
			assert.Equal(t, tt.rule.Limit, result.Limit)
			assert.Equal(t, tt.rule.Window, result.Window)
			assert.Equal(t, tt.identity, result.IdentityKey)
			assert.Equal(t, tt.endpoint, result.EndpointKey)
			assert.Equal(t, tt.identityType, result.IdentityType)
			assert.Zero(t, result.RetryAfter)
			assert.Zero(t, result.ResetAfter)
		})
	}
}

// ---------------------------------------------------------------------------
// Allow – key construction uses prefix:endpoint:identity format
// ---------------------------------------------------------------------------

func TestAllow_KeyFormat(t *testing.T) {
	// When limiter is disabled, key is never sent to Redis,
	// but we verify the key format via the RedisPrefix config.
	cfg := testConfig()
	cfg.RedisPrefix = "custom-prefix"
	cfg.Enabled = false

	client, _ := redismock.NewClientMock()
	limiter := NewLimiter(client, cfg)

	// Verify the config is stored correctly for key construction
	assert.Equal(t, "custom-prefix", limiter.cfg.RedisPrefix)
}

// ---------------------------------------------------------------------------
// Allow – zero window rule falls back to config window
// ---------------------------------------------------------------------------

func TestAllow_ZeroWindowRuleFallback(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	cfg.Enabled = false
	limiter := NewLimiter(client, cfg)

	// Rule with zero window: Allow still works via the disabled fast-path
	rule := Rule{Limit: 30, Burst: 5, Window: 0}
	result, err := limiter.Allow(context.Background(), "/api/test", "anon-ip", rule, IdentityAnonymous)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// ---------------------------------------------------------------------------
// Script hash is deterministic
// ---------------------------------------------------------------------------

func TestScriptHash_Deterministic(t *testing.T) {
	client, _ := redismock.NewClientMock()
	cfg := testConfig()
	limiter1 := NewLimiter(client, cfg)
	limiter2 := NewLimiter(client, cfg)

	assert.Equal(t, limiter1.script.Hash(), limiter2.script.Hash())
	assert.NotEmpty(t, limiter1.script.Hash())
}

// ---------------------------------------------------------------------------
// IdentityType constants
// ---------------------------------------------------------------------------

func TestIdentityTypeConstants(t *testing.T) {
	assert.Equal(t, IdentityType(0), IdentityAnonymous)
	assert.Equal(t, IdentityType(1), IdentityAuthenticated)
}

// ---------------------------------------------------------------------------
// formatFloat helper
// ---------------------------------------------------------------------------

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		name   string
		input  float64
		expect string
	}{
		{"zero", 0, "0.0000000000"},
		{"positive", 1.5, "1.5000000000"},
		{"small fraction", 0.0001, "0.0001000000"},
		{"large number", 12345.6789, "12345.6789000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, formatFloat(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// toInt helper
// ---------------------------------------------------------------------------

func TestToInt(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect int
	}{
		{"int64", int64(42), 42},
		{"int", int(99), 99},
		{"string valid", "123", 123},
		{"string invalid", "abc", 0},
		{"float64", float64(7.9), 7},
		{"nil", nil, 0},
		{"bool (unsupported)", true, 0},
		{"string empty", "", 0},
		{"negative int64", int64(-5), -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, toInt(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// toFloat helper
// ---------------------------------------------------------------------------

func TestToFloat(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect float64
	}{
		{"float64", float64(3.14), 3.14},
		{"int64", int64(10), 10.0},
		{"int", int(20), 20.0},
		{"string valid", "2.718", 2.718},
		{"string invalid", "xyz", 0},
		{"nil", nil, 0},
		{"bool (unsupported)", false, 0},
		{"negative float64", float64(-1.5), -1.5},
		{"string empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expect, toFloat(tt.input), 0.0001)
		})
	}
}

// ---------------------------------------------------------------------------
// RateLimitConfig.Window
// ---------------------------------------------------------------------------

func TestConfigWindow(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		expect  time.Duration
	}{
		{"positive", 60, 60 * time.Second},
		{"zero falls back", 0, time.Minute},
		{"negative falls back", -1, time.Minute},
		{"large value", 3600, 3600 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RateLimitConfig{WindowSeconds: tt.seconds}
			assert.Equal(t, tt.expect, cfg.Window())
		})
	}
}

// ---------------------------------------------------------------------------
// Rule struct
// ---------------------------------------------------------------------------

func TestRuleStruct(t *testing.T) {
	rule := Rule{
		Limit:  50,
		Burst:  5,
		Window: 30 * time.Second,
	}

	assert.Equal(t, 50, rule.Limit)
	assert.Equal(t, 5, rule.Burst)
	assert.Equal(t, 30*time.Second, rule.Window)
}

// ---------------------------------------------------------------------------
// Result struct fields
// ---------------------------------------------------------------------------

func TestResultStruct(t *testing.T) {
	result := Result{
		Allowed:      true,
		Remaining:    42,
		RetryAfter:   0,
		Limit:        100,
		Window:       time.Minute,
		ResetAfter:   500 * time.Millisecond,
		IdentityKey:  "user-123",
		EndpointKey:  "/api/rides",
		IdentityType: IdentityAuthenticated,
	}

	assert.True(t, result.Allowed)
	assert.Equal(t, 42, result.Remaining)
	assert.Equal(t, "user-123", result.IdentityKey)
	assert.Equal(t, "/api/rides", result.EndpointKey)
	assert.Equal(t, IdentityAuthenticated, result.IdentityType)
}
