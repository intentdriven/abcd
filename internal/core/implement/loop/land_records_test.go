package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestARecordsCommitIsAdoptedOnlyForWhatALandingMoves: a commit on the judged
// head carrying the landing's subject is taken as the records commit a killed
// call made only when its content could be one: the records this landing names,
// and a repointed link elsewhere under the record tree. Anything else is
// refused, and the head stays the judged one.
func TestARecordsCommitIsAdoptedOnlyForWhatALandingMoves(t *testing.T) {
	type edit struct {
		name  string
		write func(t *testing.T, f *landFixture, l Lane)
	}
	commit := func(t *testing.T, f *landFixture, l Lane, name, body string) {
		t.Helper()
		abs := filepath.Join(l.Worktree, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		f.repo.Git("-C", l.Worktree, "add", "--", name)
		f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", landSubject(st, l))
	}
	for _, c := range []edit{
		{"a source file", func(t *testing.T, f *landFixture, l Lane) { commit(t, f, l, "cmd/unjudged.go", "package main\n") }},
		{"a file under the record tree that is no record", func(t *testing.T, f *landFixture, l Lane) {
			commit(t, f, l, ".abcd/config.json", "{}\n")
		}},
		{"a page under the record tree whose text changed", func(t *testing.T, f *landFixture, l Lane) {
			commit(t, f, l, ".abcd/development/plans/a-plan.md", "unjudged text\n")
		}},
		{"a merge", func(t *testing.T, f *landFixture, l Lane) {
			f.repo.Git("-C", l.Worktree, "checkout", "-q", "-b", "side", l.BaseSHA)
			if err := os.WriteFile(filepath.Join(l.Worktree, "side.go"), []byte("package side\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			f.repo.Git("-C", l.Worktree, "add", "--", "side.go")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "side")
			f.repo.Git("-C", l.Worktree, "checkout", "-q", l.Branch)
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			f.repo.Git("-C", l.Worktree, "merge", "-q", "--no-ff", "-m", landSubject(st, l), "side")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validated(t)
			f.step(t)
			l = currentLane(t, f.repo, f.runID)
			judged := l.HeadSHA
			c.write(t, f, l)
			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if err == nil || !strings.Contains(err.Error(), "not the landing's records commit") {
				t.Fatalf("the commit is refused: %v", err)
			}
			if got := currentLane(t, f.repo, f.runID); got.HeadSHA != judged || (got.Landing != nil && got.Landing.RecordsDone) {
				t.Fatalf("the head stays the judged one: %+v", got)
			}
		})
	}
	t.Run("a repointed link", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.validated(t)
		page := ".abcd/development/plans/a-plan.md"
		l := currentLane(t, f.repo, f.runID)
		f.step(t)
		l = currentLane(t, f.repo, f.runID)
		judged := l.HeadSHA
		// A page holding a link, committed as judged work, then the link's move.
		commit(t, f, l, page, "see ../../work/issues/open/iss-1-a.md\nsee [a](itd-1-a.md)\n")
		base := strings.TrimSpace(f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
		if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[0].HeadSHA = base
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
		l = currentLane(t, f.repo, f.runID)
		commit(t, f, l, page, "see ../../work/issues/resolved/iss-1-a.md\nsee [a](../shipped/itd-1-a.md)\n")
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			t.Fatalf("a link's move is what a landing makes: %v", err)
		}
		if got := currentLane(t, f.repo, f.runID); got.HeadSHA == judged || got.HeadSHA == base {
			t.Fatalf("the commit is taken: %+v", got)
		}
	})
}
