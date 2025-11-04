# API Documentation

## Overview

This document provides detailed API documentation for the Ride Hailing Platform backend services.

Base URLs:
- Auth Service: `http://localhost:8081`
- Rides Service: `http://localhost:8082`
- Geo Service: `http://localhost:8083`

All API requests and responses use JSON format.

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
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
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
    "created_at": "2024-01-01T00:00:00Z"
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
  "pickup_longitude": -74.0060,
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
    "pickup_longitude": -74.0060,
    "pickup_address": "New York, NY",
    "dropoff_latitude": 40.7589,
    "dropoff_longitude": -73.9851,
    "dropoff_address": "Times Square, NY",
    "estimated_distance": 5.2,
    "estimated_duration": 18,
    "estimated_fare": 12.50,
    "surge_multiplier": 1.0,
    "requested_at": "2024-01-01T00:00:00Z"
  }
}
```

### GET /api/v1/rides/:id

Get ride details by ID.

**Response:** `200 OK`

### GET /api/v1/rides

Get user's ride history. Supports pagination.

**Query Parameters:**
- `page` (default: 1)
- `per_page` (default: 10, max: 100)

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
      "final_fare": 13.20,
      "completed_at": "2024-01-01T00:30:00Z"
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
  "longitude": -74.0060
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
    "longitude": -74.0060,
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

### POST /api/v1/geo/distance

Calculate distance and ETA between two points.

**Request Body:**
```json
{
  "from_latitude": 40.7128,
  "from_longitude": -74.0060,
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

- `400` - Bad Request: Invalid input data
- `401` - Unauthorized: Missing or invalid authentication
- `403` - Forbidden: Insufficient permissions
- `404` - Not Found: Resource not found
- `409` - Conflict: Resource already exists
- `500` - Internal Server Error: Server-side error

## Rate Limiting

Currently not implemented. Future versions will include rate limiting.

## Pagination

List endpoints support pagination with query parameters:
- `page`: Page number (default: 1)
- `per_page`: Items per page (default: 10, max: 100)

## Versioning

API version is included in the URL path: `/api/v1/...`
