package ridehistory

import (
	"time"

	"github.com/google/uuid"
)

// RideHistoryEntry represents a past ride with full details
type RideHistoryEntry struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	RiderID          uuid.UUID  `json:"rider_id" db:"rider_id"`
	DriverID         *uuid.UUID `json:"driver_id,omitempty" db:"driver_id"`
	Status           string     `json:"status" db:"status"`

	// Route
	PickupAddress    string     `json:"pickup_address" db:"pickup_address"`
	PickupLatitude   float64    `json:"pickup_latitude" db:"pickup_latitude"`
	PickupLongitude  float64    `json:"pickup_longitude" db:"pickup_longitude"`
	DropoffAddress   string     `json:"dropoff_address" db:"dropoff_address"`
	DropoffLatitude  float64    `json:"dropoff_latitude" db:"dropoff_latitude"`
	DropoffLongitude float64    `json:"dropoff_longitude" db:"dropoff_longitude"`
	Distance         float64    `json:"distance_km" db:"distance"`
	Duration         int        `json:"duration_minutes" db:"duration"`

	// Pricing
	EstimatedFare    float64    `json:"estimated_fare" db:"estimated_fare"`
	FinalFare        *float64   `json:"final_fare,omitempty" db:"final_fare"`
	SurgeMultiplier  float64    `json:"surge_multiplier" db:"surge_multiplier"`
	DiscountAmount   float64    `json:"discount_amount" db:"discount_amount"`
	Currency         string     `json:"currency" db:"currency"`

	// Rating
	Rating           *int       `json:"rating,omitempty" db:"rating"`
	Feedback         *string    `json:"feedback,omitempty" db:"feedback"`

	// Cancellation
	CancellationReason *string  `json:"cancellation_reason,omitempty" db:"cancellation_reason"`

	// Timestamps
	RequestedAt      time.Time  `json:"requested_at" db:"requested_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	StartedAt        *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
}

// Receipt represents a detailed ride receipt
type Receipt struct {
	ReceiptID       string           `json:"receipt_id"`
	RideID          uuid.UUID        `json:"ride_id"`
	IssuedAt        time.Time        `json:"issued_at"`
	RiderName       string           `json:"rider_name"`
	RiderEmail      *string          `json:"rider_email,omitempty"`

	// Trip summary
	PickupAddress   string           `json:"pickup_address"`
	DropoffAddress  string           `json:"dropoff_address"`
	Distance        float64          `json:"distance_km"`
	Duration        int              `json:"duration_minutes"`
	TripDate        string           `json:"trip_date"`
	TripStartTime   string           `json:"trip_start_time"`
	TripEndTime     string           `json:"trip_end_time"`

	// Fare breakdown
	FareBreakdown   []FareLineItem   `json:"fare_breakdown"`
	Subtotal        float64          `json:"subtotal"`
	Discounts       float64          `json:"discounts"`
	Fees            float64          `json:"fees"`
	Tip             float64          `json:"tip"`
	Total           float64          `json:"total"`
	Currency        string           `json:"currency"`

	// Payment
	PaymentMethod   string           `json:"payment_method"`
	PaymentLast4    *string          `json:"payment_last4,omitempty"`

	// Driver
	DriverName      *string          `json:"driver_name,omitempty"`
	VehicleInfo     *string          `json:"vehicle_info,omitempty"`
	LicensePlate    *string          `json:"license_plate,omitempty"`

	// Map
	RouteMapURL     *string          `json:"route_map_url,omitempty"`
}

// FareLineItem represents a line in the fare breakdown
type FareLineItem struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"` // charge, discount, fee, tip
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// HistoryFilters filters ride history
type HistoryFilters struct {
	Status   *string    `json:"status,omitempty"`
	FromDate *time.Time `json:"from_date,omitempty"`
	ToDate   *time.Time `json:"to_date,omitempty"`
	MinFare  *float64   `json:"min_fare,omitempty"`
	MaxFare  *float64   `json:"max_fare,omitempty"`
}

// RideStats summarizes a user's ride activity
type RideStats struct {
	TotalRides       int     `json:"total_rides"`
	CompletedRides   int     `json:"completed_rides"`
	CancelledRides   int     `json:"cancelled_rides"`
	TotalSpent       float64 `json:"total_spent"`
	TotalDistance     float64 `json:"total_distance_km"`
	TotalDuration    int     `json:"total_duration_minutes"`
	AverageFare      float64 `json:"average_fare"`
	AverageDistance   float64 `json:"average_distance_km"`
	AverageRating    float64 `json:"average_rating_given"`
	FavoritePickup   *string `json:"favorite_pickup,omitempty"`
	FavoriteDropoff  *string `json:"favorite_dropoff,omitempty"`
	Currency         string  `json:"currency"`
	Period           string  `json:"period"`
}

// FrequentRoute represents a commonly taken route
type FrequentRoute struct {
	PickupAddress  string  `json:"pickup_address"`
	DropoffAddress string  `json:"dropoff_address"`
	RideCount      int     `json:"ride_count"`
	AverageFare    float64 `json:"average_fare"`
	LastRideAt     string  `json:"last_ride_at"`
}
