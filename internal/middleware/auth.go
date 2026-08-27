package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/response"
	"github.com/vance1852/waste-dispatch/internal/service"
)

const (
	UserKey    = "current_user"
	SessionKey = "current_session"
)

// Auth returns a middleware that validates bearer tokens and injects the user into the context.
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		session, user, err := authSvc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrSessionExpired):
				response.Unauthorized(c, "session has expired")
			case errors.Is(err, domain.ErrSessionRevoked):
				response.Unauthorized(c, "session has been revoked")
			case errors.Is(err, domain.ErrUserInactive):
				response.Forbidden(c, "account is inactive")
			default:
				response.Unauthorized(c, "invalid token")
			}
			c.Abort()
			return
		}

		c.Set(UserKey, user)
		c.Set(SessionKey, session)
		c.Next()
	}
}

// RequireRole returns a middleware that restricts access to users with specific roles.
func RequireRole(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		for _, role := range roles {
			if user.Role == role {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "insufficient permissions")
		c.Abort()
	}
}

// CurrentUser retrieves the authenticated user from the context.
func CurrentUser(c *gin.Context) *domain.User {
	if v, ok := c.Get(UserKey); ok {
		if u, ok := v.(*domain.User); ok {
			return u
		}
	}
	return nil
}

// CurrentSession retrieves the session from the context.
func CurrentSession(c *gin.Context) *domain.Session {
	if v, ok := c.Get(SessionKey); ok {
		if s, ok := v.(*domain.Session); ok {
			return s
		}
	}
	return nil
}

// extractBearerToken pulls the token from the Authorization header.
func extractBearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
