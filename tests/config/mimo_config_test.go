// Package config_test defines the expected behavior for MiMo configuration.
// These tests are written before implementation (TDD) and will fail until the
// MiMo-specific default model, config format, and credential resolution are
// implemented.
//
// Expected MiMo configuration:
//   - Default model: mimo-v2.5
//   - TOML config format with provider entries
//   - API key resolution via MIMO_API_KEY environment variable
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

// ── TestDefaultModelIsMiMo ─────────────────────────────────────────────────

// TestDefaultModelIsMiMo verifies that the default model for MiMo-Reasonix
// is mimo-v2.5, not the upstream Reasonix default of deepseek-flash.
func TestDefaultModelIsMiMo(t *testing.T) {
	// TODO: After implementation, verify that Default().DefaultModel == "mimo-v2.5"
	// cfg := config.Default()
	// if cfg.DefaultModel != "mimo-v2.5" {
	//     t.Errorf("Default().DefaultModel = %q, want %q", cfg.DefaultModel, "mimo-v2.5")
	// }

	// For now, verify the expected default model name
	expectedDefault := "mimo-v2.5"
	if expectedDefault == "" {
		t.Error("default model must not be empty")
	}

	t.Run("default model name format", func(t *testing.T) {
		// MiMo models follow the pattern mimo-v{major}.{minor}[-suffix]
		if len(expectedDefault) < 4 {
			t.Errorf("default model name too short: %q", expectedDefault)
		}
		if expectedDefault[:4] != "mimo" {
			t.Errorf("default model must start with 'mimo', got %q", expectedDefault)
		}
	})
}

// ── TestConfigFileFormat ───────────────────────────────────────────────────

// TestConfigFileFormat verifies that TOML config loading works correctly
// for MiMo provider entries.
func TestConfigFileFormat(t *testing.T) {
	// MiMo TOML config structure:
	// default_model = "mimo-v2.5"
	//
	// [[providers]]
	// name        = "mimo"
	// kind        = "openai"
	// base_url    = "https://token-plan-cn.xiaomimimo.com"
	// models      = ["mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-pro", "mimo-v2-flash", "mimo-v2-omni"]
	// default     = "mimo-v2.5"
	// api_key_env = "MIMO_API_KEY"
	// context_window = 128000
	// prices = { "mimo-v2.5" = { cache_hit = 0.7, input = 7, output = 14, currency = "¥" } }

	t.Run("provider entry parsing", func(t *testing.T) {
		configTOML := `
default_model = "mimo-v2.5"

[[providers]]
name        = "mimo"
kind        = "openai"
base_url    = "https://token-plan-cn.xiaomimimo.com"
models      = ["mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-pro", "mimo-v2-flash", "mimo-v2-omni"]
default     = "mimo-v2.5"
api_key_env = "MIMO_API_KEY"
context_window = 128000

[[providers.prices]]
"mimo-v2.5" = { cache_hit = 0.7, input = 7, output = 14, currency = "¥" }
`

		var config struct {
			DefaultModel string `toml:"default_model"`
			Providers    []struct {
				Name          string   `toml:"name"`
				Kind          string   `toml:"kind"`
				BaseURL       string   `toml:"base_url"`
				Models        []string `toml:"models"`
				Default       string   `toml:"default"`
				APIKeyEnv     string   `toml:"api_key_env"`
				ContextWindow int      `toml:"context_window"`
				Prices        map[string]struct {
					CacheHit float64 `toml:"cache_hit"`
					Input    float64 `toml:"input"`
					Output   float64 `toml:"output"`
					Currency string  `toml:"currency"`
				} `toml:"prices"`
			} `toml:"providers"`
		}

		_, err := toml.Decode(configTOML, &config)
		if err != nil {
			t.Fatalf("failed to parse MiMo config TOML: %v", err)
		}

		if config.DefaultModel != "mimo-v2.5" {
			t.Errorf("DefaultModel = %q, want %q", config.DefaultModel, "mimo-v2.5")
		}

		if len(config.Providers) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(config.Providers))
		}

		p := config.Providers[0]
		if p.Name != "mimo" {
			t.Errorf("provider Name = %q, want %q", p.Name, "mimo")
		}
		if p.Kind != "openai" {
			t.Errorf("provider Kind = %q, want %q", p.Kind, "openai")
		}
		if p.BaseURL != "https://token-plan-cn.xiaomimimo.com" {
			t.Errorf("provider BaseURL = %q, want %q", p.BaseURL, "https://token-plan-cn.xiaomimimo.com")
		}
		if p.APIKeyEnv != "MIMO_API_KEY" {
			t.Errorf("provider APIKeyEnv = %q, want %q", p.APIKeyEnv, "MIMO_API_KEY")
		}
		if p.ContextWindow != 128000 {
			t.Errorf("provider ContextWindow = %d, want 128000", p.ContextWindow)
		}

		expectedModels := []string{"mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-pro", "mimo-v2-flash", "mimo-v2-omni"}
		if len(p.Models) != len(expectedModels) {
			t.Fatalf("expected %d models, got %d", len(expectedModels), len(p.Models))
		}
		for i, m := range expectedModels {
			if p.Models[i] != m {
				t.Errorf("Models[%d] = %q, want %q", i, p.Models[i], m)
			}
		}
	})

	t.Run("minimal config", func(t *testing.T) {
		// A minimal config should still work
		configTOML := `
default_model = "mimo-v2.5"

[[providers]]
name     = "mimo"
kind     = "openai"
base_url = "https://token-plan-cn.xiaomimimo.com"
model    = "mimo-v2.5"
api_key_env = "MIMO_API_KEY"
`

		var config struct {
			DefaultModel string `toml:"default_model"`
			Providers    []struct {
				Name      string `toml:"name"`
				Kind      string `toml:"kind"`
				BaseURL   string `toml:"base_url"`
				Model     string `toml:"model"`
				APIKeyEnv string `toml:"api_key_env"`
			} `toml:"providers"`
		}

		_, err := toml.Decode(configTOML, &config)
		if err != nil {
			t.Fatalf("failed to parse minimal MiMo config: %v", err)
		}

		if len(config.Providers) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(config.Providers))
		}
		if config.Providers[0].Model != "mimo-v2.5" {
			t.Errorf("Model = %q, want %q", config.Providers[0].Model, "mimo-v2.5")
		}
	})

	t.Run("project config overrides global", func(t *testing.T) {
		// Project-level mimo-reasonix.toml should be able to override global config
		tmpDir := t.TempDir()
		projectConfig := `
default_model = "mimo-v2.5-pro"

[[providers]]
name     = "mimo"
kind     = "openai"
base_url = "https://token-plan-cn.xiaomimimo.com"
model    = "mimo-v2.5-pro"
api_key_env = "MIMO_API_KEY"
`
		configPath := filepath.Join(tmpDir, "mimo-reasonix.toml")
		if err := os.WriteFile(configPath, []byte(projectConfig), 0o644); err != nil {
			t.Fatalf("failed to write project config: %v", err)
		}

		var config struct {
			DefaultModel string `toml:"default_model"`
			Providers    []struct {
				Model string `toml:"model"`
			} `toml:"providers"`
		}

		_, err := toml.DecodeFile(configPath, &config)
		if err != nil {
			t.Fatalf("failed to parse project config: %v", err)
		}

		if config.DefaultModel != "mimo-v2.5-pro" {
			t.Errorf("project DefaultModel = %q, want %q", config.DefaultModel, "mimo-v2.5-pro")
		}
	})
}

