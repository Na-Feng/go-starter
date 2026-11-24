# Fullstack Starter (Go + React Web + Taro)

AI 时代高确定性全栈开发基座 Monorepo。

## 初始化（首次拉取到本地后执行）

```bash
# 1. 拉取 Go 依赖并生成 go.sum
cd apps/server && go mod tidy && cd ../..

# 2. 安装前端与小程序依赖（已通过 .npmrc 隔离 workspace 依赖）
pnpm install

# 3. 准备 PostgreSQL 与 Redis（本机已安装服务可直接跳过；若用 Docker 则执行 make up）
#    后端配置位于 apps/server/configs/config.yaml（本地私有覆盖可新建 config.local.yaml），首次运行会自动建库

# 4. 同步 OpenAPI 契约并跑通三端全量检查（含真实库集成测试）
make gen-api && make check-all && make test-int
```

## 日常启动
- 后端：`make dev-server`（默认管理员 `admin` / `admin123`，接口文档 `http://localhost:8080/docs`）
- Web 后台：`make dev-web`（`http://localhost:5173`）
- Taro 小程序：`make dev-taro`（用微信开发者工具打开 `apps/taro`）

## 跨工具通用 AI 工作流剧本（`.agents/skills/`）
采用开放 Agent Skills 标准路径 `.agents/skills/<name>/SKILL.md`（兼容 Devin、Codex、Cursor、Windsurf 等主流 Agent 自动发现，其他工具亦可通过根 `AGENTS.md` 任务路由表按路径直接加载）：
- `add-feature`：按范围裁剪的新增/改造功能剧本（Go + OpenAPI 契约 + Arco Web + Taro）
- `verify-all`：按改动端执行静态检查、`test-int`、运行时实测与 `git diff` 净度清理剧本
- `reflect`：故障复盘与“信任倒金字塔”底层约束下沉剧本
- `autonomous`：无人值守开发剧本（`auto/*` 分支隔离、3 次失败硬熔断、独立只读审查与 CI 自动合入）

## 开启无人值守 Auto-merge（GitHub 仓库设置）
在 GitHub 仓库的 `Settings -> General -> Pull Requests` 中勾选 **Allow auto-merge** 与 **Allow squash merging**，并在 `Settings -> Actions -> General -> Workflow permissions` 允许 GitHub Actions 创建与批准操作后：
- 普通业务 PR 在 `.github/workflows/ci.yml`（`check-all` + `integration-test` + `risk-gate`）全绿后将自动 Squash 合入 `main`；
- 触碰高危路径（`internal/kernel/`、AST/Lint 守卫配置、`.github/`、`.agents/`、`AGENTS.md` 或迁移中含 `DROP`/`TRUNCATE`）的 PR 会自动挂起等待人工审批。
