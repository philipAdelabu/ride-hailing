# ADR-0008: Secrets Rotation Strategy

## Status

Accepted

## Context

Production security requires robust secrets management with regular rotation:

1. **JWT signing keys**: Compromised keys allow unauthorized access until revoked.
2. **API keys**: External service credentials (Stripe, Twilio) are high-value targets.
3. **Database credentials**: Leaked passwords grant full data access.
4. **Hardcoded secrets**: Environment variables in version control create permanent vulnerabilities.

The platform must support zero-downtime rotation with grace periods.

## Decision

Implement **environment-based secrets management** with automated rotation via `pkg/jwtkeys` and `pkg/secrets`.

### JWT Key Rotation

```go
// From pkg/jwtkeys/manager.go
type Config struct {
    RotationInterval time.Duration  // Default: 30 days
    GracePeriod      time.Duration  // Default: 30 days
    Store            Store
}

func (m *Manager) StartAutoRotation(ctx context.Context) {
    interval := m.rotationInterval / 4
    go func() {
        ticker := time.NewTicker(interval)
        for {
            select {
            case <-ticker.C:
                _ = m.EnsureRotation(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

### Environment Variable Injection

```go
// From pkg/config/config.go - secrets loaded from environment, never hardcoded
JWT: JWTConfig{
    Secret:        getEnv("JWT_SECRET", ""),
    RotationHours: getEnvAsInt("JWT_ROTATION_HOURS", 24*30),
    GraceHours:    getEnvAsInt("JWT_ROTATION_GRACE_HOURS", 24*30),
    VaultAddress:  getEnv("JWT_KEYS_VAULT_ADDR", ""),
}
```

### External Secrets Manager Integration

Supports Vault, AWS Secrets Manager, GCP Secret Manager, and Kubernetes secrets.

## Rotation Schedule

| Secret Type | Rotation | Grace Period |
|-------------|----------|--------------|
| JWT signing keys | Monthly | 30 days |
| API keys (Stripe, Twilio, Maps) | Quarterly | 7 days |
| Database passwords | Annually | 24 hours |

## Emergency Rotation Procedure

1. **Generate**: Create new secret in secrets manager
2. **Deploy**: Push environment variable update to all pods
3. **Verify**: Confirm new secret is active via health checks
4. **Revoke**: Mark old key as revoked (`SigningKey.Revoked = true`)

## Consequences

### Positive

- **Zero-downtime rotation**: Grace periods prevent token invalidation.
- **Audit trail**: Secrets manager integrations provide access logging.
- **Centralized management**: Single source of truth for credentials.
- **Provider flexibility**: Vault, AWS, GCP, and Kubernetes supported.

### Negative

- **Operational complexity**: Rotation schedules require monitoring.
- **Cache invalidation**: TTL-based caching may delay propagation.
- **Emergency response**: Compromised secrets require manual intervention.

## References

- [pkg/jwtkeys/manager.go](/pkg/jwtkeys/manager.go) - JWT key rotation
- [pkg/config/config.go](/pkg/config/config.go) - Environment configuration
