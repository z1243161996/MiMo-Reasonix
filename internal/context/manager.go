package context

import (
	"context"
	"fmt"
	"strings"

	"mimo-reasonix/internal/provider"
)

// Manager coordinates token counting, history folding, and prefix caching
// for a single agent session. It provides a unified interface for the agent
// loop to evaluate context state and trigger compaction when needed.
type Manager struct {
	counter *TokenCounter
	folder  *Folder
	prefix  *PrefixCache

	// State tracking for soft-compact notice.
	softCompactNoticed bool
}

// ManagerConfig holds the full configuration for a context manager.
type ManagerConfig struct {
	ContextWindow int
	Folder        FolderConfig
}

// NewManager creates a context manager with the given configuration.
func NewManager(cfg ManagerConfig) *Manager {
	counter := NewTokenCounter(cfg.ContextWindow)
	folder := NewFolder(counter, cfg.Folder)
	prefix := NewPrefixCache()

	return &Manager{
		counter: counter,
		folder:  folder,
		prefix:  prefix,
	}
}

// Counter returns the token counter for calibration and estimation.
func (m *Manager) Counter() *TokenCounter {
	return m.counter
}

// Folder returns the history folder for direct fold operations.
func (m *Manager) Folder() *Folder {
	return m.folder
}

// Prefix returns the prefix cache for cache hit/miss tracking.
func (m *Manager) Prefix() *PrefixCache {
	return m.prefix
}

// Evaluate examines the current token usage and returns the appropriate action.
// It returns:
//   - decision: what fold action to take (none, snip, compact, force)
//   - softNotice: whether to emit a soft-compact notice
//   - usageFraction: the fraction of context window used
func (m *Manager) Evaluate(promptTokens int) (decision FoldDecision, softNotice bool, usageFraction float64) {
	usageFraction = m.counter.ContextWindowUsage(promptTokens)
	decision, softNotice = m.folder.ShouldFold(promptTokens)
	return
}

// ValidatePrefix checks whether the cacheable prefix (system prompt + tools)
// has changed since the last check. Returns true on cache hit, false on miss.
func (m *Manager) ValidatePrefix(messages []provider.Message, tools []provider.ToolSchema) bool {
	return m.prefix.CheckPrefix(messages, tools)
}

// PrepareForProviderCall should be called before each provider API call.
// It calibrates the token counter from the last usage and validates the
// prefix cache. Returns diagnostics for logging.
func (m *Manager) PrepareForProviderCall(
	messages []provider.Message,
	tools []provider.ToolSchema,
	lastUsage *provider.Usage,
) (prefixHit bool, diagnostics string) {
	// Calibrate from real usage if available.
	if lastUsage != nil && lastUsage.PromptTokens > 0 {
		charCount := CharsOfMessages(messages)
		if charCount > 0 {
			m.counter.Calibrate(lastUsage.PromptTokens, charCount)
		}
	}

	// Validate prefix cache.
	prefixHit = m.ValidatePrefix(messages, tools)

	hits, misses, ratio := m.prefix.Stats()
	diagnostics = fmt.Sprintf("prefix=%v cache_hits=%d cache_misses=%d ratio=%.2f tok_per_char=%.4f",
		boolToHit(prefixHit), hits, misses, ratio, m.counter.TokPerChar())

	return prefixHit, diagnostics
}

