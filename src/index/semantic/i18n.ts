/** EN+ZH for semantic-search prompts only; tool descriptions stay English to preserve prompt-cache. */

export type Locale = "en" | "zh";

let cachedLocale: Locale | null = null;

export function detectLocale(): Locale {
  if (cachedLocale) return cachedLocale;
  const override = (process.env.MIMO_REASONIX_LANG ?? "").toLowerCase();
  if (override === "zh" || override === "en") {
    cachedLocale = override;
    return cachedLocale;
  }
  const env = process.env.LANG ?? process.env.LC_ALL ?? process.env.LC_MESSAGES ?? "";
  if (/^zh[-_]/i.test(env)) {
    cachedLocale = "zh";
    return "zh";
  }
  try {
    const sys = new Intl.DateTimeFormat().resolvedOptions().locale ?? "";
    if (/^zh[-_]/i.test(sys)) {
      cachedLocale = "zh";
      return "zh";
    }
  } catch {
    /* ignore — fall through to default */
  }
  cachedLocale = "en";
  return "en";
}

/** Reset the cached locale. Tests use this; production never needs it. */
export function resetLocaleCache(): void {
  cachedLocale = null;
}

/** Falls back to English so partial dictionary updates never show "[missing]". */
export function t(key: keyof typeof EN, vars: Record<string, string | number> = {}): string {
  const loc = detectLocale();
  const dict = loc === "zh" ? ZH : EN;
  const tpl = dict[key] ?? EN[key];
  return tpl.replace(/\{(\w+)\}/g, (_m, name) => {
    const v = vars[name];
    return v === undefined ? `{${name}}` : String(v);
  });
}

const EN = {
  // ── preflight ─────────────────────────────────────────────────────
  ollamaNotFound:
    "✗ `ollama` not found on PATH.\n  Install from https://ollama.com (one-time, ~150 MB), then retry.\n",
  daemonNotReachableHint:
    "✗ Ollama daemon not reachable. Run `ollama serve` and retry, or pass --yes to start it automatically.\n",
  daemonStartConfirm: "Ollama daemon isn't running. Start `ollama serve` now?",
  daemonAbortStart: "✗ aborted — start `ollama serve` yourself and retry.\n",
  daemonStarting: "▸ starting `ollama serve`…\n",
  daemonStartTimeout:
    "✗ daemon didn't come up within 15s. Try `ollama serve` in a separate terminal and retry.\n",
  daemonReady: "✓ daemon up{pid}\n",
  modelNotPulledHint:
    '✗ embedding model "{model}" not pulled. Run `ollama pull {model}` and retry, or pass --yes to pull it automatically.\n',
  modelPullConfirm:
    'Embedding model "{model}" isn\'t pulled yet. Pull it now? (~274 MB for nomic-embed-text)',
  modelAbortPull: "✗ aborted — pull the model yourself and retry.\n",
  modelPulling: "▸ pulling {model}…\n",
  modelPullFailed: "✗ `ollama pull {model}` failed (exit {code}).\n",
  modelPulled: "✓ {model} pulled\n",

  // ── progress ─────────────────────────────────────────────────────
  // The TTY-mode progress writer paints `<spinner> <status>  <elapsed>s`
  // every 120ms. The status itself comes from one of these keys based
  // on the current phase. {files}, {done}, {total}, {pct} are
  // substituted by the writer.
  progressStarting: "starting…",
  progressScan: "scanning project · {files} files",
  progressEmbed: "embedding {done}/{total} chunks · {pct}%",
  progressEmbedHeartbeat: "  {done}/{total}\n",
  progressScanLine: "scanning files…\n",
  progressEmbedLine: "embedding {total} chunks across {files} files…\n",
  // Final result line after a successful build.
  indexSuccess:
    "✓ indexed {scanned} files ({changed} changed, {added} new chunks, {removed} stale removed) in {seconds}s\n",
  indexSuccessWithSkips:
    "✓ indexed {scanned} files ({changed} changed, {added} new chunks, {removed} stale removed, {skipped} skipped due to embed errors) in {seconds}s\n",
  indexNothingToDo: "  (nothing to do — re-run with --rebuild to force a full rebuild)\n",
  indexFailed: "✗ index failed: {msg}\n",

  // ── /semantic slash ──────────────────────────────────────────────
  slashHeader: "semantic_search status",
  slashEnabled: "✓ enabled — index built, tool registered.",
  slashEnabledDetail: "  index size: {chunks} chunks across {files} files",
  slashEnabledHowto: "  the model will call semantic_search automatically when it fits.",
  slashIndexMissing: "✗ no index built yet for this project.",
  slashHowToBuild: "  to enable, exit Reasonix and run in your shell:\n      reasonix index",
  slashOllamaMissing: "  prerequisite: install Ollama from https://ollama.com",
  slashDaemonDown:
    "  Ollama is installed but the daemon isn't running. start it with: ollama serve",
  slashIndexInfo:
    "  what semantic_search does: cross-language code understanding via local embeddings.\n  better than grep when you describe WHAT something does, not WHICH token to find.",
} as const;

