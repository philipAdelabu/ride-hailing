package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for pricing
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new pricing repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetActiveVersionID returns the currently active pricing version ID
func (r *Repository) GetActiveVersionID(ctx context.Context) (uuid.UUID, error) {
	query := `
		SELECT id FROM pricing_config_versions
		WHERE status = 'active'
		  AND (effective_from IS NULL OR effective_from <= NOW())
		  AND (effective_until IS NULL OR effective_until > NOW())
		ORDER BY effective_from DESC NULLS LAST
		LIMIT 1
	`

	var versionID uuid.UUID
	err := r.db.QueryRow(ctx, query).Scan(&versionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get active pricing version: %w", err)
	}

	return versionID, nil
}

// GetPricingConfigsForResolution retrieves all configs needed to resolve pricing for a location
func (r *Repository) GetPricingConfigsForResolution(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID, zoneID, rideTypeID *uuid.UUID) ([]*PricingConfig, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id, zone_id, ride_type_id,
		       base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct,
		       surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active, created_at, updated_at
		FROM pricing_configs
		WHERE version_id = $1
		  AND is_active = true
		  AND (
		    -- Global config (no location scope)
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL AND zone_id IS NULL)
		    -- Country level
		    OR (country_id = $2 AND region_id IS NULL AND city_id IS NULL AND zone_id IS NULL)
		    -- Region level
		    OR (region_id = $3 AND city_id IS NULL AND zone_id IS NULL)
		    -- City level
		    OR (city_id = $4 AND zone_id IS NULL)
		    -- Zone level
		    OR zone_id = $5
		  )
		  AND (ride_type_id IS NULL OR ride_type_id = $6)
		ORDER BY
		  -- Hierarchy level (more specific = higher priority)
		  CASE
		    WHEN zone_id IS NOT NULL THEN 4
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC,
		  -- Ride type specificity (specific ride type > all ride types)
		  CASE WHEN ride_type_id IS NOT NULL THEN 1 ELSE 0 END DESC
	`

	rows, err := r.db.Query(ctx, query, versionID, countryID, regionID, cityID, zoneID, rideTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing configs: %w", err)
	}
	defer rows.Close()

	configs := make([]*PricingConfig, 0)
	for rows.Next() {
		config := &PricingConfig{}
		var cancellationFeesJSON []byte

		err := rows.Scan(
			&config.ID, &config.VersionID, &config.CountryID, &config.RegionID,
			&config.CityID, &config.ZoneID, &config.RideTypeID,
			&config.BaseFare, &config.PerKmRate, &config.PerMinuteRate,
			&config.MinimumFare, &config.BookingFee,
			&config.PlatformCommissionPct, &config.DriverIncentivePct,
			&config.SurgeMinMultiplier, &config.SurgeMaxMultiplier,
			&config.TaxRatePct, &config.TaxInclusive, &cancellationFeesJSON,
			&config.IsActive, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing config: %w", err)
		}

		// Parse cancellation fees JSON
		if len(cancellationFeesJSON) > 0 {
			if err := json.Unmarshal(cancellationFeesJSON, &config.CancellationFees); err != nil {
				return nil, fmt.Errorf("failed to parse cancellation fees: %w", err)
			}
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// GetZoneFees retrieves zone fees for pickup and dropoff zones
func (r *Repository) GetZoneFees(ctx context.Context, versionID uuid.UUID, pickupZoneID, dropoffZoneID *uuid.UUID, rideTypeID *uuid.UUID) ([]*ZoneFee, error) {
	if pickupZoneID == nil && dropoffZoneID == nil {
		return nil, nil
	}

	query := `
		SELECT id, zone_id, version_id, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule,
		       is_active, created_at, updated_at
		FROM zone_fees
		WHERE version_id = $1
		  AND is_active = true
		  AND (zone_id = $2 OR zone_id = $3)
		  AND (ride_type_id IS NULL OR ride_type_id = $4)
	`

	rows, err := r.db.Query(ctx, query, versionID, pickupZoneID, dropoffZoneID, rideTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone fees: %w", err)
	}
	defer rows.Close()

	fees := make([]*ZoneFee, 0)
	for rows.Next() {
		fee := &ZoneFee{}
		var scheduleJSON []byte

		err := rows.Scan(
			&fee.ID, &fee.ZoneID, &fee.VersionID, &fee.FeeType, &fee.RideTypeID,
			&fee.Amount, &fee.IsPercentage, &fee.AppliesPickup, &fee.AppliesDropoff,
			&scheduleJSON, &fee.IsActive, &fee.CreatedAt, &fee.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan zone fee: %w", err)
		}

		if len(scheduleJSON) > 0 {
			fee.Schedule = &FeeSchedule{}
			if err := json.Unmarshal(scheduleJSON, fee.Schedule); err != nil {
				return nil, fmt.Errorf("failed to parse fee schedule: %w", err)
			}
		}

		fees = append(fees, fee)
	}

	return fees, nil
}

// GetTimeMultipliers retrieves time-based multipliers
func (r *Repository) GetTimeMultipliers(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, t time.Time) ([]*TimeMultiplier, error) {
	dayOfWeek := int(t.Weekday())
	timeOfDay := t.Format("15:04")

	query := `
		SELECT id, version_id, country_id, region_id, city_id, name,
		       days_of_week, start_time, end_time, multiplier, priority,
		       is_active, created_at, updated_at
		FROM time_multipliers
		WHERE version_id = $1
		  AND is_active = true
		  AND $2 = ANY(days_of_week)
		  AND (
		    (start_time <= end_time AND $3::time >= start_time AND $3::time <= end_time)
		    OR (start_time > end_time AND ($3::time >= start_time OR $3::time <= end_time))
		  )
		  AND (
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL)
		    OR (country_id = $4 AND region_id IS NULL AND city_id IS NULL)
		    OR (region_id = $5 AND city_id IS NULL)
		    OR city_id = $6
		  )
		ORDER BY priority DESC
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, versionID, dayOfWeek, timeOfDay, countryID, regionID, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time multipliers: %w", err)
	}
	defer rows.Close()

	multipliers := make([]*TimeMultiplier, 0)
	for rows.Next() {
		m := &TimeMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
			&m.Name, &m.DaysOfWeek, &m.StartTime, &m.EndTime, &m.Multiplier,
			&m.Priority, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time multiplier: %w", err)
		}
		multipliers = append(multipliers, m)
	}

	return multipliers, nil
}

