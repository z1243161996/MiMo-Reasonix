package context

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"mimo-reasonix/internal/provider"
)

// --- Token Counter Tests ---

func TestEstimateTextTokens_Empty(t *testing.T) {
	if got := EstimateTextTokens(""); got != 0 {
		t.Errorf("EstimateTextTokens(\"\") = %d, want 0", got)
	}
}

func TestEstimateTextTokens_English(t *testing.T) {
	// The algorithm returns max(runes, (bytes+3)/4). For pure ASCII, runes == bytes,
	// so it returns the character count. For mixed content with many CJK characters
	// appended (more runes than bytes/4), it returns rune count.
	s := strings.Repeat("abc", 100) // 300 ASCII chars, runes == bytes
	got := EstimateTextTokens(s)
	if got != 300 {
		t.Errorf("EstimateTextTokens returned %d for %d-char ASCII string, want 300", got, len(s))
	}
}

func TestEstimateTextTokens_CJK(t *testing.T) {
	s := "这是一个测试字符串用于令牌估算"
	got := EstimateTextTokens(s)
	runes := 12 // number of CJK characters
	// CJK should be closer to rune count than byte/4
	if got < runes-2 || got > runes+5 {
		t.Errorf("EstimateTextTokens returned %d for CJK text (runes=%d), expected near rune count", got, runes)
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	m := provider.Message{
		Role:    provider.RoleUser,
		Content: "Hello, world!",
	}
	got := EstimateMessageTokens(m)
	// Should be > 4 (framing overhead) + some tokens for content
	if got <= 4 {
		t.Errorf("EstimateMessageTokens = %d, want > 4 (framing overhead)", got)
	}
}

func TestEstimateMessageTokens_WithToolCalls(t *testing.T) {
	m := provider.Message{
		Role:    provider.RoleAssistant,
		Content: "Let me read the file.",
		ToolCalls: []provider.ToolCall{
			{ID: "call_1", Name: "Read", Arguments: `{"file_path":"/tmp/test.txt"}`},
		},
	}
	got := EstimateMessageTokens(m)
	// Should be larger than a plain message due to tool call overhead
	plain := EstimateMessageTokens(provider.Message{Role: provider.RoleAssistant, Content: "Let me read the file."})
	if got <= plain {
		t.Errorf("EstimateMessageTokens with tool calls = %d, plain = %d, want tool calls to add overhead", got, plain)
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a helpful assistant."},
		{Role: provider.RoleUser, Content: "Hello!"},
		{Role: provider.RoleAssistant, Content: "Hi there!"},
	}
	got := EstimateMessagesTokens(msgs)
	if got <= 0 {
		t.Errorf("EstimateMessagesTokens = %d, want > 0", got)
	}
}

func TestNewTokenCounter(t *testing.T) {
	tc := NewTokenCounter(128000)
	if tc.contextWindow != 128000 {
		t.Errorf("contextWindow = %d, want 128000", tc.contextWindow)
	}
	if tc.tokPerChar != fallbackTokPerChar {
		t.Errorf("tokPerChar = %f, want %f", tc.tokPerChar, fallbackTokPerChar)
	}
	if tc.IsCalibrated() {
		t.Error("should not be calibrated before Calibrate()")
	}
}

func TestTokenCounter_Calibrate(t *testing.T) {
	tc := NewTokenCounter(128000)

	// Calibrate with 1000 tokens over 4000 chars = 0.25
	tc.Calibrate(1000, 4000)
	if !tc.IsCalibrated() {
		t.Error("should be calibrated after valid Calibrate()")
	}
	if tc.TokPerChar() != 0.25 {
		t.Errorf("TokPerChar = %f, want 0.25", tc.TokPerChar())
	}
}

func TestTokenCounter_Calibrate_Invalid(t *testing.T) {
	tc := NewTokenCounter(128000)

	// Zero values should not calibrate
	tc.Calibrate(0, 4000)
	if tc.IsCalibrated() {
		t.Error("should not be calibrated with zero promptTokens")
	}

	// Out-of-range ratio should not calibrate
	tc.Calibrate(100, 10) // ratio = 10, way above maxTokPerChar
	if tc.IsCalibrated() {
		t.Error("should not be calibrated with out-of-range ratio")
	}
}

func TestTokenCounter_ContextWindowUsage(t *testing.T) {
	tc := NewTokenCounter(100000)

	got := tc.ContextWindowUsage(50000)
	if got != 0.5 {
		t.Errorf("ContextWindowUsage(50000) = %f, want 0.5", got)
	}

	got = tc.ContextWindowUsage(0)
	if got != 0.0 {
		t.Errorf("ContextWindowUsage(0) = %f, want 0.0", got)
	}
}

func TestTokenCounter_ContextWindowUsage_NoWindow(t *testing.T) {
	tc := NewTokenCounter(0)
	got := tc.ContextWindowUsage(50000)
	if got != 0.0 {
		t.Errorf("ContextWindowUsage with no window = %f, want 0.0", got)
	}
}

func TestTokenCounter_RemainingTokens(t *testing.T) {
	tc := NewTokenCounter(100000)

	got := tc.RemainingTokens(70000, 0.8)
	// 80% of 100000 = 80000, remaining = 80000 - 70000 = 10000
	if got != 10000 {
		t.Errorf("RemainingTokens(70000, 0.8) = %d, want 10000", got)
	}

	// Over limit
	got = tc.RemainingTokens(90000, 0.8)
	if got != 0 {
		t.Errorf("RemainingTokens(90000, 0.8) = %d, want 0 (over limit)", got)
	}
}

func TestTokenCounter_FoldThresholdTokens(t *testing.T) {
	tc := NewTokenCounter(100000)

	got := tc.FoldThresholdTokens(0.8)
	if got != 80000 {
		t.Errorf("FoldThresholdTokens(0.8) = %d, want 80000", got)
	}
}

func TestTokenCounter_EstimateMessagesTokensCalibrated(t *testing.T) {
	tc := NewTokenCounter(128000)
	tc.Calibrate(1000, 4000) // ratio = 0.25

	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello, world!"},
	}
	got := tc.EstimateMessagesTokensCalibrated(msgs)
	if got <= 0 {
		t.Errorf("EstimateMessagesTokensCalibrated = %d, want > 0", got)
	}
}

