package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/release"
)

// intent_target_cli_test.go — itd-2609212103572513 at the front door: the
// target verb and plan's --target write the field, each refusal exits 2 with
// nothing written, and the preview and the cut list the targeted intent in
// text, --json and the pre-flight report without refusing on it.

// TestIntentTargetAtTheCLI is criterion 1 at the CLI.
func TestIntentTargetAtTheCLI(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	draftBefore, _ := os.ReadFile(filepath.Join(repo, cliDrafts, "itd-10-alpha.md"))

	// A draft is sent to plan --target, exit 2, nothing written.
	if _, err := runCLIErr(t, "intent", "target", "itd-10", "v0.11.0"); err == nil || exitCodeOf(err) != 2 ||
		!strings.Contains(err.Error(), "abcd intent plan itd-10 --target") {
		t.Fatalf("target on a draft must refuse on exit 2 naming plan --target: exit = %d (%v)", exitCodeOf(err), err)
	}
	if after, _ := os.ReadFile(filepath.Join(repo, cliDrafts, "itd-10-alpha.md")); string(after) != string(draftBefore) {
		t.Fatal("a refused target wrote the draft")
	}
	// A bundle takes no --target: each member's target is its own.
	writeRepoFile(t, repo, cliDrafts+"/itd-11-beta.md", cliDraftWithAC("itd-11", "beta"))
	if _, err := runCLIErr(t, "intent", "plan", "itd-10", "itd-11", "--bundle", "ab", "--target", "next"); err == nil || exitCodeOf(err) != 2 ||
		!strings.Contains(err.Error(), "abcd intent target") {
		t.Fatalf("plan --bundle --target must refuse on exit 2 naming the target verb: exit = %d (%v)", exitCodeOf(err), err)
	}
	// An illegal value is refused by plan before anything moves.
	if _, err := runCLIErr(t, "intent", "plan", "itd-10", "--target", "soon"); err == nil || exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "vX.Y.Z") {
		t.Fatalf("plan --target soon must refuse on exit 2: exit = %d (%v)", exitCodeOf(err), err)
	}

	// plan --target plans the draft and writes the target in the same write.
	out := string(runCLI(t, "intent", "plan", "itd-10", "--target", "v0.11.0"))
	if !strings.Contains(out, "drafts -> planned") || !strings.Contains(out, "target stamped: v0.11.0") {
		t.Fatalf("plan --target render:\n%s", out)
	}
	planned := filepath.Join(repo, cliPlanned, "itd-10-alpha.md")
	if body, _ := os.ReadFile(planned); !strings.Contains(string(body), "\ntarget_release: v0.11.0\n") {
		t.Fatalf("the planned record must carry the target:\n%s", body)
	}

	// The target verb replaces it and says what it replaced.
	raw := runCLI(t, "intent", "target", "itd-10", "next", "--json")
	var res struct {
		IntentID string `json:"intent_id"`
		Target   string `json:"target_release"`
		Previous string `json:"previous"`
		Written  bool   `json:"written"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("target --json: %v\n%s", err, raw)
	}
	if res.IntentID != "itd-10" || res.Target != "next" || res.Previous != "v0.11.0" || !res.Written {
		t.Fatalf("target payload = %+v", res)
	}
	text := string(runCLI(t, "intent", "target", "itd-10", "next"))
	if !strings.Contains(text, "abcd intent target — itd-10 already targets next; nothing written") {
		t.Fatalf("an unchanged target must say nothing was written:\n%s", text)
	}
	text = string(runCLI(t, "intent", "target", "itd-10", "v0.12.0"))
	if !strings.Contains(text, "abcd intent target — itd-10 targets v0.12.0 (was next)") {
		t.Fatalf("target render:\n%s", text)
	}
	if _, err := runCLIErr(t, "intent", "target", "itd-10"); err == nil || exitCodeOf(err) != 2 {
		t.Fatalf("target with no version must refuse on exit 2: exit = %d (%v)", exitCodeOf(err), err)
	}
}

// TestLaunchPreviewAndCutListTheTargetedIntent is criterion 2 at the front
// door: the preview and the cut list a targeted planned intent in text, in
// --json and in the preview's pre-flight report, and the cut proceeds.
func TestLaunchPreviewAndCutListTheTargetedIntent(t *testing.T) {
	r := shipRenderableRepo(t)
	r.Write(".abcd/development/intents/planned/itd-91-targeted.md",
		"---\nid: itd-91\nimpact: additive\ntarget_release: v0.4.1\n---\n# Targeted\n")
	r.Commit("an intent names the release it must land by")

	out, err := shipIn(t, r, "launch", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	var rep launch.DryRunReport
	if err := json.Unmarshal(out, &rep); err != nil {
		t.Fatalf("dry-run JSON: %v\n%s", err, out)
	}
	if len(rep.Targets) != 1 || rep.Targets[0].ID != "itd-91" || rep.Targets[0].Target != "v0.4.1" {
		t.Fatalf("the preview must list the targeted intent: %+v", rep.Targets)
	}
	if written := readPreflight(t, r.Root(), rep.ReportPath); len(written.Targets) != 1 {
		t.Fatalf("the preview's pre-flight report must carry the targeted intent: %+v", written.Targets)
	}
	plain, err := shipIn(t, r, "launch", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, plain)
	}
	if !strings.Contains(string(plain), "targeted:       itd-91 targets v0.4.1, not shipped") {
		t.Errorf("the plain preview must list the targeted intent:\n%s", plain)
	}

	// The cut: the emit step lists it and stays ready (exit 0).
	emitted, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("a target must not refuse the cut: exit = %d\n%s", code, emitted)
	}
	if !strings.Contains(string(emitted), "targeted:   itd-91 targets v0.4.1, not shipped") {
		t.Errorf("the emit step must list the targeted intent:\n%s", emitted)
	}
	js, err := shipIn(t, r, "launch", "ship", "--json")
	if err != nil {
		t.Fatalf("emit --json: %v\n%s", err, js)
	}
	var cut release.Cut
	if err := json.Unmarshal(js, &cut); err != nil {
		t.Fatalf("emit JSON: %v\n%s", err, js)
	}
	if !cut.Ready || len(cut.Targets) != 1 || cut.Targets[0].ID != "itd-91" {
		t.Fatalf("the cut JSON must list the targeted intent and stay ready: ready=%v %+v", cut.Ready, cut.Targets)
	}

	// The ingest that writes the release reports the same list.
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	shipped, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("the cut must proceed with a target unshipped: exit = %d\n%s", code, shipped)
	}
	if !strings.Contains(string(shipped), "targeted:   itd-91 targets v0.4.1, not shipped") || !strings.Contains(string(shipped), "wrote:") {
		t.Errorf("the written cut must list the targeted intent:\n%s", shipped)
	}
	// Criterion 3 (ruling BS1 of 2026-09-29): the cut passed the target, so
	// the same write moves it to `next` and the report says so.
	if !strings.Contains(string(shipped), "moved:      itd-91 targets next (targeted v0.4.1)") {
		t.Errorf("the written cut must report the moved target:\n%s", shipped)
	}
	rec, err := os.ReadFile(filepath.Join(r.Root(), ".abcd/development/intents/planned/itd-91-targeted.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rec), "target_release: next\n") {
		t.Errorf("the cut must rewrite the missed target to next:\n%s", rec)
	}
}

// targetedLines returns every `targeted:` line of a cut render, trimmed, so an
// assertion judges each line whole rather than a substring of the report.
func targetedLines(out []byte) map[string]string {
	lines := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "targeted:")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		id, _, _ := strings.Cut(rest, " ")
		lines[id] = rest
	}
	return lines
}

// TestTheCutsDryRunNamesTheTargetsItMoves is iss-2610020718369838 at the
// front door: the cut's dry run — the emit step of the ship verb and the
// read-only changelog preview, which share one render — marks the target the
// cut passes as moving to next, and leaves the target past the cut unmarked,
// so a reader no longer compares each target with the derived tag by hand.
// The JSON carries the same list.
func TestTheCutsDryRunNamesTheTargetsItMoves(t *testing.T) {
	r := shipRenderableRepo(t)
	r.Write(".abcd/development/intents/planned/itd-91-due.md",
		"---\nid: itd-91\nimpact: additive\ntarget_release: v0.4.1\n---\n# Due\n")
	r.Write(".abcd/development/intents/planned/itd-93-later.md",
		"---\nid: itd-93\nimpact: additive\ntarget_release: v0.5.0\n---\n# Later\n")
	r.Commit("one target the cut reaches, one past it")

	const moved = "itd-91 targets v0.4.1, not shipped (.abcd/development/intents/planned/itd-91-due.md); the cut moves it to next"
	const stands = "itd-93 targets v0.5.0, not shipped (.abcd/development/intents/planned/itd-93-later.md)"
	for _, args := range [][]string{{"launch", "ship"}, {"changelog"}} {
		out, err := shipIn(t, r, args...)
		if code := exitCodeOf(err); code != 0 {
			t.Fatalf("%v: a target must not refuse the cut: exit = %d\n%s", args, code, out)
		}
		lines := targetedLines(out)
		if lines["itd-91"] != moved {
			t.Errorf("%v: the target the cut reaches must be named as moving:\n got %q\nwant %q", args, lines["itd-91"], moved)
		}
		if lines["itd-93"] != stands {
			t.Errorf("%v: the target past the cut must stay unmarked:\n got %q\nwant %q", args, lines["itd-93"], stands)
		}
	}

	js, err := shipIn(t, r, "launch", "ship", "--json")
	if err != nil {
		t.Fatalf("emit --json: %v\n%s", err, js)
	}
	var cut release.Cut
	if err := json.Unmarshal(js, &cut); err != nil {
		t.Fatalf("emit JSON: %v\n%s", err, js)
	}
	if len(cut.Moves) != 1 || cut.Moves[0].ID != "itd-91" || cut.Moves[0].From != "v0.4.1" {
		t.Errorf("the cut JSON must carry the one move: %+v", cut.Moves)
	}
}
