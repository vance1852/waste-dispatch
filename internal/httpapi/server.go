package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/vance1852/waste-dispatch/internal/domain"
	"github.com/vance1852/waste-dispatch/internal/httpapi/handler"
	"github.com/vance1852/waste-dispatch/internal/middleware"
	"github.com/vance1852/waste-dispatch/internal/service"
	"github.com/vance1852/waste-dispatch/internal/storage/sqlite"
)

// Server holds all HTTP handler dependencies.
type Server struct {
	router   *gin.Engine
	authSvc  *service.AuthService
	handlers struct {
		auth     *handler.AuthHandler
		vehicle  *handler.VehicleHandler
		point    *handler.PointHandler
		task     *handler.TaskHandler
		incident *handler.IncidentHandler
		credit   *handler.CreditHandler
	}
	db  *sql.DB
	log zerolog.Logger
}

// NewServer creates and wires up the HTTP server.
func NewServer(
	authSvc *service.AuthService,
	vehicleSvc *service.VehicleService,
	pointSvc *service.PointService,
	taskSvc *service.TaskService,
	incidentSvc *service.IncidentService,
	creditSvc *service.CreditService,
	db *sql.DB,
	log zerolog.Logger,
	debug bool,
) *Server {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		router:  gin.New(),
		authSvc: authSvc,
		db:      db,
		log:     log,
	}

	s.handlers.auth = handler.NewAuthHandler(authSvc)
	s.handlers.vehicle = handler.NewVehicleHandler(vehicleSvc)
	s.handlers.point = handler.NewPointHandler(pointSvc)
	s.handlers.task = handler.NewTaskHandler(taskSvc)
	s.handlers.incident = handler.NewIncidentHandler(incidentSvc)
	s.handlers.credit = handler.NewCreditHandler(creditSvc)

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Logger(s.log))
	s.router.Use(gin.Recovery())
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	v1 := s.router.Group("/api/v1")

	// Public auth routes.
	auth := v1.Group("/auth")
	{
		auth.POST("/login", s.handlers.auth.Login)
		auth.POST("/register", s.handlers.auth.Register)
	}

	// Authenticated routes.
	protected := v1.Group("")
	protected.Use(middleware.Auth(s.authSvc))
	{
		// Auth.
		protected.POST("/auth/logout", s.handlers.auth.Logout)
		protected.GET("/auth/me", s.handlers.auth.Me)

		// Vehicles.
		vehicles := protected.Group("/vehicles")
		{
			vehicles.GET("", s.handlers.vehicle.List)
			vehicles.GET("/:id", s.handlers.vehicle.Get)
			vehicles.POST("", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.vehicle.Create)
			vehicles.PUT("/:id/assign", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.vehicle.AssignDriver)
			vehicles.PUT("/:id/release", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.vehicle.Release)
			vehicles.DELETE("/:id", middleware.RequireRole(domain.RoleAdmin), s.handlers.vehicle.Delete)
		}

		// Collection points.
		points := protected.Group("/points")
		{
			points.GET("", s.handlers.point.List)
			points.GET("/over-capacity", s.handlers.point.ListOverCapacity)
			points.GET("/:id", s.handlers.point.Get)
			points.POST("", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.point.Create)
			points.PUT("/:id/load", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator, domain.RoleDriver), s.handlers.point.UpdateLoad)
			points.DELETE("/:id", middleware.RequireRole(domain.RoleAdmin), s.handlers.point.Delete)
		}

		// Collection tasks.
		tasks := protected.Group("/tasks")
		{
			tasks.GET("", s.handlers.task.List)
			tasks.GET("/:id", s.handlers.task.Get)
			tasks.POST("", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.task.Create)
			tasks.PUT("/:id/start", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator, domain.RoleDriver), s.handlers.task.Start)
			tasks.PUT("/:id/complete", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator, domain.RoleDriver), s.handlers.task.Complete)
			tasks.PUT("/:id/fail", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator, domain.RoleDriver), s.handlers.task.Fail)
			tasks.PUT("/:id/cancel", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.task.Cancel)
		}

		// Incidents.
		incidents := protected.Group("/incidents")
		{
			incidents.GET("", s.handlers.incident.List)
			incidents.GET("/:id", s.handlers.incident.Get)
			incidents.POST("", s.handlers.incident.Report)
			incidents.PUT("/:id/assign", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.incident.Assign)
			incidents.PUT("/:id/resolve", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.incident.Resolve)
		}

		// Credits.
		credits := protected.Group("/credits")
		{
			credits.GET("/:resident_id/balance", s.handlers.credit.GetBalance)
			credits.GET("/:resident_id/transactions", s.handlers.credit.ListTransactions)
			credits.POST("/:resident_id/earn", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator), s.handlers.credit.Earn)
			credits.POST("/:resident_id/redeem", middleware.RequireRole(domain.RoleAdmin, domain.RoleOperator, domain.RoleResident), s.handlers.credit.Redeem)
		}
	}
}

// Handler returns the underlying http.Handler for use with net/http.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) healthCheck(c *gin.Context) {
	if err := sqlite.HealthCheck(s.db); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
