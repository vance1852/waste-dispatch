package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/response"
	"github.com/vance1852/waste-dispatch/internal/repository"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// PointHandler handles collection-point HTTP endpoints.
type PointHandler struct {
	svc *service.PointService
}

// NewPointHandler creates a new PointHandler.
func NewPointHandler(svc *service.PointService) *PointHandler {
	return &PointHandler{svc: svc}
}

type createPointRequest struct {
	Name            string                  `json:"name" binding:"required"`
	Address         string                  `json:"address" binding:"required"`
	Latitude        float64                 `json:"latitude"`
	Longitude       float64                 `json:"longitude"`
	District        string                  `json:"district"`
	WasteCategories []domain.WasteCategory  `json:"waste_categories"`
	CapacityKg      float64                 `json:"capacity_kg" binding:"required,gt=0"`
	Notes           string                  `json:"notes"`
}

// Create handles POST /api/v1/points.
func (h *PointHandler) Create(c *gin.Context) {
	var req createPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	p, err := h.svc.CreatePoint(c.Request.Context(), service.CreatePointRequest{
		Name:            req.Name,
		Address:         req.Address,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		District:        req.District,
		WasteCategories: req.WasteCategories,
		CapacityKg:      req.CapacityKg,
		Notes:           req.Notes,
	})
	if err != nil {
		switch err {
		case domain.ErrPointAlreadyExists:
			response.Conflict(c, "collection point already exists")
		default:
			response.InternalServerError(c, "failed to create point")
		}
		return
	}
	response.Created(c, p)
}

// Get handles GET /api/v1/points/:id.
func (h *PointHandler) Get(c *gin.Context) {
	id := c.Param("id")
	p, err := h.svc.GetPoint(c.Request.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrPointNotFound:
			response.NotFound(c, "collection point not found")
		default:
			response.InternalServerError(c, "failed to get point")
		}
		return
	}
	response.OK(c, p)
}

// List handles GET /api/v1/points.
func (h *PointHandler) List(c *gin.Context) {
	filter := repository.PointFilter{
		Status:   domain.PointStatus(c.Query("status")),
		District: c.Query("district"),
		Limit:    parseIntQuery(c, "limit", 20),
		Offset:   parseIntQuery(c, "offset", 0),
	}

	points, total, err := h.svc.ListPoints(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "failed to list points")
		return
	}
	if points == nil {
		points = []*domain.CollectionPoint{}
	}
	response.Paginated(c, points, total, filter.Limit, filter.Offset)
}

// ListOverCapacity handles GET /api/v1/points/over-capacity.
func (h *PointHandler) ListOverCapacity(c *gin.Context) {
	points, err := h.svc.ListOverCapacity(c.Request.Context(), 0.8)
	if err != nil {
		response.InternalServerError(c, "failed to list over-capacity points")
		return
	}
	if points == nil {
		points = []*domain.CollectionPoint{}
	}
	response.OK(c, points)
}

type updateLoadRequest struct {
	CurrentLoadKg float64 `json:"current_load_kg" binding:"gte=0"`
}

// UpdateLoad handles PUT /api/v1/points/:id/load.
func (h *PointHandler) UpdateLoad(c *gin.Context) {
	id := c.Param("id")
	var req updateLoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	p, err := h.svc.UpdateLoad(c.Request.Context(), service.UpdateLoadRequest{
		PointID:       id,
		CurrentLoadKg: req.CurrentLoadKg,
	})
	if err != nil {
		switch err {
		case domain.ErrPointNotFound:
			response.NotFound(c, "collection point not found")
		case domain.ErrPointVersionConflict:
			response.Conflict(c, "concurrent modification detected, please retry")
		default:
			response.InternalServerError(c, "failed to update load")
		}
		return
	}
	response.OK(c, p)
}

// Delete handles DELETE /api/v1/points/:id.
func (h *PointHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeletePoint(c.Request.Context(), id); err != nil {
		switch err {
		case domain.ErrPointNotFound:
			response.NotFound(c, "collection point not found")
		default:
			response.InternalServerError(c, "failed to delete point")
		}
		return
	}
	response.NoContent(c)
}
