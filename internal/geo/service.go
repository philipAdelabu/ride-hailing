package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
)

const (
	driverLocationPrefix = "driver:location:"
	driverLocationTTL    = 5 * time.Minute
	searchRadiusKm       = 10.0 // Search radius in kilometers
)

// DriverLocation represents a driver's location
type DriverLocation struct {
	DriverID  uuid.UUID `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

// Service handles geolocation business logic
type Service struct {
	redis *redisClient.Client
}

// NewService creates a new geo service
func NewService(redis *redisClient.Client) *Service {
	return &Service{redis: redis}
}

// UpdateDriverLocation updates a driver's current location
func (s *Service) UpdateDriverLocation(ctx context.Context, driverID uuid.UUID, latitude, longitude float64) error {
	location := &DriverLocation{
		DriverID:  driverID,
		Latitude:  latitude,
		Longitude: longitude,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(location)
	if err != nil {
		return common.NewInternalServerError("failed to marshal location data")
	}

	key := fmt.Sprintf("%s%s", driverLocationPrefix, driverID.String())
	if err := s.redis.SetWithExpiration(ctx, key, data, driverLocationTTL); err != nil {
		return common.NewInternalServerError("failed to update driver location")
	}

	return nil
}

// GetDriverLocation retrieves a driver's current location
func (s *Service) GetDriverLocation(ctx context.Context, driverID uuid.UUID) (*DriverLocation, error) {
	key := fmt.Sprintf("%s%s", driverLocationPrefix, driverID.String())
	data, err := s.redis.GetString(ctx, key)
	if err != nil {
		return nil, common.NewNotFoundError("driver location not found")
	}

	var location DriverLocation
	if err := json.Unmarshal([]byte(data), &location); err != nil {
		return nil, common.NewInternalServerError("failed to unmarshal location data")
	}

	return &location, nil
}

// FindNearbyDrivers finds drivers near a given location
func (s *Service) FindNearbyDrivers(ctx context.Context, latitude, longitude float64, maxDrivers int) ([]*DriverLocation, error) {
	// In a production system, this would use Redis GeoSpatial commands or a spatial database
	// For simplicity, we'll return nearby drivers (this is a placeholder implementation)

	// TODO: Implement proper geospatial search using Redis GEOADD/GEORADIUS
	// or PostgreSQL PostGIS for production

	return []*DriverLocation{}, nil
}

// CalculateDistance calculates distance between two coordinates in kilometers
func (s *Service) CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // km

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return math.Round(distance*100) / 100
}

// CalculateETA calculates estimated time of arrival in minutes
func (s *Service) CalculateETA(distance float64) int {
	const averageSpeed = 40.0 // km/h in city traffic
	eta := (distance / averageSpeed) * 60
	return int(math.Round(eta))
}
