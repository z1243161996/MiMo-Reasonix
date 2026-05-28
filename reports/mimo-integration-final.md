# Reasonix × MiMo 集成方案（终版）

> 数据来源：lobehub/model-bank 模型卡片、erupts/Mimo.java API 适配器、Reasonix 源码分析  
> 编写日期：2026-05-28

---

## 1. MiMo 模型数据（已确认）

### 1.1 模型家族

| 模型 | 参数量 | Context | 模态 | 发布日期 | 定位 |
|---|---|---|---|---|---|
| **mimo-v2.5-pro** | 1T total / 42B active | 1M tokens | 文本 | 2026-03-03 | 旗舰，对标 Claude Opus 4.6 |
| mimo-v2.5 | 1T total / 42B active | 1M tokens | 全模态 | — | Pro 级别的 50% 成本 |
| mimo-v2-pro | 1T+ total / 42B active | 1M tokens | 文本 | — | 高强度 agent 工作流 |
| mimo-v2-flash | 309B total / 15B active | 262K tokens | 文本 | — | 开源 MoE，极致效率 |
| mimo-v2-omni | — | 262K tokens | 文本+视觉+语音 | — | GUI 操作原生支持 |

### 1.2 MiMo-V2.5-Pro 关键规格

- **架构**：1T total / 42B active hybrid-attention（MoE），1M context window
- **能力**：functionCall ✓, reasoning ✓, search ✓, structuredOutput ✓
- **Agentic 基准**：在 ClawEval、GDPVal、SWE-bench Pro 上对标 Claude Opus 4.6
- **可维持 1000+ 次工具调用**的复杂长程任务
- **API 端点**：`https://token-plan-cn.xiaomimimo.com`（OpenAI 兼容）
- **API 认证**：`token` 参数（小米内部 token-plan 机制）
- **Thinking 模式**：支持 `returnThinking` + `sendThinking`（即支持 reasoning/thinking）
- **缓存**：支持前缀缓存，缓存读取定价（cacheRead 约为缓存未命中输入的 10%）
- **最大输出**：131,072 tokens
- **定价**：CNY，分级缓存定价（0-256K cache read ¥1.4/M → 256K-1M ¥0.7/M → ...）

### 1.3 MiMo-V2-Flash 关键规格

- **架构**：309B total / 15B active MoE，开源
- **Agent 基准**：全球 Top 2 开源模型
- **编码能力**：超越所有开源模型，对标 Claude 4.5 Sonnet
- **效率**：推理成本仅 Claude 的 2.5%，生成速度 2×
- **Context**：262,144 tokens
- **缓存**：cacheRead ¥0.07/M tokens

---

## 2. 与 DeepSeek 的对比（架构层面）

| 维度 | DeepSeek v4-flash/pro | MiMo-V2.5-Pro |
|---|---|---|
| 架构 | 不公开（推测 MoE） | **1T total / 42B active** hybrid-attention MoE |
| Context | **1M tokens** | **1M tokens** ✓（同等） |
| 前缀缓存 | 字节级前缀缓存 | ✓ 有（cacheRead 定价独立） |
| Thinking 模式 | `thinking.type: enabled/disabled` | `returnThinking` + `sendThinking`（推理模式） |
| 工具调用格式 | OpenAI compatible + DSML 扩展 | **OpenAI compatible**（纯粹，无 DSML） |
| 最大输出 | 不公开 | **131,072 tokens** |
| API 兼容 | 类 OpenAI（略有差异） | **纯 OpenAI 兼容**（extends OpenAI class） |
| 定价模型 | USD / 1M tokens | **CNY / 分级缓存** |
| 开源情况 | 闭源 | **V2-Flash 开源** |

### 关键发现

**MiMo 对 Reasonix 集成的适配度远高于预期**：

1. **1M context window** → 与 DeepSeek v4 相同，`DEEPSEEK_CONTEXT_TOKENS` 可直接复用
2. **前缀缓存** → Pillar 1（Cache-First Loop）**完全成立**，三区域架构可直接复用
3. **纯 OpenAI 兼容** → 无需 DSML 清洗，`stripHallucinatedToolMarkup` 可大幅简化甚至移除
4. **Thinking 模式** → `isThinkingModeModel("mimo-v2.5-pro")` 应返回 `true`，`thinkingModeForModel` 应返回 `"enabled"`
5. **无 DSML 扩展** → Tool-call repair 管线中 `scavenge`（从 reasoning_content 挖掘 DSML）可跳过

---

## 3. 集成方案

