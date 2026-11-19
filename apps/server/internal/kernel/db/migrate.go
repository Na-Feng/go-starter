package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	dbassets "go-starter/db"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, command string) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	goose.SetBaseFS(dbassets.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	switch command {
	case "up":
		return goose.UpContext(ctx, sqlDB, "migrations")
	case "down":
		return goose.DownContext(ctx, sqlDB, "migrations")
	case "status":
		return goose.StatusContext(ctx, sqlDB, "migrations")
	default:
		return fmt.Errorf("unsupported migrate command: %s (use up, down, or status)", command)
	}
}