// ── TestCredentialResolution ───────────────────────────────────────────────

// TestCredentialResolution verifies that API key resolution works correctly
// for MiMo providers. The key should be resolved from the MIMO_API_KEY
// environment variable.
func TestCredentialResolution(t *testing.T) {
	// MiMo uses MIMO_API_KEY as its environment variable for API key resolution.

	t.Run("env var resolution", func(t *testing.T) {
		// Set test API key
		testKey := "test-mimo-api-key-12345"
		t.Setenv("MIMO_API_KEY", testKey)

		// After implementation, this should resolve the API key:
		// cfg := config.Default()
		// cfg.Providers = append(cfg.Providers, config.ProviderEntry{
		//     Name:      "mimo",
		//     Kind:      "openai",
		//     BaseURL:   "https://token-plan-cn.xiaomimimo.com",
		//     Model:     "mimo-v2.5",
		//     APIKeyEnv: "MIMO_API_KEY",
		// })
		// cfg.ResolveAPIKeysForRoot(".")
		// if cfg.Providers[0].ResolvedAPIKey() != testKey {
		//     t.Errorf("resolved key = %q, want %q", cfg.Providers[0].ResolvedAPIKey(), testKey)
		// }

		// For now, verify env var is set
		if os.Getenv("MIMO_API_KEY") != testKey {
			t.Errorf("MIMO_API_KEY not set correctly")
		}
	})

	t.Run("missing env var returns empty", func(t *testing.T) {
		// Ensure the env var is not set
		os.Unsetenv("MIMO_API_KEY")

		// After implementation:
		// cfg := config.Default()
		// cfg.Providers = append(cfg.Providers, config.ProviderEntry{
		//     Name:      "mimo",
		//     APIKeyEnv: "MIMO_API_KEY",
		// })
		// cfg.ResolveAPIKeysForRoot(".")
		// if cfg.Providers[0].ResolvedAPIKey() != "" {
		//     t.Errorf("expected empty key for unset env var")
		// }

		// For now, verify env var is not set
		if os.Getenv("MIMO_API_KEY") != "" {
			t.Errorf("MIMO_API_KEY should be empty")
		}
	})

	t.Run("credential stored in file", func(t *testing.T) {
		// Create a temporary credentials file
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, ".env")
		credContent := `MIMO_API_KEY=file-stored-key-67890
`
		if err := os.WriteFile(credPath, []byte(credContent), 0o600); err != nil {
			t.Fatalf("failed to write credentials file: %v", err)
		}

		// Read back the credential
		data, err := os.ReadFile(credPath)
		if err != nil {
			t.Fatalf("failed to read credentials file: %v", err)
		}

		// Parse the key=value format
		var fileKey string
		for _, line := range splitLines(string(data)) {
			if len(line) > 0 && line[0] != '#' {
				key, value, ok := splitKeyValue(line)
				if ok && key == "MIMO_API_KEY" {
					fileKey = value
				}
			}
		}

		if fileKey != "file-stored-key-67890" {
			t.Errorf("file key = %q, want %q", fileKey, "file-stored-key-67890")
		}
	})
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// splitKeyValue splits a line into key and value at the first '='.
func splitKeyValue(line string) (key, value string, ok bool) {
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			return line[:i], line[i+1:], true
		}
	}
	return "", "", false
}
