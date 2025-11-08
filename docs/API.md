# API Documentation

## Overview

This document provides detailed API documentation for the Ride Hailing Platform backend services.

Base URLs:

-   Auth Service: `http://localhost:8081`
-   Rides Service: `http://localhost:8082`
-   Geo Service: `http://localhost:8083`
-   Payments Service: `http://localhost:8084`
-   Notifications Service: `http://localhost:8085`
-   Real-time Service: `http://localhost:8086`
-   Mobile Service: `http://localhost:8087`
-   Admin Service: `http://localhost:8088`
-   Promos Service: `http://localhost:8089`
-   Scheduler Service: `http://localhost:8090`
-   Analytics Service: `http://localhost:8091`
-   Fraud Service: `http://localhost:8092`
-   ML ETA Service: `http://localhost:8093`

All API requests and responses use JSON format. All endpoints use `/api/v1/` prefix.

## Authentication

Most endpoints require authentication using JWT tokens. Include the token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

## Common Response Format

### Success Response

```json
{
  "success": true,
  "data": { ... }
}
```

### Error Response

```json
{
	"success": false,
	"error": {
		"code": 400,
		"message": "Error description"
	}
}
```

## Auth Service API

### POST /api/v1/auth/register

Register a new user (rider or driver).

**Request Body:**

```json
{
	"email": "user@example.com",
	"password": "password123",
	"phone_number": "+1234567890",
	"first_name": "John",
	"last_name": "Doe",
	"role": "rider"
}
```

**Response:** `201 Created`

```json
{
	"success": true,
	"data": {
		"id": "uuid",
		"email": "user@example.com",
		"phone_number": "+1234567890",
		"first_name": "John",
		"last_name": "Doe",
		"role": "rider",
		"is_active": true,
		"is_verified": false,
		"created_at": "2025-01-01T00:00:00Z",
		"updated_at": "2025-01-01T00:00:00Z"
	}
}
```

### POST /api/v1/auth/login

Authenticate and receive a JWT token.

**Request Body:**

```json
{
	"email": "user@example.com",
	"password": "password123"
}
```

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"user": {
			"id": "uuid",
			"email": "user@example.com",
			"first_name": "John",
			"last_name": "Doe",
			"role": "rider"
		},
		"token": "eyJhbGciOiJIUzI1NiIs..."
	}
}
```

### GET /api/v1/auth/profile

Get current user profile. Requires authentication.

**Headers:**

```
Authorization: Bearer <token>
```

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"id": "uuid",
		"email": "user@example.com",
		"phone_number": "+1234567890",
		"first_name": "John",
		"last_name": "Doe",
		"role": "rider",
		"is_active": true,
		"created_at": "2025-01-01T00:00:00Z"
	}
}
```

### PUT /api/v1/auth/profile

Update user profile. Requires authentication.

**Headers:**

```
Authorization: Bearer <token>
```

**Request Body:**

```json
{
	"first_name": "John",
	"last_name": "Smith",
	"phone_number": "+1234567890"
}
```

**Response:** `200 OK`

## Rides Service API

### POST /api/v1/rides

Create a new ride request. Requires rider authentication.

**Headers:**

```
Authorization: Bearer <token>
```

**Request Body:**

```json
{
	"pickup_latitude": 40.7128,
	"pickup_longitude": -74.006,
	"pickup_address": "New York, NY",
	"dropoff_latitude": 40.7589,
	"dropoff_longitude": -73.9851,
	"dropoff_address": "Times Square, NY"
}
```

**Response:** `201 Created`

```json
{
	"success": true,
	"data": {
		"id": "uuid",
		"rider_id": "uuid",
		"status": "requested",
		"pickup_latitude": 40.7128,
		"pickup_longitude": -74.006,
		"pickup_address": "New York, NY",
		"dropoff_latitude": 40.7589,
		"dropoff_longitude": -73.9851,
		"dropoff_address": "Times Square, NY",
		"estimated_distance": 5.2,
		"estimated_duration": 18,
		"estimated_fare": 12.5,
		"surge_multiplier": 1.0,
		"requested_at": "2025-01-01T00:00:00Z"
	}
}
```

