package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// HealthCheck returns a health check handler
func HealthCheck(serviceName, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{
			Status:  "healthy",
			Service: serviceName,
			Version: version,
		})
	}
}

// LivenessProbe returns a simple liveness check
// This endpoint indicates whether the service is running
// It should always return 200 OK unless the service is completely broken
func LivenessProbe(serviceName, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{
			Status:  "alive",
			Service: serviceName,
			Version: version,
		})
	}
}

// ReadinessProbe returns a readiness check with dependency validation
// This endpoint indicates whether the service is ready to accept traffic
// It checks critical dependencies (database, redis, etc.)
func ReadinessProbe(serviceName, version string, checks map[string]func() error) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "ready"
		checkResults := make(map[string]string)
		allHealthy := true

		for name, checkFunc := range checks {
			if err := checkFunc(); err != nil {
				checkResults[name] = "unhealthy: " + err.Error()
				status = "not ready"
				allHealthy = false
			} else {
				checkResults[name] = "healthy"
			}
		}

		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, HealthResponse{
			Status:  status,
			Service: serviceName,
			Version: version,
			Checks:  checkResults,
		})
	}
}

// HealthCheckWithDeps returns a health check handler with dependency checks
func HealthCheckWithDeps(serviceName, version string, checks map[string]func() error) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "healthy"
		checkResults := make(map[string]string)

		for name, checkFunc := range checks {
			if err := checkFunc(); err != nil {
				checkResults[name] = "unhealthy: " + err.Error()
				status = "unhealthy"
			} else {
				checkResults[name] = "healthy"
			}
		}

		statusCode := http.StatusOK
		if status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, HealthResponse{
			Status:  status,
			Service: serviceName,
			Version: version,
			Checks:  checkResults,
		})
	}
}
