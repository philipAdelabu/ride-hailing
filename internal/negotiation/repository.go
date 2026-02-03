package negotiation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for negotiation
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new negotiation repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateSession creates a new negotiation session
func (r *Repository) CreateSession(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO negotiation_sessions (
			id, rider_id,
			pickup_latitude, pickup_longitude, pickup_address,
			dropoff_latitude, dropoff_longitude, dropoff_address,
			country_id, region_id, city_id, pickup_zone_id, dropoff_zone_id,
			ride_type_id, currency_code, estimated_distance, estimated_duration, estimated_fare,
			fair_price_min, fair_price_max, system_suggested_price,
			rider_initial_offer, status, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
		RETURNING created_at
	`

	session.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		session.ID, session.RiderID,
		session.PickupLatitude, session.PickupLongitude, session.PickupAddress,
		session.DropoffLatitude, session.DropoffLongitude, session.DropoffAddress,
		session.CountryID, session.RegionID, session.CityID, session.PickupZoneID, session.DropoffZoneID,
		session.RideTypeID, session.CurrencyCode, session.EstimatedDistance, session.EstimatedDuration, session.EstimatedFare,
		session.FairPriceMin, session.FairPriceMax, session.SystemSuggestedPrice,
		session.RiderInitialOffer, session.Status, session.ExpiresAt,
	).Scan(&session.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByID retrieves a session by ID
func (r *Repository) GetSessionByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	query := `
		SELECT id, rider_id,
		       pickup_latitude, pickup_longitude, pickup_address,
		       dropoff_latitude, dropoff_longitude, dropoff_address,
		       country_id, region_id, city_id, pickup_zone_id, dropoff_zone_id,
		       ride_type_id, currency_code, estimated_distance, estimated_duration, estimated_fare,
		       fair_price_min, fair_price_max, system_suggested_price,
		       rider_initial_offer, status, accepted_offer_id, accepted_driver_id, accepted_price,
		       expires_at, created_at, accepted_at, completed_at, cancelled_at, cancellation_reason
		FROM negotiation_sessions
		WHERE id = $1
	`

	session := &Session{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.RiderID,
		&session.PickupLatitude, &session.PickupLongitude, &session.PickupAddress,
		&session.DropoffLatitude, &session.DropoffLongitude, &session.DropoffAddress,
		&session.CountryID, &session.RegionID, &session.CityID, &session.PickupZoneID, &session.DropoffZoneID,
		&session.RideTypeID, &session.CurrencyCode, &session.EstimatedDistance, &session.EstimatedDuration, &session.EstimatedFare,
		&session.FairPriceMin, &session.FairPriceMax, &session.SystemSuggestedPrice,
		&session.RiderInitialOffer, &session.Status, &session.AcceptedOfferID, &session.AcceptedDriverID, &session.AcceptedPrice,
		&session.ExpiresAt, &session.CreatedAt, &session.AcceptedAt, &session.CompletedAt, &session.CancelledAt, &session.CancellationReason,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetActiveSessionByRider retrieves an active session for a rider
func (r *Repository) GetActiveSessionByRider(ctx context.Context, riderID uuid.UUID) (*Session, error) {
	query := `
		SELECT id, rider_id,
		       pickup_latitude, pickup_longitude, pickup_address,
		       dropoff_latitude, dropoff_longitude, dropoff_address,
		       country_id, region_id, city_id, pickup_zone_id, dropoff_zone_id,
		       ride_type_id, currency_code, estimated_distance, estimated_duration, estimated_fare,
		       fair_price_min, fair_price_max, system_suggested_price,
		       rider_initial_offer, status, accepted_offer_id, accepted_driver_id, accepted_price,
		       expires_at, created_at, accepted_at, completed_at, cancelled_at, cancellation_reason
		FROM negotiation_sessions
		WHERE rider_id = $1 AND status = 'active' AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	session := &Session{}
	err := r.db.QueryRow(ctx, query, riderID).Scan(
		&session.ID, &session.RiderID,
		&session.PickupLatitude, &session.PickupLongitude, &session.PickupAddress,
		&session.DropoffLatitude, &session.DropoffLongitude, &session.DropoffAddress,
		&session.CountryID, &session.RegionID, &session.CityID, &session.PickupZoneID, &session.DropoffZoneID,
		&session.RideTypeID, &session.CurrencyCode, &session.EstimatedDistance, &session.EstimatedDuration, &session.EstimatedFare,
		&session.FairPriceMin, &session.FairPriceMax, &session.SystemSuggestedPrice,
		&session.RiderInitialOffer, &session.Status, &session.AcceptedOfferID, &session.AcceptedDriverID, &session.AcceptedPrice,
		&session.ExpiresAt, &session.CreatedAt, &session.AcceptedAt, &session.CompletedAt, &session.CancelledAt, &session.CancellationReason,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return session, nil
}

// UpdateSessionStatus updates a session's status
func (r *Repository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status SessionStatus) error {
	query := `
		UPDATE negotiation_sessions
		SET status = $1
		WHERE id = $2 AND status = 'active'
	`

	_, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	return nil
}

// AcceptOffer atomically accepts an offer and updates the session
func (r *Repository) AcceptOffer(ctx context.Context, sessionID, offerID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Get the offer
	var driverID uuid.UUID
	var offeredPrice float64
	err = tx.QueryRow(ctx, `
		SELECT driver_id, offered_price FROM negotiation_offers
		WHERE id = $1 AND session_id = $2 AND status = 'pending'
	`, offerID, sessionID).Scan(&driverID, &offeredPrice)
	if err != nil {
		return fmt.Errorf("offer not found or not pending: %w", err)
	}

	// Update the session (with status guard to prevent race conditions)
	tag, err := tx.Exec(ctx, `
		UPDATE negotiation_sessions
		SET status = 'accepted',
		    accepted_offer_id = $1,
		    accepted_driver_id = $2,
		    accepted_price = $3,
		    accepted_at = $4
		WHERE id = $5 AND status = 'active'
	`, offerID, driverID, offeredPrice, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("session is no longer active")
	}

	// Update the accepted offer
	_, err = tx.Exec(ctx, `
		UPDATE negotiation_offers
		SET status = 'accepted', accepted_at = $1
		WHERE id = $2
	`, now, offerID)
	if err != nil {
		return fmt.Errorf("failed to update offer: %w", err)
	}

	// Expire all other pending offers in this session
	_, err = tx.Exec(ctx, `
		UPDATE negotiation_offers
		SET status = 'expired', expired_at = $1
		WHERE session_id = $2 AND id != $3 AND status = 'pending'
	`, now, sessionID, offerID)
	if err != nil {
		return fmt.Errorf("failed to expire other offers: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateOffer creates a new offer
func (r *Repository) CreateOffer(ctx context.Context, offer *Offer) error {
	// First, supersede any pending offers from this driver
	_, err := r.db.Exec(ctx, `
		UPDATE negotiation_offers
		SET status = 'superseded'
		WHERE session_id = $1 AND driver_id = $2 AND status = 'pending'
	`, offer.SessionID, offer.DriverID)
	if err != nil {
		return fmt.Errorf("failed to supersede old offers: %w", err)
	}

	query := `
		INSERT INTO negotiation_offers (
			id, session_id, driver_id,
			offered_price, currency_code,
			driver_latitude, driver_longitude, estimated_pickup_time,
			driver_rating, driver_total_rides, vehicle_model, vehicle_color,
			status, is_counter_offer, parent_offer_id, counter_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING created_at
	`

	offer.ID = uuid.New()
	err = r.db.QueryRow(ctx, query,
		offer.ID, offer.SessionID, offer.DriverID,
		offer.OfferedPrice, offer.CurrencyCode,
		offer.DriverLatitude, offer.DriverLongitude, offer.EstimatedPickupTime,
		offer.DriverRating, offer.DriverTotalRides, offer.VehicleModel, offer.VehicleColor,
		offer.Status, offer.IsCounterOffer, offer.ParentOfferID, offer.CounterBy,
	).Scan(&offer.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	return nil
}

// GetOfferByID retrieves an offer by ID
func (r *Repository) GetOfferByID(ctx context.Context, id uuid.UUID) (*Offer, error) {
	query := `
		SELECT id, session_id, driver_id,
		       offered_price, currency_code,
		       driver_latitude, driver_longitude, estimated_pickup_time,
		       driver_rating, driver_total_rides, vehicle_model, vehicle_color,
		       status, is_counter_offer, parent_offer_id, counter_by,
		       created_at, accepted_at, rejected_at, expired_at
		FROM negotiation_offers
		WHERE id = $1
	`

	offer := &Offer{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&offer.ID, &offer.SessionID, &offer.DriverID,
		&offer.OfferedPrice, &offer.CurrencyCode,
		&offer.DriverLatitude, &offer.DriverLongitude, &offer.EstimatedPickupTime,
		&offer.DriverRating, &offer.DriverTotalRides, &offer.VehicleModel, &offer.VehicleColor,
		&offer.Status, &offer.IsCounterOffer, &offer.ParentOfferID, &offer.CounterBy,
		&offer.CreatedAt, &offer.AcceptedAt, &offer.RejectedAt, &offer.ExpiredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get offer: %w", err)
	}

	return offer, nil
}

// GetOffersBySession retrieves all offers for a session
func (r *Repository) GetOffersBySession(ctx context.Context, sessionID uuid.UUID) ([]*Offer, error) {
	query := `
		SELECT id, session_id, driver_id,
		       offered_price, currency_code,
		       driver_latitude, driver_longitude, estimated_pickup_time,
		       driver_rating, driver_total_rides, vehicle_model, vehicle_color,
		       status, is_counter_offer, parent_offer_id, counter_by,
		       created_at, accepted_at, rejected_at, expired_at
		FROM negotiation_offers
		WHERE session_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get offers: %w", err)
	}
	defer rows.Close()

	offers := make([]*Offer, 0)
	for rows.Next() {
		offer := &Offer{}
		err := rows.Scan(
			&offer.ID, &offer.SessionID, &offer.DriverID,
			&offer.OfferedPrice, &offer.CurrencyCode,
			&offer.DriverLatitude, &offer.DriverLongitude, &offer.EstimatedPickupTime,
			&offer.DriverRating, &offer.DriverTotalRides, &offer.VehicleModel, &offer.VehicleColor,
			&offer.Status, &offer.IsCounterOffer, &offer.ParentOfferID, &offer.CounterBy,
			&offer.CreatedAt, &offer.AcceptedAt, &offer.RejectedAt, &offer.ExpiredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan offer: %w", err)
		}
		offers = append(offers, offer)
	}

	return offers, nil
}

// GetSettings retrieves negotiation settings for a location
func (r *Repository) GetSettings(ctx context.Context, countryID, regionID, cityID *uuid.UUID) (*Settings, error) {
	query := `
		SELECT id, country_id, region_id, city_id,
		       negotiation_enabled, session_timeout_seconds, max_offers_per_session,
		       max_counter_offers, offer_timeout_seconds, min_price_multiplier,
		       max_price_multiplier, max_active_sessions_per_driver,
		       min_driver_rating_to_negotiate, min_driver_rides_to_negotiate,
		       block_drivers_with_high_price_streak, price_deviation_threshold,
		       created_at, updated_at
		FROM negotiation_settings
		WHERE (city_id = $1 OR (city_id IS NULL AND region_id = $2) OR (city_id IS NULL AND region_id IS NULL AND country_id = $3) OR (city_id IS NULL AND region_id IS NULL AND country_id IS NULL))
		ORDER BY
		  CASE
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC
		LIMIT 1
	`

	settings := &Settings{}
	err := r.db.QueryRow(ctx, query, cityID, regionID, countryID).Scan(
		&settings.ID, &settings.CountryID, &settings.RegionID, &settings.CityID,
		&settings.NegotiationEnabled, &settings.SessionTimeoutSeconds, &settings.MaxOffersPerSession,
		&settings.MaxCounterOffers, &settings.OfferTimeoutSeconds, &settings.MinPriceMultiplier,
		&settings.MaxPriceMultiplier, &settings.MaxActiveSessionsPerDriver,
		&settings.MinDriverRatingToNegotiate, &settings.MinDriverRidesToNegotiate,
		&settings.BlockDriversWithHighPriceStreak, &settings.PriceDeviationThreshold,
		&settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &DefaultSettings, nil
		}
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

// CountDriverActiveOffers counts active offers by a driver
func (r *Repository) CountDriverActiveOffers(ctx context.Context, driverID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(DISTINCT o.session_id)
		FROM negotiation_offers o
		JOIN negotiation_sessions s ON o.session_id = s.id
		WHERE o.driver_id = $1 AND o.status = 'pending' AND s.status = 'active'
	`

	var count int
	err := r.db.QueryRow(ctx, query, driverID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active offers: %w", err)
	}

	return count, nil
}

// ExpireSession marks a session as expired
func (r *Repository) ExpireSession(ctx context.Context, sessionID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Update session
	_, err = tx.Exec(ctx, `
		UPDATE negotiation_sessions
		SET status = 'expired'
		WHERE id = $1 AND status = 'active'
	`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to expire session: %w", err)
	}

	// Expire all pending offers
	_, err = tx.Exec(ctx, `
		UPDATE negotiation_offers
		SET status = 'expired', expired_at = $1
		WHERE session_id = $2 AND status = 'pending'
	`, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to expire offers: %w", err)
	}

	return tx.Commit(ctx)
}

// GetExpiredSessions retrieves sessions that have expired but not yet marked
func (r *Repository) GetExpiredSessions(ctx context.Context) ([]*Session, error) {
	query := `
		SELECT id, rider_id, status, expires_at, created_at
		FROM negotiation_sessions
		WHERE status = 'active' AND expires_at <= NOW()
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(&session.ID, &session.RiderID, &session.Status, &session.ExpiresAt, &session.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
