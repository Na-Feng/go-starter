# AGENTS.md — Taro 小程序 AI 协作规范

本仓库与 `go-starter` 后端配套，采用 **Taro (React + TypeScript 无 DOM) + TanStack Query + `src/core` 纯逻辑隔离 + Biome/ESLint 硬约束**。

## 1. AI 产物与依赖约束
- 方案文档放 `docs/ai/YYYY-MM-DD-主题.md`，临时文件放 `.tmp/`。禁止在根目录新建零散 `.md`。
- 禁止随意引入新依赖；严禁使用 `any` 和非空断言 `!`。

## 2. 小程序架构与编码红线
- **无 DOM 约束**：`tsconfig.json` 已剔除 `DOM` lib，禁止任何 `window`/`document`/`localStorage` 以及 HTML 标签（统一用 `@tarojs/components`）。
- **核心逻辑隔离**：纯业务计算、状态转换与校验规则放 `src/core/`，严禁导入 `@tarojs/*` 或 `wx`（由 ESLint 锁死），确保能用 Vitest 脱离模拟器直接测试。
- **主包/分包隔离**：TabBar 与登录页放 `src/pages/`，其余业务全部放 `src/subpackages/<domain>/`，严禁主包反向引用分包。
- **契约与请求**：接口类型由 `pnpm gen:api` 从 Go `/openapi.yaml` 生成；请求统一走 `src/lib/request.ts`，禁止裸调 `Taro.request` 或 `useEffect`。

## 3. 验证与交付标准（Inner Loop）
1. **静态与纯逻辑单测**：每次改完代码必须执行 `pnpm check`（Biome + ESLint + TypeScript + Vitest）全绿。
2. **微信开发者工具直连验收**：执行 `pnpm build:weapp` 编译到 `dist/` 后，通过微信开发者工具官方「工具 Skills」直接在模拟器中打开、操作并验证页面渲染与无报错（不需要额外编写 `miniprogram-automator` 脚本）。
3. **故障复盘与沉淀**：遇到 Bug 或被人工打回时，先复盘为何 `pnpm check` 未提前拦截，经人工确认后再将规则下沉到 TypeScript、ESLint/Biome 或 `src/core/` 回归测试。
