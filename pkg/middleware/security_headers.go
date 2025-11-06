package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking by disallowing iframe embedding
		c.Writer.Header().Set("X-Frame-Options", "DENY")

		// Enable browser's XSS protection
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS and set HSTS (HTTP Strict Transport Security)
		// Only set in production; max-age is 1 year
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy - restrictive default
		// Adjust based on your application's needs
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		// Referrer Policy - don't send referrer to cross-origin requests
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature Policy)
		// Disable potentially dangerous features
		c.Writer.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")

		c.Next()
	}
}
