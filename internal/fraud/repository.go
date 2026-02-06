package fraud

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles fraud detection data operations
type Repository struct {
	db *pgxpool.Pool
}

// Ensure the concrete repository satisfies the service's requirements.
var _ FraudRepository = (*Repository)(nil)

// NewRepository creates a new fraud repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateFraudAlert creates a new fraud alert
func (r *Repository) CreateFraudAlert(ctx context.Context, alert *FraudAlert) error {
	detailsJSON, err := json.Marshal(alert.Details)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO fraud_alerts (
			id, user_id, alert_type, alert_level, status, description,
			details, risk_score, detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Exec(ctx, query,
		alert.ID,
		alert.UserID,
		alert.AlertType,
		alert.AlertLevel,
		alert.Status,
		alert.Description,
		detailsJSON,
		alert.RiskScore,
		alert.DetectedAt,
	)

	return err
}

// GetFraudAlertByID retrieves a fraud alert by ID
func (r *Repository) GetFraudAlertByID(ctx context.Context, alertID uuid.UUID) (*FraudAlert, error) {
	query := `
		SELECT id, user_id, alert_type, alert_level, status, description,
		       details, risk_score, detected_at, investigated_at, investigated_by,
		       resolved_at, notes, action_taken
		FROM fraud_alerts
		WHERE id = $1
	`

	var alert FraudAlert
	var detailsJSON []byte
	var investigatedAt, resolvedAt sql.NullTime
	var investigatedBy sql.NullString
	var notes, actionTaken sql.NullString

	err := r.db.QueryRow(ctx, query, alertID).Scan(
		&alert.ID,
		&alert.UserID,
		&alert.AlertType,
		&alert.AlertLevel,
		&alert.Status,
		&alert.Description,
		&detailsJSON,
		&alert.RiskScore,
		&alert.DetectedAt,
		&investigatedAt,
		&investigatedBy,
		&resolvedAt,
		&notes,
		&actionTaken,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
		alert.Details = make(map[string]interface{})
	}

	if investigatedAt.Valid {
		alert.InvestigatedAt = &investigatedAt.Time
	}
	if investigatedBy.Valid {
		investigatedByUUID, _ := uuid.Parse(investigatedBy.String)
		alert.InvestigatedBy = &investigatedByUUID
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}
	if notes.Valid {
		alert.Notes = notes.String
	}
	if actionTaken.Valid {
		alert.ActionTaken = actionTaken.String
	}

	return &alert, nil
}

// GetAlertsByUser retrieves all fraud alerts for a user
func (r *Repository) GetAlertsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*FraudAlert, error) {
	query := `
		SELECT id, user_id, alert_type, alert_level, status, description,
		       details, risk_score, detected_at, investigated_at, investigated_by,
		       resolved_at, notes, action_taken
		FROM fraud_alerts
		WHERE user_id = $1
		ORDER BY detected_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]*FraudAlert, 0)
	for rows.Next() {
		var alert FraudAlert
		var detailsJSON []byte
		var investigatedAt, resolvedAt sql.NullTime
		var investigatedBy sql.NullString
		var notes, actionTaken sql.NullString

		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.AlertType,
			&alert.AlertLevel,
			&alert.Status,
			&alert.Description,
			&detailsJSON,
			&alert.RiskScore,
			&alert.DetectedAt,
			&investigatedAt,
			&investigatedBy,
			&resolvedAt,
			&notes,
			&actionTaken,
		)

		if err != nil {
			continue
		}

		if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
			alert.Details = make(map[string]interface{})
		}

		if investigatedAt.Valid {
			alert.InvestigatedAt = &investigatedAt.Time
		}
		if investigatedBy.Valid {
			investigatedByUUID, _ := uuid.Parse(investigatedBy.String)
			alert.InvestigatedBy = &investigatedByUUID
		}
		if resolvedAt.Valid {
			alert.ResolvedAt = &resolvedAt.Time
		}
		if notes.Valid {
			alert.Notes = notes.String
		}
		if actionTaken.Valid {
			alert.ActionTaken = actionTaken.String
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// GetPendingAlerts retrieves all pending fraud alerts
func (r *Repository) GetPendingAlerts(ctx context.Context, limit, offset int) ([]*FraudAlert, error) {
	query := `
		SELECT id, user_id, alert_type, alert_level, status, description,
		       details, risk_score, detected_at, investigated_at, investigated_by,
		       resolved_at, notes, action_taken
		FROM fraud_alerts
		WHERE status IN ('pending', 'investigating')
		ORDER BY alert_level DESC, detected_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]*FraudAlert, 0)
	for rows.Next() {
		var alert FraudAlert
		var detailsJSON []byte
		var investigatedAt, resolvedAt sql.NullTime
		var investigatedBy sql.NullString
		var notes, actionTaken sql.NullString

		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.AlertType,
			&alert.AlertLevel,
			&alert.Status,
			&alert.Description,
			&detailsJSON,
			&alert.RiskScore,
			&alert.DetectedAt,
			&investigatedAt,
			&investigatedBy,
			&resolvedAt,
			&notes,
			&actionTaken,
		)

		if err != nil {
			continue
		}

		if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
			alert.Details = make(map[string]interface{})
		}

		if investigatedAt.Valid {
			alert.InvestigatedAt = &investigatedAt.Time
		}
		if investigatedBy.Valid {
			investigatedByUUID, _ := uuid.Parse(investigatedBy.String)
			alert.InvestigatedBy = &investigatedByUUID
		}
		if resolvedAt.Valid {
			alert.ResolvedAt = &resolvedAt.Time
		}
		if notes.Valid {
			alert.Notes = notes.String
		}
		if actionTaken.Valid {
			alert.ActionTaken = actionTaken.String
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// GetAlertsByUserWithTotal retrieves all fraud alerts for a user with total count
func (r *Repository) GetAlertsByUserWithTotal(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*FraudAlert, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM fraud_alerts WHERE user_id = $1`
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated alerts
	alerts, err := r.GetAlertsByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// GetPendingAlertsWithTotal retrieves all pending fraud alerts with total count
func (r *Repository) GetPendingAlertsWithTotal(ctx context.Context, limit, offset int) ([]*FraudAlert, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM fraud_alerts WHERE status = 'pending'`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated alerts
	alerts, err := r.GetPendingAlerts(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// UpdateAlertStatus updates the status of a fraud alert
func (r *Repository) UpdateAlertStatus(ctx context.Context, alertID uuid.UUID, status FraudAlertStatus, investigatedBy *uuid.UUID, notes, actionTaken string) error {
	query := `
		UPDATE fraud_alerts
		SET status = $2,
		    investigated_at = CASE WHEN $3::uuid IS NOT NULL THEN NOW() ELSE investigated_at END,
		    investigated_by = COALESCE($3, investigated_by),
		    resolved_at = CASE WHEN $2 IN ('confirmed', 'false_positive', 'resolved') THEN NOW() ELSE resolved_at END,
		    notes = COALESCE(NULLIF($4, ''), notes),
		    action_taken = COALESCE(NULLIF($5, ''), action_taken),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, alertID, status, investigatedBy, notes, actionTaken)
	return err
}

// GetUserRiskProfile retrieves or creates a risk profile for a user
func (r *Repository) GetUserRiskProfile(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	query := `
		SELECT user_id, risk_score, total_alerts, critical_alerts,
		       confirmed_fraud_cases, last_alert_at, account_suspended, last_updated
		FROM user_risk_profiles
		WHERE user_id = $1
	`

	var profile UserRiskProfile
	var lastAlertAt sql.NullTime

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.RiskScore,
		&profile.TotalAlerts,
		&profile.CriticalAlerts,
		&profile.ConfirmedFraudCases,
		&lastAlertAt,
		&profile.AccountSuspended,
		&profile.LastUpdated,
	)

	if err != nil {
		// If profile doesn't exist, return a new one
		if err == sql.ErrNoRows || err.Error() == "no rows in result set" {
			return &UserRiskProfile{
				UserID:              userID,
				RiskScore:           0,
				TotalAlerts:         0,
				CriticalAlerts:      0,
				ConfirmedFraudCases: 0,
				AccountSuspended:    false,
				LastUpdated:         time.Now(),
			}, nil
		}
		return nil, err
	}

	if lastAlertAt.Valid {
		profile.LastAlertAt = &lastAlertAt.Time
	}

	return &profile, nil
}

// UpdateUserRiskProfile updates a user's risk profile
func (r *Repository) UpdateUserRiskProfile(ctx context.Context, profile *UserRiskProfile) error {
	query := `
		INSERT INTO user_risk_profiles (
			user_id, risk_score, total_alerts, critical_alerts,
			confirmed_fraud_cases, last_alert_at, account_suspended, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			risk_score = EXCLUDED.risk_score,
			total_alerts = EXCLUDED.total_alerts,
			critical_alerts = EXCLUDED.critical_alerts,
			confirmed_fraud_cases = EXCLUDED.confirmed_fraud_cases,
			last_alert_at = EXCLUDED.last_alert_at,
			account_suspended = EXCLUDED.account_suspended,
			last_updated = EXCLUDED.last_updated
	`

	_, err := r.db.Exec(ctx, query,
		profile.UserID,
		profile.RiskScore,
		profile.TotalAlerts,
		profile.CriticalAlerts,
		profile.ConfirmedFraudCases,
		profile.LastAlertAt,
		profile.AccountSuspended,
		profile.LastUpdated,
	)

	return err
}

// GetPaymentFraudIndicators analyzes payment patterns for a user
func (r *Repository) GetPaymentFraudIndicators(ctx context.Context, userID uuid.UUID) (*PaymentFraudIndicators, error) {
	query := `
		SELECT
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_attempts,
			COUNT(CASE WHEN status = 'refunded' THEN 1 END) as chargebacks,
			COUNT(DISTINCT payment_method_id) as payment_methods,
			COUNT(CASE WHEN amount > 1000 THEN 1 END) as suspicious_transactions
		FROM payments
		WHERE user_id = $1
		  AND created_at >= NOW() - INTERVAL '30 days'
	`

	indicators := &PaymentFraudIndicators{UserID: userID}

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&indicators.FailedPaymentAttempts,
		&indicators.ChargebackCount,
		&indicators.MultiplePaymentMethods,
		&indicators.SuspiciousTransactions,
	)

	if err != nil {
		return nil, err
	}

	// Check for rapid payment method changes (>3 changes in 7 days)
	changeQuery := `
		SELECT COUNT(DISTINCT payment_method_id) as changes
		FROM payments
		WHERE user_id = $1
		  AND created_at >= NOW() - INTERVAL '7 days'
	`
	var changes int
	r.db.QueryRow(ctx, changeQuery, userID).Scan(&changes)
	indicators.RapidPaymentChanges = changes > 3

	// Calculate risk score
	indicators.RiskScore = calculatePaymentRiskScore(indicators)

	return indicators, nil
}

// GetRideFraudIndicators analyzes ride patterns for a user
func (r *Repository) GetRideFraudIndicators(ctx context.Context, userID uuid.UUID) (*RideFraudIndicators, error) {
	query := `
		SELECT
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancellations,
			COUNT(*) as total_rides
		FROM rides
		WHERE (rider_id = $1 OR driver_id = $1)
		  AND requested_at >= NOW() - INTERVAL '30 days'
	`

	indicators := &RideFraudIndicators{UserID: userID}
	var totalRides int

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&indicators.ExcessiveCancellations,
		&totalRides,
	)

	if err != nil {
		return nil, err
	}

	// Check for promo code abuse (>5 promo codes used in 30 days)
	promoQuery := `
		SELECT COUNT(DISTINCT promo_code_id)
		FROM rides
		WHERE rider_id = $1
		  AND promo_code_id IS NOT NULL
		  AND requested_at >= NOW() - INTERVAL '30 days'
	`
	var promoCount int
	r.db.QueryRow(ctx, promoQuery, userID).Scan(&promoCount)
	indicators.PromoAbuse = promoCount > 5

	// Detect unusual patterns
	if totalRides > 0 {
		cancellationRate := float64(indicators.ExcessiveCancellations) / float64(totalRides)
		indicators.UnusualRidePatterns = cancellationRate > 0.5 || totalRides > 100
	}

	// Calculate risk score
	indicators.RiskScore = calculateRideRiskScore(indicators)

	return indicators, nil
}

// Helper functions to calculate risk scores
func calculatePaymentRiskScore(indicators *PaymentFraudIndicators) float64 {
	score := 0.0

	// Failed attempts (0-30 points)
	score += float64(indicators.FailedPaymentAttempts) * 3
	if score > 30 {
		score = 30
	}

	// Chargebacks (0-30 points)
	score += float64(indicators.ChargebackCount) * 15
	if score > 60 {
		score = 60
	}

	// Multiple payment methods (0-20 points)
	if indicators.MultiplePaymentMethods > 5 {
		score += 20
	} else if indicators.MultiplePaymentMethods > 3 {
		score += 10
	}

	// Suspicious transactions (0-20 points)
	score += float64(indicators.SuspiciousTransactions) * 5
	if score > 100 {
		score = 100
	}

	// Rapid changes (10 points)
	if indicators.RapidPaymentChanges {
		score += 10
	}

	if score > 100 {
		score = 100
	}

	return score
}

func calculateRideRiskScore(indicators *RideFraudIndicators) float64 {
	score := 0.0

	// Excessive cancellations (0-40 points)
	score += float64(indicators.ExcessiveCancellations) * 2
	if score > 40 {
		score = 40
	}

	// Unusual patterns (20 points)
	if indicators.UnusualRidePatterns {
		score += 20
	}

	// Fake GPS (30 points)
	if indicators.FakeGPSDetected {
		score += 30
	}

	// Collision (30 points)
	if indicators.CollisionWithDriver {
		score += 30
	}

	// Promo abuse (20 points)
	if indicators.PromoAbuse {
		score += 20
	}

	if score > 100 {
		score = 100
	}

	return score
}

// GetAccountFraudIndicators analyzes account patterns for a user
func (r *Repository) GetAccountFraudIndicators(ctx context.Context, userID uuid.UUID) (*AccountFraudIndicators, error) {
	indicators := &AccountFraudIndicators{UserID: userID}

	// Check for suspicious email patterns (e.g., temp email services)
	emailQuery := `
		SELECT email FROM users WHERE id = $1
	`
	var email string
	r.db.QueryRow(ctx, emailQuery, userID).Scan(&email)

	// Simple check for known temp email patterns
	tempEmailPatterns := []string{"tempmail", "throwaway", "guerrillamail", "10minutemail", "mailinator"}
	for _, pattern := range tempEmailPatterns {
		if len(email) > 0 && contains(email, pattern) {
			indicators.SuspiciousEmailPattern = true
			break
		}
	}

	// Check for rapid account creation (multiple accounts from same IP in short time)
	// This would require IP tracking - placeholder for now
	indicators.RapidAccountCreation = false

	// Check for multiple accounts on same device (would require device fingerprinting)
	indicators.MultipleAccountsSameDevice = false

	// Check for VPN usage (would require IP analysis)
	indicators.VPNUsage = false

	// Check for fake phone number patterns
	phoneQuery := `
		SELECT phone_number FROM users WHERE id = $1
	`
	var phone string
	r.db.QueryRow(ctx, phoneQuery, userID).Scan(&phone)

	// Simple validation - check for obviously fake patterns like all same digits
	if len(phone) > 0 && isRepeatingPattern(phone) {
		indicators.FakePhoneNumber = true
	}

	// Calculate risk score
	indicators.RiskScore = calculateAccountRiskScore(indicators)

	return indicators, nil
}

// GetFraudStatistics retrieves fraud detection statistics for a time period
func (r *Repository) GetFraudStatistics(ctx context.Context, startDate, endDate time.Time) (*FraudStatistics, error) {
	stats := &FraudStatistics{
		Period: fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
	}

	query := `
		SELECT
			COUNT(*) as total_alerts,
			COUNT(CASE WHEN alert_level = 'critical' THEN 1 END) as critical_alerts,
			COUNT(CASE WHEN alert_level = 'high' THEN 1 END) as high_alerts,
			COUNT(CASE WHEN alert_level = 'medium' THEN 1 END) as medium_alerts,
			COUNT(CASE WHEN alert_level = 'low' THEN 1 END) as low_alerts,
			COUNT(CASE WHEN status = 'confirmed' THEN 1 END) as confirmed_fraud,
			COUNT(CASE WHEN status = 'false_positive' THEN 1 END) as false_positives,
			COUNT(CASE WHEN status IN ('pending', 'investigating') THEN 1 END) as pending_investigation,
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(resolved_at, NOW()) - detected_at)) / 60), 0) as avg_response_time
		FROM fraud_alerts
		WHERE detected_at >= $1 AND detected_at <= $2
	`

	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&stats.TotalAlerts,
		&stats.CriticalAlerts,
		&stats.HighAlerts,
		&stats.MediumAlerts,
		&stats.LowAlerts,
		&stats.ConfirmedFraudCases,
		&stats.FalsePositives,
		&stats.PendingInvestigation,
		&stats.AverageResponseTime,
	)

	if err != nil {
		return nil, err
	}

	// Calculate estimated loss prevented (placeholder calculation)
	// Assumes average fraudulent transaction of $50
	stats.EstimatedLossPrevented = float64(stats.ConfirmedFraudCases) * 50.0

	return stats, nil
}

// GetFraudPatterns retrieves detected fraud patterns
func (r *Repository) GetFraudPatterns(ctx context.Context, limit int) ([]*FraudPattern, error) {
	query := `
		SELECT id, pattern_type, description, occurrences, affected_users,
		       first_detected, last_detected, details, severity
		FROM fraud_patterns
		WHERE is_active = true
		ORDER BY severity DESC, last_detected DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patterns := make([]*FraudPattern, 0)
	for rows.Next() {
		var pattern FraudPattern
		var detailsJSON []byte
		var affectedUsersArray []string

		err := rows.Scan(
			&pattern.PatternType,
			&pattern.Description,
			&pattern.Occurrences,
			&affectedUsersArray,
			&pattern.FirstDetected,
			&pattern.LastDetected,
			&detailsJSON,
			&pattern.Severity,
		)

		if err != nil {
			continue
		}

		// Parse affected users
		for _, userStr := range affectedUsersArray {
			if userID, err := uuid.Parse(userStr); err == nil {
				pattern.AffectedUsers = append(pattern.AffectedUsers, userID)
			}
		}

		// Parse details JSON
		if err := json.Unmarshal(detailsJSON, &pattern.Details); err != nil {
			pattern.Details = make(map[string]interface{})
		}

		patterns = append(patterns, &pattern)
	}

	return patterns, nil
}

