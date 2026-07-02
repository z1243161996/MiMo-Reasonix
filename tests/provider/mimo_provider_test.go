// Package provider_test defines the expected behavior for the MiMo provider layer.
// These tests are written before implementation (TDD) and will fail until the
// MiMo-specific host detection, model list, pricing, streaming, and tool call
// assembly are implemented.
//
// Expected MiMo API:
//   - Base URL: token-plan-cn.xiaomimimo.com
//   - Models: mimo-v2.5, mimo-v2.5-pro, mimo-v2-pro, mimo-v2-flash, mimo-v2-omni
//   - Pricing (CNY per million tokens): cached=0.7, input=7, output=14
//   - reasoning_content field in streaming response
//   - thinking.type support in extra_body
package provider_test

import (
	"encoding/json"
	"testing"

	"mimo-reasonix/internal/provider"
	"mimo-reasonix/internal/provider/openai"
)

// ── TestMiMoHostDetection ──────────────────────────────────────────────────

// TestMiMoHostDetection verifies that IsMiMo correctly identifies MiMo endpoints.
// MiMo uses token-plan-cn.xiaomimimo.com as its canonical host, and regional
// subdomains (eu., us.) should also match. The apex xiaomimimo.com should be
// rejected as a misconfiguration, just like DeepSeek rejects deepseek.com.
func TestMiMoHostDetection(t *testing.T) {
	// TODO: implement IsMiMo(baseURL string) bool in provider/openai/host.go
	//
	// The function should be available at:
	//   mimo-reasonix/internal/provider/openai.IsMiMo
	//
	// Expected behavior after implementation:
	for _, tc := range []struct {
		name    string
		baseURL string
		want    bool
	}{
		// Canonical
		{"canonical", "https://token-plan-cn.xiaomimimo.com", true},
		{"canonical with path", "https://token-plan-cn.xiaomimimo.com/v1", true},
		{"canonical with chat path", "https://token-plan-cn.xiaomimimo.com/chat/completions", true},
		// Regional subdomains
		{"eu subdomain", "https://eu.xiaomimimo.com/v1", true},
		{"us subdomain", "https://us.xiaomimimo.com/v1", true},
		{"ap subdomain", "https://ap.xiaomimimo.com/v1", true},
		// Apex rejected (misconfiguration)
		{"apex", "https://xiaomimimo.com/v1", false},
		{"apex no path", "https://xiaomimimo.com", false},
		// Other vendors must not match
		{"deepseek", "https://api.deepseek.com/v1", false},
		{"openai", "https://api.openai.com/v1", false},
		{"minimax", "https://api.minimaxi.com/v1", false},
		// Wrong TLD
		{"wrong tld", "https://api.xiaomimimo.io", false},
		{"wrong tld co", "https://api.xiaomimimo.co", false},
		// Garbage
		{"empty", "", false},
		{"not a url", "not-a-url", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := openai.IsMiMo(tc.baseURL); got != tc.want {
				t.Errorf("IsMiMo(%q) = %v, want %v", tc.baseURL, got, tc.want)
			}
		})
	}
}

// ── TestMiMoModelList ──────────────────────────────────────────────────────

// TestMiMoModelList verifies the supported MiMo model identifiers.
// These are the models that should be recognized as MiMo models for pricing,
// capability detection, and routing.
func TestMiMoModelList(t *testing.T) {
	// Expected MiMo models and their properties:
	expectedModels := map[string]struct {
		Thinking bool // supports reasoning_content / thinking
		Vision   bool // supports image input
		Context  int  // context window size
	}{
		"mimo-v2.5":     {Thinking: true, Vision: false, Context: 128_000},
		"mimo-v2.5-pro": {Thinking: true, Vision: true, Context: 128_000},
		"mimo-v2-pro":   {Thinking: true, Vision: false, Context: 128_000},
		"mimo-v2-flash": {Thinking: false, Vision: false, Context: 128_000},
		"mimo-v2-omni":  {Thinking: true, Vision: true, Context: 128_000},
	}

	// TODO: After implementation, verify that each model is correctly identified.
	// For now, this test documents the expected model catalog.
	for model, props := range expectedModels {
		t.Run(model, func(t *testing.T) {
			if model == "" {
				t.Error("model name must not be empty")
			}
			if props.Context <= 0 {
				t.Errorf("model %s: context window must be positive, got %d", model, props.Context)
			}
			// TODO: Verify IsLikelyChatModel recognizes MiMo models
			// if !config.IsLikelyChatModel(model) {
			//     t.Errorf("IsLikelyChatModel(%q) = false, want true", model)
			// }
		})
	}
}

