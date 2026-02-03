package geography

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for geography
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new geography repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetActiveCountries retrieves all active countries
func (r *Repository) GetActiveCountries(ctx context.Context) ([]*Country, error) {
	query := `
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries
		WHERE is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get countries: %w", err)
	}
	defer rows.Close()

	countries := make([]*Country, 0)
	for rows.Next() {
		c := &Country{}
		err := rows.Scan(
			&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
			&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
			&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
			&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, c)
	}

	return countries, nil
}

// GetCountryByCode retrieves a country by its ISO code
func (r *Repository) GetCountryByCode(ctx context.Context, code string) (*Country, error) {
	query := `
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries
		WHERE code = $1
	`

	c := &Country{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
		&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
		&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
		&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return c, nil
}

// GetCountryByID retrieves a country by its ID
func (r *Repository) GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error) {
	query := `
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries
		WHERE id = $1
	`

	c := &Country{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
		&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
		&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
		&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return c, nil
}

// GetRegionsByCountry retrieves all active regions for a country
func (r *Repository) GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error) {
	query := `
		SELECT id, country_id, code, name, native_name, timezone, is_active,
		       launched_at, created_at, updated_at
		FROM regions
		WHERE country_id = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, countryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get regions: %w", err)
	}
	defer rows.Close()

	regions := make([]*Region, 0)
	for rows.Next() {
		reg := &Region{}
		err := rows.Scan(
			&reg.ID, &reg.CountryID, &reg.Code, &reg.Name, &reg.NativeName,
			&reg.Timezone, &reg.IsActive, &reg.LaunchedAt, &reg.CreatedAt, &reg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan region: %w", err)
		}
		regions = append(regions, reg)
	}

	return regions, nil
}

// GetRegionByID retrieves a region by its ID with country
func (r *Repository) GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error) {
	query := `
		SELECT r.id, r.country_id, r.code, r.name, r.native_name, r.timezone,
		       r.is_active, r.launched_at, r.created_at, r.updated_at,
		       c.id, c.code, c.code3, c.name, c.native_name, c.currency_code,
		       c.default_language, c.timezone, c.phone_prefix, c.is_active,
		       c.launched_at, c.regulations, c.payment_methods,
		       c.required_driver_documents, c.created_at, c.updated_at
		FROM regions r
		JOIN countries c ON r.country_id = c.id
		WHERE r.id = $1
	`

	reg := &Region{Country: &Country{}}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&reg.ID, &reg.CountryID, &reg.Code, &reg.Name, &reg.NativeName,
		&reg.Timezone, &reg.IsActive, &reg.LaunchedAt, &reg.CreatedAt, &reg.UpdatedAt,
		&reg.Country.ID, &reg.Country.Code, &reg.Country.Code3, &reg.Country.Name,
		&reg.Country.NativeName, &reg.Country.CurrencyCode, &reg.Country.DefaultLanguage,
		&reg.Country.Timezone, &reg.Country.PhonePrefix, &reg.Country.IsActive,
		&reg.Country.LaunchedAt, &reg.Country.Regulations, &reg.Country.PaymentMethods,
		&reg.Country.RequiredDriverDocuments, &reg.Country.CreatedAt, &reg.Country.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	return reg, nil
}

