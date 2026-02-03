package experiments

import (
	"time"

	"github.com/google/uuid"
)

// ========================================
// FEATURE FLAGS
// ========================================

// FlagStatus represents the status of a feature flag
type FlagStatus string

const (
	FlagStatusActive   FlagStatus = "active"
	FlagStatusInactive FlagStatus = "inactive"
	FlagStatusArchived FlagStatus = "archived"
)

// FlagType represents what kind of flag this is
type FlagType string

const (
	FlagTypeBoolean    FlagType = "boolean"    // Simple on/off
	FlagTypePercentage FlagType = "percentage" // Gradual rollout percentage
	FlagTypeUserList   FlagType = "user_list"  // Specific user IDs
	FlagTypeSegment    FlagType = "segment"    // User segment based
)

// FeatureFlag represents a feature toggle
type FeatureFlag struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Key         string     `json:"key" db:"key"` // e.g., "enable_pool_rides", "new_pricing_ui"
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	FlagType    FlagType   `json:"flag_type" db:"flag_type"`
	Status      FlagStatus `json:"status" db:"status"`

	// Boolean flags
	Enabled bool `json:"enabled" db:"enabled"`

	// Percentage rollout
	RolloutPercentage int `json:"rollout_percentage" db:"rollout_percentage"` // 0-100

	// User targeting
	AllowedUserIDs []uuid.UUID `json:"allowed_user_ids,omitempty"` // Specific users
	BlockedUserIDs []uuid.UUID `json:"blocked_user_ids,omitempty"` // Excluded users

	// Segment targeting
	SegmentRules *SegmentRules `json:"segment_rules,omitempty"`

	// Metadata
	Tags      []string  `json:"tags,omitempty"`
	CreatedBy uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// SegmentRules defines user segment targeting rules
type SegmentRules struct {
	// User properties
	Roles       []string `json:"roles,omitempty"`       // e.g., ["rider", "driver"]
	Countries   []string `json:"countries,omitempty"`    // ISO country codes
	Cities      []string `json:"cities,omitempty"`       // City names
	MinRides    *int     `json:"min_rides,omitempty"`    // Minimum completed rides
	MaxRides    *int     `json:"max_rides,omitempty"`    // Maximum completed rides
	MinRating   *float64 `json:"min_rating,omitempty"`   // Minimum user rating
	AccountAge  *int     `json:"account_age_days,omitempty"` // Days since registration
	Platform    []string `json:"platform,omitempty"`     // ["ios", "android", "web"]
	AppVersion  *string  `json:"min_app_version,omitempty"` // Minimum app version
	LoyaltyTier []string `json:"loyalty_tier,omitempty"` // ["gold", "platinum"]
}

// FlagOverride represents a per-user override for a feature flag
type FlagOverride struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FlagID    uuid.UUID `json:"flag_id" db:"flag_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	Reason    string    `json:"reason" db:"reason"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedBy uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ========================================
// A/B EXPERIMENTS
// ========================================

// ExperimentStatus represents the status of an experiment
type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusArchived  ExperimentStatus = "archived"
)

// Experiment represents an A/B test
type Experiment struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	Key         string           `json:"key" db:"key"` // e.g., "pricing_page_v2"
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description" db:"description"`
	Hypothesis  string           `json:"hypothesis" db:"hypothesis"`
	Status      ExperimentStatus `json:"status" db:"status"`

	// Traffic allocation
	TrafficPercentage int `json:"traffic_percentage" db:"traffic_percentage"` // % of eligible users in experiment

	// Targeting
	SegmentRules *SegmentRules `json:"segment_rules,omitempty"`

	// Metrics
	PrimaryMetric   string   `json:"primary_metric" db:"primary_metric"` // e.g., "conversion_rate"
	SecondaryMetrics []string `json:"secondary_metrics,omitempty"`

	// Configuration
	MinSampleSize int `json:"min_sample_size" db:"min_sample_size"` // Minimum users per variant
	ConfidenceLevel float64 `json:"confidence_level" db:"confidence_level"` // Default 0.95

	// Timeline
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	CreatedBy   uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Variant represents a variant in an A/B experiment
type Variant struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	ExperimentID  uuid.UUID              `json:"experiment_id" db:"experiment_id"`
	Key           string                 `json:"key" db:"key"` // e.g., "control", "variant_a"
	Name          string                 `json:"name" db:"name"`
	Description   string                 `json:"description" db:"description"`
	IsControl     bool                   `json:"is_control" db:"is_control"`
	Weight        int                    `json:"weight" db:"weight"` // Traffic weight (e.g., 50 for 50%)
	Config        map[string]interface{} `json:"config,omitempty"`   // Variant-specific configuration
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// ExperimentAssignment records which variant a user was assigned to
type ExperimentAssignment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ExperimentID uuid.UUID `json:"experiment_id" db:"experiment_id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	VariantID    uuid.UUID `json:"variant_id" db:"variant_id"`
	AssignedAt   time.Time `json:"assigned_at" db:"assigned_at"`
}

// ExperimentEvent records an event for experiment analysis
type ExperimentEvent struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	ExperimentID uuid.UUID              `json:"experiment_id" db:"experiment_id"`
	UserID       uuid.UUID              `json:"user_id" db:"user_id"`
	VariantID    uuid.UUID              `json:"variant_id" db:"variant_id"`
	EventType    string                 `json:"event_type" db:"event_type"` // e.g., "impression", "click", "conversion"
	EventValue   *float64               `json:"event_value,omitempty" db:"event_value"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// VariantMetrics contains calculated metrics for a variant
