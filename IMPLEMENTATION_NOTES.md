# Implementation Notes - Complete System Documentation

## Last Updated: 2025-11-05

---

## System Overview

This is a **production-ready ride-hailing platform** built with Go, featuring 8 microservices that handle everything from user authentication to real-time WebSocket communication.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Client Applications                           │
│              (Mobile Apps, Web Dashboard, Admin Panel)           │
└────┬──────────┬──────────┬──────────┬──────────┬───────────────┘
     │          │          │          │          │
┌────▼─────┐┌──▼────┐┌───▼───┐┌─────▼────┐┌───▼──────┐
│   Auth   ││ Rides ││  Geo  ││ Payments ││  Notifs  │
│  :8081   ││ :8082 ││ :8083 ││  :8084   ││  :8085   │
└────┬─────┘└───┬───┘└───┬───┘└────┬─────┘└────┬─────┘
     │          │        │         │            │
┌────▼─────┐┌──▼────┐┌──▼───┐
│ Realtime ││Mobile ││Admin │
│  :8086   ││ :8087 ││ :8088│
└────┬─────┘└───┬───┘└──┬───┘
     │          │       │
     └──────────┴───────┴──────────────────┐
                         │                 │
            ┌────────────▼────────────┐    │
            │   PostgreSQL Database   │    │
            │      (Persistent)       │    │
            └────────────┬────────────┘    │
                         │                 │
            ┌────────────▼────────────┐    │
            │     Redis Cluster       │◄───┘
            │  (Cache + GeoSpatial +  │
            │       WebSocket)        │
            └─────────────────────────┘
