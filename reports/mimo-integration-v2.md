# Reasonix × MiMo 集成方案 — 重新评估

> 评估日期：2026-05-28  
> 上一版问题：误将 MiMo 当作 MCP 网关服务设计，实际 MiMo 是 LLM 模型 API

---

## 1. 问题重定义

### 1.1 上一版的根本错误

上一版将 MiMo 理解为 Xiaomi PTDE Gateway (cszm-mcp) 的代名词，设计了一套 MCP 工具桥接方案。但证据表明 **MiMo（`mimo-v2.5-pro`）是一个 LLM 模型 API**，在 Reasonix 测试中作为 `thirdPartyModel` 使用：

```typescript
// tests/loop.test.ts:667
const thirdPartyModel = "mimo-v2.5-pro";
```

**正确的命题**：如何让 Reasonix 以 MiMo 作为模型后端运行，而不是 DeepSeek。

### 1.2 两类集成方案对比

| 维度 | ❌ 上一版（错误） | ✅ 正确方案 |
|---|---|---|
| MiMo 是什么 | MCP 网关服务 | OpenAI-compatible LLM API |
| 集成方式 | 在 Reasonix 上注册 90+ MCP 工具 | 替换 DeepSeekClient 为 MiMo 后端 |
| 核心挑战 | API 调用缓存/限流 | 模型行为差异适配 |
| 涉及模块 | tools.ts, mcp/ | client.ts, loop.ts, repair/, thinking.ts, stats.ts |

---

## 2. Reasonix 现有第三方模型支持分析

### 2.1 已支持的适配点

Reasonix 在设计上已预留第三方模型接入路径，从源码中共发现 **7 个适配点**：

```typescript
// 1. 端点覆盖（src/config.ts:670-671）
DEEPSEEK_BASE_URL / DEEPSEEK_API_BASE_URL → baseUrl 重定向
// → MiMo 只需设置 baseUrl 即可路由

// 2. 定价覆盖（src/config.ts:707-720）
config.json: { pricingOverride: { "mimo-v2.5-pro": { ... } } }
// → 支持任意模型的 cost 显示

// 3. Thinking 模式豁免（src/loop/thinking.ts:8-14）
thinkingModeForModel("mimo-v2.5-pro") → undefined
// → 未知模型跳过 extra_body.thinking.type，避免 400

// 4. Reasoning 识别豁免（src/loop/thinking.ts:2-6）  
isThinkingModeModel("mimo-v2.5-pro") → false
// → 不强制 reasoning_content 往返

// 5. Force-summary 模型跟随（src/loop/force-summary.ts:41-45）
// 使用 ctx.model 而非硬编码 "deepseek-v4-flash"
// → 第三方端点不会因不认识的 model name 400

// 6. Effort 枚举过滤（src/cli/ui/effort-choices.ts:7）
// 非 DeepSeek 端点自动隐藏 "max" 选项
// → 用户界面不会展示不支持的能力

// 7. Context window 回退（src/telemetry/stats.ts:45）
DEFAULT_CONTEXT_TOKENS = 131_072  // 第三方模型回退值
// → 1M context 是 DeepSeek 特有的，MiMo 使用回退值
```

### 2.2 现有的差距

| Reasonix 特性 | 深度绑定 DeepSeek | 对 MiMo 的影响 |
|---|---|---|
| 前缀缓存三区域设计 | `ImmutablePrefix` 依赖于 DeepSeek 字节级前缀缓存 | **核心冲突** —— MiMo 若缓存机制不同，整个 Pillar 1 失效 |
| DSML hallucination 清洗 | `stripHallucinatedToolMarkup` 匹配 `｜DSML｜` 和 `<function_calls>` 模式 | MiMo 可能使用不同的工具调用格式，清洗规则需适配 |
| Tool-call repair pipeline | Flatten/Scavenge/Truncation/Storm 针对 DeepSeek 行为调优 | MiMo 故障模式未知，需要基于实测数据重新校准阈值 |
| BPE tokenizer | `data/deepseek-tokenizer.json.gz` V3 tokenizer | MiMo 可能使用不同 tokenizer，token 计数不准 |
| 模型列表获取 | `GET /models` → `ModelList` 使用 DeepSeek API 结构 | MiMo 的 `/models` 返回格式可能不同 |
| Balance 查询 | `GET /user/balance` DeepSeek 特有端点 | MiMo 不支持此端点，余额显示需适配 |
| 价格/成本计算 | `DEEPSEEK_PRICING` 硬编码表格 | 需通过 `pricingOverride` 配置 |
| Context window | 硬编码 `DEEPSEEK_CONTEXT_TOKENS[model]` 或 fallback 131072 | 需确认 MiMo 实际 context 大小 |

