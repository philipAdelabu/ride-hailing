# ADR-0001: Microservices Architecture

## Status

Accepted

## Context

As the ride-hailing platform grew in complexity, the monolithic architecture presented several challenges:

1. **Scaling bottlenecks**: Different features have vastly different scaling requirements. Driver location updates require real-time processing at high frequency, while analytics can tolerate batch processing.

2. **Team ownership**: Multiple teams working on the same codebase led to merge conflicts, deployment coordination overhead, and difficulty in establishing clear ownership boundaries.

3. **Technology constraints**: A monolith forces uniform technology choices. Some domains (ML-based ETA prediction, real-time geo operations) benefit from specialized optimizations.

4. **Deployment risk**: A single bug in one feature could bring down the entire platform during deployment.

5. **Independent development velocity**: Teams were blocked waiting for other features to be ready for deployment.

## Decision

Split the platform into 14 independent microservices, each with clear domain boundaries:

| Service | Port | Domain |
|---------|------|--------|
| auth | 8081 | User authentication, JWT management |
| rides | 8082 | Ride lifecycle, booking, matching |
| geo | 8083 | Driver locations, H3 indexing, geocoding |
| payments | 8084 | Payment processing, Stripe integration |
| notifications | 8085 | Push, SMS, email notifications |
| analytics | 8086 | Business metrics, reporting |
| admin | 8087 | Admin dashboard operations |
| promos | 8089 | Promotions, discounts, coupons |
| fraud | 8090 | Fraud detection, risk scoring |
| realtime | 8091 | WebSocket connections, live updates |
| mobile | 8092 | Mobile BFF (Backend for Frontend) |
| scheduler | 8093 | Scheduled rides, cron jobs |
| ml-eta | 8094 | ML-based ETA predictions |
| negotiation | 8095 | Price negotiation workflows |

Each service follows the same structural pattern:

```
cmd/<service>/main.go      # Entry point with server setup
internal/<service>/        # Domain logic (handler, service, repository)
```

Example from `cmd/auth/main.go`:

```go
const (
    serviceName = "auth-service"
    version     = "1.0.0"
)

func main() {
    cfg, err := config.Load(serviceName)
    // ... service initialization
    repo := auth.NewRepository(db)
    service := auth.NewService(repo, keyManager, cfg.JWT.Expiration)
    handler := auth.NewHandler(service)
    handler.RegisterRoutes(router, keyManager)
}
```

## Consequences

### Positive

- **Independent scaling**: Geo service scales horizontally during peak hours while auth service remains at baseline.
- **Team autonomy**: Teams own services end-to-end, from development to production operations.
- **Fault isolation**: A crash in the notifications service does not affect ride booking.
- **Technology flexibility**: ML-ETA service can use specialized ML libraries without affecting other services.
- **Faster deployments**: Services deploy independently, reducing blast radius.

### Negative

- **Operational complexity**: 14 services require monitoring, logging, and alerting infrastructure.
- **Network overhead**: Inter-service communication adds latency compared to in-process calls.
- **Data consistency**: Distributed transactions require saga patterns or eventual consistency.
- **Local development**: Running all services locally requires Docker Compose orchestration.

### Mitigations

- **Circuit breakers**: Each service uses resilience patterns for external calls (see `pkg/resilience/circuit_breaker.go`).
- **Service mesh ready**: Services expose health endpoints (`/healthz`, `/health/live`, `/health/ready`) for Kubernetes.
- **Centralized logging**: All services use structured logging with correlation IDs.
- **Docker Compose**: `docker-compose.yml` orchestrates all 14 services for local development.

## References

- [cmd/auth/main.go](/cmd/auth/main.go) - Auth service entry point
- [cmd/rides/main.go](/cmd/rides/main.go) - Rides service with circuit breakers
- [docker-compose.yml](/docker-compose.yml) - Service orchestration
- [pkg/resilience/circuit_breaker.go](/pkg/resilience/circuit_breaker.go) - Inter-service resilience