// CompactIfNeeded evaluates the context and performs compaction if necessary.
// It returns the post-compaction messages (or the original if no compaction
// was needed), whether compaction happened, and any error.
func (m *Manager) CompactIfNeeded(
	ctx context.Context,
	messages []provider.Message,
	tools []provider.ToolSchema,
	lastUsage *provider.Usage,
) (result []provider.Message, didCompact bool, err error) {
	if m.counter.contextWindow <= 0 || lastUsage == nil || lastUsage.PromptTokens == 0 {
		return messages, false, nil
	}

	promptTokens := lastUsage.PromptTokens
	decision, softNotice, _ := m.Evaluate(promptTokens)

	// Emit soft notice if applicable.
	_ = softNotice // caller handles notice emission via event sink

	switch decision {
	case FoldNone:
		m.softCompactNoticed = false
		return messages, false, nil

	case FoldSnip:
		// Snipping is handled by the agent directly (PruneStaleToolResults).
		// The context manager just signals the decision.
		return messages, false, nil

	case FoldCompact, FoldForceCompact:
		// Plan the fold.
		region, ok := m.folder.PlanFold(messages)
		if !ok {
			return messages, false, nil
		}

		force := decision == FoldForceCompact
		if !force && !FoldEconomics(region.Fold) {
			return messages, false, nil
		}

		// Try LLM summarization.
		summary, sumErr := m.folder.SummarizeWithRetry(ctx, region.Fold, "")
		if sumErr != nil {
			// Mechanical fold fallback.
			summary = MechanicalFoldDigest(len(region.Fold), "")
		}

		// Build compacted message list.
		compacted := make([]provider.Message, 0, region.Head+len(region.Kept)+1+len(messages)-region.Start)
		compacted = append(compacted, messages[:region.Head]...)
		compacted = append(compacted, BuildCompacted(region, summary)...)
		compacted = append(compacted, messages[region.Start:]...)

		return compacted, true, nil
	}

	return messages, false, nil
}

// ResetState clears the soft-compact notice flag and prefix cache state.
// Call this when starting a new conversation turn or after a manual compact.
func (m *Manager) ResetState() {
	m.softCompactNoticed = false
	m.prefix.Reset()
}

// ContextWindow returns the configured context window size in tokens.
func (m *Manager) ContextWindow() int {
	return m.counter.contextWindow
}

// SetContextWindow updates the context window size (e.g., after provider switch).
func (m *Manager) SetContextWindow(size int) {
	m.counter.contextWindow = size
}

func boolToHit(b bool) string {
	if b {
		return "hit"
	}
	return "miss"
}

// SummarySystemPrompt is the prompt used to steer the summarizer to distill
// older history into a structured briefing.
const SummarySystemPrompt = `You are compacting the earlier part of a coding agent's conversation to save context.
The agent keeps your summary alongside the user's own turns (kept verbatim) and the recent tail; your job is to fold the assistant/tool work into a briefing it can resume from.
Write under these exact headings, omitting a heading only if it has no content:

## Standing facts & constraints
Everything the user stated that still governs the work — names, paths, IDs, versions, tokens, preferences, and hard "never do X" rules — in their own words. Be exhaustive; this is the durable contract, so prefer over- to under-including.

## Goal
The user's request and intent.

## Decisions & rationale
Key choices made so far and why — so they are not re-litigated or reversed.

## Files & code
Files read or modified, with the specific facts that matter: signatures, line locations, data shapes, and exact edits applied. Be concrete; this is what lets the agent act without re-reading everything.

## Commands & outcomes
Commands run (builds, tests, git) and their relevant results — what passed, what failed, and the error text that matters.

## Errors & fixes
Problems hit and how they were resolved (or not), so the same dead ends are not repeated.

## Pending & next step
What is still in progress or unstarted, and the single most concrete next action to take.

Rules: be terse — bullet points and fragments, not prose. Preserve identifiers, paths, and numbers exactly. Do NOT invent anything not present in the messages; if something is unknown, leave it out rather than guessing.`

// RenderTranscriptForSummarization builds the summary prompt messages for the
// LLM summarizer call.
func RenderTranscriptForSummarization(region []provider.Message, instructions string) []provider.Message {
	sys := SummarySystemPrompt
	if strings.TrimSpace(instructions) != "" {
		sys += "\n\nAdditional focus for this compaction (prioritize keeping this):\n" + strings.TrimSpace(instructions)
	}
	return []provider.Message{
		{Role: provider.RoleSystem, Content: sys},
		{Role: provider.RoleUser, Content: RenderTranscript(region)},
	}
}
