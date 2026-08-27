package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/response"
	"github.com/vance1852/waste-dispatch/internal/middleware"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// CreditHandler handles resident-credit HTTP endpoints.
type CreditHandler struct {
	svc *service.CreditService
}

// NewCreditHandler creates a new CreditHandler.
func NewCreditHandler(svc *service.CreditService) *CreditHandler {
	return &CreditHandler{svc: svc}
}

// GetBalance handles GET /api/v1/credits/:resident_id/balance.
func (h *CreditHandler) GetBalance(c *gin.Context) {
	residentID := c.Param("resident_id")
	credit, err := h.svc.GetBalance(c.Request.Context(), residentID)
	if err != nil {
		response.InternalServerError(c, "failed to get balance")
		return
	}
	response.OK(c, credit)
}

// ListTransactions handles GET /api/v1/credits/:resident_id/transactions.
func (h *CreditHandler) ListTransactions(c *gin.Context) {
	residentID := c.Param("resident_id")
	limit := parseIntQuery(c, "limit", 20)
	offset := parseIntQuery(c, "offset", 0)

	txs, total, err := h.svc.ListTransactions(c.Request.Context(), residentID, limit, offset)
	if err != nil {
		response.InternalServerError(c, "failed to list transactions")
		return
	}
	if txs == nil {
		txs = []*domain.CreditTransaction{}
	}
	response.Paginated(c, txs, total, limit, offset)
}

type earnCreditRequest struct {
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	RefID          string `json:"ref_id"`
	Description    string `json:"description"`
}

// Earn handles POST /api/v1/credits/:resident_id/earn.
func (h *CreditHandler) Earn(c *gin.Context) {
	residentID := c.Param("resident_id")
	var req earnCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user := middleware.CurrentUser(c)
	createdBy := ""
	if user != nil {
		createdBy = user.ID
	}

	tx, err := h.svc.EarnCredit(c.Request.Context(), service.EarnCreditRequest{
		ResidentID:     residentID,
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
		RefID:          req.RefID,
		Description:    req.Description,
		CreatedBy:      createdBy,
	})
	if err != nil {
		response.InternalServerError(c, "failed to earn credit")
		return
	}
	response.Created(c, tx)
}

type redeemCreditRequest struct {
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	RefID          string `json:"ref_id"`
	Description    string `json:"description"`
}

// Redeem handles POST /api/v1/credits/:resident_id/redeem.
func (h *CreditHandler) Redeem(c *gin.Context) {
	residentID := c.Param("resident_id")
	var req redeemCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user := middleware.CurrentUser(c)
	createdBy := ""
	if user != nil {
		createdBy = user.ID
	}

	tx, err := h.svc.RedeemCredit(c.Request.Context(), service.RedeemCreditRequest{
		ResidentID:     residentID,
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
		RefID:          req.RefID,
		Description:    req.Description,
		CreatedBy:      createdBy,
	})
	if err != nil {
		switch err {
		case domain.ErrInsufficientCredit:
			response.UnprocessableEntity(c, "insufficient credit balance")
		default:
			response.InternalServerError(c, "failed to redeem credit")
		}
		return
	}
	response.Created(c, tx)
}
