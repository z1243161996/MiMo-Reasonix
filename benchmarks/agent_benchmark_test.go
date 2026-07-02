package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"mimo-reasonix/internal/agent"
	"mimo-reasonix/internal/provider"
	"mimo-reasonix/internal/tool"
)

// BenchmarkAgentLoop benchmarks the core session operations that drive the
// agent harness: message append, snapshot, and replace. The full agent loop
// requires a live provider, so we benchmark the hot-path primitives the loop
// depends on.
func BenchmarkAgentLoop(b *testing.B) {
	b.Run("session_add", func(b *testing.B) {
		s := agent.NewSession("system prompt")
		for b.Loop() {
			s.Add(provider.Message{
				Role:    provider.RoleUser,
				Content: "test message",
			})
		}
	})

	b.Run("session_snapshot", func(b *testing.B) {
		s := agent.NewSession("system prompt")
		for i := 0; i < 100; i++ {
			s.Add(provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf("message %d", i),
			})
			s.Add(provider.Message{
				Role:    provider.RoleAssistant,
				Content: fmt.Sprintf("response %d", i),
			})
		}
		b.ResetTimer()
		for b.Loop() {
			_ = s.Snapshot()
		}
	})

	b.Run("session_replace", func(b *testing.B) {
		s := agent.NewSession("system prompt")
		for i := 0; i < 50; i++ {
			s.Add(provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf("message %d", i),
			})
		}
		msgs := make([]provider.Message, 25)
		for i := range msgs {
			msgs[i] = provider.Message{
				Role:    provider.RoleAssistant,
				Content: "compacted",
			}
		}
		b.ResetTimer()
		for b.Loop() {
			s.Replace(msgs)
		}
	})

	b.Run("session_has_content", func(b *testing.B) {
		s := agent.NewSession("system prompt")
		for i := 0; i < 20; i++ {
			s.Add(provider.Message{Role: provider.RoleUser, Content: "msg"})
			s.Add(provider.Message{Role: provider.RoleAssistant, Content: "resp"})
		}
		b.ResetTimer()
		for b.Loop() {
			_ = s.HasContent()
		}
	})
}

// BenchmarkToolExecution benchmarks the tool Registry operations: adding
// tools, looking them up, and generating schema lists for provider requests.
// The full tool execution (calling Tool.Execute) is I/O-bound and varies
// by tool, so we benchmark the registry management overhead.
func BenchmarkToolExecution(b *testing.B) {
	b.Run("registry_add", func(b *testing.B) {
		for b.Loop() {
			r := tool.NewRegistry()
			for _, t := range mockTools(10) {
				r.Add(t)
			}
		}
	})

	b.Run("registry_get", func(b *testing.B) {
		r := tool.NewRegistry()
		for _, t := range mockTools(25) {
			r.Add(t)
		}
		b.ResetTimer()
		for b.Loop() {
			_, _ = r.Get("tool-12")
		}
	})

	b.Run("registry_get_miss", func(b *testing.B) {
		r := tool.NewRegistry()
		for _, t := range mockTools(25) {
			r.Add(t)
		}
		b.ResetTimer()
		for b.Loop() {
			_, _ = r.Get("nonexistent-tool")
		}
	})

	b.Run("registry_schemas", func(b *testing.B) {
		r := tool.NewRegistry()
		for _, t := range mockTools(25) {
			r.Add(t)
		}
		b.ResetTimer()
		for b.Loop() {
			_ = r.Schemas()
		}
	})

	b.Run("registry_remove_prefix", func(b *testing.B) {
		for b.Loop() {
			r := tool.NewRegistry()
			for _, t := range mockTools(25) {
				r.Add(t)
			}
			// Add MCP-prefixed tools
			for i := 0; i < 10; i++ {
				r.Add(&mockTool{name: fmt.Sprintf("mcp__server__tool%d", i)})
			}
			_ = r.RemovePrefix("mcp__server__")
		}
	})

	b.Run("tool_execute_noop", func(b *testing.B) {
		t := &noopTool{}
		args := json.RawMessage(`{"input": "test"}`)
		ctx := context.Background()
		for b.Loop() {
			_, _ = t.Execute(ctx, args)
		}
	})
}

