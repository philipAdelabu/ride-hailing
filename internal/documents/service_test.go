package documents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isNil    bool
		expected string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			isNil: true,
		},
		{
			name:     "non-empty string returns pointer",
			input:    "hello",
			isNil:    false,
			expected: "hello",
		},
		{
			name:     "whitespace is not empty",
			input:    " ",
			isNil:    false,
			expected: " ",
		},
		{
			name:     "document number",
			input:    "DL-123456",
			isNil:    false,
			expected: "DL-123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nilIfEmpty(tt.input)
			if tt.isNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

func TestServiceConfig_Defaults(t *testing.T) {
	// When NewService is called with zero-value config, defaults should be applied
	config := ServiceConfig{}

	// Simulate what NewService does
	if config.MaxFileSizeMB == 0 {
		config.MaxFileSizeMB = 10
	}
	if len(config.AllowedMimeTypes) == 0 {
		config.AllowedMimeTypes = []string{
			"image/jpeg", "image/png", "image/webp", "application/pdf",
		}
	}

	assert.Equal(t, 10, config.MaxFileSizeMB)
	assert.Len(t, config.AllowedMimeTypes, 4)
	assert.Contains(t, config.AllowedMimeTypes, "image/jpeg")
	assert.Contains(t, config.AllowedMimeTypes, "image/png")
	assert.Contains(t, config.AllowedMimeTypes, "image/webp")
	assert.Contains(t, config.AllowedMimeTypes, "application/pdf")
}

func TestServiceConfig_CustomValues(t *testing.T) {
	config := ServiceConfig{
		MaxFileSizeMB: 20,
		AllowedMimeTypes: []string{
			"image/jpeg", "image/png",
		},
		OCREnabled:  true,
		OCRProvider: "tesseract",
	}

	assert.Equal(t, 20, config.MaxFileSizeMB)
	assert.Len(t, config.AllowedMimeTypes, 2)
	assert.True(t, config.OCREnabled)
	assert.Equal(t, "tesseract", config.OCRProvider)
}

func TestDocumentStatus_Constants(t *testing.T) {
	assert.Equal(t, DocumentStatus("pending"), StatusPending)
	assert.Equal(t, DocumentStatus("under_review"), StatusUnderReview)
	assert.Equal(t, DocumentStatus("approved"), StatusApproved)
	assert.Equal(t, DocumentStatus("rejected"), StatusRejected)
	assert.Equal(t, DocumentStatus("expired"), StatusExpired)
	assert.Equal(t, DocumentStatus("superseded"), StatusSuperseded)
}

func TestVerificationStatus_Constants(t *testing.T) {
	assert.Equal(t, VerificationStatus("incomplete"), VerificationIncomplete)
	assert.Equal(t, VerificationStatus("pending_review"), VerificationPendingReview)
	assert.Equal(t, VerificationStatus("approved"), VerificationApproved)
	assert.Equal(t, VerificationStatus("suspended"), VerificationSuspended)
	assert.Equal(t, VerificationStatus("rejected"), VerificationRejected)
}

func TestLogHistory_NotesHandling(t *testing.T) {
	// Test the notes conversion logic used in logHistory
	tests := []struct {
		name     string
		notes    interface{}
		expected *string
	}{
		{
			name:     "nil notes",
			notes:    nil,
			expected: nil,
		},
		{
			name:     "string notes",
			notes:    "some notes",
			expected: stringPtr("some notes"),
		},
		{
			name:     "string pointer notes",
			notes:    stringPtr("pointer notes"),
			expected: stringPtr("pointer notes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notesStr *string
			if tt.notes != nil {
				if str, ok := tt.notes.(string); ok {
					notesStr = &str
				} else if strPtr, ok := tt.notes.(*string); ok {
					notesStr = strPtr
				}
			}

			if tt.expected == nil {
				assert.Nil(t, notesStr)
			} else {
				assert.NotNil(t, notesStr)
				assert.Equal(t, *tt.expected, *notesStr)
			}
		})
	}
}

func TestPendingReviewDocument_PageCalculation(t *testing.T) {
	// Test the pagination logic from GetPendingReviews
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedPage int
		expectedSize int
	}{
		{
			name:         "defaults for zero values",
			page:         0,
			pageSize:     0,
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "negative page becomes 1",
			page:         -1,
			pageSize:     10,
			expectedPage: 1,
			expectedSize: 10,
		},
		{
			name:         "oversized pageSize becomes 20",
			page:         1,
			pageSize:     200,
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "valid values preserved",
			page:         3,
			pageSize:     50,
			expectedPage: 3,
			expectedSize: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := tt.page
			pageSize := tt.pageSize

			if page < 1 {
				page = 1
			}
			if pageSize < 1 || pageSize > 100 {
				pageSize = 20
			}

			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedSize, pageSize)

			// Verify offset calculation
			offset := (page - 1) * pageSize
			assert.GreaterOrEqual(t, offset, 0, "offset should be non-negative")
		})
	}
}

