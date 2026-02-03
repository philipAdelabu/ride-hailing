package recording

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"go.uber.org/zap"
)

// Service handles recording business logic
type Service struct {
	repo          *Repository
	storageClient storage.Storage
	bucketName    string
	maxDuration   int   // Maximum recording duration in seconds
	maxFileSize   int64 // Maximum file size in bytes
}

// Config holds service configuration
type Config struct {
	BucketName  string
	MaxDuration int   // seconds
	MaxFileSize int64 // bytes
}

// NewService creates a new recording service
func NewService(repo *Repository, storageClient storage.Storage, cfg Config) *Service {
	maxDuration := cfg.MaxDuration
	if maxDuration == 0 {
		maxDuration = 7200 // 2 hours default
	}

	maxFileSize := cfg.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = 500 * 1024 * 1024 // 500MB default
	}

	return &Service{
		repo:          repo,
		storageClient: storageClient,
		bucketName:    cfg.BucketName,
		maxDuration:   maxDuration,
		maxFileSize:   maxFileSize,
	}
}

// ========================================
// RECORDING OPERATIONS
// ========================================

// StartRecording starts a new recording for a ride
func (s *Service) StartRecording(ctx context.Context, userID uuid.UUID, userType string, req *StartRecordingRequest) (*StartRecordingResponse, error) {
	// Check if there's already an active recording
	existing, _ := s.repo.GetActiveRecordingForRide(ctx, req.RideID, userID)
	if existing != nil {
		return nil, common.NewBadRequestError("recording already in progress for this ride", nil)
	}

	// Check consent from other party
	// In production, you'd check if the other party has consented
	// For now, we'll proceed (consent check would be done at ride start)

	// Determine quality
	quality := req.Quality
	if quality == "" {
		quality = "medium"
	}

	// Calculate expiry based on retention policy
	expiresAt := time.Now().AddDate(0, 0, 7) // Default 7 days

	// Create recording record
	recording := &RideRecording{
		ID:              uuid.New(),
		RideID:          req.RideID,
		UserID:          userID,
		UserType:        userType,
		RecordingType:   req.RecordingType,
		Status:          RecordingStatusInitialized,
		RetentionPolicy: RetentionStandard,
		Format:          getFormatForType(req.RecordingType),
		Quality:         quality,
		Encrypted:       true,
		StartedAt:       time.Now(),
		ExpiresAt:       expiresAt,
		DeviceInfo:      req.DeviceInfo,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateRecording(ctx, recording); err != nil {
		return nil, common.NewInternalServerError("failed to create recording")
	}

	// Generate pre-signed upload URL
	objectKey := fmt.Sprintf("recordings/%s/%s/%s.%s",
		req.RideID.String(),
		userID.String(),
		recording.ID.String(),
		getExtensionForType(req.RecordingType),
	)

	contentType := "audio/m4a"
	if req.RecordingType == RecordingTypeVideo {
		contentType = "video/mp4"
	}
	presigned, err := s.storageClient.GetPresignedUploadURL(ctx, objectKey, contentType, time.Hour)
	if err != nil {
		logger.Error("failed to generate upload URL", zap.Error(err))
		return nil, common.NewInternalServerError("failed to generate upload URL")
	}
	uploadURL := presigned.URL

	// Update status to recording
	_ = s.repo.UpdateRecordingStatus(ctx, recording.ID, RecordingStatusRecording)

	logger.Info("Recording started",
		zap.String("recording_id", recording.ID.String()),
		zap.String("ride_id", req.RideID.String()),
		zap.String("user_id", userID.String()),
		zap.String("type", string(req.RecordingType)),
	)

	return &StartRecordingResponse{
		RecordingID: recording.ID,
		UploadURL:   uploadURL,
		UploadID:    objectKey,
		MaxDuration: s.maxDuration,
		MaxFileSize: s.maxFileSize,
		Status:      string(RecordingStatusRecording),
		Message:     "Recording started. Upload data to the provided URL.",
	}, nil
}

// StopRecording stops an active recording
func (s *Service) StopRecording(ctx context.Context, userID uuid.UUID, req *StopRecordingRequest) (*StopRecordingResponse, error) {
	recording, err := s.repo.GetRecording(ctx, req.RecordingID)
	if err != nil {
		return nil, common.NewNotFoundError("recording not found", err)
	}

	if recording.UserID != userID {
		return nil, common.NewForbiddenError("not authorized to stop this recording")
	}

	if recording.Status != RecordingStatusRecording && recording.Status != RecordingStatusPaused {
		return nil, common.NewBadRequestError("recording is not active", nil)
	}

	endedAt := time.Now()
	duration := int(endedAt.Sub(recording.StartedAt).Seconds())

	if err := s.repo.UpdateRecordingStopped(ctx, req.RecordingID, endedAt); err != nil {
		return nil, common.NewInternalServerError("failed to stop recording")
	}

	logger.Info("Recording stopped",
		zap.String("recording_id", req.RecordingID.String()),
		zap.Int("duration_seconds", duration),
	)

	return &StopRecordingResponse{
		RecordingID:     req.RecordingID,
		Status:          string(RecordingStatusStopped),
		DurationSeconds: duration,
		Message:         "Recording stopped. Complete the upload to finalize.",
	}, nil
}

// CompleteUpload finalizes a recording after upload is complete
func (s *Service) CompleteUpload(ctx context.Context, userID uuid.UUID, req *CompleteUploadRequest) error {
	recording, err := s.repo.GetRecording(ctx, req.RecordingID)
	if err != nil {
		return common.NewNotFoundError("recording not found", err)
	}

	if recording.UserID != userID {
		return common.NewForbiddenError("not authorized to complete this upload")
	}

	if req.TotalSize > s.maxFileSize {
		return common.NewBadRequestError(fmt.Sprintf("file size exceeds maximum allowed (%d bytes)", s.maxFileSize), nil)
	}

	// Generate the file URL
	objectKey := fmt.Sprintf("recordings/%s/%s/%s.%s",
		recording.RideID.String(),
		recording.UserID.String(),
		recording.ID.String(),
		getExtensionForType(recording.RecordingType),
	)

	fileURL := s.storageClient.GetURL(objectKey)

	if err := s.repo.UpdateRecordingCompleted(ctx, req.RecordingID, fileURL, req.TotalSize, req.DurationSeconds); err != nil {
		return common.NewInternalServerError("failed to complete upload")
	}

	// Trigger async processing (transcoding, thumbnail generation, etc.)
	go s.processRecording(context.Background(), recording.ID)

	logger.Info("Recording upload completed",
		zap.String("recording_id", req.RecordingID.String()),
		zap.Int64("file_size", req.TotalSize),
		zap.Int("duration", req.DurationSeconds),
	)

	return nil
}

// GetRecording gets a recording with access URL
func (s *Service) GetRecording(ctx context.Context, userID uuid.UUID, recordingID uuid.UUID) (*GetRecordingResponse, error) {
	recording, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return nil, common.NewNotFoundError("recording not found", err)
	}

	// Check authorization - user must be the recorder or admin
	// In production, you'd also allow the other ride participant
	if recording.UserID != userID {
		return nil, common.NewForbiddenError("not authorized to access this recording")
	}

	response := &GetRecordingResponse{
		Recording: recording,
	}

	// Generate temporary access URL if file exists
	if recording.FileURL != nil && recording.Status == RecordingStatusCompleted {
		objectKey := fmt.Sprintf("recordings/%s/%s/%s.%s",
			recording.RideID.String(),
			recording.UserID.String(),
			recording.ID.String(),
			getExtensionForType(recording.RecordingType),
		)

		presigned, err := s.storageClient.GetPresignedDownloadURL(ctx, objectKey, time.Hour)
		if err == nil {
			response.AccessURL = presigned.URL
			response.ExpiresIn = 3600 // 1 hour
		}

		// Log access
		s.logAccess(ctx, recording.ID, userID, "view")
	}

	return response, nil
}

