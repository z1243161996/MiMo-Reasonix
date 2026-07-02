package repair

import (
	"encoding/json"
	"testing"
)

// --- Scavenger tests ---

func TestScavengerExtract_TaggedJSON(t *testing.T) {
	s := NewScavenger()
	text := `I need to read the file. Let me call:
<read_file>{"path": "/etc/hosts"}</read_file>`
	calls := s.Extract(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("expected name read_file, got %s", calls[0].Name)
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatalf("args not valid JSON: %v", err)
	}
	if args["path"] != "/etc/hosts" {
		t.Errorf("expected path /etc/hosts, got %s", args["path"])
	}
}

func TestScavengerExtract_MultilineTaggedJSON(t *testing.T) {
	s := NewScavenger()
	text := `Let me edit the file:
<edit_file>
{
  "path": "/tmp/test.txt",
  "content": "hello"
}
</edit_file>`
	calls := s.Extract(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "edit_file" {
		t.Errorf("expected name edit_file, got %s", calls[0].Name)
	}
}

func TestScavengerExtract_FunctionStyle(t *testing.T) {
	s := NewScavenger()
	text := `I'll use read_file({"path": "/etc/passwd"}) to check the file.`
	calls := s.Extract(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("expected name read_file, got %s", calls[0].Name)
	}
}

func TestScavengerExtract_MarkdownCodeBlock(t *testing.T) {
	s := NewScavenger()
	text := `Here is the tool call:
` + "```json" + `
{"tool": "bash", "arguments": {"command": "ls -la"}}
` + "```" + `
Done.`
	calls := s.Extract(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "bash" {
		t.Errorf("expected name bash, got %s", calls[0].Name)
	}
}

func TestScavengerExtract_AnnotationStyle(t *testing.T) {
	s := NewScavenger()
	text := `I will run [TOOL_CALL: bash {"command": "echo hello"}] now.`
	calls := s.Extract(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "bash" {
		t.Errorf("expected name bash, got %s", calls[0].Name)
	}
}

func TestScavengerExtract_Deduplication(t *testing.T) {
	s := NewScavenger()
	text := `<read_file>{"path": "/etc/hosts"}</read_file> and also <read_file>{"path": "/etc/hosts"}</read_file>`
	calls := s.Extract(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call (deduped), got %d", len(calls))
	}
}

func TestScavengerExtract_MultipleDifferentCalls(t *testing.T) {
	s := NewScavenger()
	text := `<read_file>{"path": "/a"}</read_file> then <write_file>{"path": "/b", "content": "x"}</write_file>`
	calls := s.Extract(text)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	names := map[string]bool{}
	for _, c := range calls {
		names[c.Name] = true
	}
	if !names["read_file"] || !names["write_file"] {
		t.Errorf("expected both read_file and write_file, got %v", names)
	}
}

func TestScavengerExtract_NoMatch(t *testing.T) {
	s := NewScavenger()
	text := `This is just plain text with no tool calls.`
	calls := s.Extract(text)
	if len(calls) != 0 {
		t.Fatalf("expected 0 calls, got %d", len(calls))
	}
}

func TestScavengerExtract_Empty(t *testing.T) {
	s := NewScavenger()
	calls := s.Extract("")
	if len(calls) != 0 {
		t.Fatalf("expected 0 calls, got %d", len(calls))
	}
}

func TestScavengerExtract_InvalidJSON(t *testing.T) {
	s := NewScavenger()
	text := `<read_file>{not valid json}</read_file>`
	calls := s.Extract(text)
	if len(calls) != 0 {
		t.Fatalf("expected 0 calls (invalid JSON), got %d", len(calls))
	}
}

func TestScavengerExtractFirst(t *testing.T) {
	s := NewScavenger()
	text := `<read_file>{"path": "/a"}</read_file> then <write_file>{"path": "/b"}</write_file>`
	call := s.ExtractFirst(text)
	if call == nil {
		t.Fatal("expected a call")
	}
	if call.Name != "read_file" {
		t.Errorf("expected first call read_file, got %s", call.Name)
	}
}

func TestScavengerExtractFirst_Empty(t *testing.T) {
	s := NewScavenger()
	call := s.ExtractFirst("no tool calls here")
	if call != nil {
		t.Fatalf("expected nil, got %+v", call)
	}
}

func TestLooksLikeToolCall(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{`<read_file>{"path": "/a"}</read_file>`, true},
		{`read_file({"path": "/a"})`, true},
		{"```json\n{\"tool\": \"bash\"}\n```", true},
		{`[TOOL_CALL: bash {"command": "ls"}]`, true},
		{"just plain text", false},
		{"", false},
	}
	for _, tt := range tests {
		got := LooksLikeToolCall(tt.text)
		if got != tt.want {
			t.Errorf("LooksLikeToolCall(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

// --- Truncator tests ---

func TestTruncatorRepair_UnclosedBrace(t *testing.T) {
	tr := NewTruncator()
	input := `{"path": "/etc/hosts", "content": "hello`
	fixed, ok := tr.Repair(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestTruncatorRepair_UnclosedArray(t *testing.T) {
	tr := NewTruncator()
	input := `{"items": [1, 2, 3`
	fixed, ok := tr.Repair(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestTruncatorRepair_MissingValue(t *testing.T) {
	tr := NewTruncator()
	input := `{"key":`
	fixed, ok := tr.Repair(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestTruncatorRepair_MissingComma(t *testing.T) {
	tr := NewTruncator()
	input := `{"a": 1, "b":`
	fixed, ok := tr.Repair(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestTruncatorRepair_MidString(t *testing.T) {
	tr := NewTruncator()
	input := `{"key": "valu`
	fixed, ok := tr.Repair(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestTruncatorRepair_AlreadyValid(t *testing.T) {
	tr := NewTruncator()
	input := `{"key": "value"}`
	fixed, ok := tr.Repair(input)
	if ok {
		t.Error("expected no repair needed")
	}
	if fixed != input {
		t.Errorf("expected unchanged output, got %s", fixed)
	}
}

func TestTruncatorRepair_Empty(t *testing.T) {
	tr := NewTruncator()
	fixed, ok := tr.Repair("")
	if ok {
		t.Error("expected no repair for empty input")
	}
	if fixed != "" {
		t.Errorf("expected empty output, got %q", fixed)
	}
}

func TestTruncatorRepair_NestedStructure(t *testing.T) {
	tr := NewTruncator()
	input := `{"outer": {"inner": [1, 2,`
	fixed, ok := tr.Repair(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestRepairJSON_Convenience(t *testing.T) {
	input := `{"command": "echo hello"`
	fixed, ok := RepairJSON(input)
	if !ok {
		t.Fatal("expected repair to succeed")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired JSON not valid: %s", fixed)
	}
}

func TestIsTruncatedJSON(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`{"key": "value"}`, false},     // valid JSON
		{`{"key": "valu`, true},         // truncated string
		{`{"a": [1, 2`, true},           // truncated array
		{`{`, true},                     // just opening brace
		{`not json at all`, false},      // not JSON-like
		{``, false},                     // empty
		{`"hello`, true},                // truncated string value
	}
	for _, tt := range tests {
		got := IsTruncatedJSON(tt.input)
		if got != tt.want {
			t.Errorf("IsTruncatedJSON(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- StormBreaker tests ---

func TestStormBreaker_BelowThreshold(t *testing.T) {
	sb := NewStormBreaker(3)
	outcomes := []ToolOutcome{
		{Name: "bash", Error: "exit status 1"},
	}
	intervene, _, count := sb.Check(outcomes)
	if intervene {
		t.Error("should not intervene below threshold")
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
}

func TestStormBreaker_AtThreshold(t *testing.T) {
	sb := NewStormBreaker(3)
	outcomes := []ToolOutcome{
		{Name: "bash", Error: "exit status 1"},
	}
	// Run 3 times with same signature.
	for i := 0; i < 3; i++ {
		intervene, msg, count := sb.Check(outcomes)
		if i < 2 {
			if intervene {
				t.Errorf("should not intervene at iteration %d", i)
			}
		} else {
			if !intervene {
				t.Error("should intervene at threshold")
			}
			if msg == "" {
				t.Error("expected non-empty intervention message")
			}
			if count != 3 {
				t.Errorf("expected count 3, got %d", count)
			}
		}
	}
}

func TestStormBreaker_ResetOnSuccess(t *testing.T) {
	sb := NewStormBreaker(3)
	// Two failures.
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	// Success resets.
	intervene, _, count := sb.Check([]ToolOutcome{{Name: "bash", Error: ""}})
	if intervene {
		t.Error("should not intervene after success")
	}
	if count != 0 {
		t.Errorf("expected count 0 after reset, got %d", count)
	}
}

func TestStormBreaker_ResetOnDifferentError(t *testing.T) {
	sb := NewStormBreaker(3)
	// Two failures with same error.
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	// Different error resets.
	intervene, _, count := sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 2"}})
	if intervene {
		t.Error("should not intervene after different error")
	}
	if count != 1 {
		t.Errorf("expected count 1 after reset, got %d", count)
	}
}

func TestStormBreaker_BlockedResets(t *testing.T) {
	sb := NewStormBreaker(3)
	// Two failures.
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	// Blocked resets.
	intervene, _, count := sb.Check([]ToolOutcome{{Name: "bash", Error: "blocked", Blocked: true}})
	if intervene {
		t.Error("should not intervene after blocked")
	}
	if count != 0 {
		t.Errorf("expected count 0 after blocked reset, got %d", count)
	}
}

func TestStormBreaker_MultiCallBatch(t *testing.T) {
	sb := NewStormBreaker(3)
	outcomes := []ToolOutcome{
		{Name: "bash", Error: "exit status 1"},
		{Name: "read_file", Error: "file not found"},
	}
	for i := 0; i < 3; i++ {
		intervene, msg, _ := sb.Check(outcomes)
		if i < 2 && intervene {
			t.Errorf("should not intervene at iteration %d", i)
		}
		if i == 2 {
			if !intervene {
				t.Error("should intervene at threshold for multi-call batch")
			}
			if msg == "" {
				t.Error("expected non-empty message")
			}
		}
	}
}

func TestStormBreaker_EmptyOutcomes(t *testing.T) {
	sb := NewStormBreaker(3)
	intervene, _, count := sb.Check(nil)
	if intervene {
		t.Error("should not intervene with empty outcomes")
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestStormBreaker_ResetMethod(t *testing.T) {
	sb := NewStormBreaker(3)
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	sb.Check([]ToolOutcome{{Name: "bash", Error: "exit status 1"}})
	sb.Reset()
	if sb.Count != 0 {
		t.Errorf("expected count 0 after reset, got %d", sb.Count)
	}
	if sb.CurrentSig != "" {
		t.Errorf("expected empty sig after reset")
	}
}

func TestBatchSignature(t *testing.T) {
	// All failures → valid signature.
	outcomes := []ToolOutcome{
		{Name: "bash", Error: "exit status 1"},
		{Name: "bash", Error: "exit status 1"},
	}
	sig, ok := batchSignature(outcomes)
	if !ok {
		t.Error("expected ok=true for all failures")
	}
	if sig == "" {
		t.Error("expected non-empty signature")
	}

	// Success → no signature.
	_, ok = batchSignature([]ToolOutcome{{Name: "bash", Error: ""}})
	if ok {
		t.Error("expected ok=false for success")
	}

	// Blocked → no signature.
	_, ok = batchSignature([]ToolOutcome{{Name: "bash", Error: "blocked", Blocked: true}})
	if ok {
		t.Error("expected ok=false for blocked")
	}

	// Empty → no signature.
	_, ok = batchSignature(nil)
	if ok {
		t.Error("expected ok=false for empty")
	}
}

// --- RepeatSuccessTracker tests ---

func TestRepeatSuccessTracker_BelowThreshold(t *testing.T) {
	rst := NewRepeatSuccessTracker()
	blocked, _ := rst.Record("write_file", `{"path":"/a","content":"hello"}`)
	if blocked {
		t.Error("should not block below threshold")
	}
	// Second call is at threshold (2), so it should block.
	blocked, _ = rst.Record("write_file", `{"path":"/a","content":"hello"}`)
	if !blocked {
		t.Error("should block at threshold")
	}
}

func TestRepeatSuccessTracker_AtThreshold(t *testing.T) {
	rst := NewRepeatSuccessTracker()
	args := `{"path":"/a","content":"hello"}`
	for i := 0; i < 2; i++ {
		rst.Record("write_file", args)
	}
	blocked, msg := rst.Record("write_file", args)
	if !blocked {
		t.Error("should block at threshold")
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestRepeatSuccessTracker_DifferentArgs(t *testing.T) {
	rst := NewRepeatSuccessTracker()
	// Same tool, different args → should not block.
	for i := 0; i < 3; i++ {
		blocked, _ := rst.Record("write_file", `{"path":"/a","content":"`+string(rune('x'+i))+`"}`)
		if blocked {
			t.Errorf("should not block with different args at iteration %d", i)
		}
	}
}

func TestRepeatSuccessTracker_Reset(t *testing.T) {
	rst := NewRepeatSuccessTracker()
	args := `{"path":"/a"}`
	rst.Record("write_file", args)
	rst.Record("write_file", args)
	rst.Reset()
	blocked, _ := rst.Record("write_file", args)
	if blocked {
		t.Error("should not block after reset")
	}
}

func TestRepeatSuccessTracker_Count(t *testing.T) {
	rst := NewRepeatSuccessTracker()
	args := `{"path":"/a"}`
	rst.Record("write_file", args)
	rst.Record("write_file", args)
	if got := rst.Count("write_file", args); got != 2 {
		t.Errorf("expected count 2, got %d", got)
	}
	if got := rst.Count("write_file", `{"path":"/b"}`); got != 0 {
		t.Errorf("expected count 0 for different args, got %d", got)
	}
}

// --- NormalizeJSON tests ---

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		input   string
		wantOK  bool
	}{
		{`{"b": 1, "a": 2}`, true},
		{`[3, 1, 2]`, true},
		{`"hello"`, true},
		{`42`, true},
		{`not json`, false},
		{``, false},
		{`{`, false},
	}
	for _, tt := range tests {
		_, ok := NormalizeJSON(tt.input)
		if ok != tt.wantOK {
			t.Errorf("NormalizeJSON(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
		}
	}
}

func TestNormalizeJSON_PreservesContent(t *testing.T) {
	input := `{"key": "value", "num": 42}`
	normalized, ok := NormalizeJSON(input)
	if !ok {
		t.Fatal("expected success")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(normalized), &m); err != nil {
		t.Fatalf("normalized JSON invalid: %v", err)
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m["key"])
	}
	if m["num"] != float64(42) {
		t.Errorf("expected num=42, got %v", m["num"])
	}
}

// --- Pipeline tests ---

func TestPipeline_RepairArgs_Truncated(t *testing.T) {
	p := NewPipeline()
	input := `{"command": "echo hello"`
	fixed, ok := p.RepairArgs(input)
	if !ok {
		t.Fatal("expected repair")
	}
	if !json.Valid([]byte(fixed)) {
		t.Fatalf("repaired args not valid JSON: %s", fixed)
	}
}

func TestPipeline_RepairArgs_Valid(t *testing.T) {
	p := NewPipeline()
	input := `{"command": "echo hello"}`
	fixed, ok := p.RepairArgs(input)
	if ok {
		t.Error("expected no repair for valid JSON")
	}
	if fixed != input {
		t.Errorf("expected unchanged, got %s", fixed)
	}
}

func TestPipeline_RepairArgs_Empty(t *testing.T) {
	p := NewPipeline()
	fixed, ok := p.RepairArgs("")
	if ok {
		t.Error("expected no repair for empty")
	}
	if fixed != "" {
		t.Errorf("expected empty, got %q", fixed)
	}
}

func TestPipeline_ScavengeCalls(t *testing.T) {
	p := NewPipeline()
	text := `<read_file>{"path": "/etc/hosts"}</read_file>`
	calls := p.ScavengeCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("expected read_file, got %s", calls[0].Name)
	}
}
