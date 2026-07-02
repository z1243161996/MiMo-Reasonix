package repair

import (
	"encoding/json"
	"strings"
	"unicode"
)

// Truncator attempts to repair truncated JSON arguments by closing open
// brackets, braces, and strings. This handles the common case where the
// model's output tokens are exhausted mid-argument, leaving incomplete JSON.
type Truncator struct {
	// MaxRepairAttempts limits how many closing characters we try to append.
	MaxRepairAttempts int
}

// NewTruncator creates a Truncator with default settings.
func NewTruncator() *Truncator {
	return &Truncator{
		MaxRepairAttempts: 16,
	}
}

// Repair attempts to fix truncated JSON. It returns the repaired JSON and true
// if the input was truncated and could be fixed, or the original and false
// if no repair was needed or possible.
func (t *Truncator) Repair(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, false
	}

	// If it's already valid JSON, no repair needed.
	if json.Valid([]byte(raw)) {
		return raw, false
	}

	// Detect whether this looks like a truncation (as opposed to other JSON errors).
	if !looksTruncated(raw) {
		return raw, false
	}

	repaired := t.attemptRepair(raw)
	if repaired != raw && json.Valid([]byte(repaired)) {
		return repaired, true
	}

	return raw, false
}

// looksTruncated returns true if the string appears to be a truncated JSON
// value rather than simply malformed. Truncation indicators:
//   - Ends mid-string (unclosed quote)
//   - Ends after a colon or comma (missing value)
//   - Ends after an opening bracket/brace
//   - Ends mid-number or mid-keyword
//   - Starts like JSON but ends with a letter (part of an incomplete value)
func looksTruncated(raw string) bool {
	raw = strings.TrimRightFunc(raw, unicode.IsSpace)
	if raw == "" {
		return false
	}
	last := raw[len(raw)-1]
	first := raw[0]

	// Quick check: if it doesn't look like JSON at all, it's not truncated JSON.
	// JSON values start with {, [, ", digit, or minus.
	// Also allow 't' (true), 'f' (false), 'n' (null) but only if they match
	// the keyword prefix.
	if first != '{' && first != '[' && first != '"' && first != '-' &&
		!(first >= '0' && first <= '9') {
		// Check if it starts with a complete JSON keyword.
		lowerRaw := strings.ToLower(raw)
		if !strings.HasPrefix(lowerRaw, "true") &&
			!strings.HasPrefix(lowerRaw, "false") &&
			!strings.HasPrefix(lowerRaw, "null") {
			return false
		}
	}

	// Ends mid-string: unclosed quote.
	if last == '"' {
		// Count unescaped quotes at the end to see if the string is closed.
		if !stringIsClosed(raw) {
			return true
		}
		// String is closed but the overall JSON might still be truncated
		// (missing closing brackets). Check if there are open brackets.
		if hasOpenBrackets(raw) {
			return true
		}
	}

	// Ends after opening delimiter: missing content and closing.
	if last == '{' || last == '[' || last == ':' || last == ',' {
		return true
	}

	// Ends mid-keyword (true, false, null).
	lower := strings.ToLower(raw)
	for _, kw := range []string{"true", "false", "null"} {
		if strings.HasSuffix(lower, kw) {
			// Keyword is complete. Check if it's in a valid JSON position.
			// For now, treat complete keywords as not truncated.
			continue
		}
		// Partial keyword match.
		for i := 1; i < len(kw); i++ {
			if strings.HasSuffix(lower, kw[:i]) {
				return true
			}
		}
	}

	// Ends mid-number (digit at end with no following structure).
	if last >= '0' && last <= '9' {
		return true
	}

	// If the string starts like JSON and ends with a letter, it's likely
	// truncated (part of an incomplete string value or key).
	// We need to check if we're inside an unclosed string by tracking quotes.
	// A complete string ends with '"' that closes the opening '"'.
	inStr := false
	for _, ch := range raw {
		if ch == '"' {
			inStr = !inStr
		}
	}
	// If we're inside an unclosed string and the last char is not a quote,
	// the string is truncated.
	if inStr && last != '"' {
		return true
	}

	return false
}

// stringIsClosed checks whether a string ending with '"' has its opening
// quote properly matched (all intermediate quotes are escaped).
func stringIsClosed(raw string) bool {
	// Walk backwards from the last character (which is '"').
	// Count consecutive backslashes immediately before it.
	i := len(raw) - 2 // character before closing quote
	escCount := 0
	for i >= 0 && raw[i] == '\\' {
		escCount++
		i--
	}
	// An even number of backslashes means the closing quote is real (not escaped).
	return escCount%2 == 0
}

// hasOpenBrackets reports whether the raw string has unmatched opening
// brackets or braces that would need to be closed for valid JSON.
func hasOpenBrackets(raw string) bool {
	var stack []rune
	inString := false
	escaped := false

	for _, ch := range raw {
		if escaped {
			escaped = false
			continue
		}
		if inString {
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 && stack[len(stack)-1] == ch {
				stack = stack[:len(stack)-1]
			}
		}
	}

	return len(stack) > 0
}

// attemptRepair tries to close an open JSON structure by appending the
// minimal set of closing characters. It handles the case where we need to
// add a value after a colon or comma before closing.
func (t *Truncator) attemptRepair(raw string) string {
	// Track the stack of open delimiters.
	var stack []rune
	inString := false
	escaped := false
	lastNonSpace := rune(0)

	for _, ch := range raw {
		if escaped {
			escaped = false
			continue
		}
		if inString {
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
			lastNonSpace = ch
		case '{':
			stack = append(stack, '}')
			lastNonSpace = ch
		case '[':
			stack = append(stack, ']')
			lastNonSpace = ch
		case ':', ',':
			lastNonSpace = ch
		default:
			if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
				lastNonSpace = ch
			}
		}
	}

	// If we're in the middle of a string, close it first.
	if inString {
		raw += `"`
	}

	// If the last meaningful character was ':' or ',', we need to add a value
	// before closing. Use 'null' as the placeholder value.
	var suffix strings.Builder
	if lastNonSpace == ':' || lastNonSpace == ',' {
		suffix.WriteString("null")
	}

	// Close any open delimiters in reverse order.
	for i := len(stack) - 1; i >= 0; i-- {
		suffix.WriteByte(byte(stack[i]))
	}

	if suffix.Len() == 0 {
		return raw
	}
	return raw + suffix.String()
}

// RepairJSON attempts to repair any truncated JSON, returning the result.
// This is a convenience function that creates a default Truncator.
func RepairJSON(raw string) (string, bool) {
	t := NewTruncator()
	return t.Repair(raw)
}

// IsTruncatedJSON reports whether raw looks like truncated JSON that might
// be repairable (useful for logging/diagnostics).
func IsTruncatedJSON(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if json.Valid([]byte(raw)) {
		return false
	}
	return looksTruncated(raw)
}
