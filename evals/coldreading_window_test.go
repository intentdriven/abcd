//go:build smoke || coldreading

package evals

// The window eval: every committed preset entry is re-measured on every
// committed change, and an entry that has drifted past the window it declares
// fails here rather than reaching a reader (spc-2609020626048722, "The eval, and
// what keeps it from passing vacuously").
//
// It joins the lane with no Makefile or workflow edit, exactly as the harness
// README says a later eval does. What it delivers is what the intent claims and
// no more: drift past a window is caught by the cold-reading eval lane, which is
// not yet a required check (iss-2608311632382737, iss-2608311051046981).
//
// The oracle rule holds. `bannedImports` already refuses `internal/core/reading`,
// so this file parses the committed preset with its own minimal struct and reads
// the measurement out of the binary's JSON: the eval's account of the
// declaration is independent of the assembler's.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// windowPositions are the positions this eval measures, named here rather than
// derived. The comparative position is exempt BY NAME: its object is bounded by
// the widening run it is handed rather than by the tree, so a window over the
// tree measures nothing about it, and the comparative channel's own eval covers
// it (spc-2609020626048722).
var windowPositions = []string{"widening", "entailment", "detection"}

// windowBreach is one entry measured past its declaration.
type windowBreach struct {
	Position  string
	TokensEst int
	Bytes     int
	Declared  int
}

func (b windowBreach) String() string {
	return fmt.Sprintf("%s: measured ~%d estimated tokens over %d bytes against a declaration "+
		"of %d", b.Position, b.TokensEst, b.Bytes, b.Declared)
}

// TestEveryCommittedEntryFitsItsDeclaredWindow is spc-2609020626048722's ac-3:
// each of the three assembling positions' committed entries measures at or below
// the window it declares, over a clean clone of HEAD.
func TestEveryCommittedEntryFitsItsDeclaredWindow(t *testing.T) {
	clone := cloneHeadDetached(t)
	breaches := windowBreaches(t, clone)
	if len(breaches) == 0 {
		return
	}
	lines := make([]string, 0, len(breaches))
	for _, b := range breaches {
		lines = append(lines, b.String())
	}
	t.Fatalf("%d committed entry(s) measure past the window they declare:\n%s\n\n"+
		"A declaration is what the entry was calibrated for. Re-measure by dry run and move "+
		"the declaration in %s, or narrow the entry's kinds — either is a commit that records "+
		"why.", len(breaches), strings.Join(lines, "\n"), presetConfigRel)
}