// GetWeatherMultiplier retrieves weather-based multiplier
func (r *Repository) GetWeatherMultiplier(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, condition string) (*WeatherMultiplier, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active, created_at
		FROM weather_multipliers
		WHERE version_id = $1
		  AND is_active = true
		  AND weather_condition = $2
		  AND (
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL)
		    OR (country_id = $3 AND region_id IS NULL AND city_id IS NULL)
		    OR (region_id = $4 AND city_id IS NULL)
		    OR city_id = $5
		  )
		ORDER BY
		  CASE
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC
		LIMIT 1
	`

	m := &WeatherMultiplier{}
	err := r.db.QueryRow(ctx, query, versionID, condition, countryID, regionID, cityID).Scan(
		&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
		&m.WeatherCondition, &m.Multiplier, &m.IsActive, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather multiplier: %w", err)
	}

	return m, nil
}

// GetActiveEventMultipliers retrieves event-based multipliers
func (r *Repository) GetActiveEventMultipliers(ctx context.Context, versionID uuid.UUID, cityID, zoneID *uuid.UUID, t time.Time) ([]*EventMultiplier, error) {
	query := `
		SELECT id, version_id, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes,
		       multiplier, expected_demand_increase, is_active, created_at
		FROM event_multipliers
		WHERE version_id = $1
		  AND is_active = true
		  AND (city_id = $2 OR zone_id = $3)
		  AND $4 >= (starts_at - (pre_event_minutes || ' minutes')::interval)
		  AND $4 <= (ends_at + (post_event_minutes || ' minutes')::interval)
		ORDER BY multiplier DESC
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, versionID, cityID, zoneID, t)
	if err != nil {
		return nil, fmt.Errorf("failed to get event multipliers: %w", err)
	}
	defer rows.Close()

	multipliers := make([]*EventMultiplier, 0)
	for rows.Next() {
		m := &EventMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.ZoneID, &m.CityID, &m.EventName,
			&m.EventType, &m.StartsAt, &m.EndsAt, &m.PreEventMinutes,
			&m.PostEventMinutes, &m.Multiplier, &m.ExpectedDemandIncrease,
			&m.IsActive, &m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event multiplier: %w", err)
		}
		multipliers = append(multipliers, m)
	}

	return multipliers, nil
}

// GetSurgeThresholds retrieves surge thresholds for a location
func (r *Repository) GetSurgeThresholds(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID) ([]*SurgeThreshold, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier,
		       is_active, created_at
		FROM surge_thresholds
		WHERE version_id = $1
		  AND is_active = true
		  AND (
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL)
		    OR (country_id = $2 AND region_id IS NULL AND city_id IS NULL)
		    OR (region_id = $3 AND city_id IS NULL)
		    OR city_id = $4
		  )
		ORDER BY
		  CASE
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC,
		  demand_supply_ratio_min ASC
	`

	rows, err := r.db.Query(ctx, query, versionID, countryID, regionID, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get surge thresholds: %w", err)
	}
	defer rows.Close()

	thresholds := make([]*SurgeThreshold, 0)
	for rows.Next() {
		t := &SurgeThreshold{}
		err := rows.Scan(
			&t.ID, &t.VersionID, &t.CountryID, &t.RegionID, &t.CityID,
			&t.DemandSupplyRatioMin, &t.DemandSupplyRatioMax, &t.Multiplier,
			&t.IsActive, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan surge threshold: %w", err)
		}
		thresholds = append(thresholds, t)
	}

	return thresholds, nil
}

// GetZoneName retrieves a zone name by ID
func (r *Repository) GetZoneName(ctx context.Context, zoneID uuid.UUID) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT name FROM pricing_zones WHERE id = $1`, zoneID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("failed to get zone name: %w", err)
	}
	return name, nil
}
