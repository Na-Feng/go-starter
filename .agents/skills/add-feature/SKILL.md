---
name: add-feature
description: 在本全栈仓库新增或改造业务功能（支持仅后端、仅管理端、仅小程序或全栈联调）时使用。
---

# 按范围裁剪的新增功能剧本（Universal Agent Skill）

严格按以下步骤执行，**严禁改动本次范围之外的子应用**：

1. **锁定范围与复杂度门控**
   - 先明确勾选本次涉及的端：`[ ] server`、`[ ] web`、`[ ] taro`（未勾选的端禁止触碰）。
   - **复杂改动先对齐**：若涉及跨模块交互、复杂状态机或核心表结构调整，先只读相关代码并在对话中输出简明方案（或按需写入 `docs/specs/YYYY-MM-DD-<slug>.md`，含 `status: draft` Frontmatter），**经人工确认后再编码**；简单 CRUD 可直接实施。

2. **后端数据与业务层（仅当涉及 `server` 时执行）**
   - 在 `apps/server/db/migrations/` 新增递增 SQL 迁移文件；若新增权限码，必须同步插入 `permissions` 表种子数据。
   - 参考 `internal/modules/sample/` 或 `iam/`，在 `internal/modules/<domain>/` 实现 DTO、Repository（`db.Conn(ctx, r.pool)`）、Service（多步写操作用 `txMgr.WithinTx` + `auditRecorder.Record`）、Handler（通过 `*auth.Router` 的 `Public`/`Authenticated`/`Protected` 注册）与 `module_test.go`。
   - 严禁跨业务模块直接 `import`（`internal/arch_test.go` 已自动对所有模块执行物理隔离，跨模块调用请在调用方定义最小接口并由 `cmd/server/main.go` 注入，同时建议同步 `.golangci.yml`）。
   - 涉及新表、复杂查询或事务逻辑时，补充 `integration_test.go`（`//go:build integration`）或状态不变量测试。

3. **契约与类型同步（后端接口或权限码有变动时必做）**
   - 同步更新 `apps/server/docs/openapi.yaml` 的 `paths`（含响应体 `data` Schema 与 `security` 声明）、`components.schemas` 及 `PermissionCode` 枚举；若新增了 DTO 结构体，将其加入 `openapi_contract_test.go` 的 `bindings` 映射以启用反射字段互锁。
   - 执行 `make gen-api`（或仅运行受影响端的 `pnpm --filter <app> gen:api`）刷新 `src/api/openapi.gen.d.ts`，严禁手改生成文件；新增模型别名在 `src/api/schema.d.ts` 中派生。

4. **Web 管理后台（仅当涉及 `web` 时执行）**
   - 统一使用 `@arco-design/web-react` 组件、强类型 `api.get/post/put/patch` 客户端与受 `PermissionCode` 约束的 `<Can perm="...">`。
   - 业务页面禁止 `useEffect`、`setTimeout`/`setInterval`、裸 `fetch`、直接操作 `localStorage` 及超过 2 行的解释性注释。
   - 若新增或调整页面路由/菜单/页面级权限（`routePerm`），同步更新 `apps/web/src/feature-map.ts` 与 `App.tsx` 的 `FEATURE_COMPONENTS`。

5. **Taro 小程序（仅当涉及 `taro` 时执行）**
   - 纯业务计算与校验函数放入 `apps/taro/src/core/`（保持零 Taro/wx 依赖）并补齐 Vitest 单测或属性测试。
   - 新增业务页面放入 `src/subpackages/<domain>/`，同步更新 `src/app.config.ts` 与 `src/core/feature-map.ts`。

6. **按范围交付自检**
   - 运行受影响端的确定性检查：`make check-server`、`make check-web`、`make check-taro`（跨端修改直接运行根目录 `make check-all`）。