func TestCharCount(t *testing.T) {
	m := provider.Message{
		Role:    provider.RoleUser,
		Content: "Hello",
		ToolCalls: []provider.ToolCall{
			{ID: "call_1", Name: "Read", Arguments: "{}"},
		},
	}
	got := CharCount(m)
	// Content "Hello" = 5, Name "Read" = 4, Arguments "{}" = 2
	if got != 11 {
		t.Errorf("CharCount = %d, want 11", got)
	}
}

func TestCharsOfMessages(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello"},
		{Role: provider.RoleAssistant, Content: "World"},
	}
	got := CharsOfMessages(msgs)
	if got != 10 {
		t.Errorf("CharsOfMessages = %d, want 10", got)
	}
}

func TestRenderTranscript(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello"},
		{Role: provider.RoleAssistant, Content: "Hi there!"},
	}
	got := RenderTranscript(msgs)
	if !strings.Contains(got, "[user]\nHello\n\n") {
		t.Errorf("transcript missing user message: %s", got)
	}
	if !strings.Contains(got, "[assistant]\nHi there!\n") {
		t.Errorf("transcript missing assistant message: %s", got)
	}
}

func TestRenderTranscript_WithToolCalls(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleAssistant, Content: "", ToolCalls: []provider.ToolCall{
			{ID: "call_1", Name: "Read", Arguments: `{"file_path":"test.go"}`},
		}},
	}
	got := RenderTranscript(msgs)
	if !strings.Contains(got, "[assistant calls Read]") {
		t.Errorf("transcript missing tool call: %s", got)
	}
}

func TestRenderTranscript_ToolResult(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleTool, Name: "Read", Content: "file contents"},
	}
	got := RenderTranscript(msgs)
	if !strings.Contains(got, "[tool Read result]\nfile contents\n\n") {
		t.Errorf("transcript missing tool result: %s", got)
	}
}

func TestRenderTranscript_SystemMessage(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are helpful."},
	}
	got := RenderTranscript(msgs)
	if !strings.Contains(got, "[system]\nYou are helpful.\n\n") {
		t.Errorf("transcript missing system message: %s", got)
	}
}

// --- Prefix Cache Tests ---

