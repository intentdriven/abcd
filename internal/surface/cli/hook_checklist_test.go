package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/checklist"
)

// hook_checklist_test.go — the checklist re-injected at every session start
// (spc-2610230805429429 step 3, technical criteria 1 and 3, story 2): every
// source injects it, its budget is what the rest of the hook's one output
// leaves of the host's cap, a failing file falls back to a scanned last good
// copy with one line saying why, and both failing inject that line alone.

// checklistIntro is the render's first line over a checklist it can show.
const checklistIntro = "This working copy's checklist (`abcd checklist` lists every field):"

// hookStart runs `abcd hook session-start` for source in root and answers its
// stdout, failing on a non-zero exit: a checklist never fails a start.
func hookStart(t *testing.T, root, source string) string {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"session_id": "s-" + source, "cwd": root, "source": source, "hook_event_name": "SessionStart"})
	so, se, code := runSessionStart(string(b), "hook", "session-start")
	if code != 0 {
		t.Fatalf("session-start %s: exit %d\nstdout: %s\nstderr: %s", source, code, so, se)
	}
	return so
}

// checklistPart is the hook's output from the checklist's first line on, and
// what came before it.
func checklistPart(stdout, firstLine string) (rest, part string) {
	i := strings.Index(stdout, firstLine)
	if i < 0 {
		return stdout, ""
	}
	return stdout[:i], stdout[i:]
}

// assertNoTokenRun is the step 1 tests' strict no-echo check: no run of four
// of the token's characters, its prefix included, appears in text.
func assertNoTokenRun(t *testing.T, what, text, tok string) {
	t.Helper()
	for i := 0; i+4 <= len(tok); i++ {
		if strings.Contains(text, tok[i:i+4]) {
			t.Errorf("%s repeats part of the secret (%d..%d)", what, i, i+4)
			return
		}
	}
}

// writeItems writes n open items with long titles through the checked write.
func writeItems(t *testing.T, root string, n int) {
	t.Helper()
	c := checklist.Checklist{Schema: checklist.Schema, UpdatedAt: "2026-10-23T08:00:00Z", UpdatedBy: testSessionID}
	for i := 0; i < n; i++ {
		c.Items = append(c.Items, checklist.Item{
			ID: fmt.Sprintf("item-%d", i), Title: fmt.Sprintf("Item %d ", i) + strings.Repeat("x", 150), State: checklist.Working,
		})
	}
	if err := checklist.Write(root, c); err != nil {
		t.Fatal(err)
	}
}

func TestSessionStartInjectsTheChecklistForEverySource(t *testing.T) {
	root := checklistCheckout(t)
	mustChecklist(t, "add", "Land step three")
	mustChecklist(t, "add", "Wait for the review", "--waits-on", "pr:42")
	want := mustChecklist(t, "render")
	for _, source := range []string{"startup", "resume", "clear", "compact"} {
		so := hookStart(t, root, source)
		_, part := checklistPart(so, checklistIntro)
		// The checklist is the output's last part, and it is the verb's own
		// text: the verb and the hook print one render for one budget.
		if part != want {
			t.Errorf("%s: the hook's checklist is\n%q\nwant the render verb's\n%q", source, part, want)
		}
	}
}

