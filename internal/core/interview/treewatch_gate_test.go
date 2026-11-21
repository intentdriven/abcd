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

// gateWords is the plain words a refusal leads with when the pre-push gate
// would now let a push of head through with no preflight.
func gateWords(head string) string {
	return "the pre-push gate would now let a push of " + head[:12] + " through with no preflight run"
}

// TestARouteThatLetsAPushSkipThePreflightStopsTheInterview
// (iss-2610040739124513, iss-2611012144422634): the pre-push gate lets a
// commit through with no new preflight when a receipt for it stands in the
// local tier of any worktree git lists, or when a remote-tracking ref outside
// every receipt-less prefix holds it. So a role that, during a turn, makes
// the gate accept this checkout's HEAD where it did not before is stopped
// after that dispatch, naming in plain words what now lets the push through,
// with nothing more drawn and the answer given before it kept: a worktree
// registered at the role's own turn directory holding a receipt there, a
// receipt planted in a sibling worktree's local tier, and a remote-tracking
// ref planted at HEAD, loose or in the packed refs, from the main worktree
// and from a linked one.
func TestARouteThatLetsAPushSkipThePreflightStopsTheInterview(t *testing.T) {
	receipts := receiptsRel + "/"
	for _, c := range []struct {
		name string
		// write is what the role writes, given the run, its HEAD, the
		// repository's common directory and a sibling worktree; named is
		// what the refusal must name (a suffix of one of its paths).
		write func(r *writtenRun, head, common, lane string) (files map[string]string, named string)
		// listed: git must list the turn directory as a worktree after the
		// run, so the probe is shown to be one git honours.
		listed bool
		// linked runs the interview in a linked worktree.
		linked bool
	}{
		{name: "a worktree registered at the turn directory, holding a receipt", listed: true, write: func(r *writtenRun, head, common, _ string) (map[string]string, string) {
			return map[string]string{
				filepath.Join(common, "worktrees", "planted", "gitdir"):    turnDirToken + "/.git\n",
				filepath.Join(common, "worktrees", "planted", "HEAD"):      head + "\n",
				filepath.Join(common, "worktrees", "planted", "commondir"): "../..\n",
				"turns/" + receipts + head:                                 "commit " + head + "\n",
			}, receipts + head
		}},
		{name: "a receipt in a sibling worktree's local tier", write: func(r *writtenRun, head, _, lane string) (map[string]string, string) {
			return map[string]string{filepath.Join(lane, filepath.FromSlash(receiptsRel), head): "commit " + head + "\n"}, "/lane/" + receipts + head
		}},
		{name: "a receipt in a sibling worktree's local tier, the interview in a linked worktree", linked: true, write: func(r *writtenRun, head, _, lane string) (map[string]string, string) {
			return map[string]string{filepath.Join(lane, filepath.FromSlash(receiptsRel), head): "commit " + head + "\n"}, "/lane/" + receipts + head
		}},
		{name: "a remote-tracking ref planted at HEAD", write: func(r *writtenRun, head, common, _ string) (map[string]string, string) {
			return map[string]string{filepath.Join(common, "refs", "remotes", "origin", "planted"): head + "\n"}, "refs/remotes/origin/planted"
		}},
		{name: "a remote-tracking ref planted at HEAD, the interview in a linked worktree", linked: true, write: func(r *writtenRun, head, common, _ string) (map[string]string, string) {
			return map[string]string{filepath.Join(common, "refs", "remotes", "origin", "planted"): head + "\n"}, "refs/remotes/origin/planted"
		}},
		{name: "a remote-tracking ref planted at HEAD in the packed refs", write: func(r *writtenRun, head, common, _ string) (map[string]string, string) {
			packed, err := os.ReadFile(filepath.Join(common, "packed-refs"))
			if err != nil {
				t.Fatal(err)
			}
			return map[string]string{filepath.Join(common, "packed-refs"): string(packed) + head + " refs/remotes/origin/packed\n"}, "refs/remotes/origin/packed"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			r.git.Git("pack-refs", "--all")
			lane := laneFor(t, r, "lane")
			if c.linked {
				r.w.Repo = laneFor(t, r, "mine")
			}
			head := r.git.Git("rev-parse", "HEAD")
			common := filepath.Join(r.repo, ".git")
			files, named := c.write(r, head, common, lane)
			stubAlso(t, script, 2, files)
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on %s", err, named)
			}
			if !slices.ContainsFunc(uc.Paths, func(got string) bool { return strings.HasSuffix(got, named) }) || !strings.Contains(err.Error(), named) {
				t.Fatalf("the refusal does not name %s: %v", named, err)
			}
			if !strings.Contains(err.Error(), gateWords(head)) {
				t.Fatalf("the refusal does not say %q in plain words: %v", gateWords(head), err)
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
			if c.listed {
				list := r.git.Git("worktree", "list", "--porcelain")
				if !strings.Contains(list, "/"+TurnsRel+"/") {
					t.Fatalf("git does not list the planted worktree, so the probe is not one git honours:\n%s", list)
				}
			}
		})
	}
}

