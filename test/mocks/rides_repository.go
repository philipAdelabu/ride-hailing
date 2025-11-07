package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/mock"
)

// MockRidesRepository is a mock implementation of the rides repository
type MockRidesRepository struct {
	mock.Mock
}

// CreateRide mocks creating a ride
func (m *MockRidesRepository) CreateRide(ctx context.Context, ride *models.Ride) error {
	args := m.Called(ctx, ride)
	return args.Error(0)
}

// GetRideByID mocks getting a ride by ID
func (m *MockRidesRepository) GetRideByID(ctx context.Context, id uuid.UUID) (*models.Ride, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

// UpdateRideStatus mocks updating ride status
func (m *MockRidesRepository) UpdateRideStatus(ctx context.Context, id uuid.UUID, status string, driverID *uuid.UUID) error {
	args := m.Called(ctx, id, status, driverID)
	return args.Error(0)
}

// UpdateRideCompletion mocks updating ride completion details
func (m *MockRidesRepository) UpdateRideCompletion(ctx context.Context, id uuid.UUID, actualDistance float64, actualDuration int, finalFare float64) error {
	args := m.Called(ctx, id, actualDistance, actualDuration, finalFare)
	return args.Error(0)
}

// UpdateRideRating mocks updating ride rating
func (m *MockRidesRepository) UpdateRideRating(ctx context.Context, id uuid.UUID, rating int, feedback string) error {
	args := m.Called(ctx, id, rating, feedback)
	return args.Error(0)
}

// GetRidesByRider mocks getting rides by rider
func (m *MockRidesRepository) GetRidesByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, error) {
	args := m.Called(ctx, riderID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ride), args.Error(1)
}

// GetRidesByDriver mocks getting rides by driver
func (m *MockRidesRepository) GetRidesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ride), args.Error(1)
}

// GetPendingRides mocks getting pending rides
func (m *MockRidesRepository) GetPendingRides(ctx context.Context) ([]*models.Ride, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ride), args.Error(1)
}

// GetUserProfile mocks getting user profile
func (m *MockRidesRepository) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UpdateUserProfile mocks updating user profile
func (m *MockRidesRepository) UpdateUserProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, phoneNumber string) error {
	args := m.Called(ctx, userID, firstName, lastName, phoneNumber)
	return args.Error(0)
}
