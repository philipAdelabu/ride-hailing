package helpers

import (
	"testing"

	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
)

// AssertUserEqual asserts that two users are equal (excluding sensitive fields)
func AssertUserEqual(t *testing.T, expected, actual *models.User) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Email, actual.Email)
	assert.Equal(t, expected.PhoneNumber, actual.PhoneNumber)
	assert.Equal(t, expected.FirstName, actual.FirstName)
	assert.Equal(t, expected.LastName, actual.LastName)
	assert.Equal(t, expected.Role, actual.Role)
	assert.Equal(t, expected.IsActive, actual.IsActive)
	assert.Equal(t, expected.IsVerified, actual.IsVerified)
}

// AssertDriverEqual asserts that two drivers are equal
func AssertDriverEqual(t *testing.T, expected, actual *models.Driver) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.UserID, actual.UserID)
	assert.Equal(t, expected.LicenseNumber, actual.LicenseNumber)
	assert.Equal(t, expected.VehicleModel, actual.VehicleModel)
	assert.Equal(t, expected.VehiclePlate, actual.VehiclePlate)
	assert.Equal(t, expected.VehicleColor, actual.VehicleColor)
	assert.Equal(t, expected.VehicleYear, actual.VehicleYear)
}

// AssertValidJWT asserts that a string is a valid JWT token format
func AssertValidJWT(t *testing.T, token string) {
	assert.NotEmpty(t, token)
	// JWT tokens should have 3 parts separated by dots
	assert.Contains(t, token, ".")
}

// AssertPasswordNotInResponse asserts that password hash is not in the response
func AssertPasswordNotInResponse(t *testing.T, user *models.User) {
	assert.Empty(t, user.PasswordHash, "Password hash should not be in response")
}