type VariantMetrics struct {
	VariantID       uuid.UUID `json:"variant_id"`
	VariantKey      string    `json:"variant_key"`
	SampleSize      int       `json:"sample_size"`
	Impressions     int       `json:"impressions"`
	Conversions     int       `json:"conversions"`
	ConversionRate  float64   `json:"conversion_rate"`
	AvgEventValue   float64   `json:"avg_event_value"`
	TotalEventValue float64   `json:"total_event_value"`
}

// ExperimentResults contains the results of an experiment
type ExperimentResults struct {
	Experiment      *Experiment      `json:"experiment"`
	Variants        []VariantMetrics `json:"variants"`
	Winner          *string          `json:"winner,omitempty"` // Variant key if there's a winner
	IsSignificant   bool             `json:"is_significant"`
	PValue          *float64         `json:"p_value,omitempty"`
	Uplift          *float64         `json:"uplift,omitempty"` // % improvement of best variant vs control
	CanConclude     bool             `json:"can_conclude"`     // Enough sample size?
	RecommendedAction string         `json:"recommended_action"` // "continue", "conclude_winner", "conclude_no_winner"
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateFlagRequest creates a new feature flag
type CreateFlagRequest struct {
	Key               string        `json:"key" binding:"required"`
	Name              string        `json:"name" binding:"required"`
	Description       string        `json:"description"`
	FlagType          FlagType      `json:"flag_type" binding:"required"`
	Enabled           bool          `json:"enabled"`
	RolloutPercentage int           `json:"rollout_percentage"`
	SegmentRules      *SegmentRules `json:"segment_rules,omitempty"`
	Tags              []string      `json:"tags,omitempty"`
}

// UpdateFlagRequest updates a feature flag
type UpdateFlagRequest struct {
	Name              *string       `json:"name,omitempty"`
	Description       *string       `json:"description,omitempty"`
	Enabled           *bool         `json:"enabled,omitempty"`
	RolloutPercentage *int          `json:"rollout_percentage,omitempty"`
	SegmentRules      *SegmentRules `json:"segment_rules,omitempty"`
	Tags              []string      `json:"tags,omitempty"`
}

// CreateOverrideRequest creates a flag override for a user
type CreateOverrideRequest struct {
	UserID    uuid.UUID  `json:"user_id" binding:"required"`
	Enabled   bool       `json:"enabled"`
	Reason    string     `json:"reason" binding:"required"`
	ExpiresAt *string    `json:"expires_at,omitempty"` // ISO8601
}

// CreateExperimentRequest creates a new experiment
type CreateExperimentRequest struct {
	Key               string        `json:"key" binding:"required"`
	Name              string        `json:"name" binding:"required"`
	Description       string        `json:"description"`
	Hypothesis        string        `json:"hypothesis" binding:"required"`
	TrafficPercentage int           `json:"traffic_percentage" binding:"required"`
	PrimaryMetric     string        `json:"primary_metric" binding:"required"`
	SecondaryMetrics  []string      `json:"secondary_metrics,omitempty"`
	MinSampleSize     int           `json:"min_sample_size"`
	ConfidenceLevel   float64       `json:"confidence_level"`
	SegmentRules      *SegmentRules `json:"segment_rules,omitempty"`
	Variants          []CreateVariantInput `json:"variants" binding:"required,min=2"`
}

// CreateVariantInput defines a variant when creating an experiment
type CreateVariantInput struct {
	Key         string                 `json:"key" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	IsControl   bool                   `json:"is_control"`
	Weight      int                    `json:"weight" binding:"required"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// TrackEventRequest records an experiment event
type TrackEventRequest struct {
	ExperimentKey string                 `json:"experiment_key" binding:"required"`
	EventType     string                 `json:"event_type" binding:"required"`
	EventValue    *float64               `json:"event_value,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EvaluateFlagResponse returns the flag evaluation result for a user
type EvaluateFlagResponse struct {
	Key     string      `json:"key"`
	Enabled bool        `json:"enabled"`
	Source  string      `json:"source"` // "override", "segment", "percentage", "default"
	Variant *string     `json:"variant,omitempty"` // For experiment-linked flags
}

// EvaluateFlagsResponse returns multiple flag evaluations
type EvaluateFlagsResponse struct {
	Flags map[string]EvaluateFlagResponse `json:"flags"`
}

// UserContext provides context for flag evaluation
type UserContext struct {
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Platform    string    `json:"platform"`
	AppVersion  string    `json:"app_version"`
	TotalRides  int       `json:"total_rides"`
	Rating      float64   `json:"rating"`
	AccountAge  int       `json:"account_age_days"`
	LoyaltyTier string   `json:"loyalty_tier"`
}
