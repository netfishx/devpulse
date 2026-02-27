package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	riverlib "github.com/riverqueue/river"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/activity"
	"github.com/ethanwang/devpulse/api/internal/apperror"
	"github.com/ethanwang/devpulse/api/internal/auth"
	"github.com/ethanwang/devpulse/api/internal/config"
	"github.com/ethanwang/devpulse/api/internal/datasource"
	"github.com/ethanwang/devpulse/api/internal/github"
	mw "github.com/ethanwang/devpulse/api/internal/middleware"
	"github.com/ethanwang/devpulse/api/internal/oauth"
	riversetup "github.com/ethanwang/devpulse/api/internal/river"
	"github.com/ethanwang/devpulse/api/internal/summary"
)

func main() {
	cfg := config.Load()

	// Database (pgxpool)
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create connection pool", "error", err)
		return
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		return
	}
	slog.Info("database connected")

	// Dependency injection
	queries := dbgen.New(pool)

	// GitHub client
	ghClient := github.NewClient(nil)

	// River workers
	workers := riverlib.NewWorkers()
	ghSyncWorker := github.NewSyncWorker(queries, ghClient)
	riverlib.AddWorker(workers, ghSyncWorker)

	aggWorker := summary.NewAggregateWorker(queries)
	riverlib.AddWorker(workers, aggWorker)

	// River periodic jobs
	periodicJobs := []*riverlib.PeriodicJob{
		riverlib.NewPeriodicJob(
			riverlib.PeriodicInterval(1*time.Hour),
			func() (riverlib.JobArgs, *riverlib.InsertOpts) {
				return github.SyncArgs{}, nil
			},
			&riverlib.PeriodicJobOpts{RunOnStart: true},
		),
		riverlib.NewPeriodicJob(
			riverlib.PeriodicInterval(24*time.Hour),
			func() (riverlib.JobArgs, *riverlib.InsertOpts) {
				return summary.AggregateArgs{}, nil
			},
			nil, // Don't run on start — aggregation is for yesterday's data
		),
	}

	// Create and start River client
	riverClient, err := riversetup.NewClient(pool, workers, periodicJobs)
	if err != nil {
		slog.Error("failed to create river client", "error", err)
		return
	}
	if err := riverClient.Start(context.Background()); err != nil {
		slog.Error("failed to start river client", "error", err)
		return
	}
	defer riverClient.Stop(context.Background()) //nolint:errcheck
	slog.Info("river started")

	authSvc := auth.NewService(queries, cfg.JWTSecret)
	authHandler := auth.NewHandler(authSvc)

	oauthSvc := oauth.NewService(queries, oauth.GitHubConfig{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		CallbackURL:  cfg.GitHubCallbackURL,
	})
	oauthHandler := oauth.NewHandler(oauthSvc)

	// Echo
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.HTTPErrorHandler = apperror.ErrorHandler(false)

	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	api := e.Group("/api")
	authHandler.RegisterPublicRoutes(api)

	protected := api.Group("")
	protected.Use(mw.JWTAuth(cfg.JWTSecret))
	authHandler.RegisterProtectedRoutes(protected)
	oauthHandler.RegisterRoutes(protected)

	activitySvc := activity.NewService(queries)
	activityHandler := activity.NewHandler(activitySvc)
	activityHandler.RegisterRoutes(protected)

	summarySvc := summary.NewService(queries)
	summaryHandler := summary.NewHandler(summarySvc)
	summaryHandler.RegisterRoutes(protected)

	dsSvc := datasource.NewService(queries)
	dsHandler := datasource.NewHandler(dsSvc)
	dsHandler.RegisterRoutes(protected)

	slog.Info("starting server", "port", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "error", err)
	}
}