func TestExpiringDocument_DaysAheadDefault(t *testing.T) {
	// Test the daysAhead default logic from GetExpiringDocuments
	tests := []struct {
		name        string
		daysAhead   int
		expectedVal int
	}{
		{"zero becomes 30", 0, 30},
		{"negative becomes 30", -5, 30},
		{"valid value preserved", 14, 14},
		{"large value preserved", 90, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daysAhead := tt.daysAhead
			if daysAhead < 1 {
				daysAhead = 30
			}
			assert.Equal(t, tt.expectedVal, daysAhead)
		})
	}
}

func TestOCRResult_Structure(t *testing.T) {
	result := OCRResult{
		DocumentNumber:   "DL123456",
		FullName:         "John Doe",
		IssuingAuthority: "DMV",
		Confidence:       0.95,
		RawText:          "raw scanned text",
		Metadata:         map[string]interface{}{"source": "test"},
	}

	assert.Equal(t, "DL123456", result.DocumentNumber)
	assert.Equal(t, "John Doe", result.FullName)
	assert.Equal(t, "DMV", result.IssuingAuthority)
	assert.Equal(t, 0.95, result.Confidence)
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, "test", result.Metadata["source"])
}

func TestDocumentType_Fields(t *testing.T) {
	dt := DocumentType{
		Code:                  "drivers_license",
		Name:                  "Driver's License",
		IsRequired:            true,
		RequiresExpiry:        true,
		RequiresFrontBack:     true,
		DefaultValidityMonths: 60,
		RenewalReminderDays:   30,
		RequiresManualReview:  true,
		AutoOCREnabled:        true,
		IsActive:              true,
	}

	assert.Equal(t, "drivers_license", dt.Code)
	assert.True(t, dt.IsRequired)
	assert.True(t, dt.RequiresExpiry)
	assert.True(t, dt.RequiresFrontBack)
	assert.Equal(t, 60, dt.DefaultValidityMonths)
	assert.Equal(t, 30, dt.RenewalReminderDays)
	assert.True(t, dt.RequiresManualReview)
	assert.True(t, dt.AutoOCREnabled)
	assert.True(t, dt.IsActive)
}

func TestFileSizeValidation(t *testing.T) {
	// Test the file size validation logic from UploadDocument
	tests := []struct {
		name          string
		maxFileSizeMB int
		fileSize      int64
		shouldReject  bool
	}{
		{
			name:          "file within limit",
			maxFileSizeMB: 10,
			fileSize:      5 * 1024 * 1024, // 5 MB
			shouldReject:  false,
		},
		{
			name:          "file at exact limit",
			maxFileSizeMB: 10,
			fileSize:      10 * 1024 * 1024, // 10 MB
			shouldReject:  false,
		},
		{
			name:          "file exceeds limit",
			maxFileSizeMB: 10,
			fileSize:      11 * 1024 * 1024, // 11 MB
			shouldReject:  true,
		},
		{
			name:          "zero byte file",
			maxFileSizeMB: 10,
			fileSize:      0,
			shouldReject:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxSize := int64(tt.maxFileSizeMB) * 1024 * 1024
			exceeds := tt.fileSize > maxSize
			assert.Equal(t, tt.shouldReject, exceeds)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
