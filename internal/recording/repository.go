package recording

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for recordings
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new recording repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// RECORDING OPERATIONS
// ========================================

// CreateRecording creates a new recording record
func (r *Repository) CreateRecording(ctx context.Context, recording *RideRecording) error {
	query := `
		INSERT INTO ride_recordings (
			id, ride_id, user_id, user_type, recording_type, status, retention_policy,
			format, quality, encrypted, started_at, expires_at, device_info, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
	`

	_, err := r.db.Exec(ctx, query,
		recording.ID, recording.RideID, recording.UserID, recording.UserType,
		recording.RecordingType, recording.Status, recording.RetentionPolicy,
		recording.Format, recording.Quality, recording.Encrypted,
		recording.StartedAt, recording.ExpiresAt, recording.DeviceInfo,
	)

	return err
}

// GetRecording gets a recording by ID
func (r *Repository) GetRecording(ctx context.Context, recordingID uuid.UUID) (*RideRecording, error) {
	query := `
		SELECT id, ride_id, user_id, user_type, recording_type, status, retention_policy,
		       file_url, thumbnail_url, file_size, duration_seconds, format, quality,
		       encrypted, encryption_key_id, upload_id, chunks_received, total_chunks,
		       started_at, ended_at, uploaded_at, processed_at, expires_at,
		       device_info, created_at, updated_at
		FROM ride_recordings
		WHERE id = $1
	`

	rec := &RideRecording{}
	err := r.db.QueryRow(ctx, query, recordingID).Scan(
		&rec.ID, &rec.RideID, &rec.UserID, &rec.UserType, &rec.RecordingType,
		&rec.Status, &rec.RetentionPolicy, &rec.FileURL, &rec.ThumbnailURL,
		&rec.FileSize, &rec.DurationSeconds, &rec.Format, &rec.Quality,
		&rec.Encrypted, &rec.EncryptionKeyID, &rec.UploadID, &rec.ChunksReceived,
		&rec.TotalChunks, &rec.StartedAt, &rec.EndedAt, &rec.UploadedAt,
		&rec.ProcessedAt, &rec.ExpiresAt, &rec.DeviceInfo, &rec.CreatedAt, &rec.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return rec, nil
}

// GetRecordingsByRide gets all recordings for a ride
func (r *Repository) GetRecordingsByRide(ctx context.Context, rideID uuid.UUID) ([]*RideRecording, error) {
	query := `
		SELECT id, ride_id, user_id, user_type, recording_type, status, retention_policy,
		       file_url, thumbnail_url, file_size, duration_seconds, format, quality,
		       encrypted, started_at, ended_at, expires_at, created_at
		FROM ride_recordings
		WHERE ride_id = $1 AND status != 'deleted'
		ORDER BY started_at ASC
	`

	rows, err := r.db.Query(ctx, query, rideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []*RideRecording
	for rows.Next() {
		rec := &RideRecording{}
		err := rows.Scan(
			&rec.ID, &rec.RideID, &rec.UserID, &rec.UserType, &rec.RecordingType,
			&rec.Status, &rec.RetentionPolicy, &rec.FileURL, &rec.ThumbnailURL,
			&rec.FileSize, &rec.DurationSeconds, &rec.Format, &rec.Quality,
			&rec.Encrypted, &rec.StartedAt, &rec.EndedAt, &rec.ExpiresAt, &rec.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		recordings = append(recordings, rec)
	}

	return recordings, nil
}

// GetActiveRecordingForRide gets the active recording for a ride by user
func (r *Repository) GetActiveRecordingForRide(ctx context.Context, rideID, userID uuid.UUID) (*RideRecording, error) {
	query := `
		SELECT id, ride_id, user_id, user_type, recording_type, status, retention_policy,
		       upload_id, started_at, created_at
		FROM ride_recordings
		WHERE ride_id = $1 AND user_id = $2 AND status IN ('initialized', 'recording', 'paused')
		ORDER BY created_at DESC
		LIMIT 1
	`

	rec := &RideRecording{}
	err := r.db.QueryRow(ctx, query, rideID, userID).Scan(
		&rec.ID, &rec.RideID, &rec.UserID, &rec.UserType, &rec.RecordingType,
		&rec.Status, &rec.RetentionPolicy, &rec.UploadID, &rec.StartedAt, &rec.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return rec, nil
}

// UpdateRecordingStatus updates the status of a recording
func (r *Repository) UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status RecordingStatus) error {
	query := `
		UPDATE ride_recordings
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, status, recordingID)
	return err
}

// UpdateRecordingStopped updates a recording when stopped
func (r *Repository) UpdateRecordingStopped(ctx context.Context, recordingID uuid.UUID, endedAt time.Time) error {
	query := `
		UPDATE ride_recordings
		SET status = 'stopped', ended_at = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, endedAt, recordingID)
	return err
}

// UpdateRecordingUpload updates upload progress
func (r *Repository) UpdateRecordingUpload(ctx context.Context, recordingID uuid.UUID, uploadID string, chunksReceived, totalChunks int) error {
	query := `
		UPDATE ride_recordings
		SET upload_id = $1, chunks_received = $2, total_chunks = $3,
		    status = CASE WHEN $2 = $3 THEN 'uploading' ELSE status END,
		    updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, uploadID, chunksReceived, totalChunks, recordingID)
	return err
}

// UpdateRecordingCompleted updates a recording when upload is complete
func (r *Repository) UpdateRecordingCompleted(ctx context.Context, recordingID uuid.UUID, fileURL string, fileSize int64, durationSeconds int) error {
	query := `
		UPDATE ride_recordings
		SET status = 'uploaded', file_url = $1, file_size = $2, duration_seconds = $3,
		    uploaded_at = NOW(), updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, fileURL, fileSize, durationSeconds, recordingID)
	return err
}

// UpdateRecordingProcessed marks a recording as processed
func (r *Repository) UpdateRecordingProcessed(ctx context.Context, recordingID uuid.UUID, thumbnailURL *string) error {
	query := `
		UPDATE ride_recordings
		SET status = 'completed', thumbnail_url = $1, processed_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, thumbnailURL, recordingID)
	return err
}

// UpdateRetentionPolicy updates the retention policy
func (r *Repository) UpdateRetentionPolicy(ctx context.Context, recordingID uuid.UUID, policy RetentionPolicy, newExpiry time.Time) error {
	query := `
		UPDATE ride_recordings
		SET retention_policy = $1, expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, policy, newExpiry, recordingID)
	return err
}

