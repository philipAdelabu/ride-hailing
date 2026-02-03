package geography

import (
	"time"

	"github.com/google/uuid"
)

// Country represents a country in the system
type Country struct {
	ID                      uuid.UUID `json:"id" db:"id"`
	Code                    string    `json:"code" db:"code"`         // ISO 3166-1 alpha-2
	Code3                   string    `json:"code3" db:"code3"`       // ISO 3166-1 alpha-3
	Name                    string    `json:"name" db:"name"`
	NativeName              *string   `json:"native_name,omitempty" db:"native_name"`
	CurrencyCode            string    `json:"currency_code" db:"currency_code"`
	DefaultLanguage         string    `json:"default_language" db:"default_language"`
	Timezone                string    `json:"timezone" db:"timezone"`
	PhonePrefix             string    `json:"phone_prefix" db:"phone_prefix"`
	IsActive                bool      `json:"is_active" db:"is_active"`
	LaunchedAt              *time.Time `json:"launched_at,omitempty" db:"launched_at"`
	Regulations             JSON      `json:"regulations" db:"regulations"`
	PaymentMethods          JSON      `json:"payment_methods" db:"payment_methods"`
	RequiredDriverDocuments JSON      `json:"required_driver_documents" db:"required_driver_documents"`
	CreatedAt               time.Time `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time `json:"updated_at" db:"updated_at"`
}

// Region represents a state/province within a country
type Region struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	CountryID  uuid.UUID  `json:"country_id" db:"country_id"`
	Code       string     `json:"code" db:"code"`
	Name       string     `json:"name" db:"name"`
	NativeName *string    `json:"native_name,omitempty" db:"native_name"`
	Timezone   *string    `json:"timezone,omitempty" db:"timezone"` // Override country timezone
	IsActive   bool       `json:"is_active" db:"is_active"`
	LaunchedAt *time.Time `json:"launched_at,omitempty" db:"launched_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`

	// Joined fields
	Country *Country `json:"country,omitempty" db:"-"`
}

// City represents a city within a region
type City struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	RegionID        uuid.UUID  `json:"region_id" db:"region_id"`
	Name            string     `json:"name" db:"name"`
	NativeName      *string    `json:"native_name,omitempty" db:"native_name"`
	Timezone        *string    `json:"timezone,omitempty" db:"timezone"` // Override region/country timezone
	CenterLatitude  float64    `json:"center_latitude" db:"center_latitude"`
	CenterLongitude float64    `json:"center_longitude" db:"center_longitude"`
	Boundary        *string    `json:"boundary,omitempty" db:"boundary"` // WKT geometry
	Population      *int       `json:"population,omitempty" db:"population"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	LaunchedAt      *time.Time `json:"launched_at,omitempty" db:"launched_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// Joined fields
	Region *Region `json:"region,omitempty" db:"-"`
}

// PricingZone represents a special pricing zone (airport, downtown, etc.)
type PricingZone struct {
	ID              uuid.UUID `json:"id" db:"id"`
	CityID          uuid.UUID `json:"city_id" db:"city_id"`
	Name            string    `json:"name" db:"name"`
	ZoneType        string    `json:"zone_type" db:"zone_type"` // airport, downtown, transit_hub, event_venue, etc.
	Boundary        string    `json:"boundary" db:"boundary"`   // WKT geometry
	CenterLatitude  float64   `json:"center_latitude" db:"center_latitude"`
	CenterLongitude float64   `json:"center_longitude" db:"center_longitude"`
	Priority        int       `json:"priority" db:"priority"` // Higher priority takes precedence when overlapping
	IsActive        bool      `json:"is_active" db:"is_active"`
	Metadata        JSON      `json:"metadata" db:"metadata"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields
	City *City `json:"city,omitempty" db:"-"`
}

// JSON is a helper type for JSONB fields
type JSON map[string]interface{}

// Location represents a geographic point
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ResolvedLocation contains all geographic hierarchy for a point
type ResolvedLocation struct {
	Location    Location     `json:"location"`
	Country     *Country     `json:"country,omitempty"`
	Region      *Region      `json:"region,omitempty"`
	City        *City        `json:"city,omitempty"`
	PricingZone *PricingZone `json:"pricing_zone,omitempty"`
	Timezone    string       `json:"timezone"`
}

// DriverRegion represents a driver's operating region
type DriverRegion struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	DriverID   uuid.UUID  `json:"driver_id" db:"driver_id"`
	RegionID   uuid.UUID  `json:"region_id" db:"region_id"`
	IsPrimary  bool       `json:"is_primary" db:"is_primary"`
	IsVerified bool       `json:"is_verified" db:"is_verified"`
	VerifiedAt *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`

	// Joined fields
	Region *Region `json:"region,omitempty" db:"-"`
}

// ZoneType constants
const (
	ZoneTypeAirport       = "airport"
	ZoneTypeDowntown      = "downtown"
	ZoneTypeTransitHub    = "transit_hub"
	ZoneTypeEventVenue    = "event_venue"
	ZoneTypeBorderCrossing = "border_crossing"
	ZoneTypeTollZone      = "toll_zone"
)

// CountryResponse is the API response for country
type CountryResponse struct {
	ID             uuid.UUID `json:"id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	NativeName     *string   `json:"native_name,omitempty"`
	CurrencyCode   string    `json:"currency_code"`
	Timezone       string    `json:"timezone"`
	PhonePrefix    string    `json:"phone_prefix"`
	IsActive       bool      `json:"is_active"`
	PaymentMethods []string  `json:"payment_methods,omitempty"`
}

// RegionResponse is the API response for region
type RegionResponse struct {
	ID        uuid.UUID        `json:"id"`
	Code      string           `json:"code"`
	Name      string           `json:"name"`
	NativeName *string         `json:"native_name,omitempty"`
	Timezone  string           `json:"timezone"`
	IsActive  bool             `json:"is_active"`
	Country   *CountryResponse `json:"country,omitempty"`
}

// CityResponse is the API response for city
type CityResponse struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	NativeName      *string         `json:"native_name,omitempty"`
	Timezone        string          `json:"timezone"`
	CenterLatitude  float64         `json:"center_latitude"`
	CenterLongitude float64         `json:"center_longitude"`
	Population      *int            `json:"population,omitempty"`
	IsActive        bool            `json:"is_active"`
	Region          *RegionResponse `json:"region,omitempty"`
}

// ResolveLocationRequest is the API request for location resolution
type ResolveLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// ResolveLocationResponse is the API response for location resolution
type ResolveLocationResponse struct {
	Country     *CountryResponse `json:"country,omitempty"`
	Region      *RegionResponse  `json:"region,omitempty"`
	City        *CityResponse    `json:"city,omitempty"`
	PricingZone *PricingZoneResponse `json:"pricing_zone,omitempty"`
	Timezone    string           `json:"timezone"`
}

// PricingZoneResponse is the API response for pricing zone
type PricingZoneResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	ZoneType string    `json:"zone_type"`
	Priority int       `json:"priority"`
}
