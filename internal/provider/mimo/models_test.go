package mimo

import (
	"testing"
)

func TestIsMiMoModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{
			name:  "valid mimo-v2.5",
			model: "mimo-v2.5",
			want:  true,
		},
		{
			name:  "valid mimo-v2.5-pro",
			model: "mimo-v2.5-pro",
			want:  true,
		},
		{
			name:  "valid mimo-v2-pro",
			model: "mimo-v2-pro",
			want:  true,
		},
		{
			name:  "valid mimo-v2-flash",
			model: "mimo-v2-flash",
			want:  true,
		},
		{
			name:  "valid mimo-v2-omni",
			model: "mimo-v2-omni",
			want:  true,
		},
		{
			name:  "invalid empty string",
			model: "",
			want:  false,
		},
		{
			name:  "invalid gpt model",
			model: "gpt-4o",
			want:  false,
		},
		{
			name:  "invalid claude model",
			model: "claude-3.5-sonnet",
			want:  false,
		},
		{
			name:  "invalid case sensitive",
			model: "MIMO-V2.5",
			want:  false,
		},
		{
			name:  "invalid partial match",
			model: "mimo-v2",
			want:  false,
		},
		{
			name:  "invalid mimo-v2.5-asr (audio model)",
			model: "mimo-v2.5-asr",
			want:  false,
		},
		{
			name:  "invalid mimo-v2.5-tts (tts model)",
			model: "mimo-v2.5-tts",
			want:  false,
		},
		{
			name:  "valid xiaomi/mimo-v2.5 (prefixed)",
			model: "xiaomi/mimo-v2.5",
			want:  true,
		},
		{
			name:  "valid xiaomi/mimo-v2.5-pro (prefixed)",
			model: "xiaomi/mimo-v2.5-pro",
			want:  true,
		},
		{
			name:  "invalid deepseek model",
			model: "deepseek-v4-flash",
			want:  false,
		},
		{
			name:  "invalid qwen model",
			model: "qwen-max",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMiMoModel(tt.model); got != tt.want {
				t.Errorf("IsMiMoModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestMiMoModelsList(t *testing.T) {
	// Verify the MiMoModels list contains expected models
	expectedModels := map[string]bool{
		"mimo-v2.5":     true,
		"mimo-v2.5-pro": true,
		"mimo-v2-pro":   true,
		"mimo-v2-flash": true,
		"mimo-v2-omni":  true,
	}

	if len(MiMoModels) != len(expectedModels) {
		t.Errorf("MiMoModels has %d entries, want %d", len(MiMoModels), len(expectedModels))
	}

	for _, model := range MiMoModels {
		if !expectedModels[model] {
			t.Errorf("MiMoModels contains unexpected model: %q", model)
		}
	}
}

func TestDefaultPricing(t *testing.T) {
	// Verify default pricing values
	if DefaultPricing.CachedInput != 0.7 {
		t.Errorf("DefaultPricing.CachedInput = %v, want 0.7", DefaultPricing.CachedInput)
	}
	if DefaultPricing.Input != 7.0 {
		t.Errorf("DefaultPricing.Input = %v, want 7.0", DefaultPricing.Input)
	}
	if DefaultPricing.Output != 14.0 {
		t.Errorf("DefaultPricing.Output = %v, want 14.0", DefaultPricing.Output)
	}
}

func TestMiMoPricingStructure(t *testing.T) {
	// Test that MiMoPricing struct works correctly
	pricing := MiMoPricing{
		CachedInput: 1.0,
		Input:       2.0,
		Output:      3.0,
	}

	if pricing.CachedInput != 1.0 {
		t.Errorf("MiMoPricing.CachedInput = %v, want 1.0", pricing.CachedInput)
	}
	if pricing.Input != 2.0 {
		t.Errorf("MiMoPricing.Input = %v, want 2.0", pricing.Input)
	}
	if pricing.Output != 3.0 {
		t.Errorf("MiMoPricing.Output = %v, want 3.0", pricing.Output)
	}
}
