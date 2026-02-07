# Security Documentation

This document provides a comprehensive overview of the security architecture, controls, and practices implemented in the ride-hailing platform. It is intended for security auditors, enterprise customers, and development teams.

## Table of Contents

1. [Security Overview](#security-overview)
2. [Authentication & Authorization](#authentication--authorization)
3. [API Security](#api-security)
4. [Data Protection](#data-protection)
5. [Threat Model](#threat-model)
6. [Security Monitoring](#security-monitoring)
7. [Incident Response](#incident-response)
8. [Security Checklist for Development](#security-checklist-for-development)

---

## Security Overview

### Security-First Design Principles

The ride-hailing platform is built with security as a foundational requirement, not an afterthought. Our security architecture follows these core principles:

- **Least Privilege**: All components operate with the minimum permissions required for their function
- **Zero Trust**: Every request is authenticated and authorized, regardless of network origin
- **Secure by Default**: Security controls are enabled by default; insecure options require explicit configuration
- **Defense in Depth**: Multiple overlapping security layers protect against single points of failure
- **Fail Secure**: System failures default to a secure state, denying access rather than granting it

### Defense in Depth Strategy

Our multi-layered security approach includes:

| Layer | Controls |
|-------|----------|
| Network | TLS 1.2+, HTTPS enforcement, CORS policies |
| Application | JWT authentication, RBAC, input validation |
| API Gateway | Rate limiting, request filtering, WAF integration |
| Data | Encryption at rest, encryption in transit, PII masking |
| Monitoring | Sentry error tracking, fraud detection, audit logging |
| Process | Security scanning in CI, code reviews, dependency audits |

---

## Authentication & Authorization

### JWT-Based Authentication Flow

Authentication is handled via JSON Web Tokens (JWT) with the following flow:

1. User submits credentials (phone/email + password or OTP)
2. Server validates credentials and issues signed JWT access token
3. Client includes token in `Authorization: Bearer <token>` header
4. Middleware validates token signature and expiration
5. User identity and role are extracted from token claims

**Implementation Reference**: `pkg/middleware/auth.go`

```go
// Claims structure embedded in JWT tokens
type Claims struct {
    UserID uuid.UUID       `json:"user_id"`
    Email  string          `json:"email"`
    Role   models.UserRole `json:"role"`
    jwt.RegisteredClaims
}
```

The `AuthMiddlewareWithProvider` function validates tokens using configurable key providers, supporting both static secrets and rotated keys.

### Token Lifecycle

| Token Type | Lifetime | Purpose |
|------------|----------|---------|
| Access Token | 15-60 minutes | API request authentication |
| Refresh Token | 7-30 days | Obtaining new access tokens |
| OTP Codes | 5 minutes | One-time verification |
| Trusted Device Token | 30 days | Remember trusted devices for 2FA bypass |

### Key Rotation Strategy

The platform implements automated JWT signing key rotation through the `jwtkeys` package:

**Implementation Reference**: `pkg/jwtkeys/manager.go`

- **Rotation Interval**: Keys rotate every 30 days by default (configurable)
- **Grace Period**: Old keys remain valid for 30 days after rotation
- **Key Identification**: Each token includes a `kid` (Key ID) header for key resolution
- **Storage Options**:
  - File-based storage for single-node deployments
  - HashiCorp Vault integration for enterprise deployments (`pkg/jwtkeys/store_vault.go`)
  - In-memory storage for testing

```go
// Key rotation is automatic via background goroutine
manager.StartAutoRotation(ctx)
manager.StartAutoRefresh(ctx, 5*time.Minute)
```

### Role-Based Access Control (RBAC)

The platform enforces strict role-based access control:

| Role | Permissions |
|------|-------------|
| `rider` | Book rides, view ride history, manage payment methods |
| `driver` | Accept rides, update location, manage earnings |
| `admin` | Full system access, user management, fraud investigation |
| `corporate_admin` | Manage corporate accounts, view team usage |

**Implementation Reference**: `pkg/middleware/auth.go`

```go
// RequireRole middleware enforces role-based access
func RequireRole(roles ...models.UserRole) gin.HandlerFunc
```

### Multi-Factor Authentication (2FA)

The 2FA service provides comprehensive multi-factor authentication:

**Implementation Reference**: `internal/twofa/service.go`

**Supported Methods**:
- **SMS OTP**: 6-digit codes sent via SMS, valid for 5 minutes
- **TOTP**: Time-based one-time passwords (Google Authenticator compatible)
- **Backup Codes**: 10 single-use recovery codes generated on 2FA enrollment

**Security Controls**:
- OTP codes are bcrypt-hashed before storage
- Rate limiting: Maximum 5 OTP requests per hour per phone number
- Maximum 3 verification attempts per OTP
- Trusted device management for reducing 2FA friction
- Audit logging for all 2FA events

```go
// OTPs are hashed before storage
otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
```

---

## API Security

### HTTPS/TLS Requirements

- All API endpoints require HTTPS (TLS 1.2 or higher)
- HTTP Strict Transport Security (HSTS) is enforced with a 1-year max-age
- SSL/TLS termination occurs at the load balancer or API gateway

### Rate Limiting

Rate limiting is implemented using a Redis-backed token bucket algorithm:

**Implementation Reference**: `pkg/ratelimit/limiter.go`, `pkg/middleware/rate_limit.go`

| Identity Type | Default Limit | Burst Allowance |
|---------------|---------------|-----------------|
| Anonymous (IP-based) | Configurable per endpoint | Configurable |
| Authenticated (User ID) | Higher limits | Configurable |

**Response Headers**:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining in current window
- `X-RateLimit-Reset`: Seconds until limit resets
- `Retry-After`: Seconds to wait when rate limited (429 responses)

### Request Validation and Sanitization

- All request bodies are validated against defined schemas using Gin binding
- Input length limits are enforced to prevent buffer overflow attacks
- SQL queries use parameterized statements exclusively (no string concatenation)
- User-provided data is sanitized before logging or error messages

### CORS Policy

Cross-Origin Resource Sharing is configured to allow only trusted origins. The policy is enforced at the API gateway level with the following restrictions:

- Allowed origins are explicitly whitelisted
- Credentials are only allowed for whitelisted origins
- Preflight requests (`OPTIONS`) are handled appropriately

### Security Headers

**Implementation Reference**: `pkg/middleware/security_headers.go`

The `SecurityHeaders()` middleware applies the following protective headers:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevent MIME type sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `X-XSS-Protection` | `1; mode=block` | Enable browser XSS filter |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Enforce HTTPS |
| `Content-Security-Policy` | Restrictive policy | Prevent XSS and injection |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer leakage |
| `Permissions-Policy` | Disabled dangerous features | Restrict browser APIs |

### Idempotency for Critical Operations

Critical operations (payments, ride creation) support idempotency keys to prevent duplicate processing:

- Clients provide `X-Idempotency-Key` header with unique request identifier
- Server caches responses for idempotent operations
- Duplicate requests return cached response without re-execution

---

## Data Protection

### Encryption at Rest

- **Database**: PostgreSQL with encrypted storage volumes
- **Secrets**: Stored in HashiCorp Vault with AES-256 encryption
- **Backups**: Encrypted using customer-managed or platform-managed keys
- **File Storage**: Server-side encryption for all uploaded files

### Encryption in Transit

- All network traffic is encrypted using TLS 1.2 or higher
- Internal service-to-service communication uses mTLS where applicable
- Database connections require SSL (`sslmode=require` or `verify-full`)
- Redis connections support TLS encryption

### Sensitive Data Handling

**Password Storage**:
- Passwords are hashed using bcrypt with cost factor 10+
- Never logged, displayed, or transmitted in plaintext

**Payment Information**:
- Credit card details are tokenized via payment processor (Stripe)
- PCI DSS compliance maintained by not storing raw card data
- Only last 4 digits and card type stored for display purposes

**API Keys and Secrets**:
- Environment variables or secret management systems (Vault)
- Never committed to source control
- Rotated regularly with automated key rotation

### PII Protection Strategies

- Phone numbers are masked in logs and audit trails (`+1234****`)
- Email addresses are partially obscured in customer support contexts
- Location data is retained only for operational and legal requirements
- Data retention policies enforce automatic deletion of old PII
- GDPR/CCPA compliance with data export and deletion capabilities

---

## Threat Model

### Authentication Attacks

| Threat | Mitigation |
|--------|------------|
| **Brute Force** | Rate limiting on login endpoints (5 attempts/hour), exponential backoff, account lockout after repeated failures |
| **Credential Stuffing** | Rate limiting per IP, CAPTCHA on suspicious traffic, 2FA requirement for sensitive actions |
| **Token Theft** | Short token lifetimes, secure cookie flags, token binding to device fingerprint |

### Injection Attacks

| Threat | Mitigation |
|--------|------------|
| **SQL Injection** | Parameterized queries only, ORM with safe query building, input validation |
| **XSS (Cross-Site Scripting)** | CSP headers, output encoding, React's built-in XSS protection |
| **Command Injection** | No shell command execution with user input, input sanitization |
| **NoSQL Injection** | Not applicable (PostgreSQL only) |

### Session Hijacking

- JWT tokens are not stored in localStorage (vulnerable to XSS)
- HttpOnly and Secure flags on session cookies
- Token refresh rotation invalidates old refresh tokens
- Session binding to IP/User-Agent with anomaly detection

### Man-in-the-Middle Attacks

- HTTPS enforced for all communications
- HSTS prevents protocol downgrade attacks
- Certificate pinning in mobile applications
- Public key pinning for critical API endpoints

### Denial of Service

| Attack Vector | Mitigation |
|---------------|------------|
| **Application Layer** | Rate limiting, request size limits, timeout configurations |
| **Network Layer** | Cloud provider DDoS protection, CDN caching |
| **Resource Exhaustion** | Connection pooling, memory limits, graceful degradation |

---

## Security Monitoring

### Error Tracking (Sentry Integration)

**Implementation Reference**: `pkg/errors/sentry.go`

Sentry provides real-time error tracking and alerting:

- **Automatic Error Capture**: Unhandled exceptions and panics are reported
- **Contextual Information**: User ID, request details, correlation IDs attached
- **Sensitive Data Filtering**: Authorization headers, cookies, API keys are redacted
- **Environment Separation**: Distinct projects for development, staging, production

```go
// Sensitive headers are automatically sanitized
sensitiveHeaders := map[string]bool{
    "Authorization": true,
    "Cookie":        true,
    "X-API-Key":     true,
    "X-Auth-Token":  true,
}
```

### Fraud Detection

**Implementation Reference**: `internal/fraud/service.go`

The fraud detection service provides comprehensive protection:

**Detection Capabilities**:
- Payment fraud (failed attempts, chargebacks, suspicious transactions)
- Ride fraud (excessive cancellations, fake GPS, promo abuse)
- Account fraud (multiple accounts, VPN usage, suspicious patterns)

**Risk Scoring**:
- Users receive dynamic risk scores (0-100)
- Scores above 70 trigger fraud alerts
- Scores above 90 trigger automatic account suspension

**Alert Levels**:
- `Low`: Score 0-49, informational only
- `Medium`: Score 50-69, enhanced monitoring
- `High`: Score 70-89, investigation required
- `Critical`: Score 90+, immediate action required

### Suspicious Activity Alerts

The platform generates alerts for:

- Failed authentication attempts exceeding threshold
- Unusual login locations or device changes
- High-value transactions from new devices
- API usage patterns indicating automation/scraping
- Sudden changes in user behavior patterns

### Audit Logging

All security-relevant events are logged with:

- Timestamp (UTC)
- User ID (if authenticated)
- IP address and User-Agent
- Action type and status
- Additional context as JSON

**Logged Events**:
- Authentication attempts (success/failure)
- 2FA enrollment and verification
- Password changes
- Role/permission changes
- Payment method additions
- Admin actions on user accounts

---

## Incident Response

### Security Contact Information

| Contact | Email | Response Time |
|---------|-------|---------------|
| Security Team | security@[domain].com | 24 hours |
| Emergency Hotline | [phone number] | 1 hour |
| Bug Bounty Program | [platform URL] | 48 hours |

### Vulnerability Disclosure Process

1. **Report**: Submit vulnerability details to security@[domain].com
2. **Acknowledge**: Security team acknowledges within 24-48 hours
3. **Triage**: Vulnerability is assessed for severity and impact
4. **Remediate**: Fix is developed, tested, and deployed
5. **Notify**: Reporter is notified when fix is deployed
6. **Disclose**: Coordinated public disclosure if appropriate

### Incident Severity Levels

| Level | Description | Response Time | Examples |
|-------|-------------|---------------|----------|
| **P0 - Critical** | Active exploitation, data breach | Immediate (< 1 hour) | Unauthorized data access, active attack |
| **P1 - High** | Imminent threat, severe vulnerability | < 4 hours | RCE vulnerability, auth bypass |
| **P2 - Medium** | Significant vulnerability | < 24 hours | XSS, information disclosure |
| **P3 - Low** | Minor issue, hardening | < 1 week | Missing headers, configuration issues |

### Response Procedures

1. **Containment**: Isolate affected systems, revoke compromised credentials
2. **Investigation**: Determine scope, root cause, and impact
3. **Eradication**: Remove threat, patch vulnerabilities
4. **Recovery**: Restore systems, verify integrity
5. **Post-Incident**: Document lessons learned, update procedures

---

## Security Checklist for Development

### Code Review Requirements

All code changes must:

- [ ] Pass automated security scanning (Gosec)
- [ ] Be reviewed by at least one other developer
- [ ] Include security considerations in PR description
- [ ] Not introduce new security warnings or vulnerabilities
- [ ] Follow secure coding guidelines

### Security Testing (Gosec in CI)

**Implementation Reference**: `.github/workflows/ci.yml`

The CI pipeline includes automated security scanning:

```yaml
- name: Run Gosec Security Scanner
  uses: securego/gosec@v2.22.11
  with:
    args: "-no-fail -fmt sarif -out gosec-results.sarif ./..."

- name: Upload Gosec results
  uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: gosec-results.sarif
```

**Scanned Vulnerabilities**:
- SQL injection patterns
- Hardcoded credentials
- Insecure cryptographic usage
- Path traversal vulnerabilities
- Integer overflow risks

### Dependency Scanning

- Dependencies are reviewed for known vulnerabilities
- `go mod tidy` is enforced in CI
- Regular dependency updates through automated PRs
- License compliance checking

### Secrets Management

**Do**:
- Use environment variables for secrets
- Store secrets in HashiCorp Vault for production
- Rotate secrets regularly
- Use different secrets per environment

**Do Not**:
- Commit secrets to source control
- Log secrets or include in error messages
- Share secrets via insecure channels
- Use default or weak secrets

### Pre-Deployment Checklist

- [ ] All security tests pass
- [ ] No high/critical vulnerabilities in dependencies
- [ ] Secrets are properly configured (not defaults)
- [ ] Security headers are enabled
- [ ] Rate limiting is configured
- [ ] TLS certificates are valid
- [ ] Monitoring and alerting are configured
- [ ] Rollback plan is documented

---

## Appendix: Security Configuration Reference

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `JWT_SECRET` | Legacy JWT signing secret | Yes (deprecated) |
| `JWT_KEY_FILE` | Path to key rotation file | Recommended |
| `VAULT_ADDRESS` | HashiCorp Vault URL | Production |
| `VAULT_TOKEN` | Vault authentication token | Production |
| `SENTRY_DSN` | Sentry error tracking DSN | Recommended |
| `RATE_LIMIT_ENABLED` | Enable rate limiting | Yes |

### Recommended Security Headers for Nginx/Load Balancer

```nginx
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
```

---

*Last Updated: February 2026*
*Document Version: 1.0*
