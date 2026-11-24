# AGENTS.md — Taro 小程序 AI 协作规范

本应用与 Go 后端配套，采用 **Taro 4 (React + TS 无 DOM) + OpenAPI 派生契约 + `src/core` 纯逻辑隔离 + ESLint `anti-slop` 硬约束**。

## 1. 产物与依赖约束
- 临时草稿放根 `.tmp/`，正式方案/决策统一按根 `AGENTS.md` 归入根 `docs/{specs,adr,postmortems,runbooks}/`；禁止随意引入新 npm 依赖。
- 严禁使用 `any`、非空断言 `!` 以及任何 `eslint-disable` / `biome-ignore` / `@ts-ignore` 规则豁免注释。

## 2. 小程序架构与编码红线
- **无 DOM 与标签约束**：`tsconfig.json` 已剔除 `DOM` lib，禁止 `window`/`document` 及原生 HTML 标签（统一用 `@tarojs/components`）。
- **核心逻辑与物化记忆**：纯业务校验、状态转换与页面功能地图统一放在 `src/core/`（严禁导入 `@tarojs/*`、`wx` 或定时器）；新增页面或分包路由时同步更新 `src/app.config.ts` 与 `src/core/feature-map.ts`（由单测强制对齐）。
- **主包/分包与防凑合隔离**：主包放 `src/pages/`，业务放 `src/subpackages/<domain>/`，严禁主包/`lib`/`core` 反向引用分包；页面层严禁 `useEffect`、`setTimeout`/`setInterval`、裸 `Taro.request`、直接读写 `Taro.*Storage*` 及超过 2 行的解释性注释。
- **契约与请求**：运行 `pnpm gen:api` 刷新 `src/api/openapi.gen.d.ts`（禁止手改），业务统一调用 `src/lib/request.ts` 的 `api.get/post`（自动从 OpenAPI 推导请求体与响应类型）。

## 3. 验证与交付标准（Inner Loop）
1. **静态与纯逻辑单测**：每次修改后必须执行 `pnpm check`（`check:api` + Biome + ESLint + `tsc` + Vitest 属性与对齐单测）全绿。
2. **微信开发者工具验收**：具备小程序平台构建环境时编译至 `dist/`，对照 `src/core/feature-map.ts` 通过 `wechatide-skill` 在模拟器中实测。
3. **故障沉淀**：遇到问题优先下沉至 TypeScript 类型、ESLint/Biome 规则或 `src/core/` 属性测试，经人工确认后落地。
