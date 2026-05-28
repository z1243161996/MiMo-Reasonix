# Reasonix × 小米 MiMo 集成方案设计

> 基于 Reasonix v0.52.0 架构知识 + 小米 MiMo API 生态系统  
> 设计日期：2026-05-28

---

## 1. MiMo API 生态特征分析

### 1.1 网关架构

小米 PTDE Gateway 采用 **统一网关 + apiCode 路由** 模式：

```
                                    ┌─ dataplatform (数据上报)
                                    ├─ projecthome (项目首页)
                                    ├─ projectoverview (项目总览)
Client ──► PTDE Gateway ──►─────────├─ projectplan (测试策略)
  (cszm-mcp)         /api/{apiCode} ├─ projectrequirement (需求/任务)
                                    ├─ micasepool (用例管理)
                                    ├─ projectrisk (风险/问题)
                                    ├─ projectmanpower (人力/工时)
                                    ├─ projectservice (操作记录)
                                    ├─ miatp (自动化测试)
                                    ├─ alkaid-front (硬测平台)
                                    ├─ qualityBoard (TR评审)
                                    ├─ reportcenter (报告中心)
                                    └─ cltask (CL分析)
```

### 1.2 关键特征

| 特征 | 详情 |
|---|---|
| 认证模型 | AK/SK（cszm-mcp）+ 用户 token（issue-service）+ Jira token |
| 路由方式 | apiCode 映射（非路径直连），例如 `POST 4bccee180ebf` → 数据上报 |
| 鉴权注入 | Header 透传 `projectCode`、`projectStageId`、`versionUpgradeId` |
| 网络约束 | 需小米内网或 VPN，npm 源为内部 registry |
| 工具暴露 | cszm-mcp 以 MCP stdio 形式暴露 14 个服务、90+ 个 API |
| 跨服务关联 | 需求(projectrequirement) ↔ 任务 ↔ 用例(micasepool) ↔ 缺陷(mi-jira) |

### 1.3 与 DeepSeek API 的差异

| 维度 | DeepSeek API | MiMo API (cszm-mcp) |
|---|---|---|
| 协议 | HTTP SSE (REST) | MCP stdio (JSON-RPC) |
| 调用模式 | 单向：发送消息 → 接收流式响应 | 工具调用：LLM 选择工具 → 返回结构化结果 |
| 缓存机制 | 前缀缓存（字节级匹配） | 无内置缓存（每次 MCP 调用独立） |
| 成本模型 | token × 单价 | 内部服务，无 token 成本 |
| 响应格式 | 流式文本/JSON | 结构化 JSON（按 apiCode 路由到不同服务） |

---

## 2. 方案设计：Reasonix 作为小米研发 Agent 平台

### 2.1 总体架构

```
┌──────────────────────────────────────────────────────┐
│                  Reasonix Core                       │
│  ┌────────────┐  ┌────────────┐  ┌───────────────┐  │
│  │CacheFirst  │  │ Tool-Call  │  │  Cost Control │  │
│  │   Loop     │  │   Repair   │  │  (flash-first)│  │
│  └─────┬──────┘  └─────┬──────┘  └───────┬───────┘  │
│        │               │                  │          │
│  ┌─────▼───────────────▼──────────────────▼───────┐  │
│  │              MiMo Tool Bridge                  │  │
│  │  ┌──────────────────────────────────────────┐  │  │
│  │  │  MiMoGatewayClient (MCP transport)       │  │  │
│  │  │  · 14 service adapters                   │  │  │
│  │  │  · apiCode → ToolSpec 映射               │  │  │
│  │  │  · 鉴权 header 自动注入                  │  │  │
│  │  │  · 结果缓存 + 压缩                       │  │  │
│  │  └──────────────────────────────────────────┘  │  │
│  └──────────────────────┬─────────────────────────┘  │
└─────────────────────────┼────────────────────────────┘
                          │ MCP stdio
┌─────────────────────────▼────────────────────────────┐
│                  cszm-mcp Server                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │           PTDE Gateway (one.mi.com)             │ │
│  │  /api/4bccee180ebf  /api/37698e144abf  ...      │ │
│  └─────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
```