// GetRecordingsForRide gets all recordings for a ride
func (s *Service) GetRecordingsForRide(ctx context.Context, userID uuid.UUID, rideID uuid.UUID) ([]*RideRecording, error) {
	recordings, err := s.repo.GetRecordingsByRide(ctx, rideID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get recordings")
	}

	return recordings, nil
}

// DeleteRecording soft-deletes a recording
func (s *Service) DeleteRecording(ctx context.Context, userID uuid.UUID, recordingID uuid.UUID) error {
	recording, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return common.NewNotFoundError("recording not found", err)
	}

	if recording.UserID != userID {
		return common.NewForbiddenError("not authorized to delete this recording")
	}

	// Don't allow deletion of recordings involved in active disputes
	// In production, you'd check for active disputes here

	// Delete from storage
	if recording.FileURL != nil {
		objectKey := fmt.Sprintf("recordings/%s/%s/%s.%s",
			recording.RideID.String(),
			recording.UserID.String(),
			recording.ID.String(),
			getExtensionForType(recording.RecordingType),
		)
		_ = s.storageClient.Delete(ctx, objectKey)
	}

	if err := s.repo.MarkRecordingDeleted(ctx, recordingID); err != nil {
		return common.NewInternalServerError("failed to delete recording")
	}

	logger.Info("Recording deleted",
		zap.String("recording_id", recordingID.String()),
		zap.String("user_id", userID.String()),
	)

	return nil
}

