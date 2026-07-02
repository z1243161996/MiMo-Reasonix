package context

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mimo-reasonix/internal/provider"
)

const (
	// DefaultSnipRatio is the threshold at which stale tool results are snipped
	// before full compaction.
	DefaultSnipRatio = 0.6

	// DefaultCompactRatio is the threshold at which compaction is triggered.
	DefaultCompactRatio = 0.8

	// DefaultForceCompactRatio is the high-water mark that forces compaction
	// even for low-value folds.
	DefaultForceCompactRatio = 0.9

	// DefaultCompactTarget is the safety cap: the kept tail never exceeds this
	// fraction of the context window.
	DefaultCompactTarget = 0.5

	// DefaultTailTokens is the verbatim recent-tail budget in tokens.
	DefaultTailTokens = 16384

	// MinRecentKeep is the minimum number of recent messages to preserve.
	MinRecentKeep = 2

	// MinCompactMessages is the minimum compactable region size.
	MinCompactMessages = 2

	// MinFoldTokens is the minimum token count for a fold to be worthwhile.
	MinFoldTokens = 400

	// MaxPinnedFirstUserTokens is the ceiling on pinning the first user turn
	// verbatim; larger first turns stay foldable.
	MaxPinnedFirstUserTokens = 1500

	// PinnedFirstUserWindowFrac is the maximum fraction of the window that a
	// pinned first user turn may occupy.
	PinnedFirstUserWindowFrac = 0.15

	// SummaryTimeout bounds one summarizer call.
	SummaryTimeout = 90 * time.Second
)

// Summary tags wrap the compaction summary so the model can distinguish it
// from live user input.
const (
	SummaryTagOpen  = "<compaction-summary>"
	SummaryTagClose = "</compaction-summary>"
)

// Summarizer is the interface the folder uses to call the LLM for summarization.
// The agent implements this by wrapping its provider.Stream call.
type Summarizer interface {
	Summarize(ctx context.Context, messages []provider.Message) (string, error)
}

// Folder manages history folding: splitting messages into head/tail, calling
// the LLM to summarize the head, and preserving pinned constraints.
type Folder struct {
	counter       *TokenCounter
	snipRatio     float64
	compactRatio  float64
	forceRatio    float64
	compactTarget float64
	tailTokens    int
	recentKeep    int
	summarizer    Summarizer
	archiveDir    string
}

// FolderConfig holds configuration for the history folder.
type FolderConfig struct {
	SnipRatio     float64 // 0.6 default
	CompactRatio  float64 // 0.8 default
	ForceRatio    float64 // 0.9 default
	CompactTarget float64 // 0.5 default
	TailTokens    int     // 16384 default
	RecentKeep    int     // 2 default
	Summarizer    Summarizer
	ArchiveDir    string
}

// NewFolder creates a folder with the given configuration and token counter.
func NewFolder(counter *TokenCounter, cfg FolderConfig) *Folder {
	if cfg.SnipRatio <= 0 {
		cfg.SnipRatio = DefaultSnipRatio
	}
	if cfg.CompactRatio <= 0 {
		cfg.CompactRatio = DefaultCompactRatio
	}
	if cfg.ForceRatio <= 0 {
		cfg.ForceRatio = DefaultForceCompactRatio
	}
	if cfg.CompactTarget <= 0 {
		cfg.CompactTarget = DefaultCompactTarget
	}
	if cfg.TailTokens <= 0 {
		cfg.TailTokens = DefaultTailTokens
	}
	if cfg.RecentKeep < MinRecentKeep {
		cfg.RecentKeep = MinRecentKeep
	}
	return &Folder{
		counter:       counter,
		snipRatio:     cfg.SnipRatio,
		compactRatio:  cfg.CompactRatio,
		forceRatio:    cfg.ForceRatio,
		compactTarget: cfg.CompactTarget,
		tailTokens:    cfg.TailTokens,
		recentKeep:    cfg.RecentKeep,
		summarizer:    cfg.Summarizer,
		archiveDir:    cfg.ArchiveDir,
	}
}

// FoldDecision describes what action the context manager should take based
// on the current token usage.
type FoldDecision int

