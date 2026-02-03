package recording

import (
	"time"

	"github.com/google/uuid"
)

// RecordingType represents the type of recording
type RecordingType string

const (
	RecordingTypeAudio RecordingType = "audio"
	RecordingTypeVideo RecordingType = "video"
)

// RecordingStatus represents the status of a recording
type RecordingStatus string

const (
	RecordingStatusInitialized RecordingStatus = "initialized"
	RecordingStatusRecording   RecordingStatus = "recording"
	RecordingStatusPaused      RecordingStatus = "paused"
	RecordingStatusStopped     RecordingStatus = "stopped"
	RecordingStatusUploading   RecordingStatus = "uploading"
	RecordingStatusUploaded    RecordingStatus = "uploaded"
	RecordingStatusProcessing  RecordingStatus = "processing"
	RecordingStatusCompleted   RecordingStatus = "completed"
	RecordingStatusFailed      RecordingStatus = "failed"
	RecordingStatusDeleted     RecordingStatus = "deleted"
)

// RetentionPolicy defines how long recordings are kept
type RetentionPolicy string

const (
	RetentionStandard  RetentionPolicy = "standard"  // 7 days
	RetentionExtended  RetentionPolicy = "extended"  // 30 days (for disputes)
	RetentionPermanent RetentionPolicy = "permanent" // For incidents
)

// RideRecording represents an audio/video recording of a ride
type RideRecording struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	RideID          uuid.UUID       `json:"ride_id" db:"ride_id"`
	UserID          uuid.UUID       `json:"user_id" db:"user_id"`
	UserType        string          `json:"user_type" db:"user_type"` // rider, driver
	RecordingType   RecordingType   `json:"recording_type" db:"recording_type"`
	Status          RecordingStatus `json:"status" db:"status"`
	RetentionPolicy RetentionPolicy `json:"retention_policy" db:"retention_policy"`

	// File information
	FileURL          *string `json:"file_url,omitempty" db:"file_url"`
	ThumbnailURL     *string `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	FileSize         *int64  `json:"file_size_bytes,omitempty" db:"file_size"`
	DurationSeconds  *int    `json:"duration_seconds,omitempty" db:"duration_seconds"`
	Format           string  `json:"format" db:"format"`
	Quality          string  `json:"quality" db:"quality"` // low, medium, high
	Encrypted        bool    `json:"encrypted" db:"encrypted"`
	EncryptionKeyID  *string `json:"encryption_key_id,omitempty" db:"encryption_key_id"`

	// Upload tracking
	UploadID         *string    `json:"upload_id,omitempty" db:"upload_id"`
	ChunksReceived   int        `json:"chunks_received" db:"chunks_received"`
	TotalChunks      int        `json:"total_chunks" db:"total_chunks"`

	// Timestamps
	StartedAt        time.Time  `json:"started_at" db:"started_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	UploadedAt       *time.Time `json:"uploaded_at,omitempty" db:"uploaded_at"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`

	// Metadata
	DeviceInfo       *string    `json:"device_info,omitempty" db:"device_info"`
	StartLocation    *Location  `json:"start_location,omitempty"`
	EndLocation      *Location  `json:"end_location,omitempty"`
}

