.PHONY: help up down gen-api check-contract check-all check-server check-web check-taro test-int dev-server dev-web dev-taro

help: ## 显示全栈 Monorepo 可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## 启动统一本地 PostgreSQL 与 Redis
	docker compose up -d --wait

down: ## 停止本地依赖容器
	docker compose down

gen-api: ## 从 Go OpenAPI 契约一键同步 Web 与 Taro 的 TypeScript 类型
	pnpm --filter web-starter gen:api
	pnpm --filter taro-starter gen:api

check-contract: ## 校验双端 OpenAPI 生成类型与后端契约零漂移
	pnpm --filter web-starter check:api
	pnpm --filter taro-starter check:api

check-server: ## 运行 Go 后端内循环检查
	$(MAKE) -C apps/server check

check-web: ## 运行 Web 后台内循环检查
	pnpm --filter web-starter check

check-taro: ## 运行 Taro 小程序内循环检查
	pnpm --filter taro-starter check

check-all: check-server check-web check-taro ## 一键运行三端全量静态检查与单元测试

test-int: ## 运行 Go 后端真实 Postgres/Redis 集成测试
	$(MAKE) -C apps/server test-int

dev-server: ## 启动 Go 后端服务
	$(MAKE) -C apps/server dev

dev-web: ## 启动 Web 管理后台
	pnpm --filter web-starter dev

dev-taro: ## 编译并监听 Taro 微信小程序
	pnpm --filter taro-starter dev:weapp
