package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// RequestLogger logs HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
		}

		reqLogger := logger.WithContext(c.Request.Context())

		if len(c.Errors) > 0 {
			reqLogger.Error("Request completed with errors", append(fields, zap.String("errors", c.Errors.String()))...)
		} else {
			reqLogger.Info("Request completed", fields...)
		}
	}
}
