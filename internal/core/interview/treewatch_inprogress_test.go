package interview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestAnOperationInProgressWrittenInTheOwnGitDirStopsTheInterview: git keeps
// a rebase, an am, a cherry-pick or revert sequence, and a merge in progress
// as files in the checkout's own git directory, and the person's next
// `--continue`, `--abort` or commit runs what they say: a todo line "exec
// <cmd>" runs a command, MERGE_HEAD makes the next commit a merge. None is
// listed by git status, so a role that writes one while it runs is stopped
// after that dispatch, naming it, from the main worktree (where its git
// directory is the common one) and from a linked one (its own entry under
// worktrees/).
func TestAnOperationInProgressWrittenInTheOwnGitDirStopsTheInterview(t *testing.T) {
	for _, c := range []struct {
		name string
		// linked runs the interview in a linked worktree.
		linked bool
		// rel is the path the role writes under the checkout's own git
		// directory, and body its content; head in body is HEAD's commit.
		rel, body string
	}{
		{name: "a rebase todo with an exec line", rel: "rebase-merge/git-rebase-todo", body: "exec touch pwned\n"},
		{name: "a rebase todo with an exec line, the interview in a linked worktree", linked: true, rel: "rebase-merge/git-rebase-todo", body: "exec touch pwned\n"},
		{name: "an am in progress", rel: "rebase-apply/next", body: "1\n"},
		{name: "a sequencer todo", rel: "sequencer/todo", body: "exec touch pwned\n"},
		{name: "MERGE_HEAD", rel: "MERGE_HEAD", body: "head\n"},
		{name: "MERGE_HEAD, the interview in a linked worktree", linked: true, rel: "MERGE_HEAD", body: "head\n"},
		{name: "CHERRY_PICK_HEAD", rel: "CHERRY_PICK_HEAD", body: "head\n"},
		{name: "REVERT_HEAD", rel: "REVERT_HEAD", body: "head\n"},
		{name: "MERGE_MSG", rel: "MERGE_MSG", body: "a message the next commit takes\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			mine := r.repo
			if c.linked {
				mine = laneFor(t, r, "mine")
				r.w.Repo = mine
			}
			gitDir := r.git.Git("-C", mine, "rev-parse", "--absolute-git-dir")
			head := r.git.Git("-C", mine, "rev-parse", "HEAD")
			written := filepath.Join(gitDir, filepath.FromSlash(c.rel))
			stubAlso(t, script, 2, map[string]string{written: strings.ReplaceAll(c.body, "head", head)})
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on %s", err, c.rel)
			}
			if !slices.ContainsFunc(uc.Paths, func(got string) bool { return strings.HasSuffix(got, c.rel) }) {
				t.Fatalf("the refusal does not name %s: %v", c.rel, err)
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
		})
	}
}

// TestAPeersOperationInProgressDoesNotStopTheInterview: a rebase, or a merge
// left for its commit, that a peer starts in its own worktree writes that
// worktree's git directory, which is the peer's own work: started while the
// role writes the first question, it leaves the interview running to its
// outcome, the peer in a linked worktree beside the main one, and in the
// main worktree beside a linked one (where its state stands in the common
// directory itself).
func TestAPeersOperationInProgressDoesNotStopTheInterview(t *testing.T) {
	// breakFirst is a rebase that stops before its first step, leaving
	// rebase-merge/ in the peer's git directory.
	const breakFirst = `GIT_SEQUENCE_EDITOR="printf 'break\\n' >" git rebase -q -i HEAD~1`
	const mergeLeft = fixtureGit + ` merge -q --no-ff --no-commit side`
	for _, c := range []struct {
		name string
		// linked runs the interview in a linked worktree, the peer in the
		// main one.
		linked bool
		peer   string
		// left is what the peer's operation leaves in its git directory.
		left string
	}{
		{name: "a rebase in a linked worktree", peer: breakFirst, left: "rebase-merge"},
		{name: "a rebase in the main worktree, the interview in a linked one", linked: true, peer: breakFirst, left: "rebase-merge"},
		{name: "a merge left for its commit in a linked worktree", peer: mergeLeft, left: "MERGE_HEAD"},
		{name: "a merge left for its commit in the main worktree, the interview in a linked one", linked: true, peer: mergeLeft, left: "MERGE_HEAD"},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			r.git.Commit("second")
			r.git.Git("branch", "side")
			r.git.Git("commit", "-q", "--allow-empty", "-m", "main's own")
			r.git.Git("checkout", "-q", "side")
			r.git.Git("commit", "-q", "--allow-empty", "-m", "side's own")
			r.git.Git("checkout", "-q", "-")
			peer := r.repo
			if c.linked {
				r.w.Repo = laneFor(t, r, "mine")
			} else {
				peer = laneFor(t, r, "lane")
			}
			stubPeer(t, script, 1, r.git.Env(), stubCmd{Dir: peer, Script: c.peer})
			res, err := r.w.Run(context.Background())
			if err != nil {
				t.Fatalf("err = %v, want the interview run to its outcome past a peer's %s", err, c.left)
			}
			if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
				t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
			}
			p := r.git.Git("-C", peer, "rev-parse", "--git-path", c.left)
			if !filepath.IsAbs(p) {
				p = filepath.Join(peer, p)
			}
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("the peer's %s is not there (%v), so the case proves nothing", c.left, err)
			}
		})
	}
}
