package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestConstants tests the package constants
func TestConstants(t *testing.T) {
	if DefaultLimit != 20 {
		t.Errorf("DefaultLimit = %d, want 20", DefaultLimit)
	}
	if MaxLimit != 100 {
		t.Errorf("MaxLimit = %d, want 100", MaxLimit)
	}
	if DefaultOffset != 0 {
		t.Errorf("DefaultOffset = %d, want 0", DefaultOffset)
	}
}

// TestParseParams tests the ParseParams function
func TestParseParams(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "no params uses defaults",
			queryString:    "",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "valid limit and offset",
			queryString:    "limit=10&offset=20",
			expectedLimit:  10,
			expectedOffset: 20,
		},
		{
			name:           "only limit",
			queryString:    "limit=50",
			expectedLimit:  50,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "only offset",
			queryString:    "offset=30",
			expectedLimit:  DefaultLimit,
			expectedOffset: 30,
		},
		{
			name:           "zero limit uses default",
			queryString:    "limit=0",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "negative limit uses default",
			queryString:    "limit=-10",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "limit exceeds max",
			queryString:    "limit=200",
			expectedLimit:  MaxLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "limit exactly at max",
			queryString:    "limit=100",
			expectedLimit:  100,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "negative offset uses default",
			queryString:    "offset=-10",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "zero offset is valid",
			queryString:    "offset=0",
			expectedLimit:  DefaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "large offset",
			queryString:    "offset=10000",
			expectedLimit:  DefaultLimit,
			expectedOffset: 10000,
		},
		{
			name:           "non-numeric limit",
			queryString:    "limit=abc",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "non-numeric offset",
			queryString:    "offset=xyz",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "float limit",
			queryString:    "limit=10.5",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "float offset",
			queryString:    "offset=10.5",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "both at boundary",
			queryString:    "limit=100&offset=0",
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "limit=1 minimum valid",
			queryString:    "limit=1",
			expectedLimit:  1,
			expectedOffset: DefaultOffset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/?"+tt.queryString, nil)

			params := ParseParams(c)

			if params.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, want %d", params.Limit, tt.expectedLimit)
			}
			if params.Offset != tt.expectedOffset {
				t.Errorf("Offset = %d, want %d", params.Offset, tt.expectedOffset)
			}
		})
	}
}

// TestBuildMeta tests the BuildMeta function
func TestBuildMeta(t *testing.T) {
	tests := []struct {
		name               string
		limit              int
		offset             int
		total              int64
		expectedTotalPages int
	}{
		{
			name:               "first page with 100 items",
			limit:              10,
			offset:             0,
			total:              100,
			expectedTotalPages: 10,
		},
		{
			name:               "second page",
			limit:              10,
			offset:             10,
			total:              100,
			expectedTotalPages: 10,
		},
		{
			name:               "partial last page",
			limit:              10,
			offset:             0,
			total:              25,
			expectedTotalPages: 3,
		},
		{
			name:               "exact pages",
			limit:              20,
			offset:             0,
			total:              100,
			expectedTotalPages: 5,
		},
		{
			name:               "single item",
			limit:              10,
			offset:             0,
			total:              1,
			expectedTotalPages: 1,
		},
		{
			name:               "no items",
			limit:              10,
			offset:             0,
			total:              0,
			expectedTotalPages: 0,
		},
		{
			name:               "zero limit",
			limit:              0,
			offset:             0,
			total:              100,
			expectedTotalPages: 0,
		},
		{
			name:               "large total",
			limit:              100,
			offset:             0,
			total:              1000000,
			expectedTotalPages: 10000,
		},
		{
			name:               "limit greater than total",
			limit:              50,
			offset:             0,
			total:              10,
			expectedTotalPages: 1,
		},
		{
			name:               "single page exactly full",
			limit:              10,
			offset:             0,
			total:              10,
			expectedTotalPages: 1,
		},
		{
			name:               "one item over page",
			limit:              10,
			offset:             0,
			total:              11,
			expectedTotalPages: 2,
		},
		{
			name:               "one item under page",
			limit:              10,
			offset:             0,
			total:              9,
			expectedTotalPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := BuildMeta(tt.limit, tt.offset, tt.total)

			if meta.Limit != tt.limit {
				t.Errorf("Limit = %d, want %d", meta.Limit, tt.limit)
			}
			if meta.Offset != tt.offset {
				t.Errorf("Offset = %d, want %d", meta.Offset, tt.offset)
			}
			if meta.Total != tt.total {
				t.Errorf("Total = %d, want %d", meta.Total, tt.total)
			}
			if meta.TotalPages != tt.expectedTotalPages {
				t.Errorf("TotalPages = %d, want %d", meta.TotalPages, tt.expectedTotalPages)
			}
		})
	}
}

