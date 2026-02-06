package negotiation

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the status of a negotiation session
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusAccepted  SessionStatus = "accepted"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusExpired   SessionStatus = "expired"
	SessionStatusCancelled SessionStatus = "cancelled"
	SessionStatusNoDrivers SessionStatus = "no_drivers"
)

// OfferStatus represents the status of an offer
type OfferStatus string

const (
	OfferStatusPending    OfferStatus = "pending"
	OfferStatusAccepted   OfferStatus = "accepted"
	OfferStatusRejected   OfferStatus = "rejected"
	OfferStatusWithdrawn  OfferStatus = "withdrawn"
	OfferStatusExpired    OfferStatus = "expired"
	OfferStatusSuperseded OfferStatus = "superseded"
)

// Session represents a negotiation session
type Session struct {
	ID                   uuid.UUID     `json:"id" db:"id"`
	RiderID              uuid.UUID     `json:"rider_id" db:"rider_id"`

	// Ride details
	PickupLatitude       float64       `json:"pickup_latitude" db:"pickup_latitude"`
	PickupLongitude      float64       `json:"pickup_longitude" db:"pickup_longitude"`
	PickupAddress        string        `json:"pickup_address" db:"pickup_address"`
	DropoffLatitude      float64       `json:"dropoff_latitude" db:"dropoff_latitude"`
	DropoffLongitude     float64       `json:"dropoff_longitude" db:"dropoff_longitude"`
	DropoffAddress       string        `json:"dropoff_address" db:"dropoff_address"`

	// Location context
	CountryID            *uuid.UUID    `json:"country_id,omitempty" db:"country_id"`
	RegionID             *uuid.UUID    `json:"region_id,omitempty" db:"region_id"`
	CityID               *uuid.UUID    `json:"city_id,omitempty" db:"city_id"`
	PickupZoneID         *uuid.UUID    `json:"pickup_zone_id,omitempty" db:"pickup_zone_id"`
	DropoffZoneID        *uuid.UUID    `json:"dropoff_zone_id,omitempty" db:"dropoff_zone_id"`

	// Pricing context
	RideTypeID           *uuid.UUID    `json:"ride_type_id,omitempty" db:"ride_type_id"`
	CurrencyCode         string        `json:"currency_code" db:"currency_code"`
	EstimatedDistance    float64       `json:"estimated_distance" db:"estimated_distance"`
	EstimatedDuration    int           `json:"estimated_duration" db:"estimated_duration"`
	EstimatedFare        float64       `json:"estimated_fare" db:"estimated_fare"`

	// Fair price bounds
	FairPriceMin         float64       `json:"fair_price_min" db:"fair_price_min"`
	FairPriceMax         float64       `json:"fair_price_max" db:"fair_price_max"`
	SystemSuggestedPrice float64       `json:"system_suggested_price" db:"system_suggested_price"`

	// Rider's initial offer
	RiderInitialOffer    *float64      `json:"rider_initial_offer,omitempty" db:"rider_initial_offer"`

	// Session state
	Status               SessionStatus `json:"status" db:"status"`
	AcceptedOfferID      *uuid.UUID    `json:"accepted_offer_id,omitempty" db:"accepted_offer_id"`
	AcceptedDriverID     *uuid.UUID    `json:"accepted_driver_id,omitempty" db:"accepted_driver_id"`
	AcceptedPrice        *float64      `json:"accepted_price,omitempty" db:"accepted_price"`

	// Timing
	ExpiresAt            time.Time     `json:"expires_at" db:"expires_at"`
	CreatedAt            time.Time     `json:"created_at" db:"created_at"`
	AcceptedAt           *time.Time    `json:"accepted_at,omitempty" db:"accepted_at"`
	CompletedAt          *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt          *time.Time    `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason   *string       `json:"cancellation_reason,omitempty" db:"cancellation_reason"`

	// Joined data
	Offers               []*Offer      `json:"offers,omitempty" db:"-"`
}

// Offer represents a driver's offer in a negotiation
type Offer struct {
	ID                  uuid.UUID   `json:"id" db:"id"`
	SessionID           uuid.UUID   `json:"session_id" db:"session_id"`
	DriverID            uuid.UUID   `json:"driver_id" db:"driver_id"`

	// Offer details
	OfferedPrice        float64     `json:"offered_price" db:"offered_price"`
	CurrencyCode        string      `json:"currency_code" db:"currency_code"`

	// Driver context
	DriverLatitude      *float64    `json:"driver_latitude,omitempty" db:"driver_latitude"`
	DriverLongitude     *float64    `json:"driver_longitude,omitempty" db:"driver_longitude"`
	EstimatedPickupTime *int        `json:"estimated_pickup_time,omitempty" db:"estimated_pickup_time"`
	DriverRating        *float64    `json:"driver_rating,omitempty" db:"driver_rating"`
	DriverTotalRides    *int        `json:"driver_total_rides,omitempty" db:"driver_total_rides"`
	VehicleModel        *string     `json:"vehicle_model,omitempty" db:"vehicle_model"`
	VehicleColor        *string     `json:"vehicle_color,omitempty" db:"vehicle_color"`

	// Offer state
	Status              OfferStatus `json:"status" db:"status"`

	// Counter-offer support
	IsCounterOffer      bool        `json:"is_counter_offer" db:"is_counter_offer"`
	ParentOfferID       *uuid.UUID  `json:"parent_offer_id,omitempty" db:"parent_offer_id"`
	CounterBy           *string     `json:"counter_by,omitempty" db:"counter_by"`

	// Timing
	CreatedAt           time.Time   `json:"created_at" db:"created_at"`
	AcceptedAt          *time.Time  `json:"accepted_at,omitempty" db:"accepted_at"`
	RejectedAt          *time.Time  `json:"rejected_at,omitempty" db:"rejected_at"`
	ExpiredAt           *time.Time  `json:"expired_at,omitempty" db:"expired_at"`
}

// Settings represents negotiation settings for a region
type Settings struct {
	ID                              uuid.UUID  `json:"id" db:"id"`
	CountryID                       *uuid.UUID `json:"country_id,omitempty" db:"country_id"`
	RegionID                        *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID                          *uuid.UUID `json:"city_id,omitempty" db:"city_id"`

	NegotiationEnabled              bool       `json:"negotiation_enabled" db:"negotiation_enabled"`
	SessionTimeoutSeconds           int        `json:"session_timeout_seconds" db:"session_timeout_seconds"`
	MaxOffersPerSession             int        `json:"max_offers_per_session" db:"max_offers_per_session"`
	MaxCounterOffers                int        `json:"max_counter_offers" db:"max_counter_offers"`
	OfferTimeoutSeconds             int        `json:"offer_timeout_seconds" db:"offer_timeout_seconds"`
	MinPriceMultiplier              float64    `json:"min_price_multiplier" db:"min_price_multiplier"`
	MaxPriceMultiplier              float64    `json:"max_price_multiplier" db:"max_price_multiplier"`
	MaxActiveSessionsPerDriver      int        `json:"max_active_sessions_per_driver" db:"max_active_sessions_per_driver"`
	MinDriverRatingToNegotiate      *float64   `json:"min_driver_rating_to_negotiate,omitempty" db:"min_driver_rating_to_negotiate"`
	MinDriverRidesToNegotiate       *int       `json:"min_driver_rides_to_negotiate,omitempty" db:"min_driver_rides_to_negotiate"`
	BlockDriversWithHighPriceStreak *int       `json:"block_drivers_with_high_price_streak,omitempty" db:"block_drivers_with_high_price_streak"`
	PriceDeviationThreshold         *float64   `json:"price_deviation_threshold,omitempty" db:"price_deviation_threshold"`

	CreatedAt                       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                       time.Time  `json:"updated_at" db:"updated_at"`
}

// DriverPricingStats represents a driver's pricing behavior stats
type DriverPricingStats struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	DriverID              uuid.UUID  `json:"driver_id" db:"driver_id"`
	RegionID              *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID                *uuid.UUID `json:"city_id,omitempty" db:"city_id"`

	TotalOffers           int        `json:"total_offers" db:"total_offers"`
	AcceptedOffers        int        `json:"accepted_offers" db:"accepted_offers"`
	AverageOfferPrice     *float64   `json:"average_offer_price,omitempty" db:"average_offer_price"`
	AverageAcceptedPrice  *float64   `json:"average_accepted_price,omitempty" db:"average_accepted_price"`

	PriceDeviationAvg     *float64   `json:"price_deviation_avg,omitempty" db:"price_deviation_avg"`
	HighPriceOffers       int        `json:"high_price_offers" db:"high_price_offers"`
	LowPriceOffers        int        `json:"low_price_offers" db:"low_price_offers"`

	PricingFairnessScore  *float64   `json:"pricing_fairness_score,omitempty" db:"pricing_fairness_score"`
	ResponseRate          *float64   `json:"response_rate,omitempty" db:"response_rate"`
	AverageResponseTime   *int       `json:"average_response_time,omitempty" db:"average_response_time"`

	ConsecutiveHighOffers int        `json:"consecutive_high_offers" db:"consecutive_high_offers"`
	FlaggedForReview      bool       `json:"flagged_for_review" db:"flagged_for_review"`
	LastReviewedAt        *time.Time `json:"last_reviewed_at,omitempty" db:"last_reviewed_at"`

	PeriodStart           time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd             *time.Time `json:"period_end,omitempty" db:"period_end"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// Participant represents a driver invited to a session
type Participant struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	SessionID           uuid.UUID  `json:"session_id" db:"session_id"`
	DriverID            uuid.UUID  `json:"driver_id" db:"driver_id"`

	NotifiedAt          *time.Time `json:"notified_at,omitempty" db:"notified_at"`
	NotificationMethod  *string    `json:"notification_method,omitempty" db:"notification_method"`

	ViewedAt            *time.Time `json:"viewed_at,omitempty" db:"viewed_at"`
	RespondedAt         *time.Time `json:"responded_at,omitempty" db:"responded_at"`
	ResponseType        *string    `json:"response_type,omitempty" db:"response_type"`

	DistanceToPickup    *float64   `json:"distance_to_pickup,omitempty" db:"distance_to_pickup"`
	EstimatedArrivalMin *int       `json:"estimated_arrival_min,omitempty" db:"estimated_arrival_min"`

	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// API Request/Response types

// StartSessionRequest represents a request to start a negotiation session
type StartSessionRequest struct {
	PickupLatitude   float64    `json:"pickup_latitude" binding:"required"`
	PickupLongitude  float64    `json:"pickup_longitude" binding:"required"`
	PickupAddress    string     `json:"pickup_address" binding:"required"`
	DropoffLatitude  float64    `json:"dropoff_latitude" binding:"required"`
	DropoffLongitude float64    `json:"dropoff_longitude" binding:"required"`
	DropoffAddress   string     `json:"dropoff_address" binding:"required"`
	RideTypeID       *uuid.UUID `json:"ride_type_id,omitempty"`
	InitialOffer     *float64   `json:"initial_offer,omitempty"`
}

// SessionResponse represents a session API response
type SessionResponse struct {
	ID                   uuid.UUID     `json:"id"`
	Status               SessionStatus `json:"status"`
	PickupAddress        string        `json:"pickup_address"`
	DropoffAddress       string        `json:"dropoff_address"`
	CurrencyCode         string        `json:"currency_code"`
	EstimatedFare        float64       `json:"estimated_fare"`
	FairPriceMin         float64       `json:"fair_price_min"`
	FairPriceMax         float64       `json:"fair_price_max"`
	SystemSuggestedPrice float64       `json:"system_suggested_price"`
	RiderInitialOffer    *float64      `json:"rider_initial_offer,omitempty"`
	AcceptedPrice        *float64      `json:"accepted_price,omitempty"`
	ExpiresAt            time.Time     `json:"expires_at"`
	OffersCount          int           `json:"offers_count"`
	Offers               []*OfferResponse `json:"offers,omitempty"`
}

// SubmitOfferRequest represents a driver's offer submission
type SubmitOfferRequest struct {
	OfferedPrice        float64  `json:"offered_price" binding:"required,gt=0"`
	DriverLatitude      *float64 `json:"driver_latitude,omitempty"`
	DriverLongitude     *float64 `json:"driver_longitude,omitempty"`
	EstimatedPickupTime *int     `json:"estimated_pickup_time,omitempty"`
}

// CounterOfferRequest represents a counter-offer
type CounterOfferRequest struct {
	OfferedPrice float64 `json:"offered_price" binding:"required,gt=0"`
}

// OfferResponse represents an offer API response
type OfferResponse struct {
	ID                  uuid.UUID   `json:"id"`
	DriverID            uuid.UUID   `json:"driver_id"`
	OfferedPrice        float64     `json:"offered_price"`
	Status              OfferStatus `json:"status"`
	EstimatedPickupTime *int        `json:"estimated_pickup_time,omitempty"`
	DriverRating        *float64    `json:"driver_rating,omitempty"`
	VehicleModel        *string     `json:"vehicle_model,omitempty"`
	VehicleColor        *string     `json:"vehicle_color,omitempty"`
	IsCounterOffer      bool        `json:"is_counter_offer"`
	CreatedAt           time.Time   `json:"created_at"`
}

// FairPriceCalculation represents the result of fair price calculation
type FairPriceCalculation struct {
	MinPrice          float64 `json:"min_price"`
	MaxPrice          float64 `json:"max_price"`
	SuggestedPrice    float64 `json:"suggested_price"`
	EstimatedFare     float64 `json:"estimated_fare"`
	HistoricalAverage *float64 `json:"historical_average,omitempty"`
	SampleSize        int     `json:"sample_size"`
}

// WebSocket message types

// WSMessageType represents WebSocket message types
type WSMessageType string

const (
	WSTypeSessionUpdate  WSMessageType = "session.update"
	WSTypeOfferNew       WSMessageType = "offer.new"
	WSTypeOfferUpdate    WSMessageType = "offer.update"
	WSTypeOfferAccepted  WSMessageType = "offer.accepted"
	WSTypeSessionExpired WSMessageType = "session.expired"
	WSTypeDriverInvite   WSMessageType = "driver.invite"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      WSMessageType `json:"type"`
	SessionID uuid.UUID     `json:"session_id"`
	Data      interface{}   `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
}

// Default settings
var DefaultSettings = Settings{
	NegotiationEnabled:              true,
	SessionTimeoutSeconds:           300,
	MaxOffersPerSession:             20,
	MaxCounterOffers:                3,
	OfferTimeoutSeconds:             60,
	MinPriceMultiplier:              0.70,
	MaxPriceMultiplier:              1.50,
	MaxActiveSessionsPerDriver:      5,
}
