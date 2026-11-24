package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
	TestDatabaseURL    string
	TestRedisAddr      string
	TestRedisPassword  string
	TestRedisDB        int
}

type fileConfig struct {
	App struct {
		Env       string `yaml:"env"`
		LogLevel  string `yaml:"log_level"`
		LogFormat string `yaml:"log_format"`
	} `yaml:"app"`
	HTTP struct {
		Addr               string        `yaml:"addr"`
		CORSAllowedOrigins []string      `yaml:"cors_allowed_origins"`
		RequestTimeout     time.Duration `yaml:"request_timeout"`
		ReadTimeout        time.Duration `yaml:"read_timeout"`
		WriteTimeout       time.Duration `yaml:"write_timeout"`
		ShutdownTimeout    time.Duration `yaml:"shutdown_timeout"`
		RateLimitPerMin    int64         `yaml:"rate_limit_per_min"`
	} `yaml:"http"`
	Postgres struct {
		DSN         string `yaml:"dsn"`
		MaxConns    int32  `yaml:"max_conns"`
		MinConns    int32  `yaml:"min_conns"`
		AutoMigrate bool   `yaml:"auto_migrate"`
	} `yaml:"postgres"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	JWT struct {
		Secret     string        `yaml:"secret"`
		AccessTTL  time.Duration `yaml:"access_ttl"`
		RefreshTTL time.Duration `yaml:"refresh_ttl"`
	} `yaml:"jwt"`
	Storage struct {
		UploadDir        string `yaml:"upload_dir"`
		MaxUploadMB      int64  `yaml:"max_upload_mb"`
		MaxLargeUploadMB int64  `yaml:"max_large_upload_mb"`
		ImageMaxWidth    int    `yaml:"image_max_width"`
		ImageJPEGQuality int    `yaml:"image_jpeg_quality"`
	} `yaml:"storage"`
	Worker struct {
		Concurrency int `yaml:"concurrency"`
	} `yaml:"worker"`
	Test struct {
		PostgresDSN   string `yaml:"postgres_dsn"`
		RedisAddr     string `yaml:"redis_addr"`
		RedisPassword string `yaml:"redis_password"`
		RedisDB       int    `yaml:"redis_db"`
	} `yaml:"test"`
}

func defaultFileConfig() fileConfig {
	var fc fileConfig
	fc.App.Env = "dev"
	fc.App.LogLevel = "info"
	fc.App.LogFormat = "text"

	fc.HTTP.Addr = ":8080"
	fc.HTTP.CORSAllowedOrigins = []string{"*"}
	fc.HTTP.RequestTimeout = 15 * time.Second
	fc.HTTP.ReadTimeout = 15 * time.Second
	fc.HTTP.WriteTimeout = 15 * time.Second
	fc.HTTP.ShutdownTimeout = 10 * time.Second
	fc.HTTP.RateLimitPerMin = 120

	fc.Postgres.DSN = "postgres://postgres:postgres@localhost:5432/starter?sslmode=disable"
	fc.Postgres.MaxConns = 20
	fc.Postgres.MinConns = 2
	fc.Postgres.AutoMigrate = false

	fc.Redis.Addr = "localhost:6379"
	fc.Redis.DB = 0

	fc.JWT.Secret = "dev-insecure-secret-change-in-prod"
	fc.JWT.AccessTTL = 2 * time.Hour
	fc.JWT.RefreshTTL = 7 * 24 * time.Hour

	fc.Storage.UploadDir = "./uploads"
	fc.Storage.MaxUploadMB = 10
	fc.Storage.MaxLargeUploadMB = 500
	fc.Storage.ImageMaxWidth = 1920
	fc.Storage.ImageJPEGQuality = 80

	fc.Worker.Concurrency = 5

	fc.Test.PostgresDSN = "postgres://postgres:postgres@localhost:5432/starter_test?sslmode=disable"
	fc.Test.RedisAddr = "localhost:6379"
	fc.Test.RedisDB = 15
	return fc
}

// Load 按「内置默认值 < configs/config.yaml < configs/config.local.yaml < 显式环境变量」顺序加载并严格校验配置。
func Load() (Config, error) {
	if explicit := strings.TrimSpace(os.Getenv("CONFIG_FILE")); explicit != "" {
		return LoadFromFiles(explicit)
	}
	basePath := locateConfigFile()
	if basePath == "" {
		return LoadFromFiles()
	}
	localPath := filepath.Join(filepath.Dir(basePath), "config.local.yaml")
	if _, err := os.Stat(localPath); err == nil {
		return LoadFromFiles(basePath, localPath)
	}
	return LoadFromFiles(basePath)
}

// LoadFromFiles 从给定 YAML 文件列表依次覆盖解码（启用 KnownFields 拦截拼写错误），再叠加容器/CI 环境变量。
func LoadFromFiles(paths ...string) (Config, error) {
	fc := defaultFileConfig()
	for _, p := range paths {
		if err := decodeStrictYAMLFile(p, &fc); err != nil {
			return Config{}, err
		}
	}

	cfg := Config{
		AppEnv:             fc.App.Env,
		HTTPAddr:           fc.HTTP.Addr,
		LogLevel:           fc.App.LogLevel,
		LogFormat:          fc.App.LogFormat,
		DatabaseURL:        fc.Postgres.DSN,
		DBMaxConns:         fc.Postgres.MaxConns,
		DBMinConns:         fc.Postgres.MinConns,
		AutoMigrate:        fc.Postgres.AutoMigrate,
		RedisAddr:          fc.Redis.Addr,
		RedisPassword:      fc.Redis.Password,
		RedisDB:            fc.Redis.DB,
		JWTSecret:          fc.JWT.Secret,
		JWTAccessTTL:       fc.JWT.AccessTTL,
		JWTRefreshTTL:      fc.JWT.RefreshTTL,
		CORSAllowedOrigins: fc.HTTP.CORSAllowedOrigins,
		RequestTimeout:     fc.HTTP.RequestTimeout,
		RateLimitPerMin:    fc.HTTP.RateLimitPerMin,
		ShutdownTimeout:    fc.HTTP.ShutdownTimeout,
		ReadTimeout:        fc.HTTP.ReadTimeout,
		WriteTimeout:       fc.HTTP.WriteTimeout,
		UploadDir:          fc.Storage.UploadDir,
		MaxUploadMB:        fc.Storage.MaxUploadMB,
		MaxLargeUploadMB:   fc.Storage.MaxLargeUploadMB,
		ImageMaxWidth:      fc.Storage.ImageMaxWidth,
		ImageJPEGQuality:   fc.Storage.ImageJPEGQuality,
		WorkerConcurrency:  fc.Worker.Concurrency,
		TestDatabaseURL:    fc.Test.PostgresDSN,
		TestRedisAddr:      fc.Test.RedisAddr,
		TestRedisPassword:  fc.Test.RedisPassword,
		TestRedisDB:        fc.Test.RedisDB,
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func decodeStrictYAMLFile(path string, dst *fileConfig) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid config %s: %w", path, err)
	}
	return nil
}

func locateConfigFile() string {
	candidates := []string{
		filepath.Join("configs", "config.yaml"),
		filepath.Join("..", "configs", "config.yaml"),
		filepath.Join("..", "..", "configs", "config.yaml"),
		filepath.Join("..", "..", "..", "configs", "config.yaml"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv("APP_ENV"); v != "" {
		c.AppEnv = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		c.HTTPAddr = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		c.LogFormat = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		c.DatabaseURL = v
	}
	if v := os.Getenv("AUTO_MIGRATE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.AutoMigrate = b
		}
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		c.RedisAddr = v
	}
	if v, ok := os.LookupEnv("REDIS_PASSWORD"); ok {
		c.RedisPassword = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RedisDB = n
		}
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWTSecret = v
	}
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		c.TestDatabaseURL = v
	}
	if v := os.Getenv("TEST_REDIS_ADDR"); v != "" {
		c.TestRedisAddr = v
	}
	if v, ok := os.LookupEnv("TEST_REDIS_PASSWORD"); ok {
		c.TestRedisPassword = v
	}
	if v := os.Getenv("TEST_REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.TestRedisDB = n
		}
	}
}

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return errors.New("http.addr cannot be empty")
	}
	if c.DatabaseURL == "" {
		return errors.New("postgres.dsn cannot be empty")
	}
	if c.AppEnv == "prod" && (c.JWTSecret == "" || c.JWTSecret == "dev-insecure-secret-change-in-prod") {
		return errors.New("jwt.secret (or JWT_SECRET) must be set to a strong key in prod")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("http.shutdown_timeout must be positive")
	}
	return nil
}
