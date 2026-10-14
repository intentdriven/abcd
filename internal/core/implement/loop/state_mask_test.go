package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// maskForgeRunes are the runes the mask tests plant: a bidi override and the
// C1 control sequence introducer.
var maskForgeRunes = []string{"‮", "\u009b"}

// forgeStoredText plants bad in every repository- and tool-derived text field
// a run's state.json stores (iss-2610121542240257), building the lists that
// carry them where the run has none yet, and returns the forged file.
func forgeStoredText(t *testing.T, stored []byte, bad string) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(stored, &doc); err != nil {
		t.Fatal(err)
	}
	lane := doc["lanes"].([]any)[0].(map[string]any)
	lane["step_title"] = "guard the map" + bad
	lane["syncs"] = []any{map[string]any{"at": "2026-10-12T10:00:00Z", "siblings": []any{"lane-2"}, "merged": strings.Repeat("b", 40),
		"conflicted": true, "paths": []any{"docs/" + bad + "readme.md"}}}
	if aws, ok := lane["awaits"].([]any); ok && len(aws) > 0 {
		aws[0].(map[string]any)["kept"] = "git: could not save" + bad
	} else {
		t.Fatal("the forged run's lane awaits no agent")
	}
	lane["landing"] = map[string]any{"closes": false, "records_done": false, "body_checked": false, "armed": false,
		"pr_url": "https://forge.example/pr/1" + bad, "merge": "squash" + bad}
	lane["receipts"] = []any{map[string]any{"role": "implementer", "receipt": "lane-1/receipt.json", "model": "Claude:claude-opus-5-5",
		"route": map[string]any{"asked": "claude", "ran": "claude", "model": "claude-opus" + bad}}}
	doc["pending"] = []any{map[string]any{"number": 2, "title": "second" + bad + "step", "needs": []any{}}}
	doc["pick"] = map[string]any{"lane": "lane-1", "entry": "pursued itd-1" + bad,
		"pick":     map[string]any{"chosen": map[string]any{"id": "itd-1"}, "runner_up": nil, "tie_broken_by_age": false, "candidates": []any{}, "rule": "r", "falsifier": "f"},
		"excluded": []any{map[string]any{"id": "itd-2", "check": "run", "reason": "spec says" + bad}}}
	rec := doc["record"].([]any)
	rec[0].(map[string]any)["note"] = "opened" + bad
	doc["transcripts"] = []any{map[string]any{"at": "2026-10-12T10:00:00Z", "path": "~/t" + bad + ".jsonl", "session": "s" + bad,
		"stored": "~/store/s" + bad, "wrote": true, "scan_gap": "gap" + bad}}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestAStateMasksTheTextItStoresWhenShown: a state.json whose stored step
// titles, pending titles, pick entry and exclusion reason, record notes, kept
// reason, sync paths, landing URL and merge rule, a runner's reported model
// and transcripts carry U+202E or U+009B is read, not refused (a run abcd
// wrote stays readable), and the read, the record and the run list carry
// every one of them masked.
func TestAStateMasksTheTextItStoresWhenShown(t *testing.T) {
	repo, runID, _, _ := issueAwaiting(t)
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID)))
	stored := stateBytes(t, repo.Root(), runID)
	for _, bad := range maskForgeRunes {
		if err := os.WriteFile(statePath, forgeStoredText(t, stored, bad), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatalf("%q: a state whose stored text carries the rune is read, not refused: %v", bad, err)
		}
		rec, err := ReadRecord(repo.Root(), runID)
		if err != nil {
			t.Fatalf("%q: the record reads: %v", bad, err)
		}
		runs, err := Runs(repo.Root())
		if err != nil {
			t.Fatalf("%q: the run list reads: %v", bad, err)
		}
		for what, v := range map[string]any{"the read": st, "the record": rec, "the run list": runs} {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if strings.ContainsAny(string(b), strings.Join(maskForgeRunes, "")) {
				t.Errorf("%q, %s: a raw rune survives: %s", bad, what, b)
			}
		}
		if st.Lanes[0].StepTitle != "guard the map?" || st.Lanes[0].Syncs[0].Paths[0] != "docs/?readme.md" ||
			st.Lanes[0].Landing.PRURL != "https://forge.example/pr/1?" || st.Lanes[0].Receipts[0].Route.Model != "claude-opus?" ||
			st.Pick.Excluded[0].Reason != "spec says?" || st.Transcripts[0].Session != "s?" {
			t.Errorf("%q: each forged field reads back masked, its clean text kept: %+v", bad, st.Lanes[0])
		}
	}
}

