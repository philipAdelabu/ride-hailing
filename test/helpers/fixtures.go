package helpers

import (
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateTestUser creates a test user with default values
func CreateTestUser() *models.User {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	return &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PhoneNumber:  "+1234567890",
		PasswordHash: string(hashedPassword),
		FirstName:    "John",
		LastName:     "Doe",
		Role:         "rider",
		IsActive:     true,
		IsVerified:   false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// CreateTestDriver creates a test driver with default values
func CreateTestDriver(userID uuid.UUID) *models.Driver {
	return &models.Driver{
		ID:            uuid.New(),
		UserID:        userID,
		LicenseNumber: "DL123456789",
		VehicleModel:  "Toyota Camry",
		VehiclePlate:  "ABC-1234",
		VehicleColor:  "Silver",
		VehicleYear:   2020,
		IsAvailable:   false,
		IsOnline:      false,
		Rating:        0.0,
		TotalRides:    0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// CreateTestRegisterRequest creates a test registration request
func CreateTestRegisterRequest() *models.RegisterRequest {
	return &models.RegisterRequest{
		Email:       "newuser@example.com",
		Password:    "SecurePassword123!",
		PhoneNumber: "+1234567890",
		FirstName:   "Jane",
		LastName:    "Smith",
		Role:        "rider",
	}
}

// CreateTestLoginRequest creates a test login request
func CreateTestLoginRequest() *models.LoginRequest {
	return &models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
}

// CreateTestRide creates a test ride with default values
func CreateTestRide(riderID, driverID uuid.UUID) *models.Ride {
	return &models.Ride{
		ID:                uuid.New(),
		RiderID:           riderID,
		DriverID:          &driverID,
		PickupLatitude:    37.7749,
		PickupLongitude:   -122.4194,
		DropoffLatitude:   37.8044,
		DropoffLongitude:  -122.2712,
		Status:            "pending",
		EstimatedFare:     25.50,
		EstimatedDistance: 10.5,
		EstimatedDuration: 20,
		SurgeMultiplier:   1.0,
		RequestedAt:       time.Now(),
	}
}

// CreateTestPayment creates a test payment with default values
func CreateTestPayment(rideID, riderID, driverID uuid.UUID, amount float64) *models.Payment {
	return &models.Payment{
		ID:             uuid.New(),
		RideID:         rideID,
		RiderID:        riderID,
		DriverID:       driverID,
		Amount:         amount,
		Currency:       "USD",
		Status:         "pending",
		PaymentMethod:  "credit_card",
		Commission:     amount * 0.20,
		DriverEarnings: amount * 0.80,
	}
}
