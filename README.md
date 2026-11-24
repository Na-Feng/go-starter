# Fullstack Starter (Go + React Web + Taro)

AI 时代高确定性全栈开发基座 Monorepo。

## 初始化（首次拉取到本地后执行）

```bash
# 1. 拉取 Go 依赖并生成 go.sum
cd apps/server && go mod tidy && cd ../..

# 2. 安装前端与小程序依赖（已通过 .npmrc 隔离 workspace 依赖）
pnpm install

# 3. 准备 PostgreSQL 与 Redis（本机已安装服务可直接跳过；若用 Docker 则执行 make up）
#    后端配置位于 apps/server/configs/config.yaml（本地私有覆盖可新建 config.local.yaml），首次运行会自动建库

# 4. 同步 OpenAPI 契约并跑通三端全量检查（含真实库集成测试）
make gen-api && make check-all && make test-int
```

## 日常启动
- 后端：`make dev-server`（默认管理员 `admin` / `admin123`，接口文档 `http://localhost:8080/docs`）
- Web 后台：`make dev-web`（`http://localhost:5173`）
- Taro 小程序：`make dev-taro`（用微信开发者工具打开 `apps/taro`）

## 跨工具通用 AI 工作流剧本（`.agents/skills/`）
采用开放 Agent Skills 标准路径 `.agents/skills/<name>/SKILL.md`（兼容 Devin、Codex、Cursor、Windsurf 等主流 Agent 自动发现，其他工具亦可通过根 `AGENTS.md` 任务路由表按路径直接加载）：
- `add-feature`：按范围裁剪的新增/改造功能剧本（Go + OpenAPI 契约 + Arco Web + Taro）
- `verify-all`：按改动端执行静态检查、`test-int`、运行时实测与 `git diff` 净度清理剧本
- `reflect`：故障复盘与“信任倒金字塔”底层约束下沉剧本
- `spec-check`：长任务开工前规格只读挑刺与消歧定稿剧本
- `autonomous`：长任务无人值守剧本（`auto/*` 单分支原子串行、零提问记假设、2 次受阻根因会诊、审查封顶防死循环与 CI 自动合入）

---

## 新业务开发与 Agent 交互指南

工程约束与编码步骤已固化在代码库守卫和 Skills 中。向 Agent 派活时**无需编写实现步骤说明**，只需交代 **「涉及端范围 + 业务规则/权限 + 验收证据」**：

### 1. 开新项目初始化（拉模板建新仓后的第 1 条指令）
```text
基于当前基座初始化新项目 <project-name>：把 Go module 名、默认库名 starter / starter_test（configs/config.yaml）、.golangci.yml 与 arch_test.go 里的包前缀统一改为 <project-name>，按需清理 sample 示例业务模块（同步更新 openapi.yaml、迁移与双端示例页），最后跑通 make gen-api && make check-all && make test-int。
```

### 2. 日常中小型需求（占 80% 场景）
- **单点小改（改文案、调字段、修样式 —— 不调工作流剧本，秒改秒测）**
  ```text
  【仅 web】在用户管理表格增加一列“创建时间”，跑通 make check-web。
  ```
- **常规业务功能（10~20 分钟级切片 —— 锁死端范围，走 `add-feature`）**
  ```text
  按 add-feature 新增【商品品牌管理】模块（范围：server + web，不改 taro）：
  1. 字段：品牌名（2~50 字唯一）、Logo 地址（复用上传接口）、排序权重、启停状态；
  2. 权限：读 catalog:brand:read，写 catalog:brand:write，写操作进事务并记审计日志；
  3. 跑通 make check-all && make test-int，并启动服务用浏览器实测新增与启停。
  ```

### 3. 复杂跨端大模块 / 挂机长任务（两窗口接力法）
1. **窗口一：3 分钟规格排雷定稿（调 `spec-check`）**
   ```text
   我要做【优惠券发放与核销】模块（涉及 server + web + taro）：后台建券配置库存与有效期，小程序领券并在下单时防超领核销。请按 spec-check 生成 docs/specs/ 规格草案，挑出权限、并发超领和异常退回的边界漏洞供我确认。
   ```
   确认关键业务分歧、将规格头部更新为 `status: accepted` 并提交。
2. **窗口二：另开全新干净会话挂机执行（调 `autonomous`）**
   ```text
   按 autonomous 剧本完整实现 docs/specs/YYYY-MM-DD-coupon.md，在 auto/coupon 分支原子串行完成后推送并输出核对报告。
   ```
   新开窗口可避免前期讨论污染上下文；跑完直接审阅输出的【规格逐条核对表 + 默认假设清单 + `[!]` 遗留项】。

### 4. 修 Bug 与被打回时的复利沉淀（调 `reflect`）
```text
修好这个并发重复核销的 Bug 并补上集成测试；修完后按 reflect 复盘为什么现有检查未能提前拦截，给出一条下沉到类型、AST 守卫或状态机测试的加固方案。
```

### 5. 高效协作的 3 个习惯
- **一模块一窗口**：每完成一个独立模块提交后即新开对话窗口——持久记忆已在 `openapi.yaml`、`feature-map.ts`、`db/migrations/` 与 `docs/specs/` 中，短上下文准确度最高且最省 Token。
- **审骨架与证据而非逐行盯实现**：优先检查 `db/migrations/*.sql`、`openapi.yaml`、`feature-map.ts` 三处真源与 `make check-all && make test-int` / 浏览器实测结果。
- **守住底座边界**：常规业务需求默认不应修改 `internal/kernel/`、`arch_test.go`、`eslint.config.js` 或引入新第三方依赖；若 Agent 申请触碰，务必先确认必要性。

---

## 开启无人值守 Auto-merge（GitHub 仓库设置）
在 GitHub 仓库的 `Settings -> General -> Pull Requests` 中勾选 **Allow auto-merge** 与 **Allow squash merging**，并在 `Settings -> Actions -> General -> Workflow permissions` 允许 GitHub Actions 创建与批准操作后：
- 普通业务 PR 在 `.github/workflows/ci.yml`（`check-all` + `integration-test` + `risk-gate`）全绿后将自动 Squash 合入 `main`；
- 触碰高危路径（`internal/kernel/`、AST/Lint 守卫配置、`.github/`、`.agents/`、`AGENTS.md` 或迁移中含 `DROP`/`TRUNCATE`）的 PR 会自动挂起等待人工审批。
