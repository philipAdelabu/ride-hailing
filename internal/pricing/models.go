package pricing

import (
	"time"

	"github.com/google/uuid"
)

// PricingConfig represents a pricing configuration at any hierarchy level
type PricingConfig struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	VersionID            uuid.UUID  `json:"version_id" db:"version_id"`
	CountryID            *uuid.UUID `json:"country_id,omitempty" db:"country_id"`
	RegionID             *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID               *uuid.UUID `json:"city_id,omitempty" db:"city_id"`
	ZoneID               *uuid.UUID `json:"zone_id,omitempty" db:"zone_id"`
	RideTypeID           *uuid.UUID `json:"ride_type_id,omitempty" db:"ride_type_id"`

	// Core pricing (nullable for inheritance)
	BaseFare          *float64 `json:"base_fare,omitempty" db:"base_fare"`
	PerKmRate         *float64 `json:"per_km_rate,omitempty" db:"per_km_rate"`
	PerMinuteRate     *float64 `json:"per_minute_rate,omitempty" db:"per_minute_rate"`
	MinimumFare       *float64 `json:"minimum_fare,omitempty" db:"minimum_fare"`
	BookingFee        *float64 `json:"booking_fee,omitempty" db:"booking_fee"`

	// Commission
	PlatformCommissionPct *float64 `json:"platform_commission_pct,omitempty" db:"platform_commission_pct"`
	DriverIncentivePct    *float64 `json:"driver_incentive_pct,omitempty" db:"driver_incentive_pct"`

	// Surge limits
	SurgeMinMultiplier *float64 `json:"surge_min_multiplier,omitempty" db:"surge_min_multiplier"`
	SurgeMaxMultiplier *float64 `json:"surge_max_multiplier,omitempty" db:"surge_max_multiplier"`

	// Tax
	TaxRatePct   *float64 `json:"tax_rate_pct,omitempty" db:"tax_rate_pct"`
	TaxInclusive *bool    `json:"tax_inclusive,omitempty" db:"tax_inclusive"`

	// Cancellation fees (JSON array)
	CancellationFees []CancellationFee `json:"cancellation_fees,omitempty" db:"cancellation_fees"`

	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CancellationFee represents a tiered cancellation fee
type CancellationFee struct {
	AfterMinutes int     `json:"after_minutes"`
	Fee          float64 `json:"fee"`
	FeeType      string  `json:"fee_type"` // "fixed" or "percentage"
}

// ResolvedPricing represents the fully resolved pricing for a location
type ResolvedPricing struct {
	// Source IDs for audit
	VersionID  uuid.UUID  `json:"version_id"`
	CountryID  *uuid.UUID `json:"country_id,omitempty"`
	RegionID   *uuid.UUID `json:"region_id,omitempty"`
	CityID     *uuid.UUID `json:"city_id,omitempty"`
	ZoneID     *uuid.UUID `json:"zone_id,omitempty"`
	RideTypeID *uuid.UUID `json:"ride_type_id,omitempty"`

	// Resolved values (guaranteed non-nil)
	BaseFare              float64 `json:"base_fare"`
	PerKmRate             float64 `json:"per_km_rate"`
	PerMinuteRate         float64 `json:"per_minute_rate"`
	MinimumFare           float64 `json:"minimum_fare"`
	BookingFee            float64 `json:"booking_fee"`
	PlatformCommissionPct float64 `json:"platform_commission_pct"`
	DriverIncentivePct    float64 `json:"driver_incentive_pct"`
	SurgeMinMultiplier    float64 `json:"surge_min_multiplier"`
	SurgeMaxMultiplier    float64 `json:"surge_max_multiplier"`
	TaxRatePct            float64 `json:"tax_rate_pct"`
	TaxInclusive          bool    `json:"tax_inclusive"`
	CancellationFees      []CancellationFee `json:"cancellation_fees"`

	// Inheritance chain for debugging
	InheritanceChain []string `json:"inheritance_chain,omitempty"`
}

