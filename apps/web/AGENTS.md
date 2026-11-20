# AGENTS.md — Web 管理后台 AI 协作规范

本应用与 Go 后端配套，采用 **TypeScript 严格模式 + OpenAPI 派生契约 + TanStack Query + 内联 ESLint `anti-slop` 硬约束 + 物化 Feature Map**。

## 1. 产物与依赖约束
- 临时草稿放根 `.tmp/`，正式方案/决策统一按根 `AGENTS.md` 归入根 `docs/{specs,adr,postmortems,runbooks}/`；禁止随意引入新 npm 依赖。
- 严禁使用 `any`、非空断言 `!` 以及 `eslint-disable` / `biome-ignore` / `@ts-ignore` 豁免注释（已由 Biome 与 ESLint 锁死）。

## 2. 架构与编码红线
- **契约与路径锁死**：`src/api/openapi.gen.d.ts` 由 `pnpm gen:api` 自动生成（禁止手改），业务引用 `src/api/schema.d.ts`；请求统一走 `apiRequest`（路径受 `ApiEndpoint` 编译期约束），按钮权限用 `<Can perm="...">`（受 `PermissionCode` 枚举约束）。
- **物化记忆对齐**：新增或修改页面路由、标题、权限或交互入口时，必须同步更新 `src/feature-map.ts`（由 `feature-map.test.tsx` 挂载 `<App />` 强制校验对齐）。
- **防凑合限制**：`src/pages/` 与 `src/features/` 严禁调用 `useEffect`、`setTimeout`/`setInterval`、裸 `fetch`、直接读写 `localStorage`/`sessionStorage`，并禁止超过 2 行的连续解释注释或 `TODO/HACK` 标记。
- **UI 组件规范**：统一使用 `@arco-design/web-react`，查阅类型定义对齐 Props。

## 3. 验证与交付标准（Inner Loop）
1. **确定性静态与单测**：每次修改后必须执行 `pnpm check`（契约零漂移 `check:api` + Biome + ESLint `anti-slop` + `tsc` + Vitest）全绿。
2. **运行时浏览器验收**：涉及页面交互改动时，优先在本地对照 `src/feature-map.ts` 使用 `ego-browser` 在独立 Space 实测关键流并确认控制台 0 报错。
3. **故障沉淀**：遇到回归问题时，按「TypeScript 类型 > ESLint/Biome 规则 > Vitest 属性/DOM 测试 > `AGENTS.md`」优先级经人工确认后下沉。
