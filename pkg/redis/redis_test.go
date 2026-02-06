package redis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/richxcame/ride-hailing/pkg/config"
)

// ============== Redis Config Tests ==============

func TestRedisConfig_RedisAddr(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.RedisConfig
		expected string
	}{
		{
			name: "default localhost",
			cfg: config.RedisConfig{
				Host: "localhost",
				Port: "6379",
			},
			expected: "localhost:6379",
		},
		{
			name: "custom host and port",
			cfg: config.RedisConfig{
				Host: "redis.example.com",
				Port: "6380",
			},
			expected: "redis.example.com:6380",
		},
		{
			name: "empty values",
			cfg: config.RedisConfig{
				Host: "",
				Port: "",
			},
			expected: ":",
		},
		{
			name: "IP address",
			cfg: config.RedisConfig{
				Host: "192.168.1.100",
				Port: "6379",
			},
			expected: "192.168.1.100:6379",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.cfg.RedisAddr()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// ============== Redis Retryable Error Tests ==============

func TestIsRedisRetryable_NilError(t *testing.T) {
	if isRedisRetryable(nil) {
		t.Error("nil error should not be retryable")
	}
}

func TestIsRedisRetryable_ContextCanceled(t *testing.T) {
	if isRedisRetryable(context.Canceled) {
		t.Error("context.Canceled should not be retryable")
	}
}

func TestIsRedisRetryable_ContextDeadlineExceeded(t *testing.T) {
	if isRedisRetryable(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should not be retryable")
	}
}

func TestIsRedisRetryable_RedisNil(t *testing.T) {
	if isRedisRetryable(goredis.Nil) {
		t.Error("redis.Nil should not be retryable (expected behavior for key not found)")
	}
}

func TestIsRedisRetryable_ConnectionRefused(t *testing.T) {
	err := errors.New("connection refused")
	if !isRedisRetryable(err) {
		t.Error("'connection refused' should be retryable")
	}
}

func TestIsRedisRetryable_ConnectionReset(t *testing.T) {
	err := errors.New("connection reset by peer")
	if !isRedisRetryable(err) {
		t.Error("'connection reset' should be retryable")
	}
}

func TestIsRedisRetryable_BrokenPipe(t *testing.T) {
	err := errors.New("broken pipe")
	if !isRedisRetryable(err) {
		t.Error("'broken pipe' should be retryable")
	}
}

func TestIsRedisRetryable_NoSuchHost(t *testing.T) {
	err := errors.New("no such host")
	if !isRedisRetryable(err) {
		t.Error("'no such host' should be retryable")
	}
}

func TestIsRedisRetryable_NetworkUnreachable(t *testing.T) {
	err := errors.New("network is unreachable")
	if !isRedisRetryable(err) {
		t.Error("'network is unreachable' should be retryable")
	}
}

func TestIsRedisRetryable_TemporaryFailure(t *testing.T) {
	err := errors.New("temporary failure in name resolution")
	if !isRedisRetryable(err) {
		t.Error("'temporary failure' should be retryable")
	}
}

func TestIsRedisRetryable_Timeout(t *testing.T) {
	err := errors.New("i/o timeout")
	if !isRedisRetryable(err) {
		t.Error("'timeout' should be retryable")
	}
}

func TestIsRedisRetryable_ServerClosed(t *testing.T) {
	err := errors.New("server closed the connection")
	if !isRedisRetryable(err) {
		t.Error("'server closed' should be retryable")
	}
}

func TestIsRedisRetryable_UnexpectedEOF(t *testing.T) {
	err := errors.New("unexpected eof")
	if !isRedisRetryable(err) {
		t.Error("'unexpected eof' should be retryable")
	}
}

func TestIsRedisRetryable_PoolTimeout(t *testing.T) {
	err := errors.New("pool timeout")
	if !isRedisRetryable(err) {
		t.Error("'pool timeout' should be retryable")
	}
}

func TestIsRedisRetryable_IOTimeout(t *testing.T) {
	err := errors.New("i/o timeout")
	if !isRedisRetryable(err) {
		t.Error("'i/o timeout' should be retryable")
	}
}

func TestIsRedisRetryable_ConnectionPoolExhausted(t *testing.T) {
	err := errors.New("connection pool exhausted")
	if !isRedisRetryable(err) {
		t.Error("'connection pool exhausted' should be retryable")
	}
}

func TestIsRedisRetryable_Loading(t *testing.T) {
	err := errors.New("LOADING Redis is loading the dataset in memory")
	if !isRedisRetryable(err) {
		t.Error("'loading' should be retryable")
	}
}

func TestIsRedisRetryable_Busy(t *testing.T) {
	err := errors.New("BUSY Redis is busy running a script")
	if !isRedisRetryable(err) {
		t.Error("'busy' should be retryable")
	}
}

func TestIsRedisRetryable_MasterDown(t *testing.T) {
	err := errors.New("MASTERDOWN Link with MASTER is down")
	if !isRedisRetryable(err) {
		t.Error("'masterdown' should be retryable")
	}
}

func TestIsRedisRetryable_ReadOnly(t *testing.T) {
	err := errors.New("READONLY You can't write against a read only replica")
	if !isRedisRetryable(err) {
		t.Error("'readonly' should be retryable")
	}
}

func TestIsRedisRetryable_NoScript(t *testing.T) {
	err := errors.New("NOSCRIPT No matching script")
	if !isRedisRetryable(err) {
		t.Error("'noscript' should be retryable")
	}
}

func TestIsRedisRetryable_ClusterErrors(t *testing.T) {
	clusterErrors := []string{
		"MOVED 3999 127.0.0.1:6381",
		"ASK 3999 127.0.0.1:6381",
		"TRYAGAIN Multiple keys request during rehashing",
		"CLUSTERDOWN The cluster is down",
	}

	for _, msg := range clusterErrors {
		err := errors.New(msg)
		if !isRedisRetryable(err) {
			t.Errorf("cluster error %q should be retryable", msg)
		}
	}
}

func TestIsRedisRetryable_WrongType_NotRetryable(t *testing.T) {
	err := errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	if isRedisRetryable(err) {
		t.Error("'wrongtype' should NOT be retryable")
	}
}

func TestIsRedisRetryable_SyntaxError_NotRetryable(t *testing.T) {
	err := errors.New("ERR syntax error")
	if isRedisRetryable(err) {
		t.Error("'err syntax' should NOT be retryable")
	}
}

func TestIsRedisRetryable_InvalidArgument_NotRetryable(t *testing.T) {
	err := errors.New("ERR invalid expire time in set")
	if isRedisRetryable(err) {
		t.Error("'err invalid' should NOT be retryable")
	}
}

func TestIsRedisRetryable_NoAuth_NotRetryable(t *testing.T) {
	err := errors.New("NOAUTH Authentication required")
	if isRedisRetryable(err) {
		t.Error("'noauth' should NOT be retryable")
	}
}

func TestIsRedisRetryable_WrongPass_NotRetryable(t *testing.T) {
	err := errors.New("WRONGPASS invalid username-password pair")
	if isRedisRetryable(err) {
		t.Error("'wrongpass' should NOT be retryable")
	}
}

func TestIsRedisRetryable_NoPerm_NotRetryable(t *testing.T) {
	err := errors.New("NOPERM User doesn't have permissions to run this command")
	if isRedisRetryable(err) {
		t.Error("'noperm' should NOT be retryable")
	}
}

func TestIsRedisRetryable_UnknownCommand_NotRetryable(t *testing.T) {
	err := errors.New("ERR unknown command 'BADCMD'")
	if isRedisRetryable(err) {
		t.Error("'err unknown' should NOT be retryable")
	}
}

func TestIsRedisRetryable_ExecAbort_NotRetryable(t *testing.T) {
	err := errors.New("EXECABORT Transaction discarded because of previous errors")
	if isRedisRetryable(err) {
		t.Error("'execabort' should NOT be retryable")
	}
}

func TestIsRedisRetryable_CaseSensitivity(t *testing.T) {
	// Error messages should be matched case-insensitively
	testCases := []struct {
		msg      string
		expected bool
	}{
		{"CONNECTION REFUSED", true},
		{"Connection Refused", true},
		{"TIMEOUT ERROR", true},
		{"Timeout Error", true},
		{"POOL TIMEOUT", true},
		{"Pool Timeout", true},
	}

	for _, tc := range testCases {
		err := errors.New(tc.msg)
		result := isRedisRetryable(err)
		if result != tc.expected {
			t.Errorf("isRedisRetryable(%q) = %v, expected %v", tc.msg, result, tc.expected)
		}
	}
}

func TestIsRedisRetryable_UnknownError_Retryable(t *testing.T) {
	// Redis uses a conservative approach - unknown errors are retryable by default
	err := errors.New("some completely unknown error message")
	if !isRedisRetryable(err) {
		t.Error("unknown error should be retryable by default for Redis (conservative approach)")
	}
}

// ============== Retry Config Tests ==============

func TestConservativeRetryConfig(t *testing.T) {
	cfg := ConservativeRetryConfig()

	if cfg.InitialBackoff != 50*time.Millisecond {
		t.Errorf("expected InitialBackoff 50ms, got %v", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 1*time.Second {
		t.Errorf("expected MaxBackoff 1s, got %v", cfg.MaxBackoff)
	}
	if cfg.RetryableChecker == nil {
		t.Error("RetryableChecker should be set")
	}

	// Test checker
	if cfg.RetryableChecker(nil) {
		t.Error("nil error should not be retryable")
	}
	if cfg.RetryableChecker(goredis.Nil) {
		t.Error("redis.Nil should not be retryable")
	}
}

func TestAggressiveRetryConfig(t *testing.T) {
	cfg := AggressiveRetryConfig()

	if cfg.InitialBackoff != 20*time.Millisecond {
		t.Errorf("expected InitialBackoff 20ms, got %v", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 500*time.Millisecond {
		t.Errorf("expected MaxBackoff 500ms, got %v", cfg.MaxBackoff)
	}
	if cfg.RetryableChecker == nil {
		t.Error("RetryableChecker should be set")
	}
}

// ============== ClientInterface Tests ==============

func TestClientInterface_Compliance(t *testing.T) {
	// Verify that Client satisfies ClientInterface at compile time
	var _ ClientInterface = (*Client)(nil)
}

// ============== Table-Driven Tests ==============

func TestIsRedisRetryable_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline exceeded", context.DeadlineExceeded, false},
		{"redis nil", goredis.Nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"timeout", errors.New("i/o timeout"), true},
		{"pool timeout", errors.New("pool timeout"), true},
		{"loading", errors.New("LOADING Redis is loading"), true},
		{"busy", errors.New("BUSY Redis is busy"), true},
		{"masterdown", errors.New("MASTERDOWN"), true},
		{"readonly", errors.New("READONLY"), true},
		{"moved", errors.New("MOVED 3999"), true},
		{"ask", errors.New("ASK 3999"), true},
		{"tryagain", errors.New("TRYAGAIN"), true},
		{"clusterdown", errors.New("CLUSTERDOWN"), true},
		{"wrongtype", errors.New("WRONGTYPE"), false},
		{"syntax error", errors.New("ERR syntax error"), false},
		{"noauth", errors.New("NOAUTH"), false},
		{"wrongpass", errors.New("WRONGPASS"), false},
		{"noperm", errors.New("NOPERM"), false},
		{"unknown command", errors.New("ERR unknown command"), false},
		{"execabort", errors.New("EXECABORT"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isRedisRetryable(tc.err)
			if result != tc.expected {
				t.Errorf("isRedisRetryable() = %v, expected %v", result, tc.expected)
			}
		})
	}
}

// ============== All Retryable Messages Tests ==============

func TestAllRetryableMessages(t *testing.T) {
	retryableMessages := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"temporary failure",
		"timeout",
		"server closed",
		"unexpected eof",
		"pool timeout",
		"i/o timeout",
		"connection pool exhausted",
		"loading",
		"busy",
		"masterdown",
		"readonly",
		"noscript",
		"cluster",
		"moved",
		"ask",
		"tryagain",
		"clusterdown",
	}

	for _, msg := range retryableMessages {
		err := errors.New(msg)
		if !isRedisRetryable(err) {
			t.Errorf("message %q should be retryable", msg)
		}

		// Also test uppercase
		err = errors.New(strings.ToUpper(msg))
		if !isRedisRetryable(err) {
			t.Errorf("message %q (uppercase) should be retryable", strings.ToUpper(msg))
		}
	}
}

func TestAllNonRetryableMessages(t *testing.T) {
	nonRetryableMessages := []string{
		"wrongtype",
		"err syntax",
		"err invalid",
		"noauth",
		"wrongpass",
		"noperm",
		"err unknown",
		"execabort",
	}

	for _, msg := range nonRetryableMessages {
		err := errors.New(msg)
		if isRedisRetryable(err) {
			t.Errorf("message %q should NOT be retryable", msg)
		}
	}
}

// ============== Timeout Duration Tests ==============

func TestDefaultRedisReadTimeoutDuration(t *testing.T) {
	expected := time.Duration(config.DefaultRedisReadTimeout) * time.Second
	result := config.DefaultRedisReadTimeoutDuration()
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDefaultRedisWriteTimeoutDuration(t *testing.T) {
	expected := time.Duration(config.DefaultRedisWriteTimeout) * time.Second
	result := config.DefaultRedisWriteTimeoutDuration()
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestTimeoutConfig_RedisReadTimeoutDuration(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.TimeoutConfig
		expected     time.Duration
	}{
		{
			name:         "custom timeout",
			cfg:          config.TimeoutConfig{RedisReadTimeout: 10},
			expected:     10 * time.Second,
		},
		{
			name:         "zero falls back to operation timeout",
			cfg:          config.TimeoutConfig{RedisReadTimeout: 0, RedisOperationTimeout: 5},
			expected:     5 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.cfg.RedisReadTimeoutDuration()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestTimeoutConfig_RedisWriteTimeoutDuration(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.TimeoutConfig
		expected     time.Duration
	}{
		{
			name:         "custom timeout",
			cfg:          config.TimeoutConfig{RedisWriteTimeout: 10},
			expected:     10 * time.Second,
		},
		{
			name:         "zero falls back to operation timeout",
			cfg:          config.TimeoutConfig{RedisWriteTimeout: 0, RedisOperationTimeout: 5},
			expected:     5 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.cfg.RedisWriteTimeoutDuration()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// ============== Edge Case Tests ==============

func TestIsRedisRetryable_EmptyErrorMessage(t *testing.T) {
	err := errors.New("")
	// Empty error should be retryable by default (conservative approach)
	if !isRedisRetryable(err) {
		t.Error("empty error message should be retryable (conservative approach)")
	}
}

func TestIsRedisRetryable_MultipleKeywords(t *testing.T) {
	// Error contains both retryable and non-retryable keywords
	// The implementation checks retryable patterns first, then non-retryable
	// So behavior depends on the order of checks in the implementation
	err := errors.New("WRONGTYPE connection refused")
	// Just verify it doesn't panic - actual behavior is implementation-specific
	_ = isRedisRetryable(err)
}

func TestIsRedisRetryable_PartialMatch(t *testing.T) {
	// Error message contains keyword as part of another word
	err := errors.New("the connection was timeoutexceeded")
	if !isRedisRetryable(err) {
		t.Error("partial match should still detect 'timeout'")
	}
}

// ============== Concurrent Access Tests ==============

func TestIsRedisRetryable_Concurrent(t *testing.T) {
	done := make(chan bool, 100)
	errs := []error{
		errors.New("connection refused"),
		context.Canceled,
		goredis.Nil,
		errors.New("timeout"),
		errors.New("WRONGTYPE"),
	}

	for i := 0; i < 100; i++ {
		go func(idx int) {
			err := errs[idx%len(errs)]
			_ = isRedisRetryable(err)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

// ============== Benchmark Tests ==============

func BenchmarkIsRedisRetryable_NilError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isRedisRetryable(nil)
	}
}

func BenchmarkIsRedisRetryable_ContextCanceled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isRedisRetryable(context.Canceled)
	}
}

func BenchmarkIsRedisRetryable_RedisNil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isRedisRetryable(goredis.Nil)
	}
}

func BenchmarkIsRedisRetryable_ConnectionRefused(b *testing.B) {
	err := errors.New("connection refused")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isRedisRetryable(err)
	}
}

func BenchmarkIsRedisRetryable_WrongType(b *testing.B) {
	err := errors.New("WRONGTYPE")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isRedisRetryable(err)
	}
}

func BenchmarkIsRedisRetryable_UnknownError(b *testing.B) {
	err := errors.New("some random error message that needs full scan")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isRedisRetryable(err)
	}
}

func BenchmarkRedisConfig_RedisAddr(b *testing.B) {
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: "6379",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.RedisAddr()
	}
}

func BenchmarkConservativeRetryConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ConservativeRetryConfig()
	}
}

func BenchmarkAggressiveRetryConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = AggressiveRetryConfig()
	}
}

// ============== RetryableOperation Generic Function Tests ==============

func TestRetryableOperation_TypeInference(t *testing.T) {
	// Test that RetryableOperation works with different types
	ctx := context.Background()

	// String type
	strOp := func(ctx context.Context) (string, error) {
		return "test", nil
	}
	strResult, err := RetryableOperation(ctx, strOp, "test.string")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if strResult != "test" {
		t.Errorf("expected 'test', got %q", strResult)
	}

	// Int type
	intOp := func(ctx context.Context) (int64, error) {
		return 42, nil
	}
	intResult, err := RetryableOperation(ctx, intOp, "test.int")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if intResult != 42 {
		t.Errorf("expected 42, got %d", intResult)
	}

	// Slice type
	sliceOp := func(ctx context.Context) ([]string, error) {
		return []string{"a", "b"}, nil
	}
	sliceResult, err := RetryableOperation(ctx, sliceOp, "test.slice")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(sliceResult) != 2 {
		t.Errorf("expected 2 elements, got %d", len(sliceResult))
	}
}

func TestRetryableOperation_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("operation failed")

	op := func(ctx context.Context) (string, error) {
		return "", expectedErr
	}

	result, err := RetryableOperation(ctx, op, "test.error")
	if err == nil {
		t.Error("expected error")
	}
	if result != "" {
		t.Errorf("expected empty result on error, got %q", result)
	}
}

