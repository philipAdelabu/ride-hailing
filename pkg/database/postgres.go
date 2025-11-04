package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/config"
)

// NewPostgresPool creates a new PostgreSQL connection pool
func NewPostgresPool(cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := cfg.DSN()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// Close closes the database connection pool
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