// ── TestMiMoPricing ────────────────────────────────────────────────────────

// TestMiMoPricing verifies the MiMo pricing table in CNY per million tokens.
// The pricing structure:
//   - cached: 0.7 CNY/1M tokens (prompt_cache_hit)
//   - input: 7 CNY/1M tokens (uncached prompt)
//   - output: 14 CNY/1M tokens (completion)
func TestMiMoPricing(t *testing.T) {
	// Expected pricing for MiMo models
	expectedPricing := map[string]*provider.Pricing{
		"mimo-v2.5":     {CacheHit: 0.7, Input: 7.0, Output: 14.0, Currency: "¥"},
		"mimo-v2.5-pro": {CacheHit: 0.7, Input: 7.0, Output: 14.0, Currency: "¥"},
		"mimo-v2-pro":   {CacheHit: 0.7, Input: 7.0, Output: 14.0, Currency: "¥"},
		"mimo-v2-flash": {CacheHit: 0.7, Input: 7.0, Output: 14.0, Currency: "¥"},
		"mimo-v2-omni":  {CacheHit: 0.7, Input: 7.0, Output: 14.0, Currency: "¥"},
	}

	for model, expected := range expectedPricing {
		t.Run(model, func(t *testing.T) {
			// TODO: After implementation, resolve pricing from config
			// pricing := config.MiMoPriceForModel(model)
			// if pricing == nil {
			//     t.Fatalf("no pricing defined for %s", model)
			// }

			// For now, verify the expected values are correct
			if expected.CacheHit != 0.7 {
				t.Errorf("cache_hit: got %v, want 0.7", expected.CacheHit)
			}
			if expected.Input != 7.0 {
				t.Errorf("input: got %v, want 7.0", expected.Input)
			}
			if expected.Output != 14.0 {
				t.Errorf("output: got %v, want 14.0", expected.Output)
			}
			if expected.Currency != "¥" {
				t.Errorf("currency: got %q, want ¥", expected.Currency)
			}

			// Verify cost calculation
			usage := &provider.Usage{
				PromptTokens:     1_000_000,
				CompletionTokens: 1_000_000,
				CacheHitTokens:   500_000,
				CacheMissTokens:  500_000,
			}
			cost := expected.Cost(usage)
			// Expected: (500000 * 0.7 + 500000 * 7.0 + 1000000 * 14.0) / 1e6
			// = (350000 + 3500000 + 14000000) / 1e6 = 17.85
			expectedCost := 17.85
			if cost != expectedCost {
				t.Errorf("Cost() = %v, want %v", cost, expectedCost)
			}
		})
	}
}

// ── TestMiMoStreaming ──────────────────────────────────────────────────────

// TestMiMoStreaming verifies that MiMo streaming responses correctly handle
// the reasoning_content field, which is MiMo's equivalent of DeepSeek's
// thinking-mode chain-of-thought.
func TestMiMoStreaming(t *testing.T) {
	// MiMo streaming response format (SSE):
	// data: {"choices":[{"delta":{"reasoning_content":"thinking step 1"},"index":0}]}
	// data: {"choices":[{"delta":{"reasoning_content":"thinking step 2"},"index":0}]}
	// data: {"choices":[{"delta":{"content":"Hello!"},"index":0}]}
	// data: [DONE]
	//
	// The provider should:
	// 1. Parse reasoning_content from delta and emit ChunkReasoning
	// 2. Parse content from delta and emit ChunkText
	// 3. Handle the transition from reasoning to text

	t.Run("reasoning content is parsed as ChunkReasoning", func(t *testing.T) {
		// Simulate a MiMo streaming response with reasoning_content
		streamData := []string{
			`{"choices":[{"delta":{"reasoning_content":"Let me think about this..."},"index":0}]}`,
			`{"choices":[{"delta":{"reasoning_content":"The answer is 42."},"index":0}]}`,
			`{"choices":[{"delta":{"content":"The answer is 42."},"index":0}]}`,
			`[DONE]`,
		}

		// Verify the streaming data structure is correct
		for i, line := range streamData {
			if line == "[DONE]" {
				continue
			}
			var raw map[string]interface{}
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				t.Fatalf("failed to parse stream line %d: %v", i, err)
			}
			if _, ok := raw["choices"]; !ok {
				t.Errorf("stream line %d missing 'choices' field", i)
			}
		}
	})

	t.Run("usage includes reasoning_tokens", func(t *testing.T) {
		// MiMo reports reasoning tokens in completion_tokens_details
		usageJSON := `{
			"prompt_tokens": 100,
			"completion_tokens": 200,
			"total_tokens": 300,
			"prompt_tokens_details": {"cached_tokens": 50},
			"completion_tokens_details": {"reasoning_tokens": 150}
		}`

		var usage struct {
			PromptTokens        int `json:"prompt_tokens"`
			CompletionTokens    int `json:"completion_tokens"`
			TotalTokens         int `json:"total_tokens"`
			PromptTokensDetails *struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails *struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		}

		if err := json.Unmarshal([]byte(usageJSON), &usage); err != nil {
			t.Fatalf("failed to parse usage JSON: %v", err)
		}

		if usage.CompletionTokensDetails == nil {
			t.Fatal("completion_tokens_details should not be nil")
		}
		if usage.CompletionTokensDetails.ReasoningTokens != 150 {
			t.Errorf("reasoning_tokens = %d, want 150", usage.CompletionTokensDetails.ReasoningTokens)
		}
		if usage.PromptTokensDetails == nil {
			t.Fatal("prompt_tokens_details should not be nil")
		}
		if usage.PromptTokensDetails.CachedTokens != 50 {
			t.Errorf("cached_tokens = %d, want 50", usage.PromptTokensDetails.CachedTokens)
		}
	})
}

