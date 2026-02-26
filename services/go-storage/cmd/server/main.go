package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/zynqcloud/go-storage/internal/cleanup"
	"github.com/zynqcloud/go-storage/internal/config"
	"github.com/zynqcloud/go-storage/internal/handler"
	"github.com/zynqcloud/go-storage/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "err", err)
		os.Exit(1)
	}

	backend, err := store.NewLocal(cfg.StoragePath)
	if err != nil {
		logger.Error("failed to initialise storage backend", "err", err)
		os.Exit(1)
	}

	// Root context — cancelled when a shutdown signal arrives.
	// All long-running background goroutines receive this context so they
	// stop cleanly without needing their own signal wiring.
	ctx, cancel := context.WithCancel(context.Background())

	// Session cleanup goroutine reclaims disk space from abandoned uploads.
	// A client that calls InitUpload then disconnects (crash, timeout, network
	// drop) leaves a session directory that would otherwise live forever.
	var cleanupDone <-chan struct{}
	if cfg.SessionTTLHours > 0 {
		uploadsDir := filepath.Join(cfg.StoragePath, ".uploads")
		ttl := time.Duration(cfg.SessionTTLHours) * time.Hour
		cleanupDone = cleanup.RunPeriodic(ctx, uploadsDir, ttl, 1*time.Hour, logger)
		logger.Info("session cleanup enabled",
			"ttl_hours", cfg.SessionTTLHours,
			"uploads_dir", uploadsDir,
		)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.New(cfg, backend, logger),
		// ReadHeaderTimeout closes Slowloris: a client that never finishes
		// sending headers holds a goroutine until this fires.
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout and WriteTimeout are intentionally disabled (0 = no limit).
		//
		// Why: a 10 GB file uploaded at 1 MB/s takes ~170 minutes. Any finite
		// ReadTimeout will silently abort slow uploads. nginx enforces the outer
		// connection timeout via proxy_read_timeout 3600s — that is the correct
		// layer to set upper-bound limits. Go's ReadHeaderTimeout already
		// protects against Slowloris, so disabling ReadTimeout is safe.
		//
		// Same reasoning applies to WriteTimeout for large streaming downloads.
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		logger.Info("storage service starting",
			"port", cfg.Port,
			"root", cfg.StoragePath,
			"max_concurrent_uploads", cfg.MaxConcurrentUploads,
			"session_ttl_hours", cfg.SessionTTLHours,
		)
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

	// Cancel the root context first so background goroutines (cleanup, future
	// workers) stop accepting new work before the HTTP server drains.
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}

	// Wait for the cleanup goroutine to finish its current pass.
	if cleanupDone != nil {
		<-cleanupDone
	}

	logger.Info("storage service stopped")
}
