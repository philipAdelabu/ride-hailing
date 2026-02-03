package experiments

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles experiment and feature flag data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new experiments repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// FEATURE FLAGS
// ========================================

// CreateFlag creates a new feature flag
func (r *Repository) CreateFlag(ctx context.Context, flag *FeatureFlag) error {
	allowedJSON, _ := json.Marshal(flag.AllowedUserIDs)
	blockedJSON, _ := json.Marshal(flag.BlockedUserIDs)
	segmentJSON, _ := json.Marshal(flag.SegmentRules)
	tagsJSON, _ := json.Marshal(flag.Tags)

	query := `
		INSERT INTO feature_flags (
			id, key, name, description, flag_type, status,
			enabled, rollout_percentage, allowed_user_ids, blocked_user_ids,
			segment_rules, tags, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.Exec(ctx, query,
		flag.ID, flag.Key, flag.Name, flag.Description, flag.FlagType,
		flag.Status, flag.Enabled, flag.RolloutPercentage,
		allowedJSON, blockedJSON, segmentJSON, tagsJSON,
		flag.CreatedBy, flag.CreatedAt, flag.UpdatedAt,
	)
	return err
}

// GetFlagByKey retrieves a feature flag by its key
func (r *Repository) GetFlagByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	query := `
		SELECT id, key, name, description, flag_type, status,
			   enabled, rollout_percentage, allowed_user_ids, blocked_user_ids,
			   segment_rules, tags, created_by, created_at, updated_at
		FROM feature_flags
		WHERE key = $1
	`

	flag := &FeatureFlag{}
	var allowedJSON, blockedJSON, segmentJSON, tagsJSON []byte
	err := r.db.QueryRow(ctx, query, key).Scan(
		&flag.ID, &flag.Key, &flag.Name, &flag.Description, &flag.FlagType,
		&flag.Status, &flag.Enabled, &flag.RolloutPercentage,
		&allowedJSON, &blockedJSON, &segmentJSON, &tagsJSON,
		&flag.CreatedBy, &flag.CreatedAt, &flag.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(allowedJSON, &flag.AllowedUserIDs)
	json.Unmarshal(blockedJSON, &flag.BlockedUserIDs)
	json.Unmarshal(segmentJSON, &flag.SegmentRules)
	json.Unmarshal(tagsJSON, &flag.Tags)

	return flag, nil
}

// GetFlagByID retrieves a feature flag by ID
func (r *Repository) GetFlagByID(ctx context.Context, id uuid.UUID) (*FeatureFlag, error) {
	query := `
		SELECT id, key, name, description, flag_type, status,
			   enabled, rollout_percentage, allowed_user_ids, blocked_user_ids,
			   segment_rules, tags, created_by, created_at, updated_at
		FROM feature_flags
		WHERE id = $1
	`

	flag := &FeatureFlag{}
	var allowedJSON, blockedJSON, segmentJSON, tagsJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&flag.ID, &flag.Key, &flag.Name, &flag.Description, &flag.FlagType,
		&flag.Status, &flag.Enabled, &flag.RolloutPercentage,
		&allowedJSON, &blockedJSON, &segmentJSON, &tagsJSON,
		&flag.CreatedBy, &flag.CreatedAt, &flag.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(allowedJSON, &flag.AllowedUserIDs)
	json.Unmarshal(blockedJSON, &flag.BlockedUserIDs)
	json.Unmarshal(segmentJSON, &flag.SegmentRules)
	json.Unmarshal(tagsJSON, &flag.Tags)

	return flag, nil
}

// ListFlags lists all feature flags
func (r *Repository) ListFlags(ctx context.Context, status *FlagStatus, limit, offset int) ([]*FeatureFlag, error) {
	query := `
		SELECT id, key, name, description, flag_type, status,
			   enabled, rollout_percentage, allowed_user_ids, blocked_user_ids,
			   segment_rules, tags, created_by, created_at, updated_at
		FROM feature_flags
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	var statusStr *string
	if status != nil {
		s := string(*status)
		statusStr = &s
	}

	rows, err := r.db.Query(ctx, query, statusStr, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFlags(rows)
}

// UpdateFlag updates a feature flag
func (r *Repository) UpdateFlag(ctx context.Context, flag *FeatureFlag) error {
	segmentJSON, _ := json.Marshal(flag.SegmentRules)
	tagsJSON, _ := json.Marshal(flag.Tags)

	query := `
		UPDATE feature_flags
		SET name = $2, description = $3, enabled = $4,
			rollout_percentage = $5, segment_rules = $6, tags = $7,
			updated_at = $8
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		flag.ID, flag.Name, flag.Description, flag.Enabled,
		flag.RolloutPercentage, segmentJSON, tagsJSON,
		flag.UpdatedAt,
	)
	return err
}

// UpdateFlagStatus updates the status of a flag
func (r *Repository) UpdateFlagStatus(ctx context.Context, id uuid.UUID, status FlagStatus) error {
	query := `UPDATE feature_flags SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, status)
	return err
}

