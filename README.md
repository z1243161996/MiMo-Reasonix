# MiMo-Reasonix

[![CI](https://github.com/z1243161996/MiMo-Reasonix/actions/workflows/ci.yml/badge.svg)](https://github.com/z1243161996/MiMo-Reasonix/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/z1243161996/MiMo-Reasonix)](https://goreportcard.com/report/github.com/z1243161996/MiMo-Reasonix)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**A config- and plugin-driven coding agent CLI powered by Xiaomi MiMo models.**

MiMo-Reasonix is a Go rewrite of [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix), adapted for Xiaomi MiMo model branding and configuration. It provides a high-performance CLI-based coding agent with support for multiple AI models, tool calling, and plugin systems.

## ✨ Features

- 🚀 **High Performance**: Static Go binary (~30MB), zero dependencies
- 🤖 **Multi-Model Support**: MiMo, DeepSeek, Anthropic Claude, and OpenAI-compatible providers
- 🔧 **Tool Calling**: Bash, file operations, web search, code analysis, and more
- 🧠 **Context Management**: Intelligent token counting, history folding, and prefix caching
- 🔌 **Plugin System**: MCP (Model Context Protocol) support for extending functionality
- 💻 **Desktop App**: Wails-based GUI (coming soon)
- 🌐 **HTTP/SSE Server**: Serve as an API endpoint
- 📝 **Memory System**: Project memory and context persistence

## 🚀 Quick Start

### Installation

```bash
# Install from source
go install github.com/z1243161996/MiMo-Reasonix/cmd/reasonix@latest

# Or build from source
git clone https://github.com/z1243161996/MiMo-Reasonix.git
cd MiMo-Reasonix
make build
```

### Basic Usage

```bash
# Set your API key
export MIMO_API_KEY="your-api-key"

# Start interactive chat
mimo-reasonix

# Run a specific task
mimo-reasonix run "explain this codebase"

# Start HTTP server
mimo-reasonix serve

# Run configuration wizard
mimo-reasonix setup
```

## 📖 Documentation

- [Configuration Guide](docs/CONFIG_PATHS.md) - Complete configuration reference
- [Desktop Evaluation](docs/DESKTOP_EVALUATION.md) - Desktop app framework comparison
- [Benchmarks](benchmarks/README.md) - Performance benchmarks and profiling

## ⚙️ Configuration

MiMo-Reasonix uses TOML configuration with the following priority:

1. Command-line flags
2. `./mimo-reasonix.toml` (project-local)
3. `~/.mimo-reasonix/config.toml` (user-global)
4. Built-in defaults

### Minimal Configuration

```toml
default_model = "xiaomi/mimo-v2.5"

[[providers]]
name        = "xiaomi"
kind        = "openai"
base_url    = "https://token-plan-cn.xiaomimimo.com/v1"
models      = ["xiaomi/mimo-v2.5", "xiaomi/mimo-v2.5-pro", "xiaomi/mimo-v2-flash"]
default     = "xiaomi/mimo-v2.5"
api_key_env = "MIMO_API_KEY"
context_window = 1000000
```

### MiMo Model Pricing (per 1M tokens)

| Model | Input | Output | Cached Input |
|-------|-------|--------|--------------|
| xiaomi/mimo-v2.5 | ¥7.0 | ¥14.0 | ¥0.7 |
| xiaomi/mimo-v2.5-pro | ¥3.0 | ¥6.0 | ¥0.025 |
| xiaomi/mimo-v2-flash | ¥0.70 | ¥2.10 | ¥0.07 |

## 🔧 Subcommands

| Command | Description |
|---------|-------------|
| `mimo-reasonix` | Start interactive chat session |
| `mimo-reasonix run <task>` | Execute task non-interactively |
| `mimo-reasonix serve` | Start HTTP+SSE server |
| `mimo-reasonix setup` | Run configuration wizard |
| `mimo-reasonix config` | Manage configuration settings |
| `mimo-reasonix doctor` | Show diagnostic information |
| `mimo-reasonix mcp` | Manage MCP servers |
| `mimo-reasonix init` | Initialize project memory |
| `mimo-reasonix version` | Show version information |

## 🧪 Testing

### Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/provider/mimo/...
go test ./internal/config/...
go test ./internal/cli/...
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
| internal/config | 79.5% |
| internal/cli | 56.9% |

### Benchmarks

Run performance benchmarks:

```bash
# Run all benchmarks
make bench

# Run CPU benchmarks
make bench-cpu

# Run memory benchmarks
make bench-memory
```

## 🏗️ Architecture

```
mimo-reasonix/
├── cmd/                    # CLI entry points
├── internal/               # Core packages (51 packages)
│   ├── agent/              # Agent loop and orchestration
│   ├── provider/           # AI model providers (MiMo, DeepSeek, Anthropic)
│   ├── tool/               # Tool system and built-in tools
│   ├── plugin/             # MCP plugin system
│   ├── config/             # Configuration management
│   ├── context/            # Context manager (token counting, folding)
│   ├── repair/             # Tool repair pipeline
│   └── ...
├── benchmarks/             # Performance benchmarks
├── desktop/                # Desktop app (Wails PoC)
├── docs/                   # Documentation
└── tests/                  # Integration tests
```

## 🔨 Development

### Prerequisites

- Go 1.23+
- Make (optional)

### Build

```bash
# Build binary
make build

# Build for multiple platforms
make release
```

### Code Quality

```bash
# Run linter
make lint

# Format code
gofmt -w .

# Run vet
go vet ./...
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow existing code patterns
- Add tests for new functionality
- Update documentation as needed
- Run `make lint` before committing

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [DeepSeek-Reasonix](https://github.com/esengine/DeepSeek-Reasonix) - Original TypeScript implementation
- [Xiaomi MiMo](https://github.com/XiaomiMiMo) - MiMo model series
- [Go](https://golang.org/) - Programming language
- [Wails](https://wails.io/) - Desktop app framework

## 📞 Support

- 🐛 [Report Bug](https://github.com/z1243161996/MiMo-Reasonix/issues)
- 💡 [Request Feature](https://github.com/z1243161996/MiMo-Reasonix/issues)
- 📖 [Documentation](https://github.com/z1243161996/MiMo-Reasonix/tree/main/docs)

---

**Made with ❤️ by [z1243161996](https://github.com/z1243161996)**
