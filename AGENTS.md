# AGENTS.md — 全栈 Monorepo AI 协作总纲

本仓库为 **Go 后端 + React Web 后台 + Taro 微信小程序** 一体化 Monorepo。确定性检查已下沉至编译器、Go AST 守卫、ESLint `anti-slop` 插件与契约漂移校验，此处仅保留跨端协作流程：

## 1. 仓库边界与产物位置
- **全栈文档归口**：跨端需求拆解、架构方案统一放在根目录 `docs/ai/YYYY-MM-DD-主题.md`；临时脚本放 `.tmp/`。禁止在根目录或子应用内新建零散 `.md`。
- **子应用物理边界**：
  - `apps/server/`: Go 后端（`internal/arch_test.go` 自动隔离所有业务模块并拦截违规 API/补丁注释）。
  - `apps/web/`: React 管理后台（业务层禁 `useEffect`/`setTimeout`/裸 `fetch`/规则豁免，页面记忆见 `src/feature-map.ts`）。
  - `apps/taro/`: Taro 4 小程序（无 `DOM`、隔离 `src/core/` 与主分包，页面记忆见 `src/core/feature-map.ts`）。

## 2. 跨端契约与物化记忆同步
1. 修改或新增后端接口/权限码时，同步更新 `apps/server/docs/openapi.yaml`（权限码需与 `db/migrations/*.sql` 一致，由 `openapi_contract_test.go` 强制校验）。
2. 在根目录执行 `make gen-api`，幂等刷新双端 `src/api/openapi.gen.d.ts`（禁止手改生成文件，派生别名维护在 `src/api/schema.d.ts`）。
3. 新增或调整前端/小程序路由页面时，同步更新对应 `feature-map.ts`。

## 3. 全栈验收与故障沉淀（Inner Loop）
- **交付前必检**：执行根目录 `make check-all`（串联 Go 构建/AST 守卫/属性单测 + 双端契约零漂移/Biome/ESLint/TSC/Vitest）；涉及 SQL 或迁移改动时加跑 `make test-int`。
- **运行时验收**：Web 端优先使用 `ego-browser` 在独立 Space 结合 `src/feature-map.ts` 实测；小程序端通过 `wechatide-skill` 在模拟器实测。
- **故障沉淀闭环**：修复 Bug 或被打回后，先复盘为何 `make check-all` 未提前拦截，按「类型/API 封装 > AST/Lint 规则 > 属性/回归测试 > `AGENTS.md`」优先级提出加固方案，经人工确认后落地。