func TestComputePrefixHash(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are helpful."},
		{Role: provider.RoleUser, Content: "Hello!"},
	}
	tools := []provider.ToolSchema{
		{Name: "Read", Parameters: []byte(`{"type":"object"}`)},
	}

	hash1, prefixLen := ComputePrefixHash(messages, tools)
	if hash1 == "" {
		t.Error("ComputePrefixHash returned empty hash")
	}
	if prefixLen != 1 {
		t.Errorf("prefixLen = %d, want 1", prefixLen)
	}

	// Same inputs should produce same hash
	hash2, _ := ComputePrefixHash(messages, tools)
	if hash1 != hash2 {
		t.Errorf("same inputs produced different hashes: %s vs %s", hash1, hash2)
	}
}

func TestComputePrefixHash_NoSystem(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello!"},
	}
	hash, prefixLen := ComputePrefixHash(messages, nil)
	if hash == "" {
		t.Error("ComputePrefixHash returned empty hash")
	}
	if prefixLen != 0 {
		t.Errorf("prefixLen = %d, want 0 (no system message)", prefixLen)
	}
}

func TestComputePrefixHash_DifferentSystem(t *testing.T) {
	m1 := []provider.Message{{Role: provider.RoleSystem, Content: "System A"}}
	m2 := []provider.Message{{Role: provider.RoleSystem, Content: "System B"}}

	h1, _ := ComputePrefixHash(m1, nil)
	h2, _ := ComputePrefixHash(m2, nil)

	if h1 == h2 {
		t.Error("different system prompts should produce different hashes")
	}
}

func TestComputePrefixHash_DifferentTools(t *testing.T) {
	m := []provider.Message{{Role: provider.RoleSystem, Content: "System"}}
	t1 := []provider.ToolSchema{{Name: "Read"}}
	t2 := []provider.ToolSchema{{Name: "Write"}}

	h1, _ := ComputePrefixHash(m, t1)
	h2, _ := ComputePrefixHash(m, t2)

	if h1 == h2 {
		t.Error("different tools should produce different hashes")
	}
}

func TestPrefixCache_CheckPrefix(t *testing.T) {
	pc := NewPrefixCache()
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are helpful."},
	}

	// First call should be a miss (cold start)
	hit := pc.CheckPrefix(messages, nil)
	if hit {
		t.Error("first CheckPrefix should return false (cold start)")
	}

	// Same prefix should be a hit
	hit = pc.CheckPrefix(messages, nil)
	if !hit {
		t.Error("second CheckPrefix with same prefix should return true")
	}

	// Changed system should be a miss
	messages[0].Content = "You are a coding assistant."
	hit = pc.CheckPrefix(messages, nil)
	if hit {
		t.Error("CheckPrefix after system change should return false")
	}

	hits, misses, _ := pc.Stats()
	if hits != 1 || misses != 2 {
		t.Errorf("stats: hits=%d misses=%d, want hits=1 misses=2", hits, misses)
	}
}

func TestPrefixCache_Reset(t *testing.T) {
	pc := NewPrefixCache()
	m := []provider.Message{{Role: provider.RoleSystem, Content: "System"}}
	pc.CheckPrefix(m, nil) // cold start miss
	pc.CheckPrefix(m, nil) // hit

	pc.Reset()
	hits, misses, _ := pc.Stats()
	if hits != 0 || misses != 0 {
		t.Errorf("after Reset: hits=%d misses=%d, want 0,0", hits, misses)
	}
}

func TestPrefixCache_Invalidate(t *testing.T) {
	pc := NewPrefixCache()
	m := []provider.Message{{Role: provider.RoleSystem, Content: "System"}}
	pc.CheckPrefix(m, nil) // cold start
	pc.CheckPrefix(m, nil) // hit

	pc.Invalidate()
	// Next check should miss
	hit := pc.CheckPrefix(m, nil)
	if hit {
		t.Error("CheckPrefix after Invalidate should return false")
	}
}

func TestPrefixCache_Stats(t *testing.T) {
	pc := NewPrefixCache()
	m := []provider.Message{{Role: provider.RoleSystem, Content: "System"}}

	pc.CheckPrefix(m, nil) // miss
	pc.CheckPrefix(m, nil) // hit
	pc.CheckPrefix(m, nil) // hit

	hits, misses, ratio := pc.Stats()
	if hits != 2 || misses != 1 {
		t.Errorf("stats: hits=%d misses=%d, want 2,1", hits, misses)
	}
	if ratio != 2.0/3.0 {
		t.Errorf("ratio = %f, want %f", ratio, 2.0/3.0)
	}
}

