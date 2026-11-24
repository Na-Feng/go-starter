---
name: reflect
description: 修复 Bug、测试回归失败或被人工打回重写后，进行根因复盘并将经验下沉为仓库确定性硬约束时使用。
---

# 故障复盘与工程沉淀剧本（Universal Agent Skill）

完成问题修复后不要直接结束，按以下步骤输出简洁复盘提案：

1. **根因与漏检追问**
   - 明确指出本次缺陷的直接根因，并回答：**为什么现有的 `make check-all`、`make test-int`、OpenAPI 契约校验或类型系统未能提前拦截它？**

2. **按“信任倒金字塔”顺位选择 1 条最优加固落点**
   - **第一顺位（API / 类型物理收窄）**：能否调整 `internal/kernel/`、`src/lib/` 函数签名或 `openapi.yaml` / TypeScript 类型，让错误写法在编译期根本无法通过？
   - **第二顺位（AST / Linter 确定性规则）**：能否在 `apps/server/internal/arch_test.go`、`.golangci.yml` 或双端 `eslint.config.js`（`anti-slop`）/ `biome.json` 中增加 1 条带清晰修复指引的静态检查？
   - **第三顺位（属性 / 状态机 / 回归测试）**：是否已在 `invariant_test.go`、`module_test.go`、`integration_test.go` 或前端 `.test.ts(x)` 中补齐能稳定复现该类缺陷的测试？
   - **第四顺位（自然语言规范）**：仅当前三层均无法物理表达时，才精简 1 句话补充到对应 `AGENTS.md`。

3. **人工确认与归档**
   - 将具体加固改动呈报给开发者，**获得明确确认后**再修改内核、Linter 或守卫测试，并重跑 `make check-all` 验证。
   - 若属于典型架构级/跨端事故，同步在 `docs/postmortems/YYYY-MM-DD-<slug>.md` 留存简短复盘记录。
