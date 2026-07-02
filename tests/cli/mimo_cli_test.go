// Package cli_test defines the expected behavior for the MiMo CLI binary.
// These tests are written before implementation (TDD) and will fail until the
// binary renaming and subcommand registration are implemented.
//
// Expected MiMo CLI:
//   - Binary name: mimo-reasonix (not reasonix)
//   - Subcommands: chat, run, setup, doctor, mcp, plugin, skill, config, etc.
package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ── TestBinaryName ─────────────────────────────────────────────────────────

// TestBinaryName verifies that the binary is named mimo-reasonix, not reasonix.
// This is the user-facing command name.
func TestBinaryName(t *testing.T) {
	// Expected binary names
	expectedBinary := "mimo-reasonix"
	expectedBinDir := "bin"

	// Check that the Makefile builds to the correct name
	makefilePath := filepath.Join("..", "..", "Makefile")
	data, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Skip("Makefile not found, skipping binary name check")
	}

	makefileContent := string(data)

	t.Run("Makefile builds mimo-reasonix binary", func(t *testing.T) {
		// The Makefile should have a build target that produces mimo-reasonix
		if !containsString(makefileContent, "mimo-reasonix") {
			t.Error("Makefile should reference 'mimo-reasonix' binary name")
		}
	})

	t.Run("build produces mimo-reasonix executable", func(t *testing.T) {
		// After building, bin/mimo-reasonix should exist
		binaryPath := filepath.Join("..", "..", expectedBinDir, expectedBinary)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Skipf("binary not built yet: %s", binaryPath)
		}

		// Verify it's executable
		info, err := os.Stat(binaryPath)
		if err != nil {
			t.Fatalf("failed to stat binary: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Error("binary is not executable")
		}
	})

	t.Run("binary responds to --version", func(t *testing.T) {
		binaryPath := filepath.Join("..", "..", expectedBinDir, expectedBinary)
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Skipf("binary not built yet: %s", binaryPath)
		}

		cmd := exec.Command(binaryPath, "--version")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("--version failed: %v, output: %s", err, string(out))
		}
		if len(out) == 0 {
			t.Error("--version produced no output")
		}
	})
}

// ── TestSubcommands ────────────────────────────────────────────────────────

// TestSubcommands verifies that all expected subcommands exist.
// These are the core subcommands that MiMo-Reasonix should support.
func TestSubcommands(t *testing.T) {
	// Expected subcommands (from Reasonix upstream + MiMo additions)
	expectedSubcommands := []string{
		// Core subcommands
		"chat",   // Interactive chat TUI
		"run",    // Non-interactive execution
		"setup",  // Initial configuration
		"doctor", // Health check
		"config", // Configuration management
		// Plugin/MCP subcommands
		"mcp",    // MCP server management
		"plugin", // Plugin management
		// Session management
		"resume", // Resume a previous session
		// Memory/Context
		"memory", // Memory management
		// Model management
		"model",  // Model switching
		"effort", // Reasoning effort control
		// Utility
		"upgrade", // Self-upgrade
	}

	binaryPath := filepath.Join("..", "..", "bin", "mimo-reasonix")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("binary not built yet: %s", binaryPath)
	}

	for _, subcmd := range expectedSubcommands {
		t.Run(subcmd, func(t *testing.T) {
			// After implementation, verify that the subcommand is registered
			// by checking --help output or running the subcommand with --help
			cmd := exec.Command(binaryPath, subcmd, "--help")
			out, err := cmd.CombinedOutput()
			if err != nil {
				// Some subcommands might not have --help, check stderr
				t.Logf("subcommand %s --help: %v, output: %s", subcmd, err, string(out))
			}
			// The subcommand should at least be recognized (not "unknown command")
			if containsString(string(out), "unknown command") {
				t.Errorf("subcommand %q not recognized", subcmd)
			}
		})
	}

	t.Run("unknown subcommand returns error", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "nonexistent-subcommand")
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("unknown subcommand should return error")
		}
		if !containsString(string(out), "unknown") && !containsString(string(out), "invalid") {
			t.Logf("output for unknown subcommand: %s", string(out))
		}
	})
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