// TestASiblingEntryMadeCarryingConfigurationStopsTheInterview
// (iss-2610040739124513): a worktree entry made beside the checkout during a
// dispatch is a peer's own work only as `git worktree add` makes one, with a
// commondir leading to this checkout's common directory and no configuration
// of its own. One made carrying its own configuration (config.worktree), or
// with a commondir leading to another common directory whose configuration
// git would run there, stops the interview, naming the file, within one turn
// and across two: a sibling's entry the role removes in one turn and makes
// again so in the next is stopped in the next.
func TestASiblingEntryMadeCarryingConfigurationStopsTheInterview(t *testing.T) {
	alias := "[alias]\n\tst = !sh -c true\n"
	for _, c := range []struct {
		name string
		// write is what the role writes in the turn named, given the run,
		// its HEAD and the repository's common directory; named is what the
		// refusal must name.
		write func(r *writtenRun, head, common string) map[string]string
		named string
		// removeFirst removes the sibling entry lane in turn 1, so write
		// makes it again in turn 2.
		removeFirst bool
	}{
		{name: "an entry made with its own configuration", named: ".git/worktrees/planted/config.worktree", write: func(r *writtenRun, head, common string) map[string]string {
			return map[string]string{
				filepath.Join(common, "worktrees", "planted", "gitdir"):          filepath.Join(t.TempDir(), "planted", ".git") + "\n",
				filepath.Join(common, "worktrees", "planted", "HEAD"):            head + "\n",
				filepath.Join(common, "worktrees", "planted", "commondir"):       "../..\n",
				filepath.Join(common, "worktrees", "planted", "config.worktree"): alias,
			}
		}},
		{name: "an entry made with its commondir leading elsewhere", named: ".git/worktrees/planted/commondir", write: func(r *writtenRun, head, common string) map[string]string {
			fake := t.TempDir()
			return map[string]string{
				filepath.Join(fake, "HEAD"):                                "ref: refs/heads/main\n",
				filepath.Join(fake, "config"):                              "[core]\n\trepositoryformatversion = 0\n" + alias,
				filepath.Join(common, "worktrees", "planted", "gitdir"):    filepath.Join(t.TempDir(), "planted", ".git") + "\n",
				filepath.Join(common, "worktrees", "planted", "HEAD"):      head + "\n",
				filepath.Join(common, "worktrees", "planted", "commondir"): fake + "\n",
			}
		}},
		{name: "a sibling's entry removed in one turn and made again with its commondir leading elsewhere in the next", named: ".git/worktrees/lane/commondir", removeFirst: true, write: func(r *writtenRun, head, common string) map[string]string {
			fake := t.TempDir()
			return map[string]string{
				filepath.Join(fake, "HEAD"):                             "ref: refs/heads/main\n",
				filepath.Join(fake, "config"):                           "[core]\n\trepositoryformatversion = 0\n\tfsmonitor = true\n",
				filepath.Join(common, "worktrees", "lane", "gitdir"):    filepath.Join(t.TempDir(), "lane", ".git") + "\n",
				filepath.Join(common, "worktrees", "lane", "HEAD"):      head + "\n",
				filepath.Join(common, "worktrees", "lane", "commondir"): fake + "\n",
			}
		}},
		{name: "a sibling's entry removed in one turn and made again with its own configuration in the next", named: ".git/worktrees/lane/config.worktree", removeFirst: true, write: func(r *writtenRun, head, common string) map[string]string {
			return map[string]string{
				filepath.Join(common, "worktrees", "lane", "gitdir"):          filepath.Join(t.TempDir(), "lane", ".git") + "\n",
				filepath.Join(common, "worktrees", "lane", "HEAD"):            head + "\n",
				filepath.Join(common, "worktrees", "lane", "commondir"):       "../..\n",
				filepath.Join(common, "worktrees", "lane", "config.worktree"): alias,
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			asks := []string{stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone}
			turn, drawn := 2, 1
			if c.removeFirst {
				asks = []string{stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubAsk("Product Q3", "Is the third answer complete?"), stubDone}
				turn, drawn = 3, 2
			}
			script := stubOnPath(t, asks...)
			r := newWrittenRun(t, routedToClaude)
			r.git.Commit("base")
			laneFor(t, r, "lane")
			head := r.git.Git("rev-parse", "HEAD")
			common := filepath.Join(r.repo, ".git")
			if c.removeFirst {
				stubPeer(t, script, 2, r.git.Env(), stubCmd{Dir: r.repo, Script: "rm -rf .git/worktrees/lane"})
			}
			stubAlso(t, script, turn, c.write(r, head, common))
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on %s", err, c.named)
			}
			if !slices.Contains(uc.Paths, c.named) || !strings.Contains(err.Error(), c.named) {
				t.Fatalf("the refusal does not name %s: %v", c.named, err)
			}
			if len(r.asked) != drawn || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != drawn {
				t.Fatalf("record %+v", rec)
			}
		})
	}
}