```

### Services

| Service | Port | Purpose | Status |
|---------|------|---------|--------|
| Auth | 8081 | User authentication, JWT tokens | ✅ Production |
| Rides | 8082 | Ride lifecycle management | ✅ Production |
| Geo | 8083 | Location tracking + driver matching | ✅ Production |
| Payments | 8084 | Stripe integration + wallets | ✅ Production |
| Notifications | 8085 | Multi-channel notifications | ✅ Production |
| Real-time | 8086 | WebSocket + chat | ✅ Production |
| Mobile | 8087 | Mobile-optimized APIs | ✅ Production |
| Admin | 8088 | Admin dashboard backend | ✅ Production |

---

## Service Details

### 1. Auth Service (Port 8081)

**Purpose**: User authentication and authorization

**Key Files**:
- [cmd/auth/main.go](cmd/auth/main.go) - Entry point
- [internal/auth/repository.go](internal/auth/repository.go) - User database operations
- [internal/auth/service.go](internal/auth/service.go) - Auth business logic
- [internal/auth/handler.go](internal/auth/handler.go) - HTTP endpoints
- [pkg/middleware/auth.go](pkg/middleware/auth.go) - JWT middleware

**Endpoints**:
- `POST /api/v1/auth/register` - Register new user (rider/driver)
- `POST /api/v1/auth/login` - Login and get JWT token
- `POST /api/v1/auth/refresh` - Refresh JWT token
- `GET /healthz` - Health check

**Features**:
- JWT-based authentication
- Password hashing (bcrypt)
- Role-based access (rider, driver, admin)
- Token refresh mechanism

**Database Tables**:
- `users` - User accounts with roles

---

### 2. Rides Service (Port 8082)

**Purpose**: Complete ride lifecycle management

**Key Files**:
- [cmd/rides/main.go](cmd/rides/main.go) - Entry point
- [internal/rides/repository.go](internal/rides/repository.go) - Ride database operations
- [internal/rides/service.go](internal/rides/service.go) - Ride business logic
- [internal/rides/handler.go](internal/rides/handler.go) - HTTP endpoints

**Endpoints**:
- `POST /api/v1/rides` - Request a new ride
- `GET /api/v1/rides/:id` - Get ride details
- `POST /api/v1/rides/:id/accept` - Driver accepts ride
- `POST /api/v1/rides/:id/start` - Start ride
- `POST /api/v1/rides/:id/complete` - Complete ride
- `POST /api/v1/rides/:id/cancel` - Cancel ride
- `POST /api/v1/rides/:id/rate` - Rate completed ride
- `GET /api/v1/rides/history` - Get ride history (with filters)
- `GET /api/v1/rides/:id/receipt` - Get ride receipt

**Features**:
- Ride request with fare estimation
- Driver acceptance workflow
- Status tracking (requested → accepted → in_progress → completed)
- Cancellation with reasons
- Rating and feedback system
- Advanced filtering (status, date range)
- Receipt generation

**Database Tables**:
- `rides` - All ride records
- `drivers` - Driver profiles

**Ride Statuses**:
1. `requested` - Rider created request
2. `accepted` - Driver accepted
3. `in_progress` - Ride started
4. `completed` - Successfully finished
5. `cancelled` - Cancelled by rider/driver

---

### 3. Geo Service (Port 8083)

**Purpose**: Location tracking and driver matching

**Key Files**:
- [cmd/geo/main.go](cmd/geo/main.go) - Entry point
- [internal/geo/service.go](internal/geo/service.go) - Geo business logic with Redis GeoSpatial
- [internal/geo/repository.go](internal/geo/repository.go) - Location database operations
- [internal/geo/handler.go](internal/geo/handler.go) - HTTP endpoints
- [pkg/redis/redis.go](pkg/redis/redis.go) - Redis GeoSpatial methods

**Endpoints**:
- `POST /api/v1/geo/location` - Update driver location
- `GET /api/v1/geo/nearby` - Find nearby drivers
- `GET /api/v1/geo/location/:id` - Get specific location
- `GET /healthz` - Health check

**Features**:
- Real-time driver location updates
- Redis GEORADIUS for efficient nearby search
- 10km search radius (configurable)
- Driver status tracking (available/busy/offline)
- Automatic geo index maintenance
- Distance-based sorting

**Database Tables**:
- `driver_locations` - Persistent location history

**Redis Data Structures**:
- Key: `drivers:geo:index` (GEO sorted set)
- Stores: `driver_id → (longitude, latitude)`
- Commands: GEOADD, GEORADIUS, ZREM

**Algorithm**:
1. Driver updates location → Saved to PostgreSQL + Redis GEO index
2. Rider requests ride → GEORADIUS finds drivers within 10km
3. Filter by availability status
4. Sort by distance (ascending)
5. Return top 5 drivers

---

### 4. Payments Service (Port 8084)

**Purpose**: Payment processing and wallet management

**Key Files**:
- [cmd/payments/main.go](cmd/payments/main.go) - Entry point
- [internal/payments/repository.go](internal/payments/repository.go) - Payment database operations
- [internal/payments/stripe.go](internal/payments/stripe.go) - Stripe API wrapper
- [internal/payments/service.go](internal/payments/service.go) - Payment business logic
- [internal/payments/handler.go](internal/payments/handler.go) - HTTP endpoints

**Endpoints**:
- `POST /api/v1/payments/process` - Process ride payment
- `POST /api/v1/wallet/topup` - Add funds to wallet
- `GET /api/v1/wallet` - Get wallet balance
- `GET /api/v1/wallet/transactions` - Transaction history
- `POST /api/v1/payments/:id/refund` - Process refund
- `GET /api/v1/payments/:id` - Get payment details
- `POST /api/v1/webhooks/stripe` - Stripe webhooks

**Features**:
- Stripe Payment Intent integration
- Wallet system (top-up, balance check)
- Dual payment methods (wallet or Stripe)
- Automatic driver payouts (80/20 split)
- Platform commission (20%)
- Refunds with cancellation fees (10%)
- Transaction history with pagination
- Webhook handling for async events

**Database Tables**:
- `wallets` - User wallet balances
- `payments` - Payment records
- `wallet_transactions` - All wallet transactions

**Payment Flow**:
1. Ride completes → Payment triggered
2. If wallet method:
   - Deduct from rider wallet
   - Credit driver wallet (80%)
   - Platform keeps commission (20%)
3. If Stripe method:
   - Create Payment Intent
   - Charge rider
   - Payout to driver (80%)
4. Record all transactions

**Commission Split**:
- Driver: 80% of fare
- Platform: 20% commission

**Refund Policy**:
- Full refund - 10% cancellation fee
- Cancellation fee goes to driver

---

### 5. Notifications Service (Port 8085)

**Purpose**: Multi-channel notification delivery

**Key Files**:
- [cmd/notifications/main.go](cmd/notifications/main.go) - Entry point with background worker
- [internal/notifications/repository.go](internal/notifications/repository.go) - Notification database
- [internal/notifications/firebase.go](internal/notifications/firebase.go) - Firebase push notifications
- [internal/notifications/twilio.go](internal/notifications/twilio.go) - Twilio SMS
- [internal/notifications/email.go](internal/notifications/email.go) - SMTP email
- [internal/notifications/service.go](internal/notifications/service.go) - Notification business logic
- [internal/notifications/handler.go](internal/notifications/handler.go) - HTTP endpoints

**Endpoints**:
- `GET /api/v1/notifications` - List notifications (paginated)
- `GET /api/v1/notifications/unread/count` - Get unread count
- `POST /api/v1/notifications/:id/read` - Mark as read
- `POST /api/v1/notifications/send` - Send notification
- `POST /api/v1/notifications/schedule` - Schedule notification
- `POST /api/v1/notifications/ride/requested` - Ride requested event
- `POST /api/v1/notifications/ride/accepted` - Ride accepted event
- `POST /api/v1/notifications/ride/started` - Ride started event
- `POST /api/v1/notifications/ride/completed` - Ride completed event
- `POST /api/v1/notifications/ride/cancelled` - Ride cancelled event
- `POST /api/v1/admin/notifications/bulk` - Bulk send (admin only)

**Features**:
- **Firebase Cloud Messaging** - Push notifications
- **Twilio** - SMS notifications
- **SMTP** - Email with HTML templates
- Multi-channel support (send to all channels)
- Scheduled notifications (future delivery)
- Bulk notifications (to all users)
- Background worker (processes pending every 1 minute)
- Email templates:
  - Welcome email
  - Ride confirmation
  - Ride receipt
  - Custom templates

**Database Tables**:
- `notifications` - All notification records

**Notification Types**:
- `ride_requested` - Sent to available drivers
- `ride_accepted` - Sent to rider
- `ride_started` - Sent to rider
- `ride_completed` - Sent to both rider & driver
- `ride_cancelled` - Sent to affected party
- `payment_received` - Sent to driver
- `welcome` - Sent on registration

**Background Worker**:
- Runs every 1 minute
- Processes pending notifications
- Retries failed notifications
- Updates status (pending → sent → failed)

---

### 6. Real-time Service (Port 8086)

**Purpose**: WebSocket connections and real-time updates

**Key Files**:
- [cmd/realtime/main.go](cmd/realtime/main.go) - Entry point
- [pkg/websocket/client.go](pkg/websocket/client.go) - WebSocket client management
- [pkg/websocket/hub.go](pkg/websocket/hub.go) - Central hub for connections
- [internal/realtime/service.go](internal/realtime/service.go) - Real-time business logic
- [internal/realtime/handler.go](internal/realtime/handler.go) - HTTP + WebSocket endpoints

**Endpoints**:
- `GET /ws` - WebSocket upgrade endpoint
- `POST /api/v1/internal/broadcast` - Internal broadcast API

**WebSocket Message Types**:
1. `location_update` - Driver sends location → Broadcast to riders in same ride
2. `ride_status` - Ride status changed → Broadcast to ride participants
3. `chat_message` - In-app chat → Send to other party + save to Redis
4. `typing` - Typing indicator → Send to other party
5. `join_ride` - Join ride room → Subscribe to ride events
6. `leave_ride` - Leave ride room → Unsubscribe from ride events

**Features**:
- WebSocket Hub pattern
- Client read/write pumps
- Ping/pong heartbeat (60s interval)
- Message buffering (256 messages)
- Room-based messaging (ride-specific)
- Broadcast to user/ride/all
- Redis-backed chat history (24h TTL)
- Driver location caching (5min TTL)

**WebSocket Connection Flow**:
1. Client connects to `/ws` with JWT token
2. Upgrade HTTP to WebSocket
3. Create Client instance
4. Register with Hub
5. Start ReadPump (goroutine)
6. Start WritePump (goroutine)
7. Hub routes messages based on type

**Redis Data Structures**:
- `ride:chat:{rideID}` - List of chat messages (24h TTL)
- `driver:location:{driverID}` - Latest location JSON (5min TTL)

**Hub Architecture**:
```go
type Hub struct {
    clients    map[string]*Client       // userID → Client
    rides      map[string]map[string]*Client  // rideID → userID → Client
    Register   chan *Client
    Unregister chan *Client
    Broadcast  chan *BroadcastMessage
    handlers   map[string]MessageHandler
}
```

---

### 7. Mobile Service (Port 8087)

**Purpose**: Mobile app optimized APIs

**Key Files**:
- [cmd/mobile/main.go](cmd/mobile/main.go) - Entry point
- [internal/favorites/repository.go](internal/favorites/repository.go) - Favorite locations database
- [internal/favorites/service.go](internal/favorites/service.go) - Favorites business logic
- [internal/favorites/handler.go](internal/favorites/handler.go) - Favorites HTTP endpoints
- [internal/rides/handler.go](internal/rides/handler.go) - Enhanced with mobile endpoints

**Endpoints**:
- `GET /api/v1/rides/history` - Ride history with filters
- `GET /api/v1/rides/:id/receipt` - Trip receipt
- `POST /api/v1/rides/:id/rate` - Rate ride
- `POST /api/v1/favorites` - Add favorite location
- `GET /api/v1/favorites` - List favorite locations
- `PUT /api/v1/favorites/:id` - Update favorite location
- `DELETE /api/v1/favorites/:id` - Delete favorite location
- `GET /api/v1/profile` - Get user profile
- `PUT /api/v1/profile` - Update user profile

**Features**:
- **Ride History**:
  - Filter by status (completed, cancelled, all)
  - Filter by date range (start_date, end_date)
  - Pagination (limit, offset)
  - Total count for pagination UI

- **Favorite Locations**:
  - Save frequently visited places (Home, Work, Gym, etc.)
  - Quick address selection
  - Lat/lng coordinates
  - Custom names

- **Trip Receipts**:
  - Detailed fare breakdown
  - Pickup/dropoff addresses
  - Distance and duration
  - Payment method
  - Driver info

- **Ratings**:
  - 1-5 star rating
  - Optional feedback text
  - Updates driver average rating

**Database Tables**:
- `favorite_locations` - Saved addresses

**Ride History Filters**:
```json
{
  "status": "completed",  // or "cancelled", optional
  "start_date": "2025-01-01",  // optional
  "end_date": "2025-11-05",    // optional
  "limit": 20,
  "offset": 0
}
```

**Receipt Format**:
```json
{
  "ride_id": "uuid",
  "pickup_address": "123 Main St",
  "dropoff_address": "456 Oak Ave",
  "distance_km": 5.2,
  "duration_minutes": 15,
  "base_fare": 5.00,
  "distance_fare": 7.80,
  "time_fare": 3.75,
  "surge_multiplier": 1.5,
  "total_fare": 24.83,
  "payment_method": "wallet",
  "driver_name": "John Doe",
  "driver_rating": 4.8,
  "completed_at": "2025-11-05T14:30:00Z"
}
```

---

### 8. Admin Service (Port 8088)

**Purpose**: Admin dashboard backend

**Key Files**:
- [cmd/admin/main.go](cmd/admin/main.go) - Entry point
- [internal/admin/repository.go](internal/admin/repository.go) - Admin database operations
- [internal/admin/service.go](internal/admin/service.go) - Admin business logic
- [internal/admin/handler.go](internal/admin/handler.go) - HTTP endpoints
- [pkg/middleware/admin.go](pkg/middleware/admin.go) - Admin role middleware

**Endpoints**:
- `GET /api/v1/admin/dashboard` - Dashboard statistics
- `GET /api/v1/admin/users` - List all users (paginated)
- `GET /api/v1/admin/users/:id` - Get user details
- `POST /api/v1/admin/users/:id/suspend` - Suspend user
- `POST /api/v1/admin/users/:id/activate` - Activate user
- `GET /api/v1/admin/drivers/pending` - Pending driver approvals
- `POST /api/v1/admin/drivers/:id/approve` - Approve driver
- `POST /api/v1/admin/drivers/:id/reject` - Reject driver
- `GET /api/v1/admin/rides/recent` - Recent rides (monitoring)
- `GET /api/v1/admin/rides/stats` - Ride statistics

**Features**:
- **Dashboard Stats**:
  - User statistics (total, riders, drivers, active)
  - All-time ride statistics
  - Today's ride statistics
  - Revenue metrics

- **User Management**:
  - List all users with pagination
  - View user details
  - Suspend/activate user accounts

- **Driver Management**:
  - List drivers pending approval
  - Approve driver applications
  - Reject driver applications

- **Ride Monitoring**:
  - Recent rides (last 50)
  - Ride statistics with date filters
  - Revenue tracking

- **Analytics**:
  - Total rides / completed / cancelled / active
  - Total revenue
  - Average fare
  - User growth metrics

**Security**:
- All endpoints require JWT authentication
- `RequireAdmin()` middleware checks role
- Only users with role="admin" can access

**Dashboard Response**:
```json
{
  "users": {
    "total_users": 1250,
    "total_riders": 1000,
    "total_drivers": 250,
    "active_users": 950
  },
  "rides": {
    "total_rides": 5000,
    "completed_rides": 4500,
    "cancelled_rides": 400,
    "active_rides": 100,
    "total_revenue": 125000.50,
    "avg_fare": 25.00
  },
  "today_rides": {
    "total_rides": 150,
    "completed_rides": 120,
    "cancelled_rides": 25,
    "active_rides": 5,
    "total_revenue": 3500.00,
    "avg_fare": 23.33
  }
}
```

---

## Database Schema

### users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    phone_number VARCHAR,
    first_name VARCHAR,
    last_name VARCHAR,
    role VARCHAR NOT NULL,  -- 'rider', 'driver', 'admin'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### drivers
```sql
CREATE TABLE drivers (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    license_number VARCHAR,
    vehicle_model VARCHAR,
    vehicle_plate VARCHAR,
    vehicle_color VARCHAR,
    vehicle_year INTEGER,
    is_available BOOLEAN DEFAULT false,
    is_online BOOLEAN DEFAULT false,
    rating DECIMAL(3,2),
    total_rides INTEGER DEFAULT 0,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### rides
```sql
CREATE TABLE rides (
    id UUID PRIMARY KEY,
    rider_id UUID REFERENCES users(id),
    driver_id UUID REFERENCES drivers(id),
    status VARCHAR,  -- 'requested', 'accepted', 'in_progress', 'completed', 'cancelled'
    pickup_latitude DECIMAL,
    pickup_longitude DECIMAL,
    pickup_address TEXT,
    dropoff_latitude DECIMAL,
    dropoff_longitude DECIMAL,
    dropoff_address TEXT,
    estimated_distance DECIMAL,
    estimated_duration INTEGER,
    estimated_fare DECIMAL,
    actual_distance DECIMAL,
    actual_duration INTEGER,
    final_fare DECIMAL,
    surge_multiplier DECIMAL DEFAULT 1.0,
    requested_at TIMESTAMP,
    accepted_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    cancellation_reason TEXT,
    rating INTEGER,  -- 1-5 stars
    feedback TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### wallets
```sql
CREATE TABLE wallets (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) UNIQUE,
    balance DECIMAL(10,2) DEFAULT 0.00,
    currency VARCHAR DEFAULT 'USD',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### payments
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    ride_id UUID REFERENCES rides(id),
    rider_id UUID REFERENCES users(id),
    driver_id UUID REFERENCES users(id),
    amount DECIMAL(10,2),
    currency VARCHAR DEFAULT 'USD',
    payment_method VARCHAR,  -- 'wallet', 'stripe'
    status VARCHAR,  -- 'pending', 'completed', 'failed', 'refunded'
    stripe_payment_id VARCHAR,
    stripe_charge_id VARCHAR,
    metadata JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### wallet_transactions
```sql
CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY,
    wallet_id UUID REFERENCES wallets(id),
    type VARCHAR,  -- 'credit', 'debit'
    amount DECIMAL(10,2),
    description TEXT,
    reference_type VARCHAR,  -- 'ride', 'topup', 'payout', 'refund'
    reference_id UUID,
    balance_before DECIMAL(10,2),
    balance_after DECIMAL(10,2),
    created_at TIMESTAMP
);
```

### notifications
```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    type VARCHAR,  -- 'ride_requested', 'ride_accepted', etc.
    channel VARCHAR,  -- 'push', 'sms', 'email'
    title VARCHAR,
    body TEXT,
    data JSONB,
    status VARCHAR,  -- 'pending', 'sent', 'failed'
    scheduled_at TIMESTAMP,
    sent_at TIMESTAMP,
    read_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### driver_locations
```sql
CREATE TABLE driver_locations (
    id UUID PRIMARY KEY,
    driver_id UUID REFERENCES drivers(id),
    latitude DECIMAL,
    longitude DECIMAL,
    heading DECIMAL,
    speed DECIMAL,
    accuracy DECIMAL,
    created_at TIMESTAMP
);
```

### favorite_locations
```sql
CREATE TABLE favorite_locations (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    name VARCHAR,  -- 'Home', 'Work', etc.
    address TEXT,
    latitude DECIMAL,
    longitude DECIMAL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

---

## Redis Data Structures

### 1. GeoSpatial Index (Driver Locations)
**Key**: `drivers:geo:index`
**Type**: Sorted Set (ZSET) with geospatial encoding
**Purpose**: Find nearby drivers efficiently

```bash
# Add driver location
GEOADD drivers:geo:index -74.0060 40.7128 "driver-uuid-1"

# Find drivers within 10km
GEORADIUS drivers:geo:index -74.0060 40.7128 10 km WITHDIST ASC COUNT 5

# Remove offline driver
ZREM drivers:geo:index "driver-uuid-1"
```

### 2. Chat History
**Key**: `ride:chat:{rideID}`
**Type**: List
**TTL**: 24 hours
**Purpose**: Store in-app chat messages

```bash
# Add message
RPUSH ride:chat:uuid-123 '{"from":"rider","msg":"On my way","ts":1699123456}'

# Get all messages
LRANGE ride:chat:uuid-123 0 -1

# Set expiry
EXPIRE ride:chat:uuid-123 86400
```

### 3. Driver Location Cache
**Key**: `driver:location:{driverID}`
**Type**: String (JSON)
**TTL**: 5 minutes
**Purpose**: Cache latest driver location

```bash
# Set location
SET driver:location:uuid-456 '{"lat":40.7128,"lng":-74.0060,"ts":1699123456}' EX 300

# Get location
GET driver:location:uuid-456
```

---

## Environment Variables

### Required for All Services
```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ride_hailing
JWT_SECRET=your-secret-key-change-in-production
```

### Redis Configuration
```bash
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
```

### Payments Service (Port 8084)
```bash
STRIPE_API_KEY=sk_test_51xxxxx...  # Get from Stripe Dashboard
```

### Notifications Service (Port 8085)
```bash
# Firebase (Optional - for push notifications)
FIREBASE_CREDENTIALS_PATH=/path/to/firebase-credentials.json

# Twilio (Optional - for SMS)
TWILIO_ACCOUNT_SID=ACxxxxxxxxx
TWILIO_AUTH_TOKEN=xxxxxxxxx
TWILIO_FROM_NUMBER=+1234567890

# SMTP (Optional - for email)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your@email.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=noreply@ridehailing.com
SMTP_FROM_NAME=RideHailing
```

---

## Quick Start Commands

### Local Development

```bash
# Install dependencies
go mod download

# Build all services
go build -o bin/auth ./cmd/auth
go build -o bin/rides ./cmd/rides
go build -o bin/geo ./cmd/geo
go build -o bin/payments ./cmd/payments
go build -o bin/notifications ./cmd/notifications
go build -o bin/realtime ./cmd/realtime
go build -o bin/mobile ./cmd/mobile
go build -o bin/admin ./cmd/admin

# Run a single service
./bin/auth

# Or use go run
go run cmd/auth/main.go
```

### Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f payments-service

# Stop all services
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```

### Health Checks

```bash
# Check all services
curl http://localhost:8081/healthz  # Auth
curl http://localhost:8082/healthz  # Rides
curl http://localhost:8083/healthz  # Geo
curl http://localhost:8084/healthz  # Payments
curl http://localhost:8085/healthz  # Notifications
curl http://localhost:8086/healthz  # Real-time
curl http://localhost:8087/healthz  # Mobile
curl http://localhost:8088/healthz  # Admin
```

---

## Testing

### Integration Testing Flow

1. **Register a rider**
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

2. **Login and get token**
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@test.com",
    "password": "password123"
  }'

# Save the token!
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

3. **Top up wallet**
```bash
curl -X POST http://localhost:8084/api/v1/wallet/topup \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50.00,
    "stripe_payment_method": "pm_card_visa"
  }'
```

4. **Request a ride**
```bash
curl -X POST http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "pickup_latitude": 40.7128,
    "pickup_longitude": -74.0060,
    "pickup_address": "New York, NY",
    "dropoff_latitude": 40.7589,
    "dropoff_longitude": -73.9851,
    "dropoff_address": "Times Square, NY"
  }'

