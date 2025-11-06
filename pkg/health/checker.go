package health

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"
)

// DatabaseChecker returns a health check function for PostgreSQL database
func DatabaseChecker(db *sql.DB) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.PingContext(ctx)
	}
}

// RedisChecker returns a health check function for Redis
func RedisChecker(client *redis.Client) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return client.Ping(ctx).Err()
	}
}

// HTTPEndpointChecker returns a health check function for HTTP endpoints
// Useful for checking external service dependencies
func HTTPEndpointChecker(url string) func() error {
	return func() error {
		// This is a placeholder - implement if needed
		// You can use http.Get with a timeout to check external services
		return nil
	}
}