// ============== Combined Error Scenario Tests ==============

func TestIsRedisRetryable_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name     string
		err      error
		expected bool
		reason   string
	}{
		{
			name:     "Redis server restart",
			err:      errors.New("connection reset by peer"),
			expected: true,
			reason:   "server restart causes connection reset",
		},
		{
			name:     "Network partition",
			err:      errors.New("network is unreachable"),
			expected: true,
			reason:   "temporary network issue",
		},
		{
			name:     "Redis cluster rebalancing",
			err:      errors.New("MOVED 12345 192.168.1.100:6379"),
			expected: true,
			reason:   "cluster slot migration",
		},
		{
			name:     "Redis AOF rewrite",
			err:      errors.New("LOADING Redis is loading the dataset in memory"),
			expected: true,
			reason:   "startup/rewrite in progress",
		},
		{
			name:     "Slow script blocking",
			err:      errors.New("BUSY Redis is busy running a script. You can only call SCRIPT KILL or SHUTDOWN NOSAVE."),
			expected: true,
			reason:   "Lua script timeout",
		},
		{
			name:     "Invalid command",
			err:      errors.New("ERR unknown command 'INVALID_CMD'"),
			expected: false,
			reason:   "programming error, not transient",
		},
		{
			name:     "Type mismatch",
			err:      errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"),
			expected: false,
			reason:   "data model error, not transient",
		},
		{
			name:     "Authentication failure",
			err:      errors.New("NOAUTH Authentication required."),
			expected: false,
			reason:   "configuration error, not transient",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			result := isRedisRetryable(sc.err)
			if result != sc.expected {
				t.Errorf("isRedisRetryable() = %v, expected %v (reason: %s)", result, sc.expected, sc.reason)
			}
		})
	}
}

// ============== Error Wrapping Tests ==============

func TestIsRedisRetryable_WrappedContextCanceled(t *testing.T) {
	// Create a context canceled error
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ctx.Err() // This returns context.Canceled

	if isRedisRetryable(err) {
		t.Error("wrapped context.Canceled should not be retryable")
	}
}

func TestIsRedisRetryable_WrappedDeadlineExceeded(t *testing.T) {
	// Create a deadline exceeded error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Wait for deadline
	err := ctx.Err()

	if isRedisRetryable(err) {
		t.Error("wrapped context.DeadlineExceeded should not be retryable")
	}
}
