// Package repair provides a pipeline for fixing common tool-call pathologies
// before results are fed back to the model. It handles three categories:
//
//  1. Scavenge: extracts tool calls embedded in reasoning text when the model
//     outputs them incorrectly (e.g. inside markdown code blocks).
//
//  2. Truncation: repairs truncated JSON arguments by closing open brackets.
//
//  3. Storm breaker: detects death-spiral loops where the same tool call fails
//     identically multiple times and injects a directive to change approach.
package repair

import (
	"encoding/json"
	"strings"
)

// Result holds the outcome of running the repair pipeline on a tool call.
type Result struct {
	// Repaired is the (possibly modified) arguments after repair.
	Repaired string
	// Fixed indicates whether any repair was applied.
	Fixed bool
	// Method describes which repair was applied (e.g. "scavenge", "truncation").
	Method string
}

// Pipeline runs a sequence of repair passes on tool-call arguments.
type Pipeline struct {
	scavenger    *Scavenger
	truncator    *Truncator
	stormBreaker *StormBreaker
}

// NewPipeline constructs a Pipeline with default settings.
func NewPipeline() *Pipeline {
	return &Pipeline{
		scavenger:    NewScavenger(),
		truncator:    NewTruncator(),
		stormBreaker: NewStormBreaker(DefaultStormBreakThreshold),
	}
}

// RepairArgs runs all applicable repair passes on the raw arguments string.
// It returns the (possibly unchanged) arguments and whether any fix was applied.
func (p *Pipeline) RepairArgs(raw string) (string, bool) {
	if raw == "" {
		return raw, false
	}

	// Pass 1: truncation repair (close open JSON structures).
	if fixed, ok := p.truncator.Repair(raw); ok {
		return fixed, true
	}

	// Pass 2: scavenging is not applicable to raw args (it operates on text).
	return raw, false
}

// ScavengeCalls extracts tool calls embedded in reasoning or answer text.
// It returns any discovered ToolCall-like structs (name + arguments).
func (p *Pipeline) ScavengeCalls(text string) []ScavengedCall {
	return p.scavenger.Extract(text)
}

// ScavengedCall is a tool call extracted from text by the scavenger.
type ScavengedCall struct {
	Name      string
	Arguments string
}

// NormalizeJSON attempts to parse and re-serialize JSON to a canonical form.
// Returns the normalized string and true on success, or the original and false.
func NormalizeJSON(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, false
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw, false
	}
	b, err := json.Marshal(v)
	if err != nil {
		return raw, false
	}
	return string(b), true
}