const ZH: Partial<Record<keyof typeof EN, string>> = {
  ollamaNotFound:
    "✗ 未找到 `ollama`。\n  请访问 https://ollama.com 安装（一次性，约 150 MB），然后重试。\n",
  daemonNotReachableHint:
    "✗ Ollama 守护进程未启动。请运行 `ollama serve` 后重试，或加 --yes 让我自动启动。\n",
  daemonStartConfirm: "Ollama 守护进程未运行。现在启动 `ollama serve` 吗？",
  daemonAbortStart: "✗ 已取消——请自行运行 `ollama serve` 后重试。\n",
  daemonStarting: "▸ 正在启动 `ollama serve`…\n",
  daemonStartTimeout: "✗ 15 秒内守护进程未就绪。请在另一个终端运行 `ollama serve` 后重试。\n",
  daemonReady: "✓ 守护进程已启动{pid}\n",
  modelNotPulledHint:
    '✗ 嵌入模型 "{model}" 未下载。请运行 `ollama pull {model}` 后重试，或加 --yes 让我自动下载。\n',
  modelPullConfirm: '嵌入模型 "{model}" 还未下载。现在下载吗？（nomic-embed-text 约 274 MB）',
  modelAbortPull: "✗ 已取消——请自行下载模型后重试。\n",
  modelPulling: "▸ 正在下载 {model}…\n",
  modelPullFailed: "✗ `ollama pull {model}` 失败（退出码 {code}）。\n",
  modelPulled: "✓ {model} 下载完成\n",

  progressStarting: "正在启动…",
  progressScan: "扫描项目 · 已扫描 {files} 个文件",
  progressEmbed: "正在向量化 {done}/{total} 个片段 · {pct}%",
  progressEmbedHeartbeat: "  {done}/{total}\n",
  progressScanLine: "正在扫描文件…\n",
  progressEmbedLine: "正在向量化 {total} 个片段（涉及 {files} 个文件）…\n",
  indexSuccess:
    "✓ 已建立索引：扫描 {scanned} 个文件（{changed} 个有变化，新增 {added} 个片段，移除 {removed} 个过期）；耗时 {seconds}s\n",
  indexSuccessWithSkips:
    "✓ 已建立索引：扫描 {scanned} 个文件（{changed} 个有变化，新增 {added} 个片段，移除 {removed} 个过期，跳过 {skipped} 个嵌入失败的片段）；耗时 {seconds}s\n",
  indexNothingToDo: "  （没有变化——加 --rebuild 强制重建）\n",
  indexFailed: "✗ 建立索引失败：{msg}\n",

  slashHeader: "semantic_search 状态",
  slashEnabled: "✓ 已启用——索引已建好，工具已注册。",
  slashEnabledDetail: "  索引规模：{chunks} 个片段，{files} 个文件",
  slashEnabledHowto: "  模型在合适的时候会自动调用 semantic_search。",
  slashIndexMissing: "✗ 当前项目还没有索引。",
  slashHowToBuild: "  启用方式：退出 Reasonix，在终端运行：\n      reasonix index",
  slashOllamaMissing: "  前置依赖：从 https://ollama.com 安装 Ollama",
  slashDaemonDown: "  已装 Ollama 但守护进程未启动，请运行：ollama serve",
  slashIndexInfo:
    '  semantic_search 用本地 embedding 做跨语言代码理解。\n  当你描述"做什么"而不是具体 token 时，比 grep 更好。',
};
