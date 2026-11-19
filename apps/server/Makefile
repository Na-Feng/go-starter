APP_NAME := server
CMD_PATH := ./cmd/server
BIN_PATH := ./bin/$(APP_NAME)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/starter?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/starter_test?sslmode=disable
TEST_REDIS_ADDR ?= localhost:6379
TEST_REDIS_DB ?= 15

.PHONY: help up down dev build run test test-int fuzz bench fmt vet lint check clean sqlc migrate-up migrate-down migrate-status migrate-create docker-build

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## 启动本地 Postgres 与 Redis
	docker compose up -d --wait

down: ## 停止本地依赖容器
	docker compose down

dev: ## 本地启动服务（Text 日志，默认开启 AUTO_MIGRATE）
	APP_ENV=dev LOG_FORMAT=text AUTO_MIGRATE=true DATABASE_URL="$(DATABASE_URL)" go run $(CMD_PATH)

build: ## 编译生产二进制文件
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_PATH) $(CMD_PATH)

run: build ## 编译并运行
	$(BIN_PATH)

test: ## 运行全部单元测试（含竞态检测）
	go test -race -v ./...

test-int: ## 运行真实 Postgres/Redis 集成测试（需提供包含 test 的独立库）
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" TEST_REDIS_ADDR="$(TEST_REDIS_ADDR)" TEST_REDIS_DB="$(TEST_REDIS_DB)" go test -race -tags=integration -v ./...

fmt: ## 格式化代码
	gofmt -s -w .

vet: ## 静态检查
	go vet ./...

lint: ## 运行 golangci-lint（需本地安装 golangci-lint）
	golangci-lint run ./...

check: fmt vet test ## 一键内循环验证（格式化 + 静态检查 + 竞态测试；若已装 golangci-lint 可接 lint）
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "[hint] 未检测到 golangci-lint，已跳过额外 Lint 检查"; fi

sqlc: ## 根据 db/queries 生成强类型 SQL 代码（需安装 sqlc）
	sqlc generate

migrate-up: ## 使用内置 goose 执行全部待迁移脚本
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down: ## 回滚最近一次数据库迁移
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down

migrate-status: ## 查看当前数据库迁移版本状态
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate status

migrate-create: ## 创建新的 SQL 迁移文件（用法：make migrate-create name=add_users）
	@test -n "$(name)" || (echo "请传入 name，例如: make migrate-create name=add_users" && exit 1)
	goose -dir db/migrations create $(name) sql

docker-build: ## 构建生产 Docker 镜像
	docker build -t $(APP_NAME):latest .

fuzz: ## 运行原生 Fuzz 模糊测试（各 10 秒）
	go test -fuzz=FuzzDecodeAndValidate -fuzztime=10s ./internal/kernel/httpx
	go test -fuzz=FuzzOpenSeekerPathTraversal -fuzztime=10s ./internal/kernel/storage

bench: ## 运行性能基准与内存分配测试
	go test -bench=. -benchmem ./...

clean: ## 清理构建产物
	rm -rf ./bin
