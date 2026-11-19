# AGENTS.md — Web 管理后台 AI 协作规范

本仓库与 Go 后端基座配套，采用 **TypeScript 严格模式 + TanStack Query + 禁用业务层 useEffect + Biome/ESLint 硬约束**。

## 1. AI 产物与依赖约束
- 方案文档放 `docs/ai/YYYY-MM-DD-主题.md`，临时文件放 `.tmp/`。禁止在根目录新建零散 `.md`。
- 禁止随意引入新依赖；严禁在代码中使用 `any`（已由 Biome `noExplicitAny` 锁死）。

## 2. 架构与编码红线
- **契约先行**：所有后端接口类型与路径以 `src/api/schema.d.ts`（由 `pnpm gen:api` 从 Go `/openapi.yaml` 生成）为准，禁止手写硬编码接口返回类型。
- **禁用 `useEffect`**：`src/pages/` 与 `src/features/` 严禁调用 `useEffect`（由 `eslint.config.js` 物理拦截）。数据加载用 `useQuery`，提交用 `useMutation`，派生状态直接计算。
- **UI 组件规范**：统一使用 `@arco-design/web-react`（Table、Form、Upload、Message 等）；遇到组件 Prop 不确定时直接查阅 `node_modules/@arco-design/web-react` 类型定义，禁止用 `as any` 绕过。
- **权限控制**：受保护按钮与菜单使用 `<Can perm="iam:user:write">` 包裹，与 Go 后端 `RequirePermission` 保持一致。

## 3. 运行时验证与交付标准（Inner Loop）
1. **静态与单测**：每次改完代码必须执行 `pnpm check`（Biome + ESLint + TypeScript 严格校验 + Vitest 单测）全绿。
2. **本地浏览器交互验收（`ego(lite)`）**：新增或修改页面逻辑后，优先在本地通过 `ego-browser` 在独立 Space 中打开对应路由（默认 `http://localhost:5173`）完成登录、表单提交、列表渲染与控制台 0 报错验证（仅在 macOS 本地可用 `ego(lite)` 时执行；CI 确定性回归由 Playwright 承担）。
3. **故障复盘与沉淀**：遇到页面 Bug 或被人工打回时，先复盘为何 `pnpm check` 或浏览器验收未提前拦截，经人工确认后再将规则下沉到 TypeScript 类型、ESLint/Biome 规则或回归测试。
