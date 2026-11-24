# AGENTS.md — 全栈 Monorepo AI 协作总纲

本仓库为 **Go 后端 + React Web 后台（Arco Design）+ Taro 微信小程序** 一体化 Monorepo。确定性约束已下沉至编译器、Go AST 守卫、ESLint `anti-slop` 插件与契约零漂移检查，任何 Agent 工具均遵循以下入口：

## 1. 任务路由表（按复杂度分流，通用剧本位于 `.agents/skills/`）
| 任务类型 | 执行路由 | 检查与验收范围 |
| :--- | :--- | :--- |
| **简单小改**（改文案、调样式、单文件小修） | **跳过工作流剧本**，直接修改 | 仅跑受影响端的 `make check-server` / `check-web` / `check-taro` |
| **新增/改造功能**（单端或跨端） | 加载 `add-feature`（`.agents/skills/add-feature/SKILL.md`），先锁定范围 | 仅同步和检查受影响端（跨端跑 `make check-all`） |
| **功能验收 / 发版前自检** | 加载 `verify-all`（`.agents/skills/verify-all/SKILL.md`） | 按改动端串联静态检查、`test-int`、运行时实测与 `git diff` 净度清理 |
| **修完 Bug / 被打回重写** | 加载 `reflect`（`.agents/skills/reflect/SKILL.md`） | 追问漏检根因，按「类型 > AST/Lint > 测试 > `AGENTS.md`」确认后沉淀 |
| **无人值守 / 自动开发** | 加载 `autonomous`（`.agents/skills/autonomous/SKILL.md`） | `auto/*` 分支隔离 + 3次失败熔断 + 独立只读审查 + CI Squash Auto-merge |

## 2. 仓库边界与产物位置
- **文档生命周期归口**（由 `arch_test.go` 强制校验）：临时草稿与一次性调研放 `.tmp/`（不入库）；正式文档统一归口根 `docs/`：需求拆解 `docs/specs/YYYY-MM-DD-<slug>.md`、架构决策 `docs/adr/NNNN-<slug>.md`（两者须含 `status: draft|accepted|superseded` Frontmatter）、故障复盘 `docs/postmortems/YYYY-MM-DD-<slug>.md`、运维手册 `docs/runbooks/<slug>.md`。严禁在别处新建零散 `.md`。
- **子应用物理边界**：`apps/server/`（`internal/arch_test.go` 零配置模块隔离与注释/API 守卫）、`apps/web/`（页面记忆见 `src/feature-map.ts`）、`apps/taro/`（隔离 `src/core/` 与主分包，页面记忆见 `src/core/feature-map.ts`）。

## 3. 跨端契约与物化记忆同步
1. 修改或新增后端接口/权限码时，同步更新 `apps/server/docs/openapi.yaml`（权限码需与 `db/migrations/*.sql` 一致，由 `openapi_contract_test.go` 强制校验）。
2. 执行 `make gen-api` 幂等刷新双端 `src/api/openapi.gen.d.ts`（严禁手改生成文件，新增类型别名维护在 `src/api/schema.d.ts`）。
3. 新增或调整前端/小程序路由页面时，同步更新对应 `feature-map.ts`。
