# AGENTS.md — Go 基座 AI 协作精简规范

本仓库采用 **极简内核 + 垂直业务切片（Vertical Slice）** 架构。所有可静态校验的硬规则已下沉至 `.golangci.yml` 与 `Makefile`，此处仅保留需人工/AI 判断的工程约定：

## 1. 产物与变更边界
- **AI 文档位置**：方案、需求拆解等统一放 `docs/ai/YYYY-MM-DD-主题.md`；临时脚本放 `.tmp/`。禁止在根目录或 `internal/` 新建零散 `.md` 文档。
- **内核只读 & 依赖克制**：做业务需求时禁止修改 `internal/kernel/`；禁止擅自 `go get` 新第三方库（先说明理由并确认）。
- **拒绝凑合补丁**：禁止在业务代码里用 `time.Sleep` 或长篇注释解释“为什么这样绕过”，遇到架构冲突直接修复根因。

## 2. 模块组织与样板对齐
- **垂直聚合**：新业务放 `internal/modules/<domain>/`（简单模块参考 `sample/`，复杂模块参考 `iam/`）。
- **新增模块时**：同步在 `.golangci.yml` 的 `depguard` 中为新模块补充跨模块引用隔离，并更新 `docs/openapi.yaml`。
- **数据库与接口约定**：表变更先写 `db/migrations/`；多步写操作用 `txMgr.WithinTx` + `auditRecorder.Record`；入参统一 `httpx.DecodeAndValidate`；响应统一 `httpx.OK` / `Created` / `Fail`。

## 3. 交付前唯一标准（Inner Loop）
每次改完代码必须执行 `make check`（涉及 `db/migrations/` 或 `repo_pg.go` 改动时还需执行 `make test-int`）并确认全绿才能汇报完成；严禁通过删减或弱化原有测试断言来绕过检查。
## 4. 故障复盘与工程沉淀
修复 Bug 或被人工打回重写后，必须先做根因复盘（**为什么现有 `make check` / `make test-int` 未能提前拦截？**），再按「**API/类型约束（`kernel`）> `.golangci.yml` 规则 > 集成回归测试 > `AGENTS.md`**」的优先级提出 1 条杜绝同类问题的底层加固方案，**经人工确认后**再落地到对应底层文件。
