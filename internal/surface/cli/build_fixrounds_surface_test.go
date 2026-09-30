package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ruling DR1 (2026-09-29) at the surface: `abcd build --fix-rounds <n>` sets
// the fix rounds a lane may take before it is handed back, beside the pace,
// bundled 3, and a malformed value is refused naming the accepted form.

// TestBuildTakesTheFixRoundCapBesideThePace: with no flag the run's pace
// carries the bundled 3 fix rounds, --fix-rounds wins and is named in the
// text, and --fix-rounds three is refused at the pace stage writing nothing.
func TestBuildTakesTheFixRoundCapBesideThePace(t *testing.T) {
	repo := buildRepo(t)
	out := mustImplement(t, "build", "itd-10", "--json")
	var res struct {
		Pace struct {
			FixRounds struct {
				Value int
				Layer string
			} `json:"fix_rounds"`
		} `json:"pace"`
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Pace.FixRounds.Value != 3 || res.Pace.FixRounds.Layer != "bundled" {
		t.Fatalf("the bundled cap is 3 fix rounds: %s", out)
	}
	status := mustImplement(t, "implement", "status")
	if !strings.Contains(status, "3 fix rounds before a lane is handed back (bundled)") {
		t.Fatalf("the status names the cap and its layer:\n%s", status)
	}

	ref := refusalDocs(t, 2, "build", "itd-10", "--json", "--fix-rounds", "4")
	if ref["stage"] != "pace" || !strings.Contains(ref["reason"].(string), "--fix-rounds 4") {
		t.Fatalf("a resume naming another cap is refused: %v", ref)
	}
	resetRuns(t, repo.Root())
	code, text, errOut := implementCLI(t, "build", "itd-10", "--fix-rounds", "1")
	if code != 0 || !strings.Contains(text, "1 fix round before a lane is handed back (--fix-rounds 1, the flag layer)") {
		t.Fatalf("--fix-rounds wins and is named: exit %d\n%s\n%s", code, text, errOut)
	}
	resetRuns(t, repo.Root())
	ref = refusalDocs(t, 2, "build", "itd-10", "--json", "--fix-rounds", "three")
	if ref["stage"] != "pace" || !strings.Contains(ref["reason"].(string), "three") || !strings.Contains(ref["remedy"].(string), "--fix-rounds") {
		t.Fatalf("a malformed cap is refused naming the accepted form: %v", ref)
	}
	runDirAbsent(t, repo.Root())
}

// resetRuns removes the checkout's run tier, so the next build starts a run.
func resetRuns(t *testing.T, root string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(root, ".abcd", ".work.local", "run")); err != nil {
		t.Fatal(err)
	}
}
