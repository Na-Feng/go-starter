# Taro Starter（微信小程序）

与 `go-starter` 后端基座无缝对接的 Taro 4 (React + TypeScript) 小程序骨架。

## 快速开始

```bash
# 1. 安装依赖
pnpm install

# 2. 启动 Go 后端后同步 OpenAPI 类型
pnpm gen:api

# 3. 运行静态约束与核心逻辑单测
pnpm check

# 4. 编译微信小程序（产出在 dist/，可用微信开发者工具直接打开或通过工具 Skills 验收）
pnpm dev:weapp
```