func TestPrefixCache_CheckSystemPromptOnly(t *testing.T) {
	pc := NewPrefixCache()
	m := []provider.Message{{Role: provider.RoleSystem, Content: "System"}}

	hit := pc.CheckSystemPromptOnly(m)
	if hit {
		t.Error("first CheckSystemPromptOnly should miss")
	}

	hit = pc.CheckSystemPromptOnly(m)
	if !hit {
		t.Error("second CheckSystemPromptOnly with same prompt should hit")
	}

	m[0].Content = "Changed!"
	hit = pc.CheckSystemPromptOnly(m)
	if hit {
		t.Error("CheckSystemPromptOnly after change should miss")
	}
}

// --- Folder Tests ---

func TestNewFolder_Defaults(t *testing.T) {
	counter := NewTokenCounter(128000)
	f := NewFolder(counter, FolderConfig{})

	if f.snipRatio != DefaultSnipRatio {
		t.Errorf("snipRatio = %f, want %f", f.snipRatio, DefaultSnipRatio)
	}
	if f.compactRatio != DefaultCompactRatio {
		t.Errorf("compactRatio = %f, want %f", f.compactRatio, DefaultCompactRatio)
	}
	if f.forceRatio != DefaultForceCompactRatio {
		t.Errorf("forceRatio = %f, want %f", f.forceRatio, DefaultForceCompactRatio)
	}
	if f.recentKeep != MinRecentKeep {
		t.Errorf("recentKeep = %d, want %d", f.recentKeep, MinRecentKeep)
	}
}

func TestFolder_ShouldFold_None(t *testing.T) {
	counter := NewTokenCounter(100000)
	f := NewFolder(counter, FolderConfig{})

	decision, _ := f.ShouldFold(40000) // 40% - below soft threshold
	if decision != FoldNone {
		t.Errorf("ShouldFold(40000) = %d, want FoldNone", decision)
	}
}

func TestFolder_ShouldFold_SoftNotice(t *testing.T) {
	counter := NewTokenCounter(100000)
	f := NewFolder(counter, FolderConfig{})

	_, softNotice := f.ShouldFold(55000) // 55% - above soft (50%), below snip (60%)
	if !softNotice {
		t.Error("ShouldFold(55000) should emit soft notice")
	}
}

func TestFolder_ShouldFold_Snip(t *testing.T) {
	counter := NewTokenCounter(100000)
	f := NewFolder(counter, FolderConfig{})

	decision, _ := f.ShouldFold(65000) // 65% - above snip (60%), below compact (80%)
	if decision != FoldSnip {
		t.Errorf("ShouldFold(65000) = %d, want FoldSnip", decision)
	}
}

func TestFolder_ShouldFold_Compact(t *testing.T) {
	counter := NewTokenCounter(100000)
	f := NewFolder(counter, FolderConfig{})

	decision, _ := f.ShouldFold(85000) // 85% - above compact (80%), below force (90%)
	if decision != FoldCompact {
		t.Errorf("ShouldFold(85000) = %d, want FoldCompact", decision)
	}
}

func TestFolder_ShouldFold_ForceCompact(t *testing.T) {
	counter := NewTokenCounter(100000)
	f := NewFolder(counter, FolderConfig{})

	decision, _ := f.ShouldFold(95000) // 95% - above force (90%)
	if decision != FoldForceCompact {
		t.Errorf("ShouldFold(95000) = %d, want FoldForceCompact", decision)
	}
}

func TestFolder_ShouldFold_NoWindow(t *testing.T) {
	counter := NewTokenCounter(0)
	f := NewFolder(counter, FolderConfig{})

	decision, _ := f.ShouldFold(50000)
	if decision != FoldNone {
		t.Errorf("ShouldFold with no window = %d, want FoldNone", decision)
	}
}

func TestFoldEconomics_TooSmall(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Hi"},
	}
	if FoldEconomics(msgs) {
		t.Error("FoldEconomics should return false for tiny region")
	}
}

func TestFoldEconomics_LargeEnough(t *testing.T) {
	// Create a message with enough content to exceed MinFoldTokens
	longContent := strings.Repeat("This is a test message with some content. ", 50)
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: longContent},
		{Role: provider.RoleAssistant, Content: longContent},
	}
	if !FoldEconomics(msgs) {
		t.Error("FoldEconomics should return true for large region")
	}
}

func TestMechanicalFoldDigest(t *testing.T) {
	got := MechanicalFoldDigest(5, "/tmp/archive.jsonl")
	if !strings.Contains(got, "5 earlier message(s)") {
		t.Errorf("MechanicalFoldDigest missing message count: %s", got)
	}
	if !strings.Contains(got, "archived to /tmp/archive.jsonl") {
		t.Errorf("MechanicalFoldDigest missing archive path: %s", got)
	}
}

