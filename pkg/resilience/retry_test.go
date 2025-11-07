package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testError          = errors.New("test error")
	retryableError     = errors.New("retryable error")
	nonRetryableError  = errors.New("non-retryable error")
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return "success", nil
	}

	result, err := Retry(ctx, config, operation)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, attemptCount, "should only attempt once on success")
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialBackoff = 10 * time.Millisecond
	config.MaxBackoff = 50 * time.Millisecond
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, testError
		}
		return "success", nil
	}

	start := time.Now()
	result, err := Retry(ctx, config, operation)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, attemptCount, "should attempt 3 times")
	// Should have waited at least for 2 backoffs
	assert.Greater(t, duration, 10*time.Millisecond, "should have backed off")
}

func TestRetry_FailureAfterMaxAttempts(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialBackoff = 1 * time.Millisecond
	config.MaxAttempts = 3
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, testError
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, testError, err)
	assert.Equal(t, 3, attemptCount, "should attempt max times")
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()
	config.InitialBackoff = 50 * time.Millisecond
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount == 2 {
			// Cancel context during second attempt
			cancel()
		}
		return nil, testError
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, context.Canceled) || err == testError)
	assert.LessOrEqual(t, attemptCount, 2, "should stop after context cancellation")
}

func TestRetry_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := DefaultRetryConfig()
	config.InitialBackoff = 100 * time.Millisecond
	config.MaxAttempts = 5
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, testError
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	// Should timeout before completing all retries
	assert.Less(t, attemptCount, 5, "should timeout before all attempts")
}

func TestRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.RetryableErrors = []error{retryableError}
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, nonRetryableError
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, nonRetryableError, err)
	assert.Equal(t, 1, attemptCount, "should not retry non-retryable error")
}

func TestRetry_RetryableErrorList(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialBackoff = 1 * time.Millisecond
	config.RetryableErrors = []error{retryableError}
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, retryableError
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, config.MaxAttempts, attemptCount, "should retry retryable error")
}

func TestRetry_CustomRetryableChecker(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialBackoff = 1 * time.Millisecond
	config.MaxAttempts = 3
	attemptCount := 0

	// Custom checker that only retries testError
	config.RetryableChecker = func(err error) bool {
		return errors.Is(err, testError)
	}

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, testError
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 3, attemptCount, "should retry based on custom checker")
}

func TestRetry_CircuitBreakerOpenNotRetried(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialBackoff = 1 * time.Millisecond
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, ErrCircuitOpen
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrCircuitOpen, err)
	assert.Equal(t, 1, attemptCount, "should not retry circuit breaker open errors")
}

func TestRetry_ContextCanceledNotRetried(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialBackoff = 1 * time.Millisecond
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, context.Canceled
	}

	result, err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 1, attemptCount, "should not retry context canceled errors")
}

func TestCalculateBackoff_ExponentialGrowth(t *testing.T) {
	config := RetryConfig{
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		EnableJitter:      false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},   // 1 * 2^0
		{2, 2 * time.Second},   // 1 * 2^1
		{3, 4 * time.Second},   // 1 * 2^2
		{4, 8 * time.Second},   // 1 * 2^3
		{5, 16 * time.Second},  // 1 * 2^4
		{6, 30 * time.Second},  // capped at max
		{10, 30 * time.Second}, // capped at max
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.attempt)), func(t *testing.T) {
			backoff := calculateBackoff(tt.attempt, config)
			assert.Equal(t, tt.expected, backoff)
		})
	}
}

func TestCalculateBackoff_WithJitter(t *testing.T) {
	config := RetryConfig{
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		EnableJitter:      true,
	}

	backoff1 := calculateBackoff(3, config)
	backoff2 := calculateBackoff(3, config)
	backoff3 := calculateBackoff(3, config)

	// With jitter, backoffs should be random
	// At least one should be different (very high probability)
	different := backoff1 != backoff2 || backoff2 != backoff3 || backoff1 != backoff3
	assert.True(t, different, "jitter should produce different backoff values")

	// All should be less than or equal to the expected backoff
	expected := 4 * time.Second
	assert.LessOrEqual(t, backoff1, expected)
	assert.LessOrEqual(t, backoff2, expected)
	assert.LessOrEqual(t, backoff3, expected)
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.InitialBackoff)
	assert.Equal(t, 30*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
	assert.True(t, config.EnableJitter)
}

func TestAggressiveRetryConfig(t *testing.T) {
	config := AggressiveRetryConfig()

	assert.Equal(t, 5, config.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, config.InitialBackoff)
	assert.Equal(t, 16*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
	assert.True(t, config.EnableJitter)
}

func TestConservativeRetryConfig(t *testing.T) {
	config := ConservativeRetryConfig()

	assert.Equal(t, 2, config.MaxAttempts)
	assert.Equal(t, 2*time.Second, config.InitialBackoff)
	assert.Equal(t, 10*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
	assert.True(t, config.EnableJitter)
}

func TestIsRetryableHTTPStatus(t *testing.T) {
	tests := []struct {
		statusCode int
		retryable  bool
	}{
		{200, false}, // OK
		{201, false}, // Created
		{400, false}, // Bad Request
		{401, false}, // Unauthorized
		{403, false}, // Forbidden
		{404, false}, // Not Found
		{408, true},  // Request Timeout
		{429, true},  // Too Many Requests
		{500, true},  // Internal Server Error
		{502, true},  // Bad Gateway
		{503, true},  // Service Unavailable
		{504, true},  // Gateway Timeout
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			result := IsRetryableHTTPStatus(tt.statusCode)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestRetryWithBreaker_Integration(t *testing.T) {
	ctx := context.Background()
	retryConfig := DefaultRetryConfig()
	retryConfig.InitialBackoff = 1 * time.Millisecond
	retryConfig.MaxAttempts = 3

	breakerSettings := Settings{
		Name:             "test-breaker",
		Interval:         100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 2,
	}
	breaker := NewCircuitBreaker(breakerSettings, NoopFallback)

	attemptCount := 0
	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 2 {
			return nil, testError
		}
		return "success", nil
	}

	result, err := RetryWithBreaker(ctx, retryConfig, breaker, operation)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 2, attemptCount)
}

func TestRetry_ZeroMaxAttempts(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.MaxAttempts = 0
	attemptCount := 0

	operation := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return "success", nil
	}

	result, err := Retry(ctx, config, operation)

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, attemptCount, "should attempt at least once even with MaxAttempts=0")
}

func TestAddJitter(t *testing.T) {
	duration := 10 * time.Second

	// Run multiple times to ensure jitter is working
	results := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		jittered := addJitter(duration)
		results[jittered] = true

		// Jittered value should be between 0 and duration
		assert.GreaterOrEqual(t, jittered, time.Duration(0))
		assert.LessOrEqual(t, jittered, duration)
	}

	// Should have at least some variation (very high probability with 10 samples)
	assert.Greater(t, len(results), 1, "jitter should produce different values")
}

func TestAddJitter_ZeroDuration(t *testing.T) {
	duration := time.Duration(0)
	jittered := addJitter(duration)
	assert.Equal(t, time.Duration(0), jittered)
}

func TestShouldRetry_NilError(t *testing.T) {
	config := DefaultRetryConfig()
	result := shouldRetry(nil, config)
	assert.False(t, result, "should not retry nil error")
}
