package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope wraps all API responses.
type Envelope struct {
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError holds a stable error code and human-readable message.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaginatedEnvelope is used for list endpoints.
type PaginatedEnvelope struct {
	Data      interface{} `json:"data"`
	Total     int         `json:"total"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
	RequestID string      `json:"request_id,omitempty"`
}

// requestID extracts the X-Request-ID header set by middleware.
func requestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return c.GetHeader("X-Request-ID")
}

// OK sends a 200 response with a data payload.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Data: data, RequestID: requestID(c)})
}

// Created sends a 201 response with a data payload.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Data: data, RequestID: requestID(c)})
}

// NoContent sends a 204 response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Paginated sends a 200 paginated response.
func Paginated(c *gin.Context, data interface{}, total, limit, offset int) {
	c.JSON(http.StatusOK, PaginatedEnvelope{
		Data:      data,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
		RequestID: requestID(c),
	})
}

// BadRequest sends a 400 error.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Envelope{
		Error:     &APIError{Code: "BAD_REQUEST", Message: msg},
		RequestID: requestID(c),
	})
}

// Unauthorized sends a 401 error.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Envelope{
		Error:     &APIError{Code: "UNAUTHORIZED", Message: msg},
		RequestID: requestID(c),
	})
}

// Forbidden sends a 403 error.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Envelope{
		Error:     &APIError{Code: "FORBIDDEN", Message: msg},
		RequestID: requestID(c),
	})
}

// NotFound sends a 404 error.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Envelope{
		Error:     &APIError{Code: "NOT_FOUND", Message: msg},
		RequestID: requestID(c),
	})
}

// Conflict sends a 409 error.
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, Envelope{
		Error:     &APIError{Code: "CONFLICT", Message: msg},
		RequestID: requestID(c),
	})
}

// UnprocessableEntity sends a 422 error.
func UnprocessableEntity(c *gin.Context, msg string) {
	c.JSON(http.StatusUnprocessableEntity, Envelope{
		Error:     &APIError{Code: "UNPROCESSABLE_ENTITY", Message: msg},
		RequestID: requestID(c),
	})
}

// InternalServerError sends a 500 error.
func InternalServerError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Envelope{
		Error:     &APIError{Code: "INTERNAL_ERROR", Message: msg},
		RequestID: requestID(c),
	})
}
