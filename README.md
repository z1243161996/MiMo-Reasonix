<p align="center">
  <img src="docs/logo.svg" alt="MiMo-Reasonix" width="640"/>
</p>

<p align="center">
  <strong>English</strong>
  &nbsp;·&nbsp;
  <a href="./README.zh-CN.md">简体中文</a>
  &nbsp;·&nbsp;
  <a href="./README.ja-JP.md">日本語</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/z1243161996/MiMo-Reasonix">Website</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md">Guide</a>
  &nbsp;·&nbsp;
  <a href="./docs/ARCHITECTURE.md">Architecture</a>
  &nbsp;·&nbsp;
  <a href="./benchmarks/">Benchmarks</a>
  &nbsp;·&nbsp;
  <strong><a href="https://discord.gg/XF78rEME2D">Discord</a></strong>
</p>

<p align="center">
  <a href="https://github.com/esengine/MiMo-Reasonix"><img src="https://img.shields.io/github/license/esengine/MiMo-Reasonix.svg?style=flat-square&color=8b949e&labelColor=161b22" alt="license"/></a>
  <a href="https://github.com/esengine/MiMo-Reasonix"><img src="https://img.shields.io/node/v/reasonix.svg?style=flat-square&color=5fa04e&labelColor=161b22&logo=nodedotjs&logoColor=white" alt="node"/></a>
  <a href="https://discord.gg/XF78rEME2D"><img src="https://img.shields.io/badge/discord-join-5865F2.svg?style=flat-square&labelColor=161b22&logo=discord&logoColor=white" alt="Discord"/></a>
</p>

<br/>

<h3 align="center">A MiMo-native AI coding agent for your terminal.</h3>
<p align="center">Fork of DeepSeek-Reasonix, tuned for Xiaomi MiMo models — 1M context, hybrid-attention MoE, prefix-cache optimized.</p>

<br/>

<p align="center">
  <img src="docs/assets/hero-terminal.svg" alt="MiMo-Reasonix code mode — assistant proposes a SEARCH/REPLACE edit; nothing on disk until /apply" width="860"/>
</p>

<br/>

> [!TIP]
> **Cache stability isn't a feature you turn on; it's an invariant the loop is designed around.** The same three-pillar architecture that delivers 99.82% cache-hit rates on DeepSeek applies to MiMo — every layer is tuned to the byte-stable prefix-cache mechanic.

> [!NOTE]
> **MiMo-Reasonix** is a fork of [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) (v0.52.0), re-targeted to Xiaomi MiMo models as the default backend. DeepSeek models remain fully supported as alternatives.

<br/>

## Why MiMo?

MiMo-V2.5 delivers **Pro-level agentic performance at half the cost** of the flagship tier, with a 1M-token context window and native thinking/reasoning support.

| Model | Params | Context | Pricing (CNY / 1M tokens) | Use Case |
|---|---|---|---|---|
| **mimo-v2.5** ★ default | 1T total / 42B active | 1M | ¥0.7 cache read · ¥7 input · ¥14 output | Daily coding, PR review, refactoring |
| mimo-v2.5-pro | 1T total / 42B active | 1M | ¥1.4 cache read · ¥14 input · ¥28 output | Complex architecture, security audits |
| mimo-v2-flash | 309B total / 15B active | 262K | ¥0.07 cache read · ¥0.7 input · ¥2.8 output | Ultra-cheap explore/research subtasks |
| mimo-v2-omni | — | 262K | ¥0.56 cache read · ¥2.8 input · ¥14 output | Multimodal (images, audio, video, GUI) |

> **MiMo-V2-Flash is open source** — 309B total / 15B active MoE, ranks Top 2 globally among open-source models on agent benchmarks. Coding capability surpasses all open-source models and rivals Claude 4.5 Sonnet, at **2.5% of the inference cost** and **2× faster generation speed**.

<br/>

## Install

Requires Node ≥ 22. Works on macOS · Linux · Windows.

~~~bash
npm install -g mimo-reasonix
mimo-reasonix code my-project   # paste a MiMo API key on first run
~~~

Or one-shot:

~~~bash
cd my-project
npx mimo-reasonix code
~~~

**Short alias:** `mrnx` is available as a shorter command:

~~~bash
npm install -g mimo-reasonix   # exposes both `mimo-reasonix` and `mrnx`
mrnx code my-project
~~~

