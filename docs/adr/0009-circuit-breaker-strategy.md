# ADR-0009: Circuit Breaker Strategy

## Status

Accepted

## Context

The platform depends on multiple external services (Promos, ML-ETA, Notifications, Stripe, Maps). When these become slow or unavailable, cascading failures can bring down the entire system. A single degraded dependency should not block ride bookings.

## Decision

Implement the **circuit breaker pattern** for all external calls using `pkg/resilience/circuit_breaker.go`, built on [sony/gobreaker](https://github.com/sony/gobreaker).

### Three States

- **Closed**: Requests flow normally; failures are counted
- **Open**: Requests fail immediately without calling upstream
- **Half-Open**: Limited requests test if service has recovered

### Implementation

```go
// From pkg/resilience/circuit_breaker.go
type CircuitBreaker struct {
    breaker  *gobreaker.CircuitBreaker
    fallback FallbackFunc
}

func (c *CircuitBreaker) Execute(ctx context.Context, op Operation) (interface{}, error) {
    result, err := c.breaker.Execute(func() (interface{}, error) {
        return op(ctx)
    })
    if errors.Is(err, gobreaker.ErrOpenState) {
        if c.fallback != nil {
            return c.fallback(ctx, err)
        }
        return nil, ErrCircuitOpen
    }
    return result, err
}
```

### Configuration

```go
// From pkg/config/config.go
type CircuitBreakerConfig struct {
    FailureThreshold int  // Default: 5 consecutive failures
    SuccessThreshold int  // Default: 1 success to close
    TimeoutSeconds   int  // Default: 30s in open state
}
```

## Services Protected

| Service | Failure Threshold | Timeout | Fallback |
|---------|-------------------|---------|----------|
| Promos | 5 | 30s | Skip discount, use base fare |
| ML-ETA | 3 | 15s | Distance-based estimation |
| Notifications | 10 | 60s | Queue for retry |
| Stripe | 3 | 30s | Return pending status |

### Usage in Rides Service

```go
// From internal/rides/service.go
func (s *Service) postToPromos(ctx context.Context, path string, body interface{}) ([]byte, error) {
    return s.promosBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
        return s.promosClient.Post(ctx, path, body, nil)
    })
}
```

## Fallback Strategies

```go
// From pkg/resilience/fallback.go
func StaticFallback(defaultValue interface{}) FallbackFunc {
    return func(ctx context.Context, err error) (interface{}, error) {
        return defaultValue, nil  // e.g., surgeMultiplier = 1.0
    }
}
```

## Monitoring

Circuit state exposed via Prometheus:

```go
// From pkg/resilience/metrics.go
breakerStateGauge     // 0=closed, 0.5=half-open, 1=open
breakerRequestsTotal  // Total operations through breaker
breakerFailuresTotal  // Operations that failed
breakerFallbacksTotal // Fallbacks triggered
```

## Consequences

### Positive

- **Fail fast**: Open circuits prevent wasted time on unavailable services.
- **Cascading prevention**: Isolated failures do not propagate system-wide.
- **Automatic recovery**: Half-open state tests health without manual intervention.
- **Observable**: Prometheus metrics enable alerting on state changes.

### Negative

- **False positives**: Transient issues may trip breakers unnecessarily.
- **Fallback complexity**: Each service requires sensible degraded behavior.
- **Thundering herd**: Instances may trip and probe simultaneously.

## References

- [pkg/resilience/circuit_breaker.go](/pkg/resilience/circuit_breaker.go) - Implementation
- [pkg/resilience/fallback.go](/pkg/resilience/fallback.go) - Fallback functions
- [pkg/resilience/metrics.go](/pkg/resilience/metrics.go) - Prometheus metrics
- [internal/rides/service.go](/internal/rides/service.go) - Usage example
