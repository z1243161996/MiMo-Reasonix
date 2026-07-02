// Package context provides token counting, history folding, and prefix caching
// for managing LLM context windows.
package context

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"mimo-reasonix/internal/provider"
)

const (
	// fallbackTokPerChar is the default tokens-per-character ratio before any
	// real usage data is available (~4 chars/token for English text).
	fallbackTokPerChar = 0.25

	// minTokPerChar and maxTokPerChar bound the calibrated ratio so absurd
	// values (all-CJK, code-heavy) don't produce wild estimates.
	minTokPerChar = 0.05
	maxTokPerChar = 2.0
)

// TokenCounter estimates token usage for messages and text. It tracks a
// calibrated tokens-per-character ratio derived from real provider usage
// and falls back to a conservative heuristic when no calibration data exists.
type TokenCounter struct {
	tokPerChar    float64
	calibrated    bool
	contextWindow int
}

// NewTokenCounter creates a counter with the given context window size.
// Before calibration, it uses the fallback ratio of ~0.25 tokens/char.
func NewTokenCounter(contextWindow int) *TokenCounter {
	return &TokenCounter{
		tokPerChar:    fallbackTokPerChar,
		contextWindow: contextWindow,
	}
}

// Calibrate updates the tokens-per-character ratio from a real provider usage
// report and the actual character count of the messages that were sent. This
// makes subsequent estimates track the provider's tokenizer accurately.
func (tc *TokenCounter) Calibrate(promptTokens int, charCount int) {
	if promptTokens <= 0 || charCount <= 0 {
		return
	}
	r := float64(promptTokens) / float64(charCount)
	if r >= minTokPerChar && r <= maxTokPerChar {
		tc.tokPerChar = r
		tc.calibrated = true
	}
}

// TokPerChar returns the current tokens-per-character ratio.
func (tc *TokenCounter) TokPerChar() float64 {
	return tc.tokPerChar
}

// IsCalibrated reports whether Calibrate has been called with valid data.
func (tc *TokenCounter) IsCalibrated() bool {
	return tc.calibrated
}

// EstimateTextTokens estimates the token count for a single text string.
// It uses a cross-language approximation: English-ish text trends near 4 bytes
// per token, while CJK-heavy text is closer to one rune per token.
func EstimateTextTokens(s string) int {
	if s == "" {
		return 0
	}
	bytes := len(s)
	runes := utf8.RuneCountInString(s)
	byBytes := (bytes + 3) / 4
	if runes > byBytes {
		return runes
	}
	return byBytes
}

// EstimateMessageTokens estimates the token count for a single message,
// including content, tool calls, and framing overhead.
func EstimateMessageTokens(m provider.Message) int {
	total := 4 // chat-message framing overhead
	total += EstimateTextTokens(m.Content)
	total += EstimateTextTokens(m.ReasoningContent)
	total += EstimateTextTokens(m.Name)
	total += EstimateTextTokens(m.ToolCallID)
	for _, tc := range m.ToolCalls {
		total += 8
		total += EstimateTextTokens(tc.ID)
		total += EstimateTextTokens(tc.Name)
		total += EstimateTextTokens(tc.Arguments)
	}
	return total
}

// EstimateMessagesTokens estimates the total token count for a slice of messages.
func EstimateMessagesTokens(msgs []provider.Message) int {
	total := 0
	for _, m := range msgs {
		total += EstimateMessageTokens(m)
	}
	return total
}

// EstimateMessagesTokensCalibrated estimates tokens using the calibrated
// tokens-per-character ratio, which tracks the provider's actual tokenizer
// more closely than the static heuristic.
func (tc *TokenCounter) EstimateMessagesTokensCalibrated(msgs []provider.Message) int {
	total := 0
	for _, m := range msgs {
		total += 4 // framing
		total += int(float64(CharCount(m)) * tc.tokPerChar)
	}
	return total
}

// CharCount counts the characters that ride to the provider for one message —
// content plus tool-call names and arguments, but not reasoning (stripped on send).
func CharCount(m provider.Message) int {
	n := len(m.Content)
	for _, tc := range m.ToolCalls {
		n += len(tc.Name) + len(tc.Arguments)
	}
	return n
}

// CharsOfMessages counts total characters across all messages.
func CharsOfMessages(msgs []provider.Message) int {
	n := 0
	for _, m := range msgs {
		n += CharCount(m)
	}
	return n
}

// ContextWindowUsage returns the fraction of the context window currently used
// by the given prompt token count. Returns 0 when the window is unconfigured.
func (tc *TokenCounter) ContextWindowUsage(promptTokens int) float64 {
	if tc.contextWindow <= 0 {
		return 0
	}
	return float64(promptTokens) / float64(tc.contextWindow)
}

// RemainingTokens returns the number of tokens remaining before the given
// threshold ratio of the context window. Returns 0 when the window is unconfigured.
func (tc *TokenCounter) RemainingTokens(promptTokens int, threshold float64) int {
	if tc.contextWindow <= 0 {
		return 0
	}
	limit := int(float64(tc.contextWindow) * threshold)
	remaining := limit - promptTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// FoldThresholdTokens returns the token count at which folding should trigger.
func (tc *TokenCounter) FoldThresholdTokens(ratio float64) int {
	if tc.contextWindow <= 0 {
		return 0
	}
	return int(float64(tc.contextWindow) * ratio)
}

// RenderTranscript flattens messages into a readable transcript for summarization.
func RenderTranscript(msgs []provider.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case provider.RoleUser:
			fmt.Fprintf(&b, "[user]\n%s\n\n", m.Content)
		case provider.RoleAssistant:
			if m.Content != "" {
				fmt.Fprintf(&b, "[assistant]\n%s\n", m.Content)
			}
			for _, tc := range m.ToolCalls {
				fmt.Fprintf(&b, "[assistant calls %s] %s\n", tc.Name, summarizeToolArgs(tc.Arguments))
			}
			b.WriteString("\n")
		case provider.RoleTool:
			fmt.Fprintf(&b, "[tool %s result]\n%s\n\n", m.Name, m.Content)
		case provider.RoleSystem:
			fmt.Fprintf(&b, "[system]\n%s\n\n", m.Content)
		}
	}
	return b.String()
}

func summarizeToolArgs(args string) string {
	if args == "" {
		return "(no arguments)"
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return fmt.Sprintf("(%d bytes)", len(args))
	}
	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return fmt.Sprintf("{%s} (%d keys)", strings.Join(keys, ", "), len(parsed))
}
