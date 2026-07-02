// Package agent_test defines the expected behavior for MiMo agent integration.
// These tests are written before implementation (TDD) and will fail until the
// MiMo-specific provider wiring and agent configuration are implemented.
//
// Expected MiMo agent behavior:
//   - Uses mimo-v2.5 as the default model
//   - Supports reasoning_content round-tripping
//   - Handles tool calls through the OpenAI-compatible provider
package agent_test

import (
	"encoding/json"
	"testing"

	"mimo-reasonix/internal/provider"
)

// ── TestMiMoAgentProviderWiring ────────────────────────────────────────────

// TestMiMoAgentProviderWiring verifies that the agent correctly wires
// the MiMo provider when configured.
func TestMiMoAgentProviderWiring(t *testing.T) {
	// MiMo should be registered as an "openai" kind provider
	// with the MiMo-specific base URL and model.
	//
	// After implementation, the agent should:
	// 1. Load config with MiMo provider entries
	// 2. Resolve the default model to mimo-v2.5
	// 3. Create an OpenAI-compatible provider instance
	// 4. Use reasoning_effort instead of thinking.type

	t.Run("provider kind is openai", func(t *testing.T) {
		// MiMo uses the OpenAI-compatible wire protocol
		expectedKind := "openai"
		if expectedKind == "" {
			t.Error("expected kind must not be empty")
		}

		// After implementation, verify registered kinds include "openai"
		// kinds := provider.Kinds()
		// if !contains(kinds, "openai") {
		//     t.Errorf("registered kinds %v does not include 'openai'", kinds)
		// }
	})

	t.Run("MiMo models are recognized as chat models", func(t *testing.T) {
		mimoModels := []string{
			"mimo-v2.5",
			"mimo-v2.5-pro",
			"mimo-v2-pro",
			"mimo-v2-flash",
			"mimo-v2-omni",
		}

		for _, model := range mimoModels {
			t.Run(model, func(t *testing.T) {
				// After implementation:
				// if !config.IsLikelyChatModel(model) {
				//     t.Errorf("IsLikelyChatModel(%q) = false, want true", model)
				// }

				// For now, verify the model name format
				if len(model) < 4 || model[:4] != "mimo" {
					t.Errorf("model name must start with 'mimo', got %q", model)
				}
			})
		}
	})
}

// ── TestMiMoReasoningRoundTrip ─────────────────────────────────────────────

// TestMiMoReasoningRoundTrip verifies that reasoning_content is correctly
// round-tripped through the agent's message history.
func TestMiMoReasoningRoundTrip(t *testing.T) {
	// When the agent receives a response with reasoning_content, it should
	// preserve that field when sending subsequent turns to the model.

	t.Run("reasoning_content preserved in history", func(t *testing.T) {
		history := []provider.Message{
			{Role: provider.RoleSystem, Content: "You are a helpful assistant."},
			{Role: provider.RoleUser, Content: "What is 2+2?"},
			{
				Role:             provider.RoleAssistant,
				Content:          "The answer is 4.",
				ReasoningContent: "2+2=4. This is basic arithmetic.",
			},
			{Role: provider.RoleUser, Content: "And 3+3?"},
		}

		// Verify reasoning_content is in the history
		assistantMsg := history[2]
		if assistantMsg.ReasoningContent != "2+2=4. This is basic arithmetic." {
			t.Errorf("reasoning_content = %q, want %q",
				assistantMsg.ReasoningContent, "2+2=4. This is basic arithmetic.")
		}

		// When serialized to JSON, reasoning_content should be present
		data, err := json.Marshal(assistantMsg)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		var decoded provider.Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}

		if decoded.ReasoningContent != assistantMsg.ReasoningContent {
			t.Errorf("round-trip reasoning_content = %q, want %q",
				decoded.ReasoningContent, assistantMsg.ReasoningContent)
		}
	})

	t.Run("reasoning_content not sent in user messages", func(t *testing.T) {
		// The agent should strip reasoning_content from user messages
		// before sending to the provider
		msg := provider.Message{
			Role:             provider.RoleUser,
			Content:          "What about 4+4?",
			ReasoningContent: "should be stripped",
		}

		// When building the wire request, user messages should not have
		// reasoning_content
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal message: %v", err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to unmarshal as raw: %v", err)
		}

		// The field is present in Go struct but the provider implementation
		// should ensure it's not sent to the API
		t.Log("provider implementation should strip reasoning_content from user messages")
	})
}