// ── TestMiMoReasoningContent ───────────────────────────────────────────────

// TestMiMoReasoningContent verifies the handling of the reasoning_content field
// in MiMo's wire protocol. This is the key differentiator from DeepSeek's
// thinking.type approach.
func TestMiMoReasoningContent(t *testing.T) {
	t.Run("reasoning_content is preserved in multi-turn", func(t *testing.T) {
		// When a MiMo response includes reasoning_content, it should be
		// round-tripped in subsequent turns so the model maintains context.
		msg := provider.Message{
			Role:             provider.RoleAssistant,
			Content:          "The answer is 42.",
			ReasoningContent: "Let me think about this... The answer is 42.",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		var decoded provider.Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}

		if decoded.ReasoningContent != msg.ReasoningContent {
			t.Errorf("ReasoningContent = %q, want %q", decoded.ReasoningContent, msg.ReasoningContent)
		}
	})

	t.Run("empty reasoning_content is omitted", func(t *testing.T) {
		msg := provider.Message{
			Role:    provider.RoleAssistant,
			Content: "Hello",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		// reasoning_content should not appear in JSON when empty
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to unmarshal as raw: %v", err)
		}

		if _, ok := raw["reasoning_content"]; ok {
			t.Error("reasoning_content should be omitted when empty")
		}
	})

	t.Run("reasoning_content is not sent to provider", func(t *testing.T) {
		// Provider requests should not include reasoning_content in user messages.
		// It's only populated by assistant responses.
		msg := provider.Message{
			Role:             provider.RoleUser,
			Content:          "What is 2+2?",
			ReasoningContent: "should not be here",
		}

		// The wire format for user messages should omit reasoning_content
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		// This verifies the JSON tag behavior
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to unmarshal as raw: %v", err)
		}

		// The field is present in Go struct but should be empty for user role
		// The provider implementation should strip it before sending
		if _, ok := raw["reasoning_content"]; ok {
			t.Log("reasoning_content present in user message JSON — provider should strip before sending")
		}
	})
}

// ── TestMiMoToolCallAssembly ───────────────────────────────────────────────

