---
name: verify-all
description: 完成功能开发或发版前，按改动范围执行静态检查、契约校验、集成测试、diff 净度清理与运行时实测验收时使用。
---

# 按范围实测验收剧本（Universal Agent Skill）

禁止仅凭“代码已写完”汇报完成，必须按本次实际改动范围收集确定性验证证据：

1. **静态、契约与单测门禁（必跑受影响端）**
   - 单端改动执行对应 `make check-server` / `make check-web` / `make check-taro`；跨端改动执行 `make check-all`（含 OpenAPI 契约零漂移 `check-contract`、AST/ESLint 防凑合检查、TypeScript 编译与单测）。

2. **真实数据库与 Redis 集成测试（改动 `db/migrations/`、`repo_pg.go` 或鉴权/缓存时必跑）**
   - 本地具备容器环境时执行 `make up && make test-int`（确认连接独立 `*_test` 测试库）。

3. **Web 端运行时交互验收（改动 `apps/web/` 页面逻辑时）**
   - 对照 `apps/web/src/feature-map.ts` 的路由与交互清单，启动服务并通过当前环境可用的浏览器验证工具（如 `ego-browser`、Browser Preview / DevTools MCP 或端到端脚本）实测登录、提交、分页渲染与控制台 0 报错。

4. **Taro 小程序运行时验收（改动 `apps/taro/` 页面逻辑时）**
   - 对照 `apps/taro/src/core/feature-map.ts`，具备小程序构建与模拟器环境时编译 `dist/` 并在微信开发者工具模拟器（如通过官方自动化工具 / CLI / MCP / Skill）验证页面渲染与交互无报错。

5. **交付前 `git diff` 净度自查（Unslop）**
   - 执行 `git diff` 逐行审视本次变更：清理显而易见的废话注释（如“定义变量”、“调用接口”）、未使用的辅助代码、调试输出及遗留临时文件（临时产物仅允许留在已忽略的 `.tmp/`）。

6. **证据化汇报**
   - 简明列出实际执行通过的命令、运行时实测结果与未覆盖项说明（如本地缺少 Docker 时明确标注交由 CI `integration-test` 承担），严禁掩盖任何失败。
