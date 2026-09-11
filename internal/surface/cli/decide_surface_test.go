package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decide"
)

// decideRepo lays out an ADR store already holding the hand-numbered ordinals,
// so the surface test proves the WIRING end to end against the shape a real
// checkout has.
//
// It is a git working tree, not a bare temporary directory: a directory outside
// every repository is no longer a place a decision is minted, because the front
// door resolves the checkout root and refuses when there is none
// (iss-2609091707224329).
func decideRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	gitInitAt(t, root)
	root = realPath(t, root)
	abs := filepath.Join(root, filepath.FromSlash(decide.ADRsRelDir), "0058-a-reading-is-commissioned.md")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: adr-58\nslug: a-reading-is-commissioned\nstatus: accepted\n---\n\n# ADR-58: A reading is commissioned\n"
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestDecideWiredFromTheCLI is the wiring proof: the verb reaches decide.Create
// from the front door, mints a timestamp id rather than the next ordinal, and
// reports the id and the path in both renders.
func TestDecideWiredFromTheCLI(t *testing.T) {
	repo := decideRepo(t)
	t.Chdir(repo)

	out := runCLI(t, "decide", "Decision records mint through the timestamp seam", "--json")
	var res decide.Decision
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode result: %v\n%s", err, out)
	}
	if !regexp.MustCompile(`^adr-[0-9]{16}$`).MatchString(res.ID) {
		t.Fatalf("minted id %q is not the timestamp shape", res.ID)
	}
	if res.ID == "adr-59" {
		t.Fatal("the verb allocated the next ordinal instead of stamping the clock")
	}
	// The literal directory, not decide.ADRsRelDir: a check written against the
	// constant follows the constant wherever it moves, so it could never notice
	// the record landing outside the decisions store.
	const wantDir = ".abcd/development/decisions/adrs/"
	if !strings.HasPrefix(res.Path, wantDir) {
		t.Errorf("path = %q, want a record under %s", res.Path, wantDir)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(res.Path))); err != nil {
		t.Errorf("the reported record is not on disk: %v", err)
	}

	// The text render is what a human sees; it must name the id and the path.
	text := string(runCLI(t, "decide", "A second decision on the same day"))
	if !strings.Contains(text, "adr-") || !strings.Contains(text, ".abcd/development/decisions/adrs/") {
		t.Errorf("text render is missing the id or the path:\n%s", text)
	}
}

// TestDecideRefusalsExitTwo proves the operand faults reach the operator as a
// diagnostic rather than a written record: no title, and a title with nothing
// slug-able in it.
func TestDecideRefusalsExitTwo(t *testing.T) {
	repo := decideRepo(t)
	t.Chdir(repo)

	for _, tc := range []struct{ name, arg string }{
		{"empty", ""},
		{"no-slugable-characters", "///"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if out, err := runCLIErr(t, "decide", tc.arg); err == nil {
				t.Fatalf("decide %q must refuse, got: %s", tc.arg, out)
			}
		})
	}
	entries, err := os.ReadDir(filepath.Join(repo, filepath.FromSlash(decide.ADRsRelDir)))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("a refused decide wrote a record: %d entries in the store", len(entries))
	}
}
