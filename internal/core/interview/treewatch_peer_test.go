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

// fixtureGit is the identity a stubbed peer's git commands commit under, as
// the fixture's own Git pins it per command.
const fixtureGit = "git -c user.email=fixture@example.invalid -c user.name=Fixture -c commit.gpgsign=false"

// laneFor adds a linked worktree on its own branch beside r's checkout, with
// a local tier, and returns its path.
func laneFor(t *testing.T, r *writtenRun, branch string) string {
	t.Helper()
	lane := filepath.Join(t.TempDir(), branch)
	r.git.Git("worktree", "add", "-q", "-b", branch, lane)
	if err := os.MkdirAll(filepath.Join(lane, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	return lane
}

// TestAPeersRefChangesInAnotherWorktreeDoNotStopTheInterview
// (iss-2611012144422634): every worktree of a checkout shares one git
// directory, so a peer session committing in its own worktree moves a branch
// ref the interview's checkout shares but does not hold. The interview's
// tripwire is scoped to its own checkout's branch, so a peer's ordinary
// commit, a record branch it writes by plumbing, or its packing the refs,
// while the role writes the first question, leaves the interview running to
// its outcome, from the checkout's main worktree and from a linked one.
func TestAPeersRefChangesInAnotherWorktreeDoNotStopTheInterview(t *testing.T) {
	for _, c := range []struct {
		name string
		// peer is what the peer runs while the role runs, given the
		// interview's checkout and the peer's worktree.
		peer func(mine, peer string) []stubCmd
		// linked runs the interview in a linked worktree, the peer in the
		// main one.
		linked bool
	}{
		{name: "a commit on the peer's worktree's branch", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: fixtureGit + " commit -q --allow-empty -m peer"}}
		}},
		{name: "a commit on the main worktree's branch, the interview in a linked one", linked: true, peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: fixtureGit + " commit -q --allow-empty -m peer"}}
		}},
		{name: "a record branch written by plumbing", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: `c=$(` + fixtureGit + ` commit-tree "$(git rev-parse 'HEAD^{tree}')" -p HEAD -m record) && git update-ref refs/heads/records/planning-1101 "$c"`}}
		}},
		{name: "the refs packed, the interview's branch among them", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: "git pack-refs --all"}}
		}},
		{name: "a branch deleted from the packed refs", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: "git branch -q records/old && git pack-refs --all && git branch -q -D records/old"}}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			r.git.Commit("base")
			mine, peer := r.repo, laneFor(t, r, "lane")
			if c.linked {
				mine, peer = laneFor(t, r, "mine"), r.repo
				r.w.Repo = mine
			}
			stubPeer(t, script, 1, r.git.Env(), c.peer(mine, peer)...)
			res, err := r.w.Run(context.Background())
			if err != nil {
				t.Fatalf("err = %v, want the interview run to its outcome past a peer's work in its own worktree", err)
			}
			if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
				t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
			}
		})
	}
}

// TestAChangeTheInterviewMustNotIgnoreStillStopsIt (iss-2611012144422634):
// the scoping keeps what the interview's own contract decides. Its own
// checkout's branch moving while the role runs (a commit there, from the
// main worktree or a linked one) stops it, the branch named in plain words
// with where it moved from and to; and a record it does not grant, edited
// beside the one it does, stops it naming that record alone.
func TestAChangeTheInterviewMustNotIgnoreStillStopsIt(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "the main worktree's branch moved"
		if linked {
			name = "a linked worktree's branch moved"
		}
		t.Run(name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			r.git.Commit("base")
			mine, branch := r.repo, "main"
			if linked {
				mine, branch = laneFor(t, r, "mine"), "mine"
				r.w.Repo = mine
			}
			from := r.git.Git("rev-parse", "refs/heads/"+branch)
			stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: mine, Script: fixtureGit + " commit -q --allow-empty -m moved"})
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on its own branch moving", err)
			}
			if len(uc.Paths) != 1 || !strings.HasSuffix(uc.Paths[0], ".git/refs/heads/"+branch) {
				t.Fatalf("paths %v, want the branch's ref alone", uc.Paths)
			}
			to := r.git.Git("rev-parse", "refs/heads/"+branch)
			want := "the branch " + branch + " this checkout is on moved from " + from[:12] + " to " + to[:12]
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("the refusal does not say %q in plain words: %v", want, err)
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
		})
	}
	t.Run("a record the interview does not grant", func(t *testing.T) {
		script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
		r := newWrittenRun(t, routedToClaude)
		granted, other := ".abcd/development/intents/planned/itd-1-one.md", ".abcd/development/intents/planned/itd-2-two.md"
		r.git.Write(granted, "one\n")
		r.git.Write(other, "two\n")
		r.git.Commit("records")
		r.w.MayChange = []string{granted}
		stubAlso(t, script, 2, map[string]string{granted: "one, edited\n", other: "two, edited\n"})
		_, err := r.w.Run(context.Background())
		var uc *UnexpectedChangesError
		if !errors.As(err, &uc) || !slices.Equal(uc.Paths, []string{other}) || !strings.Contains(err.Error(), other) {
			t.Fatalf("err = %v, want the interview stopped naming %s alone", err, other)
		}
	})
}
