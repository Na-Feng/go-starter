package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	LogLevel           string
	LogFormat          string
	DatabaseURL        string
	DBMaxConns         int32
	DBMinConns         int32
	AutoMigrate        bool
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	CORSAllowedOrigins []string
	RequestTimeout     time.Duration
	RateLimitPerMin    int64
	ShutdownTimeout    time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	UploadDir          string
	MaxUploadMB        int64
	MaxLargeUploadMB   int64
	ImageMaxWidth      int
	ImageJPEGQuality   int
	WorkerConcurrency  int
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "dev"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/starter?sslmode=disable"),
		DBMaxConns:         int32(getIntEnv("DB_MAX_CONNS", 20)),
		DBMinConns:         int32(getIntEnv("DB_MIN_CONNS", 2)),
		AutoMigrate:        getBoolEnv("AUTO_MIGRATE", false),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getIntEnv("REDIS_DB", 0),
		JWTSecret:          getEnv("JWT_SECRET", "dev-insecure-secret-change-in-prod"),
		JWTAccessTTL:       getDurationEnv("JWT_ACCESS_TTL", 2*time.Hour),
		JWTRefreshTTL:      getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		CORSAllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
		RequestTimeout:     getDurationEnv("REQUEST_TIMEOUT", 15*time.Second),
		RateLimitPerMin:    int64(getIntEnv("RATE_LIMIT_PER_MIN", 120)),
		ShutdownTimeout:    getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:        getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:       getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
		UploadDir:          getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadMB:        int64(getIntEnv("MAX_UPLOAD_MB", 10)),
		MaxLargeUploadMB:   int64(getIntEnv("MAX_LARGE_UPLOAD_MB", 500)),
		ImageMaxWidth:      getIntEnv("IMAGE_MAX_WIDTH", 1920),
		ImageJPEGQuality:   getIntEnv("IMAGE_JPEG_QUALITY", 80),
		WorkerConcurrency:  getIntEnv("WORKER_CONCURRENCY", 5),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return errors.New("HTTP_ADDR cannot be empty")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL cannot be empty")
	}
	if c.AppEnv == "prod" && (c.JWTSecret == "" || c.JWTSecret == "dev-insecure-secret-change-in-prod") {
		return errors.New("JWT_SECRET must be set in prod")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("SHUTDOWN_TIMEOUT must be positive")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getSliceEnv(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	return fallback
}
