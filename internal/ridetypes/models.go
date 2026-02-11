package ridetypes

import (
	"time"

	"github.com/google/uuid"
)

// RideType represents a ride type (Economy, Premium, XL, etc.)
type RideType struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Description   *string   `json:"description,omitempty" db:"description"`
	BaseFare      float64   `json:"base_fare" db:"base_fare"`
	PerKmRate     float64   `json:"per_km_rate" db:"per_km_rate"`
	PerMinuteRate float64   `json:"per_minute_rate" db:"per_minute_rate"`
	MinimumFare   float64   `json:"minimum_fare" db:"minimum_fare"`
	Capacity      int       `json:"capacity" db:"capacity"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CreateRideTypeRequest is the request body for creating a ride type
type CreateRideTypeRequest struct {
	Name          string  `json:"name" binding:"required"`
	Description   *string `json:"description,omitempty"`
	BaseFare      float64 `json:"base_fare" binding:"required,gt=0"`
	PerKmRate     float64 `json:"per_km_rate" binding:"required,gt=0"`
	PerMinuteRate float64 `json:"per_minute_rate" binding:"required,gt=0"`
	MinimumFare   float64 `json:"minimum_fare" binding:"required,gt=0"`
	Capacity      int     `json:"capacity" binding:"required,gt=0"`
	IsActive      bool    `json:"is_active"`
}

// UpdateRideTypeRequest is the request body for updating a ride type
type UpdateRideTypeRequest struct {
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	BaseFare      *float64 `json:"base_fare,omitempty"`
	PerKmRate     *float64 `json:"per_km_rate,omitempty"`
	PerMinuteRate *float64 `json:"per_minute_rate,omitempty"`
	MinimumFare   *float64 `json:"minimum_fare,omitempty"`
	Capacity      *int     `json:"capacity,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
}