const (
	// FoldNone means no action is needed.
	FoldNone FoldDecision = iota
	// FoldSnip means stale tool results should be snipped.
	FoldSnip
	// FoldCompact means a full compaction should run.
	FoldCompact
	// FoldForceCompact means compaction should run even for low-value folds.
	FoldForceCompact
)

// ShouldFold evaluates the current token usage and returns the appropriate
// fold decision and whether the soft-compact notice should fire.
func (f *Folder) ShouldFold(promptTokens int) (decision FoldDecision, softNotice bool) {
	if f.counter.contextWindow <= 0 {
		return FoldNone, false
	}

	snip := int(float64(f.counter.contextWindow) * f.snipRatio)
	compact := int(float64(f.counter.contextWindow) * f.compactRatio)
	force := int(float64(f.counter.contextWindow) * f.forceRatio)
	soft := int(float64(f.counter.contextWindow) * DefaultSoftCompactRatio)

	if promptTokens >= soft && promptTokens < snip {
		return FoldNone, true // soft notice
	}
	if promptTokens >= snip && promptTokens < compact {
		return FoldSnip, false
	}
	if promptTokens >= compact && promptTokens < force {
		return FoldCompact, false
	}
	if promptTokens >= force {
		return FoldForceCompact, false
	}
	return FoldNone, false
}

const (
	// DefaultSoftCompactRatio is the threshold for reporting growing context
	// without compaction (cache-stable prefix kept intact).
	DefaultSoftCompactRatio = 0.5
)

// FoldRegion describes the split of messages for compaction.
type FoldRegion struct {
	// Head is the count of leading messages preserved verbatim (system prompt,
	// pinned first user turn, prior summaries).
	Head int
	// Start is where the preserved recent tail begins. Messages in [Head, Start)
	// are compacted.
	Start int
	// Kept are the messages within the compactable region that are preserved
	// verbatim (small user turns, pinned constraints, prior summaries).
	Kept []provider.Message
	// Fold are the messages that should be summarized.
	Fold []provider.Message
	// Tail are the recent messages preserved verbatim.
	Tail []provider.Message
}

// PlanFold locates the compaction region and partitions messages into
// kept/fold/tail groups. Returns false when there is too little to compact.
func (f *Folder) PlanFold(messages []provider.Message) (FoldRegion, bool) {
	if len(messages) == 0 {
		return FoldRegion{}, false
	}

	head := pinnedPrefixLen(messages, f.counter)
	if f.counter.contextWindow > 0 {
		budget := f.tailTokens
		if maxByWin := int(float64(f.counter.contextWindow) * f.compactTarget); maxByWin < budget {
			budget = maxByWin
		}
		start := tailStart(messages, head, budget, f.counter.TokPerChar(), f.recentKeep)
		if start < head {
			start = head
		}
		region := messages[head:start]
		if len(region) < MinCompactMessages {
			return FoldRegion{}, false
		}

		kept, fold := partitionFold(region, f.counter)
		if len(fold) == 0 {
			return FoldRegion{}, false
		}

		return FoldRegion{
			Head:  head,
			Start: start,
			Kept:  kept,
			Fold:  fold,
			Tail:  messages[start:],
		}, true
	}

	// No window — use fixed message count.
	start := len(messages) - f.recentKeep
	if start < 0 {
		start = 0
	}
	for start > head && messages[start].Role == provider.RoleTool {
		start--
	}
	region := messages[head:start]
	if len(region) < MinCompactMessages {
		return FoldRegion{}, false
	}
	kept, fold := partitionFold(region, f.counter)
	if len(fold) == 0 {
		return FoldRegion{}, false
	}
	return FoldRegion{
		Head:  head,
		Start: start,
		Kept:  kept,
		Fold:  fold,
		Tail:  messages[start:],
	}, true
}

// FoldEconomics estimates whether compacting the given region saves enough
// tokens to justify the summarization API call.
func FoldEconomics(region []provider.Message) bool {
	return EstimateMessagesTokens(region) >= MinFoldTokens
}

// Summarize calls the LLM to produce a summary of the foldable messages.
// Returns the summary text or an error.
func (f *Folder) Summarize(ctx context.Context, fold []provider.Message, instructions string) (string, error) {
	if f.summarizer == nil {
		return "", errors.New("no summarizer configured")
	}
	ctx, cancel := context.WithTimeout(ctx, SummaryTimeout)
	defer cancel()
	return f.summarizer.Summarize(ctx, fold)
}

