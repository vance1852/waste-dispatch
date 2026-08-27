package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/response"
	"github.com/vance1852/waste-dispatch/internal/repository"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// VehicleHandler handles vehicle-related HTTP endpoints.
type VehicleHandler struct {
	svc *service.VehicleService
}

// NewVehicleHandler creates a new VehicleHandler.
func NewVehicleHandler(svc *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

type createVehicleRequest struct {
	PlateNumber string              `json:"plate_number" binding:"required"`
	Type        domain.VehicleType  `json:"type" binding:"required"`
	CapacityKg  float64             `json:"capacity_kg" binding:"required,gt=0"`
	Notes       string              `json:"notes"`
}

// Create handles POST /api/v1/vehicles.
func (h *VehicleHandler) Create(c *gin.Context) {
	var req createVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	v, err := h.svc.CreateVehicle(c.Request.Context(), service.CreateVehicleRequest{
		PlateNumber: req.PlateNumber,
		Type:        req.Type,
		CapacityKg:  req.CapacityKg,
		Notes:       req.Notes,
	})
	if err != nil {
		switch err {
		case domain.ErrVehicleAlreadyExists:
			response.Conflict(c, "vehicle with this plate number already exists")
		default:
			response.InternalServerError(c, "failed to create vehicle")
		}
		return
	}
	response.Created(c, v)
}

// Get handles GET /api/v1/vehicles/:id.
func (h *VehicleHandler) Get(c *gin.Context) {
	id := c.Param("id")
	v, err := h.svc.GetVehicle(c.Request.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrVehicleNotFound:
			response.NotFound(c, "vehicle not found")
		default:
			response.InternalServerError(c, "failed to get vehicle")
		}
		return
	}
	response.OK(c, v)
}

// List handles GET /api/v1/vehicles.
func (h *VehicleHandler) List(c *gin.Context) {
	filter := repository.VehicleFilter{
		Status: domain.VehicleStatus(c.Query("status")),
		Type:   domain.VehicleType(c.Query("type")),
		Limit:  parseIntQuery(c, "limit", 20),
		Offset: parseIntQuery(c, "offset", 0),
	}

	vehicles, total, err := h.svc.ListVehicles(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "failed to list vehicles")
		return
	}
	if vehicles == nil {
		vehicles = []*domain.Vehicle{}
	}
	response.Paginated(c, vehicles, total, filter.Limit, filter.Offset)
}

type assignDriverRequest struct {
	DriverID string `json:"driver_id" binding:"required"`
}

// AssignDriver handles PUT /api/v1/vehicles/:id/assign.
func (h *VehicleHandler) AssignDriver(c *gin.Context) {
	id := c.Param("id")
	var req assignDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	v, err := h.svc.AssignDriver(c.Request.Context(), id, req.DriverID)
	if err != nil {
		switch err {
		case domain.ErrVehicleNotFound:
			response.NotFound(c, "vehicle not found")
		case domain.ErrVehicleInvalidTransition:
			response.UnprocessableEntity(c, "vehicle cannot be assigned in its current state")
		case domain.ErrVehicleVersionConflict:
			response.Conflict(c, "concurrent modification detected, please retry")
		default:
			response.InternalServerError(c, "failed to assign driver")
		}
		return
	}
	response.OK(c, v)
}

// Release handles PUT /api/v1/vehicles/:id/release.
func (h *VehicleHandler) Release(c *gin.Context) {
	id := c.Param("id")
	v, err := h.svc.ReleaseVehicle(c.Request.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrVehicleNotFound:
			response.NotFound(c, "vehicle not found")
		case domain.ErrVehicleInvalidTransition:
			response.UnprocessableEntity(c, "vehicle cannot be released in its current state")
		case domain.ErrVehicleVersionConflict:
			response.Conflict(c, "concurrent modification detected, please retry")
		default:
			response.InternalServerError(c, "failed to release vehicle")
		}
		return
	}
	response.OK(c, v)
}

// Delete handles DELETE /api/v1/vehicles/:id.
func (h *VehicleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteVehicle(c.Request.Context(), id); err != nil {
		switch err {
		case domain.ErrVehicleNotFound:
			response.NotFound(c, "vehicle not found")
		default:
			response.InternalServerError(c, "failed to delete vehicle")
		}
		return
	}
	response.NoContent(c)
}

func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	v := c.Query(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
