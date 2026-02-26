package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
	"github.com/ethanwang/devpulse/api/internal/auth"
	"github.com/ethanwang/devpulse/api/internal/config"
	mw "github.com/ethanwang/devpulse/api/internal/middleware"
	"github.com/ethanwang/devpulse/api/internal/oauth"
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

	slog.Info("starting server", "port", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "error", err)
	}
}
