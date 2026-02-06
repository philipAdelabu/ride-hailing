package geography

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for geography repository operations
type RepositoryInterface interface {
	GetActiveCountries(ctx context.Context) ([]*Country, error)
	GetCountryByCode(ctx context.Context, code string) (*Country, error)
	GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error)
	GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error)
	GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error)
	GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error)
	GetCityByID(ctx context.Context, id uuid.UUID) (*City, error)
	ResolveLocation(ctx context.Context, lat, lng float64) (*ResolvedLocation, error)
	FindNearestCity(ctx context.Context, lat, lng float64, maxDistanceKm float64) (*City, error)
	GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error)
	GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error)
	AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error
	CreateCountry(ctx context.Context, country *Country) error
	CreateRegion(ctx context.Context, region *Region) error
	CreateCity(ctx context.Context, city *City) error
	CreatePricingZone(ctx context.Context, zone *PricingZone) error
}
