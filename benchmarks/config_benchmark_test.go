package benchmarks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"

	"mimo-reasonix/internal/config"
	"mimo-reasonix/internal/provider"
)

// BenchmarkConfigLoading measures the cost of a full config assembly:
// defaults -> user TOML -> project TOML -> credential resolution.
// The benchmark uses a temp directory to avoid filesystem pollution.
func BenchmarkConfigLoading(b *testing.B) {
	b.Run("defaults_only", func(b *testing.B) {
		for b.Loop() {
			_ = config.Default()
		}
	})

	b.Run("from_toml_file", func(b *testing.B) {
		dir := b.TempDir()
		writeTestTOML(b, filepath.Join(dir, "mimo-reasonix.toml"), minimalTOML())
		b.ResetTimer()
		for b.Loop() {
			_, _ = config.LoadForRoot(dir)
		}
	})

	b.Run("from_large_toml", func(b *testing.B) {
		dir := b.TempDir()
		writeTestTOML(b, filepath.Join(dir, "mimo-reasonix.toml"), largeTOML())
		b.ResetTimer()
		for b.Loop() {
			_, _ = config.LoadForRoot(dir)
		}
	})
}

// BenchmarkConfigParsing measures raw TOML decode performance without
// the full config merge pipeline, isolating parser throughput.
func BenchmarkConfigParsing(b *testing.B) {
	benchmarks := []struct {
		name string
		toml string
	}{
		{"minimal", minimalTOML()},
		{"medium", mediumTOML()},
		{"large", largeTOML()},
		{"providers_heavy", providersHeavyTOML()},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				var cfg config.Config
				_, _ = toml.Decode(bm.toml, &cfg)
			}
		})
	}
}

// BenchmarkCredentialResolution measures the cost of resolving a credential
// key through the multi-source resolution chain (environment -> .env files).
// Uses b.Setenv for deterministic, side-effect-free resolution.
func BenchmarkCredentialResolution(b *testing.B) {
	b.Run("from_environment", func(b *testing.B) {
		b.Setenv("BENCH_CRED_KEY", "test-value-12345")
		for b.Loop() {
			_ = config.CredentialIsSet("BENCH_CRED_KEY")
		}
	})

	b.Run("not_set", func(b *testing.B) {
		os.Unsetenv("BENCH_CRED_NONEXISTENT")
		for b.Loop() {
			_ = config.CredentialIsSet("BENCH_CRED_NONEXISTENT")
		}
	})

	b.Run("resolve_credential", func(b *testing.B) {
		b.Setenv("BENCH_RESOLVE_KEY", "resolve-value")
		for b.Loop() {
			config.ResolveCredential("BENCH_RESOLVE_KEY")
		}
	})
}

// BenchmarkProviderEntryModelList measures ProviderEntry.ModelList() and
// HasModel() which are called on every model switch and validation pass.
func BenchmarkProviderEntryModelList(b *testing.B) {
	entry := config.ProviderEntry{
		Name:   "bench-provider",
		Kind:   "openai",
		Models: []string{"model-a", "model-b", "model-c", "model-d", "model-e"},
	}

	b.Run("ModelList", func(b *testing.B) {
		for b.Loop() {
			_ = entry.ModelList()
		}
	})

	b.Run("HasModel_hit", func(b *testing.B) {
		for b.Loop() {
			_ = entry.HasModel("model-c")
		}
	})

	b.Run("HasModel_miss", func(b *testing.B) {
		for b.Loop() {
			_ = entry.HasModel("nonexistent")
		}
	})

	b.Run("DefaultModel", func(b *testing.B) {
		for b.Loop() {
			_ = entry.DefaultModel()
		}
	})
}

