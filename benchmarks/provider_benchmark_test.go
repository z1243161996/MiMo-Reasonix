package benchmarks

import (
	"encoding/json"
	"fmt"
	"testing"

	_ "mimo-reasonix/internal/provider/anthropic"
	_ "mimo-reasonix/internal/provider/openai"

	"mimo-reasonix/internal/provider"
)

// BenchmarkProviderInitialization measures the cost of constructing a provider
// instance from a resolved config. This covers factory lookup, struct assembly,
// and HTTP client creation — the cold-start path every new session pays.
func BenchmarkProviderInitialization(b *testing.B) {
	benchmarks := []struct {
		name string
		cfg  provider.Config
	}{
		{
			name: "openai",
			cfg: provider.Config{
				Name:    "bench-openai",
				BaseURL: "https://api.example.com/v1",
				Model:   "test-model",
				APIKey:  "test-key",
				Extra: map[string]any{
					"api_key_env": "BENCH_TEST_KEY",
				},
			},
		},
		{
			name: "anthropic",
			cfg: provider.Config{
				Name:    "bench-anthropic",
				BaseURL: "https://api.anthropic.com",
				Model:   "claude-test",
				APIKey:  "test-key",
				Extra: map[string]any{
					"api_key_env": "BENCH_TEST_KEY",
				},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				_, err := provider.New(bm.name, bm.cfg)
				if err != nil {
					b.Fatalf("provider.New(%q): %v", bm.name, err)
				}
			}
		})
	}
}

// BenchmarkProviderInitializationParallel measures concurrent provider creation
// to verify the registry's thread safety and contention characteristics.
func BenchmarkProviderInitializationParallel(b *testing.B) {
	cfg := provider.Config{
		Name:    "bench-parallel",
		BaseURL: "https://api.example.com/v1",
		Model:   "test-model",
		APIKey:  "test-key",
		Extra: map[string]any{
			"api_key_env": "BENCH_TEST_KEY",
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.New("openai", cfg)
			if err != nil {
				b.Errorf("provider.New: %v", err)
			}
		}
	})
}

// BenchmarkModelResolution measures the cost of resolving a model reference
// string (e.g. "xiaomi/mimo-v2.5", "deepseek", "mimo-v2.5-pro") to a
// fully-resolved ProviderEntry. This is the hot path on every new session
// and every model switch in the desktop UI.
func BenchmarkModelResolution(b *testing.B) {
	cfg := providerConfig()

	refs := []struct {
		name string
		ref  string
	}{
		{"provider_slash_model", "xiaomi/mimo-v2.5"},
		{"provider_name", "deepseek-flash"},
		{"bare_model", "mimo-v2.5-pro"},
		{"provider_slash_model_deepseek", "deepseek-flash/deepseek-v4-flash"},
		{"unresolved", "nonexistent/model"},
	}

	for _, bm := range refs {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				cfg.ResolveModel(bm.ref)
			}
		})
	}
}

// BenchmarkPricingCalculation measures the cost of computing spend estimates
// from token usage records. This runs on every streamed chunk and every
// turn summary, so sub-microsecond performance matters at scale.
func BenchmarkPricingCalculation(b *testing.B) {
	pricing := &provider.Pricing{
		CacheHit: 0.02,
		Input:    1.0,
		Output:   2.0,
		Currency: "¥",
	}

	benchmarks := []struct {
		name  string
		usage provider.Usage
	}{
		{
			name: "small_turn",
			usage: provider.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
				TotalTokens:      1500,
			},
		},
		{
			name: "large_turn_with_cache",
			usage: provider.Usage{
				PromptTokens:     100000,
				CompletionTokens: 5000,
				TotalTokens:      105000,
				CacheHitTokens:   80000,
				CacheMissTokens:  20000,
			},
		},
		{
			name: "reasoning_heavy",
			usage: provider.Usage{
				PromptTokens:     50000,
				CompletionTokens: 20000,
				TotalTokens:      70000,
				ReasoningTokens:  15000,
				CacheHitTokens:   30000,
				CacheMissTokens:  20000,
			},
		},
		{
			name:  "zero_usage",
			usage: provider.Usage{},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				_ = pricing.Cost(&bm.usage)
			}
		})
	}
}

// BenchmarkNormalizeMessages measures the cost of the tool-call pairing
// repair that runs on every provider request. Healthy histories should
// take the fast path (zero allocation); malformed ones exercise the
// repair logic.
func BenchmarkNormalizeMessages(b *testing.B) {
	b.Run("well_formed", func(b *testing.B) {
		msgs := makeWellFormedMessages(20)
		for b.Loop() {
			_ = provider.NormalizeMessages(msgs)
		}
	})

	b.Run("with_orphan_tools", func(b *testing.B) {
		msgs := makeOrphanToolMessages(20)
		for b.Loop() {
			_ = provider.NormalizeMessages(msgs)
		}
	})

	b.Run("with_truncated_args", func(b *testing.B) {
		msgs := makeTruncatedArgMessages(10)
		for b.Loop() {
			_ = provider.NormalizeMessages(msgs)
		}
	})

	b.Run("empty", func(b *testing.B) {
		for b.Loop() {
			_ = provider.NormalizeMessages(nil)
		}
	})
}