// SummarizeWithRetry retries one non-timeout failure; a timeout or second
// failure returns the error so the caller falls back to mechanical folding.
func (f *Folder) SummarizeWithRetry(ctx context.Context, fold []provider.Message, instructions string) (string, error) {
	summary, err := f.Summarize(ctx, fold, instructions)
	if err == nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return summary, err
	}
	return f.Summarize(ctx, fold, instructions)
}

// BuildCompacted constructs the post-compaction message list by combining the
// head, kept messages, the summary, and the tail.
func BuildCompacted(region FoldRegion, summary string) []provider.Message {
	// The summary wraps in summary tags so the model can distinguish it.
	summaryContent := SummaryTagOpen + "\n" +
		"Summary of earlier conversation (older messages were compacted to save context):\n" +
		summary + "\n" +
		SummaryTagClose

	result := make([]provider.Message, 0, region.Head+len(region.Kept)+1+len(region.Tail))
	// We don't have access to the full messages here, so the caller must
	// prepend messages[:region.Head] and append region.Tail.
	result = append(result, region.Kept...)
	result = append(result, provider.Message{
		Role:    provider.RoleUser,
		Content: summaryContent,
	})
	result = append(result, region.Tail...)
	return result
}

// MechanicalFoldDigest is the deterministic stand-in used when the summarizer
// is unreachable.
func MechanicalFoldDigest(n int, archive string) string {
	where := "."
	if archive != "" {
		where = " (archived to " + archive + ")."
	}
	return fmt.Sprintf("%d earlier message(s) were folded here to free context, but the automatic summary was unavailable%s Ask the user if you need details from before this point.", n, where)
}

// --- internal helpers ---

// pinnedPrefixLen counts the leading messages a fold keeps verbatim: the system
// prompt, the first user turn (when small enough), and any prior summaries.
func pinnedPrefixLen(messages []provider.Message, counter *TokenCounter) int {
	i := 0
	if i < len(messages) && messages[i].Role == provider.RoleSystem {
		i++
	}
	if i < len(messages) && messages[i].Role == provider.RoleUser && !isCompactionSummary(messages[i]) && pinnableUserTurn(messages[i], counter) {
		i++
	}
	for i < len(messages) && isCompactionSummary(messages[i]) {
		i++
	}
	return i
}

// pinnableUserTurn reports whether a user turn is small enough to keep verbatim.
func pinnableUserTurn(m provider.Message, counter *TokenCounter) bool {
	budget := MaxPinnedFirstUserTokens
	if counter.contextWindow > 0 {
		if f := int(float64(counter.contextWindow) * PinnedFirstUserWindowFrac); f < budget {
			budget = f
		}
	}
	return int(float64(CharCount(m))*counter.TokPerChar()) <= budget
}

// isCompactionSummary reports whether m is a rolling summary from a prior fold.
func isCompactionSummary(m provider.Message) bool {
	return m.Role == provider.RoleUser &&
		strings.HasPrefix(strings.TrimLeft(m.Content, "\n "), SummaryTagOpen)
}

// partitionFold splits a compaction region into kept and fold groups.
// Small user turns and prior summaries are kept; everything else folds.
func partitionFold(region []provider.Message, counter *TokenCounter) (kept, fold []provider.Message) {
	for _, m := range region {
		if isCompactionSummary(m) || (m.Role == provider.RoleUser && pinnableUserTurn(m, counter)) {
			kept = append(kept, m)
		} else {
			fold = append(fold, m)
		}
	}
	return kept, fold
}

// tailStart walks newest-to-oldest, growing the verbatim tail until the next
// message would push its token estimate past budgetTokens (but never below
// minKeep messages).
func tailStart(messages []provider.Message, head, budgetTokens int, tokPerChar float64, minKeep int) int {
	start := len(messages)
	acc := 0
	for i := len(messages) - 1; i > head; i-- {
		c := int(float64(CharCount(messages[i])) * tokPerChar)
		if len(messages)-i > minKeep && acc+c > budgetTokens {
			break
		}
		acc += c
		start = i
	}
	// Align off any tool result so the tail never begins with an orphan.
	for start > head && start < len(messages) && messages[start].Role == provider.RoleTool {
		start--
	}
	return start
}
