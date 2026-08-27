package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"
const RequestIDHeader = "X-Request-ID"

// RequestID injects a unique request ID into every request context.
// It uses the incoming X-Request-ID header if present, otherwise generates one.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(c *gin.Context) string {
	if id, ok := c.Get(RequestIDKey); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
