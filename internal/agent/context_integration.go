package agent

import (
	"context"
	"fmt"

	ctxpkg "mimo-reasonix/internal/context"
	"mimo-reasonix/internal/event"
	"mimo-reasonix/internal/provider"
)

// ctxManagerField is the optional context manager that provides an alternative
// code path for token counting, prefix caching, and fold decisions. When set,
// the agent uses it alongside the existing compaction logic; when nil, behavior
// is unchanged. This field is intentionally separate from the existing
// contextWindow/compactRatio fields to allow gradual migration.
type ctxManagerField struct {
	mgr *ctxpkg.Manager
}

// SetContextManager installs a ctxpkg.Manager on the agent. When installed,
// the manager's prefix cache and token counter are used for diagnostics and
// fold decisions; the existing compact.go logic still handles the actual
// message rewriting. Nil disables the manager.
func (a *Agent) SetContextManager(mgr *ctxpkg.Manager) {
	a.ctxMgr.mgr = mgr
}

// ContextManager returns the installed ctxpkg.Manager, or nil if none.
func (a *Agent) ContextManager() *ctxpkg.Manager {
	return a.ctxMgr.mgr
}

// prepareContextManager calibrates the context manager's token counter from
// the last usage and validates the prefix cache. Called before each provider
// API call. Returns diagnostics for the usage event.
func (a *Agent) prepareContextManager(
	messages []provider.Message,
	tools []provider.ToolSchema,
	lastUsage *provider.Usage,
) string {
	mgr := a.ctxMgr.mgr
	if mgr == nil {
		return ""
	}

	prefixHit, diag := mgr.PrepareForProviderCall(messages, tools, lastUsage)

	// Emit a notice when the prefix cache misses (informational, not actionable).
	if !prefixHit && a.haveLastPrefixShape {
		a.sink.Emit(event.Event{
			Kind:  event.Notice,
			Level: event.LevelInfo,
			Text:  fmt.Sprintf("context manager: prefix cache miss (%s)", diag),
		})
	}

	return diag
}

// evaluateContextManager evaluates the context manager's fold decision and
// returns it for logging. The actual compaction is still handled by maybeCompact
// in compact.go; this provides supplementary diagnostics.
func (a *Agent) evaluateContextManager(lastUsage *provider.Usage) string {
	mgr := a.ctxMgr.mgr
	if mgr == nil || lastUsage == nil || lastUsage.PromptTokens == 0 {
		return ""
	}

	decision, softNotice, usage := mgr.Evaluate(lastUsage.PromptTokens)
	diag := fmt.Sprintf("decision=%d usage=%.2f%% soft_notice=%v", decision, usage*100, softNotice)

	if softNotice && !a.softCompactNoticed {
		a.softCompactNoticed = true
		a.sink.Emit(event.Event{
			Kind:  event.Notice,
			Level: event.LevelInfo,
			Text: fmt.Sprintf("context manager: context reached %.0f%% of window; keeping cache-first prefix until compact threshold",
				usage*100),
		})
	}

	return diag
}

// compactViaContextManager attempts compaction through the context manager.
// Returns true if compaction was performed, false if the manager declined
// (e.g., economics check failed). The existing compact.go path is still
// called as a fallback.
func (a *Agent) compactViaContextManager(
	ctx context.Context,
	usage *provider.Usage,
) bool {
	mgr := a.ctxMgr.mgr
	if mgr == nil || usage == nil || usage.PromptTokens == 0 {
		return false
	}

	if a.contextWindow <= 0 {
		return false
	}

	msgs := a.session.Messages
	schemas := a.toolSchemas()

	compacted, didCompact, err := mgr.CompactIfNeeded(ctx, msgs, schemas, usage)
	if err != nil {
		a.sink.Emit(event.Event{
			Kind:  event.Notice,
			Level: event.LevelWarn,
			Text:  fmt.Sprintf("context manager compact failed: %v", err),
		})
		return false
	}

	if didCompact {
		a.session.Replace(compacted)
		a.session.IncrementRewrite()
		a.sink.Emit(event.Event{
			Kind: event.CompactionDone,
			Compaction: event.Compaction{
				Trigger: "auto",
			},
		})
		return true
	}

	return false
}

// toolSchemas extracts the current tool schemas from the agent's registry.
func (a *Agent) toolSchemas() []provider.ToolSchema {
	if a.tools == nil {
		return nil
	}
	return a.tools.Schemas()
}
