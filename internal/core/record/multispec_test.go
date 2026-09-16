package record

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The dispatch pages of a multi-spec intent say what is true: the intent lists
// every spec that realises it and points at the OPEN one, and the closed spec
// says the intent stays planned until its sibling closes.
func TestDescribeMultiSpecIntentAndItsClosedSpec(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "planned", "itd-20", "half",
		"---\nid: itd-20\nslug: half\nspec_id: spc-30\nkind: standalone\n---\n\n# H\n\n## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- **Given** x, **then** y.\n"+recordGrounds)
	write(t, repo, ".abcd/development/specs/closed/spc-30-half.md",
		"---\nid: spc-30\nslug: half\nintent: itd-20\n---\n# half\n\nThe delivered half.\n")
	write(t, repo, ".abcd/development/specs/open/spc-31-rest.md",
		"---\nid: spc-31\nslug: rest\nintent: itd-20\n---\n# rest\n\nThe remainder, written and ready to build.\n")

	d, err := Describe(repo, "itd-20")
	if err != nil {
		t.Fatal(err)
	}
	if d.Links["specs"] != "spc-30 (closed), spc-31 (open)" {
		t.Fatalf("intent must list every spec that realises it: %+v", d.Links)
	}
	if !strings.Contains(strings.Join(d.NextMoves, " "), "spec close spc-31") {
		t.Fatalf("the move must name the OPEN spec: %v", d.NextMoves)
	}

	ds, err := Describe(repo, "spc-30")
	if err != nil {
		t.Fatal(err)
	}
	moves := strings.Join(ds.NextMoves, " ")
	if !strings.Contains(moves, "spc-31") {
		t.Fatalf("a closed spec must say which sibling keeps the intent planned: %v", ds.NextMoves)
	}
}

// F7: a spec store that cannot be read is SAID so on the dispatch page. Dropping
// the line silently presents a multi-spec intent as a single-spec one.
func TestDescribeIntentSaysWhenTheSpecStoreIsUnreadable(t *testing.T) {
	repo := t.TempDir()
	intentFixture(t, repo, "shipped", "itd-21", "done",
		"---\nid: itd-21\nslug: done\nspec_id: spc-40\nkind: standalone\nimpact: fix\n---\n\n# D\n")
	// A symlinked bucket is what spec.Load refuses to follow.
	if err := os.MkdirAll(filepath.Join(repo, ".abcd/development/specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repo, filepath.Join(repo, ".abcd/development/specs/open")); err != nil {
		t.Fatal(err)
	}

	d, err := Describe(repo, "itd-21")
	if err != nil {
		t.Fatalf("an unreadable spec store must not fail the whole render: %v", err)
	}
	if got := d.Links["specs"]; !strings.Contains(got, "unreadable") {
		t.Fatalf("the page must say the store could not be read, got %q", got)
	}
}