---

## 3. 架构影响分析

### 3.1 Pillar 1 — Cache-First Loop 的命运

这是最大的不确定性。Reasonix 的整个架构建立在 DeepSeek 的字节级前缀缓存之上。MiMo 的缓存行为决定了方案走向：

**场景 A：MiMo 有前缀缓存（与 DeepSeek 类似）**

```
✅ Pillar 1 直接复用
  · ImmutablePrefix → 相同策略
  · AppendOnlyLog → 相同策略
  · VolatileScratch → 相同策略
⚠️  需确认：
  · 缓存命中率指标是否暴露（prompt_cache_hit_tokens）
  · 缓存窗口大小（DeepSeek 是完整上下文）
  · 缓存失效条件（是否也按字节匹配？）
```

**场景 B：MiMo 有缓存但机制不同（如语义缓存、KV-cache）**

```
⚠️ Pillar 1 需部分重写
  · ImmutablePrefix 可能仍有价值（减少重复 token 化）
  · AppendOnlyLog 策略可能不必那么严格
  · tokenizer 必须与 MiMo 一致（否则 token 计数错误）
❓ 核心问题：如果缓存不是字节级别，三区域划分的价值还剩多少？
```

**场景 C：MiMo 无缓存**

```
❌ Pillar 1 完全失效
  · ImmutablePrefix → 无意义开销
  · AppendOnlyLog 严格追加 → 无意义约束
  · 整个 context-manager.ts 需要从头设计
  · Reasonix 失去了核心差异化价值
```

**建议**：优先探明 MiMo 的缓存机制，这是决定方案可行性的第一优先级。

### 3.2 Pillar 2 — Tool-Call Repair 的适配

MiMo 的工具调用故障模式可能与 DeepSeek 完全不同，需要：

| DeepSeek 已知故障 | 修复策略 | MiMo 需验证 |
|---|---|---|
| JSON 嵌入 `<think>` 中 | Scavenge 从 reasoning_content 挖掘 | MiMo 是否有 reasoning_content 字段？格式是否一致？ |
| Schema >10 参数丢失 | Flatten 展平 schema | MiMo 对不同复杂度 schema 的容忍度？ |
| 重复调用风暴 | Storm 滑动窗口检测 | MiMo 的重复调用倾向？阈值需要重新标定 |
| JSON 截断 | Truncation 括号补齐 + 续写 | MiMo 的 JSON 完整性表现？ |
| DSML 标记幻觉 | `stripHallucinatedToolMarkup` | MiMo 是否有类似原生标记？需要采集样本 |

**实施路径**：
1. 采集 MiMo 在 50+ 工具调用场景下的原始输出样本
2. 统计故障类型分布
3. 调整 repair pipeline 的参数和优先级
4. 可能新增 MiMo 专属修复工序（如中文括号归一化）

### 3.3 Pillar 3 — Cost Control 的适配

| 机制 | DeepSeek 行为 | MiMo 场景 |
|---|---|---|
| Flash-first 分层 | v4-flash 1×, v4-pro 12× | 内网模型无账单，分层无意义 |
| Auto-compaction (>3000 tokens) | 减少缓存未命中 token | 仍有价值——减少 MiMo 处理量降低延迟 |
| `/model` 切换 | flash ↔ pro | 可切换不同 MiMo 版本（如 v2.5-pro ↔ v2.5-lite） |
| NEEDS_PRO 自升级 | 模型发出标记 → 自动升级 | 可改造为任务复杂度判断 → 切换更强版本 |
| 成本显示 | $/turn, $/session | 改为延迟/token 计数显示 |

