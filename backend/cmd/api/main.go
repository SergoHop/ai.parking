package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SergoHop/ai.parking/backend/internal/auth"
	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/repository/postgres"
	httptransport "github.com/SergoHop/ai.parking/backend/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	created, err := auth.EnsureAdmin(ctx, db, cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		return err
	}
	if created {
		slog.Info("admin user created", "username", cfg.AdminUsername)
	}

	tokens := auth.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	router := httptransport.NewRouter(cfg, httptransport.Deps{
		DB:      db,
		Lots:    db,
		Spots:   db,
		Cameras: db,
		Users:   db,
		Tokens:  tokens,
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("http server started", "port", cfg.HTTPPort, "env", cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("server stopped")
	return nil
}
