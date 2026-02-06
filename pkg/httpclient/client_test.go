package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/richxcame/ride-hailing/pkg/resilience"
)

// TestNewClient tests the NewClient constructor
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		timeout []time.Duration
	}{
		{
			name:    "with base URL only",
			baseURL: "https://api.example.com",
			timeout: nil,
		},
		{
			name:    "with custom timeout",
			baseURL: "https://api.example.com",
			timeout: []time.Duration{5 * time.Second},
		},
		{
			name:    "with zero timeout uses default",
			baseURL: "https://api.example.com",
			timeout: []time.Duration{0},
		},
		{
			name:    "with multiple timeouts uses first",
			baseURL: "https://api.example.com",
			timeout: []time.Duration{10 * time.Second, 20 * time.Second},
		},
		{
			name:    "empty base URL",
			baseURL: "",
			timeout: nil,
		},
		{
			name:    "with path in base URL",
			baseURL: "https://api.example.com/v1",
			timeout: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.timeout != nil {
				client = NewClient(tt.baseURL, tt.timeout...)
			} else {
				client = NewClient(tt.baseURL)
			}

			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.baseURL != tt.baseURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.baseURL)
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

// TestWithRetry tests the WithRetry option
func TestWithRetry(t *testing.T) {
	config := resilience.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	client := NewClient("https://api.example.com")
	option := WithRetry(config)
	option(client)

	if client.retryConfig == nil {
		t.Fatal("retryConfig is nil after WithRetry")
	}
	if client.retryConfig.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", client.retryConfig.MaxAttempts)
	}
}

// TestWithDefaultRetry tests the WithDefaultRetry option
func TestWithDefaultRetry(t *testing.T) {
	client := NewClient("https://api.example.com")
	option := WithDefaultRetry()
	option(client)

	if client.retryConfig == nil {
		t.Fatal("retryConfig is nil after WithDefaultRetry")
	}
	if client.retryConfig.RetryableChecker == nil {
		t.Error("RetryableChecker should be set")
	}
}

// TestClient_Get tests the Get method
func TestClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		path           string
		headers        map[string]string
		expectedBody   string
		expectError    bool
		expectedStatus int
	}{
		{
			name: "successful GET",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Method = %s, want GET", r.Method)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message":"success"}`))
			},
			path:           "/test",
			headers:        nil,
			expectedBody:   `{"message":"success"}`,
			expectError:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "GET with custom headers",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Custom-Header") != "custom-value" {
					t.Error("Custom header not set")
				}
				if r.Header.Get("Authorization") != "Bearer token" {
					t.Error("Authorization header not set")
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"authenticated":true}`))
			},
			path: "/auth",
			headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"Authorization":   "Bearer token",
			},
			expectedBody: `{"authenticated":true}`,
			expectError:  false,
		},
		{
			name: "GET returns 404",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"not found"}`))
			},
			path:           "/notfound",
			headers:        nil,
			expectError:    true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "GET returns 500",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"server error"}`))
			},
			path:           "/error",
			headers:        nil,
			expectError:    true,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "GET returns empty body",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			path:           "/empty",
			headers:        nil,
			expectedBody:   "",
			expectError:    false,
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := NewClient(server.URL)
			body, err := client.Get(context.Background(), tt.path, tt.headers)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if httpErr, ok := err.(*HTTPError); ok {
					if httpErr.StatusCode != tt.expectedStatus {
						t.Errorf("StatusCode = %d, want %d", httpErr.StatusCode, tt.expectedStatus)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectedBody != "" && string(body) != tt.expectedBody {
					t.Errorf("Body = %s, want %s", string(body), tt.expectedBody)
				}
			}
		})
	}
}

