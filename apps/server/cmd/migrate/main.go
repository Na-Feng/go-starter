package main

import (
	"context"
	"log/slog"
	"os"

	"go-starter/internal/kernel/config"
	"go-starter/internal/kernel/db"
)

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, 2, 1)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, cmd); err != nil {
		slog.Error("migration failed", "command", cmd, "error", err)
		os.Exit(1)
	}
}