---

## 4. 修正方案设计

### 4.1 MiMoClient — 替代 DeepSeekClient

```typescript
// src/mimo-client.ts (新文件，或直接复用 DeepSeekClient)
export class MiMoClient extends DeepSeekClient {
  constructor(opts: MiMoClientOptions) {
    super({
      apiKey: opts.apiKey,
      baseUrl: opts.baseUrl ?? "https://mimo.internal.xiaomi.com/v1",
      timeoutMs: opts.timeoutMs ?? 300_000,  // 内网更短超时
      // ... 其他选项
    });
  }

  // 可选覆盖：MiMo 特有的 chat 参数
  async chat(opts: ChatRequestOptions): Promise<ChatResponse> {
    // 注入 MiMo 特有参数（如 temperature, top_p 默认值差异）
    // 处理 MiMo 特有的响应格式差异
  }
}
```

### 4.2 关键适配清单

```typescript
// === 必须适配 ===

// 1. tokenizer（src/tokenizer.ts）
// DeepSeek V3 tokenizer → MiMo tokenizer
// 影响：token 计数、context 预算、auto-compaction 阈值
// 方案：如果 MiMo 提供 tokenizer 文件则替换 data/ 目录下的 gz 文件
//       如果没有，使用 cl100k_base (GPT-4 tokenizer) 作为近似

// 2. Thinking mode（src/loop/thinking.ts）
// 已有适配：未知模型返回 undefined，MiMo 不会收到 thinking.type
// ⚠️ 如果 MiMo 实际上支持 reasoning/thinking，需要更新白名单
isThinkingModeModel(model) {
  if (model.startsWith("mimo-")) return true; // 如果 MiMo 支持 reasoning
  // ...
}

// 3. DSML 清洗（src/loop/thinking.ts:17-25）
// stripHallucinatedToolMarkup 针对 DeepSeek ｜DSML｜ 格式
// 如果 MiMo 使用不同的标记格式，需要新增清洗规则
// 如果 MiMo 的工具调用更干净，可以减少清洗步骤

// 4. Repair 阈值（src/repair/storm.ts, src/repair/scavenge.ts）
// StormBreaker 默认 window=6, threshold=3
// Scavenge: maxScavenge 等参数
// → 需根据 MiMo 实测数据重新标定

// 5. Cost 模型（src/telemetry/stats.ts）
// pricingOverride 配置 MiMo 定价
// 或在 DEEPSEEK_PRICING 表中增加 MiMo 条目

// === 建议适配 ===

// 6. Context Window（src/telemetry/stats.ts）
// 在 DEEPSEEK_CONTEXT_TOKENS 表中增加 MiMo 的 context 大小

// 7. Context Manager（src/context-manager.ts）
// TURN_START_FOLD_THRESHOLD, TURN_END_RESULT_CAP_TOKENS
// → 根据 MiMo 的 context window 调整

// 8. Model List（src/client.ts:90-99）
// fetchModels() → GET /models
// MiMo 的模型列表格式可能与 DeepSeek 不同

// 9. Balance（src/client.ts:68-78）
// fetchBalance() → GET /user/balance
// MiMo 可能没有此端点，返回 mock 数据或隐藏余额显示

// 10. Parallel dispatch（src/loop/dispatch.ts）
// REASONIX_PARALLEL_MAX, parallelSafe 标记
// → 需确认 MiMo 是否支持并行工具调用

// === 可选适配 ===

// 11. 流式响应解析（src/client.ts）
// SSE 解析逻辑针对 DeepSeek 的 chunk 格式
// 如果 MiMo 的流式格式不同（如使用不同字段名），需要适配

// 12. 错误处理（src/loop/errors.ts）
// is4xxError, is5xxError, isDeepSeekHost
// MiMo 的错误码和处理逻辑可能不同
```

### 4.3 配置模板

