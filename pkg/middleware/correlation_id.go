package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// CorrelationIDHeader is the header name for correlation ID
	CorrelationIDHeader = "X-Request-ID"
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey = "correlation_id"
)

// CorrelationID middleware generates or extracts correlation ID for request tracing
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get correlation ID from header
		correlationID := c.GetHeader(CorrelationIDHeader)

		// If not provided, generate a new UUID
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Store in context for use by handlers
		c.Set(CorrelationIDKey, correlationID)

		// Add to response headers
		c.Writer.Header().Set(CorrelationIDHeader, correlationID)

		c.Next()
	}
}

// GetCorrelationID extracts correlation ID from gin context
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(CorrelationIDKey); exists {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	return ""
}