// TestTheWindowCheckReportsABreach is ac-4 and this eval's negative control: a
// declaration lowered to one below the figure just measured must fail, naming
// the position, the measured figure and the declaration.
//
// `windowBreaches` is the one function both tests call, so a check that stopped
// comparing would take this control down with it rather than leaving it green.
//
// What the control asserts is the EFFECT OF ITS OWN MUTATION and nothing about
// the state of the rest of the file. It once asserted that the lowered file
// produced exactly one breach, which is a claim about every entry: any unrelated
// drift past a declaration — the very condition
// TestEveryCommittedEntryFitsItsDeclaredWindow exists to report — took this
// control down with it, so on the day a real breach appeared the check that
// names it and the control that proves the check works failed together and
// neither could be read as evidence about the other. The baseline is therefore
// measured before the lowering, and every position but the target must be found
// exactly as it was left.
func TestTheWindowCheckReportsABreach(t *testing.T) {
	clone := cloneHeadDetached(t)

	// The baseline: which positions already breach, before this test changes
	// anything. Through windowBreaches, so the before and the after are one
	// function's account and a difference between them is a difference in the
	// file rather than between two notions of a breach.
	before := breachedPositions(windowBreaches(t, clone))

	const target = "widening"
	// The target is measured first, so the lowered declaration is one below a
	// real figure rather than an invented one.
	targetTokens := assembledTokens(t, clone, target)

	raw, err := os.ReadFile(filepath.Join(clone, filepath.FromSlash(presetConfigRel)))
	if err != nil {
		t.Fatalf("read the clone's preset file: %v", err)
	}
	var file map[string]json.RawMessage
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode the clone's preset file: %v", err)
	}
	var positions map[string]map[string]json.RawMessage
	if err := json.Unmarshal(file["positions"], &positions); err != nil {
		t.Fatalf("decode the clone's position entries: %v", err)
	}
	entry, ok := positions[target]
	if !ok {
		t.Fatalf("the committed file names no entry for %s, so there is nothing to lower", target)
	}
	lowered := targetTokens - 1
	entry["window"] = json.RawMessage(fmt.Sprintf(
		`{"tokens_est": %d, "measured_tokens_est": %d, "measured_bytes": 1, "measured_at": "0000000"}`,
		lowered, lowered))
	positions[target] = entry
	repacked, err := json.Marshal(positions)
	if err != nil {
		t.Fatalf("re-encode the position entries: %v", err)
	}
	file["positions"] = repacked
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatalf("re-encode the preset file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(clone, filepath.FromSlash(presetConfigRel)), out, 0o644); err != nil {
		t.Fatalf("write the lowered preset file: %v", err)
	}
	// Committed in the clone under the fixture identity, so the dirty gate and
	// the tracked-file check are satisfied by construction rather than skipped.
	gitInClone(t, clone, "add", presetConfigRel)
	gitInClone(t, clone, "-c", "user.name=abcd eval", "-c", "user.email=eval@example.invalid",
		"commit", "-q", "-m", "lower one declaration")

	breaches := windowBreaches(t, clone)
	after := breachedPositions(breaches)
	if !after[target] {
		t.Fatalf("a declaration one below the measured figure produced no breach at %s; the "+
			"breaches were %v", target, breaches)
	}
	// And nothing else moved. This is the half that says the breach belongs to
	// the lowering rather than to whatever else the file happens to hold: a
	// position that already breached still does, and one that did not still does
	// not.
	for _, position := range windowPositions {
		if position == target {
			continue
		}
		if after[position] != before[position] {
			t.Errorf("lowering the %s declaration changed whether %s breaches (before %v, "+
				"after %v); the mutation is confined to one entry and the check must be too",
				target, position, before[position], after[position])
		}
	}
	var b windowBreach
	for _, x := range breaches {
		if x.Position == target {
			b = x
		}
	}
	if b.Declared != lowered {
		t.Errorf("the breach names the declaration %d, want %d", b.Declared, lowered)
	}
	if b.TokensEst <= b.Declared || b.Bytes <= 0 {
		t.Errorf("the breach names the measured figure ~%d over %d bytes, which does not "+
			"exceed the declaration %d", b.TokensEst, b.Bytes, b.Declared)
	}
}

// breachedPositions reduces a breach list to the set of positions in it, so a
// control can compare the state before its own mutation with the state after it
// without asserting anything about how many entries breach in total.
func breachedPositions(breaches []windowBreach) map[string]bool {
	out := make(map[string]bool, len(breaches))
	for _, b := range breaches {
		out[b.Position] = true
	}
	return out
}

// presetConfigRel is the committed preset configuration, spelled here rather
// than imported: the oracle rule keeps this package out of the assembler's.
const presetConfigRel = ".abcd/config/reading-presets.json"