// GetExpiredRecordings gets recordings that have expired
func (r *Repository) GetExpiredRecordings(ctx context.Context, limit int) ([]*RideRecording, error) {
	query := `
		SELECT id, ride_id, user_id, file_url, status
		FROM ride_recordings
		WHERE expires_at < NOW() AND status NOT IN ('deleted', 'failed')
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []*RideRecording
	for rows.Next() {
		rec := &RideRecording{}
		err := rows.Scan(&rec.ID, &rec.RideID, &rec.UserID, &rec.FileURL, &rec.Status)
		if err != nil {
			return nil, err
		}
		recordings = append(recordings, rec)
	}

	return recordings, nil
}

// MarkRecordingDeleted marks a recording as deleted
func (r *Repository) MarkRecordingDeleted(ctx context.Context, recordingID uuid.UUID) error {
	query := `
		UPDATE ride_recordings
		SET status = 'deleted', file_url = NULL, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, recordingID)
	return err
}

// ========================================
// CONSENT OPERATIONS
// ========================================

// CreateConsent creates a consent record
func (r *Repository) CreateConsent(ctx context.Context, consent *RecordingConsent) error {
	query := `
		INSERT INTO recording_consents (id, ride_id, user_id, user_type, consented, consented_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (ride_id, user_id) DO UPDATE
		SET consented = EXCLUDED.consented, consented_at = EXCLUDED.consented_at
	`

	_, err := r.db.Exec(ctx, query,
		consent.ID, consent.RideID, consent.UserID, consent.UserType,
		consent.Consented, consent.ConsentedAt, consent.IPAddress, consent.UserAgent,
	)

	return err
}

// GetConsent gets consent for a ride
func (r *Repository) GetConsent(ctx context.Context, rideID, userID uuid.UUID) (*RecordingConsent, error) {
	query := `
		SELECT id, ride_id, user_id, user_type, consented, consented_at
		FROM recording_consents
		WHERE ride_id = $1 AND user_id = $2
	`

	consent := &RecordingConsent{}
	err := r.db.QueryRow(ctx, query, rideID, userID).Scan(
		&consent.ID, &consent.RideID, &consent.UserID, &consent.UserType,
		&consent.Consented, &consent.ConsentedAt,
	)

	if err != nil {
		return nil, err
	}

	return consent, nil
}

// CheckAllConsented checks if all parties have consented to recording
func (r *Repository) CheckAllConsented(ctx context.Context, rideID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) = 2 AND bool_and(consented)
		FROM recording_consents
		WHERE ride_id = $1
	`

	var allConsented bool
	err := r.db.QueryRow(ctx, query, rideID).Scan(&allConsented)
	return allConsented, err
}

// ========================================
// ACCESS LOG OPERATIONS
// ========================================

// LogAccess logs access to a recording
func (r *Repository) LogAccess(ctx context.Context, log *RecordingAccessLog) error {
	query := `
		INSERT INTO recording_access_logs (id, recording_id, accessed_by, access_type, reason, ip_address, accessed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		log.ID, log.RecordingID, log.AccessedBy, log.AccessType,
		log.Reason, log.IPAddress, log.AccessedAt,
	)

	return err
}