### GET /api/v1/rides/:id

Get ride details by ID.

**Response:** `200 OK`

### GET /api/v1/rides

Get user's ride history. Supports pagination.

**Query Parameters:**

-   `page` (default: 1)
-   `per_page` (default: 10, max: 100)

**Response:** `200 OK`

```json
{
	"success": true,
	"data": [
		{
			"id": "uuid",
			"status": "completed",
			"pickup_address": "New York, NY",
			"dropoff_address": "Times Square, NY",
			"final_fare": 13.2,
			"completed_at": "2025-01-01T00:30:00Z"
		}
	]
}
```

### GET /api/v1/driver/rides/available

Get available ride requests for drivers.

**Headers:**

```
Authorization: Bearer <driver_token>
```

**Response:** `200 OK`

### POST /api/v1/driver/rides/:id/accept

Accept a ride request. Requires driver authentication.

**Response:** `200 OK`

### POST /api/v1/driver/rides/:id/start

Start an accepted ride. Requires driver authentication.

**Response:** `200 OK`

### POST /api/v1/driver/rides/:id/complete

Complete an in-progress ride. Requires driver authentication.

**Request Body:**

```json
{
	"actual_distance": 5.4
}
```

**Response:** `200 OK`

### POST /api/v1/rides/:id/cancel

Cancel a ride. Can be called by rider or driver.

**Request Body:**

```json
{
	"reason": "Change of plans"
}
```

**Response:** `200 OK`

### POST /api/v1/rides/:id/rate

Rate a completed ride. Requires rider authentication.

**Request Body:**

```json
{
	"rating": 5,
	"feedback": "Great driver!"
}
```

**Response:** `200 OK`

### GET /api/v1/rides/surge-info

Retrieve the current surge pricing information for a latitude/longitude pair. Requires authentication (rider or driver).

**Query Parameters:**

-   `lat` (required) – Pickup latitude
-   `lon` (required) – Pickup longitude

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"surge_multiplier": 1.4,
		"is_surge_active": true,
		"message": "Increased demand - Fares are slightly higher",
		"factors": {
			"demand_ratio": 1.8,
			"demand_surge": 1.8,
			"time_multiplier": 1.2,
			"day_multiplier": 1.0,
			"zone_multiplier": 1.1,
			"weather_factor": 1.0
		}
	}
}
```

## Mobile Service API

The mobile API consolidates rider-facing functionality such as ride history, favorites, and profile management. All endpoints require the `Authorization: Bearer <token>` header.

### GET /api/v1/rides/history

Retrieve ride history with rich filtering options.

**Query Parameters:**

-   `status` – Optional ride status filter (`completed`, `cancelled`, etc.)
-   `start_date` – Optional ISO date (`YYYY-MM-DD`)
-   `end_date` – Optional ISO date (`YYYY-MM-DD`)
-   `limit` – Number of records to return (default 20)
-   `offset` – Pagination offset (default 0)

**Response:** `200 OK`

```json
{
	"rides": [
		{
			"id": "uuid",
			"status": "completed",
			"pickup_address": "New York, NY",
			"dropoff_address": "Times Square, NY",
			"final_fare": 18.75,
			"completed_at": "2025-01-01T00:30:00Z"
		}
	],
	"total": 42,
	"limit": 20,
	"offset": 0
}
```

### GET /api/v1/rides/:id/receipt

Generate a detailed receipt for a completed ride (rider or driver).

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"ride_id": "uuid",
		"date": "2025-01-01T00:30:00Z",
		"pickup_address": "New York, NY",
		"dropoff_address": "Times Square, NY",
		"distance": 5.4,
		"duration": 19,
		"base_fare": 12.5,
		"surge_multiplier": 1.3,
		"final_fare": 16.25,
		"payment_method": "wallet",
		"rider_id": "uuid",
		"driver_id": "uuid"
	}
}
```

### Favorites Endpoints

#### POST /api/v1/favorites

Create a favorite location for the authenticated user.

**Request Body:**

