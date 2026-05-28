# DeepSeek-Reasonix — 工程知识库

> 基于 v0.52.0 代码库扫描生成，最后更新 2026-05-28

## 项目身份

- **名称**：Reasonix（npm: `reasonix`，别名: `dsnix`）
- **类型**：DeepSeek 原生 AI 编码智能体 CLI + 桌面客户端
- **仓库**：https://github.com/esengine/DeepSeek-Reasonix
- **协议**：MIT
- **引擎**：TypeScript 5.6+ / ES2022 / ESM / Node ≥ 22

## 技术栈

| 层 | 技术 |
|---|---|
| 语言 | TypeScript 5.6+, ES2022, ESM |
| CLI 框架 | Commander.js |
| TUI | Ink 5 (React 18, 内树 fork) |
| 桌面 | Tauri v2, React, Virtuoso |
| 测试 | Vitest 2.x, Stryker Mutator, Testing Library |
| Lint/格式 | Biome 1.9 |
| 构建 | tsup (打包), tsx (开发运行器) |
| MCP | stdio + SSE + Streamable HTTP |
| Schema 验证 | Zod 4.x |

## 核心架构：三大支柱

### Pillar 1 — 缓存优先循环

Context 三区域划分：ImmutablePrefix（系统提示+工具定义，固定）→ AppendOnlyLog（对话历史，仅追加）→ VolatileScratch（每轮重置）。保证 DeepSeek 前缀缓存字节级稳定。

### Pillar 2 — 工具调用修复

四道工序：Flatten（深度 schema 展平）→ Scavenge（从 reasoning_content 中挖掘遗漏调用）→ Truncation（JSON 截断修复）→ Storm（重复调用风暴检测）。

### Pillar 3 — 成本控制

Flash-first 默认 → Auto 失败升级 → Pro 可选。辅助调用硬编码 flash。工具结果自动压缩（>3000 tokens）。NEEDS_PRO 自升级机制。

## 模块地图

```
src/
├── client.ts            — DeepSeek HTTP 客户端
├── loop.ts              — 核心 agent 循环
├── config.ts            — 配置模型（Zod schema）
├── tools.ts             — ToolRegistry 工具注册表
├── repair/              — 工具调用修复管线
├── mcp/                 — MCP 客户端（3 种传输）
├── core/                — 事件日志内核（Event → Reducer）
├── memory/              — 多级记忆存储
├── code/                — SEARCH/REPLACE 编辑引擎
├── cli/                 — CLI 入口 + Ink TUI
├── desktop/             — Tauri 桌面客户端
├── dashboard/           — Web Dashboard
├── ports/               — 端口接口定义
├── adapters/            — 端口适配器
├── transcript/          — 会话转录/回放
├── telemetry/           — 用量统计
├── i18n/                — 国际化
└── tools/               — 20+ 工具实现
```

## 命令速查

```sh
# 开发
npm run dev           # 运行开发版 CLI
npm run build         # 生产构建
npm run test          # 运行所有测试
npm run test:coverage # 覆盖率报告
npm run test:mutation # 变异测试
npm run lint          # 代码检查
npm run typecheck     # 类型检查
npm run verify        # 全量验证

# 全局安装后
reasonix code [dir]   # 编码模式
reasonix chat         # 纯对话模式
reasonix run "task"   # 一次性任务
reasonix doctor       # 健康检查
reasonix update       # 升级
```

## 代码约定

- 命名导出 only，禁用 `export default`
- 类型导入：`import type` 显式声明
- Biome 格式：2 空格，双引号，始终分号，行宽 100
- 测试无全局变量，forks 池隔离
- 注释默认为零，仅 why 不明确时添加（`comment-policy.test.ts` 强制执行）
- SEARCH 块必须字节精确匹配
- `dist/`、`.reasonix/semantic/`、`sessions/` 禁止手动编辑

## 架构不变量

- `ImmutablePrefix`：每会话计算一次，哈希，固定
- `AppendOnlyLog`：仅追加，无重排，无行内编辑
- `VolatileScratch`：每轮重置
- 工具结果 >3000 tokens 自动压缩
- 辅助 API 调用硬编码 v4-flash
- 非只读工具不可并行调度
- MCP 工具默认非并行安全

## 关键数字

- 源码：~84,600 行，425 个源文件
- 测试：282 个测试文件，~53,500 行
- 提交：1,489 次
- NPM 版本：v0.52.0
- 缓存命中率：99.82%（真实案例）
