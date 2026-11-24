package testkit

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"go-starter/internal/kernel/cache"
	"go-starter/internal/kernel/db"
)

// SetupPostgres 连接真实测试数据库、自动执行 goose 迁移并在测试结束时清理数据。
// 为防止误连开发/生产库，强制要求数据库名必须包含 "test"。
func SetupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/starter_test?sslmode=disable"
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if !strings.Contains(strings.ToLower(dbName), "test") {
		t.Fatalf("安全拦截：测试数据库名 %q 必须包含 'test' 字样，禁止在非测试库运行清表", dbName)
	}

	ctx := context.Background()
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

// SetupRedis 连接独立测试 Redis DB（默认 DB 15），测试结束自动 FlushDB。
func SetupRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	pass := os.Getenv("TEST_REDIS_PASSWORD")
	dbIdx := 15
	if raw := os.Getenv("TEST_REDIS_DB"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			dbIdx = n
		}
	}
	if dbIdx == 0 {
		t.Fatalf("安全拦截：禁止在默认 Redis DB 0 上运行集成测试，请使用独立 DB（如 15）")
	}

	ctx := context.Background()
	rdb, err := cache.NewRedis(ctx, addr, pass, dbIdx)
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