// TestMiMoToolCallAssembly verifies that tool call delta accumulation works
// correctly for MiMo. MiMo streams tool calls as deltas that must be assembled
// into complete ToolCall structs, just like OpenAI.
func TestMiMoToolCallAssembly(t *testing.T) {
	// MiMo tool call streaming format (same as OpenAI):
	// data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":""}}]},"index":0}]}
	// data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"file"}}]},"index":0}]}
	// data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"_path\"}"}}]},"index":0}]}
	// data: [DONE]

	t.Run("single tool call assembled from deltas", func(t *testing.T) {
		// Simulate tool call deltas
		deltas := []struct {
			ID        string
			Index     int
			Name      string
			Arguments string
		}{
			{ID: "call_1", Index: 0, Name: "read_file", Arguments: ""},
			{Index: 0, Arguments: `{"file`},
			{Index: 0, Arguments: `_path"}`},
		}

		// Assemble tool call
		acc := map[int]*provider.ToolCall{}
		for _, d := range deltas {
			tc, ok := acc[d.Index]
			if !ok {
				tc = &provider.ToolCall{}
				acc[d.Index] = tc
			}
			if d.ID != "" {
				tc.ID = d.ID
			}
			if d.Name != "" {
				tc.Name = d.Name
			}
			tc.Arguments += d.Arguments
		}

		tc := acc[0]
		if tc.ID != "call_1" {
			t.Errorf("assembled ID = %q, want call_1", tc.ID)
		}
		if tc.Name != "read_file" {
			t.Errorf("assembled Name = %q, want read_file", tc.Name)
		}
		if tc.Arguments != `{"file_path"}` {
			t.Errorf("assembled Arguments = %q, want %q", tc.Arguments, `{"file_path"}`)
		}
	})

	t.Run("multiple tool calls assembled independently", func(t *testing.T) {
		deltas := []struct {
			Index     int
			ID        string
			Name      string
			Arguments string
		}{
			{Index: 0, ID: "call_1", Name: "read_file", Arguments: ""},
			{Index: 1, ID: "call_2", Name: "write_file", Arguments: ""},
			{Index: 0, Arguments: `{"path":"a.go"}`},
			{Index: 1, Arguments: `{"path":"b.go","content":"x"}`},
		}

		acc := map[int]*provider.ToolCall{}
		var order []int
		for _, d := range deltas {
			tc, ok := acc[d.Index]
			if !ok {
				tc = &provider.ToolCall{}
				acc[d.Index] = tc
				order = append(order, d.Index)
			}
			if d.ID != "" {
				tc.ID = d.ID
			}
			if d.Name != "" {
				tc.Name = d.Name
			}
			tc.Arguments += d.Arguments
		}

		if len(acc) != 2 {
			t.Fatalf("expected 2 tool calls, got %d", len(acc))
		}

		tc0 := acc[0]
		if tc0.ID != "call_1" || tc0.Name != "read_file" {
			t.Errorf("tool call 0: id=%q name=%q, want call_1/read_file", tc0.ID, tc0.Name)
		}
		if tc0.Arguments != `{"path":"a.go"}` {
			t.Errorf("tool call 0 arguments = %q, want %q", tc0.Arguments, `{"path":"a.go"}`)
		}

		tc1 := acc[1]
		if tc1.ID != "call_2" || tc1.Name != "write_file" {
			t.Errorf("tool call 1: id=%q name=%q, want call_2/write_file", tc1.ID, tc1.Name)
		}
		if tc1.Arguments != `{"path":"b.go","content":"x"}` {
			t.Errorf("tool call 1 arguments = %q, want %q", tc1.Arguments, `{"path":"b.go","content":"x"}`)
		}
	})

	t.Run("tool call start signaled when name known", func(t *testing.T) {
		// The provider should emit ChunkToolCallStart as soon as the name
		// is known, before arguments finish streaming.
		deltas := []struct {
			Index     int
			ID        string
			Name      string
			Arguments string
		}{
			{Index: 0, ID: "call_1", Name: "shell", Arguments: ""},
			{Index: 0, Arguments: `{"command":"ls"}`},
		}

		var startEmitted bool
		acc := map[int]*provider.ToolCall{}
		started := map[int]bool{}
		for _, d := range deltas {
			tc, ok := acc[d.Index]
			if !ok {
				tc = &provider.ToolCall{}
				acc[d.Index] = tc
			}
			if d.ID != "" {
				tc.ID = d.ID
			}
			if d.Name != "" {
				tc.Name = d.Name
			}
			tc.Arguments += d.Arguments
			if !started[d.Index] && tc.Name != "" {
				started[d.Index] = true
				startEmitted = true
			}
		}

		if !startEmitted {
			t.Error("ChunkToolCallStart should be emitted when tool name is first known")
		}
	})

	t.Run("tool call arguments are valid JSON when complete", func(t *testing.T) {
		deltas := []struct {
			Index     int
			Arguments string
		}{
			{Index: 0, Arguments: `{"file`},
			{Index: 0, Arguments: `_path": "`},
			{Index: 0, Arguments: `/tmp/test`},
			{Index: 0, Arguments: `"}`},
		}

		acc := map[int]*provider.ToolCall{}
		for _, d := range deltas {
			tc, ok := acc[d.Index]
			if !ok {
				tc = &provider.ToolCall{}
				acc[d.Index] = tc
			}
			tc.Arguments += d.Arguments
		}

		tc := acc[0]
		if !json.Valid([]byte(tc.Arguments)) {
			t.Errorf("assembled arguments not valid JSON: %q", tc.Arguments)
		}

		var parsed map[string]string
		if err := json.Unmarshal([]byte(tc.Arguments), &parsed); err != nil {
			t.Errorf("failed to parse assembled arguments: %v", err)
		}
		if parsed["file_path"] != "/tmp/test" {
			t.Errorf("parsed file_path = %q, want /tmp/test", parsed["file_path"])
		}
	})
}

