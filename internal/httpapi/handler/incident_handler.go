package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/response"
	"github.com/vance1852/waste-dispatch/internal/middleware"
	"github.com/vance1852/waste-dispatch/internal/repository"
	"github.com/vance1852/waste-dispatch/internal/service"
)

// IncidentHandler handles incident-related HTTP endpoints.
type IncidentHandler struct {
	svc *service.IncidentService
}

// NewIncidentHandler creates a new IncidentHandler.
func NewIncidentHandler(svc *service.IncidentService) *IncidentHandler {
	return &IncidentHandler{svc: svc}
}

type reportIncidentRequest struct {
	Type        domain.IncidentType     `json:"type" binding:"required"`
	Severity    domain.IncidentSeverity `json:"severity"`
	PointID     string                  `json:"point_id"`
	VehicleID   string                  `json:"vehicle_id"`
	TaskID      string                  `json:"task_id"`
	Description string                  `json:"description" binding:"required"`
	OccurredAt  *time.Time              `json:"occurred_at"`
}

// Report handles POST /api/v1/incidents.
func (h *IncidentHandler) Report(c *gin.Context) {
	var req reportIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user := middleware.CurrentUser(c)
	reportedBy := ""
	if user != nil {
		reportedBy = user.ID
	}

	occurred := time.Now().UTC()
	if req.OccurredAt != nil {
		occurred = *req.OccurredAt
	}

	severity := req.Severity
	if severity == "" {
		severity = domain.IncidentSeverityMedium
	}

	incident, err := h.svc.ReportIncident(c.Request.Context(), service.ReportIncidentRequest{
		Type:        req.Type,
		Severity:    severity,
		PointID:     req.PointID,
		VehicleID:   req.VehicleID,
		TaskID:      req.TaskID,
		ReportedBy:  reportedBy,
		Description: req.Description,
		OccurredAt:  occurred,
	})
	if err != nil {
		response.InternalServerError(c, "failed to report incident")
		return
	}
	response.Created(c, incident)
}

// Get handles GET /api/v1/incidents/:id.
func (h *IncidentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	incident, err := h.svc.GetIncident(c.Request.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrIncidentNotFound:
			response.NotFound(c, "incident not found")
		default:
			response.InternalServerError(c, "failed to get incident")
		}
		return
	}
	response.OK(c, incident)
}

// List handles GET /api/v1/incidents.
func (h *IncidentHandler) List(c *gin.Context) {
	filter := repository.IncidentFilter{
		Status:    domain.IncidentStatus(c.Query("status")),
		Type:      domain.IncidentType(c.Query("type")),
		PointID:   c.Query("point_id"),
		VehicleID: c.Query("vehicle_id"),
		Limit:     parseIntQuery(c, "limit", 20),
		Offset:    parseIntQuery(c, "offset", 0),
	}

	incidents, total, err := h.svc.ListIncidents(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "failed to list incidents")
		return
	}
	if incidents == nil {
		incidents = []*domain.Incident{}
	}
	response.Paginated(c, incidents, total, filter.Limit, filter.Offset)
}

type assignIncidentRequest struct {
	AssignedTo string `json:"assigned_to" binding:"required"`
}

// Assign handles PUT /api/v1/incidents/:id/assign.
func (h *IncidentHandler) Assign(c *gin.Context) {
	id := c.Param("id")
	var req assignIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	incident, err := h.svc.AssignIncident(c.Request.Context(), service.AssignIncidentRequest{
		IncidentID: id,
		AssignedTo: req.AssignedTo,
	})
	if err != nil {
		switch err {
		case domain.ErrIncidentNotFound:
			response.NotFound(c, "incident not found")
		default:
			response.InternalServerError(c, "failed to assign incident")
		}
		return
	}
	response.OK(c, incident)
}

type resolveIncidentRequest struct {
	Resolution string `json:"resolution" binding:"required"`
}

// Resolve handles PUT /api/v1/incidents/:id/resolve.
func (h *IncidentHandler) Resolve(c *gin.Context) {
	id := c.Param("id")
	var req resolveIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user := middleware.CurrentUser(c)
	resolvedBy := ""
	if user != nil {
		resolvedBy = user.ID
	}

	incident, err := h.svc.ResolveIncident(c.Request.Context(), service.ResolveIncidentRequest{
		IncidentID: id,
		Resolution: req.Resolution,
		ResolvedBy: resolvedBy,
	})
	if err != nil {
		switch err {
		case domain.ErrIncidentNotFound:
			response.NotFound(c, "incident not found")
		case domain.ErrIncidentAlreadyResolved:
			response.UnprocessableEntity(c, "incident is already resolved")
		default:
			response.InternalServerError(c, "failed to resolve incident")
		}
		return
	}
	response.OK(c, incident)
}