// TestAStateMasksTheTextItStoresWhenWritten: text a move puts on the state
// with a raw rune is stored masked, and the caller's own state is left as it
// was; a state of clean text is written exactly as before.
func TestAStateMasksTheTextItStoresWhenWritten(t *testing.T) {
	repo, runID, _, _ := issueAwaiting(t)
	clean := stateBytes(t, repo.Root(), runID)
	for _, bad := range maskForgeRunes {
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		st.Lanes[0].StepTitle += bad
		st.Lanes[0].Awaits[0].Kept = "git: refused" + bad
		st.Record = append(st.Record, Entry{At: st.UpdatedAt, Stage: "sync", Note: "conflict in " + bad})
		st.Lanes[0].Syncs = []Sync{{At: st.UpdatedAt, Siblings: []string{}, Conflicted: true, Paths: []string{"a" + bad}}}
		if err := withLock(repo.Root(), func(root *os.Root) error { return writeState(root, st) }); err != nil {
			t.Fatal(err)
		}
		if b := stateBytes(t, repo.Root(), runID); strings.Contains(string(b), bad) || strings.ContainsAny(string(b), "‮\u009b") {
			t.Errorf("%q: the state file stores a raw rune: %s", bad, b)
		}
		if !strings.HasSuffix(st.Lanes[0].StepTitle, bad) || !strings.HasSuffix(st.Lanes[0].Syncs[0].Paths[0], bad) {
			t.Errorf("%q: the write changed the caller's state underneath it", bad)
		}
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	st.Lanes[0].StepTitle = strings.TrimSuffix(strings.TrimSuffix(st.Lanes[0].StepTitle, "?"), "?")
	st.Lanes[0].Awaits[0].Kept = ""
	st.Record = st.Record[:len(st.Record)-2]
	st.Lanes[0].Syncs = nil
	if err := withLock(repo.Root(), func(root *os.Root) error { return writeState(root, st) }); err != nil {
		t.Fatal(err)
	}
	if got := stateBytes(t, repo.Root(), runID); string(got) != string(clean) {
		t.Errorf("a clean state round-trips byte for byte:\n%s\nwant\n%s", got, clean)
	}
}

// TestAScanGapKeepsItsLinesWhenMasked: a transcript's scan gap is the missing
// tools' explanations joined by line breaks, which the record renders one
// line each; masking it keeps those breaks and masks the runes inside a line,
// whether the gap is stored by this abcd or read from a run an earlier one
// wrote.
func TestAScanGapKeepsItsLinesWhenMasked(t *testing.T) {
	gap := "gitleaks is not installed\u009b\nsee the tools registry"
	want := "gitleaks is not installed?\nsee the tools registry"
	if got := (Transcript{ScanGap: gap}).masked().ScanGap; got != want {
		t.Errorf("stored: the scan gap masks to %q, want %q", got, want)
	}
	repo, runID, _, _ := issueAwaiting(t)
	var doc map[string]any
	if err := json.Unmarshal(stateBytes(t, repo.Root(), runID), &doc); err != nil {
		t.Fatal(err)
	}
	doc["transcripts"] = []any{map[string]any{"at": "2026-10-12T10:00:00Z", "path": "~/t.jsonl", "session": "s",
		"stored": "~/store/s", "wrote": true, "scan_gap": gap}}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID))), data, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Transcripts[0].ScanGap; got != want {
		t.Errorf("read: the scan gap masks to %q, want %q", got, want)
	}
}

// TestAStateRedactsTheHomeInItsFreeTextBeforeMaskingIt: a home directory
// whose name carries a rune the mask replaces is redacted from the run's
// notes, kept reasons, fallback details, exclusion reasons and transcripts
// before the mask runs, so a surface's later redaction, which could no longer
// find the masked home, is not what keeps the home out. A step title and the
// pick's entry, each compared with the text it came from, and a lane's
// worktree, a path the loop acts on, are masked and never rewritten.
func TestAStateRedactsTheHomeInItsFreeTextBeforeMaskingIt(t *testing.T) {
	home := filepath.Join(t.TempDir(), "ho\u009bme")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	at := home + "/wt"
	st := State{
		Record:      []Entry{{Note: "git failed in " + at}},
		Fallbacks:   []runner.FallbackReceipt{{Detail: "no runner at " + at}},
		Transcripts: []Transcript{{Path: at, Stored: at, ScanGap: "gitleaks: " + at + "\nnext"}},
		Pick:        &RunPick{Entry: "picked " + at, Excluded: []Excluded{{Reason: "spec at " + at}}},
		Pending:     []PendingStep{{Title: "step " + at}},
		Lanes:       []Lane{{StepTitle: "step " + at, Worktree: at, Awaits: []Await{{Kept: "could not save " + at}}}},
	}
	got := st.masked()
	for what, v := range map[string][2]string{
		"note":             {got.Record[0].Note, "git failed in ~/wt"},
		"fallback detail":  {got.Fallbacks[0].Detail, "no runner at ~/wt"},
		"transcript path":  {got.Transcripts[0].Path, "~/wt"},
		"transcript store": {got.Transcripts[0].Stored, "~/wt"},
		"scan gap":         {got.Transcripts[0].ScanGap, "gitleaks: ~/wt\nnext"},
		"exclusion reason": {got.Pick.Excluded[0].Reason, "spec at ~/wt"},
		"kept reason":      {got.Lanes[0].Awaits[0].Kept, "could not save ~/wt"},
		"pick entry":       {got.Pick.Entry, termsafe.Sanitize("picked " + at)},
		"pending title":    {got.Pending[0].Title, termsafe.Sanitize("step " + at)},
		"step title":       {got.Lanes[0].StepTitle, termsafe.Sanitize("step " + at)},
		"worktree":         {got.Lanes[0].Worktree, termsafe.Sanitize(at)},
	} {
		if v[0] != v[1] {
			t.Errorf("%s: masked to %q, want %q", what, v[0], v[1])
		}
	}
}
