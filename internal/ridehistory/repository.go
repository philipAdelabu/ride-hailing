package ridehistory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Shared column list for ride history queries
const rideHistoryColumns = `
	r.id, r.rider_id, r.driver_id, r.status,
	r.pickup_address, r.pickup_latitude, r.pickup_longitude,
	r.dropoff_address, r.dropoff_latitude, r.dropoff_longitude,
	COALESCE(r.actual_distance, r.estimated_distance, 0),
	COALESCE(r.actual_duration, r.estimated_duration, 0),
	COALESCE(r.estimated_fare, 0),
	r.final_fare,
	COALESCE(r.surge_multiplier, 1),
	COALESCE(r.discount_amount, 0),
	COALESCE(r.currency_code, 'USD'),
	r.rating,
	r.feedback,
	r.cancellation_reason,
	r.requested_at, r.accepted_at, r.started_at, r.completed_at, r.cancelled_at`

// scanRideHistoryEntry scans a row into a RideHistoryEntry
func scanRideHistoryEntry(scan func(dest ...interface{}) error) (RideHistoryEntry, error) {
	e := RideHistoryEntry{}
	err := scan(
		&e.ID, &e.RiderID, &e.DriverID, &e.Status,
		&e.PickupAddress, &e.PickupLatitude, &e.PickupLongitude,
		&e.DropoffAddress, &e.DropoffLatitude, &e.DropoffLongitude,
		&e.Distance, &e.Duration,
		&e.EstimatedFare, &e.FinalFare,
		&e.SurgeMultiplier, &e.DiscountAmount,
		&e.Currency,
		&e.Rating, &e.Feedback,
		&e.CancellationReason,
		&e.RequestedAt, &e.AcceptedAt, &e.StartedAt, &e.CompletedAt, &e.CancelledAt,
	)
	return e, err
}

// Repository handles ride history data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ride history repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// buildFilters constructs WHERE clauses and args from filters
func buildFilters(baseWhere string, baseArg interface{}, filters *HistoryFilters) (string, []interface{}, int) {
	where := []string{baseWhere}
	args := []interface{}{baseArg}
	argIdx := 2

	if filters != nil {
		if filters.Status != nil {
			where = append(where, fmt.Sprintf("r.status = $%d", argIdx))
			args = append(args, *filters.Status)
			argIdx++
		}
		if filters.FromDate != nil {
			where = append(where, fmt.Sprintf("r.requested_at >= $%d", argIdx))
			args = append(args, *filters.FromDate)
			argIdx++
		}
		if filters.ToDate != nil {
			where = append(where, fmt.Sprintf("r.requested_at < $%d", argIdx))
			args = append(args, *filters.ToDate)
			argIdx++
		}
		if filters.MinFare != nil {
			where = append(where, fmt.Sprintf("COALESCE(r.final_fare, r.estimated_fare) >= $%d", argIdx))
			args = append(args, *filters.MinFare)
			argIdx++
		}
		if filters.MaxFare != nil {
			where = append(where, fmt.Sprintf("COALESCE(r.final_fare, r.estimated_fare) <= $%d", argIdx))
			args = append(args, *filters.MaxFare)
			argIdx++
		}
	}

	return strings.Join(where, " AND "), args, argIdx
}

// queryHistory runs the paginated history query
func (r *Repository) queryHistory(ctx context.Context, whereClause string, args []interface{}, argIdx int, limit, offset int) ([]RideHistoryEntry, int, error) {
	// Count
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM rides r WHERE %s`, whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query
	query := fmt.Sprintf(`SELECT %s FROM rides r WHERE %s ORDER BY r.requested_at DESC LIMIT $%d OFFSET $%d`,
		rideHistoryColumns, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rides []RideHistoryEntry
	for rows.Next() {
		e, err := scanRideHistoryEntry(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		rides = append(rides, e)
	}
	return rides, total, nil
}

// GetRiderHistory returns paginated ride history for a rider
func (r *Repository) GetRiderHistory(ctx context.Context, riderID uuid.UUID, filters *HistoryFilters, limit, offset int) ([]RideHistoryEntry, int, error) {
	whereClause, args, argIdx := buildFilters("r.rider_id = $1", riderID, filters)
	return r.queryHistory(ctx, whereClause, args, argIdx, limit, offset)
}

// GetDriverHistory returns paginated ride history for a driver
func (r *Repository) GetDriverHistory(ctx context.Context, driverID uuid.UUID, filters *HistoryFilters, limit, offset int) ([]RideHistoryEntry, int, error) {
	whereClause, args, argIdx := buildFilters("r.driver_id = $1", driverID, filters)
	return r.queryHistory(ctx, whereClause, args, argIdx, limit, offset)
}

// GetRideByID returns a single ride by ID
func (r *Repository) GetRideByID(ctx context.Context, rideID uuid.UUID) (*RideHistoryEntry, error) {
	query := fmt.Sprintf(`SELECT %s FROM rides r WHERE r.id = $1`, rideHistoryColumns)
	e, err := scanRideHistoryEntry(func(dest ...interface{}) error {
		return r.db.QueryRow(ctx, query, rideID).Scan(dest...)
	})
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetRiderStats returns aggregated stats for a rider
func (r *Repository) GetRiderStats(ctx context.Context, riderID uuid.UUID, from, to time.Time) (*RideStats, error) {
	stats := &RideStats{Currency: "USD"}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(CASE WHEN status = 'completed' THEN 1 END),
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN COALESCE(final_fare, estimated_fare) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN COALESCE(actual_distance, estimated_distance) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN COALESCE(actual_duration, estimated_duration) ELSE 0 END), 0),
			COALESCE(AVG(CASE WHEN status = 'completed' THEN COALESCE(final_fare, estimated_fare) END), 0),
			COALESCE(AVG(CASE WHEN status = 'completed' THEN COALESCE(actual_distance, estimated_distance) END), 0),
			COALESCE(AVG(rating), 0)
		FROM rides
		WHERE rider_id = $1 AND requested_at >= $2 AND requested_at < $3`,
		riderID, from, to,
	).Scan(
		&stats.TotalRides, &stats.CompletedRides, &stats.CancelledRides,
		&stats.TotalSpent, &stats.TotalDistance, &stats.TotalDuration,
		&stats.AverageFare, &stats.AverageDistance, &stats.AverageRating,
	)
	return stats, err
}

// GetFrequentRoutes returns commonly taken routes
func (r *Repository) GetFrequentRoutes(ctx context.Context, riderID uuid.UUID, limit int) ([]FrequentRoute, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pickup_address, dropoff_address,
			COUNT(*) as ride_count,
			AVG(COALESCE(final_fare, estimated_fare)) as avg_fare,
			MAX(requested_at)::text as last_ride
		FROM rides
		WHERE rider_id = $1 AND status = 'completed'
		GROUP BY pickup_address, dropoff_address
		HAVING COUNT(*) >= 2
		ORDER BY ride_count DESC
		LIMIT $2`, riderID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []FrequentRoute
	for rows.Next() {
		fr := FrequentRoute{}
		if err := rows.Scan(
			&fr.PickupAddress, &fr.DropoffAddress,
			&fr.RideCount, &fr.AverageFare, &fr.LastRideAt,
		); err != nil {
			return nil, err
		}
		routes = append(routes, fr)
	}
	return routes, nil
}
