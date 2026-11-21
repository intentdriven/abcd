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

// TestASiblingWorktreesOwnWorkDoesNotStopTheInterview: a sibling worktree's
// per-worktree state (its HEAD, its rebase's files, its entry made or
// removed) and the push receipts in its local tier are not the interview's:
// a peer switching branch, rebasing or running the preflight (which mints a
// push receipt in its own local tier) in another worktree, or adding one,
// while the role writes the first question, leaves the interview running to
// its outcome, from the checkout's main worktree and from a linked one. A
// receipt in the checkout's own local tier is watched
// (TestAPushReceiptInThisCheckoutsOwnTierStopsTheInterview).
func TestASiblingWorktreesOwnWorkDoesNotStopTheInterview(t *testing.T) {
	// commitFile commits a file named n in the peer's worktree.
	commitFile := func(n string) string {
		return "printf '" + n + "\\n' > " + n + ".txt && git add " + n + ".txt && " + fixtureGit + " commit -q -m " + n
	}
	for _, c := range []struct {
		name string
		// peer is what runs while the role runs, given the interview's
		// checkout and the sibling worktree.
		peer func(mine, peer string) []stubCmd
	}{
		{name: "a sibling worktree switched to another branch", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: "git checkout -q -b elsewhere"}}
		}},
		// The rebase stops after its first commit, as one stopped on a
		// conflict does, so the sibling's HEAD stands detached when the
		// role's turn ends.
		{name: "a sibling worktree's branch being rebased", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: commitFile("a") + " && " + commitFile("b") + " && { " + fixtureGit + " rebase -q --force-rebase --exec false HEAD~2 >/dev/null 2>&1; test -d \"$(git rev-parse --git-path rebase-merge)\"; }"}}
		}},
		{name: "a push receipt minted in a sibling worktree's local tier", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: mintReceipt}}
		}},
		{name: "a worktree added beside the checkout", peer: func(_, peer string) []stubCmd {
			return []stubCmd{{Dir: peer, Script: "git worktree add -q -b added ../added"}}
		}},
	} {
		for _, linked := range []bool{false, true} {
			name := c.name + ", the interview in the main worktree"
			if linked {
				name = c.name + ", the interview in a linked worktree"
			}
			t.Run(name, func(t *testing.T) {
				script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
				r := newWrittenRun(t, routedToClaude)
				ignoring(".abcd/.work.local/\n")(r)
				mine, peer := r.repo, laneFor(t, r, "lane")
				if linked {
					mine, peer = laneFor(t, r, "mine"), r.repo
					r.w.Repo = mine
				}
				stubPeer(t, script, 1, r.git.Env(), c.peer(mine, peer)...)
				res, err := r.w.Run(context.Background())
				if err != nil {
					t.Fatalf("err = %v, want the interview run to its outcome past a sibling worktree's own work", err)
				}
				if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
					t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
				}
			})
		}
	}
}

// mintReceipt mints a push receipt for HEAD in the local tier of the
// worktree it runs in, as scripts/preflight-receipt.sh does.
const mintReceipt = "mkdir -p .abcd/.work.local/preflight-receipts && h=$(git rev-parse HEAD) && printf 'commit %s\\n' \"$h\" > .abcd/.work.local/preflight-receipts/\"$h\""

// TestAPushReceiptInThisCheckoutsOwnTierStopsTheInterview: the pre-push gate
// lets a commit through on a receipt in the pushing checkout's local tier, so
// a receipt written there while the role runs would let a push skip the
// preflight. It stops the interview, from the main worktree and from a
// linked one, naming the receipt, with nothing more drawn and the answer
// given before it kept. A receipts directory that is a link is read where it
// leads, so a receipt written there stops it too, named in full.
func TestAPushReceiptInThisCheckoutsOwnTierStopsTheInterview(t *testing.T) {
	for _, c := range []struct {
		name   string
		linked bool
		// outside, when set, makes the checkout's receipts directory a link
		// to a directory outside the tree before the run.
		outside bool
	}{
		{name: "a receipt in the main worktree's own tier"},
		{name: "a receipt in a linked worktree's own tier", linked: true},
		{name: "a receipt where the own tier's receipts directory leads", outside: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			mine := r.repo
			laneFor(t, r, "lane")
			if c.linked {
				mine = laneFor(t, r, "mine")
				r.w.Repo = mine
			}
			head := r.git.Git("rev-parse", "HEAD")
			named := ".abcd/.work.local/preflight-receipts/" + head
			if c.outside {
				dir, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(mine, ".abcd", ".work.local"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(dir, filepath.Join(mine, filepath.FromSlash(".abcd/.work.local/preflight-receipts"))); err != nil {
					t.Fatal(err)
				}
				named = filepath.ToSlash(filepath.Join(dir, head))
			}
			stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: mine, Script: mintReceipt})
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) || !slices.Contains(uc.Paths, named) || !strings.Contains(err.Error(), named) {
				t.Fatalf("err = %v, want the interview stopped naming %s", err, named)
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

// TestThisCheckoutSwitchingBranchStillStopsTheInterview: the checkout's own
// HEAD decides what its next commit extends, so this checkout switched to
// another branch while the role runs stops the interview, from the main
// worktree and from a linked one, saying so in plain words.
func TestThisCheckoutSwitchingBranchStillStopsTheInterview(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "the main worktree switched"
		if linked {
			name = "a linked worktree switched"
		}
		t.Run(name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			r.git.Commit("base")
			// A sibling stands beside the checkout, as in a shared repository.
			laneFor(t, r, "lane")
			mine, branch := r.repo, "main"
			if linked {
				mine, branch = laneFor(t, r, "mine"), "mine"
				r.w.Repo = mine
			}
			stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: mine, Script: "git checkout -q -b elsewhere"})
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on its own HEAD moving", err)
			}
			if !slices.ContainsFunc(uc.Paths, func(p string) bool { return strings.HasSuffix(p, "HEAD") }) {
				t.Fatalf("paths %v, want the checkout's HEAD named", uc.Paths)
			}
			if want := "this checkout was switched from the branch " + branch + " to the branch elsewhere"; !strings.Contains(err.Error(), want) {
				t.Fatalf("the refusal does not say %q: %v", want, err)
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