// ========================================
// CONSENT OPERATIONS
// ========================================

// RecordConsent records a user's consent for recording
func (s *Service) RecordConsent(ctx context.Context, userID uuid.UUID, userType string, req *RecordingConsentRequest, ipAddress, userAgent *string) error {
	consent := &RecordingConsent{
		ID:          uuid.New(),
		RideID:      req.RideID,
		UserID:      userID,
		UserType:    userType,
		Consented:   req.Consented,
		ConsentedAt: time.Now(),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	if err := s.repo.CreateConsent(ctx, consent); err != nil {
		return common.NewInternalServerError("failed to record consent")
	}

	logger.Info("Recording consent recorded",
		zap.String("ride_id", req.RideID.String()),
		zap.String("user_id", userID.String()),
		zap.Bool("consented", req.Consented),
	)

	return nil
}

// CheckConsent checks if recording is allowed for a ride
func (s *Service) CheckConsent(ctx context.Context, rideID uuid.UUID) (bool, error) {
	return s.repo.CheckAllConsented(ctx, rideID)
}

// ========================================
// SETTINGS OPERATIONS
// ========================================

// GetSettings gets recording settings for a user
func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (*RecordingSettings, error) {
	return s.repo.GetSettings(ctx, userID)
}

// UpdateSettings updates recording settings for a user
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, settings *RecordingSettings) error {
	settings.UserID = userID
	return s.repo.UpsertSettings(ctx, settings)
}

// ========================================
// RETENTION MANAGEMENT
// ========================================

// ExtendRetention extends the retention period for a recording (e.g., for disputes)
func (s *Service) ExtendRetention(ctx context.Context, recordingID uuid.UUID, policy RetentionPolicy) error {
	var newExpiry time.Time
	switch policy {
	case RetentionExtended:
		newExpiry = time.Now().AddDate(0, 0, 30) // 30 days
	case RetentionPermanent:
		newExpiry = time.Now().AddDate(10, 0, 0) // 10 years
	default:
		newExpiry = time.Now().AddDate(0, 0, 7) // 7 days
	}

	return s.repo.UpdateRetentionPolicy(ctx, recordingID, policy, newExpiry)
}

