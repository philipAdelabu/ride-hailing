package recording

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK REPOSITORY
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateRecording(ctx context.Context, recording *RideRecording) error {
	args := m.Called(ctx, recording)
	return args.Error(0)
}

func (m *mockRepo) GetRecording(ctx context.Context, recordingID uuid.UUID) (*RideRecording, error) {
	args := m.Called(ctx, recordingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideRecording), args.Error(1)
}

func (m *mockRepo) GetRecordingsByRide(ctx context.Context, rideID uuid.UUID) ([]*RideRecording, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideRecording), args.Error(1)
}

func (m *mockRepo) GetActiveRecordingForRide(ctx context.Context, rideID, userID uuid.UUID) (*RideRecording, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideRecording), args.Error(1)
}

func (m *mockRepo) UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status RecordingStatus) error {
	args := m.Called(ctx, recordingID, status)
	return args.Error(0)
}

func (m *mockRepo) UpdateRecordingStopped(ctx context.Context, recordingID uuid.UUID, endedAt time.Time) error {
	args := m.Called(ctx, recordingID, endedAt)
	return args.Error(0)
}

func (m *mockRepo) UpdateRecordingUpload(ctx context.Context, recordingID uuid.UUID, uploadID string, chunksReceived, totalChunks int) error {
	args := m.Called(ctx, recordingID, uploadID, chunksReceived, totalChunks)
	return args.Error(0)
}

func (m *mockRepo) UpdateRecordingCompleted(ctx context.Context, recordingID uuid.UUID, fileURL string, fileSize int64, durationSeconds int) error {
	args := m.Called(ctx, recordingID, fileURL, fileSize, durationSeconds)
	return args.Error(0)
}

func (m *mockRepo) UpdateRecordingProcessed(ctx context.Context, recordingID uuid.UUID, thumbnailURL *string) error {
	args := m.Called(ctx, recordingID, thumbnailURL)
	return args.Error(0)
}

func (m *mockRepo) UpdateRetentionPolicy(ctx context.Context, recordingID uuid.UUID, policy RetentionPolicy, newExpiry time.Time) error {
	args := m.Called(ctx, recordingID, policy, newExpiry)
	return args.Error(0)
}

func (m *mockRepo) GetExpiredRecordings(ctx context.Context, limit int) ([]*RideRecording, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideRecording), args.Error(1)
}

func (m *mockRepo) MarkRecordingDeleted(ctx context.Context, recordingID uuid.UUID) error {
	args := m.Called(ctx, recordingID)
	return args.Error(0)
}

func (m *mockRepo) CreateConsent(ctx context.Context, consent *RecordingConsent) error {
	args := m.Called(ctx, consent)
	return args.Error(0)
}

func (m *mockRepo) GetConsent(ctx context.Context, rideID, userID uuid.UUID) (*RecordingConsent, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordingConsent), args.Error(1)
}

func (m *mockRepo) CheckAllConsented(ctx context.Context, rideID uuid.UUID) (bool, error) {
	args := m.Called(ctx, rideID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepo) LogAccess(ctx context.Context, log *RecordingAccessLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *mockRepo) GetAccessLogs(ctx context.Context, recordingID uuid.UUID) ([]*RecordingAccessLog, error) {
	args := m.Called(ctx, recordingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RecordingAccessLog), args.Error(1)
}

func (m *mockRepo) GetSettings(ctx context.Context, userID uuid.UUID) (*RecordingSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordingSettings), args.Error(1)
}

func (m *mockRepo) UpsertSettings(ctx context.Context, settings *RecordingSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *mockRepo) GetRecordingStats(ctx context.Context) (*RecordingStatsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordingStatsResponse), args.Error(1)
}

// ========================================
// MOCK STORAGE
// ========================================

