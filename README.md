# MiMo-Reasonix

MiMo-Reasonix is a Go rewrite of DeepSeek-Reasonix, adapted for MiMo model branding and configuration. It provides a CLI-based coding agent powered by MiMo models.

## Installation

```bash
go install mimo-reasonix@latest
```

Or build from source:

```bash
git clone https://github.com/xiaomi/mimo-reasonix.git
cd mimo-reasonix
go build -o mimo-reasonix .
```

## Quick Start

```bash
# Set your API key
export MIMO_API_KEY="your-api-key"

# Run interactively
mimo-reasonix

# Run a specific task
mimo-reasonix run "explain this codebase"

# Start an HTTP server
mimo-reasonix serve
```

## Configuration

MiMo-Reasonix looks for configuration in this order:
1. Command-line flags
2. `./mimo-reasonix.toml` (project-local)
3. `~/.mimo-reasonix/config.toml` (user-global)
4. Built-in defaults

### Default Model

The default model is `mimo-v2.5`, which is MiMo's flagship reasoning model.

### MiMo Model Pricing (per 1M tokens)

| Model | Input | Output | Cached Input |
|-------|-------|--------|--------------|
| mimo-v2.5 | 7.0 CNY | 14.0 CNY | 0.7 CNY |
| mimo-v2.5-pro | 3.0 CNY | 6.0 CNY | 0.025 CNY |
| mimo-v2-flash | 0.70 CNY | 2.10 CNY | 0.07 CNY |

### Configuration Files

- **`mimo-reasonix.example.toml`** — Full annotated configuration with all options (this file is tracked in git)
- **`mimo-reasonix.toml`** — Your project-local configuration (created from the example, git-ignored by default)

To set up your configuration:

```bash
# Copy the example to create your local config
cp mimo-reasonix.example.toml mimo-reasonix.toml

# Edit your local config
vim mimo-reasonix.toml
```

**Minimal configuration example:**

```toml
default_model = "mimo-v2.5"

[[providers]]
name        = "mimo"
kind        = "openai"
base_url    = "https://token-plan-cn.xiaomimimo.com/v1"
model       = "mimo-v2.5"
api_key_env = "MIMO_API_KEY"
```

## Subcommands

- `mimo-reasonix` — Start an interactive chat session
- `mimo-reasonix run <task>` — Execute a task non-interactively
- `mimo-reasonix serve` — Start an HTTP+SSE server
- `mimo-reasonix setup` — Run the configuration wizard
- `mimo-reasonix config` — Manage configuration settings
- `mimo-reasonix init` — Initialize project memory
- `mimo-reasonix version` — Show version information

## Environment Variables

- `MIMO_API_KEY` — API key for MiMo models
- `DEEPSEEK_API_KEY` — API key for DeepSeek models (also supported)
- `ANTHROPIC_API_KEY` — API key for Claude models (also supported)
- `REASONIX_HOME` — Override the default config directory (`~/.mimo-reasonix`)

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage report
go test ./... -coverprofile=coverage.out

# Run tests for a specific package
go test ./internal/provider/mimo/...
go test ./internal/config/...
go test ./internal/cli/...

# View coverage report
go tool cover -html=coverage.out
```

### Test Coverage

Current test coverage by package:

| Package | Coverage |
|---------|----------|
| internal/store | 100.0% |
| internal/shellsafe | 93.8% |
| internal/permission | 93.4% |
| internal/billing | 94.9% |
| internal/provider/mimo | 83.3% |
| internal/provider/anthropic | 89.2% |
| internal/provider/openai | 85.4% |
| internal/provider | 84.1% |
| internal/command | 89.8% |
| internal/planmode | 89.3% |
| internal/diff | 98.4% |
| internal/frontmatter | 96.6% |
| internal/config | 79.5% |
| internal/cli | 56.9% |

### Writing Tests

Tests follow standard Go patterns:
- Table-driven tests for multiple input scenarios
- `t.TempDir()` for temporary test directories
- `t.Setenv()` for environment variable isolation
- Helper functions like `isolateUserConfig()` for test setup

Example test structure:
```go
func TestIsMiMoModel(t *testing.T) {
    tests := []struct {
        name  string
        model string
        want  bool
    }{
        {"valid mimo-v2.5", "mimo-v2.5", true},
        {"invalid empty string", "", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := IsMiMoModel(tt.model); got != tt.want {
                t.Errorf("IsMiMoModel(%q) = %v, want %v", tt.model, got, tt.want)
            }
        })
    }
}
```

## License

See LICENSE file for details.
