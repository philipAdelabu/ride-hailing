# ADR-0004: Redis Caching Strategy

## Status

Accepted

## Context

The ride-hailing platform has several high-frequency, low-latency requirements:

1. **Driver location tracking**: Thousands of drivers update locations every 3-5 seconds.
2. **Nearby driver search**: Riders need instant results for available drivers within radius.
3. **Session management**: JWT refresh tokens and user sessions require fast access.
4. **Rate limiting**: API rate limits must be enforced with minimal latency overhead.
5. **Feature flags**: Runtime configuration for A/B testing and gradual rollouts.
6. **ETA predictions**: ML predictions cached to avoid redundant computation.

PostgreSQL alone cannot meet these performance requirements without significant infrastructure investment.

## Decision

Use **Redis** as the primary caching layer with specific TTL strategies per use case.

### Redis Client Configuration

```go
// From pkg/redis/redis.go
client := redis.NewClient(&redis.Options{
    Addr:     cfg.RedisAddr(),
    Password: cfg.Password,
    DB:       cfg.DB,

    // Timeout configuration
    DialTimeout:  5 * time.Second,
    ReadTimeout:  readTimeout,
    WriteTimeout: writeTimeout,

    // Connection pool
    PoolSize:     10,
    MinIdleConns: 5,
    PoolTimeout:  readTimeout + 1*time.Second,

    // Connection lifecycle
    ConnMaxIdleTime: 5 * time.Minute,
    ConnMaxLifetime: 1 * time.Hour,

    // Retry configuration
    MaxRetries:      3,
    MinRetryBackoff: 8 * time.Millisecond,
    MaxRetryBackoff: 512 * time.Millisecond,
})
```

### TTL Strategy by Use Case

| Use Case | TTL | Rationale |
|----------|-----|-----------|
| Feature flags | 30 seconds | Fast propagation of config changes |
| Rate limit counters | 1 minute | Per-minute rate limits |
| User sessions | 24 hours | Match JWT expiration |
| ETA predictions | 24 hours | Historical routes change slowly |
| Driver locations | 30 seconds | Stale locations become irrelevant |
| Idempotency keys | 5 minutes | Retry window for network failures |
| Geocoding results | 7 days | Addresses rarely change |

### Geo-Spatial Operations

Redis GEO commands power driver discovery:

```go
// From pkg/redis/redis.go

// GeoAdd adds a driver location to geospatial index
func (c *Client) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
    return c.Client.GeoAdd(ctx, key, &redis.GeoLocation{
        Longitude: longitude,
        Latitude:  latitude,
        Name:      member,
    }).Err()
}

// GeoRadius finds drivers within radius, sorted by distance
func (c *Client) GeoRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64, count int) ([]string, error) {
    result, err := c.Client.GeoRadius(ctx, key, longitude, latitude, &redis.GeoRadiusQuery{
        Radius:   radiusKm,
        Unit:     "km",
        WithDist: true,
        Count:    count,
        Sort:     "ASC", // Nearest first
    }).Result()
    // ...
}
```

### Cache Wrapper Pattern

```go
// From pkg/redis/redis.go
func (c *Client) Cache(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
    // Try cache first
    result, err := c.GetString(ctx, key)
    if err == nil {
        return result, nil
    }

    // Execute function on cache miss
    data, err := fn()
    if err != nil {
        return nil, err
    }

    // Store for future use
    if err := c.SetWithExpiration(ctx, key, data, ttl); err != nil {
        fmt.Printf("failed to cache key %s: %v\n", key, err)
    }

    return data, nil
}
```

### ETA Prediction Caching

```go
// From internal/mleta/service.go
func (s *Service) getHistoricalETA(ctx context.Context, pickupLat, pickupLng, dropoffLat, dropoffLng float64) (float64, error) {
    // Route key with rounded coordinates
    routeKey := fmt.Sprintf("eta:route:%.2f,%.2f:%.2f,%.2f",
        math.Round(pickupLat*100)/100,
        math.Round(pickupLng*100)/100,
        math.Round(dropoffLat*100)/100,
        math.Round(dropoffLng*100)/100)

    // Check Redis cache
    cachedETA, err := s.redis.GetString(ctx, routeKey)
    if err == nil && cachedETA != "" {
        var eta float64
        if err := json.Unmarshal([]byte(cachedETA), &eta); err == nil {
            return eta, nil
        }
    }

    // Fallback to database
    eta, err := s.repo.GetHistoricalETAForRoute(ctx, pickupLat, pickupLng, dropoffLat, dropoffLng)
    if err != nil {
        return 0, err
    }

    // Cache for 24 hours
    if etaBytes, err := json.Marshal(eta); err == nil {
        _ = s.redis.SetWithExpiration(ctx, routeKey, string(etaBytes), 24*time.Hour)
    }

    return eta, nil
}
```

## Consequences

### Positive

- **Sub-millisecond latency**: Redis operations complete in < 1ms.
- **Geo-native operations**: GEORADIUS eliminates complex SQL for proximity queries.
- **Horizontal scaling**: Redis Cluster supports sharding for growth.
- **Built-in expiration**: TTL prevents stale data accumulation.
- **Atomic operations**: Rate limiting uses INCR for thread-safe counters.

### Negative

- **Infrastructure complexity**: Redis requires separate operational expertise.
- **Eventual consistency**: Cache invalidation requires careful coordination.
- **Memory costs**: High-cardinality data (all driver locations) needs capacity planning.
- **Failure modes**: Redis unavailability requires graceful degradation.

### Graceful Degradation

Services continue operating without Redis (with reduced functionality):

```go
// From cmd/rides/main.go
redisClient, err = redisclient.NewRedisClient(&cfg.Redis)
if err != nil {
    logger.Warn("Failed to initialize Redis - idempotency and rate limiting disabled", zap.Error(err))
} else {
    if cfg.RateLimit.Enabled {
        limiter = ratelimit.NewLimiter(redisClient.Client, cfg.RateLimit)
    }
}
```

## References

- [pkg/redis/redis.go](/pkg/redis/redis.go) - Redis client with geo operations
- [pkg/redis/interface.go](/pkg/redis/interface.go) - Redis interface for testing
- [internal/mleta/service.go](/internal/mleta/service.go) - ETA caching example
- [cmd/rides/main.go](/cmd/rides/main.go) - Graceful Redis degradation
- [pkg/ratelimit/limiter.go](/pkg/ratelimit/limiter.go) - Redis-backed rate limiting
