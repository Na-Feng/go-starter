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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"go-starter/internal/kernel/audit"
	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/cache"
	"go-starter/internal/kernel/config"
	"go-starter/internal/kernel/db"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/kernel/logger"
	"go-starter/internal/kernel/middleware"
	"go-starter/internal/kernel/storage"
	"go-starter/internal/kernel/worker"
	"go-starter/internal/modules/files"
	"go-starter/internal/modules/iam"
	"go-starter/internal/modules/sample"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, cfg.LogFormat)

	// 1. 初始化 PostgreSQL、Redis 与本地存储
	if cfg.AutoMigrate && cfg.AppEnv != "prod" {
		if err := db.EnsureDatabase(context.Background(), cfg.DatabaseURL); err != nil {
			log.Error("failed to ensure database exists", "error", err)
			os.Exit(1)
		}
	}
	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := cache.NewRedis(context.Background(), cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Error("failed to connect redis", "error", err)
		os.Exit(1)
	}
	defer func() { _ = rdb.Close() }()

	if cfg.AutoMigrate {
		if err := db.Migrate(context.Background(), pool, "up"); err != nil {
			log.Error("auto migrate failed", "error", err)
			os.Exit(1)
		}
	}

	store, err := storage.NewLocalStore(cfg.UploadDir, storage.ImageOptions{MaxWidth: cfg.ImageMaxWidth, JPEGQuality: cfg.ImageJPEGQuality})
	if err != nil {
		log.Error("failed to init storage", "error", err)
		os.Exit(1)
	}

	// 2. 初始化异步任务队列（Asynq Client & Server）
	workerClient := worker.NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer func() { _ = workerClient.Close() }()

	workerSrv := worker.NewServer(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.WorkerConcurrency, log)
	if err := workerSrv.Start(); err != nil {
		log.Error("failed to start async worker", "error", err)
		os.Exit(1)
	}

	// 3. 显式组装公共内核与业务模块
	mux := http.NewServeMux()
	registerHealthRoutes(mux, pool, rdb, log)
	httpx.RegisterDocsRoutes(mux)

	txMgr := db.NewTxManager(pool)
	jwtMgr := auth.NewManager(cfg.JWTSecret)
	blacklist := auth.NewRedisBlacklist(rdb)
	permCache := iam.NewRedisPermCache(rdb)
	auditRec := audit.NewPostgresRecorder(pool)

	iamRepo := iam.NewPostgresRepo(pool)
	iamSvc := iam.NewService(iamRepo, txMgr, jwtMgr, blacklist, permCache, auditRec, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	if err := iamSvc.EnsureSeedAdmin(context.Background()); err != nil {
		log.Error("seed admin failed", "error", err)
		os.Exit(1)
	}

	apiRouter := auth.NewRouter(mux, jwtMgr, blacklist, iamSvc, log)
	iam.NewHandler(iamSvc, log).Register(apiRouter)
	files.NewHandler(store, cfg.MaxUploadMB, cfg.MaxLargeUploadMB, log).Register(apiRouter)

	sampleRepo := sample.NewPostgresRepo(pool)
	sampleSvc := sample.NewService(sampleRepo, txMgr)
	sample.NewHandler(sampleSvc, workerClient, log).Register(apiRouter)

	// 4. 挂载公共中间件链
	handler := middleware.Chain(
		mux,
		middleware.RequestID,
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.Timeout(cfg.RequestTimeout),
		middleware.RateLimit(rdb, cfg.RateLimitPerMin, time.Minute, log),
		middleware.Recoverer(log),
		middleware.AccessLog(log),
	)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 5. 监听系统退出信号，启动优雅停机
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Error("server crashed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		log.Info("shutdown signal received, draining connections and worker...")
	}

	// 6. 依次排空 HTTP 请求、停止异步 Worker，再由 defer 关闭 Redis 与 DB 连接池
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	workerSrv.Shutdown()
	log.Info("server exited cleanly")
}

func registerHealthRoutes(mux *http.ServeMux, pool *pgxpool.Pool, rdb *redis.Client, log *slog.Logger) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.OK(w, r, map[string]string{"status": "alive"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			httpx.Fail(w, r, log, errs.Unavailable("database not ready", err))
			return
		}
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			httpx.Fail(w, r, log, errs.Unavailable("redis not ready", err))
			return
		}
		httpx.OK(w, r, map[string]string{"status": "ready"})
	})
}
