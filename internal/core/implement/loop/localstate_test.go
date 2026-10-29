package loop

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestEveryRunStateACommitCarriesIsRefused: the run state, the series state
// and a run's configuration fingerprint are this checkout's own, so each one
// git tracks came with a commit and is refused as that, naming the file, before
// it is parsed as anything.
func TestEveryRunStateACommitCarriesIsRefused(t *testing.T) {
	const runID = "run-2610280000000001"
	for name, c := range map[string]struct {
		rel  string
		read func(root string) error
	}{
		"the run state":    {StateRelPath(runID), func(root string) error { _, err := ReadState(root, runID); return err }},
		"the series state": {NextSeriesRel, func(root string) error { _, _, err := readSeries(root); return err }},
		"a fingerprint": {fingerprintRel(runID), func(root string) error {
			r, err := os.OpenRoot(root)
			if err != nil {
				return err
			}
			defer r.Close()
			_, _, err = readFingerprint(r, runID)
			return err
		}},
	} {
		t.Run(name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write(".gitignore", ".abcd/.work.local/\n")
			repo.Write(c.rel, "{}\n")
			repo.Commit("seed")
			if err := c.read(repo.Root()); err != nil && strings.Contains(err.Error(), "git tracks") {
				t.Fatalf("an untracked file is refused as tracked: %v", err)
			}
			repo.Git("add", "-f", "--", c.rel)
			repo.Git("commit", "-q", "-m", "an unrelated change")
			r := mustRefusal(t, c.read(repo.Root()))
			if !strings.Contains(r.Reason, "git tracks "+c.rel) || !strings.Contains(r.Remedy, "git rm --cached") {
				t.Fatalf("want a refusal naming the tracked file and how to untrack it: %+v", r)
			}
		})
	}
}

// TestADrainStateUnderATierSpelledAnotherWayIsRefused: a commit that carries
// .abcd/.Work.Local/run/drain.json lands in the tier itself on a filesystem
// that folds case, and the drain refuses it there, by whatever spelling git
// holds it; so it does when .abcd itself is a link a commit carries.
func TestADrainStateUnderATierSpelledAnotherWayIsRefused(t *testing.T) {
	for name, entry := range map[string]struct{ mode, path string }{
		"a capital":        {"100644", ".abcd/.Work.Local/run/drain.json"},
		"a link for .abcd": {"120000", ".abcd"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := gittest.NewRepo(t)
			repo.Write(".gitignore", ".abcd/.work.local/\n")
			repo.Write(DrainStateRel, "{}\n")
			repo.Commit("seed")
			blob := repo.Git("rev-parse", "HEAD:.gitignore")
			repo.Git("update-index", "--add", "--cacheinfo", entry.mode+","+blob+","+entry.path)
			_, _, err := readDrain(repo.Root())
			r := mustRefusal(t, err)
			if r.Stage != StageDrain || !strings.Contains(r.Reason, "git tracks "+entry.path) {
				t.Fatalf("want the drain to refuse its state, naming %s: %+v", entry.path, r)
			}
		})
	}
}

// TestARunListingBesideATrackedSiblingListsTheRuns: a file git tracks beside
// the runs (the drain's state in the run tier) supplies none of them, so the
// listing stands; one under a run refuses it.
func TestARunListingBesideATrackedSiblingListsTheRuns(t *testing.T) {
	const runID = "run-2610280000000001"
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(StateRelPath(runID), "{}\n")
	repo.Write(RunRelDir+"/drain.json", "{}\n")
	repo.Commit("seed")
	repo.Git("add", "-f", "--", RunRelDir+"/drain.json")
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if ids, err := runIDs(root); err != nil || len(ids) != 1 || ids[0] != runID {
		t.Fatalf("runIDs beside a tracked sibling = %v, %v; want [%s]", ids, err, runID)
	}
	repo.Git("add", "-f", "--", StateRelPath(runID))
	if _, err := runIDs(root); err == nil || !strings.Contains(err.Error(), "git tracks "+StateRelPath(runID)) {
		t.Fatalf("runIDs over a tracked run = %v; want it refused naming the run's state", err)
	}
}