// TestClient_Post tests the Post method
func TestClient_Post(t *testing.T) {
	tests := []struct {
		name          string
		serverHandler http.HandlerFunc
		path          string
		body          interface{}
		headers       map[string]string
		expectedBody  string
		expectError   bool
	}{
		{
			name: "successful POST with body",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("Content-Type should be application/json")
				}
				body, _ := io.ReadAll(r.Body)
				var data map[string]string
				json.Unmarshal(body, &data)
				if data["name"] != "test" {
					t.Errorf("Body name = %s, want test", data["name"])
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":"123"}`))
			},
			path:         "/create",
			body:         map[string]string{"name": "test"},
			headers:      nil,
			expectedBody: `{"id":"123"}`,
			expectError:  false,
		},
		{
			name: "POST with nil body",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if len(body) != 0 {
					t.Error("Body should be empty")
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success":true}`))
			},
			path:         "/empty",
			body:         nil,
			headers:      nil,
			expectedBody: `{"success":true}`,
			expectError:  false,
		},
		{
			name: "POST with custom headers",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Request-ID") != "12345" {
					t.Error("X-Request-ID header not set")
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
			path:    "/with-headers",
			body:    map[string]string{},
			headers: map[string]string{"X-Request-ID": "12345"},
			expectError: false,
		},
		{
			name: "POST returns error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid request"}`))
			},
			path:        "/error",
			body:        map[string]string{},
			headers:     nil,
			expectError: true,
		},
		{
			name: "POST with struct body",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var data struct {
					Name  string `json:"name"`
					Value int    `json:"value"`
				}
				json.Unmarshal(body, &data)
				if data.Name != "test" || data.Value != 42 {
					t.Errorf("Body = %+v", data)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
			path: "/struct",
			body: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{Name: "test", Value: 42},
			headers:     nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := NewClient(server.URL)
			body, err := client.Post(context.Background(), tt.path, tt.body, tt.headers)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectedBody != "" && string(body) != tt.expectedBody {
					t.Errorf("Body = %s, want %s", string(body), tt.expectedBody)
				}
			}
		})
	}
}

// TestClient_PostWithIdempotency tests the PostWithIdempotency method
func TestClient_PostWithIdempotency(t *testing.T) {
	tests := []struct {
		name           string
		idempotencyKey string
		expectKeySet   bool
	}{
		{
			name:           "with provided idempotency key",
			idempotencyKey: "custom-key-123",
			expectKeySet:   true,
		},
		{
			name:           "with empty key generates UUID",
			idempotencyKey: "",
			expectKeySet:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedKey string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedKey = r.Header.Get("Idempotency-Key")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			_, err := client.PostWithIdempotency(context.Background(), "/test", nil, nil, tt.idempotencyKey)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectKeySet && receivedKey == "" {
				t.Error("Idempotency-Key header should be set")
			}

			if tt.idempotencyKey != "" && receivedKey != tt.idempotencyKey {
				t.Errorf("Idempotency-Key = %s, want %s", receivedKey, tt.idempotencyKey)
			}
		})
	}
}

// TestClient_PostWithIdempotency_WithExistingHeaders tests that existing headers are preserved
func TestClient_PostWithIdempotency_WithExistingHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("Authorization header should be preserved")
		}
		if r.Header.Get("Idempotency-Key") != "key-123" {
			t.Error("Idempotency-Key should be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	headers := map[string]string{"Authorization": "Bearer token"}
	_, err := client.PostWithIdempotency(context.Background(), "/test", nil, headers, "key-123")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestHTTPError tests the HTTPError struct
func TestHTTPError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectedString string
	}{
		{
			name:           "404 error",
			statusCode:     404,
			body:           "not found",
			expectedString: "HTTP 404: not found",
		},
		{
			name:           "500 error",
			statusCode:     500,
			body:           "internal server error",
			expectedString: "HTTP 500: internal server error",
		},
		{
			name:           "400 with JSON body",
			statusCode:     400,
			body:           `{"error":"bad request"}`,
			expectedString: `HTTP 400: {"error":"bad request"}`,
		},
		{
			name:           "empty body",
			statusCode:     503,
			body:           "",
			expectedString: "HTTP 503: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &HTTPError{
				StatusCode: tt.statusCode,
				Body:       tt.body,
			}

			if err.Error() != tt.expectedString {
				t.Errorf("Error() = %s, want %s", err.Error(), tt.expectedString)
			}
		})
	}
}

// TestIsHTTPRetryable tests the isHTTPRetryable function
func TestIsHTTPRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "500 error is retryable",
			err:      &HTTPError{StatusCode: 500, Body: ""},
			expected: true,
		},
		{
			name:     "502 error is retryable",
			err:      &HTTPError{StatusCode: 502, Body: ""},
			expected: true,
		},
		{
			name:     "503 error is retryable",
			err:      &HTTPError{StatusCode: 503, Body: ""},
			expected: true,
		},
		{
			name:     "504 error is retryable",
			err:      &HTTPError{StatusCode: 504, Body: ""},
			expected: true,
		},
		{
			name:     "429 too many requests is retryable",
			err:      &HTTPError{StatusCode: 429, Body: ""},
			expected: true,
		},
		{
			name:     "400 error is not retryable",
			err:      &HTTPError{StatusCode: 400, Body: ""},
			expected: false,
		},
		{
			name:     "401 error is not retryable",
			err:      &HTTPError{StatusCode: 401, Body: ""},
			expected: false,
		},
		{
			name:     "403 error is not retryable",
			err:      &HTTPError{StatusCode: 403, Body: ""},
			expected: false,
		},
		{
			name:     "404 error is not retryable",
			err:      &HTTPError{StatusCode: 404, Body: ""},
			expected: false,
		},
		{
			name: "generic error is retryable",
			err:  context.DeadlineExceeded,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTTPRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("isHTTPRetryable(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestClient_ContextCancellation tests that requests respect context cancellation
func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "/slow", nil)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Logf("Error message: %v", err)
	}
}

// TestClient_Get_WithRetry tests GET with retry enabled
func TestClient_Get_WithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"temporarily unavailable"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	config := resilience.RetryConfig{
		MaxAttempts:       5,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2,
		RetryableChecker:  isHTTPRetryable,
	}
	WithRetry(config)(client)

	body, err := client.Get(context.Background(), "/retry", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(string(body), "success") {
		t.Errorf("Body = %s, want success response", string(body))
	}

	if attempts != 3 {
		t.Errorf("Attempts = %d, want 3", attempts)
	}
}

// TestClient_Post_WithRetry tests POST with retry enabled
func TestClient_Post_WithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	config := resilience.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2,
		RetryableChecker:  isHTTPRetryable,
	}
	WithRetry(config)(client)

	body, err := client.Post(context.Background(), "/retry", map[string]string{"test": "data"}, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(string(body), "created") {
		t.Errorf("Body = %s, want created response", string(body))
	}
}

// TestClient_InvalidURL tests behavior with invalid URLs
func TestClient_InvalidURL(t *testing.T) {
	client := NewClient("http://invalid-host-that-does-not-exist.local")

	_, err := client.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

// TestClient_LargeResponse tests handling of large responses
func TestClient_LargeResponse(t *testing.T) {
	largeBody := strings.Repeat("a", 1024*1024) // 1MB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	body, err := client.Get(context.Background(), "/large", nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(body) != len(largeBody) {
		t.Errorf("Body length = %d, want %d", len(body), len(largeBody))
	}
}

// Benchmark tests
func BenchmarkClient_Get(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Get(ctx, "/test", nil)
	}
}

func BenchmarkClient_Post(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"created"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	body := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Post(ctx, "/test", body, nil)
	}
}