### 2.2 核心模块：MiMo Tool Bridge

这是本方案的核心创新 —— 将 Reasonix 的三支柱架构适配到 MiMo API 环境。

#### 2.2.1 Cache-First Loop 适配

MiMo MCP 工具无原生缓存，但 Reasonix 可通过 **三层缓存策略** 降低延迟和重复调用：

```
┌─ L1: 内存会话缓存 ──────────────────────────────┐
│  projectCode / stageId → 项目元信息               │
│  TTL: 会话级别，首次查询后固定                      │
├─ L2: 工具结果缓存 ───────────────────────────────┤
│  apiCode + args fingerprint → 结构化结果           │
│  TTL: 按工具类型差异化（用例列表 30min/统计数据 5min）│
├─ L3: AppendOnlyLog 去重 ─────────────────────────┤
│  同一会话内相同查询 → 引用已有结果而非重新请求        │
└──────────────────────────────────────────────────┘
```

**关键设计决策**：
- 查询类工具（`query*`, `list*`, `get*`）启用缓存
- 变更类工具（`create*`, `update*`, `delete*`）旁路缓存，执行后失效相关查询缓存
- 统计类工具使用短 TTL（默认 5 分钟）

#### 2.2.2 Tool-Call Repair 适配

MiMo API 的典型故障模式与 DeepSeek 原生工具不同，需要定制修复管线：

| 故障模式 | DeepSeek 原生 | MiMo API | 修复策略 |
|---|---|---|---|
| 参数错误 | JSON 截断/丢失 | auth header 缺失、apiCode 错误 | schema 校验 + 重试指引 |
| 重复调用 | 相同 args 风暴 | 分页重复请求 | StormBreaker 窗口检测 |
| 超时/限流 | HTTP 429 | Gateway 超时 | 指数退避 + 降级策略 |
| 权限不足 | — | 403 Forbidden | 诊断反馈 → 提示用户配置 |
| 数据为空 | — | 合法但空结果 | 区分"无数据"与"查询失败" |

**定制修复管线**：

```typescript
// MiMo 四道修复工序
class MiMoRepairPipeline {
  // 1. AuthGuard — 检测 401/403，提示配置 header
  // 2. PaginationFlattener — 自动合并分页结果（max 5 页）
  // 3. ResultCompactor — 超过 5000 字符的结果自动摘要
  // 4. CrossServiceLinker — 检测跨服务引用，自动关联查询
}
```

#### 2.2.3 Cost Control 适配

MiMo 无 token 成本，但需要控制 **调用频率** 和 **响应体积**：

| 策略 | DeepSeek 原版 | MiMo 适配 |
|---|---|---|
| 模型分级 | flash → pro | 无模型概念，跳过 |
| 自动压缩 | >3000 tokens 结果压缩 | >5000 chars 结果摘要 |
| 子 Agent | flash 硬编码 | Gateway 调用计数 + 限流 |
| 成本显示 | $/turn | API 调用次数/耗时 |

### 2.3 Reasonix Skill 设计

为 MiMo 场景创建一组专用 Skills，利用 Reasonix 的 Skill 系统（Markdown 剧本 + subagent 模式）：

```
.reasonix/skills/
├── mimo/
│   ├── query-project.md        # 项目信息查询
│   ├── query-requirement.md    # 需求查询与分析
│   ├── query-testcase.md       # 用例查询与统计
│   ├── query-task.md           # 任务看板与进度
│   ├── query-risk.md           # 风险/问题管理
│   ├── create-issue.md         # 创建问题单
│   ├── report-weekly.md        # 周报生成
│   └── code-review-bridge.md   # 代码审查 → 问题跟踪
```

