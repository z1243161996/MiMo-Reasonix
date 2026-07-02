// Command mimo-reasonix is a config- and plugin-driven coding agent CLI.
package main

import (
	"os"

	"mimo-reasonix/internal/cli"

	// Blank imports wire compile-time built-ins into their registries.
	_ "mimo-reasonix/internal/provider/anthropic"
	_ "mimo-reasonix/internal/provider/openai"
	_ "mimo-reasonix/internal/tool/builtin"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version))
}