// GetAllActiveFlags retrieves all active flags for caching
func (r *Repository) GetAllActiveFlags(ctx context.Context) ([]*FeatureFlag, error) {
	query := `
		SELECT id, key, name, description, flag_type, status,
			   enabled, rollout_percentage, allowed_user_ids, blocked_user_ids,
			   segment_rules, tags, created_by, created_at, updated_at
		FROM feature_flags
		WHERE status = 'active'
		ORDER BY key ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFlags(rows)
}

func (r *Repository) scanFlags(rows pgx.Rows) ([]*FeatureFlag, error) {
	var flags []*FeatureFlag
	for rows.Next() {
		flag := &FeatureFlag{}
		var allowedJSON, blockedJSON, segmentJSON, tagsJSON []byte
		err := rows.Scan(
			&flag.ID, &flag.Key, &flag.Name, &flag.Description, &flag.FlagType,
			&flag.Status, &flag.Enabled, &flag.RolloutPercentage,
			&allowedJSON, &blockedJSON, &segmentJSON, &tagsJSON,
			&flag.CreatedBy, &flag.CreatedAt, &flag.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(allowedJSON, &flag.AllowedUserIDs)
		json.Unmarshal(blockedJSON, &flag.BlockedUserIDs)
		json.Unmarshal(segmentJSON, &flag.SegmentRules)
		json.Unmarshal(tagsJSON, &flag.Tags)
		flags = append(flags, flag)
	}
	return flags, nil
}

// ========================================
// FLAG OVERRIDES
// ========================================

// CreateOverride creates a per-user flag override
func (r *Repository) CreateOverride(ctx context.Context, override *FlagOverride) error {
	query := `
		INSERT INTO flag_overrides (id, flag_id, user_id, enabled, reason, expires_at, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (flag_id, user_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			reason = EXCLUDED.reason,
			expires_at = EXCLUDED.expires_at
	`
	_, err := r.db.Exec(ctx, query,
		override.ID, override.FlagID, override.UserID, override.Enabled,
		override.Reason, override.ExpiresAt, override.CreatedBy, override.CreatedAt,
	)
	return err
}

// GetOverride retrieves a flag override for a user
func (r *Repository) GetOverride(ctx context.Context, flagID, userID uuid.UUID) (*FlagOverride, error) {
	query := `
		SELECT id, flag_id, user_id, enabled, reason, expires_at, created_by, created_at
		FROM flag_overrides
		WHERE flag_id = $1 AND user_id = $2
		  AND (expires_at IS NULL OR expires_at > NOW())
	`

	override := &FlagOverride{}
	err := r.db.QueryRow(ctx, query, flagID, userID).Scan(
		&override.ID, &override.FlagID, &override.UserID, &override.Enabled,
		&override.Reason, &override.ExpiresAt, &override.CreatedBy, &override.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return override, nil
}

// ListOverrides lists overrides for a flag
func (r *Repository) ListOverrides(ctx context.Context, flagID uuid.UUID) ([]*FlagOverride, error) {
	query := `
		SELECT id, flag_id, user_id, enabled, reason, expires_at, created_by, created_at
		FROM flag_overrides
		WHERE flag_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, flagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []*FlagOverride
	for rows.Next() {
		o := &FlagOverride{}
		if err := rows.Scan(&o.ID, &o.FlagID, &o.UserID, &o.Enabled, &o.Reason, &o.ExpiresAt, &o.CreatedBy, &o.CreatedAt); err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, nil
}

// DeleteOverride removes a flag override
func (r *Repository) DeleteOverride(ctx context.Context, flagID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM flag_overrides WHERE flag_id = $1 AND user_id = $2", flagID, userID)
	return err
}

// ========================================
// EXPERIMENTS
// ========================================

// CreateExperiment creates a new experiment with its variants
func (r *Repository) CreateExperiment(ctx context.Context, experiment *Experiment, variants []*Variant) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	segmentJSON, _ := json.Marshal(experiment.SegmentRules)
	metricsJSON, _ := json.Marshal(experiment.SecondaryMetrics)

	expQuery := `
		INSERT INTO experiments (
			id, key, name, description, hypothesis, status,
			traffic_percentage, segment_rules, primary_metric,
			secondary_metrics, min_sample_size, confidence_level,
			started_at, ended_at, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`
	_, err = tx.Exec(ctx, expQuery,
		experiment.ID, experiment.Key, experiment.Name, experiment.Description,
		experiment.Hypothesis, experiment.Status, experiment.TrafficPercentage,
		segmentJSON, experiment.PrimaryMetric, metricsJSON,
		experiment.MinSampleSize, experiment.ConfidenceLevel,
		experiment.StartedAt, experiment.EndedAt, experiment.CreatedBy,
		experiment.CreatedAt, experiment.UpdatedAt,
	)
	if err != nil {
		return err
	}

	variantQuery := `
		INSERT INTO experiment_variants (id, experiment_id, key, name, description, is_control, weight, config, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	for _, v := range variants {
		configJSON, _ := json.Marshal(v.Config)
		_, err = tx.Exec(ctx, variantQuery,
			v.ID, v.ExperimentID, v.Key, v.Name, v.Description,
			v.IsControl, v.Weight, configJSON, v.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetExperimentByKey retrieves an experiment by key
func (r *Repository) GetExperimentByKey(ctx context.Context, key string) (*Experiment, error) {
	query := `
		SELECT id, key, name, description, hypothesis, status,
			   traffic_percentage, segment_rules, primary_metric,
			   secondary_metrics, min_sample_size, confidence_level,
			   started_at, ended_at, created_by, created_at, updated_at
		FROM experiments
		WHERE key = $1
	`

	return r.scanExperiment(ctx, query, key)
}

// GetExperimentByID retrieves an experiment by ID
func (r *Repository) GetExperimentByID(ctx context.Context, id uuid.UUID) (*Experiment, error) {
	query := `
		SELECT id, key, name, description, hypothesis, status,
			   traffic_percentage, segment_rules, primary_metric,
			   secondary_metrics, min_sample_size, confidence_level,
			   started_at, ended_at, created_by, created_at, updated_at
		FROM experiments
		WHERE id = $1
	`

	return r.scanExperiment(ctx, query, id)
}

func (r *Repository) scanExperiment(ctx context.Context, query string, arg interface{}) (*Experiment, error) {
	exp := &Experiment{}
	var segmentJSON, metricsJSON []byte
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&exp.ID, &exp.Key, &exp.Name, &exp.Description, &exp.Hypothesis,
		&exp.Status, &exp.TrafficPercentage, &segmentJSON,
		&exp.PrimaryMetric, &metricsJSON, &exp.MinSampleSize,
		&exp.ConfidenceLevel, &exp.StartedAt, &exp.EndedAt,
		&exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(segmentJSON, &exp.SegmentRules)
	json.Unmarshal(metricsJSON, &exp.SecondaryMetrics)

	return exp, nil
}

// ListExperiments lists experiments
func (r *Repository) ListExperiments(ctx context.Context, status *ExperimentStatus, limit, offset int) ([]*Experiment, error) {
	query := `
		SELECT id, key, name, description, hypothesis, status,
			   traffic_percentage, segment_rules, primary_metric,
			   secondary_metrics, min_sample_size, confidence_level,
			   started_at, ended_at, created_by, created_at, updated_at
		FROM experiments
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	var statusStr *string
	if status != nil {
		s := string(*status)
		statusStr = &s
	}

	rows, err := r.db.Query(ctx, query, statusStr, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []*Experiment
	for rows.Next() {
		exp := &Experiment{}
		var segmentJSON, metricsJSON []byte
		err := rows.Scan(
			&exp.ID, &exp.Key, &exp.Name, &exp.Description, &exp.Hypothesis,
			&exp.Status, &exp.TrafficPercentage, &segmentJSON,
			&exp.PrimaryMetric, &metricsJSON, &exp.MinSampleSize,
			&exp.ConfidenceLevel, &exp.StartedAt, &exp.EndedAt,
			&exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(segmentJSON, &exp.SegmentRules)
		json.Unmarshal(metricsJSON, &exp.SecondaryMetrics)
		experiments = append(experiments, exp)
	}
	return experiments, nil
}

// UpdateExperimentStatus updates experiment status
func (r *Repository) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status ExperimentStatus) error {
	query := `UPDATE experiments SET status = $2, updated_at = NOW() WHERE id = $1`
	if status == ExperimentStatusRunning {
		query = `UPDATE experiments SET status = $2, started_at = NOW(), updated_at = NOW() WHERE id = $1`
	} else if status == ExperimentStatusCompleted {
		query = `UPDATE experiments SET status = $2, ended_at = NOW(), updated_at = NOW() WHERE id = $1`
	}
	_, err := r.db.Exec(ctx, query, id, status)
	return err
}

// GetVariants retrieves variants for an experiment
func (r *Repository) GetVariants(ctx context.Context, experimentID uuid.UUID) ([]*Variant, error) {
	query := `
		SELECT id, experiment_id, key, name, description, is_control, weight, config, created_at
		FROM experiment_variants
		WHERE experiment_id = $1
		ORDER BY is_control DESC, key ASC
	`

	rows, err := r.db.Query(ctx, query, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []*Variant
	for rows.Next() {
		v := &Variant{}
		var configJSON []byte
		if err := rows.Scan(&v.ID, &v.ExperimentID, &v.Key, &v.Name, &v.Description, &v.IsControl, &v.Weight, &configJSON, &v.CreatedAt); err != nil {
			return nil, err
		}
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &v.Config)
		}
		variants = append(variants, v)
	}
	return variants, nil
}

// ========================================
// ASSIGNMENTS
// ========================================

// GetAssignment gets a user's experiment assignment
func (r *Repository) GetAssignment(ctx context.Context, experimentID, userID uuid.UUID) (*ExperimentAssignment, error) {
	query := `
		SELECT id, experiment_id, user_id, variant_id, assigned_at
		FROM experiment_assignments
		WHERE experiment_id = $1 AND user_id = $2
	`

	a := &ExperimentAssignment{}
	err := r.db.QueryRow(ctx, query, experimentID, userID).Scan(
		&a.ID, &a.ExperimentID, &a.UserID, &a.VariantID, &a.AssignedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// CreateAssignment records a user's assignment to a variant
func (r *Repository) CreateAssignment(ctx context.Context, assignment *ExperimentAssignment) error {
	query := `
		INSERT INTO experiment_assignments (id, experiment_id, user_id, variant_id, assigned_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (experiment_id, user_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query,
		assignment.ID, assignment.ExperimentID, assignment.UserID,
		assignment.VariantID, assignment.AssignedAt,
	)
	return err
}

// GetAssignmentCount gets the number of assignments per variant
func (r *Repository) GetAssignmentCount(ctx context.Context, experimentID uuid.UUID) (map[uuid.UUID]int, error) {
	query := `
		SELECT variant_id, COUNT(*) as count
		FROM experiment_assignments
		WHERE experiment_id = $1
		GROUP BY variant_id
	`

	rows, err := r.db.Query(ctx, query, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var variantID uuid.UUID
		var count int
		if err := rows.Scan(&variantID, &count); err != nil {
			return nil, err
		}
		counts[variantID] = count
	}
	return counts, nil
}

// ========================================
// EVENTS
// ========================================

// RecordEvent records an experiment event
func (r *Repository) RecordEvent(ctx context.Context, event *ExperimentEvent) error {
	metadataJSON, _ := json.Marshal(event.Metadata)

	query := `
		INSERT INTO experiment_events (id, experiment_id, user_id, variant_id, event_type, event_value, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		event.ID, event.ExperimentID, event.UserID, event.VariantID,
		event.EventType, event.EventValue, metadataJSON, event.CreatedAt,
	)
	return err
}

// GetVariantMetrics calculates metrics for each variant
func (r *Repository) GetVariantMetrics(ctx context.Context, experimentID uuid.UUID) ([]VariantMetrics, error) {
	query := `
		SELECT
			v.id as variant_id,
			v.key as variant_key,
			COALESCE(counts.sample_size, 0) as sample_size,
			COALESCE(events.impressions, 0) as impressions,
			COALESCE(events.conversions, 0) as conversions,
			CASE WHEN COALESCE(events.impressions, 0) > 0
				THEN COALESCE(events.conversions, 0)::float / events.impressions
				ELSE 0
			END as conversion_rate,
			COALESCE(events.avg_value, 0) as avg_value,
			COALESCE(events.total_value, 0) as total_value
		FROM experiment_variants v
		LEFT JOIN (
			SELECT variant_id, COUNT(*) as sample_size
			FROM experiment_assignments
			WHERE experiment_id = $1
			GROUP BY variant_id
		) counts ON v.id = counts.variant_id
		LEFT JOIN (
			SELECT variant_id,
				   COUNT(*) FILTER (WHERE event_type = 'impression') as impressions,
				   COUNT(*) FILTER (WHERE event_type = 'conversion') as conversions,
				   AVG(event_value) FILTER (WHERE event_value IS NOT NULL) as avg_value,
				   SUM(event_value) FILTER (WHERE event_value IS NOT NULL) as total_value
			FROM experiment_events
			WHERE experiment_id = $1
			GROUP BY variant_id
		) events ON v.id = events.variant_id
		WHERE v.experiment_id = $1
		ORDER BY v.is_control DESC, v.key ASC
	`

	rows, err := r.db.Query(ctx, query, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []VariantMetrics
	for rows.Next() {
		m := VariantMetrics{}
		if err := rows.Scan(
			&m.VariantID, &m.VariantKey, &m.SampleSize,
			&m.Impressions, &m.Conversions, &m.ConversionRate,
			&m.AvgEventValue, &m.TotalEventValue,
		); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// GetActiveExperimentsForUser gets all running experiments a user might be eligible for
func (r *Repository) GetActiveExperimentsForUser(ctx context.Context) ([]*Experiment, error) {
	query := `
		SELECT id, key, name, description, hypothesis, status,
			   traffic_percentage, segment_rules, primary_metric,
			   secondary_metrics, min_sample_size, confidence_level,
			   started_at, ended_at, created_by, created_at, updated_at
		FROM experiments
		WHERE status = 'running'
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiments []*Experiment
	for rows.Next() {
		exp := &Experiment{}
		var segmentJSON, metricsJSON []byte
		err := rows.Scan(
			&exp.ID, &exp.Key, &exp.Name, &exp.Description, &exp.Hypothesis,
			&exp.Status, &exp.TrafficPercentage, &segmentJSON,
			&exp.PrimaryMetric, &metricsJSON, &exp.MinSampleSize,
			&exp.ConfidenceLevel, &exp.StartedAt, &exp.EndedAt,
			&exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(segmentJSON, &exp.SegmentRules)
		json.Unmarshal(metricsJSON, &exp.SecondaryMetrics)
		experiments = append(experiments, exp)
	}
	return experiments, nil
}
