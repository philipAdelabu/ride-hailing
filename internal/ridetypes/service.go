package ridetypes

import (
	"context"

	"github.com/google/uuid"
)

// Service handles ride type business logic
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new ride types service
func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// CreateRideType creates a new ride type
func (s *Service) CreateRideType(ctx context.Context, rt *RideType) error {
	return s.repo.CreateRideType(ctx, rt)
}

// GetRideTypeByID returns a ride type by ID
func (s *Service) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	return s.repo.GetRideTypeByID(ctx, id)
}

// ListRideTypes returns ride types with pagination
func (s *Service) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	return s.repo.ListRideTypes(ctx, limit, offset, includeInactive)
}

// UpdateRideType updates a ride type
func (s *Service) UpdateRideType(ctx context.Context, rt *RideType) error {
	return s.repo.UpdateRideType(ctx, rt)
}

// DeleteRideType soft-deletes a ride type
func (s *Service) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRideType(ctx, id)
}
