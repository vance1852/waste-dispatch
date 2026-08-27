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

// TaskHandler handles collection-task HTTP endpoints.
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

type createTaskRequest struct {
	PointID     string               `json:"point_id" binding:"required"`
	VehicleID   string               `json:"vehicle_id" binding:"required"`
	DriverID    string               `json:"driver_id"`
	Priority    domain.TaskPriority  `json:"priority"`
	ScheduledAt time.Time            `json:"scheduled_at" binding:"required"`
	Notes       string               `json:"notes"`
}

// Create handles POST /api/v1/tasks.
func (h *TaskHandler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user := middleware.CurrentUser(c)
	createdBy := ""
	if user != nil {
		createdBy = user.ID
	}

	task, err := h.svc.CreateTask(c.Request.Context(), service.CreateTaskRequest{
		PointID:     req.PointID,
		VehicleID:   req.VehicleID,
		DriverID:    req.DriverID,
		Priority:    req.Priority,
		ScheduledAt: req.ScheduledAt,
		Notes:       req.Notes,
		CreatedBy:   createdBy,
	})
	if err != nil {
		switch err {
		case domain.ErrPointNotFound:
			response.NotFound(c, "collection point not found")
		case domain.ErrVehicleNotFound:
			response.NotFound(c, "vehicle not found")
		case domain.ErrVehicleNotAvailable:
			response.UnprocessableEntity(c, "vehicle is not available for dispatch")
		default:
			response.InternalServerError(c, "failed to create task")
		}
		return
	}
	response.Created(c, task)
}

// Get handles GET /api/v1/tasks/:id.
func (h *TaskHandler) Get(c *gin.Context) {
	id := c.Param("id")
	task, err := h.svc.GetTask(c.Request.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrTaskNotFound:
			response.NotFound(c, "task not found")
		default:
			response.InternalServerError(c, "failed to get task")
		}
		return
	}
	response.OK(c, task)
}

// List handles GET /api/v1/tasks.
func (h *TaskHandler) List(c *gin.Context) {
	filter := repository.TaskFilter{
		Status:    domain.TaskStatus(c.Query("status")),
		VehicleID: c.Query("vehicle_id"),
		DriverID:  c.Query("driver_id"),
		PointID:   c.Query("point_id"),
		Limit:     parseIntQuery(c, "limit", 20),
		Offset:    parseIntQuery(c, "offset", 0),
	}

	tasks, total, err := h.svc.ListTasks(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "failed to list tasks")
		return
	}
	if tasks == nil {
		tasks = []*domain.CollectionTask{}
	}
	response.Paginated(c, tasks, total, filter.Limit, filter.Offset)
}

// Start handles PUT /api/v1/tasks/:id/start.
func (h *TaskHandler) Start(c *gin.Context) {
	id := c.Param("id")
	task, err := h.svc.StartTask(c.Request.Context(), id)
	if err != nil {
		handleTaskError(c, err)
		return
	}
	response.OK(c, task)
}

type completeTaskRequest struct {
	CollectedKg float64 `json:"collected_kg" binding:"gte=0"`
}

// Complete handles PUT /api/v1/tasks/:id/complete.
func (h *TaskHandler) Complete(c *gin.Context) {
	id := c.Param("id")
	var req completeTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	task, err := h.svc.CompleteTask(c.Request.Context(), id, req.CollectedKg)
	if err != nil {
		handleTaskError(c, err)
		return
	}
	response.OK(c, task)
}

type failTaskRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// Fail handles PUT /api/v1/tasks/:id/fail.
func (h *TaskHandler) Fail(c *gin.Context) {
	id := c.Param("id")
	var req failTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	task, err := h.svc.FailTask(c.Request.Context(), id, req.Reason)
	if err != nil {
		handleTaskError(c, err)
		return
	}
	response.OK(c, task)
}

// Cancel handles PUT /api/v1/tasks/:id/cancel.
func (h *TaskHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	task, err := h.svc.CancelTask(c.Request.Context(), id)
	if err != nil {
		handleTaskError(c, err)
		return
	}
	response.OK(c, task)
}

func handleTaskError(c *gin.Context, err error) {
	switch err {
	case domain.ErrTaskNotFound:
		response.NotFound(c, "task not found")
	case domain.ErrTaskInvalidTransition:
		response.UnprocessableEntity(c, "invalid task state transition")
	case domain.ErrTaskVersionConflict:
		response.Conflict(c, "concurrent modification detected, please retry")
	case domain.ErrTaskAlreadyTerminal:
		response.UnprocessableEntity(c, "task is already in a terminal state")
	default:
		response.InternalServerError(c, "task operation failed")
	}
}
