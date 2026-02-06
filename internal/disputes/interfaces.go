package disputes

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for disputes repository operations
type RepositoryInterface interface {
	// Dispute operations
	CreateDispute(ctx context.Context, d *Dispute) error
	GetDisputeByID(ctx context.Context, id uuid.UUID) (*Dispute, error)
	GetDisputeByRideAndUser(ctx context.Context, rideID, userID uuid.UUID) (*Dispute, error)
	GetUserDisputes(ctx context.Context, userID uuid.UUID, status *DisputeStatus, limit, offset int) ([]DisputeSummary, int, error)
	ResolveDispute(ctx context.Context, id uuid.UUID, status DisputeStatus, resType ResolutionType, refundAmount *float64, note string, resolvedBy uuid.UUID) error
	UpdateDisputeStatus(ctx context.Context, id uuid.UUID, status DisputeStatus) error

	// Comment operations
	CreateComment(ctx context.Context, c *DisputeComment) error
	GetCommentsByDispute(ctx context.Context, disputeID uuid.UUID, includeInternal bool) ([]DisputeComment, error)

	// Ride context
	GetRideContext(ctx context.Context, rideID uuid.UUID) (*RideContext, error)

	// Admin operations
	GetAllDisputes(ctx context.Context, status *DisputeStatus, reason *DisputeReason, limit, offset int) ([]DisputeSummary, int, error)
	GetDisputeStats(ctx context.Context, from, to time.Time) (*DisputeStats, error)
}
