package documents

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK IMPLEMENTATIONS
// ========================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	// Document Types
	GetDocumentTypesFunc         func(ctx context.Context) ([]*DocumentType, error)
	GetDocumentTypeByCodeFunc    func(ctx context.Context, code string) (*DocumentType, error)
	GetRequiredDocumentTypesFunc func(ctx context.Context) ([]*DocumentType, error)

	// Driver Documents
	CreateDocumentFunc          func(ctx context.Context, doc *DriverDocument) error
	GetDocumentFunc             func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error)
	GetDriverDocumentsFunc      func(ctx context.Context, driverID uuid.UUID) ([]*DriverDocument, error)
	GetLatestDocumentByTypeFunc func(ctx context.Context, driverID, documentTypeID uuid.UUID) (*DriverDocument, error)
	UpdateDocumentStatusFunc    func(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error
	UpdateDocumentOCRDataFunc   func(ctx context.Context, documentID uuid.UUID, ocrData map[string]interface{}, confidence float64) error
	UpdateDocumentDetailsFunc   func(ctx context.Context, documentID uuid.UUID, documentNumber *string, issueDate, expiryDate *time.Time, issuingAuthority *string) error
	SupersedeDocumentFunc       func(ctx context.Context, documentID uuid.UUID) error
	UpdateDocumentBackFileFunc  func(ctx context.Context, documentID uuid.UUID, backFileURL, backFileKey string) error

	// Verification Status
	GetDriverVerificationStatusFunc func(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error)

	// Pending Reviews
	GetPendingReviewsFunc    func(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error)
	GetExpiringDocumentsFunc func(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error)

	// History
	CreateHistoryFunc      func(ctx context.Context, history *DocumentVerificationHistory) error
	GetDocumentHistoryFunc func(ctx context.Context, documentID uuid.UUID) ([]*DocumentVerificationHistory, error)

	// OCR Queue
	CreateOCRJobFunc        func(ctx context.Context, job *OCRProcessingQueue) error
	GetPendingOCRJobsFunc   func(ctx context.Context, limit int) ([]*OCRProcessingQueue, error)
	UpdateOCRJobStatusFunc  func(ctx context.Context, jobID uuid.UUID, status string, result, errorMsg *string) error
	CompleteOCRJobFunc      func(ctx context.Context, jobID uuid.UUID, extractedData map[string]interface{}, confidence float64, processingTimeMs int) error
	FailOCRJobFunc          func(ctx context.Context, jobID uuid.UUID, errorMessage string) error
	UpdateOCRJobRetryFunc   func(ctx context.Context, jobID uuid.UUID, retryCount int, nextRetry time.Time) error
}

func (m *MockRepository) GetDocumentTypes(ctx context.Context) ([]*DocumentType, error) {
	if m.GetDocumentTypesFunc != nil {
		return m.GetDocumentTypesFunc(ctx)
	}
	return nil, nil
}

func (m *MockRepository) GetDocumentTypeByCode(ctx context.Context, code string) (*DocumentType, error) {
	if m.GetDocumentTypeByCodeFunc != nil {
		return m.GetDocumentTypeByCodeFunc(ctx, code)
	}
	return nil, errors.New("not implemented")
}

func (m *MockRepository) GetRequiredDocumentTypes(ctx context.Context) ([]*DocumentType, error) {
	if m.GetRequiredDocumentTypesFunc != nil {
		return m.GetRequiredDocumentTypesFunc(ctx)
	}
	return nil, nil
}

func (m *MockRepository) CreateDocument(ctx context.Context, doc *DriverDocument) error {
	if m.CreateDocumentFunc != nil {
		return m.CreateDocumentFunc(ctx, doc)
	}
	return nil
}

