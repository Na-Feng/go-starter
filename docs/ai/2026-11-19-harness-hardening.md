# 全栈基座信任倒金字塔加固方案（Harness Hardening）

- **日期**：2026-11-19
- **目标**：对齐《All Code Is Slop (Except Mine)》工程理念，修复初始仓库隐性断点，在不引入任何新增第三方测试/Lint 库的前提下，把“纸面约定”下沉为**代码库物理边界、编译器/AST 确定性检查、契约防漂移守卫、物化 Feature Map 与不变量属性测试**。

---

## 一、基线巡检暴露的 6 处初始断点

| 端 | 现象 | 根因 |
|---|---|---|
| Server | `gofmt` 直接报错退出 | `internal/kernel/storage/fuzz_test.go` 双引号内写了未转义反斜杠 `\w` |
| Server | `.golangci.yml` 规则完全未生效但 `make check` 全绿 | `Makefile` 在未检测到 `golangci-lint` 时仅打印 `[hint]` 并返回 `0` |
| 全栈契约 | 运行 `make gen-api` 会破坏双端编译 | `openapi.yaml` 缺失 `iam`/`samples` 路由与 Schema，且双端 `schema.d.ts` 实为手写文件 |
| Web | `pnpm install`、`tsc`、`vitest` 均失败 | `@types/react-dom@^18.3.12` 版本不存在；`hoist=false` 下漏声明自身依赖 `vite` 及 `vite/client` 类型；8 个初始文件未过 Biome 格式校验 |
| Web / Taro | `pnpm lint:rules` 直接崩溃 | Web 未挂载 `@typescript-eslint` plugin；Taro 未声明 `"type": "module"` 且 `src/lib/` 未配置 TS Parser |
| Workspace | pnpm 11 安装返回退出码 1 | `pnpm-workspace.yaml` 缺少原生二进制构建脚本 `allowBuilds` 白名单 |

---

## 二、分层加固设计（信任倒金字塔）

### 1. 底层：代码库边界与确定性静态检查（重投入，零静默跳过）

1. **OpenAPI 唯一真源与双向防漂移闭环**
   - 补全 `apps/server/docs/openapi.yaml` 全部路由（`health`、`auth`、`iam`、`samples`、`files`）与 `components.schemas`。
   - 将 `openapi-typescript` 产物拆分为纯生成文件 `src/api/openapi.gen.d.ts`（幂等覆盖、禁止手改），由 `src/api/schema.ts` 基于 `components['schemas']` 与 `paths` 派生业务类型与合法 API 路径类型 `ApiPath`。
   - **后端防漏写守卫**：Go 测试自动解析 `cmd/server` 与 `internal/modules/**` 中所有 `mux.Handle` / `mux.HandleFunc` 路由签名，强制断言其 `METHOD /path` 在 `docs/openapi.yaml` 中存在。
   - **前端防失步守卫**：根 `Makefile` 新增 `check-contract` 并并入 `make check-all`，通过临时生成与 `diff -u` 校验双端 `openapi.gen.d.ts` 与 `openapi.yaml` 零漂移。

2. **Go 标准库 `go/ast` 架构与防凑合守卫（`internal/arch_test.go`）**
   - 不依赖外部 `golangci-lint` 二进制，随 `go test ./...` 100% 强制执行：
     - **零配置垂直切片隔离**：自动扫描 `internal/modules/*` 目录，禁止任意两业务模块直接互相 import；禁止 `internal/kernel/**` 反向 import `internal/modules/**`。
     - **高危/易误用 API 禁用**：业务层禁用 `fmt.Print*`、标准库 `log`、`io/ioutil`、`time.Sleep`、`panic`、`os.Exit`、无超时 `http.Get/Post/Head/PostForm`、`context.Background/TODO`，报错直接给出正确替代 API。
     - **务实模式防借口注释**：在 `internal/modules/**` 中禁止 `TODO/FIXME/HACK/WORKAROUND/XXX`、禁止 `//nolint` 豁免、禁止函数体内超过 2 行的连续长篇解释注释（允许导出符号文档注释与 1~2 行简短意图注释）。
   - 同时保留并同步 `.golangci.yml`，支持严格模式环境叠加检查。

3. **Web / Taro 内联 ESLint `anti-slop` 插件与构建收口**
   - 修复双端 `eslint.config.js`、`tsconfig.json`、`package.json` 并将编译验证纳入检查链路。
   - 通过 ESLint v9 Flat Config 内联自定义插件（零外部 npm 依赖）扫描源码与注释 AST：
     - 拦截一切规则豁免注释（`eslint-disable`、`biome-ignore`、`@ts-ignore`、`@ts-nocheck`）。
     - 拦截补丁标记注释（`TODO`、`FIXME`、`HACK`、`WORKAROUND`、`XXX`）及业务目录内超过 2 行的连续解释性注释块。
     - 业务目录物理禁用 `setTimeout` / `setInterval` 定时器补丁，禁用业务页面绕过封装直接读写 `localStorage` / `sessionStorage` / `Taro.getStorageSync` / `Taro.setStorageSync`。
     - `apiRequest` 的请求路径收窄至基于 OpenAPI `paths` 派生的模板类型，杜绝手敲错误接口路径。

### 2. 中层：物化记忆（Feature Map）与系统不变量验证

1. **强类型物化 Feature Map + 对齐单测**
   - `apps/web/src/feature-map.ts` 与 `apps/taro/src/core/feature-map.ts`：声明每个页面的路由、模块名、权限码（与后端 RBAC 对齐）、依赖的 OpenAPI 路径（受 `keyof paths` 编译期约束）及关键交互入口。
   - 配套单测自动校验 Feature Map 与实际路由配置（`App.tsx` / `app.config.ts`）一致，杜绝文档腐烂。
2. **基于属性的不变量测试（Property-based & Invariant Tests，零新增第三方库）**
   - **Go 后端（`testing/quick` + `testing.F`）**：
     - `httpx.ParsePageQuery` 分页算术不变量（修复超大页码潜在整数溢出，证明任意整数输入下 `Page >= 1`、`1 <= PageSize <= MaxPageSize`、`Offset >= 0`）。
     - `iam.Service` 认证与 RBAC 不变量（吊销即失效、禁用账号全链路拒绝、角色变更必清权限缓存、`super_admin` 与细粒度权限集合不变量）。
     - `storage.LocalStore` 分片上传状态机不变量（乱序上传、重复覆盖幂等、缺片/篡改必拒且零残留脏文件）。
   - **Web / Taro（确定性种子 PRNG 属性测试）**：
     - 抽取纯权限判定逻辑与输入校验纯函数，执行百组以上随机属性测试验证空白裁剪幂等性、长度边界不变量与权限布尔代数性质。

### 3. 顶层：极薄 `AGENTS.md` 对齐

- 修正根与子应用 `AGENTS.md` 中的失真描述（如 React 版本、契约生成文件名、Feature Map 入口与内循环命令），保持每份规范精简聚焦于人工判断边界。
