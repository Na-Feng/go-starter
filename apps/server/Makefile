APP_NAME := server
CMD_PATH := ./cmd/server
BIN_PATH := ./bin/$(APP_NAME)

.PHONY: help up down dev build run test test-int fuzz bench fmt vet lint check clean sqlc migrate-up migrate-down migrate-status migrate-create docker-build

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## 可选：通过 Docker 启动本地 Postgres 与 Redis（本机已安装服务时无需执行）
	docker compose -f ../../docker-compose.yml up -d --wait

down: ## 可选：停止本地 Docker 容器
	docker compose -f ../../docker-compose.yml down

dev: ## 本地启动服务（默认加载 configs/config.yaml 与可选 config.local.yaml）
	go run $(CMD_PATH)

build: ## 编译生产二进制文件
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_PATH) $(CMD_PATH)

run: build ## 编译并运行
	$(BIN_PATH)

test: ## 运行全部单元测试（含竞态检测）
	go test -race -v ./...

test-int: ## 运行真实 Postgres/Redis 集成测试（加载 configs/config.yaml 的 test 段）
	go test -race -tags=integration -v ./...

fmt: ## 格式化代码
	gofmt -s -w .

vet: ## 静态检查
	go vet ./...

lint: ## 运行 golangci-lint（需本地安装 golangci-lint）
	golangci-lint run ./...

check: fmt vet ## 一键内循环验证（格式化 + go vet + 全包编译 + AST 架构/契约/竞态单测 + 可选 golangci-lint）
	go build ./...
	$(MAKE) test
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; fi

sqlc: ## 根据 db/queries 生成强类型 SQL 代码（需安装 sqlc）
	sqlc generate

migrate-up: ## 使用内置 goose 执行全部待迁移脚本
	go run ./cmd/migrate up

migrate-down: ## 回滚最近一次数据库迁移
	go run ./cmd/migrate down

migrate-status: ## 查看当前数据库迁移版本状态
	go run ./cmd/migrate status

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
