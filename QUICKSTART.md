# Quick Start Guide

Get the ride-hailing platform running in 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- Terminal/Command line access

## Step-by-Step Setup

### 1. Start All Services

```bash
# Clone the repo (if not already done)
git clone https://github.com/richxcame/ride-hailing.git
cd ride-hailing

# Start all services
docker-compose up -d
```

Wait for all services to start (about 30 seconds). You should see:
- postgres
- redis
- auth-service
- rides-service
- geo-service
- prometheus
- grafana

### 2. Check Service Health

```bash
# Auth service
curl http://localhost:8081/healthz

# Rides service
curl http://localhost:8082/healthz

# Geo service
curl http://localhost:8083/healthz
```

All should return `{"status":"healthy",...}`

### 3. Run Database Migrations

```bash
# Install migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path db/migrations \
  -database "postgresql://postgres:postgres@localhost:5432/ridehailing?sslmode=disable" \
  up
```

Or if you have Make installed:
```bash
make migrate-up
```

## Test the APIs

### 1. Register a Rider

```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@test.com",
    "password": "password123",
    "phone_number": "+1234567890",
    "first_name": "John",
    "last_name": "Doe",
    "role": "rider"
  }'
```

### 2. Register a Driver

```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "driver@test.com",
    "password": "password123",
    "phone_number": "+1234567891",
    "first_name": "Jane",
    "last_name": "Smith",
    "role": "driver"
  }'
```

### 3. Login as Rider

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@test.com",
    "password": "password123"
  }'
```

**Save the token from the response!** You'll need it for the next steps.

### 4. Login as Driver

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "driver@test.com",
    "password": "password123"
  }'
```

**Save this token too!**

### 5. Request a Ride (as Rider)

Replace `<RIDER_TOKEN>` with the token from step 3:

```bash
curl -X POST http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer <RIDER_TOKEN>" \
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

**Save the ride ID from the response!**

### 6. Get Available Rides (as Driver)

Replace `<DRIVER_TOKEN>` with the token from step 4:

```bash
curl -X GET http://localhost:8082/api/v1/driver/rides/available \
  -H "Authorization: Bearer <DRIVER_TOKEN>"
```

### 7. Accept the Ride (as Driver)

Replace `<RIDE_ID>` with the ID from step 5:

```bash
curl -X POST http://localhost:8082/api/v1/driver/rides/<RIDE_ID>/accept \
  -H "Authorization: Bearer <DRIVER_TOKEN>"
```

### 8. Update Driver Location

```bash
curl -X POST http://localhost:8083/api/v1/geo/location \
  -H "Authorization: Bearer <DRIVER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 40.7128,
    "longitude": -74.0060
  }'
```

### 9. Start the Ride (as Driver)

```bash
curl -X POST http://localhost:8082/api/v1/driver/rides/<RIDE_ID>/start \
  -H "Authorization: Bearer <DRIVER_TOKEN>"
```

### 10. Complete the Ride (as Driver)

```bash
curl -X POST http://localhost:8082/api/v1/driver/rides/<RIDE_ID>/complete \
  -H "Authorization: Bearer <DRIVER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_distance": 5.2
  }'
```

### 11. Rate the Ride (as Rider)

```bash
curl -X POST http://localhost:8082/api/v1/rides/<RIDE_ID>/rate \
  -H "Authorization: Bearer <RIDER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 5,
    "feedback": "Great ride!"
  }'
```

## Access Monitoring

### Prometheus
Open http://localhost:9090 in your browser

Try this query:
```
http_requests_total
```

### Grafana
Open http://localhost:3000 in your browser

- Username: `admin`
- Password: `admin`

Add Prometheus datasource:
- URL: `http://prometheus:9090`

## View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f auth-service
docker-compose logs -f rides-service
docker-compose logs -f geo-service
```

## Stop Services

```bash
docker-compose down
```

To remove all data:
```bash
docker-compose down -v
```

## Troubleshooting

### Services won't start
```bash
# Check logs
docker-compose logs

# Restart services
docker-compose restart
```

### Database connection errors
```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Check logs
docker-compose logs postgres
```

### Port already in use
Edit `docker-compose.yml` and change the port mappings:
```yaml
ports:
  - "8081:8080"  # Change 8081 to any available port
```

## Next Steps

- Read [README.md](README.md) for detailed information
- Check [docs/API.md](docs/API.md) for complete API documentation
- See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for deployment guides
- Review [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) for project overview

## Development Mode

To run services locally (without Docker):

```bash
# Start dependencies only
docker-compose up postgres redis -d

# In separate terminals
make run-auth
make run-rides
make run-geo
```

## Testing

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Run linter
make lint
```

---

**Congratulations!** You've successfully set up and tested the ride-hailing platform! 🎉