// GetCitiesByRegion retrieves all active cities for a region
func (r *Repository) GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error) {
	query := `
		SELECT id, region_id, name, native_name, timezone, center_latitude,
		       center_longitude, ST_AsText(boundary), population, is_active,
		       launched_at, created_at, updated_at
		FROM cities
		WHERE region_id = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cities: %w", err)
	}
	defer rows.Close()

	cities := make([]*City, 0)
	for rows.Next() {
		city := &City{}
		err := rows.Scan(
			&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
			&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
			&city.Population, &city.IsActive, &city.LaunchedAt,
			&city.CreatedAt, &city.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan city: %w", err)
		}
		cities = append(cities, city)
	}

	return cities, nil
}

// GetCityByID retrieves a city by its ID with region and country
func (r *Repository) GetCityByID(ctx context.Context, id uuid.UUID) (*City, error) {
	query := `
		SELECT city.id, city.region_id, city.name, city.native_name, city.timezone,
		       city.center_latitude, city.center_longitude, ST_AsText(city.boundary),
		       city.population, city.is_active, city.launched_at, city.created_at, city.updated_at,
		       reg.id, reg.country_id, reg.code, reg.name, reg.native_name, reg.timezone,
		       reg.is_active, reg.launched_at, reg.created_at, reg.updated_at,
		       c.id, c.code, c.code3, c.name, c.native_name, c.currency_code,
		       c.default_language, c.timezone, c.phone_prefix, c.is_active,
		       c.launched_at, c.regulations, c.payment_methods,
		       c.required_driver_documents, c.created_at, c.updated_at
		FROM cities city
		JOIN regions reg ON city.region_id = reg.id
		JOIN countries c ON reg.country_id = c.id
		WHERE city.id = $1
	`

	city := &City{Region: &Region{Country: &Country{}}}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
		&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
		&city.Population, &city.IsActive, &city.LaunchedAt, &city.CreatedAt, &city.UpdatedAt,
		&city.Region.ID, &city.Region.CountryID, &city.Region.Code, &city.Region.Name,
		&city.Region.NativeName, &city.Region.Timezone, &city.Region.IsActive,
		&city.Region.LaunchedAt, &city.Region.CreatedAt, &city.Region.UpdatedAt,
		&city.Region.Country.ID, &city.Region.Country.Code, &city.Region.Country.Code3,
		&city.Region.Country.Name, &city.Region.Country.NativeName,
		&city.Region.Country.CurrencyCode, &city.Region.Country.DefaultLanguage,
		&city.Region.Country.Timezone, &city.Region.Country.PhonePrefix,
		&city.Region.Country.IsActive, &city.Region.Country.LaunchedAt,
		&city.Region.Country.Regulations, &city.Region.Country.PaymentMethods,
		&city.Region.Country.RequiredDriverDocuments, &city.Region.Country.CreatedAt,
		&city.Region.Country.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get city: %w", err)
	}

	return city, nil
}

// ResolveLocation finds the geographic hierarchy for a lat/lng point
func (r *Repository) ResolveLocation(ctx context.Context, lat, lng float64) (*ResolvedLocation, error) {
	result := &ResolvedLocation{
		Location: Location{Latitude: lat, Longitude: lng},
	}

	// Find city containing the point (uses PostGIS ST_Contains)
	cityQuery := `
		SELECT city.id, city.region_id, city.name, city.native_name, city.timezone,
		       city.center_latitude, city.center_longitude, ST_AsText(city.boundary),
		       city.population, city.is_active, city.launched_at, city.created_at, city.updated_at,
		       reg.id, reg.country_id, reg.code, reg.name, reg.native_name, reg.timezone,
		       reg.is_active, reg.launched_at, reg.created_at, reg.updated_at,
		       c.id, c.code, c.code3, c.name, c.native_name, c.currency_code,
		       c.default_language, c.timezone, c.phone_prefix, c.is_active,
		       c.launched_at, c.regulations, c.payment_methods,
		       c.required_driver_documents, c.created_at, c.updated_at
		FROM cities city
		JOIN regions reg ON city.region_id = reg.id
		JOIN countries c ON reg.country_id = c.id
		WHERE city.is_active = true
		  AND reg.is_active = true
		  AND c.is_active = true
		  AND ST_Contains(city.boundary, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		ORDER BY city.population DESC NULLS LAST
		LIMIT 1
	`

	city := &City{Region: &Region{Country: &Country{}}}
	err := r.db.QueryRow(ctx, cityQuery, lng, lat).Scan(
		&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
		&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
		&city.Population, &city.IsActive, &city.LaunchedAt, &city.CreatedAt, &city.UpdatedAt,
		&city.Region.ID, &city.Region.CountryID, &city.Region.Code, &city.Region.Name,
		&city.Region.NativeName, &city.Region.Timezone, &city.Region.IsActive,
		&city.Region.LaunchedAt, &city.Region.CreatedAt, &city.Region.UpdatedAt,
		&city.Region.Country.ID, &city.Region.Country.Code, &city.Region.Country.Code3,
		&city.Region.Country.Name, &city.Region.Country.NativeName,
		&city.Region.Country.CurrencyCode, &city.Region.Country.DefaultLanguage,
		&city.Region.Country.Timezone, &city.Region.Country.PhonePrefix,
		&city.Region.Country.IsActive, &city.Region.Country.LaunchedAt,
		&city.Region.Country.Regulations, &city.Region.Country.PaymentMethods,
		&city.Region.Country.RequiredDriverDocuments, &city.Region.Country.CreatedAt,
		&city.Region.Country.UpdatedAt,
	)
	if err == nil {
		result.City = city
		result.Region = city.Region
		result.Country = city.Region.Country

		// Resolve timezone (city > region > country)
		if city.Timezone != nil && *city.Timezone != "" {
			result.Timezone = *city.Timezone
		} else if city.Region.Timezone != nil && *city.Region.Timezone != "" {
			result.Timezone = *city.Region.Timezone
		} else {
			result.Timezone = city.Region.Country.Timezone
		}
	}

	// Find pricing zone containing the point (highest priority)
	if result.City != nil {
		zoneQuery := `
			SELECT id, city_id, name, zone_type, ST_AsText(boundary),
			       center_latitude, center_longitude, priority, is_active,
			       metadata, created_at, updated_at
			FROM pricing_zones
			WHERE city_id = $1
			  AND is_active = true
			  AND ST_Contains(boundary, ST_SetSRID(ST_MakePoint($2, $3), 4326))
			ORDER BY priority DESC
			LIMIT 1
		`

		zone := &PricingZone{}
		err := r.db.QueryRow(ctx, zoneQuery, result.City.ID, lng, lat).Scan(
			&zone.ID, &zone.CityID, &zone.Name, &zone.ZoneType, &zone.Boundary,
			&zone.CenterLatitude, &zone.CenterLongitude, &zone.Priority,
			&zone.IsActive, &zone.Metadata, &zone.CreatedAt, &zone.UpdatedAt,
		)
		if err == nil {
			result.PricingZone = zone
		}
	}

	return result, nil
}

// FindNearestCity finds the nearest city to a point
func (r *Repository) FindNearestCity(ctx context.Context, lat, lng float64, maxDistanceKm float64) (*City, error) {
	query := `
		SELECT city.id, city.region_id, city.name, city.native_name, city.timezone,
		       city.center_latitude, city.center_longitude, ST_AsText(city.boundary),
		       city.population, city.is_active, city.launched_at, city.created_at, city.updated_at,
		       ST_Distance(
		           ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
		           ST_SetSRID(ST_MakePoint(city.center_longitude, city.center_latitude), 4326)::geography
		       ) / 1000 AS distance_km
		FROM cities city
		JOIN regions reg ON city.region_id = reg.id
		JOIN countries c ON reg.country_id = c.id
		WHERE city.is_active = true
		  AND reg.is_active = true
		  AND c.is_active = true
		ORDER BY distance_km
		LIMIT 1
	`

	city := &City{}
	var distanceKm float64
	err := r.db.QueryRow(ctx, query, lng, lat).Scan(
		&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
		&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
		&city.Population, &city.IsActive, &city.LaunchedAt, &city.CreatedAt, &city.UpdatedAt,
		&distanceKm,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearest city: %w", err)
	}

	if distanceKm > maxDistanceKm {
		return nil, fmt.Errorf("nearest city is %.2f km away, exceeds max %.2f km", distanceKm, maxDistanceKm)
	}

	return city, nil
}

// GetPricingZonesByCity retrieves all active pricing zones for a city
func (r *Repository) GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error) {
	query := `
		SELECT id, city_id, name, zone_type, ST_AsText(boundary),
		       center_latitude, center_longitude, priority, is_active,
		       metadata, created_at, updated_at
		FROM pricing_zones
		WHERE city_id = $1 AND is_active = true
		ORDER BY priority DESC, name
	`

	rows, err := r.db.Query(ctx, query, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing zones: %w", err)
	}
	defer rows.Close()

	zones := make([]*PricingZone, 0)
	for rows.Next() {
		zone := &PricingZone{}
		err := rows.Scan(
			&zone.ID, &zone.CityID, &zone.Name, &zone.ZoneType, &zone.Boundary,
			&zone.CenterLatitude, &zone.CenterLongitude, &zone.Priority,
			&zone.IsActive, &zone.Metadata, &zone.CreatedAt, &zone.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing zone: %w", err)
		}
		zones = append(zones, zone)
	}

	return zones, nil
}

// GetDriverRegions retrieves all operating regions for a driver
func (r *Repository) GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error) {
	query := `
		SELECT dr.id, dr.driver_id, dr.region_id, dr.is_primary, dr.is_verified,
		       dr.verified_at, dr.created_at,
		       reg.id, reg.country_id, reg.code, reg.name, reg.native_name,
		       reg.timezone, reg.is_active, reg.launched_at, reg.created_at, reg.updated_at
		FROM driver_regions dr
		JOIN regions reg ON dr.region_id = reg.id
		WHERE dr.driver_id = $1
		ORDER BY dr.is_primary DESC, reg.name
	`

	rows, err := r.db.Query(ctx, query, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver regions: %w", err)
	}
	defer rows.Close()

	driverRegions := make([]*DriverRegion, 0)
	for rows.Next() {
		dr := &DriverRegion{Region: &Region{}}
		err := rows.Scan(
			&dr.ID, &dr.DriverID, &dr.RegionID, &dr.IsPrimary, &dr.IsVerified,
			&dr.VerifiedAt, &dr.CreatedAt,
			&dr.Region.ID, &dr.Region.CountryID, &dr.Region.Code, &dr.Region.Name,
			&dr.Region.NativeName, &dr.Region.Timezone, &dr.Region.IsActive,
			&dr.Region.LaunchedAt, &dr.Region.CreatedAt, &dr.Region.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver region: %w", err)
		}
		driverRegions = append(driverRegions, dr)
	}

	return driverRegions, nil
}

// AddDriverRegion adds a new operating region for a driver
func (r *Repository) AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error {
	// If setting as primary, unset other primary regions first
	if isPrimary {
		_, err := r.db.Exec(ctx,
			`UPDATE driver_regions SET is_primary = false WHERE driver_id = $1`,
			driverID,
		)
		if err != nil {
			return fmt.Errorf("failed to unset primary region: %w", err)
		}
	}

	query := `
		INSERT INTO driver_regions (id, driver_id, region_id, is_primary, is_verified)
		VALUES ($1, $2, $3, $4, false)
		ON CONFLICT (driver_id, region_id) DO UPDATE SET is_primary = $4
	`

	_, err := r.db.Exec(ctx, query, uuid.New(), driverID, regionID, isPrimary)
	if err != nil {
		return fmt.Errorf("failed to add driver region: %w", err)
	}

	return nil
}

// CreateCountry creates a new country
func (r *Repository) CreateCountry(ctx context.Context, country *Country) error {
	query := `
		INSERT INTO countries (id, code, code3, name, native_name, currency_code,
		                       default_language, timezone, phone_prefix, is_active,
		                       regulations, payment_methods, required_driver_documents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at
	`

	country.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		country.ID, country.Code, country.Code3, country.Name, country.NativeName,
		country.CurrencyCode, country.DefaultLanguage, country.Timezone,
		country.PhonePrefix, country.IsActive, country.Regulations,
		country.PaymentMethods, country.RequiredDriverDocuments,
	).Scan(&country.CreatedAt, &country.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create country: %w", err)
	}

	return nil
}

// CreateRegion creates a new region
func (r *Repository) CreateRegion(ctx context.Context, region *Region) error {
	query := `
		INSERT INTO regions (id, country_id, code, name, native_name, timezone, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	region.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		region.ID, region.CountryID, region.Code, region.Name,
		region.NativeName, region.Timezone, region.IsActive,
	).Scan(&region.CreatedAt, &region.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}

	return nil
}

// CreateCity creates a new city
func (r *Repository) CreateCity(ctx context.Context, city *City) error {
	query := `
		INSERT INTO cities (id, region_id, name, native_name, timezone,
		                    center_latitude, center_longitude, boundary, population, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7,
		        CASE WHEN $8::text IS NOT NULL THEN ST_GeomFromText($8, 4326) ELSE NULL END,
		        $9, $10)
		RETURNING created_at, updated_at
	`

	city.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		city.ID, city.RegionID, city.Name, city.NativeName, city.Timezone,
		city.CenterLatitude, city.CenterLongitude, city.Boundary,
		city.Population, city.IsActive,
	).Scan(&city.CreatedAt, &city.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create city: %w", err)
	}

	return nil
}

// CreatePricingZone creates a new pricing zone
func (r *Repository) CreatePricingZone(ctx context.Context, zone *PricingZone) error {
	query := `
		INSERT INTO pricing_zones (id, city_id, name, zone_type, boundary,
		                           center_latitude, center_longitude, priority,
		                           is_active, metadata)
		VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326), $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`

	zone.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		zone.ID, zone.CityID, zone.Name, zone.ZoneType, zone.Boundary,
		zone.CenterLatitude, zone.CenterLongitude, zone.Priority,
		zone.IsActive, zone.Metadata,
	).Scan(&zone.CreatedAt, &zone.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create pricing zone: %w", err)
	}

	return nil
}