**Skill 示例：周报生成**

```markdown
---
description: 基于项目空间生成测试周报
runAs: subagent
tools: cszm-mcp
parallelSafe: true
---

# 测试周报生成器

## 输入
- projectCode: 项目编码
- projectStageId: 项目阶段 ID

## 流程
1. 查询项目基本信息 (projectoverview/summary/index/noticeconfig)
2. 查询需求列表与测试跟踪 (projectrequirement/rmProject/statisticByTestTrack)
3. 查询任务看板 (projectrequirement/testTask/board)
4. 查询风险列表 (projectrisk/projectRisk/board)
5. 查询用例执行统计 (micasepool/case/statistics)
6. 汇总生成飞书文档

## 输出
- Markdown 格式周报
- 可选：直接写入飞书文档
```

### 2.4 鉴权与配置模型

```typescript
// ~/.reasonix/mimo-config.ts
interface MiMoConfig {
  // PTDE Gateway 认证
  gateway: {
    accessKey: string;          // AK
    secretKey: string;          // SK
    baseUrl?: string;           // 默认 one.mi.com
  };

  // 项目上下文（自动注入 header）
  projectContext: {
    projectCode: string;
    projectStageId?: number;    // 项目阶段 与 大版本升级 二选一
    versionUpgradeId?: number;
  };

  // 可选：Jira 集成
  jira?: {
    token: string;
    url: string;                // jira-phone.mioffice.cn
    username: string;
  };

  // 可选：issue service 集成
  issueService?: {
    token: string;
    user: string;
  };

  // 缓存配置
  cache: {
    queryTtlMs: number;         // 默认 600_000 (10min)
    statsTtlMs: number;         // 默认 300_000 (5min)
    maxResultChars: number;     // 默认 5000
  };

  // 限流配置
  rateLimit: {
    maxConcurrent: number;      // 默认 4
    minIntervalMs: number;      // 默认 200
  };
}
```

### 2.5 工具映射：apiCode → ToolSpec

cszm-mcp 的 90+ 个 API 需要按使用场景映射为 Reasonix ToolSpec：

```typescript
// 高频查询工具（启用缓存 + 并行安全）
const QUERY_TOOLS: ToolDefinition[] = [
  {
    name: "mimo_project_info",
    description: "查询项目基本信息",
    parameters: { projectCode: z.string() },
    parallelSafe: true,
    cacheTtlMs: 600_000,
  },
  {
    name: "mimo_task_board",
    description: "查询项目任务看板（按状态泳道聚合）",
    parameters: {
      projectCode: z.string(),
      filterFlag: z.number().optional(),
      testType: z.string().optional(),
    },
    parallelSafe: true,
    cacheTtlMs: 120_000, // 任务看板 2min 短缓存
  },
  // ... 更多工具
];

// 变更工具（禁用缓存）
const MUTATION_TOOLS: ToolDefinition[] = [
  {
    name: "mimo_create_issue",
    description: "创建问题单",
    parameters: { /* ... */ },
    readOnly: false,
    cacheTtlMs: 0, // 不缓存
  },
];
```

---

## 3. 部署方案

### 3.1 环境要求

| 组件 | 要求 |
|---|---|
| Node.js | ≥ 22 |
| Reasonix | npm install -g reasonix |
| cszm-mcp | npm install -g @cnpm-test/cszm-mcp |
| 网络 | 小米内网 / VPN（pkgs.d.xiaomi.net + one.mi.com） |
| 认证 | AK/SK 配置在 `~/.reasonix/mimo.json` |

### 3.2 配置步骤

