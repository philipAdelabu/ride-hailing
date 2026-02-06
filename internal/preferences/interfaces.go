package preferences

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for preferences repository operations
type RepositoryInterface interface {
	// Rider Preferences
	GetRiderPreferences(ctx context.Context, userID uuid.UUID) (*RiderPreferences, error)
	UpsertRiderPreferences(ctx context.Context, p *RiderPreferences) error

	// Ride Overrides
	SetRideOverride(ctx context.Context, o *RidePreferenceOverride) error
	GetRideOverride(ctx context.Context, rideID uuid.UUID) (*RidePreferenceOverride, error)

	// Driver Capabilities
	GetDriverCapabilities(ctx context.Context, driverID uuid.UUID) (*DriverCapabilities, error)
	UpsertDriverCapabilities(ctx context.Context, dc *DriverCapabilities) error
}