func TestMechanicalFoldDigest_NoArchive(t *testing.T) {
	got := MechanicalFoldDigest(3, "")
	if !strings.Contains(got, "3 earlier message(s)") {
		t.Errorf("MechanicalFoldDigest missing message count: %s", got)
	}
	if strings.Contains(got, "archived to") {
		t.Errorf("MechanicalFoldDigest should not mention archive when empty: %s", got)
	}
}

// --- Manager Tests ---

func TestNewManager(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 128000})
	if m.ContextWindow() != 128000 {
		t.Errorf("ContextWindow = %d, want 128000", m.ContextWindow())
	}
	if m.Counter() == nil {
		t.Error("Counter() returned nil")
	}
	if m.Folder() == nil {
		t.Error("Folder() returned nil")
	}
	if m.Prefix() == nil {
		t.Error("Prefix() returned nil")
	}
}

func TestManager_Evaluate(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})

	decision, softNotice, usage := m.Evaluate(40000)
	if decision != FoldNone {
		t.Errorf("Evaluate(40000) decision = %d, want FoldNone", decision)
	}
	if softNotice {
		t.Error("Evaluate(40000) should not emit soft notice")
	}
	if usage != 0.4 {
		t.Errorf("Evaluate(40000) usage = %f, want 0.4", usage)
	}
}

func TestManager_Evaluate_SoftNotice(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})

	decision, softNotice, _ := m.Evaluate(55000)
	if decision != FoldNone {
		t.Errorf("Evaluate(55000) decision = %d, want FoldNone", decision)
	}
	if !softNotice {
		t.Error("Evaluate(55000) should emit soft notice")
	}
}

func TestManager_ValidatePrefix(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "System prompt"},
	}

	// First call should be a miss
	hit := m.ValidatePrefix(messages, nil)
	if hit {
		t.Error("first ValidatePrefix should miss")
	}

	// Same prefix should hit
	hit = m.ValidatePrefix(messages, nil)
	if !hit {
		t.Error("second ValidatePrefix with same prefix should hit")
	}
}

func TestManager_PrepareForProviderCall(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "System prompt"},
		{Role: provider.RoleUser, Content: "Hello!"},
	}
	lastUsage := &provider.Usage{PromptTokens: 100}

	prefixHit, diag := m.PrepareForProviderCall(messages, nil, lastUsage)
	if prefixHit {
		t.Error("first PrepareForProviderCall should miss prefix")
	}
	if !strings.Contains(diag, "prefix=miss") {
		t.Errorf("diagnostics should mention prefix=miss: %s", diag)
	}
	if !strings.Contains(diag, "tok_per_char=") {
		t.Errorf("diagnostics should mention tok_per_char: %s", diag)
	}
}

func TestManager_CompactIfNeeded_NoWindow(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 0})
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello"},
	}

	result, didCompact, err := m.CompactIfNeeded(context.Background(), msgs, nil, &provider.Usage{PromptTokens: 50000})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if didCompact {
		t.Error("should not compact with no context window")
	}
	if len(result) != len(msgs) {
		t.Errorf("result length = %d, want %d", len(result), len(msgs))
	}
}

func TestManager_CompactIfNeeded_NoUsage(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "Hello"},
	}

	result, didCompact, err := m.CompactIfNeeded(context.Background(), msgs, nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if didCompact {
		t.Error("should not compact with nil usage")
	}
	if len(result) != len(msgs) {
		t.Errorf("result length = %d, want %d", len(result), len(msgs))
	}
}

func TestManager_CompactIfNeeded_UnderThreshold(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "System"},
		{Role: provider.RoleUser, Content: "Hello"},
	}

	result, didCompact, err := m.CompactIfNeeded(
		context.Background(), msgs, nil,
		&provider.Usage{PromptTokens: 50000}, // 50% - under compact threshold
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if didCompact {
		t.Error("should not compact under threshold")
	}
	if len(result) != len(msgs) {
		t.Errorf("result length = %d, want %d", len(result), len(msgs))
	}
}

func TestManager_SetContextWindow(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})
	m.SetContextWindow(200000)
	if m.ContextWindow() != 200000 {
		t.Errorf("ContextWindow = %d, want 200000", m.ContextWindow())
	}
}

