package history

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// subagentCall is one assistant line carrying one tool call, as the host
// writes it.
func subagentCall(t *testing.T, tool string, input any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"parentUuid": "p", "isSidechain": true,
		"message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "tool_use", "id": "toolu_1", "name": tool, "input": input},
		}},
		"type": "assistant",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func subagentShell(t *testing.T, command string) string {
	t.Helper()
	return subagentCall(t, "Bash", map[string]any{"command": command, "description": "d"})
}

func subagentTranscript(lines ...string) []byte { return []byte(strings.Join(lines, "\n") + "\n") }

// subagentTestAsk is the judge's question with the host's file writers and a
// shell line that commits as the writes.
func subagentTestAsk() SubagentAsk {
	return SubagentAsk{
		File: "AGENTS.md", RunDir: ".abcd/.work.local/run", BriefFile: "brief.md",
		IsWrite: func(tool, command string) bool {
			return tool == "Write" || tool == "Edit" || strings.Contains(command, "git commit")
		},
	}
}

// TestJudgeSubagentNamesWhatWasSkipped: the two readings, each found on its
// own, in every form a call can name them.
func TestJudgeSubagentNamesWhatWasSkipped(t *testing.T) {
	read := subagentCall(t, "Read", map[string]any{"file_path": "/any/dir/AGENTS.md"})
	rules := subagentShell(t, `scripts/abcd.sh rules --prompt "x"`)
	cases := []struct {
		name  string
		lines []string
		want  SubagentReading
	}{
		{"nothing", []string{subagentShell(t, "ls")}, SubagentReading{}},
		{"both", []string{read, rules}, SubagentReading{ReadFile: true, RanRules: true}},
		{"file only", []string{read}, SubagentReading{ReadFile: true}},
		{"rules only", []string{rules}, SubagentReading{RanRules: true}},
		{"cat", []string{subagentShell(t, "cat AGENTS.md")}, SubagentReading{ReadFile: true}},
		{"sed", []string{subagentShell(t, "sed -n 1,40p ./AGENTS.md")}, SubagentReading{ReadFile: true}},
		{"a search", []string{subagentCall(t, "Grep", map[string]any{"pattern": "x", "glob": "AGENTS.md"})}, SubagentReading{ReadFile: true}},
		{"any depth", []string{subagentCall(t, "Many", map[string]any{"a": []any{map[string]any{"b": "AGENTS.md"}}})}, SubagentReading{ReadFile: true}},
		{"another name", []string{subagentShell(t, "cat AGENTS.md.bak XAGENTS.md AGENTS.mdx my-AGENTS.md AGENTS.md_x ÉAGENTS.md")}, SubagentReading{}},
		{"a key", []string{subagentCall(t, "X", map[string]any{"AGENTS.md": "y"})}, SubagentReading{}},
		{"a write of it", []string{subagentCall(t, "Write", map[string]any{"file_path": "/w/AGENTS.md", "content": "x"})}, SubagentReading{Wrote: true}},
		{"a read and a commit", []string{subagentShell(t, "cat AGENTS.md && git commit -m x")}, SubagentReading{Wrote: true}},
		{"an absolute path to rules", []string{subagentShell(t, "cd /w && /opt/x/abcd rules COMMITTING")}, SubagentReading{RanRules: true}},
		{"a quoted path to rules", []string{subagentShell(t, `'/opt/a b/abcd' rules --prompt "y"`)}, SubagentReading{RanRules: true}},
		{"an escaped space", []string{subagentShell(t, `/opt/a\ b/abcd rules`)}, SubagentReading{RanRules: true}},
		{"a double-quoted path", []string{subagentShell(t, `"/opt/a b/abcd" "rules"`)}, SubagentReading{RanRules: true}},
		{"rules on the next command", []string{subagentShell(t, "abcd; rules")}, SubagentReading{}},
		{"not followed", []string{subagentShell(t, "abcd lint rules")}, SubagentReading{}},
		{"another program", []string{subagentShell(t, "myabcd rules")}, SubagentReading{}},
		{"rules in another tool", []string{subagentCall(t, "Read", map[string]any{"command": "abcd rules"})}, SubagentReading{}},
		{"rules in quotes", []string{subagentShell(t, `echo "abcd rules"`)}, SubagentReading{}},
	}
	for _, c := range cases {
		if got := JudgeSubagent(subagentTranscript(c.lines...), subagentTestAsk()); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
	// With no conventions file there is none to read.
	ask := subagentTestAsk()
	ask.File = ""
	if got := JudgeSubagent(subagentTranscript(read, rules), ask); got != (SubagentReading{RanRules: true}) {
		t.Errorf("no file: %+v", got)
	}
}

// TestJudgeSubagentCountsOnlyToolCalls: the name counts inside the input of a
// tool call on an assistant line, and nowhere else.
func TestJudgeSubagentCountsOnlyToolCalls(t *testing.T) {
	call := subagentCall(t, "Read", map[string]any{"file_path": "AGENTS.md"})
	quoted, _ := json.Marshal(call)
	for name, line := range map[string]string{
		"a user line":                           `{"type":"user","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"AGENTS.md"}}]}}`,
		"a user line naming the assistant type": `{"type":"user","note":{"type":"assistant"},"message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"AGENTS.md"}}]}}`,
		"a copy of a call":                      `{"type":"api-request-blob","blob":` + string(quoted) + `,"copy":` + call + `}`,
		"an instructions attachment":            `{"type":"attachment","attachment":{"type":"instructions","path":"AGENTS.md"}}`,
		"a call's line outside its input":       strings.Replace(subagentCall(t, "Read", map[string]any{"file_path": "x"}), `"toolu_1"`, `"AGENTS.md"`, 1),
		"assistant text":                        `{"type":"assistant","message":{"content":[{"type":"text","text":"read AGENTS.md"}]}}`,
		"a string content":                      `{"type":"assistant","message":{"content":"AGENTS.md"}}`,
		"a torn line":                           call[:len(call)-10],
	} {
		if got := JudgeSubagent(subagentTranscript(line), subagentTestAsk()); got != (SubagentReading{}) {
			t.Errorf("%s counted: %+v", name, got)
		}
	}
	spaced := strings.Replace(call, `"type":"assistant"`, `"type": "assistant"`, 1)
	if spaced == call {
		t.Fatal("the fixture's type key was not spaced")
	}
	if got := JudgeSubagent(subagentTranscript(`not json`, "", spaced), subagentTestAsk()); !got.ReadFile {
		t.Errorf("a call on a line whose type key is spaced is not read: %+v", got)
	}
	two := `{"type":"assistant","message":{"content":[{"type":"text","text":"t"},{"type":"tool_use","name":"Bash","input":{"command":"ls"}},{"type":"tool_use","name":"Read","input":{"file_path":"AGENTS.md"}}]}}`
	if got := JudgeSubagent(subagentTranscript(two), subagentTestAsk()); !got.ReadFile {
		t.Errorf("the second call of a line is not read: %+v", got)
	}
}

// TestJudgeSubagentTakesALanesBriefForTheConventions: a call that is no write
// and names a lane's own brief, and no deeper one.
func TestJudgeSubagentTakesALanesBriefForTheConventions(t *testing.T) {
	brief := "/w/.abcd/.work.local/run/run-1/lane-1/brief.md"
	for path, want := range map[string]bool{
		brief: true,
		".abcd/.work.local/run/run-2611251757228271/lane-12/brief.md":         true,
		"x.abcd/.work.local/run/run-1/lane-1/brief.md":                        false,
		"/w/.abcd/.work.local/run/run-1/lane-1/brief.md.bak":                  false,
		"/w/.abcd/.work.local/run/run-1/lane-1/brief.md/x":                    false,
		"/w/.abcd/.work.local/run/run-1/lane-1/validate/round-1/brief.md":     false,
		"/w/.abcd/.work.local/run/run-1/lane-1/validate/round-1/fix/brief.md": false,
		"/w/.abcd/.work.local/run/run-1/brief.md":                             false,
		"/w/.abcd/.work.local/run/../lane-1/brief.md":                         false,
		"/w/other/run/run-1/lane-1/brief.md":                                  false,
	} {
		got := JudgeSubagent(subagentTranscript(subagentCall(t, "Read", map[string]any{"file_path": path})), subagentTestAsk())
		if got.ReadBrief != want || got.ReadFile {
			t.Errorf("%s: %+v, want a brief %v", path, got, want)
		}
	}
	if got := JudgeSubagent(subagentTranscript(subagentShell(t, "cat "+brief+" && git commit -m x")), subagentTestAsk()); got.ReadBrief {
		t.Errorf("a call that writes counts as the brief's reading: %+v", got)
	}
	if got := JudgeSubagent(subagentTranscript(subagentShell(t, "cat '"+brief+"'")), subagentTestAsk()); !got.ReadBrief {
		t.Errorf("a shell read of the brief is not counted: %+v", got)
	}
}

// TestSubagentReadingsCountEveryLineAndKeepTheNewest: every line is counted,
// concurrent ones included, the newest fifty are kept, each is reported once,
// and nothing the host sent grows the marker without bound.
func TestSubagentReadingsCountEveryLineAndKeepTheNewest(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if _, ok, err := ReadSubagentLines(repoRoot, testRootSHA); err != nil || ok {
		t.Fatalf("a fresh store holds lines: ok=%v err=%v", ok, err)
	}
	if n, err := TakeSubagentLines(repoRoot, testRootSHA); err != nil || n != 0 {
		t.Fatalf("a fresh store reports %d lines (%v)", n, err)
	}
	const n = 12
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = NoteSubagentLine(repoRoot, testRootSHA, SubagentLine{AgentType: "t", AgentID: fmt.Sprintf("c%d", i), Kind: SubagentFileUnread}, 5*time.Second)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
	}
	got, ok, err := ReadSubagentLines(repoRoot, testRootSHA)
	if err != nil || !ok || got.Count != n || len(got.Lines) != n {
		t.Fatalf("count=%d kept=%d ok=%v err=%v, want %d of each", got.Count, len(got.Lines), ok, err, n)
	}
	if k, err := TakeSubagentLines(repoRoot, testRootSHA); err != nil || k != n {
		t.Errorf("first take = %d (%v), want %d", k, err, n)
	}
	if k, err := TakeSubagentLines(repoRoot, testRootSHA); err != nil || k != 0 {
		t.Errorf("second take = %d (%v), want 0: a line is counted once", k, err)
	}
	for i := range subagentLinesKept + 3 {
		line := SubagentLine{AgentType: strings.Repeat("\x01", 5000), AgentID: fmt.Sprintf("a%d", i),
			Kind: SubagentNotJudged, Cause: SubagentNoTranscript, Hold: SubagentHoldBoth}
		if err := NoteSubagentLine(repoRoot, testRootSHA, line, SubagentLineLock); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err = ReadSubagentLines(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != n+subagentLinesKept+3 || len(got.Lines) != subagentLinesKept {
		t.Errorf("count=%d kept=%d, want %d counted and %d kept", got.Count, len(got.Lines), n+subagentLinesKept+3, subagentLinesKept)
	}
	last := got.Lines[len(got.Lines)-1]
	if last.AgentID != fmt.Sprintf("a%d", subagentLinesKept+2) || last.Cause != SubagentNoTranscript || last.Hold != SubagentHoldBoth || last.At.IsZero() {
		t.Errorf("the newest line = %+v", last)
	}
	if got.FirstSeen.IsZero() || got.LastSeen.Before(got.FirstSeen) {
		t.Errorf("first/last seen = %v / %v", got.FirstSeen, got.LastSeen)
	}
	if k, _ := TakeSubagentLines(repoRoot, testRootSHA); k != subagentLinesKept+3 {
		t.Errorf("a later take = %d, want the %d lines since", k, subagentLinesKept+3)
	}
	path, err := subagentReadingsPath(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the marker's mode = %v, want 0600", fi.Mode().Perm())
	}
	// A marker that no longer parses is replaced, and says when.
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := TakeSubagentLines(repoRoot, testRootSHA); err == nil {
		t.Error("an unreadable marker is taken as no lines")
	}
	if err := NoteSubagentLine(repoRoot, testRootSHA, SubagentLine{AgentID: "z", Kind: SubagentStartFailed}, SubagentLineLock); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := ReadSubagentLines(repoRoot, testRootSHA); got.Count != 1 || got.EarlierLostAt.IsZero() {
		t.Errorf("after a replaced marker: %+v", got)
	}
	// A kind outside the closed set is refused, and so is a bad root.
	if err := NoteSubagentLine(repoRoot, testRootSHA, SubagentLine{Kind: "other"}, SubagentLineLock); err == nil {
		t.Error("a kind outside the closed set was kept")
	}
	if err := NoteSubagentLine(repoRoot, "nope", SubagentLine{Kind: SubagentFileUnread}, SubagentLineLock); err == nil {
		t.Error("a root that is no commit was accepted")
	}
}
