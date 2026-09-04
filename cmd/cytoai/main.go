// Command cytoai is the Cyto AI API server entry point. It wires config ->
// Postgres -> risk.Scorer -> api.Server and listens for HTTP traffic.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codercollo/cytoai/internal/api"
	"github.com/codercollo/cytoai/internal/auth"
	"github.com/codercollo/cytoai/internal/config"
	"github.com/codercollo/cytoai/internal/db"
	"github.com/codercollo/cytoai/internal/db/queries"
	"github.com/codercollo/cytoai/internal/risk"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "cytoai: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/config.local.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	store := queries.New(pool)

	mlClient := risk.NewMLClient(cfg.MLBaseURL, nil)
	scorer, err := risk.NewScorer(mlClient, risk.Weights{BHI: cfg.Weights.BHI, RRI: cfg.Weights.RRI})
	if err != nil {
		return fmt.Errorf("construct scorer: %w", err)
	}

	authenticator := auth.New(store)

	server := api.NewServer(api.Config{
		Store:  store,
		Scorer: scorer,
		Auth:   authenticator,
		Logger: logger,
		PingDB: func(ctx context.Context) error {
			return db.Ready(ctx, pool)
		},
		MLReady:            mlClient.Ready,
		AnomalyDetector:    mlClient,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
	})

	httpSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received; draining in-flight requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
