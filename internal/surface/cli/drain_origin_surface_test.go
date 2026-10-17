package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestDrainDryRunNamesWhatOriginFinished is spc-2610161044461448 step 1 at the
// front door: in a checkout behind origin, `drain --dry-run` plans from origin's
// default branch as last fetched, names the issue open here that origin has
// resolved, the view's age and the commits behind, in text and in --json, and
// writes nothing.
func TestDrainDryRunNamesWhatOriginFinished(t *testing.T) {
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	done := captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	open := captureWithRemedy(t, "a flaky timeout in the parser test", "--category", "bug", "--remedy", "raise the timeout")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "the ledger")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	gitCmd(t, repo, "remote", "add", "origin", bare)
	gitCmd(t, repo, "push", "-q", "origin", "main")

	// Another account resolves one issue on origin; this checkout fetches
	// origin's branch but never merges it.
	other := filepath.Join(t.TempDir(), "other")
	gitCmd(t, filepath.Dir(other), "clone", "-q", bare, other)
	matches, _ := filepath.Glob(filepath.Join(other, ".abcd", "work", "issues", "open", done+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("no record of %s on origin", done)
	}
	if err := os.MkdirAll(filepath.Join(other, ".abcd", "work", "issues", "resolved"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, other, "mv", matches[0], filepath.Join(other, ".abcd", "work", "issues", "resolved", filepath.Base(matches[0])))
	gitCommit(t, other, "commit", "-q", "-m", "resolve "+done)
	gitCmd(t, other, "push", "-q", "origin", "main")
	gitCmd(t, repo, "fetch", "-q", "origin")

	before := gitCmd(t, repo, "status", "--porcelain", "--untracked-files=all")
	text := string(runCLI(t, "drain", "--dry-run"))
	for _, want := range []string{"origin/main", "1 commit(s) behind", done, "finished on origin/main", open, "drawn at this copy's first move"} {
		if !strings.Contains(text, want) {
			t.Errorf("the dry-run text does not carry %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "eligible   "+done) {
		t.Errorf("the dry run lists %s as eligible, which origin resolved:\n%s", done, text)
	}

	var out struct {
		DryRun                bool `json:"dry_run"`
		OrderDrawnAtFirstMove bool `json:"order_drawn_at_first_move"`
		Dispositions          []struct {
			ID string `json:"id"`
		} `json:"dispositions"`
		Origin *struct {
			Ref      string   `json:"ref"`
			Behind   int      `json:"behind"`
			Finished []string `json:"finished"`
		} `json:"origin"`
	}
	if err := json.Unmarshal(runCLI(t, "drain", "--dry-run", "--json"), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OrderDrawnAtFirstMove {
		t.Fatal("the --json preview of a copy with no key does not say its order is drawn at its first move")
	}
	if out.Origin == nil || out.Origin.Ref != "origin/main" || out.Origin.Behind != 1 || !slices.Equal(out.Origin.Finished, []string{done}) {
		t.Fatalf("the --json origin = %+v; want origin/main, one commit behind, %s finished", out.Origin, done)
	}
	if slices.ContainsFunc(out.Dispositions, func(d struct {
		ID string `json:"id"`
	}) bool {
		return d.ID == done
	}) {
		t.Fatalf("the --json plan lists %s", done)
	}
	if after := gitCmd(t, repo, "status", "--porcelain", "--untracked-files=all"); after != before {
		t.Fatalf("the dry run wrote: before %q, after %q", before, after)
	}
}
