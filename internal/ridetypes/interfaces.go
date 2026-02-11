package ridetypes

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for ride type repository operations
type RepositoryInterface interface {
	CreateRideType(ctx context.Context, rt *RideType) error
	GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error)
	ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error)
	UpdateRideType(ctx context.Context, rt *RideType) error
	DeleteRideType(ctx context.Context, id uuid.UUID) error
}