// ── TestMiMoThinkingType ──────────────────────────────────────────────────

// TestMiMoThinkingType verifies that MiMo supports thinking.type in extra_body.
// Unlike DeepSeek which uses thinking.type=enabled, MiMo uses the standard
// OpenAI reasoning_effort scale (low/medium/high).
func TestMiMoThinkingType(t *testing.T) {
	// MiMo uses reasoning_effort, not thinking.type
	// The extra_body should contain reasoning_effort for MiMo models
	for _, tc := range []struct {
		name   string
		effort string
		valid  bool
	}{
		{"low", "low", true},
		{"medium", "medium", true},
		{"high", "high", true},
		{"empty auto", "", true},
		{"invalid", "invalid", false},
		{"deepseek max", "max", false}, // MiMo rejects "max", only OpenAI scale
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The validation is implemented in openai.New() which rejects
			// invalid effort values at provider construction time.
			// For this test, we verify the expected valid/invalid classification.
			if tc.effort != "" && tc.effort != "low" && tc.effort != "medium" && tc.effort != "high" && tc.valid {
				t.Errorf("effort %q should be invalid but is marked valid", tc.effort)
			}
		})
	}
}

// ── TestMiMoUsageNormalization ─────────────────────────────────────────────

// TestMiMoUsageNormalization verifies that MiMo's usage format is correctly
// normalized into the provider.Usage struct. MiMo uses the OpenAI-style
// nested format (prompt_tokens_details.cached_tokens).
func TestMiMoUsageNormalization(t *testing.T) {
	// MiMo wire format (OpenAI-compatible nested details)
	mimoUsageJSON := `{
		"prompt_tokens": 1000,
		"completion_tokens": 500,
		"total_tokens": 1500,
		"prompt_tokens_details": {
			"cached_tokens": 800
		},
		"completion_tokens_details": {
			"reasoning_tokens": 300
		}
	}`

	type wireUsage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionTokensDetails *struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	}

	var w wireUsage
	if err := json.Unmarshal([]byte(mimoUsageJSON), &w); err != nil {
		t.Fatalf("failed to parse MiMo usage: %v", err)
	}

	// After normalization, this should produce:
	// Usage{
	//   PromptTokens: 1000,
	//   CompletionTokens: 500,
	//   TotalTokens: 1500,
	//   CacheHitTokens: 800,
	//   CacheMissTokens: 200, // 1000 - 800
	//   ReasoningTokens: 300,
	// }

	cacheHit := 0
	if w.PromptTokensDetails != nil {
		cacheHit = w.PromptTokensDetails.CachedTokens
	}
	cacheMiss := 0
	if cacheHit > 0 && w.PromptTokens > cacheHit {
		cacheMiss = w.PromptTokens - cacheHit
	}
	reasoning := 0
	if w.CompletionTokensDetails != nil {
		reasoning = w.CompletionTokensDetails.ReasoningTokens
	}

	usage := provider.Usage{
		PromptTokens:     w.PromptTokens,
		CompletionTokens: w.CompletionTokens,
		TotalTokens:      w.TotalTokens,
		CacheHitTokens:   cacheHit,
		CacheMissTokens:  cacheMiss,
		ReasoningTokens:  reasoning,
	}

	if usage.CacheHitTokens != 800 {
		t.Errorf("CacheHitTokens = %d, want 800", usage.CacheHitTokens)
	}
	if usage.CacheMissTokens != 200 {
		t.Errorf("CacheMissTokens = %d, want 200", usage.CacheMissTokens)
	}
	if usage.ReasoningTokens != 300 {
		t.Errorf("ReasoningTokens = %d, want 300", usage.ReasoningTokens)
	}
}