```json
{
	"name": "Home",
	"address": "123 Main St, Springfield",
	"latitude": 40.7128,
	"longitude": -74.006
}
```

**Response:** `201 Created`

```json
{
	"id": "uuid",
	"user_id": "uuid",
	"name": "Home",
	"address": "123 Main St, Springfield",
	"latitude": 40.7128,
	"longitude": -74.006,
	"created_at": "2025-01-01T00:00:00Z",
	"updated_at": "2025-01-01T00:00:00Z"
}
```

#### GET /api/v1/favorites

List all favorite locations for the authenticated user.

**Response:** `200 OK`

```json
{
	"favorites": [
		{
			"id": "uuid",
			"name": "Home",
			"address": "123 Main St, Springfield",
			"latitude": 40.7128,
			"longitude": -74.006
		}
	]
}
```

#### GET /api/v1/favorites/:id

Fetch a single favorite location by ID. Returns `404` if it does not belong to the user.

#### PUT /api/v1/favorites/:id

Update a favorite location. Request body matches the create payload. Returns the updated favorite on success.

#### DELETE /api/v1/favorites/:id

Delete a favorite location. Returns:

```json
{
	"message": "Favorite location deleted"
}
```

### Profile Endpoints

#### GET /api/v1/profile

Retrieve the authenticated user's profile information.

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"id": "uuid",
		"email": "user@example.com",
		"first_name": "John",
		"last_name": "Doe",
		"phone_number": "+1234567890",
		"role": "rider"
	}
}
```

#### PUT /api/v1/profile

Update the authenticated user's profile.

**Request Body:**

```json
{
	"first_name": "John",
	"last_name": "Smith",
	"phone_number": "+1234567890"
}
```

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"message": "Profile updated successfully"
	}
}
```

## Geo Service API

### POST /api/v1/geo/location

Update driver's current location. Requires driver authentication.

**Headers:**

```
Authorization: Bearer <driver_token>
```

**Request Body:**

```json
{
	"latitude": 40.7128,
	"longitude": -74.006
}
```

**Response:** `200 OK`

### GET /api/v1/geo/drivers/:id/location

Get a driver's current location.

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"driver_id": "uuid",
		"latitude": 40.7128,
		"longitude": -74.006,
		"timestamp": "2025-01-01T00:00:00Z"
	}
}
```

### POST /api/v1/geo/distance

Calculate distance and ETA between two points.

**Request Body:**

```json
{
	"from_latitude": 40.7128,
	"from_longitude": -74.006,
	"to_latitude": 40.7589,
	"to_longitude": -73.9851
}
```

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"distance_km": 5.2,
		"eta_minutes": 18
	}
}
```

## Health Check Endpoints

All services expose the following endpoints:

### GET /healthz

Check service health status.

**Response:** `200 OK`

```json
{
	"status": "healthy",
	"service": "auth-service",
	"version": "1.0.0"
}
```

### GET /version

Get service version information.

**Response:** `200 OK`

```json
{
	"service": "auth-service",
	"version": "1.0.0"
}
```

### GET /metrics

Prometheus metrics endpoint.

## Error Codes

-   `400` - Bad Request: Invalid input data
-   `401` - Unauthorized: Missing or invalid authentication
-   `403` - Forbidden: Insufficient permissions
-   `404` - Not Found: Resource not found
-   `409` - Conflict: Resource already exists
-   `500` - Internal Server Error: Server-side error

## Payments Service API

### POST /api/v1/payments/process

Process a payment for a completed ride. Requires authentication.

**Request Body:**

```json
{
	"ride_id": "uuid",
	"amount": 25.5,
	"payment_method": "wallet",
	"stripe_payment_method_id": "pm_card_visa"
}
```

**Response:** `200 OK`

### POST /api/v1/wallet/topup

Add funds to user's wallet. Requires authentication.

**Request Body:**

```json
{
	"amount": 50.0,
	"stripe_payment_method": "pm_card_visa"
}
```

**Response:** `200 OK`

### GET /api/v1/wallet

