# Ride Hailing Platform - Backend Services

A production-ready, scalable ride-hailing platform backend built with Go, following microservices architecture principles.

## Features

- **Authentication Service**: JWT-based authentication with user registration and login
- **Rides Service**: Complete ride lifecycle management (request, accept, start, complete, cancel)
- **Geolocation Service**: Real-time driver location tracking and distance calculation
- **Scalable Architecture**: Microservices design with independent deployment
- **Observability**: Prometheus metrics and Grafana dashboards
- **Database**: PostgreSQL with migrations
- **Caching**: Redis for high-performance data access
- **CI/CD**: Automated testing and deployment pipelines

## Tech Stack

- **Language**: Go 1.22+
- **Framework**: Gin
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Messaging**: Google Pub/Sub (optional)
- **Auth**: JWT
- **Observability**: Prometheus + Grafana
- **Deployment**: Docker + Docker Compose
- **CI/CD**: GitHub Actions

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway (Future)                  │
└────────────┬────────────────┬────────────────┬──────────────┘
             │                │                │
    ┌────────▼────────┐  ┌───▼──────┐  ┌─────▼─────┐
    │  Auth Service   │  │  Rides   │  │    Geo    │
    │   (Port 8081)   │  │ Service  │  │  Service  │
    │                 │  │(Port 8082│  │(Port 8083)│
    └────────┬────────┘  └────┬─────┘  └─────┬─────┘
             │                │              │
    ┌────────▼────────────────▼──────────────▼─────┐
    │            PostgreSQL Database                │
    └───────────────────────────────────────────────┘
                           │
    ┌──────────────────────▼────────────────────────┐
    │               Redis Cache                      │
    └────────────────────────────────────────────────┘
```

## Services

### 1. Auth Service (Port 8081)
- User registration (riders and drivers)
- User login with JWT token generation
- Profile management
- Role-based access control (RBAC)

### 2. Rides Service (Port 8082)
- Ride request creation
- Ride acceptance by drivers
- Ride lifecycle management (requested → accepted → in_progress → completed)
- Ride cancellation
- Ride rating and feedback
- Dynamic fare calculation with surge pricing
- Ride history

### 3. Geo Service (Port 8083)
- Real-time driver location updates
- Driver location retrieval
- Distance calculation (Haversine formula)
- ETA estimation
- Nearby driver matching (planned)

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.22+ (for local development)
- Make (optional, for convenience)

### Running with Docker Compose

1. **Clone the repository**
   ```bash
   git clone https://github.com/richxcame/ride-hailing.git
   cd ride-hailing
   ```

2. **Start all services**
   ```bash
   docker-compose up -d
   ```

3. **Run database migrations**
   ```bash
   make migrate-up
   # Or manually:
   migrate -path db/migrations -database "postgresql://postgres:postgres@localhost:5432/ridehailing?sslmode=disable" up
   ```

4. **Verify services are running**
   ```bash
   # Auth service
   curl http://localhost:8081/healthz

   # Rides service
   curl http://localhost:8082/healthz

   # Geo service
   curl http://localhost:8083/healthz
   ```

### Running Locally

1. **Start dependencies**
   ```bash
   docker-compose up postgres redis -d
   ```

2. **Install tools**
   ```bash
   make install-tools
   ```

3. **Set up environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run migrations**
   ```bash
   make migrate-up
   ```

5. **Run services**
   ```bash
   # Terminal 1 - Auth service
   make run-auth

   # Terminal 2 - Rides service
   make run-rides

   # Terminal 3 - Geo service
   make run-geo
   ```

## API Documentation

See [docs/API.md](docs/API.md) for complete API documentation.

### Quick Examples

#### Register User
```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@example.com",
    "password": "password123",
    "phone_number": "+1234567890",
    "first_name": "John",
    "last_name": "Doe",
    "role": "rider"
  }'
```

#### Login
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@example.com",
    "password": "password123"
  }'
```

#### Request Ride
```bash
curl -X POST http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "pickup_latitude": 40.7128,
    "pickup_longitude": -74.0060,
    "pickup_address": "New York, NY",
    "dropoff_latitude": 40.7589,
    "dropoff_longitude": -73.9851,
    "dropoff_address": "Times Square, NY"
  }'
```

## Development

### Project Structure

```
.
├── cmd/                    # Service entry points
│   ├── auth/              # Auth service main
│   ├── rides/             # Rides service main
│   └── geo/               # Geo service main
├── internal/              # Service implementations
│   ├── auth/             # Auth business logic
│   ├── rides/            # Rides business logic
│   └── geo/              # Geo business logic
├── pkg/                   # Shared packages
│   ├── common/           # Common utilities
│   ├── config/           # Configuration
│   ├── database/         # Database utilities
│   ├── logger/           # Logging
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   └── redis/            # Redis client
├── db/                   # Database files
│   └── migrations/       # SQL migrations
├── monitoring/           # Observability configs
├── .github/              # CI/CD workflows
└── docs/                 # Documentation
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint
```

### Database Migrations

```bash
# Create new migration
make migrate-create NAME=add_new_table

# Run migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Monitoring

### Prometheus
Access Prometheus at: http://localhost:9090

Available metrics:
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request duration

### Grafana
Access Grafana at: http://localhost:3000
- Username: `admin`
- Password: `admin`

## Deployment

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed deployment instructions.

## Configuration

All services are configured via environment variables. See [.env.example](.env.example) for available options.

Key configurations:
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database connection
- `REDIS_HOST`, `REDIS_PORT` - Redis connection
- `JWT_SECRET` - JWT signing secret
- `JWT_EXPIRATION` - Token expiration in hours

## Security

- All passwords are hashed using bcrypt
- JWT tokens for authentication
- Role-based access control (RBAC)
- CORS enabled
- Input validation on all endpoints
- SQL injection prevention with parameterized queries

## Future Enhancements

- Payment service integration (Stripe/Adyen)
- Notification service with Pub/Sub
- Real-time WebSocket connections
- Driver matching algorithm optimization
- Admin dashboard API
- Analytics service
- Fraud detection
- Automated tests expansion
- API Gateway with rate limiting
- Service mesh (Istio)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

---

Built with Go and microservices architecture following the agent-based development plan in [AGENTS.md](AGENTS.md)