// BenchmarkContextManagement benchmarks the context compaction primitives:
// message normalization for tool-call pairing and session size management.
// These run on every provider request and at compaction boundaries.
func BenchmarkContextManagement(b *testing.B) {
	b.Run("normalize_messages_small", func(b *testing.B) {
		msgs := buildTurns(10)
		for b.Loop() {
			_ = provider.NormalizeMessages(msgs)
		}
	})

	b.Run("normalize_messages_medium", func(b *testing.B) {
		msgs := buildTurns(50)
		for b.Loop() {
			_ = provider.NormalizeMessages(msgs)
		}
	})

	b.Run("normalize_messages_large", func(b *testing.B) {
		msgs := buildTurns(200)
		for b.Loop() {
			_ = provider.NormalizeMessages(msgs)
		}
	})

	b.Run("session_snapshot_large_history", func(b *testing.B) {
		s := agent.NewSession("system prompt")
		for i := 0; i < 500; i++ {
			s.Add(provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf("message %d with some content to simulate real usage patterns", i),
			})
			s.Add(provider.Message{
				Role:    provider.RoleAssistant,
				Content: fmt.Sprintf("response %d with generated content that would normally come from the model", i),
			})
		}
		b.ResetTimer()
		for b.Loop() {
			_ = s.Snapshot()
		}
	})

	b.Run("session_concurrent_add_snapshot", func(b *testing.B) {
		s := agent.NewSession("system prompt")
		for i := 0; i < 50; i++ {
			s.Add(provider.Message{Role: provider.RoleUser, Content: "msg"})
			s.Add(provider.Message{Role: provider.RoleAssistant, Content: "resp"})
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%10 == 0 {
					_ = s.Snapshot()
				} else {
					s.Add(provider.Message{
						Role:    provider.RoleUser,
						Content: fmt.Sprintf("concurrent msg %d", i),
					})
				}
				i++
			}
		})
	})

	b.Run("schemas_per_turn", func(b *testing.B) {
		r := tool.NewRegistry()
		for _, t := range mockTools(25) {
			r.Add(t)
		}
		b.ResetTimer()
		for b.Loop() {
			schemas := r.Schemas()
			// Simulate building a provider request
			_ = provider.Request{
				Messages:    []provider.Message{{Role: provider.RoleUser, Content: "test"}},
				Tools:       schemas,
				Temperature: 0.7,
			}
		}
	})
}

// --- helpers ---

// mockTool is a minimal tool implementation for benchmarks.
type mockTool struct {
	name string
}

func (t *mockTool) Name() string             { return t.name }
func (t *mockTool) Description() string      { return "mock tool for benchmarks" }
func (t *mockTool) Schema() json.RawMessage  { return json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}}}`) }
func (t *mockTool) ReadOnly() bool           { return true }
func (t *mockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "ok", nil
}

// noopTool is a zero-overhead tool for benchmarking Execute dispatch.
type noopTool struct{}

func (t *noopTool) Name() string             { return "noop" }
func (t *noopTool) Description() string      { return "no-op" }
func (t *noopTool) Schema() json.RawMessage  { return json.RawMessage(`{"type":"object"}`) }
func (t *noopTool) ReadOnly() bool           { return true }
func (t *noopTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "", nil
}

func mockTools(n int) []tool.Tool {
	out := make([]tool.Tool, n)
	for i := 0; i < n; i++ {
		out[i] = &mockTool{name: fmt.Sprintf("tool-%02d", i)}
	}
	return out
}

// buildTurns generates a well-formed conversation history with the given
// number of user/assistant turn pairs.
func buildTurns(n int) []provider.Message {
	msgs := make([]provider.Message, 0, n*2)
	for i := 0; i < n; i++ {
		msgs = append(msgs, provider.Message{
			Role:    provider.RoleUser,
			Content: fmt.Sprintf("user turn %d with realistic content length", i),
		})
		msgs = append(msgs, provider.Message{
			Role:    provider.RoleAssistant,
			Content: fmt.Sprintf("assistant turn %d with generated response content", i),
		})
	}
	return msgs
}
