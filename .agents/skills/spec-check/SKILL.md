---
name: spec-check
description: 启动长任务或无人值守开发前，对需求规格文档做只读交叉挑刺（消解逻辑矛盾、权限遗漏与异常边界）并定稿时使用。
---

# 长任务开工前规格排雷剧本（Universal Agent Skill）

在启动多步骤长任务或调用 `autonomous` 挂机开发前执行，目标是在写第一行代码前把模糊点与架构冲突消灭干净：

1. **落盘或读取规格草案**
   - 检查是否已有对应 `docs/specs/YYYY-MM-DD-<slug>.md`（初始头部 `status: draft`）；若用户仅给了口头需求，先整理成结构化草案。

2. **五维只读交叉挑刺（以挑错视角逐项核查，禁止此时写业务代码）**
   - **范围与跨端边界**：明确勾选涉及 `server` / `web` / `taro` 哪几端，接口路径是否与现有 `apps/server/docs/openapi.yaml` 冲突。
   - **数据模型与迁移安全**：新增表/字段的类型、唯一约束、外键删除策略是否清晰；对存量表的变更是否满足零停机要求（禁止无 `DEFAULT` 的 `ADD COLUMN NOT NULL` 或 `RENAME`）。
   - **鉴权与 RBAC 闭环**：每个新增接口属于 `Public`、`Authenticated` 还是 `Protected(permCode)`？新权限码命名是否与前端 `<Can perm="...">` 及 `feature-map.ts` 的 `routePerm` 对齐？
   - **事务、审计与并发边界**：多步写操作是否需要包入 `WithinTx` 并记录 `audit.Record`？重试或并发点击是否存在幂等风险？
   - **空态与失败体验**：列表为空、无权限、参数越界或异步任务排队时，前后端各自的确定性表现是什么？

3. **消歧定稿**
   - 若存在高风险业务决策分歧，一次性向开发者列出简明选择题确认；对于常规工程细节，直接给出推荐默认决策写入规格文档。
   - 确认无逻辑矛盾后，将文档 Frontmatter 更新为 `status: accepted`，作为后续 `autonomous` 串行开发的唯一基准。