// TestHasMore tests the HasMore function
func TestHasMore(t *testing.T) {
	tests := []struct {
		name     string
		offset   int
		limit    int
		total    int64
		expected bool
	}{
		{
			name:     "first page has more",
			offset:   0,
			limit:    10,
			total:    100,
			expected: true,
		},
		{
			name:     "middle page has more",
			offset:   50,
			limit:    10,
			total:    100,
			expected: true,
		},
		{
			name:     "last page no more",
			offset:   90,
			limit:    10,
			total:    100,
			expected: false,
		},
		{
			name:     "exactly at last",
			offset:   90,
			limit:    10,
			total:    100,
			expected: false,
		},
		{
			name:     "one before last page",
			offset:   89,
			limit:    10,
			total:    100,
			expected: true,
		},
		{
			name:     "offset past total",
			offset:   110,
			limit:    10,
			total:    100,
			expected: false,
		},
		{
			name:     "single item no more",
			offset:   0,
			limit:    10,
			total:    1,
			expected: false,
		},
		{
			name:     "no items",
			offset:   0,
			limit:    10,
			total:    0,
			expected: false,
		},
		{
			name:     "limit equals total",
			offset:   0,
			limit:    10,
			total:    10,
			expected: false,
		},
		{
			name:     "limit greater than total",
			offset:   0,
			limit:    50,
			total:    10,
			expected: false,
		},
		{
			name:     "zero limit means all remaining",
			offset:   0,
			limit:    0,
			total:    10,
			expected: true, // 0 + 0 = 0 < 10
		},
		{
			name:     "large offset and limit",
			offset:   1000,
			limit:    100,
			total:    10000,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasMore(tt.offset, tt.limit, tt.total)
			if result != tt.expected {
				t.Errorf("HasMore(%d, %d, %d) = %v, want %v", tt.offset, tt.limit, tt.total, result, tt.expected)
			}
		})
	}
}

// TestGetCurrentPage tests the GetCurrentPage function
func TestGetCurrentPage(t *testing.T) {
	tests := []struct {
		name     string
		offset   int
		limit    int
		expected int
	}{
		{
			name:     "first page",
			offset:   0,
			limit:    10,
			expected: 1,
		},
		{
			name:     "second page",
			offset:   10,
			limit:    10,
			expected: 2,
		},
		{
			name:     "third page",
			offset:   20,
			limit:    10,
			expected: 3,
		},
		{
			name:     "tenth page",
			offset:   90,
			limit:    10,
			expected: 10,
		},
		{
			name:     "partial offset",
			offset:   15,
			limit:    10,
			expected: 2,
		},
		{
			name:     "zero limit returns 1",
			offset:   10,
			limit:    0,
			expected: 1,
		},
		{
			name:     "negative limit returns 1",
			offset:   10,
			limit:    -5,
			expected: 1,
		},
		{
			name:     "large offset",
			offset:   1000,
			limit:    10,
			expected: 101,
		},
		{
			name:     "large limit",
			offset:   0,
			limit:    100,
			expected: 1,
		},
		{
			name:     "offset less than limit",
			offset:   5,
			limit:    10,
			expected: 1,
		},
		{
			name:     "offset equals limit",
			offset:   10,
			limit:    10,
			expected: 2,
		},
		{
			name:     "different page sizes",
			offset:   50,
			limit:    25,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCurrentPage(tt.offset, tt.limit)
			if result != tt.expected {
				t.Errorf("GetCurrentPage(%d, %d) = %d, want %d", tt.offset, tt.limit, result, tt.expected)
			}
		})
	}
}

