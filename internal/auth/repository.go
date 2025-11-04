package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Repository handles database operations for authentication
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new auth repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.PhoneNumber,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Role,
		user.IsActive,
		user.IsVerified,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Create wallet for the user
	walletQuery := `INSERT INTO wallets (user_id) VALUES ($1)`
	_, err = r.db.Exec(ctx, walletQuery, user.ID)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

// CreateDriver creates driver-specific information
func (r *Repository) CreateDriver(ctx context.Context, driver *models.Driver) error {
	query := `
		INSERT INTO drivers (
			id, user_id, license_number, vehicle_model, vehicle_plate,
			vehicle_color, vehicle_year, is_available, is_online, rating, total_rides
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		driver.ID,
		driver.UserID,
		driver.LicenseNumber,
		driver.VehicleModel,
		driver.VehiclePlate,
		driver.VehicleColor,
		driver.VehicleYear,
		driver.IsAvailable,
		driver.IsOnline,
		driver.Rating,
		driver.TotalRides,
	).Scan(&driver.CreatedAt, &driver.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, phone_number, password_hash, first_name, last_name,
			   role, is_active, is_verified, profile_image, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.IsVerified,
		&user.ProfileImage,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, phone_number, password_hash, first_name, last_name,
			   role, is_active, is_verified, profile_image, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.IsVerified,
		&user.ProfileImage,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (r *Repository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, phone_number = $3, profile_image = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.Exec(ctx, query,
		user.FirstName,
		user.LastName,
		user.PhoneNumber,
		user.ProfileImage,
		time.Now(),
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