### 3.1 最小可行集成（零代码改动）

```jsonc
// ~/.reasonix/config.json
{
  "baseUrl": "https://token-plan-cn.xiaomimimo.com/v1",  // ← MiMo 端点
  "apiKey": "sk-mimo-xxx",  // ← token-plan 提供的 key
  "model": "mimo-v2.5-pro",

  "pricingOverride": {
    "mimo-v2.5-pro": {
      "inputCacheHit": 0.0002,    // ¥0.7/M ≈ $0.0001/M（暂按 USD 显示）
      "inputCacheMiss": 0.002,     // ¥14/M ≈ $0.002/M
      "output": 0.004              // ¥28/M ≈ $0.004/M
    }
  },

  "contextWindowOverride": {       // ← 不需要！1M 已匹配
    // 省略，复用 DEEPSEEK_CONTEXT_TOKENS 默认值
  }
}
```

```bash
# 直接可用
reasonix code my-project --model mimo-v2.5-pro
```

### 3.2 轻量适配（推荐，1-2 天开发量）

仅需修改 4 个文件，变更量极小：

```typescript
// === 1. src/loop/thinking.ts ===
// 将 MiMo 加入 thinking 模式白名单
export function isThinkingModeModel(model: string): boolean {
  if (model.includes("reasoner")) return true;
  if (model === "deepseek-v4-flash" || model === "deepseek-v4-pro") return true;
  if (model.startsWith("mimo-")) return true;  // ← 新增：MiMo 支持 thinking
  return false;
}

export function thinkingModeForModel(model: string): "enabled" | "disabled" | undefined {
  if (model === "deepseek-chat") return "disabled";
  if (model.includes("reasoner")) return "enabled";
  if (model === "deepseek-v4-flash" || model === "deepseek-v4-pro") return "enabled";
  if (model.startsWith("mimo-")) return "enabled";  // ← 新增
  return undefined;
}
```

```typescript
// === 2. src/telemetry/stats.ts ===
// 将 MiMo 加入 context window 表
export const DEEPSEEK_CONTEXT_TOKENS: Record<string, number> = {
  "deepseek-v4-flash": 1_000_000,
  "deepseek-v4-pro": 1_000_000,
  "deepseek-chat": 1_000_000,
  "deepseek-reasoner": 1_000_000,
  "mimo-v2.5-pro": 1_000_000,     // ← 新增
  "mimo-v2-pro": 1_000_000,       // ← 新增
  "mimo-v2.5": 1_000_000,         // ← 新增
  "mimo-v2-flash": 262_144,       // ← 新增
};
```

```typescript
// === 3. src/repair/index.ts (可选优化) ===
// MiMo 纯 OpenAI 兼容，无 DSML 格式
// → 当检测到 MiMo 模型时，跳过 DSML 清洗（scavenge 保留通用 JSON 挖掘部分）
```

```typescript
// === 4. src/loop/streaming.ts (如需适配) ===
// 验证 MiMo 的 SSE 流式 chunk 格式是否与 DeepSeek 一致
// 如果不一致，添加 MiMo 专属 parser
```

### 3.3 完整适配（如需深度优化，1-2 周）

```typescript
// === 新增：src/mimo-client.ts ===
// MiMo 专用客户端，继承 DeepSeekClient，覆盖认证逻辑
export class MiMoClient extends DeepSeekClient {
  constructor(opts: MiMoClientOptions) {
    super({
      apiKey: opts.apiKey,
      baseUrl: opts.baseUrl ?? "https://token-plan-cn.xiaomimimo.com/v1",
      timeoutMs: opts.timeoutMs ?? 300_000,  // 内网更短超时
    });
  }

  // MiMo 特有的 API 端点（如 token-plan 认证刷新）
  async refreshToken(): Promise<string> { /* ... */ }
}
```

### 3.4 配置预设

```jsonc
// ~/.reasonix/presets/mimo-pro.json
{
  "baseUrl": "https://token-plan-cn.xiaomimimo.com/v1",
  "model": "mimo-v2.5-pro",
  "editMode": "review",
  "pricingOverride": {
    "mimo-v2.5-pro": {
      "inputCacheHit": 0.00014,     // cacheRead ¥1.4/M → USD
      "inputCacheMiss": 0.0028,      // input ¥14/M
      "output": 0.0056               // output ¥28/M
    }
  }
}

// ~/.reasonix/presets/mimo-flash.json
{
  "baseUrl": "https://token-plan-cn.xiaomimimo.com/v1",
  "model": "mimo-v2-flash",
  "pricingOverride": {
    "mimo-v2-flash": {
      "inputCacheHit": 0.00007,      // cacheRead ¥0.07/M
      "inputCacheMiss": 0.0007,      // input ¥0.7/M
      "output": 0.0028               // output ¥2.8/M
    }
  }
}
```

