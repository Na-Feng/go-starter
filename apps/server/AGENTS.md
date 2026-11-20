# AGENTS.md — Go 基座 AI 协作精简规范

本仓库采用 **极简内核 + 垂直业务切片（Vertical Slice）** 架构。跨模块隔离、高危 API 禁用、防补丁注释及 OpenAPI/权限码对齐已由 `internal/arch_test.go` 与 `internal/openapi_contract_test.go` 在 `go test` 阶段强制执行：

## 1. 产物与变更边界
- **文档归口**：临时脚本与草稿放根 `.tmp/`；正式方案/决策/复盘按根 `AGENTS.md` 归入根 `docs/{specs,adr,postmortems,runbooks}/`（由 `arch_test.go` 物理拦截零散 `.md`）。
- **内核只读 & 依赖克制**：做业务需求时禁止修改 `internal/kernel/`；禁止擅自 `go get` 新第三方库（先说明理由并确认）。
- **防凑合红线**：业务模块禁止 `time.Sleep`、`//nolint`、`TODO/HACK` 标记及函数内超过 2 行的解释性注释块，遇到冲突直接修复根因。

## 2. 模块组织与契约对齐
- **垂直聚合**：新业务放 `internal/modules/<domain>/`（参考 `sample/` 或 `iam/`），`arch_test.go` 会自动对其启用跨模块 import 物理隔离，同时建议同步维护 `.golangci.yml`。
- **路由与权限真源**：新增 HTTP 路由必须同步写入 `docs/openapi.yaml`；新增 `RequirePermission` 权限码必须同步写入 `db/migrations/*.sql` 与 `docs/openapi.yaml` 的 `PermissionCode` 枚举。
- **数据库与接口约定**：多步写操作用 `txMgr.WithinTx` + `auditRecorder.Record`；入参统一 `httpx.DecodeAndValidate`；响应统一 `httpx.OK` / `Created` / `Fail`。

## 3. 交付与沉淀标准（Inner Loop）
- 每次改完代码必须执行 `make check`（含 `gofmt`、`go vet`、`go build ./...`、AST/契约/属性竞态测试）全绿；涉及 `db/migrations/` 或 `repo_pg.go` 时加跑 `make test-int`。严禁删减断言或绕过检查。
- 故障复盘按「**`kernel` 类型/API 封装 > `arch_test.go` / `.golangci.yml` > 属性/集成测试 > `AGENTS.md`**」优先级下沉，经人工确认后落地。