// TestAPeersFetchOrPreflightDoesNotOpenTheGate (iss-2611012144422634): what
// the gate's judgement reads is this checkout's HEAD commit alone. A peer
// that commits in its own worktree, pushes its branch, fetches, and runs its
// preflight there (minting a receipt for its own HEAD) while the role writes
// the first question leaves the interview running to its outcome, from the
// main worktree and from a linked one, while this checkout's HEAD is a commit
// no remote holds.
func TestAPeersFetchOrPreflightDoesNotOpenTheGate(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "the interview in the main worktree"
		if linked {
			name = "the interview in a linked worktree"
		}
		t.Run(name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Commit("base")
			origin := filepath.Join(t.TempDir(), "origin.git")
			r.git.Git("init", "-q", "--bare", origin)
			r.git.Git("remote", "add", "origin", origin)
			r.git.Git("push", "-q", "origin", "HEAD:refs/heads/main")
			r.git.Git("fetch", "-q", "origin")
			peer := laneFor(t, r, "lane")
			mine := r.repo
			if linked {
				mine = laneFor(t, r, "mine")
				r.w.Repo = mine
			}
			// This checkout's HEAD is a commit of its own, on no remote.
			r.git.Git("-C", mine, "commit", "-q", "--allow-empty", "-m", "mine")
			stubPeer(t, script, 1, r.git.Env(), stubCmd{Dir: peer, Script: fixtureGit + " commit -q --allow-empty -m peer && git push -q origin lane && git fetch -q origin && " + mintReceipt})
			res, err := r.w.Run(context.Background())
			if err != nil {
				t.Fatalf("err = %v, want the interview run to its outcome past a peer's fetch and preflight", err)
			}
			if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
				t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
			}
		})
	}
}