| Command | When |
|---|---|
| `mimo-reasonix` / `mimo-reasonix code [dir]` | The coding agent. **Start here.** |
| `mimo-reasonix chat` | Plain chat — no filesystem or shell tools. |
| `mimo-reasonix run "task"` | One-shot, streams to stdout. |
| `mimo-reasonix doctor` | Health check: Node, API key, MCP wiring. |

### Switching between MiMo and DeepSeek

```bash
mimo-reasonix code --model deepseek-v4-flash   # use DeepSeek for a session
mimo-reasonix code --model mimo-v2.5-pro       # upgrade to MiMo Pro tier
```

Or persistently in `~/.mimo-reasonix/config.json`:

```json
{ "baseUrl": "https://api.deepseek.com", "model": "deepseek-v4-flash" }
```

<br/>

## What stays the same

MiMo-Reasonix inherits the full architecture from DeepSeek-Reasonix:

### Pillar 1 — Cache-First Loop

Context partitioned into ImmutablePrefix / AppendOnlyLog / VolatileScratch — keeps the byte prefix stable across turns so both MiMo and DeepSeek prefix caches stay at 99%+ hit rates.

### Pillar 2 — Tool-Call Repair

Flatten (deep schema) → Scavenge (missed tools) → Truncation (JSON repair) → Storm (repeat-call detection). MiMo's pure OpenAI-compatible tool calls eliminate the DSML hallucination cleanup needed for DeepSeek.

### Pillar 3 — Cost Control

Flash-first defaults (mimo-v2.5), auto-compaction of tool results >3000 tokens, model self-report escalation (`<<<NEEDS_PRO>>>`), and transparent per-turn cost badges.

<br/>

## How it compares

| | MiMo-Reasonix | Claude Code | Cursor | Aider |
|---|---|---|---|---|
| Backend | **MiMo** + DeepSeek | Anthropic | OpenAI / Anthropic | any |
| Default model | **mimo-v2.5** (1M ctx) | Sonnet 4.6 | GPT-5 / Sonnet | configurable |
| License | **MIT** | closed | closed | Apache 2 |
| Prefix-cache optimized | **engineered** | not applicable | not applicable | incidental |
| Cost profile | **¥7/M input** | $3/M input | subscription + use | varies |
| Web dashboard | yes | — | n/a | — |
| Plan mode · MCP · hooks · skills | yes | yes | yes | partial |

<br/>

## Configuration

One JSON file at `~/.mimo-reasonix/config.json` plus per-project overrides under `<project>/.mimo-reasonix/`.

| Topic | Quick read |
|---|---|
| [MCP servers](https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md#mcp) | stdio · SSE · Streamable HTTP |
| [Skills](https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md#skills) | Markdown playbooks, `inline` or `subagent` mode |
| [Memory](https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md#memory) | User-private knowledge pinned into prefix |
| [Hooks](https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md#hooks) | PreToolUse · PostToolUse · UserPromptSubmit · Stop |
| [Permissions](https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md#permissions) | Per-workspace shell allowlist |
| [Web search](https://github.com/z1243161996/MiMo-Reasonix/blob/main/docs/CLI-REFERENCE.md#search) | Mojeek / SearXNG / Metaso |

<br/>

## Documentation

- [**Architecture**](./docs/ARCHITECTURE.md) — three pillars: cache-first loop, tool-call repair, cost control
- [**CLI Reference**](./docs/CLI-REFERENCE.md) — every subcommand, slash command, keybinding
- [**Benchmarks**](./benchmarks/) — τ-bench-lite harness, transcripts, cost methodology
- [**Website**](https://github.com/z1243161996/MiMo-Reasonix)
- [**Contributing**](./CONTRIBUTING.md) · [**Code of Conduct**](./CODE_OF_CONDUCT.md)

<br/>

## Community

> MiMo-Reasonix is a community fork of DeepSeek-Reasonix. Original project: [esengine/DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix).

Scoped starter tickets under the [`good first issue`](https://github.com/esengine/MiMo-Reasonix/labels/good%20first%20issue) label.

<br/>

## Acknowledgments

Built on [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) by [@esengine](https://github.com/esengine) and the open-source community. MiMo model information sourced from [LobeHub model-bank](https://github.com/lobehub/lobehub).