// ZoneFee represents an additional fee for a pricing zone
type ZoneFee struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ZoneID        uuid.UUID  `json:"zone_id" db:"zone_id"`
	VersionID     uuid.UUID  `json:"version_id" db:"version_id"`
	FeeType       string     `json:"fee_type" db:"fee_type"` // pickup_fee, dropoff_fee, toll, etc.
	RideTypeID    *uuid.UUID `json:"ride_type_id,omitempty" db:"ride_type_id"`
	Amount        float64    `json:"amount" db:"amount"`
	IsPercentage  bool       `json:"is_percentage" db:"is_percentage"`
	AppliesPickup bool       `json:"applies_pickup" db:"applies_pickup"`
	AppliesDropoff bool      `json:"applies_dropoff" db:"applies_dropoff"`
	Schedule      *FeeSchedule `json:"schedule,omitempty" db:"schedule"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// FeeSchedule defines when a fee applies
type FeeSchedule struct {
	Days      []int  `json:"days"`       // 0=Sunday, 6=Saturday
	StartTime string `json:"start_time"` // HH:MM
	EndTime   string `json:"end_time"`   // HH:MM
}

// TimeMultiplier represents a time-based pricing multiplier
type TimeMultiplier struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	VersionID   uuid.UUID  `json:"version_id" db:"version_id"`
	CountryID   *uuid.UUID `json:"country_id,omitempty" db:"country_id"`
	RegionID    *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID      *uuid.UUID `json:"city_id,omitempty" db:"city_id"`
	Name        string     `json:"name" db:"name"`
	DaysOfWeek  []int      `json:"days_of_week" db:"days_of_week"` // 0=Sunday, 6=Saturday
	StartTime   string     `json:"start_time" db:"start_time"`
	EndTime     string     `json:"end_time" db:"end_time"`
	Multiplier  float64    `json:"multiplier" db:"multiplier"`
	Priority    int        `json:"priority" db:"priority"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// WeatherMultiplier represents a weather-based pricing multiplier
type WeatherMultiplier struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	VersionID        uuid.UUID  `json:"version_id" db:"version_id"`
	CountryID        *uuid.UUID `json:"country_id,omitempty" db:"country_id"`
	RegionID         *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID           *uuid.UUID `json:"city_id,omitempty" db:"city_id"`
	WeatherCondition string     `json:"weather_condition" db:"weather_condition"` // rain, snow, storm, etc.
	Multiplier       float64    `json:"multiplier" db:"multiplier"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// EventMultiplier represents an event-based pricing multiplier
type EventMultiplier struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	VersionID            uuid.UUID  `json:"version_id" db:"version_id"`
	ZoneID               *uuid.UUID `json:"zone_id,omitempty" db:"zone_id"`
	CityID               *uuid.UUID `json:"city_id,omitempty" db:"city_id"`
	EventName            string     `json:"event_name" db:"event_name"`
	EventType            string     `json:"event_type" db:"event_type"` // sports, concert, conference, etc.
	StartsAt             time.Time  `json:"starts_at" db:"starts_at"`
	EndsAt               time.Time  `json:"ends_at" db:"ends_at"`
	PreEventMinutes      int        `json:"pre_event_minutes" db:"pre_event_minutes"`
	PostEventMinutes     int        `json:"post_event_minutes" db:"post_event_minutes"`
	Multiplier           float64    `json:"multiplier" db:"multiplier"`
	ExpectedDemandIncrease *int     `json:"expected_demand_increase,omitempty" db:"expected_demand_increase"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// SurgeThreshold represents a demand/supply ratio threshold
