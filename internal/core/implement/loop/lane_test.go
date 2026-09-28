package loop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// laneStore is the lane directory of the store for repo, under the test HOME.
func laneStore(t *testing.T, repo *gittest.Repo) string {
	t.Helper()
	sha := strings.TrimSpace(repo.Git("rev-list", "--max-parents=0", "HEAD"))
	return filepath.Join(os.Getenv("HOME"), ".abcd", "worktrees", sha)
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

// TestTheWorktreeStepMakesTheLaneInTheStore is criterion 3's worktree half:
// the lane's worktree exists in abcd's form — the machine-scoped store keyed on
// the root commit, named for the run and the lane — on a branch cut from the
// default branch (not from whatever the checkout has checked out), and nothing
// is made beside the checkout.
func TestTheWorktreeStepMakesTheLaneInTheStore(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	mainSHA := strings.TrimSpace(repo.Git("rev-parse", "main"))
	repo.Git("checkout", "-q", "-b", "elsewhere")
	repo.Write("notes.txt", "work on another branch\n")
	repo.Commit("elsewhere")
	parent := filepath.Dir(repo.Root())
	before := dirNames(t, parent)

	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Performed != StepWorktree || res.Step != StepBrief {
		t.Fatalf("want the worktree step performed: %+v", res)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	wantPath := filepath.Join(laneStore(t, repo), start.RunID+"-lane-1")
	if l.Worktree != wantPath {
		t.Fatalf("the worktree lives in the store: got %q, want %q", l.Worktree, wantPath)
	}
	if l.Branch != "build/"+start.RunID+"-lane-1" {
		t.Fatalf("branch = %q", l.Branch)
	}
	if l.BaseSHA != mainSHA || l.HeadSHA != mainSHA {
		t.Fatalf("the lane is cut from the default branch %s, got base %s head %s", mainSHA, l.BaseSHA, l.HeadSHA)
	}
	if got := strings.TrimSpace(repo.Git("-C", l.Worktree, "rev-parse", "--abbrev-ref", "HEAD")); got != l.Branch {
		t.Fatalf("the worktree has the lane's branch checked out: %q", got)
	}
	if !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "branch refs/heads/"+l.Branch) {
		t.Fatal("git lists the lane's worktree")
	}
	if after := dirNames(t, parent); !slices.Equal(before, after) {
		t.Fatalf("nothing is made beside the checkout: before %v, after %v", before, after)
	}
}

// TestTheWorktreeStepFindsWhatItMadeLastTime is the Handler contract: a process
// killed after the worktree was made and before the state was written runs the
// body again, which adopts the lane's own worktree rather than failing on it or
// making a second.
func TestTheWorktreeStepFindsWhatItMadeLastTime(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	first := st.Lanes[0]
	if _, err := worktreeStep(c, &first); err != nil {
		t.Fatal(err)
	}
	again := st.Lanes[0]
	out, err := worktreeStep(c, &again)
	if err != nil {
		t.Fatalf("the body finds what it made last time: %v", err)
	}
	if again.Worktree != first.Worktree || again.Branch != first.Branch || again.BaseSHA != first.BaseSHA {
		t.Fatalf("the adopted lane is the same lane: %+v vs %+v", again, first)
	}
	if !strings.Contains(out.Note, "found") {
		t.Fatalf("the record says it found the worktree: %q", out.Note)
	}
	if n := strings.Count(repo.Git("worktree", "list", "--porcelain"), "worktree "); n != 2 {
		t.Fatalf("one lane worktree beside the main one, got %d worktrees", n)
	}
}

// TestTheWorktreeStepRefusesAPathThatCouldEscapeTheStore is the trust boundary:
// the path is derived from the run and lane ids, and a lane id that is not the
// loop's own shape — a traversal, a separator, a leading dash, a control byte —
// is refused before any directory is made, in the store or out of it.
func TestTheWorktreeStepRefusesAPathThatCouldEscapeTheStore(t *testing.T) {
	for _, bad := range []string{"../escape", "lane-1/../../x", "-lane-1", "lane-1\x01", "lane-1/x", "..", "lane-01", ""} {
		t.Run(strings.ReplaceAll(bad, "/", "_"), func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			st, err := ReadState(repo.Root(), start.RunID)
			if err != nil {
				t.Fatal(err)
			}
			st.Lanes[0].ID = bad
			root, err := os.OpenRoot(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := writeState(root, st); err != nil {
				t.Fatal(err)
			}
			_, err = Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
			r := mustRefusal(t, err)
			if r.Step != string(StepWorktree) {
				t.Fatalf("want the worktree step to refuse: %+v", r)
			}
			if _, err := os.Lstat(filepath.Join(os.Getenv("HOME"), ".abcd")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("nothing is created under the home for a refused lane: %v", err)
			}
		})
	}
	for _, name := range []string{"../x", "a/b", "-x", ".hidden", "x\x00y", "ok-name"} {
		if got := safeSegment(name); got != (name == "ok-name") {
			t.Fatalf("safeSegment(%q) = %v", name, got)
		}
	}
}

// TestTheWorktreeStepRefusesASymlinkedStore: a symlink standing in for a level
// of the store refuses the step, and nothing is made where it points.
func TestTheWorktreeStepRefusesASymlinkedStore(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(os.Getenv("HOME"), ".abcd", "worktrees")); err != nil {
		t.Fatal(err)
	}
	_, err = Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
	if r := mustRefusal(t, err); r.Step != string(StepWorktree) || !strings.Contains(r.Reason, "real directories") {
		t.Fatalf("want the symlinked store refused: %+v", r)
	}
	if names := dirNames(t, elsewhere); len(names) != 0 {
		t.Fatalf("nothing is made through the symlink: %v", names)
	}
}

