package mimo

import "strings"

// MiMoModels lists all supported MiMo models
var MiMoModels = []string{
	"mimo-v2.5",
	"mimo-v2.5-pro",
	"mimo-v2-pro",
	"mimo-v2-flash",
	"mimo-v2-omni",
}

// MiMoPricing defines pricing in CNY per million tokens
type MiMoPricing struct {
	CachedInput float64
	Input       float64
	Output      float64
}

var DefaultPricing = MiMoPricing{
	CachedInput: 0.7,
	Input:       7.0,
	Output:      14.0,
}

// IsMiMoModel checks if a model name is a MiMo model.
// It accepts both bare model names (e.g. "mimo-v2.5") and prefixed names
// (e.g. "xiaomi/mimo-v2.5").
func IsMiMoModel(model string) bool {
	// Strip optional "xiaomi/" prefix
	if _, bare, ok := strings.Cut(model, "/"); ok {
		model = bare
	}
	for _, m := range MiMoModels {
		if m == model {
			return true
		}
	}
	return false
}
