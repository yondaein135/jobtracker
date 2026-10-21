// Command api is the composition root for the job tracker backend service.
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	httpadapter "job-tracker/backend/internal/adapters/inbound/http"
	"job-tracker/backend/internal/adapters/outbound/sqlite"
	"job-tracker/backend/internal/app"
)

type config struct {
	Host          string
	Port          string
	DBPath        string
	StaticDir     string
	CORSOrigins   []string
	SeedDummyData bool
	Env           string
}

func loadConfig() config {
	port := cmp.Or(os.Getenv("PORT"), "8080")
	dbPath := cmp.Or(os.Getenv("DB_PATH"), "./data/tracker.db")
	staticDir := cmp.Or(os.Getenv("STATIC_DIR"), "./static")
	corsStr := cmp.Or(os.Getenv("CORS_ORIGINS"), "http://localhost:5173,http://127.0.0.1:5173")
	seedEnv := cmp.Or(os.Getenv("SEED_DUMMY_DATA"), "0")
	appEnv := cmp.Or(os.Getenv("APP_ENV"), "development")

	seedDummy := seedEnv == "1" || seedEnv == "true" || strings.Contains(dbPath, "dummy")

	origins := strings.Split(corsStr, ",")

	return config{
		Host:          "",
		Port:          port,
		DBPath:        dbPath,
		StaticDir:     staticDir,
		CORSOrigins:   origins,
		SeedDummyData: seedDummy,
		Env:           appEnv,
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Stdout, os.Args[1:]); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_, _ = fmt.Fprintf(os.Stderr, "service error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, _ []string) error {
	cfg := loadConfig()

	var logHandler slog.Handler
	if cfg.Env == "production" {
		logHandler = slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		logHandler = slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)

	// Initialize Outbound SQLite Adapter
	dbConn, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer dbConn.Close()

	migrateCtx, cancelMigrate := context.WithTimeout(ctx, 30*time.Second)
	defer cancelMigrate()
	if err := sqlite.Migrate(migrateCtx, dbConn); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	if cfg.SeedDummyData {
		seedCtx, cancelSeed := context.WithTimeout(ctx, 30*time.Second)
		defer cancelSeed()
		if err := sqlite.SeedDummyData(seedCtx, dbConn); err != nil {
			logger.Warn("seed dummy data warning", "error", err)
		} else {
			logger.Info("dummy data checked/seeded", "db_path", cfg.DBPath)
		}
	}
	logger.Info("database ready", "db_path", cfg.DBPath)

	// Wire Outbound Repositories
	appRepo := sqlite.NewApplicationRepository(dbConn)
	analyticsRepo := sqlite.NewAnalyticsRepository(dbConn)
	healthChecker := sqlite.NewHealthChecker(dbConn)

	// Wire Core Application Services
	appService := app.NewApplicationService(appRepo)
	analyticsService := app.NewAnalyticsService(analyticsRepo)

	// Wire Inbound HTTP Adapter
	httpHandler := httpadapter.NewHandler(appService, analyticsService, healthChecker, logger)
	router := httpadapter.NewRouter(httpHandler, cfg.CORSOrigins, cfg.StaticDir, logger)

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:                addr,
		Handler:             router,
		ReadHeaderTimeout:   5 * time.Second,
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
		MaxHeaderBytes:      1 << 20,
		MaxHeaderValueCount: 1000,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", addr, "static_dir", cfg.StaticDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server stopped gracefully")
	return nil
}
