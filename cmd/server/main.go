// Package main is the entry point for the Prompt Compression Engine.
// It wires all dependencies together and starts the HTTP server with graceful shutdown.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/n3055/backend-project/internal/api"
	"github.com/n3055/backend-project/internal/config"
	"github.com/n3055/backend-project/internal/middleware"
	"github.com/n3055/backend-project/internal/service"
	"github.com/n3055/backend-project/internal/store"
	"github.com/n3055/backend-project/pkg/logger"
)

func main() {
	// --- Load Configuration ---
	cfg := config.Load()
	log := logger.New()

	log.Info("starting prompt compression engine",
		"port", cfg.Server.Port,
		"max_keywords", cfg.Compressor.MaxKeywords,
	)

	// --- Initialize Dependencies (Dependency Injection) ---
	// Storage layer.
	sessionStore := store.NewMemoryStore()

	// Service layer.
	compressor := service.NewCompressor(cfg.Compressor)
	cache := service.NewCache()
	conversationSvc := service.NewConversationService(sessionStore, compressor, cache, log)

	// API layer.
	handler := api.NewHandler(conversationSvc, log)
	limiter := middleware.NewRateLimiter(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.BurstSize)
	router := api.NewRouter(handler, log, limiter)

	// --- HTTP Server ---
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// --- Graceful Shutdown ---
	// Start server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt signal or server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("received shutdown signal", "signal", sig.String())
	case err := <-errCh:
		log.Error("server error", "error", err)
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}

	// Graceful shutdown with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	log.Info("shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped gracefully")
}
