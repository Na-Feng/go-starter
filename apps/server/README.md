# Go Starter

面向 AI 协作与单二进制部署的 Go Web 后台开发基座（极简内核 + 垂直业务切片）。

## 快速开始

```bash
# 1. 拉取依赖并生成 go.sum
go mod tidy

# 2. 启动本地 PostgreSQL 与 Redis
make up

# 3. 运行单元与接口测试
make test

# 4. 启动服务（自动执行内置 goose 迁移，默认初始化管理员 admin / admin123）
make dev
```

启动后可访问：
- 在线接口文档：`http://localhost:8080/docs`（OpenAPI 规范：`/openapi.yaml`）
- 存活 / 就绪探针：`GET /healthz`、`GET /readyz`

## 目录结构

```text
go-starter/
├── AGENTS.md                 # AI 编程规范与架构约束
├── Makefile                  # 常用开发、迁移、测试与构建命令
├── Dockerfile                # 多阶段无 CGO 静态构建镜像
├── docker-compose.yml        # 本地 Postgres 16 + Redis 7
├── .env.example              # 环境变量配置模板
├── cmd/
│   ├── server/main.go        # 服务入口（HTTP + Asynq 同进程优雅停机）
│   └── migrate/main.go       # 内置 goose 迁移命令入口
├── db/
│   ├── embed.go              # go:embed 打包 SQL 迁移文件
│   ├── migrations/           # goose 数据库迁移（含 sample / RBAC / audit_logs）
│   └── queries/              # sqlc 查询定义
├── docs/openapi.yaml         # OpenAPI 3.0.3 接口契约
└── internal/
    ├── kernel/               # 公共内核：config, logger, errs, httpx, db, cache, auth, audit, storage, worker
    └── modules/              # 垂直业务模块：iam, files, sample
```

## 常用命令

- `make build`：编译静态二进制文件到 `./bin/server`
- `make migrate-up` / `make migrate-down` / `make migrate-status`：管理数据库迁移
- `make migrate-create name=xxx`：创建新的 SQL 迁移文件
- `make sqlc`：根据 `db/queries/` 生成强类型数据库访问代码
- `make docker-build`：构建生产 Docker 镜像

## 生产部署注意事项

1. 在 `.env` 中将 `APP_ENV=prod` 并设置强随机 `JWT_SECRET`。
2. 默认管理员 `admin` / `admin123` 仅在空库首次启动时生成，上线后请立即调用 `/api/v1/auth/change-password` 修改密码。
3. 若前端通过 Nginx 代理分片上传接口，请将 `client_max_body_size` 调整为大于单片上限（默认 `10M`）。