```bash
# 1. 安装 Reasonix
npm install -g reasonix

# 2. 安装 MiMo MCP bridge
npm install -g --registry=https://pkgs.d.xiaomi.net/artifactory/api/npm/mi-npm/ @cnpm-test/cszm-mcp

# 3. 配置 ~/.reasonix/mimo.json
cat > ~/.reasonix/mimo.json << 'EOF'
{
  "gateway": {
    "accessKey": "<your-ak>",
    "secretKey": "<your-sk>"
  },
  "projectContext": {
    "projectCode": "O16U"
  },
  "cache": {
    "queryTtlMs": 600000,
    "statsTtlMs": 300000,
    "maxResultChars": 5000
  }
}
EOF

# 4. 启动 Reasonix + MiMo
reasonix code --skill mimo
```

### 3.3 多项目切换

```bash
# 在项目目录下设置 .reasonix/mimo.json
cd /path/to/project-A
echo '{"projectContext":{"projectCode":"O16U"}}' > .reasonix/mimo.json

cd /path/to/project-B
echo '{"projectContext":{"projectCode":"O17"}}' > .reasonix/mimo.json
```

---

## 4. 典型场景

### 4.1 测试进度查询

```
用户: 看一下 O16U 项目当前的测试进度

Agent:
  → mimo_project_info(O16U)                  [缓存命中, 0ms]
  → mimo_test_statistics(O16U)               [查询中...200ms]
  → mimo_task_board(O16U, filterFlag=0)      [查询中...350ms]

输出:
  · 需求总数: 42 (已评估: 38, 未评估: 4)
  · 任务看板: 就绪 12 | 进行中 8 | 已完成 15 | 新任务 7
  · 用例执行率: 73.5% (通过率: 91.2%)
  · 风险: 高风险 2, 中风险 5
```

### 4.2 缺陷分析与关联

```
用户: 分析 ISS-202605-0000123A 这个缺陷，关联需求、用例和任务

Agent:
  → mimo_issue_detail(ISS-202605-0000123A)    [查询问题详情]
  → mimo_jira_info(jiraId=XMCR-12345)          [关联 Jira]
  → mimo_cases_by_req(requirementNo)            [关联用例]
  → mimo_task_by_code(taskCode)                 [关联任务]

输出:
  缺陷链路图 + 影响分析 + 建议处理方案
```

### 4.3 周报自动生成

```
用户: /skill mimo-weekly

Agent (自动执行):
  1. 项目概览数据采集       [projectoverview APIs]
  2. 需求/任务/用例统计     [projectrequirement APIs]
  3. 风险汇总              [projectrisk APIs]
  4. 缺陷趋势              [jira APIs]
  5. 飞书文档写入          [feishu-mcp-pro]
  6. 发送通知              [feishu bot]

输出: 飞书文档链接 + 摘要
```

---

## 5. 方案优势

| 维度 | 说明 |
|---|---|
| **零额外成本** | MiMo 调用无 token 费用，仅需内网连接 |
| **缓存加速** | 项目元信息/统计查询缓存，消除重复请求 |
| **自然语言交互** | 通过 Reasonix CLI/TUI，用中文直接查询 MiMo 平台数据 |
| **跨服务关联** | 自动关联需求-任务-用例-缺陷，生成全链路视图 |
| **可扩展** | Skill 系统支持自定义工作流（周报、问题分析、Test Plan 生成） |
| **离线友好** | 三级缓存，VPN 断开时仍可查询历史数据 |

## 6. 实施路径

| 阶段 | 任务 | 预计工期 |
|---|---|---|
| **P0** | MiMoGatewayClient MCP transport 实现 | 2 周 |
| **P0** | 核心 10 个查询工具注册 + 缓存层 | 1 周 |
| **P1** | MiMoRepairPipeline（4 道修复工序） | 1 周 |
| **P1** | 6 个常用 Skill（项目查询/需求分析/周报等） | 1 周 |
| **P2** | 变更工具注册（创建 Issue/评论等） | 1 周 |
| **P2** | CrossServiceLinker 自动关联 | 1 周 |
| **P3** | 桌面端集成 + QQ 频道接入 | 2 周 |
