package ridetypes

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for ride types
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ride types repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateRideType creates a new ride type
func (r *Repository) CreateRideType(ctx context.Context, rt *RideType) error {
	query := `
		INSERT INTO ride_types (id, name, description, base_fare, per_km_rate,
		       per_minute_rate, minimum_fare, capacity, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`
	rt.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		rt.ID, rt.Name, rt.Description, rt.BaseFare, rt.PerKmRate,
		rt.PerMinuteRate, rt.MinimumFare, rt.Capacity, rt.IsActive,
	).Scan(&rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create ride type: %w", err)
	}
	return nil
}

// GetRideTypeByID retrieves a ride type by ID
func (r *Repository) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	query := `
		SELECT id, name, description, base_fare, per_km_rate, per_minute_rate,
		       minimum_fare, capacity, is_active, created_at, updated_at
		FROM ride_types WHERE id = $1
	`
	rt := &RideType{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rt.ID, &rt.Name, &rt.Description, &rt.BaseFare, &rt.PerKmRate,
		&rt.PerMinuteRate, &rt.MinimumFare, &rt.Capacity, &rt.IsActive,
		&rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride type: %w", err)
	}
	return rt, nil
}

// ListRideTypes lists ride types with pagination
func (r *Repository) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	whereClause := ""
	if !includeInactive {
		whereClause = "WHERE is_active = true"
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ride_types %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count ride types: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, base_fare, per_km_rate, per_minute_rate,
		       minimum_fare, capacity, is_active, created_at, updated_at
		FROM ride_types %s
		ORDER BY name
		LIMIT $1 OFFSET $2
	`, whereClause)

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list ride types: %w", err)
	}
	defer rows.Close()

	items := make([]*RideType, 0)
	for rows.Next() {
		rt := &RideType{}
		err := rows.Scan(
			&rt.ID, &rt.Name, &rt.Description, &rt.BaseFare, &rt.PerKmRate,
			&rt.PerMinuteRate, &rt.MinimumFare, &rt.Capacity, &rt.IsActive,
			&rt.CreatedAt, &rt.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ride type: %w", err)
		}
		items = append(items, rt)
	}
	return items, total, nil
}

// UpdateRideType updates a ride type
func (r *Repository) UpdateRideType(ctx context.Context, rt *RideType) error {
	query := `
		UPDATE ride_types SET
			name = $2, description = $3, base_fare = $4, per_km_rate = $5,
			per_minute_rate = $6, minimum_fare = $7, capacity = $8,
			is_active = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		rt.ID, rt.Name, rt.Description, rt.BaseFare, rt.PerKmRate,
		rt.PerMinuteRate, rt.MinimumFare, rt.Capacity, rt.IsActive,
	).Scan(&rt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update ride type: %w", err)
	}
	return nil
}

// DeleteRideType soft-deletes a ride type
func (r *Repository) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE ride_types SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete ride type: %w", err)
	}
	return nil
}
