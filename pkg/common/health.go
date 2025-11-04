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