---

## 4. Pillar 影响评估（修订版）

### Pillar 1 — Cache-First Loop

**✅ 完全成立。** MiMo 支持前缀缓存，Reasonix 的三区域架构可直接复用。

优势：MiMo 的缓存定价模型是**分级阶梯**（cacheRead 前 256K ¥1.4/M → 256K-1M ¥0.7/M），建议 Reasonix 的 `ImmutablePrefix` 大小控制在 256K tokens 以下以命中最优价格层级。

### Pillar 2 — Tool-Call Repair

**⚠️ 需要重新评估。** MiMo 是纯 OpenAI 兼容的工具调用，不存在 DeepSeek 特有的 DSML hallucination 问题。

| 修复工序 | DeepSeek 表现 | MiMo 预期表现 | 建议 |
|---|---|---|---|
| Scavenge (DSML 挖掘) | 必须（DSML 嵌入 think 块） | **不需要**（纯 JSON tool_calls） | 检测到 MiMo 模型时跳过 |
| Flatten (Schema 展平) | 必须（>10 参数丢失） | **待验证** | 保留，但阈值可能不同 |
| Truncation (JSON 修复) | 必须 | 保留（通用问题） | 不变 |
| Storm (重复调用) | 必须 | 保留（通用问题） | 不变，但阈值可能不同 |

### Pillar 3 — Cost Control

**⚠️ 需调整成本显示逻辑。**

- Flash-first 策略：MiMo 有 v2-flash（15B active, 极高效率），可作为 Reasonix 的 "flash" 层
- Auto 升级：v2-flash → v2.5-pro（对标 flash → pro）
- 定价单位：CNY，需要价格换算层

---

## 5. MiMo Skill 设计

```markdown
# .reasonix/skills/mimo-code.md
---
description: MiMo code mode — optimized for Xiaomi internal development workflows
runAs: inline
---

## MiMo 编码模式优化

### 模型选择
- 常规编码：mimo-v2-flash（极致效率，对标 Claude 4.5 Sonnet 编码能力）
- 复杂重构/架构设计：mimo-v2.5-pro（旗舰，对标 Claude Opus 4.6）

### 工具调用
- MiMo 使用标准 OpenAI tool_calls 格式
- 无需 DSML 清洗
- 建议 `REASONIX_PARALLEL_MAX=5`（MiMo 可稳定处理更多并行调用）

### 缓存策略
- MiMo 缓存按字节前缀匹配
- 保持 Reasonix 默认的三区域布局即可
- ImmutablePrefix 控制在 256K tokens 内以命中最优价格层级
```

---

## 6. 实施路径

| 优先级 | 任务 | 工时 | 说明 |
|---|---|---|---|
| **P0** | `baseUrl` 最小配置 + 探针验证 | 0.5h | 先跑通 `reasonix code --model mimo-v2.5-pro` |
| **P0** | thinking.ts 白名单更新 | 5min | 2 行改动 |
| **P1** | stats.ts context window 表更新 | 5min | 5 行改动 |
| **P1** | 工具调用格式探针（50+ 场景） | 2h | 确认 MiMo 工具调用质量 |
| **P1** | pricingOverride 配置 | 10min | 配置文件 |
| **P2** | repair pipeline 调优 | 4h | 基于探针数据重标定阈值 |
| **P2** | MiMo flash/pro 分层切换 | 2h | 实现 auto 升级逻辑 |
| **P3** | MiMoClient 专用客户端 | 4h | token-plan 认证适配 |

---

## 7. 结论

**MiMo 是 Reasonix 可近乎零成本接入的理想后端**。原因：

1. **1M context** + **前缀缓存** → Pillar 1 完全成立
2. **纯 OpenAI 兼容** → 消除了 DeepSeek 最棘手的 DSML 修复需求
3. **OpenAI extends** → 与 DeepSeekClient 代码路径高度一致
4. **Flash 开源 + 极致效率** → 成本可降至 Claude 的 2.5%、生成速度 2×

Reasonix 现有第三方模型支持的 7 个适配点中有 5 个对 MiMo 是"正确但需要"的（白名单更新），而非"缺失需要开发"。真正的差异仅在于 token-plan 认证机制和 CNY 定价体系。