Get current wallet balance. Requires authentication.

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"user_id": "uuid",
		"balance": 42.5,
		"currency": "USD"
	}
}
```

### GET /api/v1/wallet/transactions

Get wallet transaction history. Requires authentication.

**Response:** `200 OK`

### POST /api/v1/payments/refund

Process a refund for a cancelled ride. Requires authentication.

**Request Body:**

```json
{
	"ride_id": "uuid",
	"reason": "Ride cancelled by driver"
}
```

**Response:** `200 OK`

### POST /api/v1/payments/webhook

Stripe webhook endpoint for payment events. Internal use.

---

## Notifications Service API

### GET /api/v1/notifications

List user's notifications. Requires authentication.

**Query Parameters:**

-   `limit` (default: 20)
-   `offset` (default: 0)

**Response:** `200 OK`

### GET /api/v1/notifications/unread

Get count of unread notifications. Requires authentication.

**Response:** `200 OK`

```json
{
	"count": 5
}
```

### PUT /api/v1/notifications/:id/read

Mark a notification as read. Requires authentication.

**Response:** `200 OK`

### POST /api/v1/notifications/send

Send a notification (admin only).

**Request Body:**

```json
{
	"user_id": "uuid",
	"title": "Promo Alert",
	"message": "50% off your next ride!",
	"channels": ["push", "sms", "email"]
}
```

**Response:** `200 OK`

### POST /api/v1/notifications/schedule

Schedule a notification for later delivery (admin only).

**Request Body:**

```json
{
	"user_id": "uuid",
	"title": "Reminder",
	"message": "Your scheduled ride is in 30 minutes",
	"scheduled_at": "2025-01-01T10:00:00Z",
	"channels": ["push"]
}
```

**Response:** `200 OK`

### POST /api/v1/notifications/bulk

Send bulk notifications (admin only).

**Request Body:**

```json
{
	"user_ids": ["uuid1", "uuid2"],
	"title": "System Maintenance",
	"message": "Platform will be down for 1 hour",
	"channels": ["push", "email"]
}
```

**Response:** `200 OK`

---

## Real-time Service API

### GET /ws?token=<jwt_token>

Establish WebSocket connection for real-time updates.

**Message Types:**

-   `join_ride` - Join a ride room
-   `location_update` - Real-time location updates
-   `ride_status` - Ride status changes
-   `chat_message` - Send/receive chat messages
-   `typing` - Typing indicators

**Example Message:**

```json
{
	"type": "chat_message",
	"payload": {
		"ride_id": "uuid",
		"message": "I'm 5 minutes away"
	}
}
```

### POST /api/v1/broadcast

Internal endpoint for broadcasting messages to WebSocket clients.

---

## Admin Service API

All endpoints require admin authentication.

### GET /api/v1/admin/dashboard

Get dashboard overview with aggregated statistics.

**Response:** `200 OK`

```json
{
	"total_users": 1250,
	"total_drivers": 340,
	"total_rides": 8932,
	"active_rides": 42,
	"revenue_today": 4532.5,
	"revenue_total": 284120.75
}
```

### GET /api/v1/admin/users

List all users with filtering and pagination.

**Query Parameters:**

-   `role` - Filter by role (rider, driver, admin)
-   `status` - Filter by status (active, suspended)
-   `page` (default: 1)
-   `per_page` (default: 20)

**Response:** `200 OK`

### GET /api/v1/admin/users/:id

Get detailed user information.

**Response:** `200 OK`

### PUT /api/v1/admin/users/:id/suspend

Suspend a user account.

**Response:** `200 OK`

### PUT /api/v1/admin/users/:id/activate

Activate a suspended user account.

**Response:** `200 OK`

### GET /api/v1/admin/drivers/pending

Get drivers pending approval.

**Response:** `200 OK`

### PUT /api/v1/admin/drivers/:id/approve

Approve a driver application.

**Response:** `200 OK`

### GET /api/v1/admin/rides

Get rides with filtering options.

**Query Parameters:**

-   `status` - Filter by status
-   `start_date` - Filter by date range
-   `end_date` - Filter by date range
-   `page` (default: 1)
-   `per_page` (default: 20)

**Response:** `200 OK`

### GET /api/v1/admin/stats

Get detailed analytics and statistics.

**Query Parameters:**

-   `start_date` - Date range start
-   `end_date` - Date range end
-   `metric` - Specific metric to retrieve

**Response:** `200 OK`

---

## Promos Service API

### POST /api/v1/promos

Create a new promo code (admin only).

**Request Body:**

```json
{
	"code": "SUMMER50",
	"discount_type": "percentage",
	"discount_value": 50,
	"max_uses": 1000,
	"expires_at": "2025-12-31T23:59:59Z"
}
```

**Response:** `201 Created`

### GET /api/v1/promos

List all promo codes (admin only).

**Response:** `200 OK`

### POST /api/v1/promos/apply

Apply a promo code to a ride. Requires authentication.

**Request Body:**

```json
{
	"code": "SUMMER50",
	"ride_id": "uuid"
}
```

**Response:** `200 OK`

```json
{
	"success": true,
	"data": {
		"discount_amount": 12.5,
		"final_amount": 12.5
	}
}
```

### POST /api/v1/promos/validate

Validate a promo code without applying it.

**Request Body:**

```json
{
	"code": "SUMMER50"
}
```

**Response:** `200 OK`

### GET /api/v1/referral/code

Get user's referral code. Requires authentication.

**Response:** `200 OK`

```json
{
	"code": "REF-ABC123",
	"uses": 5,
	"bonus_earned": 25.0
}
```

### POST /api/v1/referral/apply

Apply a referral code. Requires authentication.

**Request Body:**

```json
{
	"referral_code": "REF-ABC123"
}
```

**Response:** `200 OK`

### GET /api/v1/ride-types

Get available ride types.

**Response:** `200 OK`

```json
{
	"ride_types": [
		{
			"id": "uuid",
			"name": "Economy",
			"base_fare": 5.0,
			"per_km_rate": 1.5,
			"per_minute_rate": 0.25
		},
		{
			"id": "uuid",
			"name": "Premium",
			"base_fare": 10.0,
			"per_km_rate": 2.5,
			"per_minute_rate": 0.5
		}
	]
}
```

---

## Scheduler Service API

### POST /api/v1/scheduled-rides

Schedule a ride for future pickup. Requires authentication.

**Request Body:**

```json
{
	"pickup_latitude": 40.7128,
	"pickup_longitude": -74.006,
	"pickup_address": "New York, NY",
	"dropoff_latitude": 40.7589,
	"dropoff_longitude": -73.9851,
	"dropoff_address": "Times Square, NY",
	"scheduled_at": "2025-01-02T14:30:00Z"
}
```

**Response:** `201 Created`

### GET /api/v1/scheduled-rides

List user's scheduled rides. Requires authentication.

**Response:** `200 OK`

### GET /api/v1/scheduled-rides/:id

Get scheduled ride details. Requires authentication.

**Response:** `200 OK`

### PUT /api/v1/scheduled-rides/:id

Update a scheduled ride. Requires authentication.

**Response:** `200 OK`

### DELETE /api/v1/scheduled-rides/:id

Cancel a scheduled ride. Requires authentication.

**Response:** `200 OK`

---

## Analytics Service API

All endpoints require admin authentication.

### GET /api/v1/analytics/overview

Get high-level business metrics and KPIs.

**Query Parameters:**

-   `start_date` - Date range start (ISO format)
-   `end_date` - Date range end (ISO format)

**Response:** `200 OK`

```json
{
	"total_rides": 8932,
	"completed_rides": 8124,
	"cancelled_rides": 808,
	"total_revenue": 284120.75,
	"average_fare": 31.8,
	"completion_rate": 91.0
}
```

### GET /api/v1/analytics/revenue

Get revenue analytics and trends.

**Query Parameters:**

-   `start_date` - Date range start
-   `end_date` - Date range end
-   `group_by` - Grouping (day, week, month)

**Response:** `200 OK`

### GET /api/v1/analytics/drivers

Get driver performance analytics.

**Response:** `200 OK`

### GET /api/v1/analytics/rides

Get ride analytics and patterns.

**Response:** `200 OK`

### GET /api/v1/analytics/demand

Get demand heat maps and patterns.

**Query Parameters:**

-   `latitude` - Center latitude
-   `longitude` - Center longitude
-   `radius` - Radius in km

**Response:** `200 OK`

### POST /api/v1/analytics/export

Export analytics data to CSV/JSON.

**Request Body:**

```json
{
	"report_type": "revenue",
	"start_date": "2025-01-01",
	"end_date": "2025-01-31",
	"format": "csv"
}
```

**Response:** `200 OK`

---

## Fraud Service API

All endpoints require authentication (admin for reports).

### POST /api/v1/fraud/check-ride

Check a ride for suspicious activity.

**Request Body:**

```json
{
	"ride_id": "uuid"
}
```

**Response:** `200 OK`

```json
{
	"risk_score": 0.25,
	"is_suspicious": false,
	"flags": []
}
```

### POST /api/v1/fraud/check-payment

Check a payment for fraud indicators.

**Request Body:**

```json
{
	"payment_id": "uuid",
	"amount": 125.5,
	"user_id": "uuid"
}
```

**Response:** `200 OK`

### POST /api/v1/fraud/check-user

Check user account for suspicious patterns.

**Request Body:**

```json
{
	"user_id": "uuid"
}
```

**Response:** `200 OK`

### GET /api/v1/fraud/reports

Get fraud detection reports (admin only).

**Query Parameters:**

-   `start_date` - Date range start
-   `end_date` - Date range end
-   `risk_level` - Filter by risk level (low, medium, high)

**Response:** `200 OK`

---

## ML ETA Service API

### POST /api/v1/eta/predict

Predict ETA for a route using machine learning.

**Request Body:**

```json
{
	"pickup_latitude": 40.7128,
	"pickup_longitude": -74.006,
	"dropoff_latitude": 40.7589,
	"dropoff_longitude": -73.9851,
	"distance_km": 5.2,
	"time_of_day": 14,
	"day_of_week": 2
}
```

**Response:** `200 OK`

```json
{
	"predicted_eta_minutes": 18,
	"confidence_score": 0.87,
	"factors": {
		"distance_impact": 0.65,
		"time_impact": 0.2,
		"traffic_impact": 0.15
	}
}
```

### POST /api/v1/eta/predict/batch

Batch ETA prediction for multiple routes.

**Request Body:**

```json
{
	"routes": [
		{
			"pickup_latitude": 40.7128,
			"pickup_longitude": -74.006,
			"dropoff_latitude": 40.7589,
			"dropoff_longitude": -73.9851
		}
	]
}
```

**Response:** `200 OK`

### POST /api/v1/eta/train

Trigger model retraining (admin only).

**Response:** `200 OK`

### GET /api/v1/eta/model/stats

Get ML model statistics.

**Response:** `200 OK`

```json
{
	"model_version": "v1.2",
	"training_samples": 45230,
	"accuracy": 0.872,
	"last_trained_at": "2025-01-08T00:00:00Z"
}
```

### GET /api/v1/eta/accuracy

Get model accuracy metrics.

**Response:** `200 OK`

### POST /api/v1/eta/tune

Fine-tune model hyperparameters (admin only).

**Response:** `200 OK`

### GET /api/v1/eta/analytics

Get prediction analytics and insights (admin only).

**Response:** `200 OK`

---

## Rate Limiting

Rate limiting is implemented using Redis-backed token bucket algorithm:

-   **Default**: 120 requests per minute per user
-   **Burst**: 40 additional requests
-   **Anonymous**: 60 requests per minute per IP

Rate limit headers are included in responses:

-   `X-RateLimit-Limit` - Maximum requests per window
-   `X-RateLimit-Remaining` - Remaining requests
-   `X-RateLimit-Reset` - Time until rate limit resets

**429 Too Many Requests** response when limit exceeded.

## Pagination

List endpoints support pagination with query parameters:

-   `page`: Page number (default: 1)
-   `per_page`: Items per page (default: 10, max: 100)

## Versioning

API version is included in the URL path: `/api/v1/...`
