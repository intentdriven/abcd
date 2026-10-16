package ahoy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// TestOutboundForgeIsOptInAndIdempotent — iss-2610142210511831. A managed
// repository has the text its agents hand the forge judged by the shell guard
// only when the person asks: refusing a posting command changes how work in
// the repository goes, which is the repository's choice, never abcd's
// default. The choice is one key in config.json, which the guard reads, and
// since iss-2610152024327989 the one outbound switch records it beside the
// commit-msg hook's. A later plain install keeps it.
func TestOutboundForgeIsOptInAndIdempotent(t *testing.T) {
	newRepo := func(t *testing.T) string {
		t.Helper()
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		return repo
	}

	t.Run("never recorded unasked", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		if guard.ForgeTextOptedIn(repo) {
			t.Error("a plain install opted the repository into the forge-text check")
		}
	})

	t.Run("opting in on an already-installed repo, then a plain install keeps it", func(t *testing.T) {
		setupHermetic(t)
		repo := newRepo(t)
		if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		res, err := Install(repo, outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if !guard.ForgeTextOptedIn(repo) {
			t.Fatalf("--outbound recorded no forge-text opt-in on an already-installed repo (status=%q writes=%v)", res.Status, res.Writes)
		}
		before := treeHash(t, repo)
		res, err = Install(repo, installOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "already_up_to_date" {
			t.Errorf("re-install status = %q, want already_up_to_date", res.Status)
		}
		if msg, ok := sameTree(before, treeHash(t, repo)); !ok {
			t.Errorf("re-install changed the tree: %s", msg)
		}
		if !guard.ForgeTextOptedIn(repo) {
			t.Error("a plain install dropped the recorded opt-in")
		}
	})
}
