package repair

import (
	"fmt"
	"strings"
)

// DefaultStormBreakThreshold is the default number of identical failures
// before the storm breaker intervenes.
const DefaultStormBreakThreshold = 3

// StormBreaker detects death-spiral loops where the same tool call fails
// identically multiple times in succession and injects a directive to change
// approach. It keys on (tool_name, error) pairs rather than arguments because
// a stuck model often reworks arguments cosmetically while the underlying
// failure remains the same.
type StormBreaker struct {
	// Threshold is how many consecutive identical failures trigger intervention.
	Threshold int

	// CurrentSig is the signature of the current run of failures.
	CurrentSig string
	// Count is the number of consecutive identical failures.
	Count int
}

// NewStormBreaker creates a StormBreaker with the given threshold.
func NewStormBreaker(threshold int) *StormBreaker {
	if threshold <= 0 {
		threshold = DefaultStormBreakThreshold
	}
	return &StormBreaker{
		Threshold: threshold,
	}
}

// ToolOutcome represents the result of a single tool call for storm detection.
type ToolOutcome struct {
	// Name is the tool name.
	Name string
	// Error is the error message (empty on success).
	Error string
	// Blocked indicates the call was blocked by permissions/policy.
	Blocked bool
}

// Check evaluates a batch of tool outcomes and returns whether the storm
// breaker should intervene. It returns:
//   - intervene: true if the loop threshold has been reached
//   - message: the directive to inject into the first tool result
//   - count: the current consecutive failure count
func (sb *StormBreaker) Check(outcomes []ToolOutcome) (bool, string, int) {
	sig, ok := batchSignature(outcomes)
	if !ok {
		// Any success or block means varied progress — reset.
		sb.CurrentSig = ""
		sb.Count = 0
		return false, "", 0
	}

	if sig != sb.CurrentSig {
		sb.CurrentSig = sig
		sb.Count = 1
		return false, "", sb.Count
	}

	sb.Count++
	if sb.Count < sb.Threshold {
		return false, "", sb.Count
	}

	subject := fmt.Sprintf("%q", outcomes[0].Name)
	if len(outcomes) > 1 {
		subject = fmt.Sprintf("this batch of %d tool calls", len(outcomes))
	}

	msg := fmt.Sprintf(
		"\n\n[loop guard] %s has now failed %d times in a row with the same error. "+
			"Re-sending it — even with the wording changed — will not help: the calls keep failing the same way. "+
			"Change approach: if an argument is being truncated, write less in one call and split the work into several smaller calls; "+
			"otherwise fix the arguments, use a different tool, or explain the blocker in your final answer.",
		subject, sb.Count)

	return true, msg, sb.Count
}

// Reset clears the storm breaker state (call when a turn makes varied progress).
func (sb *StormBreaker) Reset() {
	sb.CurrentSig = ""
	sb.Count = 0
}

// batchSignature builds a per-turn fixation signature from tool outcomes.
// The signature is each call's (name, error) joined by null bytes.
// Returns ok=false when any call succeeded or was blocked (varied progress).
func batchSignature(outcomes []ToolOutcome) (string, bool) {
	if len(outcomes) == 0 {
		return "", false
	}

	var sb strings.Builder
	for _, o := range outcomes {
		if o.Error == "" || o.Blocked {
			return "", false
		}
		sb.WriteString(o.Name)
		sb.WriteByte(0)
		sb.WriteString(o.Error)
		sb.WriteByte(0)
	}
	return sb.String(), true
}

// RepeatSuccessBreakThreshold is the default number of identical successful
// write-like calls before the loop guard blocks another copy.
const RepeatSuccessBreakThreshold = 2

// RepeatSuccessTracker tracks write-like tool calls that have already
// succeeded in this user turn. This catches the complementary loop shape:
// a model keeps doing the same successful write, so there is no error for
// the failure-only storm breaker to see.
type RepeatSuccessTracker struct {
	// Threshold is how many identical successes trigger blocking.
	Threshold int
	// counts maps (tool_name, canonical_args) to occurrence count.
	counts map[string]int
}

// NewRepeatSuccessTracker creates a tracker with the default threshold.
func NewRepeatSuccessTracker() *RepeatSuccessTracker {
	return &RepeatSuccessTracker{
		Threshold: RepeatSuccessBreakThreshold,
		counts:    make(map[string]int),
	}
}

// Record records a successful write-like tool call. Returns true if this
// call should be blocked (already reached threshold).
func (rst *RepeatSuccessTracker) Record(name, canonicalArgs string) (blocked bool, message string) {
	if rst.Threshold <= 0 {
		return false, ""
	}

	key := name + "\x00" + canonicalArgs
	rst.counts[key]++

	count := rst.counts[key]
	if count < rst.Threshold {
		return false, ""
	}

	msg := fmt.Sprintf(
		"blocked: [loop guard] %q has already succeeded %d times with the same write-like arguments in this user turn. "+
			"Re-running it is unlikely to help and may burn tokens or repeat file writes. "+
			"Change approach: use edit_file or multi_edit for file changes, verify with a read/test command, "+
			"or explain the blocker in your final answer.",
		name, count)
	return true, msg
}

// Reset clears all tracked counts (call at the start of a new user turn).
func (rst *RepeatSuccessTracker) Reset() {
	rst.counts = make(map[string]int)
}

// Count returns the current count for a (name, args) pair (for diagnostics).
func (rst *RepeatSuccessTracker) Count(name, canonicalArgs string) int {
	key := name + "\x00" + canonicalArgs
	return rst.counts[key]
}