type SurgeThreshold struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	VersionID           uuid.UUID  `json:"version_id" db:"version_id"`
	CountryID           *uuid.UUID `json:"country_id,omitempty" db:"country_id"`
	RegionID            *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID              *uuid.UUID `json:"city_id,omitempty" db:"city_id"`
	DemandSupplyRatioMin float64   `json:"demand_supply_ratio_min" db:"demand_supply_ratio_min"`
	DemandSupplyRatioMax *float64  `json:"demand_supply_ratio_max,omitempty" db:"demand_supply_ratio_max"`
	Multiplier          float64    `json:"multiplier" db:"multiplier"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// FareCalculation represents a complete fare calculation
type FareCalculation struct {
	// Input
	DistanceKm  float64 `json:"distance_km"`
	DurationMin int     `json:"duration_min"`
	Currency    string  `json:"currency"`

	// Base calculation
	BaseFare       float64 `json:"base_fare"`
	DistanceCharge float64 `json:"distance_charge"`
	TimeCharge     float64 `json:"time_charge"`
	BookingFee     float64 `json:"booking_fee"`

	// Zone fees
	ZoneFeesTotal     float64            `json:"zone_fees_total"`
	ZoneFeesBreakdown []ZoneFeeBreakdown `json:"zone_fees_breakdown,omitempty"`

	// Multipliers
	TimeMultiplier    float64 `json:"time_multiplier"`
	WeatherMultiplier float64 `json:"weather_multiplier"`
	EventMultiplier   float64 `json:"event_multiplier"`
	SurgeMultiplier   float64 `json:"surge_multiplier"`
	TotalMultiplier   float64 `json:"total_multiplier"`

	// Final calculation
	Subtotal   float64 `json:"subtotal"`
	TaxRatePct float64 `json:"tax_rate_pct"`
	TaxAmount  float64 `json:"tax_amount"`
	TotalFare  float64 `json:"total_fare"`

	// Commission split
	PlatformCommissionPct float64 `json:"platform_commission_pct"`
	PlatformCommission    float64 `json:"platform_commission"`
	DriverEarnings        float64 `json:"driver_earnings"`

	// Metadata
	PricingVersionID uuid.UUID `json:"pricing_version_id"`
	WasNegotiated    bool      `json:"was_negotiated"`
	NegotiatedFare   *float64  `json:"negotiated_fare,omitempty"`
}

// ZoneFeeBreakdown represents a breakdown of an applied zone fee
type ZoneFeeBreakdown struct {
	ZoneID   uuid.UUID `json:"zone_id"`
	ZoneName string    `json:"zone_name"`
	FeeType  string    `json:"fee_type"`
	Amount   float64   `json:"amount"`
}

// EstimateRequest represents a fare estimate request
type EstimateRequest struct {
	PickupLatitude   float64    `json:"pickup_latitude" binding:"required"`
	PickupLongitude  float64    `json:"pickup_longitude" binding:"required"`
	DropoffLatitude  float64    `json:"dropoff_latitude" binding:"required"`
	DropoffLongitude float64    `json:"dropoff_longitude" binding:"required"`
	RideTypeID       *uuid.UUID `json:"ride_type_id,omitempty"`
}

// EstimateResponse represents a fare estimate response
type EstimateResponse struct {
	Currency         string  `json:"currency"`
	EstimatedFare    float64 `json:"estimated_fare"`
	MinimumFare      float64 `json:"minimum_fare"`
	SurgeMultiplier  float64 `json:"surge_multiplier"`
	DistanceKm       float64 `json:"distance_km"`
	EstimatedMinutes int     `json:"estimated_minutes"`
	FareBreakdown    *FareCalculation `json:"fare_breakdown,omitempty"`
	FormattedFare    string  `json:"formatted_fare"`
}

// WeatherCondition constants
const (
	WeatherClear     = "clear"
	WeatherCloudy    = "cloudy"
	WeatherRain      = "rain"
	WeatherHeavyRain = "heavy_rain"
	WeatherSnow      = "snow"
	WeatherStorm     = "storm"
	WeatherHeat      = "extreme_heat"
	WeatherFog       = "fog"
)

// EventType constants
const (
	EventTypeSports     = "sports"
	EventTypeConcert    = "concert"
	EventTypeConference = "conference"
	EventTypeHoliday    = "holiday"
	EventTypeFestival   = "festival"
	EventTypeOther      = "other"
)

// Default values for global fallback
var DefaultPricing = ResolvedPricing{
	BaseFare:              3.00,
	PerKmRate:             1.50,
	PerMinuteRate:         0.25,
	MinimumFare:           5.00,
	BookingFee:            1.00,
	PlatformCommissionPct: 20.00,
	DriverIncentivePct:    0.00,
	SurgeMinMultiplier:    1.00,
	SurgeMaxMultiplier:    5.00,
	TaxRatePct:            0.00,
	TaxInclusive:          false,
	CancellationFees: []CancellationFee{
		{AfterMinutes: 0, Fee: 0, FeeType: "fixed"},
		{AfterMinutes: 2, Fee: 5, FeeType: "fixed"},
		{AfterMinutes: 5, Fee: 10, FeeType: "fixed"},
	},
}
