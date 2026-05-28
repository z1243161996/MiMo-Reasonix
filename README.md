<p align="center">
  <img src="docs/logo.svg" alt="MiMo-Reasonix" width="640"/>
</p>

<p align="center">
  <a href="./README.en.md">English</a>
  &nbsp;·&nbsp;
  <strong>简体中文</strong>
  &nbsp;·&nbsp;
  <a href="./README.ja-JP.md">日本語</a>
</p>

<br/>

<h3 align="center">MiMo 原生 AI 编码智能体，运行在你的终端里。</h3>
<p align="center">DeepSeek-Reasonix 社区分支，默认使用小米 MiMo 模型 — 1M 上下文、混合注意力 MoE、前缀缓存优化。</p>

<br/>

> [!NOTE]
> **MiMo-Reasonix** 是 [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) (v0.52.0) 的分支，将默认后端切换为小米 MiMo 模型，DeepSeek 模型仍完全支持。

<br/>

## MiMo 模型与定价

| 模型 | 参数 | 上下文 | 定价 (¥ / 百万 tokens) | 适用场景 |
|---|---|---|---|---|
| **mimo-v2.5** ★ 默认 | 1T total / 42B active | 1M | ¥0.7 缓存 · ¥7 输入 · ¥14 输出 | 日常编码、PR 审查、重构 |
| mimo-v2.5-pro | 1T total / 42B active | 1M | ¥1.4 缓存 · ¥14 输入 · ¥28 输出 | 复杂架构、安全审计 |
| mimo-v2-flash | 309B total / 15B active | 262K | ¥0.07 缓存 · ¥0.7 输入 · ¥2.8 输出 | 超低成本探索子任务 |
| mimo-v2-omni | — | 262K | ¥0.56 缓存 · ¥2.8 输入 · ¥14 输出 | 多模态（图像/音频/视频/GUI） |

> **MiMo-V2-Flash 已开源** — 309B/15B active MoE，全球开源模型智能体 Top 2。编码对标 Claude 4.5 Sonnet，成本仅 2.5%，速度 2×。

<br/>

## 安装

需要 Node ≥ 22。

~~~bash
npm install -g mimo-reasonix
mimo-reasonix code my-project
~~~

短别名 `mrnx`：

~~~bash
mrnx code my-project
~~~

## 三大核心支柱

### Pillar 1 — 缓存优先循环
三区域上下文划分，确保前缀缓存 99%+ 命中率。

### Pillar 2 — 工具调用修复
Flatten → Scavenge → Truncation → Storm 四道工序，MiMo 纯 OpenAI 兼容格式免去 DSML 清洗。

### Pillar 3 — 成本控制
mimo-v2.5 默认，工具结果自动压缩，模型自报升级，透明成本显示。

<br/>

## 致谢

基于 [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) 构建，原作者 [@esengine](https://github.com/esengine)。MiMo 模型信息来源于 [LobeHub](https://github.com/lobehub/lobehub)。