// CreateFraudPattern creates a new fraud pattern
func (r *Repository) CreateFraudPattern(ctx context.Context, pattern *FraudPattern) error {
	detailsJSON, err := json.Marshal(pattern.Details)
	if err != nil {
		return err
	}

	// Convert UUID slice to string array
	affectedUsersStr := make([]string, len(pattern.AffectedUsers))
	for i, userID := range pattern.AffectedUsers {
		affectedUsersStr[i] = userID.String()
	}

	query := `
		INSERT INTO fraud_patterns (
			pattern_type, description, occurrences, affected_users,
			first_detected, last_detected, details, severity
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(ctx, query,
		pattern.PatternType,
		pattern.Description,
		pattern.Occurrences,
		affectedUsersStr,
		pattern.FirstDetected,
		pattern.LastDetected,
		detailsJSON,
		pattern.Severity,
	)

	return err
}

// UpdateFraudPattern updates an existing fraud pattern
func (r *Repository) UpdateFraudPattern(ctx context.Context, pattern *FraudPattern) error {
	detailsJSON, err := json.Marshal(pattern.Details)
	if err != nil {
		return err
	}

	// Convert UUID slice to string array
	affectedUsersStr := make([]string, len(pattern.AffectedUsers))
	for i, userID := range pattern.AffectedUsers {
		affectedUsersStr[i] = userID.String()
	}

	query := `
		UPDATE fraud_patterns
		SET occurrences = $2,
		    affected_users = $3,
		    last_detected = $4,
		    details = $5
		WHERE pattern_type = $1
	`

	_, err = r.db.Exec(ctx, query,
		pattern.PatternType,
		pattern.Occurrences,
		affectedUsersStr,
		pattern.LastDetected,
		detailsJSON,
	)

	return err
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isRepeatingPattern(s string) bool {
	if len(s) < 3 {
		return false
	}

	// Check if all characters are the same
	first := s[0]
	allSame := true
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			allSame = false
			break
		}
	}

	return allSame
}

func calculateAccountRiskScore(indicators *AccountFraudIndicators) float64 {
	score := 0.0

	// Multiple accounts same device (40 points)
	if indicators.MultipleAccountsSameDevice {
		score += 40
	}

	// Rapid account creation (30 points)
	if indicators.RapidAccountCreation {
		score += 30
	}

	// Suspicious email (20 points)
	if indicators.SuspiciousEmailPattern {
		score += 20
	}

	// Fake phone number (30 points)
	if indicators.FakePhoneNumber {
		score += 30
	}

	// VPN usage (10 points - not necessarily fraud)
	if indicators.VPNUsage {
		score += 10
	}

	if score > 100 {
		score = 100
	}

	return score
}