type mockStorage struct {
	mock.Mock
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
	args := m.Called(ctx, key, reader, size, contentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *mockStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockStorage) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *mockStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (*storage.PresignedURLResult, error) {
	args := m.Called(ctx, key, contentType, expiresIn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.PresignedURLResult), args.Error(1)
}

func (m *mockStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (*storage.PresignedURLResult, error) {
	args := m.Called(ctx, key, expiresIn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.PresignedURLResult), args.Error(1)
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockStorage) Copy(ctx context.Context, sourceKey, destKey string) error {
	args := m.Called(ctx, sourceKey, destKey)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo *mockRepo, storageClient *mockStorage) *Service {
	return &Service{
		repo:          repo,
		storageClient: storageClient,
		bucketName:    "test-bucket",
		maxDuration:   7200,                // 2 hours
		maxFileSize:   500 * 1024 * 1024,   // 500MB
	}
}

func ptrString(s string) *string {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrInt(i int) *int {
	return &i
}

// ========================================
// CONSENT RECORDING TESTS (GDPR CRITICAL)
// ========================================

func TestRecordConsent_Success(t *testing.T) {
	repo := new(mockRepo)
	storage := new(mockStorage)
	svc := newTestService(repo, storage)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)"

	repo.On("CreateConsent", ctx, mock.AnythingOfType("*recording.RecordingConsent")).Return(nil)

	req := &RecordingConsentRequest{
		RideID:    rideID,
		Consented: true,
	}

	err := svc.RecordConsent(ctx, userID, "rider", req, &ipAddress, &userAgent)
	require.NoError(t, err)

	repo.AssertExpectations(t)

	// Verify the consent was created with proper fields
	repo.AssertCalled(t, "CreateConsent", ctx, mock.MatchedBy(func(consent *RecordingConsent) bool {
		return consent.RideID == rideID &&
			consent.UserID == userID &&
			consent.UserType == "rider" &&
			consent.Consented == true &&
			consent.IPAddress != nil && *consent.IPAddress == ipAddress &&
			consent.UserAgent != nil && *consent.UserAgent == userAgent &&
			!consent.ConsentedAt.IsZero()
	}))
}

func TestRecordConsent_WithTimestamp(t *testing.T) {
	repo := new(mockRepo)
	storage := new(mockStorage)
	svc := newTestService(repo, storage)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	beforeTest := time.Now()

	repo.On("CreateConsent", ctx, mock.AnythingOfType("*recording.RecordingConsent")).Return(nil)

	req := &RecordingConsentRequest{
		RideID:    rideID,
		Consented: true,
	}

	err := svc.RecordConsent(ctx, userID, "driver", req, nil, nil)
	require.NoError(t, err)

	afterTest := time.Now()

	// Verify timestamp was set properly
	repo.AssertCalled(t, "CreateConsent", ctx, mock.MatchedBy(func(consent *RecordingConsent) bool {
		return !consent.ConsentedAt.Before(beforeTest) && !consent.ConsentedAt.After(afterTest)
	}))
}

func TestRecordConsent_Declined(t *testing.T) {
	repo := new(mockRepo)
	storage := new(mockStorage)
	svc := newTestService(repo, storage)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()
	ipAddress := "10.0.0.1"

	repo.On("CreateConsent", ctx, mock.AnythingOfType("*recording.RecordingConsent")).Return(nil)

	req := &RecordingConsentRequest{
		RideID:    rideID,
		Consented: false, // User declined consent
	}

	err := svc.RecordConsent(ctx, userID, "rider", req, &ipAddress, nil)
	require.NoError(t, err)

	repo.AssertCalled(t, "CreateConsent", ctx, mock.MatchedBy(func(consent *RecordingConsent) bool {
		return consent.Consented == false
	}))
}

func TestRecordConsent_RepoError(t *testing.T) {
	repo := new(mockRepo)
	storage := new(mockStorage)
	svc := newTestService(repo, storage)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("CreateConsent", ctx, mock.AnythingOfType("*recording.RecordingConsent")).
		Return(errors.New("database connection failed"))

	req := &RecordingConsentRequest{
		RideID:    rideID,
		Consented: true,
	}

	err := svc.RecordConsent(ctx, userID, "rider", req, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestCheckConsent_AllConsented(t *testing.T) {
	repo := new(mockRepo)
	storage := new(mockStorage)
	svc := newTestService(repo, storage)
	ctx := context.Background()
	rideID := uuid.New()

	repo.On("CheckAllConsented", ctx, rideID).Return(true, nil)

	consented, err := svc.CheckConsent(ctx, rideID)
	require.NoError(t, err)
	assert.True(t, consented)

	repo.AssertExpectations(t)
}

func TestCheckConsent_NotAllConsented(t *testing.T) {
	repo := new(mockRepo)
	storage := new(mockStorage)
	svc := newTestService(repo, storage)
	ctx := context.Background()
	rideID := uuid.New()

	repo.On("CheckAllConsented", ctx, rideID).Return(false, nil)

	consented, err := svc.CheckConsent(ctx, rideID)
	require.NoError(t, err)
	assert.False(t, consented)

	repo.AssertExpectations(t)
}

// ========================================
// START RECORDING TESTS
// ========================================

func TestStartRecording_Success_Audio(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	// No active recording
	repo.On("GetActiveRecordingForRide", ctx, rideID, userID).Return(nil, errors.New("not found"))
	repo.On("CreateRecording", ctx, mock.AnythingOfType("*recording.RideRecording")).Return(nil)
	repo.On("UpdateRecordingStatus", ctx, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)

	storageClient.On("GetPresignedUploadURL", ctx, mock.AnythingOfType("string"), "audio/m4a", time.Hour).
		Return(&storage.PresignedURLResult{
			URL:       "https://storage.example.com/upload?signature=abc123",
			Method:    "PUT",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)

	req := &StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
		Quality:       "high",
	}

	resp, err := svc.StartRecording(ctx, userID, "rider", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEqual(t, uuid.Nil, resp.RecordingID)
	assert.NotEmpty(t, resp.UploadURL)
	assert.NotEmpty(t, resp.UploadID)
	assert.Equal(t, 7200, resp.MaxDuration)
	assert.Equal(t, int64(500*1024*1024), resp.MaxFileSize)
	assert.Equal(t, string(RecordingStatusRecording), resp.Status)

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestStartRecording_Success_Video(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetActiveRecordingForRide", ctx, rideID, userID).Return(nil, errors.New("not found"))
	repo.On("CreateRecording", ctx, mock.AnythingOfType("*recording.RideRecording")).Return(nil)
	repo.On("UpdateRecordingStatus", ctx, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)

	storageClient.On("GetPresignedUploadURL", ctx, mock.AnythingOfType("string"), "video/mp4", time.Hour).
		Return(&storage.PresignedURLResult{
			URL:       "https://storage.example.com/upload?signature=xyz789",
			Method:    "PUT",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)

	req := &StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeVideo,
	}

	resp, err := svc.StartRecording(ctx, userID, "driver", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEqual(t, uuid.Nil, resp.RecordingID)
	assert.Contains(t, resp.UploadID, ".mp4")

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestStartRecording_AlreadyActive(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	existingRecording := &RideRecording{
		ID:     uuid.New(),
		RideID: rideID,
		UserID: userID,
		Status: RecordingStatusRecording,
	}
	repo.On("GetActiveRecordingForRide", ctx, rideID, userID).Return(existingRecording, nil)

	req := &StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	resp, err := svc.StartRecording(ctx, userID, "rider", req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "recording already in progress")

	repo.AssertExpectations(t)
}

func TestStartRecording_DefaultQuality(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetActiveRecordingForRide", ctx, rideID, userID).Return(nil, errors.New("not found"))
	repo.On("CreateRecording", ctx, mock.MatchedBy(func(rec *RideRecording) bool {
		return rec.Quality == "medium" // Default quality
	})).Return(nil)
	repo.On("UpdateRecordingStatus", ctx, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)

	storageClient.On("GetPresignedUploadURL", ctx, mock.AnythingOfType("string"), "audio/m4a", time.Hour).
		Return(&storage.PresignedURLResult{
			URL:       "https://storage.example.com/upload",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)

	req := &StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
		Quality:       "", // Empty quality should default to "medium"
	}

	resp, err := svc.StartRecording(ctx, userID, "rider", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	repo.AssertExpectations(t)
}

func TestStartRecording_CreateRepoError(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetActiveRecordingForRide", ctx, rideID, userID).Return(nil, errors.New("not found"))
	repo.On("CreateRecording", ctx, mock.AnythingOfType("*recording.RideRecording")).
		Return(errors.New("database error"))

	req := &StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	resp, err := svc.StartRecording(ctx, userID, "rider", req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "internal server error")

	repo.AssertExpectations(t)
}

func TestStartRecording_PresignedURLError(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetActiveRecordingForRide", ctx, rideID, userID).Return(nil, errors.New("not found"))
	repo.On("CreateRecording", ctx, mock.AnythingOfType("*recording.RideRecording")).Return(nil)

	storageClient.On("GetPresignedUploadURL", ctx, mock.AnythingOfType("string"), "audio/m4a", time.Hour).
		Return(nil, errors.New("storage service unavailable"))

	req := &StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	resp, err := svc.StartRecording(ctx, userID, "rider", req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "internal server error")

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

// ========================================
// COMPLETE UPLOAD TESTS (500MB VALIDATION)
// ========================================

func TestCompleteUpload_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeAudio,
		Status:        RecordingStatusStopped,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	storageClient.On("GetURL", mock.AnythingOfType("string")).Return("https://storage.example.com/recordings/test.m4a")
	repo.On("UpdateRecordingCompleted", ctx, recordingID, mock.AnythingOfType("string"), int64(100*1024*1024), 3600).Return(nil)
	// For async processing
	repo.On("UpdateRecordingProcessed", mock.Anything, recordingID, (*string)(nil)).Return(nil).Maybe()

	req := &CompleteUploadRequest{
		RecordingID:     recordingID,
		TotalSize:       100 * 1024 * 1024, // 100MB - under limit
		DurationSeconds: 3600,
	}

	err := svc.CompleteUpload(ctx, userID, req)
	require.NoError(t, err)

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestCompleteUpload_ExceedsMaxFileSize(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeVideo,
		Status:        RecordingStatusStopped,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	req := &CompleteUploadRequest{
		RecordingID:     recordingID,
		TotalSize:       600 * 1024 * 1024, // 600MB - EXCEEDS 500MB limit
		DurationSeconds: 3600,
	}

	err := svc.CompleteUpload(ctx, userID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file size exceeds maximum allowed")
	assert.Contains(t, err.Error(), "524288000") // 500MB in bytes

	repo.AssertExpectations(t)
}

func TestCompleteUpload_ExactlyAtLimit(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeAudio,
		Status:        RecordingStatusStopped,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	storageClient.On("GetURL", mock.AnythingOfType("string")).Return("https://storage.example.com/recordings/test.m4a")
	repo.On("UpdateRecordingCompleted", ctx, recordingID, mock.AnythingOfType("string"), int64(500*1024*1024), 7200).Return(nil)
	repo.On("UpdateRecordingProcessed", mock.Anything, recordingID, (*string)(nil)).Return(nil).Maybe()

	req := &CompleteUploadRequest{
		RecordingID:     recordingID,
		TotalSize:       500 * 1024 * 1024, // Exactly 500MB - at limit
		DurationSeconds: 7200,
	}

	err := svc.CompleteUpload(ctx, userID, req)
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestCompleteUpload_JustOverLimit(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeVideo,
		Status:        RecordingStatusStopped,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	req := &CompleteUploadRequest{
		RecordingID:     recordingID,
		TotalSize:       500*1024*1024 + 1, // 500MB + 1 byte - just over limit
		DurationSeconds: 7200,
	}

	err := svc.CompleteUpload(ctx, userID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file size exceeds maximum allowed")

	repo.AssertExpectations(t)
}

func TestCompleteUpload_NotFound(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	repo.On("GetRecording", ctx, recordingID).Return(nil, errors.New("not found"))

	req := &CompleteUploadRequest{
		RecordingID:     recordingID,
		TotalSize:       100 * 1024 * 1024,
		DurationSeconds: 600,
	}

	err := svc.CompleteUpload(ctx, userID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	repo.AssertExpectations(t)
}

func TestCompleteUpload_Unauthorized(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:     recordingID,
		UserID: otherUserID, // Different user owns this recording
		Status: RecordingStatusStopped,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	req := &CompleteUploadRequest{
		RecordingID:     recordingID,
		TotalSize:       100 * 1024 * 1024,
		DurationSeconds: 600,
	}

	err := svc.CompleteUpload(ctx, userID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

// ========================================
// STOP RECORDING TESTS
// ========================================

func TestStopRecording_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:        recordingID,
		UserID:    userID,
		Status:    RecordingStatusRecording,
		StartedAt: time.Now().Add(-30 * time.Minute),
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	repo.On("UpdateRecordingStopped", ctx, recordingID, mock.AnythingOfType("time.Time")).Return(nil)

	req := &StopRecordingRequest{
		RecordingID: recordingID,
	}

	resp, err := svc.StopRecording(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, recordingID, resp.RecordingID)
	assert.Equal(t, string(RecordingStatusStopped), resp.Status)
	assert.GreaterOrEqual(t, resp.DurationSeconds, 1800) // At least 30 minutes

	repo.AssertExpectations(t)
}

func TestStopRecording_PausedStatus(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:        recordingID,
		UserID:    userID,
		Status:    RecordingStatusPaused,
		StartedAt: time.Now().Add(-10 * time.Minute),
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	repo.On("UpdateRecordingStopped", ctx, recordingID, mock.AnythingOfType("time.Time")).Return(nil)

	req := &StopRecordingRequest{
		RecordingID: recordingID,
	}

	resp, err := svc.StopRecording(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, string(RecordingStatusStopped), resp.Status)

	repo.AssertExpectations(t)
}

func TestStopRecording_NotFound(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	repo.On("GetRecording", ctx, recordingID).Return(nil, errors.New("not found"))

	req := &StopRecordingRequest{
		RecordingID: recordingID,
	}

	resp, err := svc.StopRecording(ctx, userID, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")

	repo.AssertExpectations(t)
}

func TestStopRecording_Unauthorized(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:     recordingID,
		UserID: otherUserID,
		Status: RecordingStatusRecording,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	req := &StopRecordingRequest{
		RecordingID: recordingID,
	}

	resp, err := svc.StopRecording(ctx, userID, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

func TestStopRecording_AlreadyStopped(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:     recordingID,
		UserID: userID,
		Status: RecordingStatusCompleted, // Already completed
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	req := &StopRecordingRequest{
		RecordingID: recordingID,
	}

	resp, err := svc.StopRecording(ctx, userID, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "recording is not active")

	repo.AssertExpectations(t)
}

// ========================================
// DELETE RECORDING TESTS
// ========================================

func TestDeleteRecording_Success_WithFile(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()
	fileURL := "https://storage.example.com/recordings/test.m4a"

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeAudio,
		Status:        RecordingStatusCompleted,
		FileURL:       &fileURL,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	storageClient.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
	repo.On("MarkRecordingDeleted", ctx, recordingID).Return(nil)

	err := svc.DeleteRecording(ctx, userID, recordingID)
	require.NoError(t, err)

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestDeleteRecording_Success_NoFile(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeAudio,
		Status:        RecordingStatusInitialized,
		FileURL:       nil, // No file uploaded yet
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	repo.On("MarkRecordingDeleted", ctx, recordingID).Return(nil)

	err := svc.DeleteRecording(ctx, userID, recordingID)
	require.NoError(t, err)

	// Storage Delete should NOT be called
	storageClient.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestDeleteRecording_NotFound(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	repo.On("GetRecording", ctx, recordingID).Return(nil, errors.New("not found"))

	err := svc.DeleteRecording(ctx, userID, recordingID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	repo.AssertExpectations(t)
}

func TestDeleteRecording_Unauthorized(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:     recordingID,
		UserID: otherUserID,
		Status: RecordingStatusCompleted,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	err := svc.DeleteRecording(ctx, userID, recordingID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

// ========================================
// EXTEND RETENTION TESTS
// ========================================

func TestExtendRetention_Extended(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	recordingID := uuid.New()

	beforeCall := time.Now()

	repo.On("UpdateRetentionPolicy", ctx, recordingID, RetentionExtended, mock.AnythingOfType("time.Time")).
		Return(nil)

	err := svc.ExtendRetention(ctx, recordingID, RetentionExtended)
	require.NoError(t, err)

	// Verify the expiry was set to approximately 30 days from now
	repo.AssertCalled(t, "UpdateRetentionPolicy", ctx, recordingID, RetentionExtended,
		mock.MatchedBy(func(t time.Time) bool {
			// Should be approximately 30 days from now
			expectedMin := beforeCall.AddDate(0, 0, 29)
			expectedMax := beforeCall.AddDate(0, 0, 31)
			return t.After(expectedMin) && t.Before(expectedMax)
		}))

	repo.AssertExpectations(t)
}

func TestExtendRetention_Permanent(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	recordingID := uuid.New()

	beforeCall := time.Now()

	repo.On("UpdateRetentionPolicy", ctx, recordingID, RetentionPermanent, mock.AnythingOfType("time.Time")).
		Return(nil)

	err := svc.ExtendRetention(ctx, recordingID, RetentionPermanent)
	require.NoError(t, err)

	// Verify the expiry was set to approximately 10 years from now
	repo.AssertCalled(t, "UpdateRetentionPolicy", ctx, recordingID, RetentionPermanent,
		mock.MatchedBy(func(t time.Time) bool {
			// Should be approximately 10 years from now
			expectedMin := beforeCall.AddDate(9, 11, 0)
			expectedMax := beforeCall.AddDate(10, 1, 0)
			return t.After(expectedMin) && t.Before(expectedMax)
		}))

	repo.AssertExpectations(t)
}

func TestExtendRetention_Standard(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	recordingID := uuid.New()

	beforeCall := time.Now()

	repo.On("UpdateRetentionPolicy", ctx, recordingID, RetentionStandard, mock.AnythingOfType("time.Time")).
		Return(nil)

	err := svc.ExtendRetention(ctx, recordingID, RetentionStandard)
	require.NoError(t, err)

	// Verify the expiry was set to approximately 7 days from now
	repo.AssertCalled(t, "UpdateRetentionPolicy", ctx, recordingID, RetentionStandard,
		mock.MatchedBy(func(t time.Time) bool {
			expectedMin := beforeCall.AddDate(0, 0, 6)
			expectedMax := beforeCall.AddDate(0, 0, 8)
			return t.After(expectedMin) && t.Before(expectedMax)
		}))

	repo.AssertExpectations(t)
}

func TestExtendRetention_RepoError(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	recordingID := uuid.New()

	repo.On("UpdateRetentionPolicy", ctx, recordingID, RetentionExtended, mock.AnythingOfType("time.Time")).
		Return(errors.New("database error"))

	err := svc.ExtendRetention(ctx, recordingID, RetentionExtended)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	repo.AssertExpectations(t)
}

// ========================================
// GET RECORDING BY ID TESTS (ACCESS CONTROL)
// ========================================

func TestGetRecording_Success_Owner(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()
	fileURL := "https://storage.example.com/recordings/test.m4a"

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeAudio,
		Status:        RecordingStatusCompleted,
		FileURL:       &fileURL,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	storageClient.On("GetPresignedDownloadURL", ctx, mock.AnythingOfType("string"), time.Hour).
		Return(&storage.PresignedURLResult{
			URL:       "https://storage.example.com/signed/test.m4a",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)
	repo.On("LogAccess", ctx, mock.AnythingOfType("*recording.RecordingAccessLog")).Return(nil)

	resp, err := svc.GetRecording(ctx, userID, recordingID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotNil(t, resp.Recording)
	assert.Equal(t, recordingID, resp.Recording.ID)
	assert.NotEmpty(t, resp.AccessURL)
	assert.Equal(t, 3600, resp.ExpiresIn)

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestGetRecording_NotFound(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	repo.On("GetRecording", ctx, recordingID).Return(nil, errors.New("not found"))

	resp, err := svc.GetRecording(ctx, userID, recordingID)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")

	repo.AssertExpectations(t)
}

func TestGetRecording_Unauthorized_DifferentUser(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:     recordingID,
		UserID: otherUserID, // Different user owns this recording
		Status: RecordingStatusCompleted,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	resp, err := svc.GetRecording(ctx, userID, recordingID)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

func TestGetRecording_NoAccessURL_NotCompleted(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	recordingID := uuid.New()

	recording := &RideRecording{
		ID:     recordingID,
		UserID: userID,
		Status: RecordingStatusRecording, // Still recording, not completed
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

	resp, err := svc.GetRecording(ctx, userID, recordingID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotNil(t, resp.Recording)
	assert.Empty(t, resp.AccessURL) // No access URL since not completed
	assert.Equal(t, 0, resp.ExpiresIn)

	// Storage should NOT be called
	storageClient.AssertNotCalled(t, "GetPresignedDownloadURL", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

// ========================================
// GET RECORDINGS FOR RIDE TESTS
// ========================================

func TestGetRecordingsForRide_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	recordings := []*RideRecording{
		{ID: uuid.New(), RideID: rideID, UserID: userID, RecordingType: RecordingTypeAudio},
		{ID: uuid.New(), RideID: rideID, UserID: uuid.New(), RecordingType: RecordingTypeVideo},
	}

	repo.On("GetRecordingsByRide", ctx, rideID).Return(recordings, nil)

	result, err := svc.GetRecordingsForRide(ctx, userID, rideID)
	require.NoError(t, err)
	require.Len(t, result, 2)

	repo.AssertExpectations(t)
}

func TestGetRecordingsForRide_Empty(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetRecordingsByRide", ctx, rideID).Return([]*RideRecording{}, nil)

	result, err := svc.GetRecordingsForRide(ctx, userID, rideID)
	require.NoError(t, err)
	require.Empty(t, result)

	repo.AssertExpectations(t)
}

func TestGetRecordingsForRide_RepoError(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetRecordingsByRide", ctx, rideID).Return(nil, errors.New("database error"))

	result, err := svc.GetRecordingsForRide(ctx, userID, rideID)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "internal server error")

	repo.AssertExpectations(t)
}

// ========================================
// CLEANUP EXPIRED RECORDINGS TESTS
// ========================================

func TestCleanupExpiredRecordings_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	fileURL := "https://storage.example.com/recordings/test.m4a"

	expiredRecordings := []*RideRecording{
		{
			ID:            uuid.New(),
			RideID:        uuid.New(),
			UserID:        uuid.New(),
			RecordingType: RecordingTypeAudio,
			FileURL:       &fileURL,
		},
		{
			ID:            uuid.New(),
			RideID:        uuid.New(),
			UserID:        uuid.New(),
			RecordingType: RecordingTypeVideo,
			FileURL:       &fileURL,
		},
	}

	repo.On("GetExpiredRecordings", ctx, 100).Return(expiredRecordings, nil)
	storageClient.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
	repo.On("MarkRecordingDeleted", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)

	deleted, err := svc.CleanupExpiredRecordings(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestCleanupExpiredRecordings_NoExpired(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()

	repo.On("GetExpiredRecordings", ctx, 100).Return([]*RideRecording{}, nil)

	deleted, err := svc.CleanupExpiredRecordings(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)

	repo.AssertExpectations(t)
}

func TestCleanupExpiredRecordings_PartialFailure(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	fileURL := "https://storage.example.com/recordings/test.m4a"

	rec1ID := uuid.New()
	rec2ID := uuid.New()

	expiredRecordings := []*RideRecording{
		{ID: rec1ID, RideID: uuid.New(), UserID: uuid.New(), RecordingType: RecordingTypeAudio, FileURL: &fileURL},
		{ID: rec2ID, RideID: uuid.New(), UserID: uuid.New(), RecordingType: RecordingTypeAudio, FileURL: &fileURL},
	}

	repo.On("GetExpiredRecordings", ctx, 100).Return(expiredRecordings, nil)
	storageClient.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
	repo.On("MarkRecordingDeleted", ctx, rec1ID).Return(nil)
	repo.On("MarkRecordingDeleted", ctx, rec2ID).Return(errors.New("delete failed"))

	deleted, err := svc.CleanupExpiredRecordings(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted) // Only one succeeded

	repo.AssertExpectations(t)
}

// ========================================
// SETTINGS TESTS
// ========================================

func TestGetSettings_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()

	settings := &RecordingSettings{
		UserID:           userID,
		RecordingEnabled: true,
		DefaultType:      RecordingTypeAudio,
		DefaultQuality:   "high",
	}

	repo.On("GetSettings", ctx, userID).Return(settings, nil)

	result, err := svc.GetSettings(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, userID, result.UserID)
	assert.True(t, result.RecordingEnabled)
	assert.Equal(t, RecordingTypeAudio, result.DefaultType)
	assert.Equal(t, "high", result.DefaultQuality)

	repo.AssertExpectations(t)
}

func TestUpdateSettings_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	userID := uuid.New()

	settings := &RecordingSettings{
		RecordingEnabled: true,
		DefaultType:      RecordingTypeVideo,
		DefaultQuality:   "high",
	}

	repo.On("UpsertSettings", ctx, mock.MatchedBy(func(s *RecordingSettings) bool {
		return s.UserID == userID
	})).Return(nil)

	err := svc.UpdateSettings(ctx, userID, settings)
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

// ========================================
// ADMIN TESTS
// ========================================

func TestAdminGetRecording_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	adminID := uuid.New()
	recordingID := uuid.New()
	rideID := uuid.New()
	userID := uuid.New()
	fileURL := "https://storage.example.com/recordings/test.m4a"

	recording := &RideRecording{
		ID:            recordingID,
		RideID:        rideID,
		UserID:        userID,
		RecordingType: RecordingTypeAudio,
		Status:        RecordingStatusCompleted,
		FileURL:       &fileURL,
	}

	repo.On("GetRecording", ctx, recordingID).Return(recording, nil)
	storageClient.On("GetPresignedDownloadURL", ctx, mock.AnythingOfType("string"), time.Hour).
		Return(&storage.PresignedURLResult{
			URL:       "https://storage.example.com/signed/admin-access.m4a",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil)
	repo.On("LogAccess", ctx, mock.MatchedBy(func(log *RecordingAccessLog) bool {
		return log.AccessType == "admin_view" && log.Reason != nil && *log.Reason == "dispute investigation"
	})).Return(nil)

	resp, err := svc.AdminGetRecording(ctx, adminID, recordingID, "dispute investigation")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotNil(t, resp.Recording)
	assert.NotEmpty(t, resp.AccessURL)

	repo.AssertExpectations(t)
	storageClient.AssertExpectations(t)
}

func TestAdminGetRecording_NotFound(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()
	adminID := uuid.New()
	recordingID := uuid.New()

	repo.On("GetRecording", ctx, recordingID).Return(nil, errors.New("not found"))

	resp, err := svc.AdminGetRecording(ctx, adminID, recordingID, "investigation")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")

	repo.AssertExpectations(t)
}

func TestGetRecordingStats_Success(t *testing.T) {
	repo := new(mockRepo)
	storageClient := new(mockStorage)
	svc := newTestService(repo, storageClient)
	ctx := context.Background()

	stats := &RecordingStatsResponse{
		TotalRecordings:    1000,
		TotalDurationHours: 500.5,
		TotalStorageGB:     250.25,
		ActiveRecordings:   10,
		PendingUploads:     5,
		RecordingsByType:   map[string]int64{"audio": 800, "video": 200},
		RecordingsByStatus: map[string]int64{"completed": 900, "recording": 50},
	}

	repo.On("GetRecordingStats", ctx).Return(stats, nil)

	result, err := svc.GetRecordingStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(1000), result.TotalRecordings)
	assert.Equal(t, 500.5, result.TotalDurationHours)
	assert.Equal(t, 250.25, result.TotalStorageGB)
	assert.Equal(t, 10, result.ActiveRecordings)
	assert.Equal(t, 5, result.PendingUploads)

	repo.AssertExpectations(t)
}

// ========================================
// HELPER FUNCTION TESTS
// ========================================

func TestGetFormatForType(t *testing.T) {
	tests := []struct {
		recordingType RecordingType
		expectedFormat string
	}{
		{RecordingTypeAudio, "m4a"},
		{RecordingTypeVideo, "mp4"},
		{RecordingType("unknown"), "m4a"}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.recordingType), func(t *testing.T) {
			result := getFormatForType(tt.recordingType)
			assert.Equal(t, tt.expectedFormat, result)
		})
	}
}

func TestGetExtensionForType(t *testing.T) {
	tests := []struct {
		recordingType     RecordingType
		expectedExtension string
	}{
		{RecordingTypeAudio, "m4a"},
		{RecordingTypeVideo, "mp4"},
		{RecordingType(""), "m4a"}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.recordingType), func(t *testing.T) {
			result := getExtensionForType(tt.recordingType)
			assert.Equal(t, tt.expectedExtension, result)
		})
	}
}

// ========================================
// SERVICE CONFIGURATION TESTS
// ========================================

func TestNewService_DefaultConfig(t *testing.T) {
	repo := &Repository{}
	storage := new(mockStorage)

	cfg := Config{
		BucketName:  "test-bucket",
		MaxDuration: 0, // Should default to 7200
		MaxFileSize: 0, // Should default to 500MB
	}

	svc := NewService(repo, storage, cfg)

	assert.Equal(t, 7200, svc.maxDuration)
	assert.Equal(t, int64(500*1024*1024), svc.maxFileSize)
	assert.Equal(t, "test-bucket", svc.bucketName)
}

func TestNewService_CustomConfig(t *testing.T) {
	repo := &Repository{}
	storage := new(mockStorage)

	cfg := Config{
		BucketName:  "custom-bucket",
		MaxDuration: 3600,
		MaxFileSize: 1024 * 1024 * 1024, // 1GB
	}

	svc := NewService(repo, storage, cfg)

	assert.Equal(t, 3600, svc.maxDuration)
	assert.Equal(t, int64(1024*1024*1024), svc.maxFileSize)
	assert.Equal(t, "custom-bucket", svc.bucketName)
}

// ========================================
// RECORDING STATUS CONSTANTS TESTS
// ========================================

func TestRecordingStatus_Constants(t *testing.T) {
	assert.Equal(t, RecordingStatus("initialized"), RecordingStatusInitialized)
	assert.Equal(t, RecordingStatus("recording"), RecordingStatusRecording)
	assert.Equal(t, RecordingStatus("paused"), RecordingStatusPaused)
	assert.Equal(t, RecordingStatus("stopped"), RecordingStatusStopped)
	assert.Equal(t, RecordingStatus("uploading"), RecordingStatusUploading)
	assert.Equal(t, RecordingStatus("uploaded"), RecordingStatusUploaded)
	assert.Equal(t, RecordingStatus("processing"), RecordingStatusProcessing)
	assert.Equal(t, RecordingStatus("completed"), RecordingStatusCompleted)
	assert.Equal(t, RecordingStatus("failed"), RecordingStatusFailed)
	assert.Equal(t, RecordingStatus("deleted"), RecordingStatusDeleted)
}

func TestRecordingType_Constants(t *testing.T) {
	assert.Equal(t, RecordingType("audio"), RecordingTypeAudio)
	assert.Equal(t, RecordingType("video"), RecordingTypeVideo)
}

func TestRetentionPolicy_Constants(t *testing.T) {
	assert.Equal(t, RetentionPolicy("standard"), RetentionStandard)
	assert.Equal(t, RetentionPolicy("extended"), RetentionExtended)
	assert.Equal(t, RetentionPolicy("permanent"), RetentionPermanent)
}

// ========================================
// TABLE-DRIVEN FILE SIZE VALIDATION TESTS
// ========================================

func TestCompleteUpload_FileSizeValidation(t *testing.T) {
	tests := []struct {
		name          string
		fileSize      int64
		expectError   bool
		errorContains string
	}{
		{
			name:        "1 byte - allowed",
			fileSize:    1,
			expectError: false,
		},
		{
			name:        "1MB - allowed",
			fileSize:    1 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "100MB - allowed",
			fileSize:    100 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "499MB - allowed",
			fileSize:    499 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "500MB exactly - allowed",
			fileSize:    500 * 1024 * 1024,
			expectError: false,
		},
		{
			name:          "500MB + 1 byte - rejected",
			fileSize:      500*1024*1024 + 1,
			expectError:   true,
			errorContains: "file size exceeds maximum allowed",
		},
		{
			name:          "501MB - rejected",
			fileSize:      501 * 1024 * 1024,
			expectError:   true,
			errorContains: "file size exceeds maximum allowed",
		},
		{
			name:          "1GB - rejected",
			fileSize:      1024 * 1024 * 1024,
			expectError:   true,
			errorContains: "file size exceeds maximum allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			storageClient := new(mockStorage)
			svc := newTestService(repo, storageClient)
			ctx := context.Background()
			userID := uuid.New()
			recordingID := uuid.New()
			rideID := uuid.New()

			recording := &RideRecording{
				ID:            recordingID,
				RideID:        rideID,
				UserID:        userID,
				RecordingType: RecordingTypeAudio,
				Status:        RecordingStatusStopped,
			}

			repo.On("GetRecording", ctx, recordingID).Return(recording, nil)

			if !tt.expectError {
				storageClient.On("GetURL", mock.AnythingOfType("string")).Return("https://storage.example.com/test.m4a")
				repo.On("UpdateRecordingCompleted", ctx, recordingID, mock.AnythingOfType("string"), tt.fileSize, 600).Return(nil)
				repo.On("UpdateRecordingProcessed", mock.Anything, recordingID, (*string)(nil)).Return(nil).Maybe()
			}

			req := &CompleteUploadRequest{
				RecordingID:     recordingID,
				TotalSize:       tt.fileSize,
				DurationSeconds: 600,
			}

			err := svc.CompleteUpload(ctx, userID, req)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