// windowBreaches assembles the committed entry at each assembling position by
// dry run and returns those measuring past their declaration.
//
// Non-vacuity: the number of positions measured must equal the number of
// entries the file declares at the three named positions, and zero is refused.
// A file that lost an entry, or a run that silently measured nothing, fails here
// rather than returning an empty breach list that reads as a pass.
func windowBreaches(t *testing.T, root string) []windowBreach {
	t.Helper()
	declared := declaredWindows(t, root)
	if len(declared) == 0 {
		t.Fatalf("%s declares no window at any of %s, so this eval would pass over nothing",
			presetConfigRel, strings.Join(windowPositions, ", "))
	}
	var breaches []windowBreach
	measured := 0
	for _, position := range windowPositions {
		want, ok := declared[position]
		if !ok {
			t.Errorf("%s declares no window at the %s position", presetConfigRel, position)
			continue
		}
		size := assembledSize(t, root, position)
		measured++
		if size.Window == nil || size.Window.TokensEst != want {
			t.Errorf("the binary reports the declaration at %s as %v and this eval reads %d "+
				"from the file; the report echoes the committed entry and cannot invent one",
				position, size.Window, want)
			continue
		}
		if size.TokensEst > want {
			breaches = append(breaches, windowBreach{
				Position: position, TokensEst: size.TokensEst, Bytes: size.Bytes, Declared: want,
			})
		}
	}
	if measured != len(declared) {
		t.Fatalf("this eval measured %d position(s) and the file declares %d at %s; an entry "+
			"the eval never assembled is one nothing holds to its declaration",
			measured, len(declared), strings.Join(windowPositions, ", "))
	}
	return breaches
}

// declaredWindows parses the committed preset file with this eval's own minimal
// struct, so the declaration it holds an entry to is read from the file rather
// than from the code under test.
func declaredWindows(t *testing.T, root string) map[string]int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(presetConfigRel)))
	if err != nil {
		t.Fatalf("read %s: %v", presetConfigRel, err)
	}
	var file struct {
		SchemaVersion int `json:"schema_version"`
		Positions     map[string]struct {
			Window *struct {
				TokensEst int `json:"tokens_est"`
			} `json:"window"`
		} `json:"positions"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode %s: %v", presetConfigRel, err)
	}
	if file.SchemaVersion != 2 {
		t.Fatalf("%s is at schema version %d; a declared window is a version 2 shape",
			presetConfigRel, file.SchemaVersion)
	}
	out := map[string]int{}
	for _, position := range windowPositions {
		e, ok := file.Positions[position]
		if !ok || e.Window == nil {
			continue
		}
		out[position] = e.Window.TokensEst
	}
	return out
}

// sizeJSON is the part of an assembly's JSON result this eval reads.
type sizeJSON struct {
	Bytes     int `json:"bytes"`
	TokensEst int `json:"tokens_est"`
	Window    *struct {
		TokensEst int `json:"tokens_est"`
	} `json:"window"`
	ExceedsWindow bool `json:"exceeds_window"`
	OverTarget    bool `json:"over_target"`
}

// assembledSize dry-runs one position through the built binary and returns its
// size report.
func assembledSize(t *testing.T, root, position string) sizeJSON {
	t.Helper()
	home := t.TempDir()
	out, code := runIn(t, root, []string{"HOME=" + home},
		"reading", "assemble", "--position", position, "--target", "HEAD", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("`abcd reading assemble --position %s` exited %d:\n%s", position, code, out)
	}
	var res struct {
		Size sizeJSON `json:"size"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("decode the assembly result at %s: %v\n%s", position, err, out)
	}
	if res.Size.TokensEst <= 0 {
		t.Fatalf("the assembly at %s reports ~%d estimated tokens; a run that measured nothing "+
			"cannot be held to a declaration", position, res.Size.TokensEst)
	}
	return res.Size
}

// assembledTokens is assembledSize's total, for the negative control.
func assembledTokens(t *testing.T, root, position string) int {
	t.Helper()
	return assembledSize(t, root, position).TokensEst
}

// cloneHeadDetached clones the module root into a temporary directory and checks
// out its HEAD sha detached, so the measured tree is the checkout's COMMIT with
// no uncommitted edit in it: the assembler's dirty gate and tracked-file check
// are then satisfied by construction rather than skipped.
func cloneHeadDetached(t *testing.T) string {
	t.Helper()
	head := gitInClone(t, "..", "rev-parse", "HEAD")
	dir := filepath.Join(t.TempDir(), "clone")
	gitInClone(t, ".", "clone", "--quiet", "--no-local", "..", dir)
	gitInClone(t, dir, "checkout", "--quiet", "--detach", head)
	return dir
}

// gitInClone runs one git command under the repository's hermetic-git
// environment, failing the test on error.
func gitInClone(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}