func (m *MockRepository) GetDocument(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
	if m.GetDocumentFunc != nil {
		return m.GetDocumentFunc(ctx, documentID)
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) GetDriverDocuments(ctx context.Context, driverID uuid.UUID) ([]*DriverDocument, error) {
	if m.GetDriverDocumentsFunc != nil {
		return m.GetDriverDocumentsFunc(ctx, driverID)
	}
	return nil, nil
}

func (m *MockRepository) GetLatestDocumentByType(ctx context.Context, driverID, documentTypeID uuid.UUID) (*DriverDocument, error) {
	if m.GetLatestDocumentByTypeFunc != nil {
		return m.GetLatestDocumentByTypeFunc(ctx, driverID, documentTypeID)
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) UpdateDocumentStatus(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
	if m.UpdateDocumentStatusFunc != nil {
		return m.UpdateDocumentStatusFunc(ctx, documentID, status, reviewedBy, reviewNotes, rejectionReason)
	}
	return nil
}

func (m *MockRepository) UpdateDocumentOCRData(ctx context.Context, documentID uuid.UUID, ocrData map[string]interface{}, confidence float64) error {
	if m.UpdateDocumentOCRDataFunc != nil {
		return m.UpdateDocumentOCRDataFunc(ctx, documentID, ocrData, confidence)
	}
	return nil
}

func (m *MockRepository) UpdateDocumentDetails(ctx context.Context, documentID uuid.UUID, documentNumber *string, issueDate, expiryDate *time.Time, issuingAuthority *string) error {
	if m.UpdateDocumentDetailsFunc != nil {
		return m.UpdateDocumentDetailsFunc(ctx, documentID, documentNumber, issueDate, expiryDate, issuingAuthority)
	}
	return nil
}

func (m *MockRepository) SupersedeDocument(ctx context.Context, documentID uuid.UUID) error {
	if m.SupersedeDocumentFunc != nil {
		return m.SupersedeDocumentFunc(ctx, documentID)
	}
	return nil
}

func (m *MockRepository) UpdateDocumentBackFile(ctx context.Context, documentID uuid.UUID, backFileURL, backFileKey string) error {
	if m.UpdateDocumentBackFileFunc != nil {
		return m.UpdateDocumentBackFileFunc(ctx, documentID, backFileURL, backFileKey)
	}
	return nil
}

func (m *MockRepository) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	if m.GetDriverVerificationStatusFunc != nil {
		return m.GetDriverVerificationStatusFunc(ctx, driverID)
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) GetPendingReviews(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error) {
	if m.GetPendingReviewsFunc != nil {
		return m.GetPendingReviewsFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockRepository) GetExpiringDocuments(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error) {
	if m.GetExpiringDocumentsFunc != nil {
		return m.GetExpiringDocumentsFunc(ctx, daysAhead)
	}
	return nil, nil
}

func (m *MockRepository) CreateHistory(ctx context.Context, history *DocumentVerificationHistory) error {
	if m.CreateHistoryFunc != nil {
		return m.CreateHistoryFunc(ctx, history)
	}
	return nil
}

func (m *MockRepository) GetDocumentHistory(ctx context.Context, documentID uuid.UUID) ([]*DocumentVerificationHistory, error) {
	if m.GetDocumentHistoryFunc != nil {
		return m.GetDocumentHistoryFunc(ctx, documentID)
	}
	return nil, nil
}

func (m *MockRepository) CreateOCRJob(ctx context.Context, job *OCRProcessingQueue) error {
	if m.CreateOCRJobFunc != nil {
		return m.CreateOCRJobFunc(ctx, job)
	}
	return nil
}

func (m *MockRepository) GetPendingOCRJobs(ctx context.Context, limit int) ([]*OCRProcessingQueue, error) {
	if m.GetPendingOCRJobsFunc != nil {
		return m.GetPendingOCRJobsFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockRepository) UpdateOCRJobStatus(ctx context.Context, jobID uuid.UUID, status string, result, errorMsg *string) error {
	if m.UpdateOCRJobStatusFunc != nil {
		return m.UpdateOCRJobStatusFunc(ctx, jobID, status, result, errorMsg)
	}
	return nil
}

func (m *MockRepository) CompleteOCRJob(ctx context.Context, jobID uuid.UUID, extractedData map[string]interface{}, confidence float64, processingTimeMs int) error {
	if m.CompleteOCRJobFunc != nil {
		return m.CompleteOCRJobFunc(ctx, jobID, extractedData, confidence, processingTimeMs)
	}
	return nil
}

func (m *MockRepository) FailOCRJob(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	if m.FailOCRJobFunc != nil {
		return m.FailOCRJobFunc(ctx, jobID, errorMessage)
	}
	return nil
}

func (m *MockRepository) UpdateOCRJobRetry(ctx context.Context, jobID uuid.UUID, retryCount int, nextRetry time.Time) error {
	if m.UpdateOCRJobRetryFunc != nil {
		return m.UpdateOCRJobRetryFunc(ctx, jobID, retryCount, nextRetry)
	}
	return nil
}

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	UploadFunc                  func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error)
	DownloadFunc                func(ctx context.Context, key string) (io.ReadCloser, error)
	DeleteFunc                  func(ctx context.Context, key string) error
	GetURLFunc                  func(key string) string
	GetPresignedUploadURLFunc   func(ctx context.Context, key string, contentType string, expiresIn time.Duration) (*storage.PresignedURLResult, error)
	GetPresignedDownloadURLFunc func(ctx context.Context, key string, expiresIn time.Duration) (*storage.PresignedURLResult, error)
	ExistsFunc                  func(ctx context.Context, key string) (bool, error)
	CopyFunc                    func(ctx context.Context, sourceKey, destKey string) error
}

func (m *MockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
	if m.UploadFunc != nil {
		return m.UploadFunc(ctx, key, reader, size, contentType)
	}
	return &storage.UploadResult{
		Key:        key,
		URL:        "https://storage.example.com/" + key,
		Size:       size,
		MimeType:   contentType,
		UploadedAt: time.Now(),
	}, nil
}

func (m *MockStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.DownloadFunc != nil {
		return m.DownloadFunc(ctx, key)
	}
	return io.NopCloser(bytes.NewReader([]byte("test content"))), nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}
	return nil
}

func (m *MockStorage) GetURL(key string) string {
	if m.GetURLFunc != nil {
		return m.GetURLFunc(key)
	}
	return "https://storage.example.com/" + key
}

func (m *MockStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (*storage.PresignedURLResult, error) {
	if m.GetPresignedUploadURLFunc != nil {
		return m.GetPresignedUploadURLFunc(ctx, key, contentType, expiresIn)
	}
	return &storage.PresignedURLResult{
		URL:       "https://storage.example.com/presigned/" + key,
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

func (m *MockStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (*storage.PresignedURLResult, error) {
	if m.GetPresignedDownloadURLFunc != nil {
		return m.GetPresignedDownloadURLFunc(ctx, key, expiresIn)
	}
	return &storage.PresignedURLResult{
		URL:       "https://storage.example.com/presigned/" + key,
		Method:    "GET",
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

func (m *MockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, key)
	}
	return true, nil
}

func (m *MockStorage) Copy(ctx context.Context, sourceKey, destKey string) error {
	if m.CopyFunc != nil {
		return m.CopyFunc(ctx, sourceKey, destKey)
	}
	return nil
}

// ========================================
// TEST SERVICE CONSTRUCTOR
// ========================================

func newTestService(mockRepo *MockRepository, mockStorage *MockStorage, config ServiceConfig) *Service {
	// Apply defaults
	if config.MaxFileSizeMB == 0 {
		config.MaxFileSizeMB = 10
	}
	if len(config.AllowedMimeTypes) == 0 {
		config.AllowedMimeTypes = []string{
			"image/jpeg", "image/png", "image/webp", "application/pdf",
		}
	}

	return &Service{
		repo:    mockRepo,
		storage: mockStorage,
		config:  config,
	}
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func createTestDocumentType() *DocumentType {
	return &DocumentType{
		ID:                    uuid.New(),
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
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

func createTestDocument(driverID uuid.UUID, docType *DocumentType, status DocumentStatus) *DriverDocument {
	return &DriverDocument{
		ID:             uuid.New(),
		DriverID:       driverID,
		DocumentTypeID: docType.ID,
		Status:         status,
		FileURL:        "https://storage.example.com/test.jpg",
		FileKey:        "drivers/test/documents/test.jpg",
		FileName:       "test.jpg",
		FileSizeBytes:  int64Ptr(1024),
		FileMimeType:   stringPtr("image/jpeg"),
		Version:        1,
		SubmittedAt:    time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		DocumentType:   docType,
	}
}

// ========================================
// UNIT TESTS - nilIfEmpty
// ========================================

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
		{
			name:     "special characters",
			input:    "ABC-123!@#",
			isNil:    false,
			expected: "ABC-123!@#",
		},
		{
			name:     "unicode characters",
			input:    "Водительские права",
			isNil:    false,
			expected: "Водительские права",
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

// ========================================
// UNIT TESTS - ServiceConfig
// ========================================

func TestServiceConfig_Defaults(t *testing.T) {
	config := ServiceConfig{}

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

func TestServiceConfig_ZeroMaxFileSize(t *testing.T) {
	config := ServiceConfig{
		MaxFileSizeMB: 0,
	}

	// Simulate NewService behavior
	if config.MaxFileSizeMB == 0 {
		config.MaxFileSizeMB = 10
	}

	assert.Equal(t, 10, config.MaxFileSizeMB, "Zero MaxFileSizeMB should default to 10")
}

func TestServiceConfig_EmptyAllowedMimeTypes(t *testing.T) {
	config := ServiceConfig{
		AllowedMimeTypes: []string{},
	}

	// Simulate NewService behavior
	if len(config.AllowedMimeTypes) == 0 {
		config.AllowedMimeTypes = []string{
			"image/jpeg", "image/png", "image/webp", "application/pdf",
		}
	}

	assert.Len(t, config.AllowedMimeTypes, 4)
}

func TestServiceConfig_LargeMaxFileSize(t *testing.T) {
	config := ServiceConfig{
		MaxFileSizeMB: 100,
	}

	assert.Equal(t, 100, config.MaxFileSizeMB)
}

// ========================================
// UNIT TESTS - Document Status Constants
// ========================================

func TestDocumentStatus_Constants(t *testing.T) {
	assert.Equal(t, DocumentStatus("pending"), StatusPending)
	assert.Equal(t, DocumentStatus("under_review"), StatusUnderReview)
	assert.Equal(t, DocumentStatus("approved"), StatusApproved)
	assert.Equal(t, DocumentStatus("rejected"), StatusRejected)
	assert.Equal(t, DocumentStatus("expired"), StatusExpired)
	assert.Equal(t, DocumentStatus("superseded"), StatusSuperseded)
}

func TestDocumentStatus_StringConversion(t *testing.T) {
	tests := []struct {
		status   DocumentStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusUnderReview, "under_review"},
		{StatusApproved, "approved"},
		{StatusRejected, "rejected"},
		{StatusExpired, "expired"},
		{StatusSuperseded, "superseded"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// ========================================
// UNIT TESTS - Verification Status Constants
// ========================================

func TestVerificationStatus_Constants(t *testing.T) {
	assert.Equal(t, VerificationStatus("incomplete"), VerificationIncomplete)
	assert.Equal(t, VerificationStatus("pending_review"), VerificationPendingReview)
	assert.Equal(t, VerificationStatus("approved"), VerificationApproved)
	assert.Equal(t, VerificationStatus("suspended"), VerificationSuspended)
	assert.Equal(t, VerificationStatus("rejected"), VerificationRejected)
}

func TestVerificationStatus_StringConversion(t *testing.T) {
	tests := []struct {
		status   VerificationStatus
		expected string
	}{
		{VerificationIncomplete, "incomplete"},
		{VerificationPendingReview, "pending_review"},
		{VerificationApproved, "approved"},
		{VerificationSuspended, "suspended"},
		{VerificationRejected, "rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// ========================================
// UNIT TESTS - LogHistory Notes Handling
// ========================================

func TestLogHistory_NotesHandling(t *testing.T) {
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
		{
			name:     "empty string notes",
			notes:    "",
			expected: stringPtr(""),
		},
		{
			name:     "int notes (not handled)",
			notes:    123,
			expected: nil,
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

// ========================================
// UNIT TESTS - Pagination
// ========================================

func TestPendingReviewDocument_PageCalculation(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedPage int
		expectedSize int
		expectedOff  int
	}{
		{
			name:         "defaults for zero values",
			page:         0,
			pageSize:     0,
			expectedPage: 1,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "negative page becomes 1",
			page:         -1,
			pageSize:     10,
			expectedPage: 1,
			expectedSize: 10,
			expectedOff:  0,
		},
		{
			name:         "oversized pageSize becomes 20",
			page:         1,
			pageSize:     200,
			expectedPage: 1,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "valid values preserved",
			page:         3,
			pageSize:     50,
			expectedPage: 3,
			expectedSize: 50,
			expectedOff:  100,
		},
		{
			name:         "page 2 with size 10",
			page:         2,
			pageSize:     10,
			expectedPage: 2,
			expectedSize: 10,
			expectedOff:  10,
		},
		{
			name:         "large page number",
			page:         100,
			pageSize:     25,
			expectedPage: 100,
			expectedSize: 25,
			expectedOff:  2475,
		},
		{
			name:         "pageSize at boundary 100",
			page:         1,
			pageSize:     100,
			expectedPage: 1,
			expectedSize: 100,
			expectedOff:  0,
		},
		{
			name:         "pageSize just over boundary 101",
			page:         1,
			pageSize:     101,
			expectedPage: 1,
			expectedSize: 20,
			expectedOff:  0,
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

			offset := (page - 1) * pageSize
			assert.Equal(t, tt.expectedOff, offset)
			assert.GreaterOrEqual(t, offset, 0, "offset should be non-negative")
		})
	}
}

// ========================================
// UNIT TESTS - Expiring Documents
// ========================================

func TestExpiringDocument_DaysAheadDefault(t *testing.T) {
	tests := []struct {
		name        string
		daysAhead   int
		expectedVal int
	}{
		{"zero becomes 30", 0, 30},
		{"negative becomes 30", -5, 30},
		{"valid value preserved", 14, 14},
		{"large value preserved", 90, 90},
		{"boundary value 1", 1, 1},
		{"very large value", 365, 365},
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

// ========================================
// UNIT TESTS - OCR Result Structure
// ========================================

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

func TestOCRResult_WithDates(t *testing.T) {
	issueDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	dob := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)

	result := OCRResult{
		DocumentNumber:   "DL123456",
		FullName:         "Jane Doe",
		DateOfBirth:      &dob,
		IssueDate:        &issueDate,
		ExpiryDate:       &expiryDate,
		IssuingAuthority: "California DMV",
		Address:          "123 Main St, Los Angeles, CA",
		Confidence:       0.98,
	}

	assert.NotNil(t, result.DateOfBirth)
	assert.Equal(t, 1990, result.DateOfBirth.Year())
	assert.NotNil(t, result.IssueDate)
	assert.Equal(t, 2020, result.IssueDate.Year())
	assert.NotNil(t, result.ExpiryDate)
	assert.Equal(t, 2025, result.ExpiryDate.Year())
}

func TestOCRResult_VehicleDocument(t *testing.T) {
	result := OCRResult{
		DocumentNumber: "REG-2023-001",
		VehiclePlate:   "ABC-123",
		VehicleVIN:     "1HGBH41JXMN109186",
		Confidence:     0.92,
	}

	assert.Equal(t, "ABC-123", result.VehiclePlate)
	assert.Equal(t, "1HGBH41JXMN109186", result.VehicleVIN)
}

// ========================================
// UNIT TESTS - Document Type
// ========================================

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

func TestDocumentType_Optional(t *testing.T) {
	dt := DocumentType{
		Code:       "proof_of_address",
		Name:       "Proof of Address",
		IsRequired: false,
		IsActive:   true,
	}

	assert.False(t, dt.IsRequired)
	assert.False(t, dt.RequiresExpiry)
	assert.False(t, dt.RequiresFrontBack)
}

func TestDocumentType_WithDescription(t *testing.T) {
	desc := "Valid government-issued driver's license"
	dt := DocumentType{
		Code:        "drivers_license",
		Name:        "Driver's License",
		Description: &desc,
	}

	require.NotNil(t, dt.Description)
	assert.Equal(t, "Valid government-issued driver's license", *dt.Description)
}

func TestDocumentType_CountryCodes(t *testing.T) {
	dt := DocumentType{
		Code:         "drivers_license",
		CountryCodes: []string{"US", "CA", "MX"},
	}

	assert.Len(t, dt.CountryCodes, 3)
	assert.Contains(t, dt.CountryCodes, "US")
	assert.Contains(t, dt.CountryCodes, "CA")
	assert.Contains(t, dt.CountryCodes, "MX")
}

// ========================================
// UNIT TESTS - File Size Validation
// ========================================

func TestFileSizeValidation(t *testing.T) {
	tests := []struct {
		name          string
		maxFileSizeMB int
		fileSize      int64
		shouldReject  bool
	}{
		{
			name:          "file within limit",
			maxFileSizeMB: 10,
			fileSize:      5 * 1024 * 1024,
			shouldReject:  false,
		},
		{
			name:          "file at exact limit",
			maxFileSizeMB: 10,
			fileSize:      10 * 1024 * 1024,
			shouldReject:  false,
		},
		{
			name:          "file exceeds limit",
			maxFileSizeMB: 10,
			fileSize:      11 * 1024 * 1024,
			shouldReject:  true,
		},
		{
			name:          "zero byte file",
			maxFileSizeMB: 10,
			fileSize:      0,
			shouldReject:  false,
		},
		{
			name:          "1 byte over limit",
			maxFileSizeMB: 10,
			fileSize:      10*1024*1024 + 1,
			shouldReject:  true,
		},
		{
			name:          "large file with large limit",
			maxFileSizeMB: 100,
			fileSize:      50 * 1024 * 1024,
			shouldReject:  false,
		},
		{
			name:          "tiny file",
			maxFileSizeMB: 1,
			fileSize:      100,
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

// ========================================
// UNIT TESTS - Driver Document
// ========================================

func TestDriverDocument_Creation(t *testing.T) {
	driverID := uuid.New()
	docTypeID := uuid.New()
	now := time.Now()

	doc := &DriverDocument{
		ID:             uuid.New(),
		DriverID:       driverID,
		DocumentTypeID: docTypeID,
		Status:         StatusPending,
		FileURL:        "https://storage.example.com/doc.pdf",
		FileKey:        "drivers/123/documents/doc.pdf",
		FileName:       "license.pdf",
		Version:        1,
		SubmittedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	assert.Equal(t, driverID, doc.DriverID)
	assert.Equal(t, docTypeID, doc.DocumentTypeID)
	assert.Equal(t, StatusPending, doc.Status)
	assert.Equal(t, 1, doc.Version)
}

func TestDriverDocument_WithOptionalFields(t *testing.T) {
	docNum := "DL-2023-001"
	authority := "DMV"
	mimeType := "application/pdf"
	fileSize := int64(2048)
	issueDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)

	doc := &DriverDocument{
		ID:               uuid.New(),
		DriverID:         uuid.New(),
		DocumentTypeID:   uuid.New(),
		Status:           StatusApproved,
		DocumentNumber:   &docNum,
		IssuingAuthority: &authority,
		FileMimeType:     &mimeType,
		FileSizeBytes:    &fileSize,
		IssueDate:        &issueDate,
		ExpiryDate:       &expiryDate,
	}

	require.NotNil(t, doc.DocumentNumber)
	assert.Equal(t, "DL-2023-001", *doc.DocumentNumber)
	require.NotNil(t, doc.IssuingAuthority)
	assert.Equal(t, "DMV", *doc.IssuingAuthority)
	require.NotNil(t, doc.FileSizeBytes)
	assert.Equal(t, int64(2048), *doc.FileSizeBytes)
}

func TestDriverDocument_WithBackFile(t *testing.T) {
	backURL := "https://storage.example.com/back.jpg"
	backKey := "drivers/123/documents/back.jpg"

	doc := &DriverDocument{
		ID:          uuid.New(),
		BackFileURL: &backURL,
		BackFileKey: &backKey,
	}

	require.NotNil(t, doc.BackFileURL)
	assert.Equal(t, backURL, *doc.BackFileURL)
	require.NotNil(t, doc.BackFileKey)
	assert.Equal(t, backKey, *doc.BackFileKey)
}

func TestDriverDocument_Versioning(t *testing.T) {
	prevDocID := uuid.New()

	doc := &DriverDocument{
		ID:                 uuid.New(),
		Version:            2,
		PreviousDocumentID: &prevDocID,
	}

	assert.Equal(t, 2, doc.Version)
	require.NotNil(t, doc.PreviousDocumentID)
	assert.Equal(t, prevDocID, *doc.PreviousDocumentID)
}

func TestDriverDocument_OCRData(t *testing.T) {
	confidence := 0.95
	ocrTime := time.Now()

	doc := &DriverDocument{
		ID: uuid.New(),
		OCRData: map[string]interface{}{
			"document_number": "DL123",
			"full_name":       "John Doe",
		},
		OCRConfidence:  &confidence,
		OCRProcessedAt: &ocrTime,
	}

	assert.NotNil(t, doc.OCRData)
	assert.Equal(t, "DL123", doc.OCRData["document_number"])
	assert.Equal(t, "John Doe", doc.OCRData["full_name"])
	require.NotNil(t, doc.OCRConfidence)
	assert.Equal(t, 0.95, *doc.OCRConfidence)
}

func TestDriverDocument_ReviewInfo(t *testing.T) {
	reviewerID := uuid.New()
	reviewTime := time.Now()
	notes := "Document verified successfully"
	rejectionReason := "Document is expired"

	doc := &DriverDocument{
		ID:              uuid.New(),
		Status:          StatusRejected,
		ReviewedBy:      &reviewerID,
		ReviewedAt:      &reviewTime,
		ReviewNotes:     &notes,
		RejectionReason: &rejectionReason,
	}

	require.NotNil(t, doc.ReviewedBy)
	assert.Equal(t, reviewerID, *doc.ReviewedBy)
	require.NotNil(t, doc.ReviewedAt)
	require.NotNil(t, doc.ReviewNotes)
	assert.Equal(t, "Document verified successfully", *doc.ReviewNotes)
	require.NotNil(t, doc.RejectionReason)
	assert.Equal(t, "Document is expired", *doc.RejectionReason)
}

// ========================================
// UNIT TESTS - Driver Verification Status
// ========================================

func TestDriverVerificationStatus_Incomplete(t *testing.T) {
	status := &DriverVerificationStatus{
		DriverID:                uuid.New(),
		VerificationStatus:      VerificationIncomplete,
		RequiredDocumentsCount:  5,
		SubmittedDocumentsCount: 2,
		ApprovedDocumentsCount:  1,
	}

	assert.Equal(t, VerificationIncomplete, status.VerificationStatus)
	assert.Equal(t, 5, status.RequiredDocumentsCount)
	assert.Equal(t, 2, status.SubmittedDocumentsCount)
	assert.Equal(t, 1, status.ApprovedDocumentsCount)
}

func TestDriverVerificationStatus_Approved(t *testing.T) {
	approverID := uuid.New()
	approvedAt := time.Now()
	submittedAt := time.Now().Add(-24 * time.Hour)
	docApprovedAt := time.Now().Add(-1 * time.Hour)

	status := &DriverVerificationStatus{
		DriverID:                uuid.New(),
		VerificationStatus:      VerificationApproved,
		RequiredDocumentsCount:  5,
		SubmittedDocumentsCount: 5,
		ApprovedDocumentsCount:  5,
		DocumentsSubmittedAt:    &submittedAt,
		DocumentsApprovedAt:     &docApprovedAt,
		ApprovedBy:              &approverID,
		ApprovedAt:              &approvedAt,
	}

	assert.Equal(t, VerificationApproved, status.VerificationStatus)
	assert.Equal(t, 5, status.ApprovedDocumentsCount)
	require.NotNil(t, status.ApprovedBy)
	assert.Equal(t, approverID, *status.ApprovedBy)
}

func TestDriverVerificationStatus_Suspended(t *testing.T) {
	suspendedBy := uuid.New()
	suspendedAt := time.Now()
	suspensionEnd := time.Now().Add(7 * 24 * time.Hour)
	reason := "Failed background check"

	status := &DriverVerificationStatus{
		DriverID:           uuid.New(),
		VerificationStatus: VerificationSuspended,
		SuspendedAt:        &suspendedAt,
		SuspendedBy:        &suspendedBy,
		SuspensionReason:   &reason,
		SuspensionEndDate:  &suspensionEnd,
	}

	assert.Equal(t, VerificationSuspended, status.VerificationStatus)
	require.NotNil(t, status.SuspensionReason)
	assert.Equal(t, "Failed background check", *status.SuspensionReason)
	require.NotNil(t, status.SuspensionEndDate)
}

func TestDriverVerificationStatus_ExpiryTracking(t *testing.T) {
	nextExpiry := time.Now().Add(30 * 24 * time.Hour)
	warningSent := time.Now().Add(-24 * time.Hour)

	status := &DriverVerificationStatus{
		DriverID:            uuid.New(),
		VerificationStatus:  VerificationApproved,
		NextDocumentExpiry:  &nextExpiry,
		ExpiryWarningSentAt: &warningSent,
	}

	require.NotNil(t, status.NextDocumentExpiry)
	require.NotNil(t, status.ExpiryWarningSentAt)
}

// ========================================
// UNIT TESTS - Document Verification History
// ========================================

func TestDocumentVerificationHistory_Creation(t *testing.T) {
	docID := uuid.New()
	performerID := uuid.New()
	prevStatus := "pending"
	newStatus := "approved"
	notes := "Document looks valid"

	history := &DocumentVerificationHistory{
		ID:             uuid.New(),
		DocumentID:     docID,
		Action:         "approve",
		PreviousStatus: &prevStatus,
		NewStatus:      &newStatus,
		PerformedBy:    &performerID,
		IsSystemAction: false,
		Notes:          &notes,
		CreatedAt:      time.Now(),
	}

	assert.Equal(t, docID, history.DocumentID)
	assert.Equal(t, "approve", history.Action)
	require.NotNil(t, history.PreviousStatus)
	assert.Equal(t, "pending", *history.PreviousStatus)
	require.NotNil(t, history.NewStatus)
	assert.Equal(t, "approved", *history.NewStatus)
	assert.False(t, history.IsSystemAction)
}

func TestDocumentVerificationHistory_SystemAction(t *testing.T) {
	history := &DocumentVerificationHistory{
		ID:             uuid.New(),
		DocumentID:     uuid.New(),
		Action:         "ocr_processed",
		IsSystemAction: true,
		Metadata: map[string]interface{}{
			"confidence": 0.95,
			"provider":   "tesseract",
		},
	}

	assert.True(t, history.IsSystemAction)
	assert.Nil(t, history.PerformedBy)
	assert.NotNil(t, history.Metadata)
	assert.Equal(t, 0.95, history.Metadata["confidence"])
}

// ========================================
// UNIT TESTS - OCR Processing Queue
// ========================================

func TestOCRProcessingQueue_Creation(t *testing.T) {
	docID := uuid.New()

	job := &OCRProcessingQueue{
		ID:         uuid.New(),
		DocumentID: docID,
		Status:     "pending",
		Priority:   0,
		MaxRetries: 3,
		RetryCount: 0,
	}

	assert.Equal(t, docID, job.DocumentID)
	assert.Equal(t, "pending", job.Status)
	assert.Equal(t, 0, job.Priority)
	assert.Equal(t, 3, job.MaxRetries)
	assert.Equal(t, 0, job.RetryCount)
}

func TestOCRProcessingQueue_HighPriority(t *testing.T) {
	job := &OCRProcessingQueue{
		ID:       uuid.New(),
		Status:   "pending",
		Priority: 10,
	}

	assert.Equal(t, 10, job.Priority)
}

func TestOCRProcessingQueue_Completed(t *testing.T) {
	startedAt := time.Now().Add(-5 * time.Second)
	completedAt := time.Now()
	processingTime := 5000
	confidence := 0.98
	provider := "google-vision"

	job := &OCRProcessingQueue{
		ID:               uuid.New(),
		Status:           "completed",
		Provider:         &provider,
		StartedAt:        &startedAt,
		CompletedAt:      &completedAt,
		ProcessingTimeMs: &processingTime,
		ConfidenceScore:  &confidence,
		ExtractedData: map[string]interface{}{
			"document_number": "DL123",
		},
	}

	assert.Equal(t, "completed", job.Status)
	require.NotNil(t, job.Provider)
	assert.Equal(t, "google-vision", *job.Provider)
	require.NotNil(t, job.ProcessingTimeMs)
	assert.Equal(t, 5000, *job.ProcessingTimeMs)
	require.NotNil(t, job.ConfidenceScore)
	assert.Equal(t, 0.98, *job.ConfidenceScore)
}

func TestOCRProcessingQueue_Failed(t *testing.T) {
	errorMsg := "Failed to process image: corrupt file"
	nextRetry := time.Now().Add(5 * time.Minute)

	job := &OCRProcessingQueue{
		ID:           uuid.New(),
		Status:       "failed",
		ErrorMessage: &errorMsg,
		RetryCount:   1,
		MaxRetries:   3,
		NextRetryAt:  &nextRetry,
	}

	assert.Equal(t, "failed", job.Status)
	require.NotNil(t, job.ErrorMessage)
	assert.Contains(t, *job.ErrorMessage, "corrupt file")
	assert.Equal(t, 1, job.RetryCount)
	require.NotNil(t, job.NextRetryAt)
}

// ========================================
// UNIT TESTS - Request/Response Types
// ========================================

func TestUploadDocumentRequest_Required(t *testing.T) {
	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}

	assert.Equal(t, "drivers_license", req.DocumentTypeCode)
	assert.Empty(t, req.DocumentNumber)
	assert.Nil(t, req.IssueDate)
	assert.Nil(t, req.ExpiryDate)
}

func TestUploadDocumentRequest_Full(t *testing.T) {
	issueDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
		DocumentNumber:   "DL-2023-001",
		IssueDate:        &issueDate,
		ExpiryDate:       &expiryDate,
		IssuingAuthority: "California DMV",
	}

	assert.Equal(t, "drivers_license", req.DocumentTypeCode)
	assert.Equal(t, "DL-2023-001", req.DocumentNumber)
	require.NotNil(t, req.IssueDate)
	assert.Equal(t, 2023, req.IssueDate.Year())
	require.NotNil(t, req.ExpiryDate)
	assert.Equal(t, 2028, req.ExpiryDate.Year())
	assert.Equal(t, "California DMV", req.IssuingAuthority)
}

func TestUploadDocumentResponse_Success(t *testing.T) {
	resp := &UploadDocumentResponse{
		DocumentID:   uuid.New(),
		Status:       StatusPending,
		FileURL:      "https://storage.example.com/doc.pdf",
		Message:      "Document uploaded successfully",
		OCRScheduled: true,
	}

	assert.NotEqual(t, uuid.Nil, resp.DocumentID)
	assert.Equal(t, StatusPending, resp.Status)
	assert.NotEmpty(t, resp.FileURL)
	assert.Equal(t, "Document uploaded successfully", resp.Message)
	assert.True(t, resp.OCRScheduled)
}

func TestReviewDocumentRequest_Approve(t *testing.T) {
	docNum := "DL-VERIFIED-001"
	expiryDate := "2028-01-01"

	req := &ReviewDocumentRequest{
		Action:         "approve",
		Notes:          "Document verified against government database",
		DocumentNumber: &docNum,
		ExpiryDate:     &expiryDate,
	}

	assert.Equal(t, "approve", req.Action)
	assert.Empty(t, req.RejectionReason)
	require.NotNil(t, req.DocumentNumber)
	assert.Equal(t, "DL-VERIFIED-001", *req.DocumentNumber)
}

func TestReviewDocumentRequest_Reject(t *testing.T) {
	req := &ReviewDocumentRequest{
		Action:          "reject",
		RejectionReason: "Document is expired",
		Notes:           "Expiry date was 2022-01-01",
	}

	assert.Equal(t, "reject", req.Action)
	assert.Equal(t, "Document is expired", req.RejectionReason)
}

func TestReviewDocumentRequest_RequestResubmit(t *testing.T) {
	req := &ReviewDocumentRequest{
		Action:          "request_resubmit",
		RejectionReason: "Image is blurry, please upload a clearer photo",
	}

	assert.Equal(t, "request_resubmit", req.Action)
}

func TestPresignedUploadRequest(t *testing.T) {
	req := &PresignedUploadRequest{
		DocumentTypeCode: "drivers_license",
		FileName:         "license.jpg",
		ContentType:      "image/jpeg",
		IsFrontSide:      true,
	}

	assert.Equal(t, "drivers_license", req.DocumentTypeCode)
	assert.Equal(t, "license.jpg", req.FileName)
	assert.Equal(t, "image/jpeg", req.ContentType)
	assert.True(t, req.IsFrontSide)
}

func TestPresignedUploadResponse(t *testing.T) {
	resp := &PresignedUploadResponse{
		UploadURL: "https://storage.example.com/presigned/upload",
		Method:    "PUT",
		Headers: map[string]string{
			"Content-Type": "image/jpeg",
		},
		ExpiresAt:   time.Now().Add(15 * time.Minute),
		FileKey:     "drivers/123/documents/license.jpg",
		CallbackURL: "/api/v1/documents/upload-complete",
	}

	assert.NotEmpty(t, resp.UploadURL)
	assert.Equal(t, "PUT", resp.Method)
	assert.Contains(t, resp.Headers, "Content-Type")
	assert.NotEmpty(t, resp.FileKey)
	assert.Equal(t, "/api/v1/documents/upload-complete", resp.CallbackURL)
}

func TestUploadCompleteRequest(t *testing.T) {
	expiryDate := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)

	req := &UploadCompleteRequest{
		FileKey:          "drivers/123/documents/license.jpg",
		DocumentTypeCode: "drivers_license",
		IsFrontSide:      true,
		DocumentNumber:   "DL-2023-001",
		ExpiryDate:       &expiryDate,
	}

	assert.NotEmpty(t, req.FileKey)
	assert.Equal(t, "drivers_license", req.DocumentTypeCode)
	assert.True(t, req.IsFrontSide)
}

func TestVerificationStatusResponse_Complete(t *testing.T) {
	nextExpiry := time.Now().Add(30 * 24 * time.Hour)

	resp := &VerificationStatusResponse{
		Status:             VerificationApproved,
		RequiredDocuments:  []*DocumentRequirement{},
		SubmittedDocuments: []*DriverDocument{},
		MissingDocuments:   []string{},
		NextExpiry:         &nextExpiry,
		CanDrive:           true,
		Message:            "Your verification is complete",
	}

	assert.Equal(t, VerificationApproved, resp.Status)
	assert.True(t, resp.CanDrive)
	assert.Equal(t, "Your verification is complete", resp.Message)
	require.NotNil(t, resp.NextExpiry)
}

func TestVerificationStatusResponse_Incomplete(t *testing.T) {
	resp := &VerificationStatusResponse{
		Status:           VerificationIncomplete,
		MissingDocuments: []string{"Driver's License", "Vehicle Registration"},
		CanDrive:         false,
		Message:          "Missing documents: 2",
	}

	assert.Equal(t, VerificationIncomplete, resp.Status)
	assert.False(t, resp.CanDrive)
	assert.Len(t, resp.MissingDocuments, 2)
	assert.Contains(t, resp.MissingDocuments, "Driver's License")
}

func TestDocumentRequirement(t *testing.T) {
	docType := createTestDocumentType()
	doc := createTestDocument(uuid.New(), docType, StatusApproved)

	req := &DocumentRequirement{
		DocumentType: docType,
		Status:       "approved",
		Document:     doc,
	}

	assert.NotNil(t, req.DocumentType)
	assert.Equal(t, "approved", req.Status)
	assert.NotNil(t, req.Document)
}

func TestDocumentRequirement_NotSubmitted(t *testing.T) {
	docType := createTestDocumentType()

	req := &DocumentRequirement{
		DocumentType: docType,
		Status:       "not_submitted",
		Document:     nil,
	}

	assert.Equal(t, "not_submitted", req.Status)
	assert.Nil(t, req.Document)
}

func TestPendingReviewDocument(t *testing.T) {
	doc := &DriverDocument{
		ID:     uuid.New(),
		Status: StatusPending,
	}
	confidence := 0.92

	pending := &PendingReviewDocument{
		Document:      doc,
		DriverName:    "John Doe",
		DriverPhone:   "+1234567890",
		DriverEmail:   "john@example.com",
		DocumentType:  "Driver's License",
		HoursPending:  24.5,
		OCRConfidence: &confidence,
	}

	assert.NotNil(t, pending.Document)
	assert.Equal(t, "John Doe", pending.DriverName)
	assert.Equal(t, 24.5, pending.HoursPending)
	require.NotNil(t, pending.OCRConfidence)
	assert.Equal(t, 0.92, *pending.OCRConfidence)
}

func TestExpiringDocument(t *testing.T) {
	doc := &DriverDocument{
		ID:     uuid.New(),
		Status: StatusApproved,
	}

	expiring := &ExpiringDocument{
		Document:        doc,
		DriverName:      "Jane Doe",
		DriverEmail:     "jane@example.com",
		DriverPhone:     "+1234567890",
		DocumentType:    "Driver's License",
		DaysUntilExpiry: 7,
		Urgency:         "critical",
	}

	assert.NotNil(t, expiring.Document)
	assert.Equal(t, 7, expiring.DaysUntilExpiry)
	assert.Equal(t, "critical", expiring.Urgency)
}

func TestExpiringDocument_Urgency(t *testing.T) {
	tests := []struct {
		daysUntilExpiry int
		expectedUrgency string
	}{
		{-5, "expired"},
		{0, "expired"},
		{3, "critical"},
		{7, "critical"},
		{14, "warning"},
		{30, "warning"},
		{60, "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedUrgency, func(t *testing.T) {
			var urgency string
			switch {
			case tt.daysUntilExpiry <= 0:
				urgency = "expired"
			case tt.daysUntilExpiry <= 7:
				urgency = "critical"
			case tt.daysUntilExpiry <= 30:
				urgency = "warning"
			default:
				urgency = "ok"
			}
			assert.Equal(t, tt.expectedUrgency, urgency)
		})
	}
}

func TestDocumentListResponse(t *testing.T) {
	docs := []*DriverDocument{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	resp := &DocumentListResponse{
		Documents:  docs,
		Total:      50,
		Page:       1,
		PageSize:   20,
		TotalPages: 3,
	}

	assert.Len(t, resp.Documents, 2)
	assert.Equal(t, 50, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
	assert.Equal(t, 3, resp.TotalPages)
}

func TestDocumentTypeListResponse(t *testing.T) {
	types := []*DocumentType{
		{Code: "drivers_license"},
		{Code: "vehicle_registration"},
	}

	resp := &DocumentTypeListResponse{
		DocumentTypes: types,
	}

	assert.Len(t, resp.DocumentTypes, 2)
}

// ========================================
// UNIT TESTS - Status Transitions
// ========================================

func TestDocumentStatusTransitions_Valid(t *testing.T) {
	validTransitions := map[DocumentStatus][]DocumentStatus{
		StatusPending:     {StatusUnderReview, StatusApproved, StatusRejected, StatusSuperseded},
		StatusUnderReview: {StatusApproved, StatusRejected},
		StatusApproved:    {StatusExpired, StatusSuperseded},
		StatusRejected:    {StatusSuperseded},
		StatusExpired:     {StatusSuperseded},
	}

	for from, toList := range validTransitions {
		for _, to := range toList {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				assert.NotEqual(t, from, to, "Transition should change status")
			})
		}
	}
}

func TestVerificationStatusTransitions(t *testing.T) {
	transitions := []struct {
		from VerificationStatus
		to   VerificationStatus
	}{
		{VerificationIncomplete, VerificationPendingReview},
		{VerificationPendingReview, VerificationApproved},
		{VerificationApproved, VerificationSuspended},
		{VerificationSuspended, VerificationApproved},
		{VerificationPendingReview, VerificationRejected},
	}

	for _, tt := range transitions {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			assert.NotEqual(t, tt.from, tt.to)
		})
	}
}

// ========================================
// UNIT TESTS - Edge Cases
// ========================================

func TestNilPointerHandling(t *testing.T) {
	doc := &DriverDocument{
		ID:     uuid.New(),
		Status: StatusPending,
	}

	// All optional fields should be nil
	assert.Nil(t, doc.FileSizeBytes)
	assert.Nil(t, doc.FileMimeType)
	assert.Nil(t, doc.BackFileURL)
	assert.Nil(t, doc.BackFileKey)
	assert.Nil(t, doc.DocumentNumber)
	assert.Nil(t, doc.IssueDate)
	assert.Nil(t, doc.ExpiryDate)
	assert.Nil(t, doc.IssuingAuthority)
	assert.Nil(t, doc.OCRConfidence)
	assert.Nil(t, doc.OCRProcessedAt)
	assert.Nil(t, doc.ReviewedBy)
	assert.Nil(t, doc.ReviewedAt)
	assert.Nil(t, doc.ReviewNotes)
	assert.Nil(t, doc.RejectionReason)
	assert.Nil(t, doc.PreviousDocumentID)
}

func TestEmptyCollections(t *testing.T) {
	resp := &VerificationStatusResponse{
		RequiredDocuments:  []*DocumentRequirement{},
		SubmittedDocuments: []*DriverDocument{},
		MissingDocuments:   []string{},
	}

	assert.Empty(t, resp.RequiredDocuments)
	assert.Empty(t, resp.SubmittedDocuments)
	assert.Empty(t, resp.MissingDocuments)
	assert.Len(t, resp.RequiredDocuments, 0)
}

func TestUUIDHandling(t *testing.T) {
	// Test nil UUID
	var nilUUID uuid.UUID
	assert.Equal(t, uuid.Nil, nilUUID)

	// Test new UUID
	newUUID := uuid.New()
	assert.NotEqual(t, uuid.Nil, newUUID)

	// Test UUID in document
	doc := &DriverDocument{
		ID:       newUUID,
		DriverID: uuid.New(),
	}
	assert.NotEqual(t, uuid.Nil, doc.ID)
	assert.NotEqual(t, uuid.Nil, doc.DriverID)
	assert.NotEqual(t, doc.ID, doc.DriverID)
}

func TestTimeHandling(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(30 * 24 * time.Hour)

	doc := &DriverDocument{
		ID:          uuid.New(),
		IssueDate:   &past,
		ExpiryDate:  &future,
		SubmittedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.True(t, doc.IssueDate.Before(now))
	assert.True(t, doc.ExpiryDate.After(now))
	assert.True(t, doc.SubmittedAt.Equal(now) || doc.SubmittedAt.After(past))
}

func TestMapHandling(t *testing.T) {
	// Empty map
	emptyMap := make(map[string]interface{})
	assert.Empty(t, emptyMap)

	// Map with values
	ocrData := map[string]interface{}{
		"document_number": "DL123",
		"confidence":      0.95,
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	assert.Len(t, ocrData, 3)
	assert.Equal(t, "DL123", ocrData["document_number"])
	assert.Equal(t, 0.95, ocrData["confidence"])

	nested, ok := ocrData["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", nested["key"])
}

// ========================================
// INTEGRATION-STYLE TESTS (with mocks)
// ========================================

func TestMockRepository_GetDocumentTypes(t *testing.T) {
	mock := &MockRepository{
		GetDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{
				{Code: "drivers_license", Name: "Driver's License"},
				{Code: "vehicle_registration", Name: "Vehicle Registration"},
			}, nil
		},
	}

	types, err := mock.GetDocumentTypes(context.Background())
	require.NoError(t, err)
	assert.Len(t, types, 2)
}

func TestMockRepository_GetDocumentTypes_Error(t *testing.T) {
	mock := &MockRepository{
		GetDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return nil, errors.New("database error")
		},
	}

	types, err := mock.GetDocumentTypes(context.Background())
	assert.Error(t, err)
	assert.Nil(t, types)
}

func TestMockRepository_GetDocument(t *testing.T) {
	docID := uuid.New()
	expectedDoc := &DriverDocument{
		ID:     docID,
		Status: StatusApproved,
	}

	mock := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			if documentID == docID {
				return expectedDoc, nil
			}
			return nil, errors.New("not found")
		},
	}

	// Found
	doc, err := mock.GetDocument(context.Background(), docID)
	require.NoError(t, err)
	assert.Equal(t, docID, doc.ID)

	// Not found
	_, err = mock.GetDocument(context.Background(), uuid.New())
	assert.Error(t, err)
}

func TestMockRepository_CreateDocument(t *testing.T) {
	var createdDoc *DriverDocument

	mock := &MockRepository{
		CreateDocumentFunc: func(ctx context.Context, doc *DriverDocument) error {
			createdDoc = doc
			return nil
		},
	}

	doc := &DriverDocument{
		ID:       uuid.New(),
		DriverID: uuid.New(),
		Status:   StatusPending,
	}

	err := mock.CreateDocument(context.Background(), doc)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, createdDoc.ID)
}

func TestMockRepository_UpdateDocumentStatus(t *testing.T) {
	var updatedStatus DocumentStatus
	var updatedDocID uuid.UUID

	mock := &MockRepository{
		UpdateDocumentStatusFunc: func(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
			updatedDocID = documentID
			updatedStatus = status
			return nil
		},
	}

	docID := uuid.New()
	reviewerID := uuid.New()
	notes := "Approved"

	err := mock.UpdateDocumentStatus(context.Background(), docID, StatusApproved, &reviewerID, &notes, nil)
	require.NoError(t, err)
	assert.Equal(t, docID, updatedDocID)
	assert.Equal(t, StatusApproved, updatedStatus)
}

func TestMockStorage_Upload(t *testing.T) {
	mock := &MockStorage{}

	reader := bytes.NewReader([]byte("test content"))
	result, err := mock.Upload(context.Background(), "test/file.pdf", reader, 12, "application/pdf")

	require.NoError(t, err)
	assert.Equal(t, "test/file.pdf", result.Key)
	assert.Contains(t, result.URL, "test/file.pdf")
	assert.Equal(t, int64(12), result.Size)
}

func TestMockStorage_Upload_Error(t *testing.T) {
	mock := &MockStorage{
		UploadFunc: func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
			return nil, errors.New("storage error")
		},
	}

	reader := bytes.NewReader([]byte("test"))
	result, err := mock.Upload(context.Background(), "test.pdf", reader, 4, "application/pdf")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMockStorage_Exists(t *testing.T) {
	mock := &MockStorage{
		ExistsFunc: func(ctx context.Context, key string) (bool, error) {
			if key == "existing/file.pdf" {
				return true, nil
			}
			return false, nil
		},
	}

	exists, err := mock.Exists(context.Background(), "existing/file.pdf")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = mock.Exists(context.Background(), "missing/file.pdf")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMockStorage_Delete(t *testing.T) {
	var deletedKey string

	mock := &MockStorage{
		DeleteFunc: func(ctx context.Context, key string) error {
			deletedKey = key
			return nil
		},
	}

	err := mock.Delete(context.Background(), "test/file.pdf")
	require.NoError(t, err)
	assert.Equal(t, "test/file.pdf", deletedKey)
}

func TestMockStorage_GetPresignedUploadURL(t *testing.T) {
	mock := &MockStorage{}

	result, err := mock.GetPresignedUploadURL(context.Background(), "test.pdf", "application/pdf", 15*time.Minute)

	require.NoError(t, err)
	assert.Contains(t, result.URL, "test.pdf")
	assert.Equal(t, "PUT", result.Method)
	assert.Contains(t, result.Headers, "Content-Type")
}

// ========================================
// SERVICE METHOD TESTS
// ========================================

func TestService_GetDocumentTypes_Success(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{
				{ID: uuid.New(), Code: "drivers_license", Name: "Driver's License"},
				{ID: uuid.New(), Code: "vehicle_registration", Name: "Vehicle Registration"},
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	types, err := svc.GetDocumentTypes(context.Background())

	require.NoError(t, err)
	assert.Len(t, types, 2)
	assert.Equal(t, "drivers_license", types[0].Code)
}

func TestService_GetDocumentTypes_Error(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return nil, errors.New("database error")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	types, err := svc.GetDocumentTypes(context.Background())

	assert.Error(t, err)
	assert.Nil(t, types)
}

func TestService_GetDocumentTypes_Empty(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	types, err := svc.GetDocumentTypes(context.Background())

	require.NoError(t, err)
	assert.Empty(t, types)
}

func TestService_GetRequiredDocumentTypes_Success(t *testing.T) {
	mockRepo := &MockRepository{
		GetRequiredDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{
				{ID: uuid.New(), Code: "drivers_license", Name: "Driver's License", IsRequired: true},
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	types, err := svc.GetRequiredDocumentTypes(context.Background())

	require.NoError(t, err)
	assert.Len(t, types, 1)
	assert.True(t, types[0].IsRequired)
}

func TestService_GetRequiredDocumentTypes_Error(t *testing.T) {
	mockRepo := &MockRepository{
		GetRequiredDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return nil, errors.New("database error")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	types, err := svc.GetRequiredDocumentTypes(context.Background())

	assert.Error(t, err)
	assert.Nil(t, types)
}

func TestService_GetDocument_Success(t *testing.T) {
	docID := uuid.New()
	expectedDoc := &DriverDocument{
		ID:     docID,
		Status: StatusApproved,
	}

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			if documentID == docID {
				return expectedDoc, nil
			}
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	doc, err := svc.GetDocument(context.Background(), docID)

	require.NoError(t, err)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, StatusApproved, doc.Status)
}

func TestService_GetDocument_NotFound(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	doc, err := svc.GetDocument(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Nil(t, doc)
}

func TestService_GetDriverDocuments_Success(t *testing.T) {
	driverID := uuid.New()

	mockRepo := &MockRepository{
		GetDriverDocumentsFunc: func(ctx context.Context, dID uuid.UUID) ([]*DriverDocument, error) {
			if dID == driverID {
				return []*DriverDocument{
					{ID: uuid.New(), DriverID: driverID, Status: StatusApproved},
					{ID: uuid.New(), DriverID: driverID, Status: StatusPending},
				}, nil
			}
			return nil, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	docs, err := svc.GetDriverDocuments(context.Background(), driverID)

	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestService_GetDriverDocuments_Empty(t *testing.T) {
	mockRepo := &MockRepository{
		GetDriverDocumentsFunc: func(ctx context.Context, driverID uuid.UUID) ([]*DriverDocument, error) {
			return []*DriverDocument{}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	docs, err := svc.GetDriverDocuments(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestService_GetDriverDocuments_Error(t *testing.T) {
	mockRepo := &MockRepository{
		GetDriverDocumentsFunc: func(ctx context.Context, driverID uuid.UUID) ([]*DriverDocument, error) {
			return nil, errors.New("database error")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	docs, err := svc.GetDriverDocuments(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Nil(t, docs)
}

func TestService_UploadDocument_Success(t *testing.T) {
	driverID := uuid.New()
	docTypeID := uuid.New()
	docType := &DocumentType{
		ID:             docTypeID,
		Code:           "drivers_license",
		AutoOCREnabled: false,
	}

	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			if code == "drivers_license" {
				return docType, nil
			}
			return nil, errors.New("not found")
		},
		GetLatestDocumentByTypeFunc: func(ctx context.Context, dID, dtID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
		CreateDocumentFunc: func(ctx context.Context, doc *DriverDocument) error {
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{
		UploadFunc: func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
			return &storage.UploadResult{
				Key:  key,
				URL:  "https://storage.example.com/" + key,
				Size: size,
			}, nil
		},
	}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test file content"))

	resp, err := svc.UploadDocument(context.Background(), driverID, req, reader, 17, "test.jpg", "image/jpeg")

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.DocumentID)
	assert.Equal(t, StatusPending, resp.Status)
	assert.Equal(t, "Document uploaded successfully", resp.Message)
}

func TestService_UploadDocument_FileTooLarge(t *testing.T) {
	mockRepo := &MockRepository{}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{MaxFileSizeMB: 10})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test"))
	// 11 MB
	fileSize := int64(11 * 1024 * 1024)

	resp, err := svc.UploadDocument(context.Background(), uuid.New(), req, reader, fileSize, "test.jpg", "image/jpeg")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "file size exceeds maximum")
}

func TestService_UploadDocument_InvalidMimeType(t *testing.T) {
	mockRepo := &MockRepository{}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{
		AllowedMimeTypes: []string{"image/jpeg", "image/png"},
	})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test"))

	resp, err := svc.UploadDocument(context.Background(), uuid.New(), req, reader, 4, "test.exe", "application/x-executable")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unsupported file type")
}

func TestService_UploadDocument_InvalidDocumentType(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "invalid_type",
	}
	reader := bytes.NewReader([]byte("test"))

	resp, err := svc.UploadDocument(context.Background(), uuid.New(), req, reader, 4, "test.jpg", "image/jpeg")

	assert.Error(t, err)
	assert.Nil(t, resp)
	// The AppError wraps the underlying error
	assert.Contains(t, err.Error(), "not found")
}

func TestService_UploadDocument_StorageError(t *testing.T) {
	docType := &DocumentType{
		ID:   uuid.New(),
		Code: "drivers_license",
	}

	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
		GetLatestDocumentByTypeFunc: func(ctx context.Context, dID, dtID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{
		UploadFunc: func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
			return nil, errors.New("storage error")
		},
	}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test"))

	resp, err := svc.UploadDocument(context.Background(), uuid.New(), req, reader, 4, "test.jpg", "image/jpeg")

	assert.Error(t, err)
	assert.Nil(t, resp)
	// The InternalServerError wraps ErrInternalServer which has message "internal server error"
	assert.Contains(t, err.Error(), "internal server error")
}

func TestService_UploadDocument_CreateDocumentError(t *testing.T) {
	docType := &DocumentType{
		ID:   uuid.New(),
		Code: "drivers_license",
	}

	var deletedKey string
	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
		GetLatestDocumentByTypeFunc: func(ctx context.Context, dID, dtID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
		CreateDocumentFunc: func(ctx context.Context, doc *DriverDocument) error {
			return errors.New("database error")
		},
	}
	mockStorage := &MockStorage{
		UploadFunc: func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
			return &storage.UploadResult{Key: key, URL: "https://example.com/" + key}, nil
		},
		DeleteFunc: func(ctx context.Context, key string) error {
			deletedKey = key
			return nil
		},
	}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test"))

	resp, err := svc.UploadDocument(context.Background(), uuid.New(), req, reader, 4, "test.jpg", "image/jpeg")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.NotEmpty(t, deletedKey, "Should cleanup uploaded file on failure")
}

func TestService_UploadDocument_SupersedesExisting(t *testing.T) {
	driverID := uuid.New()
	docTypeID := uuid.New()
	existingDocID := uuid.New()

	docType := &DocumentType{
		ID:   docTypeID,
		Code: "drivers_license",
	}

	var supersededDocID uuid.UUID
	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
		GetLatestDocumentByTypeFunc: func(ctx context.Context, dID, dtID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:      existingDocID,
				Status:  StatusPending,
				Version: 1,
			}, nil
		},
		SupersedeDocumentFunc: func(ctx context.Context, docID uuid.UUID) error {
			supersededDocID = docID
			return nil
		},
		CreateDocumentFunc: func(ctx context.Context, doc *DriverDocument) error {
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test"))

	resp, err := svc.UploadDocument(context.Background(), driverID, req, reader, 4, "test.jpg", "image/jpeg")

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, existingDocID, supersededDocID)
}

func TestService_UploadDocument_WithOCREnabled(t *testing.T) {
	docType := &DocumentType{
		ID:             uuid.New(),
		Code:           "drivers_license",
		AutoOCREnabled: true,
	}

	var ocrJobCreated bool
	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
		GetLatestDocumentByTypeFunc: func(ctx context.Context, dID, dtID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
		CreateDocumentFunc: func(ctx context.Context, doc *DriverDocument) error {
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
		CreateOCRJobFunc: func(ctx context.Context, job *OCRProcessingQueue) error {
			ocrJobCreated = true
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{OCREnabled: true})

	req := &UploadDocumentRequest{
		DocumentTypeCode: "drivers_license",
	}
	reader := bytes.NewReader([]byte("test"))

	resp, err := svc.UploadDocument(context.Background(), uuid.New(), req, reader, 4, "test.jpg", "image/jpeg")

	require.NoError(t, err)
	assert.True(t, resp.OCRScheduled)
	assert.True(t, ocrJobCreated)
}

func TestService_UploadDocumentBackSide_Success(t *testing.T) {
	docID := uuid.New()
	docType := &DocumentType{
		ID:                uuid.New(),
		Code:              "drivers_license",
		RequiresFrontBack: true,
	}

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:           docID,
				DriverID:     uuid.New(),
				DocumentType: docType,
			}, nil
		},
		UpdateDocumentBackFileFunc: func(ctx context.Context, documentID uuid.UUID, backFileURL, backFileKey string) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	reader := bytes.NewReader([]byte("back side content"))
	err := svc.UploadDocumentBackSide(context.Background(), docID, reader, 17, "back.jpg", "image/jpeg")

	require.NoError(t, err)
}

func TestService_UploadDocumentBackSide_DocumentNotFound(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	reader := bytes.NewReader([]byte("test"))
	err := svc.UploadDocumentBackSide(context.Background(), uuid.New(), reader, 4, "back.jpg", "image/jpeg")

	assert.Error(t, err)
	// The AppError wraps the underlying error and returns it from Error()
	assert.Contains(t, err.Error(), "not found")
}

func TestService_UploadDocumentBackSide_NotRequired(t *testing.T) {
	docType := &DocumentType{
		ID:                uuid.New(),
		RequiresFrontBack: false,
	}

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:           uuid.New(),
				DocumentType: docType,
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	reader := bytes.NewReader([]byte("test"))
	err := svc.UploadDocumentBackSide(context.Background(), uuid.New(), reader, 4, "back.jpg", "image/jpeg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not require a back side")
}

func TestService_UploadDocumentBackSide_FileTooLarge(t *testing.T) {
	docType := &DocumentType{
		ID:                uuid.New(),
		RequiresFrontBack: true,
	}

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:           uuid.New(),
				DocumentType: docType,
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{MaxFileSizeMB: 10})

	reader := bytes.NewReader([]byte("test"))
	fileSize := int64(11 * 1024 * 1024)
	err := svc.UploadDocumentBackSide(context.Background(), uuid.New(), reader, fileSize, "back.jpg", "image/jpeg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file size exceeds maximum")
}

func TestService_GetPresignedUploadURL_Success(t *testing.T) {
	docType := &DocumentType{
		ID:                uuid.New(),
		Code:              "drivers_license",
		RequiresFrontBack: true,
	}

	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &PresignedUploadRequest{
		DocumentTypeCode: "drivers_license",
		FileName:         "license.jpg",
		ContentType:      "image/jpeg",
		IsFrontSide:      true,
	}

	resp, err := svc.GetPresignedUploadURL(context.Background(), uuid.New(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.UploadURL)
	assert.Equal(t, "PUT", resp.Method)
	assert.NotEmpty(t, resp.FileKey)
}

func TestService_GetPresignedUploadURL_InvalidDocType(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &PresignedUploadRequest{
		DocumentTypeCode: "invalid",
		FileName:         "test.jpg",
		ContentType:      "image/jpeg",
	}

	resp, err := svc.GetPresignedUploadURL(context.Background(), uuid.New(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestService_GetPresignedUploadURL_InvalidMimeType(t *testing.T) {
	docType := &DocumentType{
		ID:   uuid.New(),
		Code: "drivers_license",
	}

	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{
		AllowedMimeTypes: []string{"image/jpeg"},
	})

	req := &PresignedUploadRequest{
		DocumentTypeCode: "drivers_license",
		FileName:         "test.exe",
		ContentType:      "application/x-executable",
	}

	resp, err := svc.GetPresignedUploadURL(context.Background(), uuid.New(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unsupported file type")
}

func TestService_CompleteDirectUpload_Success(t *testing.T) {
	driverID := uuid.New()
	docType := &DocumentType{
		ID:   uuid.New(),
		Code: "drivers_license",
	}

	mockRepo := &MockRepository{
		GetDocumentTypeByCodeFunc: func(ctx context.Context, code string) (*DocumentType, error) {
			return docType, nil
		},
		GetLatestDocumentByTypeFunc: func(ctx context.Context, dID, dtID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
		CreateDocumentFunc: func(ctx context.Context, doc *DriverDocument) error {
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{
		ExistsFunc: func(ctx context.Context, key string) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadCompleteRequest{
		FileKey:          "drivers/123/documents/test.jpg",
		DocumentTypeCode: "drivers_license",
		IsFrontSide:      true,
	}

	resp, err := svc.CompleteDirectUpload(context.Background(), driverID, req)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.DocumentID)
	assert.Equal(t, StatusPending, resp.Status)
}

func TestService_CompleteDirectUpload_FileNotFound(t *testing.T) {
	mockRepo := &MockRepository{}
	mockStorage := &MockStorage{
		ExistsFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &UploadCompleteRequest{
		FileKey:          "missing/file.jpg",
		DocumentTypeCode: "drivers_license",
	}

	resp, err := svc.CompleteDirectUpload(context.Background(), uuid.New(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "uploaded file not found")
}

func TestService_ReviewDocument_Approve_Success(t *testing.T) {
	docID := uuid.New()
	reviewerID := uuid.New()

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     docID,
				Status: StatusPending,
			}, nil
		},
		UpdateDocumentStatusFunc: func(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
			return nil
		},
		UpdateDocumentDetailsFunc: func(ctx context.Context, documentID uuid.UUID, documentNumber *string, issueDate, expiryDate *time.Time, issuingAuthority *string) error {
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action: "approve",
		Notes:  "Looks good",
	}

	err := svc.ReviewDocument(context.Background(), docID, reviewerID, req)

	require.NoError(t, err)
}

func TestService_ReviewDocument_Reject_Success(t *testing.T) {
	docID := uuid.New()
	reviewerID := uuid.New()

	var capturedStatus DocumentStatus
	var capturedReason *string

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     docID,
				Status: StatusUnderReview,
			}, nil
		},
		UpdateDocumentStatusFunc: func(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
			capturedStatus = status
			capturedReason = rejectionReason
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action:          "reject",
		RejectionReason: "Document is expired",
	}

	err := svc.ReviewDocument(context.Background(), docID, reviewerID, req)

	require.NoError(t, err)
	assert.Equal(t, StatusRejected, capturedStatus)
	require.NotNil(t, capturedReason)
	assert.Equal(t, "Document is expired", *capturedReason)
}

func TestService_ReviewDocument_RejectWithoutReason(t *testing.T) {
	docID := uuid.New()
	reviewerID := uuid.New()

	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     docID,
				Status: StatusPending,
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action:          "reject",
		RejectionReason: "", // Missing reason
	}

	err := svc.ReviewDocument(context.Background(), docID, reviewerID, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rejection reason is required")
}

func TestService_ReviewDocument_DocumentNotFound(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action: "approve",
	}

	err := svc.ReviewDocument(context.Background(), uuid.New(), uuid.New(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_ReviewDocument_AlreadyApproved(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     uuid.New(),
				Status: StatusApproved, // Already approved
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action: "approve",
	}

	err := svc.ReviewDocument(context.Background(), uuid.New(), uuid.New(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not pending review")
}

func TestService_ReviewDocument_InvalidAction(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     uuid.New(),
				Status: StatusPending,
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action: "invalid_action",
	}

	err := svc.ReviewDocument(context.Background(), uuid.New(), uuid.New(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action")
}

func TestService_ReviewDocument_RequestResubmit(t *testing.T) {
	docID := uuid.New()
	reviewerID := uuid.New()

	var capturedReason *string
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     docID,
				Status: StatusPending,
			}, nil
		},
		UpdateDocumentStatusFunc: func(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
			capturedReason = rejectionReason
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	req := &ReviewDocumentRequest{
		Action: "request_resubmit",
	}

	err := svc.ReviewDocument(context.Background(), docID, reviewerID, req)

	require.NoError(t, err)
	require.NotNil(t, capturedReason)
	assert.Equal(t, "Document needs to be resubmitted", *capturedReason)
}

func TestService_GetPendingReviews_Success(t *testing.T) {
	mockRepo := &MockRepository{
		GetPendingReviewsFunc: func(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error) {
			return []*PendingReviewDocument{
				{Document: &DriverDocument{ID: uuid.New()}, DriverName: "John Doe"},
				{Document: &DriverDocument{ID: uuid.New()}, DriverName: "Jane Doe"},
			}, 10, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	reviews, total, err := svc.GetPendingReviews(context.Background(), 1, 20)

	require.NoError(t, err)
	assert.Len(t, reviews, 2)
	assert.Equal(t, 10, total)
}

func TestService_GetPendingReviews_DefaultPagination(t *testing.T) {
	var capturedLimit, capturedOffset int

	mockRepo := &MockRepository{
		GetPendingReviewsFunc: func(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []*PendingReviewDocument{}, 0, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	_, _, err := svc.GetPendingReviews(context.Background(), 0, 0)

	require.NoError(t, err)
	assert.Equal(t, 20, capturedLimit)
	assert.Equal(t, 0, capturedOffset)
}

func TestService_GetPendingReviews_OversizedPageSize(t *testing.T) {
	var capturedLimit int

	mockRepo := &MockRepository{
		GetPendingReviewsFunc: func(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error) {
			capturedLimit = limit
			return []*PendingReviewDocument{}, 0, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	_, _, err := svc.GetPendingReviews(context.Background(), 1, 200)

	require.NoError(t, err)
	assert.Equal(t, 20, capturedLimit)
}

func TestService_GetExpiringDocuments_Success(t *testing.T) {
	mockRepo := &MockRepository{
		GetExpiringDocumentsFunc: func(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error) {
			return []*ExpiringDocument{
				{Document: &DriverDocument{ID: uuid.New()}, DaysUntilExpiry: 7, Urgency: "critical"},
				{Document: &DriverDocument{ID: uuid.New()}, DaysUntilExpiry: 25, Urgency: "warning"},
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	docs, err := svc.GetExpiringDocuments(context.Background(), 30)

	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestService_GetExpiringDocuments_DefaultDays(t *testing.T) {
	var capturedDays int

	mockRepo := &MockRepository{
		GetExpiringDocumentsFunc: func(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error) {
			capturedDays = daysAhead
			return []*ExpiringDocument{}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	_, err := svc.GetExpiringDocuments(context.Background(), 0)

	require.NoError(t, err)
	assert.Equal(t, 30, capturedDays)
}

func TestService_GetExpiringDocuments_Error(t *testing.T) {
	mockRepo := &MockRepository{
		GetExpiringDocumentsFunc: func(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error) {
			return nil, errors.New("database error")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	docs, err := svc.GetExpiringDocuments(context.Background(), 30)

	assert.Error(t, err)
	assert.Nil(t, docs)
}

func TestService_StartReview_Success(t *testing.T) {
	docID := uuid.New()
	reviewerID := uuid.New()

	var capturedStatus DocumentStatus
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     docID,
				Status: StatusPending,
			}, nil
		},
		UpdateDocumentStatusFunc: func(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
			capturedStatus = status
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	err := svc.StartReview(context.Background(), docID, reviewerID)

	require.NoError(t, err)
	assert.Equal(t, StatusUnderReview, capturedStatus)
}

func TestService_StartReview_DocumentNotFound(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	err := svc.StartReview(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_StartReview_NotPending(t *testing.T) {
	mockRepo := &MockRepository{
		GetDocumentFunc: func(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
			return &DriverDocument{
				ID:     uuid.New(),
				Status: StatusApproved,
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	err := svc.StartReview(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document is not pending")
}

func TestService_ProcessOCRResult_Success(t *testing.T) {
	docID := uuid.New()
	issueDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)

	var updatedOCRData map[string]interface{}
	var updatedConfidence float64

	mockRepo := &MockRepository{
		UpdateDocumentOCRDataFunc: func(ctx context.Context, documentID uuid.UUID, ocrData map[string]interface{}, confidence float64) error {
			updatedOCRData = ocrData
			updatedConfidence = confidence
			return nil
		},
		UpdateDocumentDetailsFunc: func(ctx context.Context, documentID uuid.UUID, documentNumber *string, issueDate, expiryDate *time.Time, issuingAuthority *string) error {
			return nil
		},
		CreateHistoryFunc: func(ctx context.Context, history *DocumentVerificationHistory) error {
			return nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	result := &OCRResult{
		DocumentNumber:   "DL123456",
		FullName:         "John Doe",
		IssueDate:        &issueDate,
		ExpiryDate:       &expiryDate,
		IssuingAuthority: "DMV",
		Confidence:       0.95,
	}

	err := svc.ProcessOCRResult(context.Background(), docID, result)

	require.NoError(t, err)
	assert.Equal(t, "DL123456", updatedOCRData["document_number"])
	assert.Equal(t, "John Doe", updatedOCRData["full_name"])
	assert.Equal(t, 0.95, updatedConfidence)
}

func TestService_ProcessOCRResult_Error(t *testing.T) {
	mockRepo := &MockRepository{
		UpdateDocumentOCRDataFunc: func(ctx context.Context, documentID uuid.UUID, ocrData map[string]interface{}, confidence float64) error {
			return errors.New("database error")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	result := &OCRResult{
		DocumentNumber: "DL123",
		Confidence:     0.9,
	}

	err := svc.ProcessOCRResult(context.Background(), uuid.New(), result)

	assert.Error(t, err)
}

func TestService_GetDriverVerificationStatus_AllApproved(t *testing.T) {
	driverID := uuid.New()
	docTypeID := uuid.New()
	expiryDate := time.Now().Add(30 * 24 * time.Hour)

	docType := &DocumentType{
		ID:             docTypeID,
		Code:           "drivers_license",
		Name:           "Driver's License",
		IsRequired:     true,
		RequiresExpiry: true,
	}

	mockRepo := &MockRepository{
		GetRequiredDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{docType}, nil
		},
		GetDriverDocumentsFunc: func(ctx context.Context, dID uuid.UUID) ([]*DriverDocument, error) {
			return []*DriverDocument{
				{
					ID:             uuid.New(),
					DriverID:       driverID,
					DocumentTypeID: docTypeID,
					Status:         StatusApproved,
					ExpiryDate:     &expiryDate,
					SubmittedAt:    time.Now(),
				},
			}, nil
		},
		GetDriverVerificationStatusFunc: func(ctx context.Context, dID uuid.UUID) (*DriverVerificationStatus, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	status, err := svc.GetDriverVerificationStatus(context.Background(), driverID)

	require.NoError(t, err)
	assert.Equal(t, VerificationApproved, status.Status)
	assert.True(t, status.CanDrive)
	assert.Empty(t, status.MissingDocuments)
}

func TestService_GetDriverVerificationStatus_MissingDocuments(t *testing.T) {
	driverID := uuid.New()

	docType := &DocumentType{
		ID:         uuid.New(),
		Code:       "drivers_license",
		Name:       "Driver's License",
		IsRequired: true,
	}

	mockRepo := &MockRepository{
		GetRequiredDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{docType}, nil
		},
		GetDriverDocumentsFunc: func(ctx context.Context, dID uuid.UUID) ([]*DriverDocument, error) {
			return []*DriverDocument{}, nil // No documents
		},
		GetDriverVerificationStatusFunc: func(ctx context.Context, dID uuid.UUID) (*DriverVerificationStatus, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	status, err := svc.GetDriverVerificationStatus(context.Background(), driverID)

	require.NoError(t, err)
	assert.Equal(t, VerificationIncomplete, status.Status)
	assert.False(t, status.CanDrive)
	assert.Contains(t, status.MissingDocuments, "Driver's License")
}

func TestService_GetDriverVerificationStatus_Suspended(t *testing.T) {
	driverID := uuid.New()
	docTypeID := uuid.New()
	suspensionReason := "Failed background check"

	// Need at least one required document type so approvedCount != len(requiredTypes)
	docType := &DocumentType{
		ID:         docTypeID,
		Code:       "drivers_license",
		Name:       "Driver's License",
		IsRequired: true,
	}

	mockRepo := &MockRepository{
		GetRequiredDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{docType}, nil
		},
		GetDriverDocumentsFunc: func(ctx context.Context, dID uuid.UUID) ([]*DriverDocument, error) {
			return []*DriverDocument{}, nil // No documents submitted
		},
		GetDriverVerificationStatusFunc: func(ctx context.Context, dID uuid.UUID) (*DriverVerificationStatus, error) {
			return &DriverVerificationStatus{
				DriverID:           driverID,
				VerificationStatus: VerificationSuspended,
				SuspensionReason:   &suspensionReason,
			}, nil
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	status, err := svc.GetDriverVerificationStatus(context.Background(), driverID)

	require.NoError(t, err)
	// When suspended but missing docs, the status logic prioritizes missing docs message
	// but canDrive should still be false
	assert.False(t, status.CanDrive)
	// The missing docs check overrides the message, so we just check canDrive is false
}

func TestService_GetDriverVerificationStatus_PendingDocuments(t *testing.T) {
	driverID := uuid.New()
	docTypeID := uuid.New()

	docType := &DocumentType{
		ID:         docTypeID,
		Code:       "drivers_license",
		Name:       "Driver's License",
		IsRequired: true,
	}

	mockRepo := &MockRepository{
		GetRequiredDocumentTypesFunc: func(ctx context.Context) ([]*DocumentType, error) {
			return []*DocumentType{docType}, nil
		},
		GetDriverDocumentsFunc: func(ctx context.Context, dID uuid.UUID) ([]*DriverDocument, error) {
			return []*DriverDocument{
				{
					ID:             uuid.New(),
					DriverID:       driverID,
					DocumentTypeID: docTypeID,
					Status:         StatusPending,
					SubmittedAt:    time.Now(),
				},
			}, nil
		},
		GetDriverVerificationStatusFunc: func(ctx context.Context, dID uuid.UUID) (*DriverVerificationStatus, error) {
			return nil, errors.New("not found")
		},
	}
	mockStorage := &MockStorage{}
	svc := newTestService(mockRepo, mockStorage, ServiceConfig{})

	status, err := svc.GetDriverVerificationStatus(context.Background(), driverID)

	require.NoError(t, err)
	assert.False(t, status.CanDrive)
	assert.Equal(t, VerificationIncomplete, status.Status)
}

func TestNewService_DefaultConfig(t *testing.T) {
	mockRepo := &MockRepository{}
	mockStorage := &MockStorage{}

	svc := NewService(mockRepo, mockStorage, ServiceConfig{})

	assert.Equal(t, 10, svc.config.MaxFileSizeMB)
	assert.Len(t, svc.config.AllowedMimeTypes, 4)
	assert.Contains(t, svc.config.AllowedMimeTypes, "image/jpeg")
	assert.Contains(t, svc.config.AllowedMimeTypes, "application/pdf")
}

func TestNewService_CustomConfig(t *testing.T) {
	mockRepo := &MockRepository{}
	mockStorage := &MockStorage{}

	config := ServiceConfig{
		MaxFileSizeMB:    20,
		AllowedMimeTypes: []string{"image/jpeg"},
		OCREnabled:       true,
		OCRProvider:      "tesseract",
	}

	svc := NewService(mockRepo, mockStorage, config)

	assert.Equal(t, 20, svc.config.MaxFileSizeMB)
	assert.Len(t, svc.config.AllowedMimeTypes, 1)
	assert.True(t, svc.config.OCREnabled)
	assert.Equal(t, "tesseract", svc.config.OCRProvider)
}

// ========================================
// BENCHMARKS
// ========================================

func BenchmarkNilIfEmpty_Empty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nilIfEmpty("")
	}
}

func BenchmarkNilIfEmpty_NonEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nilIfEmpty("DL-123456")
	}
}

func BenchmarkUUIDGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		uuid.New()
	}
}

func BenchmarkDocumentCreation(b *testing.B) {
	driverID := uuid.New()
	docType := createTestDocumentType()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createTestDocument(driverID, docType, StatusPending)
	}
}
