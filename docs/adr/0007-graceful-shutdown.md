# ADR-0007: Graceful Shutdown Pattern

## Status

Accepted

## Context

The ride-hailing platform runs on Kubernetes with rolling deployments. Challenges during shutdown:

1. **In-flight requests**: Active HTTP requests must complete before termination.
2. **Database connections**: Open transactions must commit or rollback cleanly.
3. **Redis operations**: Cached data writes should complete.
4. **External integrations**: Stripe webhooks and Twilio callbacks must finish processing.
5. **WebSocket connections**: Real-time connections need graceful closure.
6. **Kubernetes compatibility**: Services must respond to SIGTERM within termination grace period.

Without graceful shutdown, users experience:
- Failed ride bookings mid-transaction
- Duplicate payment charges from incomplete idempotency
- Lost driver location updates

## Decision

Implement **signal-based graceful shutdown** with a **5-second timeout** across all services.

### Shutdown Pattern

```go
// From cmd/auth/main.go

// Create HTTP server
srv := &http.Server{
    Addr:         ":" + cfg.Server.Port,
    Handler:      router,
    ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
    WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
}

// Start server in goroutine
go func() {
    logger.Info("Server starting", zap.String("port", cfg.Server.Port))
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatal("Failed to start server", zap.Error(err))
    }
}()

// Wait for interrupt signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info("Shutting down server...")

// Graceful shutdown with 5-second timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    logger.Fatal("Server forced to shutdown", zap.Error(err))
}

logger.Info("Server stopped")
```

### Shutdown Sequence

1. **Signal received**: SIGTERM (Kubernetes) or SIGINT (Ctrl+C)
2. **Stop accepting**: `srv.Shutdown()` stops accepting new connections
3. **Drain requests**: In-flight requests have 5 seconds to complete
4. **Close dependencies**: Deferred cleanup runs in LIFO order
5. **Exit**: Process terminates with exit code 0

### Dependency Cleanup Order

```go
// From cmd/rides/main.go

func main() {
    // Resources are cleaned up in reverse order via defer

    cfg, err := config.Load(serviceName)
    defer cfg.Close()  // Last: config cleanup

    defer logger.Sync()  // Flush logs

    // Sentry flush
    defer errors.Flush(2 * time.Second)

    // OpenTelemetry tracer
    defer func() {
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := tp.Shutdown(shutdownCtx); err != nil {
            logger.Warn("Failed to shutdown tracer", zap.Error(err))
        }
    }()

    // Database connection pool
    defer database.Close(db)

    // Redis client
    defer func() {
        if err := redisClient.Close(); err != nil {
            logger.Warn("Failed to close redis client", zap.Error(err))
        }
    }()

    // NATS event bus
    defer bus.Close()

    // ... server starts ...
}
```

### Health Endpoints for Kubernetes

```go
// From pkg/common/health.go

// LivenessProbe - always healthy if process is running
func LivenessProbe(serviceName, version string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.JSON(http.StatusOK, HealthResponse{
            Status:    "alive",
            Service:   serviceName,
            Version:   version,
            Timestamp: time.Now().UTC().Format(time.RFC3339),
            Uptime:    time.Since(startTime).String(),
        })
    }
}

// ReadinessProbe - healthy when all dependencies are available
func ReadinessProbe(serviceName, version string, checks map[string]func() error) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Run dependency checks in parallel
        checkResults := make(map[string]CheckStatus)
        allHealthy := true

        for name, checkFunc := range checks {
            if err := checkFunc(); err != nil {
                checkResults[name] = CheckStatus{Status: "unhealthy", Message: err.Error()}
                allHealthy = false
            } else {
                checkResults[name] = CheckStatus{Status: "healthy"}
            }
        }

        statusCode := http.StatusOK
        if !allHealthy {
            statusCode = http.StatusServiceUnavailable
        }

        c.JSON(statusCode, HealthResponse{
            Status:  status,
            Checks:  checkResults,
        })
    }
}
```

### Kubernetes Deployment Configuration

```yaml
# Example Kubernetes deployment
spec:
  terminationGracePeriodSeconds: 30  # K8s waits up to 30s
  containers:
  - name: auth-service
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8081
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8081
      initialDelaySeconds: 5
      periodSeconds: 5
    lifecycle:
      preStop:
        exec:
          command: ["sleep", "5"]  # Allow LB to drain
```

### Context Cancellation Propagation

```go
// From cmd/auth/main.go

func main() {
    rootCtx, cancelRotation := context.WithCancel(context.Background())
    defer cancelRotation()

    // JWT key manager respects context cancellation
    keyManager.StartAutoRotation(rootCtx)

    // ... on shutdown, cancelRotation() stops background goroutines
}
```

## Consequences

### Positive

- **Zero dropped requests**: In-flight requests complete before termination.
- **Clean resource cleanup**: Database connections, Redis, and external services close properly.
- **Kubernetes native**: Health probes and signal handling integrate with K8s lifecycle.
- **Observable shutdown**: Logs indicate shutdown progress for debugging.
- **Consistent pattern**: All 14 services follow identical shutdown sequence.

### Negative

- **Delayed shutdown**: 5-second timeout extends deployment time.
- **Timeout tuning**: Some requests may need longer than 5 seconds.
- **Background task awareness**: Long-running tasks must respect context cancellation.

### Timeout Considerations

| Component | Recommended Timeout |
|-----------|-------------------|
| HTTP server | 5 seconds |
| Database queries | 2 seconds (from config) |
| Redis operations | 1 second |
| OpenTelemetry flush | 5 seconds |
| Sentry flush | 2 seconds |
| K8s grace period | 30 seconds |

### Testing Graceful Shutdown

```bash
# Start service
./auth-service &

# Send SIGTERM
kill -TERM $!

# Observe logs
# Expected output:
# {"level":"info","msg":"Shutting down server..."}
# {"level":"info","msg":"Server stopped"}
```

## References

- [cmd/auth/main.go](/cmd/auth/main.go) - Shutdown implementation
- [cmd/rides/main.go](/cmd/rides/main.go) - Complex shutdown with multiple dependencies
- [pkg/common/health.go](/pkg/common/health.go) - Kubernetes health probes
- [k8s/](/k8s/) - Kubernetes deployment manifests
- [Go net/http Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