// Location represents a geographic point
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RecordingConsent represents consent for recording
type RecordingConsent struct {
	ID           uuid.UUID `json:"id" db:"id"`
	RideID       uuid.UUID `json:"ride_id" db:"ride_id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	UserType     string    `json:"user_type" db:"user_type"`
	Consented    bool      `json:"consented" db:"consented"`
	ConsentedAt  time.Time `json:"consented_at" db:"consented_at"`
	IPAddress    *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    *string   `json:"user_agent,omitempty" db:"user_agent"`
}

// RecordingAccessLog logs who accessed a recording
type RecordingAccessLog struct {
	ID           uuid.UUID `json:"id" db:"id"`
	RecordingID  uuid.UUID `json:"recording_id" db:"recording_id"`
	AccessedBy   uuid.UUID `json:"accessed_by" db:"accessed_by"`
	AccessType   string    `json:"access_type" db:"access_type"` // view, download, share
	Reason       *string   `json:"reason,omitempty" db:"reason"`
	IPAddress    *string   `json:"ip_address,omitempty" db:"ip_address"`
	AccessedAt   time.Time `json:"accessed_at" db:"accessed_at"`
}

// RecordingSettings represents user's recording preferences
type RecordingSettings struct {
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	RecordingEnabled    bool      `json:"recording_enabled" db:"recording_enabled"`
	DefaultType         RecordingType `json:"default_type" db:"default_type"`
	DefaultQuality      string    `json:"default_quality" db:"default_quality"` // low, medium, high
	AutoRecordNightRides bool     `json:"auto_record_night_rides" db:"auto_record_night_rides"`
	AutoRecordSOSRides  bool      `json:"auto_record_sos_rides" db:"auto_record_sos_rides"`
	NotifyOnRecording   bool      `json:"notify_on_recording" db:"notify_on_recording"`
	AllowDriverRecording bool     `json:"allow_driver_recording" db:"allow_driver_recording"` // For riders
	AllowRiderRecording  bool     `json:"allow_rider_recording" db:"allow_rider_recording"`  // For drivers
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// StartRecordingRequest represents a request to start recording
type StartRecordingRequest struct {
	RideID        uuid.UUID     `json:"ride_id" binding:"required"`
	RecordingType RecordingType `json:"recording_type" binding:"required"`
	Quality       string        `json:"quality,omitempty"` // low, medium, high
	DeviceInfo    *string       `json:"device_info,omitempty"`
}

// StartRecordingResponse represents the response after starting recording
type StartRecordingResponse struct {
	RecordingID uuid.UUID `json:"recording_id"`
	UploadURL   string    `json:"upload_url"` // Pre-signed URL for streaming upload
	UploadID    string    `json:"upload_id"`
	MaxDuration int       `json:"max_duration_seconds"`
	MaxFileSize int64     `json:"max_file_size_bytes"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

// StopRecordingRequest represents a request to stop recording
type StopRecordingRequest struct {
	RecordingID uuid.UUID `json:"recording_id" binding:"required"`
	Location    *Location `json:"location,omitempty"`
}

// StopRecordingResponse represents the response after stopping recording
type StopRecordingResponse struct {
	RecordingID     uuid.UUID `json:"recording_id"`
	Status          string    `json:"status"`
	DurationSeconds int       `json:"duration_seconds"`
	Message         string    `json:"message"`
}

// UploadChunkRequest represents a chunk upload request
type UploadChunkRequest struct {
	RecordingID uuid.UUID `json:"recording_id" binding:"required"`
	ChunkNumber int       `json:"chunk_number" binding:"required"`
	TotalChunks int       `json:"total_chunks" binding:"required"`
	ChunkData   []byte    `json:"chunk_data" binding:"required"`
	Checksum    string    `json:"checksum,omitempty"` // MD5 or SHA256
}

// CompleteUploadRequest represents a request to complete chunked upload
type CompleteUploadRequest struct {
	RecordingID    uuid.UUID `json:"recording_id" binding:"required"`
	TotalSize      int64     `json:"total_size" binding:"required"`
	DurationSeconds int      `json:"duration_seconds" binding:"required"`
	Checksum       string    `json:"checksum,omitempty"`
}

// GetRecordingResponse represents the response when getting a recording
type GetRecordingResponse struct {
	Recording   *RideRecording `json:"recording"`
	AccessURL   string         `json:"access_url,omitempty"` // Temporary signed URL
	ExpiresIn   int            `json:"access_url_expires_in_seconds,omitempty"`
}

// RecordingConsentRequest represents a consent request
type RecordingConsentRequest struct {
	RideID    uuid.UUID `json:"ride_id" binding:"required"`
	Consented bool      `json:"consented" binding:"required"`
}

// RecordingStatsResponse represents recording statistics
type RecordingStatsResponse struct {
	TotalRecordings     int64   `json:"total_recordings"`
	TotalDurationHours  float64 `json:"total_duration_hours"`
	TotalStorageGB      float64 `json:"total_storage_gb"`
	ActiveRecordings    int     `json:"active_recordings"`
	PendingUploads      int     `json:"pending_uploads"`
	RecordingsByType    map[string]int64 `json:"recordings_by_type"`
	RecordingsByStatus  map[string]int64 `json:"recordings_by_status"`
}
