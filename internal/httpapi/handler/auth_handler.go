package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/response"
	"github.com/vance1852/waste-dispatch/internal/middleware"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	res, err := h.svc.Login(c.Request.Context(), service.LoginRequest{
		Username:  req.Username,
		Password:  req.Password,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			response.Unauthorized(c, "invalid username or password")
		case domain.ErrUserInactive:
			response.Forbidden(c, "account is inactive or banned")
		default:
			response.InternalServerError(c, "login failed")
		}
		return
	}

	c.JSON(http.StatusOK, response.Envelope{
		Data: gin.H{
			"token":      res.Token,
			"session_id": res.SessionID,
			"expires_at": res.ExpiresAt,
			"user":       res.User,
		},
		RequestID: middleware.GetRequestID(c),
	})
}

type registerRequest struct {
	Username string      `json:"username" binding:"required,min=3,max=64"`
	Password string      `json:"password" binding:"required,min=6"`
	FullName string      `json:"full_name"`
	Phone    string      `json:"phone"`
	Email    string      `json:"email"`
	Role     domain.Role `json:"role"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user, err := h.svc.Register(c.Request.Context(), service.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Role:     req.Role,
	})
	if err != nil {
		switch err {
		case domain.ErrUserAlreadyExists:
			response.Conflict(c, "username already taken")
		default:
			response.InternalServerError(c, "registration failed")
		}
		return
	}

	response.Created(c, user)
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	token := extractToken(c)
	if err := h.svc.Logout(c.Request.Context(), token); err != nil {
		response.InternalServerError(c, "logout failed")
		return
	}
	response.NoContent(c)
}

// Me handles GET /api/v1/auth/me.
func (h *AuthHandler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		response.Unauthorized(c, "not authenticated")
		return
	}
	response.OK(c, user)
}

func extractToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}
