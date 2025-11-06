package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/validation"
)

// ValidateRequest is a generic middleware that validates request body against a struct
// Usage: router.POST("/endpoint", middleware.ValidateRequest(&validation.CreateRideRequest{}), handler)
func ValidateRequest(requestType interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a new instance of the request type
		req := requestType

		// Bind JSON to the request struct
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate the struct
		if err := validation.ValidateStruct(req); err != nil {
			if valErr, ok := err.(*validation.ValidationError); ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":  "Validation failed",
					"fields": valErr.Errors,
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Validation failed",
					"details": err.Error(),
				})
			}
			c.Abort()
			return
		}

		// Store validated request in context for handler to use
		c.Set("validatedRequest", req)
		c.Next()
	}
}

// ValidateJSON validates JSON request body and binds it to the provided struct
// This is a helper function to be used within handlers
func ValidateJSON(c *gin.Context, req interface{}) error {
	// Bind JSON to the request struct
	if err := c.ShouldBindJSON(req); err != nil {
		return err
	}

	// Validate the struct
	return validation.ValidateStruct(req)
}

// ValidateQuery validates query parameters against a struct
func ValidateQuery(c *gin.Context, req interface{}) error {
	// Bind query parameters to the request struct
	if err := c.ShouldBindQuery(req); err != nil {
		return err
	}

	// Validate the struct
	return validation.ValidateStruct(req)
}

// ValidateURI validates URI parameters against a struct
func ValidateURI(c *gin.Context, req interface{}) error {
	// Bind URI parameters to the request struct
	if err := c.ShouldBindUri(req); err != nil {
		return err
	}

	// Validate the struct
	return validation.ValidateStruct(req)
}

// RespondWithValidationError sends a standardized validation error response
func RespondWithValidationError(c *gin.Context, err error) {
	if valErr, ok := err.(*validation.ValidationError); ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Validation failed",
			"fields": valErr.Errors,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}
}

// ValidateAndBind validates and binds request to the provided struct
// Returns true if validation passes, false otherwise (and sends error response)
func ValidateAndBind(c *gin.Context, req interface{}) bool {
	if err := ValidateJSON(c, req); err != nil {
		RespondWithValidationError(c, err)
		return false
	}
	return true
}

// ValidateAndBindQuery validates and binds query parameters to the provided struct
func ValidateAndBindQuery(c *gin.Context, req interface{}) bool {
	if err := ValidateQuery(c, req); err != nil {
		RespondWithValidationError(c, err)
		return false
	}
	return true
}

// GetValidatedRequest retrieves the validated request from context
// This is used when ValidateRequest middleware is applied
func GetValidatedRequest(c *gin.Context) (interface{}, bool) {
	req, exists := c.Get("validatedRequest")
	return req, exists
}

// Example usage in handler:
/*
func CreateRideHandler(c *gin.Context) {
	var req validation.CreateRideRequest

	// Method 1: Using ValidateAndBind helper
	if !middleware.ValidateAndBind(c, &req) {
		return // Error response already sent
	}

	// Or Method 2: Manual validation
	if err := middleware.ValidateJSON(c, &req); err != nil {
		middleware.RespondWithValidationError(c, err)
		return
	}

	// Process the validated request
	// ...
}
*/

// ValidateContentType ensures request has correct content type
func ValidateContentType(contentType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != contentType {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error":    "Unsupported content type",
				"expected": contentType,
				"received": c.ContentType(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ValidateJSONContentType ensures request has application/json content type
func ValidateJSONContentType() gin.HandlerFunc {
	return ValidateContentType("application/json")
}

// MaxBodySize limits the request body size
func MaxBodySize(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		// Try to read one byte beyond the limit to check if body is too large
		var jsonData interface{}
		decoder := json.NewDecoder(c.Request.Body)
		if err := decoder.Decode(&jsonData); err != nil {
			if err.Error() == "http: request body too large" {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error":          "Request body too large",
					"max_size_bytes": maxSize,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