// GetAccessLogs gets access logs for a recording
func (r *Repository) GetAccessLogs(ctx context.Context, recordingID uuid.UUID) ([]*RecordingAccessLog, error) {
	query := `
		SELECT id, recording_id, accessed_by, access_type, reason, ip_address, accessed_at
		FROM recording_access_logs
		WHERE recording_id = $1
		ORDER BY accessed_at DESC
	`

	rows, err := r.db.Query(ctx, query, recordingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*RecordingAccessLog
	for rows.Next() {
		log := &RecordingAccessLog{}
		err := rows.Scan(
			&log.ID, &log.RecordingID, &log.AccessedBy, &log.AccessType,
			&log.Reason, &log.IPAddress, &log.AccessedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// ========================================
// SETTINGS OPERATIONS
// ========================================

// GetSettings gets recording settings for a user
func (r *Repository) GetSettings(ctx context.Context, userID uuid.UUID) (*RecordingSettings, error) {
	query := `
		SELECT user_id, recording_enabled, default_type, default_quality,
		       auto_record_night_rides, auto_record_sos_rides, notify_on_recording,
		       allow_driver_recording, allow_rider_recording, created_at, updated_at
		FROM recording_settings
		WHERE user_id = $1
	`

	settings := &RecordingSettings{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.UserID, &settings.RecordingEnabled, &settings.DefaultType,
		&settings.DefaultQuality, &settings.AutoRecordNightRides, &settings.AutoRecordSOSRides,
		&settings.NotifyOnRecording, &settings.AllowDriverRecording, &settings.AllowRiderRecording,
		&settings.CreatedAt, &settings.UpdatedAt,
	)

	if err != nil {
		// Return default settings if not found
		return &RecordingSettings{
			UserID:               userID,
			RecordingEnabled:     false,
			DefaultType:          RecordingTypeAudio,
			DefaultQuality:       "medium",
			AutoRecordNightRides: false,
			AutoRecordSOSRides:   true,
			NotifyOnRecording:    true,
			AllowDriverRecording: true,
			AllowRiderRecording:  true,
		}, nil
	}

	return settings, nil
}

// UpsertSettings creates or updates recording settings
func (r *Repository) UpsertSettings(ctx context.Context, settings *RecordingSettings) error {
	query := `
		INSERT INTO recording_settings (
			user_id, recording_enabled, default_type, default_quality,
			auto_record_night_rides, auto_record_sos_rides, notify_on_recording,
			allow_driver_recording, allow_rider_recording, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			recording_enabled = EXCLUDED.recording_enabled,
			default_type = EXCLUDED.default_type,
			default_quality = EXCLUDED.default_quality,
			auto_record_night_rides = EXCLUDED.auto_record_night_rides,
			auto_record_sos_rides = EXCLUDED.auto_record_sos_rides,
			notify_on_recording = EXCLUDED.notify_on_recording,
			allow_driver_recording = EXCLUDED.allow_driver_recording,
			allow_rider_recording = EXCLUDED.allow_rider_recording,
			updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query,
		settings.UserID, settings.RecordingEnabled, settings.DefaultType,
		settings.DefaultQuality, settings.AutoRecordNightRides, settings.AutoRecordSOSRides,
		settings.NotifyOnRecording, settings.AllowDriverRecording, settings.AllowRiderRecording,
	)

	return err
}

// ========================================
// STATISTICS
// ========================================

// GetRecordingStats gets recording statistics
func (r *Repository) GetRecordingStats(ctx context.Context) (*RecordingStatsResponse, error) {
	stats := &RecordingStatsResponse{
		RecordingsByType:   make(map[string]int64),
		RecordingsByStatus: make(map[string]int64),
	}

	// Total recordings
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ride_recordings WHERE status != 'deleted'`).Scan(&stats.TotalRecordings)
	if err != nil {
		return nil, err
	}

	// Total duration
	var totalSeconds *int64
	err = r.db.QueryRow(ctx, `SELECT SUM(duration_seconds) FROM ride_recordings WHERE status = 'completed'`).Scan(&totalSeconds)
	if err == nil && totalSeconds != nil {
		stats.TotalDurationHours = float64(*totalSeconds) / 3600.0
	}

	// Total storage
	var totalBytes *int64
	err = r.db.QueryRow(ctx, `SELECT SUM(file_size) FROM ride_recordings WHERE file_size IS NOT NULL`).Scan(&totalBytes)
	if err == nil && totalBytes != nil {
		stats.TotalStorageGB = float64(*totalBytes) / (1024 * 1024 * 1024)
	}

	// Active recordings
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ride_recordings WHERE status = 'recording'`).Scan(&stats.ActiveRecordings)
	if err != nil {
		return nil, err
	}

	// Pending uploads
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ride_recordings WHERE status IN ('stopped', 'uploading')`).Scan(&stats.PendingUploads)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
