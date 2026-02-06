package experiments

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for experiments repository operations
type RepositoryInterface interface {
	// Feature Flags
	CreateFlag(ctx context.Context, flag *FeatureFlag) error
	GetFlagByKey(ctx context.Context, key string) (*FeatureFlag, error)
	GetFlagByID(ctx context.Context, id uuid.UUID) (*FeatureFlag, error)
	ListFlags(ctx context.Context, status *FlagStatus, limit, offset int) ([]*FeatureFlag, error)
	UpdateFlag(ctx context.Context, flag *FeatureFlag) error
	UpdateFlagStatus(ctx context.Context, id uuid.UUID, status FlagStatus) error
	GetAllActiveFlags(ctx context.Context) ([]*FeatureFlag, error)

	// Flag Overrides
	CreateOverride(ctx context.Context, override *FlagOverride) error
	GetOverride(ctx context.Context, flagID, userID uuid.UUID) (*FlagOverride, error)
	ListOverrides(ctx context.Context, flagID uuid.UUID) ([]*FlagOverride, error)
	DeleteOverride(ctx context.Context, flagID, userID uuid.UUID) error

	// Experiments
	CreateExperiment(ctx context.Context, experiment *Experiment, variants []*Variant) error
	GetExperimentByKey(ctx context.Context, key string) (*Experiment, error)
	GetExperimentByID(ctx context.Context, id uuid.UUID) (*Experiment, error)
	ListExperiments(ctx context.Context, status *ExperimentStatus, limit, offset int) ([]*Experiment, error)
	UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status ExperimentStatus) error
	GetVariants(ctx context.Context, experimentID uuid.UUID) ([]*Variant, error)

	// Assignments
	GetAssignment(ctx context.Context, experimentID, userID uuid.UUID) (*ExperimentAssignment, error)
	CreateAssignment(ctx context.Context, assignment *ExperimentAssignment) error
	GetAssignmentCount(ctx context.Context, experimentID uuid.UUID) (map[uuid.UUID]int, error)

	// Events
	RecordEvent(ctx context.Context, event *ExperimentEvent) error
	GetVariantMetrics(ctx context.Context, experimentID uuid.UUID) ([]VariantMetrics, error)
	GetActiveExperimentsForUser(ctx context.Context) ([]*Experiment, error)
}