func TestManager_ResetState(t *testing.T) {
	m := NewManager(ManagerConfig{ContextWindow: 100000})
	messages := []provider.Message{{Role: provider.RoleSystem, Content: "System"}}
	m.ValidatePrefix(messages, nil)

	m.ResetState()
	hits, misses, _ := m.Prefix().Stats()
	if hits != 0 || misses != 0 {
		t.Errorf("after ResetState: hits=%d misses=%d, want 0,0", hits, misses)
	}
}

// --- Integration Test ---

func TestIntegration_FullFlow(t *testing.T) {
	// Simulate a full context management flow:
	// 1. Create manager
	// 2. Build up context
	// 3. Evaluate and decide to compact
	// 4. Verify compaction

	m := NewManager(ManagerConfig{ContextWindow: 100000})

	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a coding assistant."},
	}

	// Simulate adding messages and checking prefix
	for i := 0; i < 5; i++ {
		messages = append(messages, provider.Message{
			Role:    provider.RoleUser,
			Content: "Message " + string(rune('A'+i)),
		})
		messages = append(messages, provider.Message{
			Role:    provider.RoleAssistant,
			Content: "Response " + string(rune('A'+i)),
		})
	}

	// Validate prefix stability
	hit := m.ValidatePrefix(messages, nil)
	// First call is cold start miss
	if hit {
		t.Error("first ValidatePrefix should miss")
	}

	// Second call should hit (same prefix)
	hit = m.ValidatePrefix(messages, nil)
	if !hit {
		t.Error("second ValidatePrefix should hit")
	}

	// Evaluate at low usage
	decision, _, usage := m.Evaluate(20000)
	if decision != FoldNone {
		t.Errorf("low usage: decision = %d, want FoldNone", decision)
	}
	if usage != 0.2 {
		t.Errorf("usage = %f, want 0.2", usage)
	}

	// Evaluate at high usage
	decision, _, usage = m.Evaluate(85000)
	if decision != FoldCompact {
		t.Errorf("high usage: decision = %d, want FoldCompact", decision)
	}
	if usage != 0.85 {
		t.Errorf("usage = %f, want 0.85", usage)
	}
}

func TestSummarySystemPrompt(t *testing.T) {
	if !strings.Contains(SummarySystemPrompt, "Standing facts") {
		t.Error("SummarySystemPrompt missing 'Standing facts' heading")
	}
	if !strings.Contains(SummarySystemPrompt, "Goal") {
		t.Error("SummarySystemPrompt missing 'Goal' heading")
	}
	if !strings.Contains(SummarySystemPrompt, "Pending") {
		t.Error("SummarySystemPrompt missing 'Pending' heading")
	}
}

func TestRenderTranscriptForSummarization(t *testing.T) {
	region := []provider.Message{
		{Role: provider.RoleUser, Content: "Do something"},
		{Role: provider.RoleAssistant, Content: "OK"},
	}
	msgs := RenderTranscriptForSummarization(region, "")
	if len(msgs) != 2 {
		t.Errorf("RenderTranscriptForSummarization returned %d messages, want 2", len(msgs))
	}
	if msgs[0].Role != provider.RoleSystem {
		t.Errorf("first message role = %s, want system", msgs[0].Role)
	}
	if !strings.Contains(msgs[0].Content, SummarySystemPrompt) {
		t.Error("system message should contain SummarySystemPrompt")
	}
	if msgs[1].Role != provider.RoleUser {
		t.Errorf("second message role = %s, want user", msgs[1].Role)
	}
}

func TestRenderTranscriptForSummarization_WithInstructions(t *testing.T) {
	region := []provider.Message{
		{Role: provider.RoleUser, Content: "Do something"},
	}
	msgs := RenderTranscriptForSummarization(region, "Focus on error handling")
	if !strings.Contains(msgs[0].Content, "Focus on error handling") {
		t.Error("system message should contain custom instructions")
	}
}

// --- SHA-256 hash verification ---

func TestComputePrefixHash_SHA256(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: "Test"},
	}
	hash, _ := ComputePrefixHash(messages, nil)

	// Verify it's a valid hex-encoded SHA-256
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64 (hex SHA-256)", len(hash))
	}

	// Verify against manual SHA-256
	h := sha256.New()
	h.Write([]byte("Test"))
	h.Write([]byte{0})
	expected := hex.EncodeToString(h.Sum(nil))
	if hash != expected {
		t.Errorf("hash = %s, want %s", hash, expected)
	}
}