# Save ride ID!
RIDE_ID="uuid-from-response"
```

5. **Connect to WebSocket (driver)**
```javascript
const ws = new WebSocket('ws://localhost:8086/ws?token=DRIVER_TOKEN');

ws.onopen = () => {
  // Join ride room
  ws.send(JSON.stringify({
    type: 'join_ride',
    payload: { ride_id: RIDE_ID }
  }));
};

ws.onmessage = (event) => {
  console.log('Message:', JSON.parse(event.data));
};
```

6. **Accept ride (driver)**
```bash
curl -X POST http://localhost:8082/api/v1/rides/$RIDE_ID/accept \
  -H "Authorization: Bearer $DRIVER_TOKEN"
```

7. **Complete ride (driver)**
```bash
curl -X POST http://localhost:8082/api/v1/rides/$RIDE_ID/complete \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_distance": 5.2,
    "actual_duration": 15
  }'
```

8. **Process payment**
```bash
curl -X POST http://localhost:8084/api/v1/payments/process \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ride_id": "'$RIDE_ID'",
    "amount": 15.50,
    "payment_method": "wallet"
  }'
```

9. **Rate ride (rider)**
```bash
curl -X POST http://localhost:8082/api/v1/rides/$RIDE_ID/rate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 5,
    "feedback": "Great driver!"
  }'
