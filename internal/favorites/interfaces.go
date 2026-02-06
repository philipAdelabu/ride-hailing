package favorites

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for favorites repository operations
type RepositoryInterface interface {
	// CreateFavorite creates a new favorite location
	CreateFavorite(ctx context.Context, favorite *FavoriteLocation) error

	// GetFavoriteByID retrieves a favorite location by ID
	GetFavoriteByID(ctx context.Context, id uuid.UUID) (*FavoriteLocation, error)

	// GetFavoritesByUser retrieves all favorite locations for a user
	GetFavoritesByUser(ctx context.Context, userID uuid.UUID) ([]*FavoriteLocation, error)

	// UpdateFavorite updates a favorite location
	UpdateFavorite(ctx context.Context, favorite *FavoriteLocation) error

	// DeleteFavorite deletes a favorite location
	DeleteFavorite(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
