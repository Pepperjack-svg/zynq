package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/zynqcloud/go-storage/internal/config"
	"github.com/zynqcloud/go-storage/internal/handler"
	"github.com/zynqcloud/go-storage/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	if cfg.ServiceToken == "" {
		logger.Warn("SERVICE_TOKEN is not set — all requests will be accepted (dev mode only)")
	}

	backend, err := store.NewLocal(cfg.StoragePath)
	if err != nil {
		logger.Error("failed to initialise storage backend", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.New(cfg, backend, logger),
		// Large timeouts accommodate slow disks and very large files.
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		logger.Info("storage service starting", "port", cfg.Port, "root", cfg.StoragePath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// shutdownSignals is defined in signals.go (os.Interrupt) and extended by
	// signals_unix.go (+ SIGTERM) via build tags — no OS-specific imports here.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, shutdownSignals...)
	<-quit

	logger.Info("shutdown signal received — draining connections")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
	logger.Info("storage service stopped")
}
