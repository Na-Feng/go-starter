---
name: autonomous
description: 执行无人值守开发任务（独立分支隔离、3次失败熔断、独立只读审查与 PR 自动合入）时使用。
---

# 无人值守开发与自动合入剧本（Universal Agent Skill）

严格按以下 5 步闭环执行，**严禁跳过分支隔离、熔断红线与独立审查**：

1. **物理分支隔离（严禁直改 `main`）**
   - 基于最新 `origin/main` 创建独立分支 `auto/<topic>`（或使用 `git worktree` 隔离工作区），全流程严禁直接向 `main` 提交或推送。

2. **契约化实施与硬熔断红线**
   - 遵循 `AGENTS.md` 与 `add-feature` / `verify-all` 完成开发与内循环验证。
   - **触发以下任一条件立即停手熔断，输出阻塞报告等待人工介入**：
     - 同一编译、Lint 或测试报错连续尝试修复 **3 次** 仍未通过；
     - 任务必须修改受保护底座（`internal/kernel/`、`arch_test.go`、`openapi_contract_test.go`、`.golangci.yml`、`eslint.config.js`、`biome.json`、`.github/`、`.agents/`）；
     - 任务需要引入新的第三方依赖，或在 SQL 迁移中执行 `DROP` / `TRUNCATE` 等破坏性语句；
   - 严禁通过规则豁免注释、`as any` 或删减/弱化原有测试断言来伪造全绿。

3. **独立只读 Review 门禁（打破“自己写自己审”）**
   - 本地跑通受影响端的 `make check-all`（及涉及 DB 时的 `make test-int`）后，启动一次**独立只读代码审查**（优先调用只读 Subagent / 审查 Skill，或切换纯审查清单视角对照原始需求与 `AGENTS.md` 逐行核查 `git diff`）：
     - 重点核查：接口鉴权与权限码越权、`WithinTx` 事务与审计完整性、Feature Map 同步、是否存在假断言或废话代码。
   - 发现问题立即修复并重新跑通全套检查，直至审查无阻塞项。

4. **轻量记录与干净工作区**
   - 临时排查草稿仅存放在 `.tmp/`（不入 Git）；仅当产生正式架构决策或规格变更时，按规范写入 `docs/adr/NNNN-<slug>.md` 或 `docs/specs/YYYY-MM-DD-<slug>.md`（含 `status` Frontmatter），严禁在仓库中遗留流水账报告。

5. **创建 PR 与门禁自动合入**
   - 推送 `auto/<topic>` 分支并创建 Pull Request，在 PR 描述中附上改动范围、关键取舍与本地检查证据。
   - 由 `.github/workflows/ci.yml` 执行全量静态检查、真实 Postgres/Redis 集成测试与 `risk-gate` 高危路径扫描：低风险 PR 全绿后将自动 Squash Merge 合入 `main`；高危路径自动挂起等待人工审批。
