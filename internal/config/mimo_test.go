package config

import (
	"testing"

	"mimo-reasonix/internal/provider"
)

func TestMiMoV25Price(t *testing.T) {
	price := mimoV25Price()
	if price == nil {
		t.Fatal("mimoV25Price() returned nil")
	}
	if price.CacheHit != 0.02 {
		t.Errorf("mimoV25Price().CacheHit = %v, want 0.02", price.CacheHit)
	}
	if price.Input != 1 {
		t.Errorf("mimoV25Price().Input = %v, want 1", price.Input)
	}
	if price.Output != 2 {
		t.Errorf("mimoV25Price().Output = %v, want 2", price.Output)
	}
	if price.Currency != "¥" {
		t.Errorf("mimoV25Price().Currency = %q, want ¥", price.Currency)
	}
}

func TestMiMoV25ProPrice(t *testing.T) {
	price := mimoV25ProPrice()
	if price == nil {
		t.Fatal("mimoV25ProPrice() returned nil")
	}
	if price.CacheHit != 0.025 {
		t.Errorf("mimoV25ProPrice().CacheHit = %v, want 0.025", price.CacheHit)
	}
	if price.Input != 3 {
		t.Errorf("mimoV25ProPrice().Input = %v, want 3", price.Input)
	}
	if price.Output != 6 {
		t.Errorf("mimoV25ProPrice().Output = %v, want 6", price.Output)
	}
	if price.Currency != "¥" {
		t.Errorf("mimoV25ProPrice().Currency = %q, want ¥", price.Currency)
	}
}

func TestMiMoV2FlashPrice(t *testing.T) {
	price := mimoV2FlashPrice()
	if price == nil {
		t.Fatal("mimoV2FlashPrice() returned nil")
	}
	if price.CacheHit != 0.07 {
		t.Errorf("mimoV2FlashPrice().CacheHit = %v, want 0.07", price.CacheHit)
	}
	if price.Input != 0.70 {
		t.Errorf("mimoV2FlashPrice().Input = %v, want 0.70", price.Input)
	}
	if price.Output != 2.10 {
		t.Errorf("mimoV2FlashPrice().Output = %v, want 2.10", price.Output)
	}
	if price.Currency != "¥" {
		t.Errorf("mimoV2FlashPrice().Currency = %q, want ¥", price.Currency)
	}
}

func TestMiMoDomesticPrices(t *testing.T) {
	tests := []struct {
		name    string
		models  []string
		wantLen int
	}{
		{
			name:    "all mimo models",
			models:  []string{"mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-flash", "mimo-v2-omni", "mimo-v2-pro"},
			wantLen: 5,
		},
		{
			name:    "single mimo model",
			models:  []string{"mimo-v2.5"},
			wantLen: 1,
		},
		{
			name:    "mixed mimo and non-mimo",
			models:  []string{"mimo-v2.5", "gpt-4o", "claude-3.5-sonnet"},
			wantLen: 1,
		},
		{
			name:    "empty list",
			models:  []string{},
			wantLen: 0,
		},
		{
			name:    "unknown model",
			models:  []string{"unknown-model"},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prices := mimoDomesticPrices(tt.models)
			if len(prices) != tt.wantLen {
				t.Errorf("mimoDomesticPrices(%v) returned %d prices, want %d", tt.models, len(prices), tt.wantLen)
			}
		})
	}
}

func TestMiMoDomesticPricesPricingValues(t *testing.T) {
	prices := mimoDomesticPrices([]string{"mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-flash"})

	// Check mimo-v2.5 pricing
	if p := prices["mimo-v2.5"]; p == nil {
		t.Error("mimo-v2.5 price not found")
	} else {
		if p.Input != 1 || p.Output != 2 {
			t.Errorf("mimo-v2.5 pricing = {Input: %v, Output: %v}, want {1, 2}", p.Input, p.Output)
		}
	}

	// Check mimo-v2.5-pro pricing
	if p := prices["mimo-v2.5-pro"]; p == nil {
		t.Error("mimo-v2.5-pro price not found")
	} else {
		if p.Input != 3 || p.Output != 6 {
			t.Errorf("mimo-v2.5-pro pricing = {Input: %v, Output: %v}, want {3, 6}", p.Input, p.Output)
		}
	}

	// Check mimo-v2-flash pricing
	if p := prices["mimo-v2-flash"]; p == nil {
		t.Error("mimo-v2-flash price not found")
	} else {
		if p.Input != 0.70 || p.Output != 2.10 {
			t.Errorf("mimo-v2-flash pricing = {Input: %v, Output: %v}, want {0.70, 2.10}", p.Input, p.Output)
		}
	}
}

func TestDefaultMiMoProviders(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// Check that MiMo providers are configured (provider names are "xiaomi" and "xiaomi-pro")
	var mimoProviders []ProviderEntry
	for _, p := range cfg.Providers {
		if p.Name == "xiaomi" || p.Name == "xiaomi-pro" {
			mimoProviders = append(mimoProviders, p)
		}
	}

	if len(mimoProviders) != 2 {
		t.Fatalf("expected 2 MiMo providers, got %d", len(mimoProviders))
	}

	// Check xiaomi provider
	for _, p := range mimoProviders {
		if p.Name == "xiaomi" {
			if p.Kind != "openai" {
				t.Errorf("xiaomi provider kind = %q, want openai", p.Kind)
			}
			if p.BaseURL != "https://token-plan-cn.xiaomimimo.com/v1" {
				t.Errorf("xiaomi provider baseURL = %q, want https://token-plan-cn.xiaomimimo.com/v1", p.BaseURL)
			}
			if p.Model != "mimo-v2.5" {
				t.Errorf("xiaomi provider model = %q, want mimo-v2.5", p.Model)
			}
			if p.APIKeyEnv != "MIMO_API_KEY" {
				t.Errorf("xiaomi provider apiKeyEnv = %q, want MIMO_API_KEY", p.APIKeyEnv)
			}
			if p.ContextWindow != 1_048_576 {
				t.Errorf("xiaomi provider contextWindow = %v, want 1048576", p.ContextWindow)
			}
			if !p.NoProxy {
				t.Error("xiaomi provider should have NoProxy=true")
			}
		}
		if p.Name == "xiaomi-pro" {
			if p.Model != "mimo-v2.5-pro" {
				t.Errorf("xiaomi-pro provider model = %q, want mimo-v2.5-pro", p.Model)
			}
		}
	}
}

func TestDefaultModelIsMiMo(t *testing.T) {
	cfg := Default()
	if cfg.DefaultModel != "xiaomi/mimo-v2.5" {
		t.Errorf("Default().DefaultModel = %q, want xiaomi/mimo-v2.5", cfg.DefaultModel)
	}
}

func TestClonePricing(t *testing.T) {
	original := &provider.Pricing{
		CacheHit: 0.02,
		Input:    1,
		Output:   2,
		Currency: "¥",
	}
	cloned := clonePricing(original)
	if cloned == nil {
		t.Fatal("clonePricing() returned nil")
	}
	if cloned == original {
		t.Error("clonePricing() returned the same pointer")
	}
	if cloned.CacheHit != original.CacheHit || cloned.Input != original.Input || cloned.Output != original.Output || cloned.Currency != original.Currency {
		t.Errorf("clonePricing() = %+v, want %+v", cloned, original)
	}
}

func TestClonePricingNil(t *testing.T) {
	if got := clonePricing(nil); got != nil {
		t.Errorf("clonePricing(nil) = %+v, want nil", got)
	}
}
