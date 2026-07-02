package cli

import (
	"os"
	"strings"
	"testing"

	"mimo-reasonix/internal/config"
)

// TestModelRefsFromConfig verifies the /model picker enumerates configured
// provider/model refs (built-in defaults when no reasonix.toml is present), and
// only those whose provider API key is set.
func TestModelRefsFromConfig(t *testing.T) {
	isolateUserConfig(t) // no reasonix.toml -> built-in default providers
	// Only DeepSeek keyed → MiMo refs must be filtered out.
	if _, err := config.SetCredential("DEEPSEEK_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}
	t.Setenv("MIMO_API_KEY", "")
	refs := modelRefs()
	if len(refs) == 0 {
		t.Fatal("expected default provider/model refs, got none")
	}
	for _, r := range refs {
		if !strings.Contains(r, "/") {
			t.Errorf("ref %q should be provider/model", r)
		}
		if strings.HasPrefix(r, "mimo") {
			t.Errorf("ref %q from a provider without an API key should be filtered out", r)
		}
	}
}

// TestModelRefsSkipsUnconfigured verifies that with no provider keys set, the
// picker offers nothing rather than listing models the user can't select.
func TestModelRefsSkipsUnconfigured(t *testing.T) {
	isolateUserConfig(t)
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("MIMO_API_KEY", "")
	if refs := modelRefs(); len(refs) != 0 {
		t.Errorf("no keys set → no refs, got %v", refs)
	}
}

// TestModelArgCompletion verifies "/model " completes to the configured refs
// through the shared completion path.
func TestModelArgCompletion(t *testing.T) {
	isolateUserConfig(t)
	if _, err := config.SetCredential("DEEPSEEK_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}
	m := newTestChatTUI()
	items, _, ok := m.slashArgItems("/model ")
	if !ok || len(items) == 0 {
		t.Fatalf("/model arg completion should offer refs, ok=%v n=%d", ok, len(items))
	}
}

// TestPersistModelWritesDefaultModel verifies that calling persistModel with a
// "provider/model" ref writes default_model = "<ref>" to the user config file
// in TOML form. This is the fix for the "default model resets on every launch"
// regression: previously /model only mutated the in-memory controller and the
// next startup read the global default.
func TestPersistModelWritesDefaultModel(t *testing.T) {
	isolateUserConfig(t)
	if _, err := config.SetCredential("DEEPSEEK_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}
	t.Setenv("MIMO_API_KEY", "")

	m := newTestChatTUI()
	m.persistModel("deepseek-flash/deepseek-v4-flash")

	body, err := os.ReadFile(config.UserConfigPath())
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.Contains(string(body), `default_model = "deepseek-flash/deepseek-v4-flash"`) {
		t.Fatalf("saved config missing default_model ref:\n%s", body)
	}
}

// TestPersistModelRejectsUnknownRef verifies that an unresolvable ref is
// silently dropped (logged to slog, not pushed to the TUI notice channel)
// and never lands in the config file. Reason: surface a "persist failed"
// notice on the input box would make /model feel broken to users whose
// stored config doesn't list the exact model ref they picked; the in-
// memory switch still goes through.
func TestPersistModelRejectsUnknownRef(t *testing.T) {
	isolateUserConfig(t)
	if _, err := config.SetCredential("DEEPSEEK_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}

	m := newTestChatTUI()
	m.persistModel("ghost/never-existed")

	if _, err := os.Stat(config.UserConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("unknown ref must not create config file, stat err=%v", err)
	}
}

// TestModelRefsIncludesMiMoModels verifies that when the MiMo API key is set,
// the model picker includes MiMo models alongside other configured providers.
func TestModelRefsIncludesMiMoModels(t *testing.T) {
	isolateUserConfig(t)
	// Set MiMo API key
	if _, err := config.SetCredential("MIMO_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}
	// Clear DeepSeek key to isolate MiMo models
	t.Setenv("DEEPSEEK_API_KEY", "")

	refs := modelRefs()
	if len(refs) == 0 {
		t.Fatal("expected MiMo model refs, got none")
	}

	// Verify MiMo models are present (provider names are "xiaomi" and "xiaomi-pro")
	hasMiMo := false
	for _, r := range refs {
		if strings.HasPrefix(r, "xiaomi/") || strings.HasPrefix(r, "xiaomi-pro/") {
			hasMiMo = true
			break
		}
	}
	if !hasMiMo {
		t.Errorf("expected MiMo model refs in %v", refs)
	}
}

// TestModelRefsDefaultModelIsMiMo verifies that the default configuration
// includes MiMo as the primary model.
func TestModelRefsDefaultModelIsMiMo(t *testing.T) {
	isolateUserConfig(t)
	// Set MiMo API key to enable MiMo providers
	if _, err := config.SetCredential("MIMO_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}

	// Verify the default config has MiMo as default model (provider/model format)
	cfg := config.Default()
	if cfg.DefaultModel != "xiaomi/mimo-v2.5" {
		t.Errorf("Default().DefaultModel = %q, want xiaomi/mimo-v2.5", cfg.DefaultModel)
	}
}

// TestPersistModelMiMoRef verifies that persisting a MiMo model ref works correctly.
func TestPersistModelMiMoRef(t *testing.T) {
	isolateUserConfig(t)
	// Set MiMo API key
	if _, err := config.SetCredential("MIMO_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}

	m := newTestChatTUI()
	m.persistModel("xiaomi/mimo-v2.5")

	body, err := os.ReadFile(config.UserConfigPath())
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.Contains(string(body), `default_model = "xiaomi/mimo-v2.5"`) {
		t.Fatalf("saved config missing MiMo default_model ref:\n%s", body)
	}
}

// TestProviderNamesIncludesMiMo verifies that provider names include MiMo
// when the API key is configured.
func TestProviderNamesIncludesMiMo(t *testing.T) {
	isolateUserConfig(t)
	// Set MiMo API key
	if _, err := config.SetCredential("MIMO_API_KEY", "test-key"); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}
	t.Setenv("DEEPSEEK_API_KEY", "")

	names := providerNames()
	if len(names) == 0 {
		t.Fatal("expected provider names, got none")
	}

	hasMiMo := false
	for _, n := range names {
		if n == "xiaomi" || n == "xiaomi-pro" {
			hasMiMo = true
			break
		}
	}
	if !hasMiMo {
		t.Errorf("expected MiMo provider in %v", names)
	}
}