// BenchmarkCanonicalizeSchema measures the cost of schema canonicalization
// that runs once per tool at registration and once per turn for the
// full schema list.
func BenchmarkCanonicalizeSchema(b *testing.B) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "shell command"},
			"timeout": {"type": "integer", "description": "seconds"},
			"env": {"type": "object", "description": "environment"}
		},
		"required": ["command"]
	}`)

	b.Run("simple", func(b *testing.B) {
		for b.Loop() {
			_ = provider.CanonicalizeSchema(schema)
		}
	})

	b.Run("large_schema", func(b *testing.B) {
		large := makeLargeSchema(50)
		for b.Loop() {
			_ = provider.CanonicalizeSchema(large)
		}
	})

	b.Run("empty", func(b *testing.B) {
		for b.Loop() {
			_ = provider.CanonicalizeSchema(nil)
		}
	})
}

// --- helpers ---

// providerConfig returns a minimal config.Config with default providers
// for model resolution benchmarks.
func providerConfig() *providerConfigForBench {
	return &providerConfigForBench{
		DefaultModel: "xiaomi/mimo-v2.5",
		Providers: []providerEntryForBench{
			{Name: "xiaomi", Kind: "openai", BaseURL: "https://token-plan-cn.xiaomimimo.com/v1", Model: "mimo-v2.5", Models: []string{"mimo-v2.5", "mimo-v2.5-pro"}},
			{Name: "deepseek-flash", Kind: "openai", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", Models: []string{"deepseek-v4-flash"}},
		},
	}
}

// We duplicate the config types here to avoid importing internal/config
// (which has heavy init-time side effects). The benchmark tests the
// resolution algorithm, not the full config merge pipeline.

type providerConfigForBench struct {
	DefaultModel string
	Providers    []providerEntryForBench
}

type providerEntryForBench struct {
	Name    string
	Kind    string
	BaseURL string
	Model   string
	Models  []string
}

func (c *providerConfigForBench) ResolveModel(ref string) (string, bool) {
	if ref == "" {
		return "", false
	}
	// "provider/model"
	if prov, model, ok := cut(ref, "/"); ok {
		for i := range c.Providers {
			p := &c.Providers[i]
			if p.Name == prov && p.HasModel(model) {
				return p.Name + "/" + model, true
			}
		}
	}
	// provider name -> default model
	for i := range c.Providers {
		p := &c.Providers[i]
		if p.Name == ref {
			return p.Name + "/" + p.DefaultModel(), true
		}
	}
	// bare model -> first provider that has it
	for i := range c.Providers {
		p := &c.Providers[i]
		if p.HasModel(ref) {
			return p.Name + "/" + ref, true
		}
	}
	return "", false
}

func (e *providerEntryForBench) HasModel(m string) bool {
	for _, x := range e.ModelList() {
		if x == m {
			return true
		}
	}
	return false
}

func (e *providerEntryForBench) ModelList() []string {
	if len(e.Models) > 0 {
		return e.Models
	}
	if e.Model != "" {
		return []string{e.Model}
	}
	return nil
}

func (e *providerEntryForBench) DefaultModel() string {
	if l := e.ModelList(); len(l) > 0 {
		return l[0]
	}
	return ""
}

func cut(s, sep string) (before, after string, found bool) {
	if i := indexOf(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func indexOf(s, sep string) int {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

// makeWellFormedMessages builds a history of alternating user/assistant
// turns with proper tool-call pairing (no repairs needed).
func makeWellFormedMessages(n int) []provider.Message {
	msgs := make([]provider.Message, 0, n)
	for i := 0; i < n; i += 2 {
		msgs = append(msgs, provider.Message{
			Role:    provider.RoleUser,
			Content: fmt.Sprintf("user message %d", i),
		})
		msgs = append(msgs, provider.Message{
			Role:    provider.RoleAssistant,
			Content: fmt.Sprintf("assistant message %d", i+1),
		})
	}
	return msgs
}

// makeOrphanToolMessages builds a history with orphan tool messages
// (no preceding assistant tool_calls).
func makeOrphanToolMessages(n int) []provider.Message {
	msgs := make([]provider.Message, 0, n)
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			msgs = append(msgs, provider.Message{
				Role:    provider.RoleTool,
				Content: "orphan tool result",
			})
		} else {
			msgs = append(msgs, provider.Message{
				Role:    provider.RoleUser,
				Content: fmt.Sprintf("message %d", i),
			})
		}
	}
	return msgs
}

// makeTruncatedArgMessages builds a history with assistant messages whose
// tool_calls have truncated (invalid JSON) arguments.
func makeTruncatedArgMessages(n int) []provider.Message {
	msgs := make([]provider.Message, 0, n*3)
	for i := 0; i < n; i++ {
		callID := fmt.Sprintf("call_%d", i)
		msgs = append(msgs, provider.Message{
			Role:    provider.RoleUser,
			Content: fmt.Sprintf("do something %d", i),
		})
		msgs = append(msgs, provider.Message{
			Role:    provider.RoleAssistant,
			Content: "",
			ToolCalls: []provider.ToolCall{
				{ID: callID, Name: "bash", Arguments: `{"command": "echo `}, // truncated JSON
			},
		})
		msgs = append(msgs, provider.Message{
			Role:       provider.RoleTool,
			Content:    "output",
			ToolCallID: callID,
			Name:       "bash",
		})
	}
	return msgs
}

// makeLargeSchema builds a JSON schema with many properties for
// canonicalization benchmarks.
func makeLargeSchema(n int) json.RawMessage {
	props := "{"
	for i := 0; i < n; i++ {
		if i > 0 {
			props += ","
		}
		props += fmt.Sprintf(`"param_%d": {"type": "string", "description": "parameter %d"}`, i, i)
	}
	props += "}"
	return json.RawMessage(fmt.Sprintf(`{"type": "object", "properties": %s}`, props))
}