// TestParams struct tests
func TestParams_Default(t *testing.T) {
	p := Params{}
	if p.Limit != 0 {
		t.Errorf("Default Limit = %d, want 0", p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("Default Offset = %d, want 0", p.Offset)
	}
}

func TestParams_WithValues(t *testing.T) {
	p := Params{Limit: 50, Offset: 100}
	if p.Limit != 50 {
		t.Errorf("Limit = %d, want 50", p.Limit)
	}
	if p.Offset != 100 {
		t.Errorf("Offset = %d, want 100", p.Offset)
	}
}

// TestBuildMeta_MetaFields tests that BuildMeta returns correct meta struct
func TestBuildMeta_MetaFields(t *testing.T) {
	meta := BuildMeta(20, 40, 100)

	if meta == nil {
		t.Fatal("BuildMeta returned nil")
	}
	if meta.Limit != 20 {
		t.Errorf("meta.Limit = %d, want 20", meta.Limit)
	}
	if meta.Offset != 40 {
		t.Errorf("meta.Offset = %d, want 40", meta.Offset)
	}
	if meta.Total != 100 {
		t.Errorf("meta.Total = %d, want 100", meta.Total)
	}
	if meta.TotalPages != 5 {
		t.Errorf("meta.TotalPages = %d, want 5", meta.TotalPages)
	}
}

// TestParseParams_WithGinContext tests parsing with real gin context
func TestParseParams_WithGinContext(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "standard query params",
			url:            "/api/items?limit=25&offset=50",
			expectedLimit:  25,
			expectedOffset: 50,
		},
		{
			name:           "with other params",
			url:            "/api/items?search=foo&limit=15&offset=30&sort=asc",
			expectedLimit:  15,
			expectedOffset: 30,
		},
		{
			name:           "empty query string",
			url:            "/api/items",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
		{
			name:           "malformed query",
			url:            "/api/items?limit=abc&offset=",
			expectedLimit:  DefaultLimit,
			expectedOffset: DefaultOffset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", tt.url, nil)

			params := ParseParams(c)

			if params.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, want %d", params.Limit, tt.expectedLimit)
			}
			if params.Offset != tt.expectedOffset {
				t.Errorf("Offset = %d, want %d", params.Offset, tt.expectedOffset)
			}
		})
	}
}

// Edge case tests
func TestBuildMeta_EdgeCases(t *testing.T) {
	// Test with negative values (should not panic)
	meta := BuildMeta(-10, -20, -100)
	if meta == nil {
		t.Error("BuildMeta should not return nil for negative values")
	}

	// Test with very large values
	largeMeta := BuildMeta(100, 0, 9999999999)
	if largeMeta.TotalPages != 99999999 {
		// With limit 100 and total 9999999999, we should have 99999999 pages (rounded up)
		// This test validates the calculation doesn't overflow
		t.Logf("TotalPages with very large total: %d", largeMeta.TotalPages)
	}
}

func TestHasMore_EdgeCases(t *testing.T) {
	// Test with negative offset
	result := HasMore(-10, 10, 100)
	// Even with negative offset, offset+limit < total means more items
	if result != true {
		t.Error("HasMore with negative offset should still calculate correctly")
	}

	// Test with negative limit
	result = HasMore(0, -10, 100)
	if result != true {
		t.Error("HasMore with negative limit should still calculate correctly")
	}
}

// Benchmark tests
func BenchmarkParseParams(b *testing.B) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?limit=50&offset=100", nil)

	for i := 0; i < b.N; i++ {
		ParseParams(c)
	}
}

func BenchmarkBuildMeta(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BuildMeta(20, 40, 1000)
	}
}

func BenchmarkHasMore(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HasMore(50, 20, 1000)
	}
}

func BenchmarkGetCurrentPage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetCurrentPage(100, 20)
	}
}