```jsonc
// ~/.reasonix/config.json
{
  "baseUrl": "https://mimo.internal.xiaomi.com/v1",
  "apiKey": "sk-mimo-xxxx",
  "model": "mimo-v2.5-pro",

  "pricingOverride": {
    "mimo-v2.5-pro": {
      "inputCacheHit": 0.0005,
      "inputCacheMiss": 0.001,
      "output": 0.002
    },
    "mimo-v2.5-lite": {
      "inputCacheHit": 0.0001,
      "inputCacheMiss": 0.0005,
      "output": 0.001
    }
  },

  "contextWindowOverride": {
    "mimo-v2.5-pro": 262144,
    "mimo-v2.5-lite": 131072
  }
}
```

### 4.4 MiMo 探针方案

在正式集成前，需要一套探针脚本验证 MiMo 的关键行为：

```bash
# 1. 基础连通性探针
curl -s https://mimo.internal.xiaomi.com/v1/models \
  -H "Authorization: Bearer $MIMO_API_KEY" | jq .

# 2. 缓存行为探针（多次发送相同 prefix）
npx tsx tools/bench-fold-cache-live.mjs \
  --endpoint https://mimo.internal.xiaomi.com/v1 \
  --model mimo-v2.5-pro

# 3. 工具调用鲁棒性探针（复杂 schema）
npx tsx tools/probe-deepseek-body-limit.mjs \
  --endpoint https://mimo.internal.xiaomi.com/v1 \
  --model mimo-v2.5-pro

# 4. Tokenizer 对齐验证
npx tsx scripts/prepare-tokenizer.ts \
  --source mimo \
  --output data/mimo-tokenizer.json.gz
```

---

## 5. 实施优先级

| 优先级 | 任务 | 理由 |
|---|---|---|
| **P0** | 缓存行为探针 | 决定 Pillar 1 是否成立——如果无缓存，整个架构价值减半 |
| **P0** | 工具调用鲁棒性探针 | 决定 Pillar 2 需要多大改动 |
| **P1** | baseUrl + pricingOverride 最小配置 | 用现有 `DeepSeekClient` 直连 MiMo，验证基本可行性 |
| **P1** | thinking mode 白名单更新 | 如果 MiMo 支持 reasoning 则开启 |
| **P2** | tokenizer 替换 | 精准的 token 计数和 context 管理 |
| **P2** | repair pipeline 参数重新标定 | 基于 P0 探针数据 |
| **P3** | context manager 阈值调整 | 基于 MiMo 实际 context window |
| **P3** | DSML 清洗规则适配 | 需要 MiMo 实际输出样本 |

## 6. 风险评估

| 风险 | 概率 | 影响 | 缓解措施 |
|---|---|---|---|
| MiMo 无前缀缓存 | 中 | 高 — Pillar 1 完全失效 | P0 探针首先验证；若无缓存，考虑仅用 MiMo 做一次性任务（`reasonix run`） |
| MiMo 工具调用质量差 | 中 | 高 — Pillar 2 需要大量定制 | P0 探针评估；如果太差则不可行 |
| MiMo API 非标准 OpenAI 格式 | 低 | 中 — 需要 Fork `DeepSeekClient` | 已知在 Reasonix 测试中以 `thirdPartyModel` 身份通过 |
| MiMo tokenizer 不公开 | 中 | 中 — token 计数不准 | 使用 cl100k_base 近似；context manager 保守估计 |
| MiMo API 不稳定（内网测试服务） | 高 | 中 — 可用性依赖网络和运维 | retry 机制可以吸收部分抖动 |

## 7. 总结

上一版方案的根本错误在于混淆了 **MiMo（LLM 模型）** 和 **PTDE Gateway（MCP 工具网关）**。修正后的方案聚焦于：

1. **MiMo 作为模型后端替换 DeepSeek**，复用 Reasonix 已有的 7 个第三方适配点
2. **Pillar 1 缓存策略是否成立**是第一优先级问题，需 P0 探针验证
3. **Pillar 2 修复管线需基于 MiMo 实测数据重新标定**，而非假设与 DeepSeek 相同
4. **Pillar 3 成本控制**从 token 价格转为延迟和吞吐量优化
5. 最小可行集成只需 `baseUrl` + `pricingOverride` 配置，其余按需渐进适配