// TestSessionStartChecklistBudgetIsWhatTheWholeOutputLeaves: the budget is the
// host's 10,000 characters for the whole hook output, less what the away block
// and the update notices take, less a margin of 500. Both of them stay whole,
// the checklist yields, and the output stays under the cap.
func TestSessionStartChecklistBudgetIsWhatTheWholeOutputLeaves(t *testing.T) {
	repo, _, _ := afterUpdateRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(checklist.TierRelPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	gitAt(t, repo, "config", "--local", "user.name", "Alice")
	writeItems(t, repo, 200)

	alone := hookStart(t, repo, "startup")
	setAway(t, "--until", "09:00")
	so := hookStart(t, repo, "startup")

	if len(so) >= sessionStartOutputCap {
		t.Fatalf("the hook's output is %d bytes, at or over the host's %d", len(so), sessionStartOutputCap)
	}
	rest, part := checklistPart(so, checklistIntro)
	assertAwayBlock(t, "session-start with a checklist", rest)
	if !strings.Contains(rest, theUpdateNewsLine) {
		t.Fatalf("the update news line is not whole before the checklist:\n%s", rest)
	}
	if part == "" || !strings.Contains(part, "left out; run `abcd checklist` to see them all.") {
		t.Fatalf("an over-budget checklist was not cut with a count:\n%s", part)
	}
	// Computed against the whole output: the checklist fills what the rest
	// left, within the margin, and no more.
	budget := sessionStartOutputCap - sessionStartMargin - len(rest)
	if len(part) > budget {
		t.Errorf("the checklist is %d bytes, over the %d the rest of the output left", len(part), budget)
	}
	if len(part) < budget-200 {
		t.Errorf("the checklist is %d bytes, well short of the %d it was left", len(part), budget)
	}
	// The away block's bytes came out of the checklist's share.
	_, alonePart := checklistPart(alone, checklistIntro)
	if len(part) >= len(alonePart) {
		t.Errorf("with the away block present the checklist kept %d bytes, not fewer than the %d it had alone", len(part), len(alonePart))
	}
}

func TestSessionStartFallsBackToTheLastGoodCopyWithoutEchoing(t *testing.T) {
	tok := secretToken()
	for name, plant := range map[string]func(c checklist.Checklist) []byte{
		"corrupt": func(checklist.Checklist) []byte { return []byte(`{"schema":"abcd/checklist/v1","items":[`) },
		"secret": func(c checklist.Checklist) []byte {
			c.Items[0].Title = "paste " + tok
			b, _ := json.Marshal(c)
			return b
		},
		"secret key": func(checklist.Checklist) []byte { return []byte(`{"` + tok + `":1,"schema":"abcd/checklist/v1"}`) },
	} {
		t.Run(name, func(t *testing.T) {
			root := checklistCheckout(t)
			mustChecklist(t, "add", "Kept item")
			c := checklist.Read(root).Checklist
			path := filepath.Join(root, filepath.FromSlash(checklist.FileRelPath))
			if err := os.WriteFile(path, plant(c), 0o600); err != nil {
				t.Fatal(err)
			}
			so := hookStart(t, root, "compact")
			_, part := checklistPart(so, "Showing the last good copy, because checklist.json ")
			if part == "" || !strings.Contains(part, "] kept-item: Kept item") {
				t.Fatalf("the hook did not inject the last good copy with its reason:\n%s", so)
			}
			if name == "secret" && !strings.Contains(part, "items[0].title") {
				t.Errorf("the reason does not name the field:\n%s", part)
			}
			assertNoTokenRun(t, "the hook's output", so, tok)
		})
	}
}

func TestSessionStartBothFailingInjectsOnlyTheLine(t *testing.T) {
	tok := secretToken()
	root := checklistCheckout(t)
	bad := fmt.Sprintf(`{"schema":%q,"updated_at":"2026-10-23T08:00:00Z","updated_by":%q,"items":[]}`, checklist.Schema, tok)
	for _, rel := range []string{checklist.FileRelPath, checklist.LastGoodRelPath} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	so := hookStart(t, root, "startup")
	want := checklist.Read(root).Reason + "\n"
	if !strings.HasPrefix(want, "No checklist to show: ") {
		t.Fatalf("the reader did not fail both files: %q", want)
	}
	if _, part := checklistPart(so, "No checklist to show: "); part != want {
		t.Errorf("the hook's checklist part is %q, want the line alone %q", part, want)
	}
	if strings.Contains(so, checklistIntro) {
		t.Errorf("both failing still injected a list:\n%s", so)
	}
	assertNoTokenRun(t, "the hook's output", so, tok)
}

// TestSessionStartWithNoChecklistInjectsNothing: the spec re-injects "a
// session with a checklist" (the intent's second criterion); one that never
// wrote a checklist is given nothing extra, not a line saying so.
func TestSessionStartWithNoChecklistInjectsNothing(t *testing.T) {
	root := checklistCheckout(t)
	so := hookStart(t, root, "startup")
	if strings.Contains(strings.ToLower(so), "checklist") {
		t.Errorf("a working copy with no checklist was given one:\n%s", so)
	}
}
