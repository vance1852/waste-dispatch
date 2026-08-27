package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vance1852/waste-dispatch/internal/audit"
	"github.com/vance1852/waste-dispatch/internal/config"
	"github.com/vance1852/waste-dispatch/internal/httpapi"
	reposqlite "github.com/vance1852/waste-dispatch/internal/repository/sqlite"
	"github.com/vance1852/waste-dispatch/internal/service"
	sqlitestore "github.com/vance1852/waste-dispatch/internal/storage/sqlite"
	"github.com/vance1852/waste-dispatch/internal/worker"
)

func main() {
	cfg := config.Load()

	// Configure logger.
	logger := buildLogger(cfg.Log)
	log.Logger = logger

	logger.Info().Msg("starting waste-dispatch server")

	// Open database.
	db, err := sqlitestore.Open(cfg.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to open database")
	}
	defer db.Close()

	// Run migrations.
	if err := sqlitestore.RunMigrations(db, cfg.Database.MigrationsPath); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}
	logger.Info().Msg("database migrations applied")

	// Wire up repositories.
	userRepo := reposqlite.NewUserRepository(db)
	vehicleRepo := reposqlite.NewVehicleRepository(db)
	pointRepo := reposqlite.NewPointRepository(db)
	taskRepo := reposqlite.NewTaskRepository(db)
	incidentRepo := reposqlite.NewIncidentRepository(db)
	creditRepo := reposqlite.NewCreditRepository(db)
	sessionRepo := reposqlite.NewSessionRepository(db)
	auditRepo := reposqlite.NewAuditRepository(db)

	// Wire up auditor.
	auditor := audit.NewAuditor(auditRepo, logger)
	_ = auditor // Available for use in services.

	// Wire up services.
	authSvc := service.NewAuthService(userRepo, sessionRepo, cfg.Auth, logger)
	vehicleSvc := service.NewVehicleService(vehicleRepo, logger)
	pointSvc := service.NewPointService(pointRepo, logger)
	taskSvc := service.NewTaskService(taskRepo, vehicleRepo, pointRepo, logger)
	incidentSvc := service.NewIncidentService(incidentRepo, logger)
	creditSvc := service.NewCreditService(creditRepo, logger)

	// Build HTTP server.
	srv := httpapi.NewServer(
		authSvc, vehicleSvc, pointSvc, taskSvc, incidentSvc, creditSvc,
		db, logger, cfg.Log.Level == "debug",
	)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      srv.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start workers.
	ctx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	recoveryWorker := worker.NewTaskRecoveryWorker(taskSvc, 5*time.Minute, 4*time.Hour, logger)
	incidentWorker := worker.NewIncidentWorker(pointSvc, incidentSvc, 10*time.Minute, 0.9, logger)

	go recoveryWorker.Run(ctx)
	go incidentWorker.Run(ctx)

	// Start HTTP server.
	go func() {
		logger.Info().Str("addr", httpServer.Addr).Msg("HTTP server listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info().Str("signal", sig.String()).Msg("shutting down")

	// Cancel workers first.
	cancelWorkers()

	// Graceful HTTP shutdown.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("forced server shutdown")
	}

	logger.Info().Msg("server stopped")
}

func buildLogger(cfg config.LogConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.Pretty {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().Timestamp().Logger()
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