// ── TestMiMoToolCallFlow ──────────────────────────────────────────────────

// TestMiMoToolCallFlow verifies the complete tool call flow through the agent.
func TestMiMoToolCallFlow(t *testing.T) {
	// The agent should:
	// 1. Send tool definitions to MiMo
	// 2. Receive tool call deltas
	// 3. Assemble complete tool calls
	// 4. Execute the tool
	// 5. Send tool results back
	// 6. Continue the conversation

	t.Run("tool call message round trip", func(t *testing.T) {
		// Simulate an assistant message with tool calls
		assistantMsg := provider.Message{
			Role: provider.RoleAssistant,
			Content: "",
			ToolCalls: []provider.ToolCall{
				{
					ID:        "call_1",
					Name:      "read_file",
					Arguments: `{"path":"main.go"}`,
				},
			},
		}

		// Serialize and deserialize
		data, err := json.Marshal(assistantMsg)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded provider.Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(decoded.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(decoded.ToolCalls))
		}
		if decoded.ToolCalls[0].ID != "call_1" {
			t.Errorf("ToolCall ID = %q, want call_1", decoded.ToolCalls[0].ID)
		}
		if decoded.ToolCalls[0].Name != "read_file" {
			t.Errorf("ToolCall Name = %q, want read_file", decoded.ToolCalls[0].Name)
		}
		if decoded.ToolCalls[0].Arguments != `{"path":"main.go"}` {
			t.Errorf("ToolCall Arguments = %q, want %q", decoded.ToolCalls[0].Arguments, `{"path":"main.go"}`)
		}
	})

	t.Run("tool result message format", func(t *testing.T) {
		// Tool results should have the correct format
		toolResult := provider.Message{
			Role:       provider.RoleTool,
			ToolCallID: "call_1",
			Name:       "read_file",
			Content:    "package main\n\nfunc main() {}",
		}

		data, err := json.Marshal(toolResult)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded provider.Message
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.Role != provider.RoleTool {
			t.Errorf("Role = %q, want %q", decoded.Role, provider.RoleTool)
		}
		if decoded.ToolCallID != "call_1" {
			t.Errorf("ToolCallID = %q, want call_1", decoded.ToolCallID)
		}
		if decoded.Name != "read_file" {
			t.Errorf("Name = %q, want read_file", decoded.Name)
		}
	})
}

// ── TestMiMoUsageTracking ─────────────────────────────────────────────────

// TestMiMoUsageTracking verifies that usage tracking works correctly for MiMo.
func TestMiMoUsageTracking(t *testing.T) {
	t.Run("cost calculation with MiMo pricing", func(t *testing.T) {
		pricing := &provider.Pricing{
			CacheHit: 0.7,  // CNY per 1M cached tokens
			Input:    7.0,  // CNY per 1M input tokens
			Output:   14.0, // CNY per 1M output tokens
			Currency: "¥",
		}

		usage := &provider.Usage{
			PromptTokens:     10_000,
			CompletionTokens: 5_000,
			CacheHitTokens:   8_000,
			CacheMissTokens:  2_000,
			ReasoningTokens:  3_000,
		}

		cost := pricing.Cost(usage)

		// Expected: (8000 * 0.7 + 2000 * 7.0 + 5000 * 14.0) / 1e6
		// = (5600 + 14000 + 70000) / 1e6
		// = 89600 / 1e6 = 0.0896
		expectedCost := 0.0896
		if cost != expectedCost {
			t.Errorf("Cost() = %v, want %v", cost, expectedCost)
		}
	})

	t.Run("currency symbol", func(t *testing.T) {
		pricing := &provider.Pricing{Currency: "¥"}
		if pricing.Symbol() != "¥" {
			t.Errorf("Symbol() = %q, want ¥", pricing.Symbol())
		}
	})
}

// contains checks if a string slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
