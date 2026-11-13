package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/SergoHop/ai.parking/backend/internal/config"
	"github.com/SergoHop/ai.parking/backend/internal/model"
	"github.com/SergoHop/ai.parking/backend/internal/repository/postgres"
	"github.com/SergoHop/ai.parking/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	file := flag.String("file", "../data/lots/demo.json", "path to demo lot JSON")
	force := flag.Bool("force", false, "delete and recreate the demo lot if it already exists")
	flag.Parse()

	if err := run(*file, *force); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run(file string, force bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	var demo model.DemoLot
	if err := json.Unmarshal(raw, &demo); err != nil {
		return fmt.Errorf("parse seed file: %w", err)
	}
	if err := service.ValidateDemoLot(demo); err != nil {
		return fmt.Errorf("validate seed file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	lotID, created, err := db.SeedDemoLot(ctx, demo, force)
	if err != nil {
		return err
	}

	if !created {
		slog.Info("demo lot already exists; use -force to rebuild", "lot_id", lotID, "name", demo.Name)
		return nil
	}

	slog.Info("demo lot seeded",
		"lot_id", lotID,
		"name", demo.Name,
		"spots", len(demo.Spots),
		"camera", demo.Camera.Name,
	)
	return nil
}
