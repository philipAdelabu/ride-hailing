package scheduler

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Database defines the database operations required by the scheduler worker.
// This interface wraps the pgxpool.Pool methods used by the worker,
// allowing for easier testing through mock implementations.
type Database interface {
	// Query executes a query that returns rows, typically a SELECT.
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)

	// Exec executes a query that doesn't return rows, typically INSERT, UPDATE, DELETE.
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}
