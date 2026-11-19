# AGENTS.md — 全栈 Monorepo AI 协作总纲

本仓库为 **Go 后端 + React Web 后台 + Taro 微信小程序** 一体化 Monorepo。各端详细编码规则见对应子目录下的 `AGENTS.md`，全局约束如下：

## 1. 仓库边界与产物位置
- **全栈文档归口**：跨端需求拆解、架构方案统一放在根目录 `docs/ai/YYYY-MM-DD-主题.md`；临时脚本放 `.tmp/`。禁止在根目录或子应用内新建零散 `.md`。
- **子应用独立性**：
  - `apps/server/`: Go 后端（禁止跨业务模块直接引用，遵循 `.golangci.yml`）。
  - `apps/web/`: React 19 管理后台（禁用 `useEffect` 与裸 `fetch`，遵循 `eslint.config.js` 与 `biome.json`）。
  - `apps/taro/`: Taro 4 小程序（无 `DOM`、隔离 `src/core/` 与主分包）。

## 2. 跨端开发与契约同步流程
1. 修改或新增后端接口时，必须先更新 `apps/server/docs/openapi.yaml`。
2. 立即在根目录执行 `make gen-api`，同步刷新 `apps/web` 与 `apps/taro` 的 `src/api/schema.d.ts`。
3. 再基于最新强类型契约开发前端与小程序页面。

## 3. 全栈验收与故障沉淀（Inner Loop）
- **交付前必检**：涉及跨端修改时，在根目录执行 `make check-all`（涉及数据库迁移或 SQL 改动时加跑 `make test-int`）。
- **运行时验收**：Web 端优先使用 `ego(lite)` 在独立 Space 实测；小程序端通过微信开发者工具官方「工具 Skills」在模拟器实测。
- **故障沉淀闭环**：修复 Bug 或被打回后，先复盘为何 `make check-all` 未能提前拦截，按「类型/API 封装 > Lint/编译器规则 > 回归测试 > AGENTS.md」优先级提出加固方案，经人工确认后落地。
