# ADR-0002: Gin Web Framework Choice

## Status

Accepted

## Context

The ride-hailing platform requires a high-performance HTTP framework capable of handling:

1. **High throughput**: Driver location updates occur every 3-5 seconds from thousands of concurrent drivers.
2. **Low latency**: Ride booking and driver matching must respond within milliseconds.
3. **Middleware ecosystem**: Cross-cutting concerns like authentication, rate limiting, logging, and tracing need composable middleware.
4. **Production readiness**: The framework must be battle-tested with a stable API.
5. **Developer experience**: Clear documentation and familiar patterns reduce onboarding time.

### Alternatives Considered

| Framework | Pros | Cons |
|-----------|------|------|
| **Echo** | Fast, good middleware | Smaller ecosystem than Gin |
| **Fiber** | Fastest benchmarks, Express-like API | Uses fasthttp (not net/http compatible) |
| **Chi** | Lightweight, stdlib compatible | Less feature-rich, more manual setup |
| **Gin** | High performance, rich middleware, large ecosystem | Slightly more opinionated |

## Decision

Use **Gin** as the web framework for all microservices. Key factors:

1. **Performance**: Gin uses httprouter, one of the fastest Go routers with zero memory allocation in most cases.

2. **Middleware chain**: Gin's middleware pattern enables clean separation of concerns.

3. **JSON handling**: Built-in binding and validation with struct tags.

4. **Production proven**: Used by major companies with extensive community support.

### Implementation Pattern

Every service follows the same Gin setup pattern:

```go
// From cmd/auth/main.go
router := gin.New()
router.Use(middleware.RecoveryWithSentry()) // Panic recovery
router.Use(middleware.SentryMiddleware())   // Error tracking
router.Use(middleware.CorrelationID())      // Request tracing
router.Use(middleware.RequestTimeout(&cfg.Timeout))
router.Use(middleware.RequestLogger(serviceName))
router.Use(middleware.CORS())
router.Use(middleware.SecurityHeaders())
router.Use(middleware.MaxBodySize(10 << 20)) // 10MB limit
router.Use(middleware.SanitizeRequest())
router.Use(middleware.Metrics(serviceName))
```

### Custom Middleware Stack

The platform implements 15+ custom middleware in `pkg/middleware/`:

| Middleware | Purpose |
|------------|---------|
| `CorrelationID()` | Generate/propagate request IDs for distributed tracing |
| `RequestLogger()` | Structured logging with zap |
| `AuthMiddleware()` | JWT validation with key rotation support |
| `RateLimiter()` | Redis-backed rate limiting |
| `Metrics()` | Prometheus metrics collection |
| `TracingMiddleware()` | OpenTelemetry span propagation |
| `RecoveryWithSentry()` | Panic recovery with Sentry reporting |
| `SecurityHeaders()` | Security headers (CSP, HSTS, etc.) |

### Route Registration Pattern

Handlers register routes using Gin's group pattern:

```go
// From internal/geo/handler.go
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
    api := r.Group("/api/v1")
    api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))

    geo := api.Group("/geo")
    {
        geo.POST("/location", middleware.RequireRole(models.RoleDriver), h.UpdateLocation)
        geo.GET("/drivers/:id/location", h.GetDriverLocation)
        geo.GET("/drivers/nearby", h.FindNearbyDrivers)
    }
}
```

## Consequences

### Positive

- **Consistent patterns**: All 14 services use identical router setup, reducing cognitive overhead.
- **Rich middleware**: Cross-cutting concerns are cleanly separated and reusable.
- **Fast JSON handling**: Gin's binding reduces boilerplate for request validation.
- **Easy testing**: `gin.SetMode(gin.TestMode)` enables clean test setup.
- **Prometheus integration**: `gin.WrapH(promhttp.Handler())` integrates seamlessly.

### Negative

- **Framework coupling**: Switching frameworks would require significant refactoring.
- **Learning curve**: Gin's context differs from standard `context.Context`.
- **Hidden magic**: Some features (like binding) hide complexity that can surprise developers.

### Performance Characteristics

Based on internal benchmarks:
- **Latency p99**: < 5ms for simple endpoints
- **Throughput**: 50k+ requests/second per instance
- **Memory**: Minimal allocations due to sync.Pool usage

## References

- [cmd/auth/main.go](/cmd/auth/main.go) - Standard Gin router setup
- [pkg/middleware/](/pkg/middleware/) - Custom middleware implementations
- [internal/geo/handler.go](/internal/geo/handler.go) - Route registration example
- [Gin Framework Documentation](https://gin-gonic.com/docs/)