// BenchmarkPriceForModel measures per-model price lookup, which is called
// for every usage report and turn summary.
func BenchmarkPriceForModel(b *testing.B) {
	entry := config.ProviderEntry{
		Name:  "bench-pricing",
		Kind:  "openai",
		Price: &provider.Pricing{CacheHit: 0.02, Input: 1.0, Output: 2.0, Currency: "¥"},
		Prices: map[string]*provider.Pricing{
			"fast-model": {CacheHit: 0.01, Input: 0.5, Output: 1.0, Currency: "¥"},
			"slow-model": {CacheHit: 0.05, Input: 5.0, Output: 10.0, Currency: "¥"},
		},
	}

	b.Run("per_model_hit", func(b *testing.B) {
		for b.Loop() {
			_ = entry.PriceForModel("fast-model")
		}
	})

	b.Run("fallback_to_global", func(b *testing.B) {
		for b.Loop() {
			_ = entry.PriceForModel("unknown-model")
		}
	})
}

// --- helpers ---

func writeTestTOML(b testing.TB, path, content string) {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("write %s: %v", path, err)
	}
}

func minimalTOML() string {
	return `
default_model = "xiaomi/mimo-v2.5"

[[providers]]
name = "xiaomi"
kind = "openai"
base_url = "https://token-plan-cn.xiaomimimo.com/v1"
model = "mimo-v2.5"
api_key_env = "MIMO_API_KEY"
`
}

func mediumTOML() string {
	return `
config_version = 4
default_model = "xiaomi/mimo-v2.5"
language = "en"

[agent]
max_steps = 25
temperature = 0.7
soft_compact_ratio = 0.5
compact_ratio = 0.8

[ui]
theme = "dark"
show_reasoning = true

[permissions]
mode = "ask"

[[providers]]
name = "xiaomi"
kind = "openai"
base_url = "https://token-plan-cn.xiaomimimo.com/v1"
model = "mimo-v2.5"
api_key_env = "MIMO_API_KEY"
context_window = 1048576

[[providers]]
name = "deepseek"
kind = "openai"
base_url = "https://api.deepseek.com"
model = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
context_window = 1000000
`
}

func largeTOML() string {
	toml := `
config_version = 4
default_model = "xiaomi/mimo-v2.5"
language = "en"
credentials_store = "auto"

[agent]
max_steps = 50
temperature = 0.7
soft_compact_ratio = 0.5
tool_result_snip_ratio = 0.6
compact_ratio = 0.8
compact_force_ratio = 0.9
auto_plan = "off"
reasoning_language = "auto"

[ui]
theme = "dark"
theme_style = "graphite"
show_reasoning = true
cursor_shape = "underline"

[permissions]
mode = "ask"
allow = ["bash(git *)"]
deny = ["write_file(*/.ssh/*)"]

[sandbox]
bash = "enforce"
network = true

[network]
proxy_mode = "auto"

[lsp]
enabled = true

[tools]
bash_timeout_seconds = 120
mcp_call_timeout_seconds = 300

[tools.search]
engine = "auto"

[tools.shell]
prefer = "auto"

[notifications]
enabled = false
turn_done = true

[bot]
enabled = false
max_steps = 25
debounce_ms = 1500
`

	for i := 0; i < 8; i++ {
		toml += `
[[providers]]
name = "provider-` + string(rune('a'+i)) + `"
kind = "openai"
base_url = "https://api.provider` + string(rune('a'+i)) + `.com/v1"
model = "model-` + string(rune('a'+i)) + `"
api_key_env = "PROVIDER_` + string(rune('A'+i)) + `_KEY"
context_window = 1000000
`
	}

	return toml
}

func providersHeavyTOML() string {
	toml := `
default_model = "xiaomi/mimo-v2.5"
`
	for i := 0; i < 20; i++ {
		name := string(rune('a' + i%26))
		toml += `
[[providers]]
name = "p-` + name + `"
kind = "openai"
base_url = "https://api.example.com/v1"
model = "model-` + name + `"
models = ["model-` + name + `-v1", "model-` + name + `-v2"]
api_key_env = "KEY_` + name + `"
context_window = 1000000

[providers.prices."model-` + name + `-v1"]
input = 1.0
output = 2.0
cache_hit = 0.1

[providers.prices."model-` + name + `-v2"]
input = 5.0
output = 10.0
cache_hit = 0.5
`
	}
	return toml
}
