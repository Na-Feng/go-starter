# Fullstack Starter (Go + React Web + Taro)

AI 时代高确定性全栈开发基座 Monorepo。

## 初始化（首次拉取到本地后执行）

```bash
# 1. 拉取 Go 依赖并生成 go.sum
cd apps/server && go mod tidy && cd ../..

# 2. 安装前端与小程序依赖（已通过 .npmrc 隔离 React 18/19）
pnpm install

# 3. 启动本地 Postgres 与 Redis
make up

# 4. 同步 OpenAPI 契约并跑通三端全量检查
make gen-api && make check-all
```

## 日常启动
- 后端：`make dev-server`（默认管理员 `admin` / `admin123`，接口文档 `http://localhost:8080/docs`）
- Web 后台：`make dev-web`（`http://localhost:5173`）
- Taro 小程序：`make dev-taro`（用微信开发者工具打开 `apps/taro`）