// CleanupExpiredRecordings deletes expired recordings
func (s *Service) CleanupExpiredRecordings(ctx context.Context) (int, error) {
	recordings, err := s.repo.GetExpiredRecordings(ctx, 100)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, rec := range recordings {
		// Delete from storage
		if rec.FileURL != nil {
			objectKey := fmt.Sprintf("recordings/%s/%s/%s.%s",
				rec.RideID.String(),
				rec.UserID.String(),
				rec.ID.String(),
				getExtensionForType(rec.RecordingType),
			)
			_ = s.storageClient.Delete(ctx, objectKey)
		}

		// Mark as deleted
		if err := s.repo.MarkRecordingDeleted(ctx, rec.ID); err != nil {
			logger.Error("failed to delete expired recording", zap.Error(err))
			continue
		}
		deleted++
	}

	logger.Info("Cleaned up expired recordings", zap.Int("count", deleted))
	return deleted, nil
}

// ========================================
// ADMIN OPERATIONS
// ========================================

// GetRecordingStats gets recording statistics
func (s *Service) GetRecordingStats(ctx context.Context) (*RecordingStatsResponse, error) {
	return s.repo.GetRecordingStats(ctx)
}

// AdminGetRecording gets a recording for admin review
func (s *Service) AdminGetRecording(ctx context.Context, adminID, recordingID uuid.UUID, reason string) (*GetRecordingResponse, error) {
	recording, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return nil, common.NewNotFoundError("recording not found", err)
	}

	response := &GetRecordingResponse{
		Recording: recording,
	}

	// Generate access URL
	if recording.FileURL != nil && recording.Status == RecordingStatusCompleted {
		objectKey := fmt.Sprintf("recordings/%s/%s/%s.%s",
			recording.RideID.String(),
			recording.UserID.String(),
			recording.ID.String(),
			getExtensionForType(recording.RecordingType),
		)

		presigned, err := s.storageClient.GetPresignedDownloadURL(ctx, objectKey, time.Hour)
		if err == nil {
			response.AccessURL = presigned.URL
			response.ExpiresIn = 3600
		}

		// Log admin access with reason
		s.logAccessWithReason(ctx, recording.ID, adminID, "admin_view", reason)
	}

	return response, nil
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func (s *Service) processRecording(ctx context.Context, recordingID uuid.UUID) {
	// In production, this would:
	// 1. Transcode to different quality levels
	// 2. Generate thumbnails for video
	// 3. Run audio/video analysis
	// 4. Update recording status to completed

	// For now, just mark as completed
	time.Sleep(2 * time.Second) // Simulate processing
	_ = s.repo.UpdateRecordingProcessed(ctx, recordingID, nil)
}

func (s *Service) logAccess(ctx context.Context, recordingID, userID uuid.UUID, accessType string) {
	log := &RecordingAccessLog{
		ID:          uuid.New(),
		RecordingID: recordingID,
		AccessedBy:  userID,
		AccessType:  accessType,
		AccessedAt:  time.Now(),
	}
	_ = s.repo.LogAccess(ctx, log)
}

func (s *Service) logAccessWithReason(ctx context.Context, recordingID, userID uuid.UUID, accessType, reason string) {
	log := &RecordingAccessLog{
		ID:          uuid.New(),
		RecordingID: recordingID,
		AccessedBy:  userID,
		AccessType:  accessType,
		Reason:      &reason,
		AccessedAt:  time.Now(),
	}
	_ = s.repo.LogAccess(ctx, log)
}

func getFormatForType(recordingType RecordingType) string {
	switch recordingType {
	case RecordingTypeVideo:
		return "mp4"
	case RecordingTypeAudio:
		return "m4a"
	default:
		return "m4a"
	}
}

func getExtensionForType(recordingType RecordingType) string {
	switch recordingType {
	case RecordingTypeVideo:
		return "mp4"
	case RecordingTypeAudio:
		return "m4a"
	default:
		return "m4a"
	}
}
