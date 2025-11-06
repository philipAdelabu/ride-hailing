package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// RepositoryInterface defines the interface for auth repository operations
type RepositoryInterface interface {
	CreateUser(ctx context.Context, user *models.User) error
	CreateDriver(ctx context.Context, driver *models.Driver) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
}
