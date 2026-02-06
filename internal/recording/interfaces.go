package recording

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines all public methods of the Repository
type RepositoryInterface interface {
	// Recording Operations
	CreateRecording(ctx context.Context, recording *RideRecording) error
	GetRecording(ctx context.Context, recordingID uuid.UUID) (*RideRecording, error)
	GetRecordingsByRide(ctx context.Context, rideID uuid.UUID) ([]*RideRecording, error)
	GetActiveRecordingForRide(ctx context.Context, rideID, userID uuid.UUID) (*RideRecording, error)
	UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status RecordingStatus) error
	UpdateRecordingStopped(ctx context.Context, recordingID uuid.UUID, endedAt time.Time) error
	UpdateRecordingUpload(ctx context.Context, recordingID uuid.UUID, uploadID string, chunksReceived, totalChunks int) error
	UpdateRecordingCompleted(ctx context.Context, recordingID uuid.UUID, fileURL string, fileSize int64, durationSeconds int) error
	UpdateRecordingProcessed(ctx context.Context, recordingID uuid.UUID, thumbnailURL *string) error
	UpdateRetentionPolicy(ctx context.Context, recordingID uuid.UUID, policy RetentionPolicy, newExpiry time.Time) error
	GetExpiredRecordings(ctx context.Context, limit int) ([]*RideRecording, error)
	MarkRecordingDeleted(ctx context.Context, recordingID uuid.UUID) error

	// Consent Operations
	CreateConsent(ctx context.Context, consent *RecordingConsent) error
	GetConsent(ctx context.Context, rideID, userID uuid.UUID) (*RecordingConsent, error)
	CheckAllConsented(ctx context.Context, rideID uuid.UUID) (bool, error)

	// Access Log Operations
	LogAccess(ctx context.Context, log *RecordingAccessLog) error
	GetAccessLogs(ctx context.Context, recordingID uuid.UUID) ([]*RecordingAccessLog, error)

	// Settings Operations
	GetSettings(ctx context.Context, userID uuid.UUID) (*RecordingSettings, error)
	UpsertSettings(ctx context.Context, settings *RecordingSettings) error

	// Statistics
	GetRecordingStats(ctx context.Context) (*RecordingStatsResponse, error)
}