// TestTheWorktreeStepNeverAdoptsWhatItDidNotMake: a directory already at the
// lane's path that git does not list as the lane's worktree is refused and left
// exactly as it was.
func TestTheWorktreeStepNeverAdoptsWhatItDidNotMake(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	squat := filepath.Join(laneStore(t, repo), start.RunID+"-lane-1")
	if err := os.MkdirAll(squat, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(squat, "mine.txt"), []byte("the user's\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
	if r := mustRefusal(t, err); r.Step != string(StepWorktree) || !strings.Contains(r.Reason, "occupied") {
		t.Fatalf("want the occupied path refused: %+v", r)
	}
	if names := dirNames(t, squat); !slices.Equal(names, []string{"mine.txt"}) {
		t.Fatalf("the squatter is left as it was: %v", names)
	}
}

// TestTheWorktreeStepRefusesAStoreLevelAnyoneElseCanWrite: a level of the store
// that already exists is held to the caller-alone test the home declarations
// and the harness data directory use (fsutil.CallersAlone), so a level another
// account could write — group- or world-writable — refuses the step before
// anything is made inside it, and the level is left as it was.
func TestTheWorktreeStepRefusesAStoreLevelAnyoneElseCanWrite(t *testing.T) {
	for _, tc := range []struct {
		level string
		mode  os.FileMode
	}{
		{".abcd", 0o777},
		{".abcd/worktrees", 0o777},
		{".abcd/worktrees", 0o770},
		{"<sha>", 0o722},
	} {
		t.Run(strings.ReplaceAll(tc.level, "/", "_")+"-"+tc.mode.String(), func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			store := laneStore(t, repo)
			if err := os.MkdirAll(store, 0o700); err != nil {
				t.Fatal(err)
			}
			level := filepath.Join(os.Getenv("HOME"), filepath.FromSlash(tc.level))
			if tc.level == "<sha>" {
				level = store
			}
			if err := os.Chmod(level, tc.mode); err != nil {
				t.Fatal(err)
			}
			_, err = Advance(repo.Root(), start.RunID, DefaultSteps(), Options{})
			r := mustRefusal(t, err)
			if r.Step != string(StepWorktree) || !strings.Contains(r.Reason, "writable by its group or by every user") {
				t.Fatalf("want the writable store level refused: %+v", r)
			}
			if names := dirNames(t, store); len(names) != 0 {
				t.Fatalf("nothing is made in the store: %v", names)
			}
			if fi, err := os.Lstat(level); err != nil || fi.Mode().Perm() != tc.mode {
				t.Fatalf("the level is left as it was: %v %v", fi, err)
			}
			if st, _ := ReadState(repo.Root(), start.RunID); st.Lanes[0].Step != StepWorktree || st.Lanes[0].Worktree != "" {
				t.Fatalf("the lane stays at its worktree step: %+v", st.Lanes[0])
			}
		})
	}
}

// TestTheWorktreeStepMakesTheStoreTheCallersAlone: every level the step makes
// is 0700, so the store it makes passes its own test on the next call.
func TestTheWorktreeStepMakesTheStoreTheCallersAlone(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Advance(repo.Root(), start.RunID, DefaultSteps(), Options{}); err != nil {
		t.Fatal(err)
	}
	home := os.Getenv("HOME")
	for _, level := range []string{filepath.Join(home, ".abcd"), filepath.Join(home, ".abcd", "worktrees"), laneStore(t, repo)} {
		if fi, err := os.Lstat(level); err != nil || fi.Mode().Perm() != 0o700 {
			t.Fatalf("%s is made 0700: %v %v", level, fi, err)
		}
	}
}

// TestALaneNameRefusalNamesTheIDItRefuses: each refusal of a malformed id
// names that id, quoted, and never carries an id it did not refuse — nor any
// unvalidated id raw into the one-line rendering.
func TestALaneNameRefusalNamesTheIDItRefuses(t *testing.T) {
	for _, tc := range []struct{ runID, laneID, refused string }{
		{"run-bad", "lane-1", "run-bad"},
		{"run-bad", "lane-1\x1b[31m", "run-bad"},
		{"run-0123456789012345", "lane-01", "lane-01"},
		{"run-0123456789012345", "lane-1\x1b[31m", "lane-1\x1b[31m"},
	} {
		_, err := laneName(tc.runID, tc.laneID)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("%q", tc.refused)) {
			t.Fatalf("laneName(%q, %q): the refusal names %q: %+v", tc.runID, tc.laneID, tc.refused, r)
		}
		if r.Lane != "" {
			t.Fatalf("laneName(%q, %q): the refusal carries no lane it did not refuse: %q", tc.runID, tc.laneID, r.Lane)
		}
		if strings.ContainsRune(r.Error(), '\x1b') {
			t.Fatalf("laneName(%q, %q): no raw control byte is rendered: %q", tc.runID, tc.laneID, r.Error())
		}
	}
}
