package testkit

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"go-starter/internal/kernel/cache"
	"go-starter/internal/kernel/config"
	"go-starter/internal/kernel/db"
)

// SetupPostgres 读取 configs/config.yaml（及本地 config.local.yaml / 环境变量覆盖），自动建测试库、执行迁移并在结束时清空业务表。
func SetupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config for integration test: %v", err)
	}
	dsn := cfg.TestDatabaseURL

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse test.postgres_dsn: %v", err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if !strings.Contains(strings.ToLower(dbName), "test") {
		t.Fatalf("安全拦截：测试数据库名 %q 必须包含 'test' 字样，禁止在非测试库运行清表", dbName)
	}

	ctx := context.Background()
	if err := db.EnsureDatabase(ctx, dsn); err != nil {
		t.Fatalf("ensure test postgres database (%s): %v", dbName, err)
	}
	pool, err := db.NewPool(ctx, dsn, 5, 1)
	if err != nil {
		t.Fatalf("connect test postgres (%s): %v", dbName, err)
	}

	if err := db.Migrate(ctx, pool, "up"); err != nil {
		pool.Close()
		t.Fatalf("migrate test postgres: %v", err)
	}

	truncate := func() {
		rows, err := pool.Query(ctx, `
			SELECT tablename FROM pg_tables
			WHERE schemaname = 'public'
			  AND tablename NOT IN ('goose_db_version', 'roles', 'permissions', 'role_permissions')
			ORDER BY tablename`)
		if err != nil {
			t.Fatalf("query public tables for truncate: %v", err)
		}
		var quoted []string
		for rows.Next() {
			var tbl string
			if err := rows.Scan(&tbl); err != nil {
				rows.Close()
				t.Fatalf("scan table name: %v", err)
			}
			quoted = append(quoted, pgx.Identifier{tbl}.Sanitize())
		}
		rows.Close()
		if len(quoted) == 0 {
			return
		}
		q := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(quoted, ", "))
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("truncate test tables (%s): %v", q, err)
		}
	}
	truncate()

	t.Cleanup(func() {
		truncate()
		pool.Close()
	})
	return pool
}

// SetupRedis 连接独立测试 Redis DB（读取 test.redis_db，默认 15），测试结束自动 FlushDB。
func SetupRedis(t *testing.T) *redis.Client {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config for redis test: %v", err)
	}
	if cfg.TestRedisDB == 0 {
		t.Fatalf("安全拦截：禁止在默认 Redis DB 0 上运行集成测试，请在 test.redis_db 使用独立 DB（如 15）")
	}

	ctx := context.Background()
	rdb, err := cache.NewRedis(ctx, cfg.TestRedisAddr, cfg.TestRedisPassword, cfg.TestRedisDB)
	if err != nil {
		t.Fatalf("connect test redis: %v", err)
	}
	_ = rdb.FlushDB(ctx).Err()

	t.Cleanup(func() {
		_ = rdb.FlushDB(ctx).Err()
		_ = rdb.Close()
	})
	return rdb
}