```

10. **Get receipt**
```bash
curl http://localhost:8087/api/v1/rides/$RIDE_ID/receipt \
  -H "Authorization: Bearer $TOKEN"
```

---

## Monitoring

### Prometheus Metrics

All services expose Prometheus metrics at `/metrics`:

```bash
# Example metrics
http_requests_total{service="auth",method="POST",endpoint="/api/v1/auth/login"}
http_request_duration_seconds{service="payments",method="POST",endpoint="/api/v1/payments/process"}
```

### Grafana

Access at: http://localhost:3000
- Username: admin
- Password: admin

**Pre-configured dashboards**:
- Service health overview
- Request latency by endpoint
- Error rates
- Database connection pool status

---

## Common Issues & Solutions

### 1. Firebase credentials not found
**Error**: `Failed to initialize Firebase`
**Solution**: Set `FIREBASE_CREDENTIALS_PATH` or leave empty to disable push notifications

### 2. Stripe webhook signature fails
**Error**: `Invalid signature`
**Solution**: Use Stripe CLI for local testing:
```bash
stripe listen --forward-to localhost:8084/api/v1/webhooks/stripe
```

### 3. Driver not found in geo search
**Error**: `No drivers available`
**Solution**:
- Ensure driver updated location recently (5min TTL)
- Check driver status is "available"
- Verify driver is within 10km radius

### 4. WebSocket connection drops
**Error**: `Connection closed unexpectedly`
**Solution**:
- Check ping/pong heartbeat (60s interval)
- Verify JWT token is valid
- Check for network issues

### 5. Payment processing fails
**Error**: `Insufficient balance`
**Solution**:
- Check wallet balance first
- Top up wallet before ride
- Or use Stripe payment method

---

## Security Best Practices

### JWT Tokens
- ✅ Tokens expire after 24 hours
- ✅ Refresh tokens available
- ✅ Tokens include user_id and role
- ⚠️ Change JWT_SECRET in production

### Password Security
- ✅ bcrypt hashing (cost=10)
- ✅ Passwords never stored in plaintext
- ✅ Passwords never returned in API responses

### API Security
- ✅ All endpoints require authentication (except login/register)
- ✅ Admin endpoints require admin role
- ✅ Input validation on all endpoints
- ⚠️ Add rate limiting in production
- ⚠️ Add request size limits

### Payment Security
- ✅ Stripe handles card data (PCI compliant)
- ✅ Webhook signature verification
- ✅ Idempotency keys for payments
- ⚠️ Use production Stripe keys in production

---

## Production Deployment Checklist

### Before Going Live

- [ ] Rotate all API keys and secrets
- [ ] Change JWT_SECRET to strong random value
- [ ] Use production Stripe API keys
- [ ] Set up Firebase production project
- [ ] Configure production SMTP credentials
- [ ] Enable HTTPS/TLS on all services
- [ ] Set up API Gateway (Kong/Nginx)
- [ ] Configure rate limiting (per user/IP)
- [ ] Set up CORS properly
- [ ] Enable database backups
- [ ] Set up log aggregation (ELK/Datadog)
- [ ] Configure error alerting (PagerDuty/Opsgenie)
- [ ] Load testing (100+ concurrent rides)
- [ ] Security audit & penetration testing
- [ ] Create runbook for on-call engineers
- [ ] Set up disaster recovery plan
- [ ] Configure auto-scaling policies
- [ ] Monitor database connection pool
- [ ] Set up uptime monitoring

---

## Performance Optimization Tips

### Database
- Use connection pooling (already configured)
- Add indexes on frequently queried columns
- Use read replicas for heavy read workloads
- Consider partitioning large tables (rides, notifications)

### Redis
- Enable persistence (RDB + AOF)
- Set maxmemory policy (allkeys-lru)
- Use Redis Cluster for high availability
- Monitor memory usage

### API Performance
- Enable gzip compression
- Use CDN for static assets
- Implement response caching (Redis)
- Optimize database queries (EXPLAIN)
- Add pagination to all list endpoints

### WebSocket Performance
- Use Redis Pub/Sub for multi-instance hubs
- Monitor connection count
- Implement reconnection logic in clients
- Consider using message queue for reliability

---

## Support & Documentation

- **PROGRESS.md** - Development progress and features completed
- **ROADMAP.md** - Future features and roadmap
- **README.md** - Project overview and quick start
- **This file** - Complete implementation reference

For questions or issues, refer to the documentation above or check the source code comments.

---

**Last Updated**: 2025-11-05
**Version**: 1.0.0 (Phase 1 Complete)
**Status**: Production Ready
