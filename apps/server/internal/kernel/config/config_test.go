package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-starter/internal/kernel/config"
)

func TestYAMLConfigLayeredMergeEnvOverrideAndStrictTypoCheck(t *testing.T) {
	dir := t.TempDir()
	baseFile := filepath.Join(dir, "config.yaml")
	localFile := filepath.Join(dir, "config.local.yaml")

	baseYAML := `
app:
  env: dev
  log_level: info
http:
  addr: ":8080"
  rate_limit_per_min: 120
postgres:
  dsn: "postgres://postgres:postgres@localhost:5432/starter?sslmode=disable"
  max_conns: 20
`
	localYAML := `
http:
  addr: ":9090"
postgres:
  max_conns: 50
`
	if err := os.WriteFile(baseFile, []byte(baseYAML), 0o644); err != nil {
		t.Fatalf("write base yaml: %v", err)
	}
	if err := os.WriteFile(localFile, []byte(localYAML), 0o644); err != nil {
		t.Fatalf("write local yaml: %v", err)
	}

	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.LoadFromFiles(baseFile, localFile)
	if err != nil {
		t.Fatalf("LoadFromFiles failed: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected local yaml override :9090, got %q", cfg.HTTPAddr)
	}
	if cfg.RateLimitPerMin != 120 {
		t.Fatalf("expected base field rate_limit_per_min=120 preserved, got %d", cfg.RateLimitPerMin)
	}
	if cfg.DBMaxConns != 50 {
		t.Fatalf("expected local max_conns=50, got %d", cfg.DBMaxConns)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected env override LOG_LEVEL=debug, got %q", cfg.LogLevel)
	}

	typoFile := filepath.Join(dir, "typo.yaml")
	if err := os.WriteFile(typoFile, []byte("postgres:\n  max_conn: 10\n"), 0o644); err != nil {
		t.Fatalf("write typo yaml: %v", err)
	}
	_, typoErr := config.LoadFromFiles(typoFile)
	if typoErr == nil || !strings.Contains(typoErr.Error(), "field max_conn not found") {
		t.Fatalf("expected strict KnownFields error for max_conn typo, got: %v", typoErr)
	}
}

func TestRepositoryDefaultConfigYAMLLoadsCleanly(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected repository configs/config.yaml to load cleanly, got: %v", err)
	}
	if !cfg.AutoMigrate || cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected default config values: %+v", cfg)
	}
}
