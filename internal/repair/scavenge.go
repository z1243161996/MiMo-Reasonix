package repair

import (
	"encoding/json"
	"regexp"
	"strings"
)

// scavengePatterns matches tool-call shapes that models sometimes embed in
// reasoning or answer text. Each pattern captures the tool name and its
// argument payload. The patterns are tried in order; the first match wins
// for a given code block.
//
// Go's regexp does not support backreferences, so pattern 1 uses a manual
// tag-matching approach in extractTaggedJSON instead.
var scavengePatterns = []*regexp.Regexp{
	// Pattern 2: tool call with function-style arguments
	// e.g.  read_file(path="/etc/hosts")
	// or    read_file({"path": "/etc/hosts"})
	regexp.MustCompile(`(?s)(\w+)\s*\((\{.*?\})\s*\)`),

	// Pattern 3: markdown code block with tool call
	// e.g.  ```json
	//         {"tool": "read_file", "arguments": {"path": "/etc/hosts"}}
	//       ```
	regexp.MustCompile("(?s)```(?:json)?\\s*\\{\\s*\"tool\"\\s*:\\s*\"([^\"]+)\"\\s*,\\s*\"arguments\"\\s*:\\s*(\\{.*?\\})\\s*\\}\\s*```"),

	// Pattern 4: tool call annotation in text
	// e.g.  [TOOL_CALL: read_file {"path": "/etc/hosts"}]
	regexp.MustCompile(`(?s)\[TOOL_CALL:\s*(\w+)\s+(\{.*?\})\]`),
}

// Scavenger extracts tool calls from text where the model incorrectly
// embedded them instead of using the proper tool-call mechanism.
type Scavenger struct {
	patterns []*regexp.Regexp
}

// NewScavenger creates a Scavenger with the default patterns.
func NewScavenger() *Scavenger {
	return &Scavenger{
		patterns: scavengePatterns,
	}
}

// Extract finds all tool calls embedded in text and returns them.
// Each returned ScavengedCall has a non-empty Name and valid JSON Arguments.
func (s *Scavenger) Extract(text string) []ScavengedCall {
	if text == "" {
		return nil
	}

	var calls []ScavengedCall
	seen := make(map[string]bool) // dedup by name+args

	// Pattern 1: manual tag matching (<name>{...}</name>) since Go regexp
	// doesn't support backreferences.
	calls = append(calls, extractTaggedJSON(text, seen)...)

	for _, pat := range s.patterns {
		matches := pat.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			name := strings.TrimSpace(match[1])
			argsRaw := strings.TrimSpace(match[2])

			if name == "" || argsRaw == "" {
				continue
			}

			// Validate that args are valid JSON.
			var v any
			if err := json.Unmarshal([]byte(argsRaw), &v); err != nil {
				continue
			}

			// Normalize the JSON for dedup.
			canonical, err := json.Marshal(v)
			if err != nil {
				continue
			}
			key := name + "\x00" + string(canonical)
			if seen[key] {
				continue
			}
			seen[key] = true

			calls = append(calls, ScavengedCall{
				Name:      name,
				Arguments: string(canonical),
			})
		}
	}

	return calls
}

// extractTaggedJSON manually matches <name>{...}</name> patterns since Go's
// regexp engine does not support backreferences for matching the closing tag.
func extractTaggedJSON(text string, seen map[string]bool) []ScavengedCall {
	var calls []ScavengedCall
	i := 0
	for i < len(text) {
		// Find opening tag: <word>
		start := strings.Index(text[i:], "<")
		if start < 0 {
			break
		}
		start += i
		endTag := strings.Index(text[start:], ">")
		if endTag < 0 {
			break
		}
		tagName := text[start+1 : start+endTag]
		// Only match valid identifier tags (word characters).
		if tagName == "" || !isIdentifier(tagName) {
			i = start + 1
			continue
		}

		// Find the content after the opening tag.
		contentStart := start + endTag + 1

		// Find the closing tag: </name>
		closingTag := "</" + tagName + ">"
		closingIdx := strings.Index(text[contentStart:], closingTag)
		if closingIdx < 0 {
			i = contentStart
			continue
		}

		content := strings.TrimSpace(text[contentStart : contentStart+closingIdx])

		// Try to parse as JSON.
		var v any
		if err := json.Unmarshal([]byte(content), &v); err == nil {
			canonical, err := json.Marshal(v)
			if err == nil {
				key := tagName + "\x00" + string(canonical)
				if !seen[key] {
					seen[key] = true
					calls = append(calls, ScavengedCall{
						Name:      tagName,
						Arguments: string(canonical),
					})
				}
			}
		}

		i = contentStart + closingIdx + len(closingTag)
	}
	return calls
}

// isIdentifier reports whether s is a valid Go identifier (letters, digits, underscores).
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			continue
		}
		if i > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

// ExtractFirst returns the first tool call found in text, or nil.
func (s *Scavenger) ExtractFirst(text string) *ScavengedCall {
	calls := s.Extract(text)
	if len(calls) == 0 {
		return nil
	}
	c := calls[0]
	return &c
}

// LooksLikeToolCall reports whether text contains something that resembles
// an embedded tool call (useful for a quick pre-check before full extraction).
func LooksLikeToolCall(text string) bool {
	if text == "" {
		return false
	}
	// Quick check for tagged pattern: <word>{...}</word>
	if strings.Contains(text, "</") && strings.Contains(text, ">") {
		return true
	}
	// Quick check for markdown code block with tool call.
	if strings.Contains(text, "```") && strings.Contains(text, `"tool"`) {
		return true
	}
	for _, pat := range scavengePatterns {
		if pat.MatchString(text) {
			return true
		}
	}
	return false
}
