package loop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// One unreadable sibling worktree (spc-2610161044461448 step 9;
// iss-2610161015397586, dcn-2610161040378713): the drain refuses the move
// once, where per-issue contention would begin, naming the worktree and the
// repair its reason calls for, rather than reading the worktree as a holder of
// every issue and passing each over as contended. A worktree another tool has
// locked, and that still reads, is no such worktree.

// absPathRe matches an absolute path where one starts in a message: at the
// start, or after a space, a quote, a bracket, a colon or an equals sign.
var absPathRe = regexp.MustCompile(`(^|[\s'"(\[:=])/[^\s'"]`)

// breakGitFile points a worktree's git file at a directory that does not
// exist, as a moved repository leaves it, and returns where it points.
func breakGitFile(t *testing.T, wt string) string {
	t.Helper()
	nowhere := filepath.Join(t.TempDir(), "nowhere")
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+nowhere+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return nowhere
}

// TestAnUnreadableWorktreeRefusesTheDrainMoveOnce: a sibling worktree whose
// git file points nowhere refuses the move naming it, with its repair, which
// is a plain `git worktree repair` and never a forced removal; the reason
// carries no absolute path, not even the one git names the git file pointing
// at; the refusal is not contention, asks no judgement, reserves nothing,
// starts no run and writes no drain state; once the repair the remedy names is
// run the same drain judges its first issue.
func TestAnUnreadableWorktreeRefusesTheDrainMoveOnce(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	broken := filepath.Join(t.TempDir(), "wt-broken")
	repo.Git("worktree", "add", "-q", "-b", "lane-broken", broken)
	nowhere := breakGitFile(t, broken)

	for move := 1; move <= 2; move++ {
		res, err := Drain(repo.Root(), Options{}, DrainOptions{})
		r := mustRefusal(t, err)
		if r.Stage != StageDrain || r.Check != CheckPeers || r.Contention {
			t.Fatalf("move %d: want the drain's own peers refusal, not contention: %+v", move, r)
		}
		if !strings.Contains(r.Reason, "the worktree at wt-broken (branch lane-broken) could not be read") {
			t.Errorf("move %d: the refusal names the worktree: %q", move, r.Reason)
		}
		if strings.Count(r.Reason, "wt-broken (branch") != 1 || strings.Contains(r.Reason, secondIssue) || strings.Contains(r.Reason, eligibleIssue) {
			t.Errorf("move %d: the worktree is named once, and no issue is: %q", move, r.Reason)
		}
		if absPathRe.MatchString(r.Reason) || strings.Contains(r.Reason, filepath.Dir(nowhere)) {
			t.Errorf("move %d: the reason carries no absolute path: %q", move, r.Reason)
		}
		if !strings.Contains(r.Reason, "nowhere") {
			t.Errorf("move %d: the reason still says where the git file points, by its name: %q", move, r.Reason)
		}
		if !strings.Contains(r.Remedy, "`git worktree repair`") {
			t.Errorf("move %d: the remedy is a plain git worktree repair, with no argument: %q", move, r.Remedy)
		}
		for _, never := range []string{"--force", "git worktree remove", "git worktree repair wt-broken"} {
			if strings.Contains(r.Remedy, never) {
				t.Errorf("move %d: the remedy never advises %q for a broken git file: %q", move, never, r.Remedy)
			}
		}
		if res.Judging != nil || len(res.Passed) != 0 {
			t.Fatalf("move %d: no judgement is asked and no issue is passed over: %+v", move, res)
		}
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a refused move writes no drain state: %v", err)
	}
	for _, id := range []string{secondIssue, eligibleIssue} {
		if sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id); sha != "" {
			t.Fatalf("a refused move reserves nothing: %s holds %s", id, sha)
		}
	}
	if n := runCount(t, repo); n != 0 {
		t.Fatalf("a refused move starts no run: %d", n)
	}

	// The repair the remedy names, run as written in the repository, is the
	// whole fix. git rewrites the broken git file and still exits 1 for having
	// found it broken, so the repair is proved by the worktree answering again,
	// not by its exit status.
	cmd := exec.Command("git", "worktree", "repair")
	cmd.Dir, cmd.Env = repo.Root(), gittest.Env(t)
	_ = cmd.Run()
	gitIn(t, broken, "rev-parse", "--git-common-dir")
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a repaired worktree lets the move through: %v", err)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue {
		t.Fatalf("the repaired drain judges its first issue: %+v", res)
	}
}

// splitRecord copies the open record of id into resolved/ under dir, so its
// ledger holds the id in two folders.
func splitRecord(t *testing.T, dir, id string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".abcd", "work", "issues", "open", id+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("the open record of %s: %v %v", id, matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	resolved := filepath.Join(dir, ".abcd", "work", "issues", "resolved")
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resolved, filepath.Base(matches[0])), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestASplitLedgerPeerIsToldToSettleItsLedger: a worktree or a branch whose
// ledger holds one id in two folders is told the reason's own advice, to move
// or remove one of each, and never to repair or remove the peer.
func TestASplitLedgerPeerIsToldToSettleItsLedger(t *testing.T) {
	for _, src := range []string{"worktree", "branch"} {
		t.Run(src, func(t *testing.T) {
			repo, _ := reservingDrainRepo(t, "commands/drain.md")
			name := "the worktree at wt-split (branch lane-split)"
			if src == "worktree" {
				wt := filepath.Join(t.TempDir(), "wt-split")
				repo.Git("worktree", "add", "-q", "-b", "lane-split", wt)
				splitRecord(t, wt, eligibleIssue)
			} else {
				name = "branch lane-split"
				repo.Git("checkout", "-q", "-b", "lane-split")
				splitRecord(t, repo.Root(), eligibleIssue)
				repo.Commit("split the record")
				repo.Git("checkout", "-q", "main")
			}
			_, err := Drain(repo.Root(), Options{}, DrainOptions{})
			r := mustRefusal(t, err)
			if r.Check != CheckPeers || r.Contention || !strings.Contains(r.Reason, name+" could not be read") {
				t.Fatalf("the split peer refuses the move, named: %+v", r)
			}
			if !strings.Contains(r.Remedy, name+": move or remove one of each") {
				t.Errorf("the remedy is the reason's own advice, for that peer: %q", r.Remedy)
			}
			for _, never := range []string{"--force", "git worktree remove", "git worktree repair", "delete the branch"} {
				if strings.Contains(r.Remedy, never) {
					t.Errorf("a split ledger is settled, not %q: %q", never, r.Remedy)
				}
			}
		})
	}
}

// TestAWorktreeOutsideHomeIsNamedByAPathAShellTakes: a worktree whose record
// folders cannot be read, outside HOME, holding a session's ignored files in
// .abcd/.work.local/, is told to settle what its reason names and is offered
// no command that removes it: a plain `git worktree remove` deletes ignored
// files without asking, so the remedy says so, names .abcd/.work.local/, and
// gives instead `git -C "<path>" status --ignored`, with the worktree named by
// a path a shell takes, quoted; that command, run as printed through a shell,
// lists the ignored files and leaves them where they are.
func TestAWorktreeOutsideHomeIsNamedByAPathAShellTakes(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	ignore, _ := os.ReadFile(filepath.Join(repo.Root(), ".gitignore"))
	if !strings.Contains(string(ignore), ".abcd/.work.local/") {
		repo.Write(".gitignore", string(ignore)+".abcd/.work.local/\n")
		repo.Commit("ignore the local tier")
	}
	parent := filepath.Join(t.TempDir(), "a dir")
	wt := filepath.Join(parent, "wt-sealed")
	repo.Git("worktree", "add", "-q", "-b", "lane-sealed", wt)
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(wt, home+string(filepath.Separator)) {
		t.Skipf("the temporary directory %s is under HOME", wt)
	}
	// A commit of its own keeps the worktree from reading as merged, and its
	// open folder sealed is what cannot be read.
	gitIn(t, wt, "commit", "-q", "--allow-empty", "-m", "the lane's own work")
	next := filepath.Join(wt, ".abcd", ".work.local", "NEXT.md")
	if err := os.MkdirAll(filepath.Dir(next), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(next, []byte("a peer's handover\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, wt, "check-ignore", "-q", ".abcd/.work.local/NEXT.md")
	sealed := filepath.Join(wt, ".abcd", "work", "issues", "open")
	if err := os.Chmod(sealed, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sealed, 0o755) })

	_, err := Drain(repo.Root(), Options{}, DrainOptions{})
	r := mustRefusal(t, err)
	if r.Check != CheckPeers || !strings.Contains(r.Reason, "the worktree at wt-sealed (branch lane-sealed) could not be read") {
		t.Fatalf("the sealed worktree refuses the move, named by its folder in prose: %+v", r)
	}
	if !strings.Contains(r.Remedy, "wt-sealed (branch lane-sealed): settle what its reason names") {
		t.Errorf("the remedy leads with settling what the reason names: %q", r.Remedy)
	}
	for _, never := range []string{"--force", "git worktree remove", "refuses", "uncommitted"} {
		if strings.Contains(r.Remedy, never) {
			t.Errorf("the remedy never has %q: %q", never, r.Remedy)
		}
	}
	for _, w := range []string{"deletes its ignored files", "`.abcd/.work.local/`", "keeps or moves"} {
		if !strings.Contains(r.Remedy, w) {
			t.Errorf("the remedy says removal deletes ignored files (%q): %q", w, r.Remedy)
		}
	}
	m := regexp.MustCompile("`(git -C '[^`]+' status --ignored)`").FindStringSubmatch(r.Remedy)
	if m == nil {
		t.Fatalf("the remedy gives git -C <path> status --ignored, quoted: %q", r.Remedy)
	}
	if strings.Contains(m[1], "-C 'wt-sealed'") {
		t.Fatalf("the command names the worktree by a path, not its folder name: %q", m[1])
	}
	if err := os.Chmod(sealed, 0o755); err != nil {
		t.Fatal(err)
	}

	// Run as printed, through a shell, from the repository.
	cmd := exec.Command("sh", "-c", m[1])
	cmd.Dir, cmd.Env = repo.Root(), gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s, run as printed: %v\n%s", m[1], err, out)
	}
	if !strings.Contains(string(out), ".abcd/.work.local/") {
		t.Fatalf("%s lists the ignored local tier: %s", m[1], out)
	}
	if _, err := os.Stat(next); err != nil {
		t.Fatalf("the ignored handover survives the check: %v", err)
	}
}

// TestACommandPathIsOneAShellTakes: a path a command names is home-redacted
// to "$HOME", the one part in double quotes, which a shell expands where it
// would not expand ~; the rest is in single quotes, inside which no shell
// expands anything, `!` included, with each ' written '\”; outside HOME the
// path is single-quoted whole, the path itself and never its folder name. HOME
// is "$HOME" only where it starts the path: met mid-path, a second time, or
// after a newline, it is single-quoted with the rest. Each form, read back by
// sh, is the path.
func TestACommandPathIsOneAShellTakes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cases := map[string]string{
		filepath.Join(home, "wt", "lane a"): `"$HOME"'/wt/lane a'`,
		home:                                `"$HOME"`,
		filepath.Join(home, "wt!x"):         `"$HOME"'/wt!x'`,
		filepath.Join(home, "it's"):         `"$HOME"'/it'\''s'`,
		"/srv/wt/lane":                      `'/srv/wt/lane'`,
		"/srv/a dir/wt":                     `'/srv/a dir/wt'`,
		"/srv/wt!1":                         `'/srv/wt!1'`,
		"/srv/it's":                         `'/srv/it'\''s'`,
		"/srv/w\"t/$x`y\\z":                 "'/srv/w\"t/$x`y\\z'",
		// HOME is "$HOME" only as the path's leading prefix: met anywhere
		// else, it is the path's own bytes, single-quoted with the rest.
		"/srv/a " + home + "/wt":   "'/srv/a " + home + "/wt'",
		home + "/a " + home + "/b": `"$HOME"'/a ` + home + `/b'`,
		home + "/":                 `"$HOME"'/'`,
		home + "/a\nb":             "\"$HOME\"'/a\nb'",
		"/srv/x\n" + home + "/y":   "'/srv/x\n" + home + "/y'",
	}
	for in, want := range cases {
		got := commandPath(in)
		if got != want {
			t.Errorf("commandPath(%q) = %s, want %s", in, got, want)
		}
		out, err := exec.Command("sh", "-c", "printf %s "+got).Output()
		if err != nil || string(out) != in {
			t.Errorf("sh reads %s back as %q (%v), want %q", got, out, err, in)
		}
	}
}

// TestACommandPathKnowsHomeInAnotherCase: on a filesystem that folds case,
// HOME spelled in another case is still HOME, as fsutil.RedactHome reads it,
// so the remedy never prints the account's home directory whole.
func TestACommandPathKnowsHomeInAnotherCase(t *testing.T) {
	if !fsutil.CaseFoldingFS() {
		t.Skip("this filesystem does not fold case")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	upper := strings.ToUpper(home)
	if upper == home {
		t.Skip("HOME has no letters to fold")
	}
	cases := map[string]string{
		upper:          `"$HOME"`,
		upper + "/wt":  `"$HOME"'/wt'`,
		upper + "x/wt": "'" + upper + "x/wt'",
	}
	for in, want := range cases {
		if got := commandPath(in); got != want {
			t.Errorf("commandPath(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestEachUnreadableReasonGetsItsOwnRemedy: the remedy follows the peer's
// reason. A broken git file is a plain `git worktree repair`; a ledger holding
// one id in two folders is the reason's own advice; a worktree another account
// owns says that account owns it, and offers no git command; a worktree whose
// owner cannot be read says so, and offers no command at all; a branch git
// cannot list is settled or deleted; and a worktree that cannot be read for
// another reason is settled, with `git -C "<path>" status --ignored` to list
// the ignored files a removal would delete, and no command that removes it.
// None advises --force, and none claims git refuses to remove uncommitted work.
func TestEachUnreadableReasonGetsItsOwnRemedy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wt := filepath.Join(home, "wt")
	own := uint32(os.Getuid())
	restore := swapPeerOwnerUID(func(p string) (uint32, error) {
		if strings.HasSuffix(p, "theirs") {
			return own + 1, nil
		}
		if strings.HasSuffix(p, "unowned") {
			return 0, errors.New("stat: permission denied")
		}
		return own, nil
	})
	defer restore()
	split := "its ledger holds one id in two places (iss-1 in open/ and resolved/), which is no status at all; move or remove one of each so the ledger says which status the record is in"
	cases := []struct {
		name  string
		peer  peers.Peer
		want  []string
		never []string
	}{
		{"a broken git file", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: wt,
			NotRead: "git refused to answer for it: fatal: not a git repository: /elsewhere/nowhere"},
			[]string{"`git worktree repair`"}, []string{"git worktree remove", "repair \""}},
		{"a split worktree ledger", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: wt, NotRead: split},
			[]string{"move or remove one of each so the ledger says which status the record is in"}, []string{"git worktree"}},
		{"a split branch ledger", peers.Peer{Source: peers.SourceBranch, Branch: "b", NotRead: split},
			[]string{"move or remove one of each"}, []string{"git worktree", "delete the branch"}},
		{"another account's worktree", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: filepath.Join(home, "theirs"),
			NotRead: "git refused to answer for it: fatal: detected dubious ownership in repository at '" + filepath.Join(home, "theirs") + "'"},
			[]string{"is owned by " + accountName(own+1) + ", not this account"}, []string{"git worktree", "safe.directory"}},
		{"git's word for another account", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: wt,
			NotRead: "git refused to answer for it: fatal: detected dubious ownership in repository at '" + wt + "'"},
			[]string{"is owned by another account, not this account"}, []string{"git worktree", "safe.directory"}},
		{"a branch git cannot list", peers.Peer{Source: peers.SourceBranch, Branch: "b", NotRead: "git could not list its records: fatal: bad object"},
			[]string{"settle what its reason names, or delete the branch once its work is spent"}, []string{"git worktree"}},
		{"a worktree whose folders cannot be read", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: wt,
			NotRead: "its record folders cannot be read: permission denied"},
			[]string{"settle what its reason names", "`git -C \"$HOME\"'/wt' status --ignored`", "deletes its ignored files", "`.abcd/.work.local/`"},
			[]string{"repair", "git worktree remove"}},
		{"a worktree whose owner cannot be read", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: filepath.Join(home, "unowned"),
			NotRead: "its record folders cannot be read: permission denied"},
			[]string{"its owner could not be read"}, []string{"`git", "repair", "remove `"}},
		{"git's word for another account, owner unread", peers.Peer{Source: peers.SourceWorktree, Branch: "b", Path: filepath.Join(home, "unowned"),
			NotRead: "git refused to answer for it: fatal: detected dubious ownership in repository at '" + filepath.Join(home, "unowned") + "'"},
			[]string{"is owned by another account, not this account"}, []string{"`git"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := peerRemedy(c.peer)
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("the remedy lacks %q: %q", w, got)
				}
			}
			for _, n := range append(c.never, "--force", "refuses", "uncommitted") {
				if strings.Contains(got, n) {
					t.Errorf("the remedy has %q: %q", n, got)
				}
			}
		})
	}
}

// TestAReasonCarriesNoAbsolutePath: a peer's reason names its own worktree
// by its display form, the checkout relative to itself, HOME as ~, and any
// other absolute path by its last element alone.
func TestAReasonCarriesNoAbsolutePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(t.TempDir(), "checkout")
	p := peers.Peer{Source: peers.SourceWorktree, Path: "/srv/peers/wt",
		NotRead: "git refused to answer for it: fatal: not a git repository: /srv/elsewhere/nowhere at '" + root + "/.git/worktrees/wt' from " +
			home + "/x and /srv/peers/wt/.git"}
	got := p.NotReadShown(root)
	want := "git refused to answer for it: fatal: not a git repository: nowhere at './.git/worktrees/wt' from ~/x and wt/.git"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// TestTheBuildSideHolderLineCarriesNoAbsoluteGitdir: the peers check a build
// or a start runs names an unreadable peer as a holder with the same reason
// the drain's refusal shows, so a broken git file pointing at a gitdir outside
// HOME is named by its last element, never by the absolute local path git
// reports (iss-2609281329007423's class).
func TestTheBuildSideHolderLineCarriesNoAbsoluteGitdir(t *testing.T) {
	repo := drainRepo(t)
	broken := filepath.Join(t.TempDir(), "wt-broken")
	repo.Git("worktree", "add", "-q", "-b", "lane-broken", broken)
	nowhere := breakGitFile(t, broken)

	snap, err := readPeers(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	row := peersCheck(secondIssue, "open", "", snap, false)
	if row.OK || !strings.Contains(row.Detail, "the worktree at wt-broken (branch lane-broken) could not be read") {
		t.Fatalf("the check names the unreadable worktree as a holder: %+v", row)
	}
	dirs := []string{filepath.Dir(nowhere)}
	if real, err := filepath.EvalSymlinks(filepath.Dir(nowhere)); err == nil {
		dirs = append(dirs, real)
	}
	for _, d := range dirs {
		if strings.Contains(row.Detail, d) {
			t.Errorf("the holder line carries the absolute gitdir under %s: %q", d, row.Detail)
		}
	}
	if absPathRe.MatchString(row.Detail) {
		t.Errorf("the holder line carries an absolute path: %q", row.Detail)
	}
	if !strings.Contains(row.Detail, "nowhere") {
		t.Errorf("the holder line still says where the git file points, by its name: %q", row.Detail)
	}
}

// TestALiveDrainFinishesAMergedLaneBeforeItRefuses: the refusal is taken
// where per-issue contention begins, after the move's housekeeping: a live
// drain whose lane has merged finishes that lane in the refused move (its
// worktree removed, the run done and the drain's lane done with its pull
// request, written), and only then refuses on the unreadable worktree.
func TestALiveDrainFinishesAMergedLaneBeforeItRefuses(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, first.Lane.Issue
	l := f.armIssueLane(t)
	f.merged(t, l.Landing.Pushed)
	broken := filepath.Join(t.TempDir(), "wt-broken")
	f.repo.Git("worktree", "add", "-q", "-b", "lane-broken", broken)
	breakGitFile(t, broken)

	_, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	r := mustRefusal(t, err)
	if r.Stage != StageDrain || r.Check != CheckPeers || r.Contention {
		t.Fatalf("the move refuses on the unreadable worktree: %+v", r)
	}
	if !strings.Contains(r.Reason, "finished 1 merged lane(s)") || !strings.Contains(r.Reason, f.runID) {
		t.Errorf("the refusal says what the move did first: %q", r.Reason)
	}
	if _, err := os.Stat(l.Worktree); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the merged lane's worktree is removed in the refused move: %v", err)
	}
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || !st.Complete() {
		t.Fatalf("the merged lane's run is done: %+v %v", st, err)
	}
	st, live, err := readDrain(f.repo.Root())
	if err != nil || !live {
		t.Fatalf("the drain is live: %v", err)
	}
	if len(st.Lanes) != 1 || st.Lanes[0].Outcome != DrainLaneDone || st.Lanes[0].PR != 7 {
		t.Fatalf("the drain's lane is written done with its pull request: %+v", st.Lanes)
	}
	if st.Stopped != "" {
		t.Fatalf("a refused move does not end the drain: %q", st.Stopped)
	}
}

// TestAPeerUnreadableAtTheStartStopsTheMove: a peer that turns unreadable
// between the move's check and the issue's start, so the start's own peers
// check finds it, stops the move with the move's refusal, rather than passing
// the issue over as contended and ending the drain empty.
func TestAPeerUnreadableAtTheStartStopsTheMove(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	wt := filepath.Join(t.TempDir(), "wt-late")
	repo.Git("worktree", "add", "-q", "-b", "lane-late", wt)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil || first.Judging == nil {
		t.Fatalf("the drain asks its first judgement: %+v %v", first, err)
	}
	prev := drainBeforeStart
	drainBeforeStart = func(string) { breakGitFile(t, wt) }
	defer func() { drainBeforeStart = prev }()

	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "the remedy changes neither what a user sees nor a trust boundary"))
	res, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path})
	r := mustRefusal(t, err)
	if r.Stage != StageDrain || r.Check != CheckPeers || r.Contention {
		t.Fatalf("the move stops with its own refusal, not contention: %+v", r)
	}
	if !strings.Contains(r.Reason, "the worktree at wt-late (branch lane-late) could not be read") || !strings.Contains(r.Remedy, "`git worktree repair`") {
		t.Fatalf("the refusal names the worktree and its repair: %+v", r)
	}
	if len(res.Passed) != 0 || runCount(t, repo) != 0 {
		t.Fatalf("no issue is passed over and no run starts: %+v", res)
	}
	// The drain reserved the issue before it asked the judgement
	// (spc-2610161044461448 step 2); no lane opened on that hold, so the
	// refused move releases it, and nothing is left reserved.
	if sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+first.Judging.Issue); sha == "" {
		t.Fatalf("the drain reserved %s before it asked the judgement", first.Judging.Issue)
	}
	if snap, err := claim.View(repo.Root(), time.Now()); err != nil {
		t.Fatal(err)
	} else if r, ok := snap.Lookup(first.Judging.Issue); !ok || r.State != claim.StateReleased {
		t.Fatalf("the judgement's hold is released, and nothing is left reserved: %+v %v", r, ok)
	}
	st, live, err := readDrain(repo.Root())
	if err != nil || !live || st.Stopped != "" {
		t.Fatalf("the drain is not ended empty: %+v %v", st, err)
	}
	if len(st.Judgements) != 1 || st.Judging != nil {
		t.Fatalf("the judgement the move took is written, so it is not asked again: %+v", st)
	}
}

// crowdOriginDefault pushes to origin's default branch a commit holding more
// record files than a reader takes, and fetches it, so the reservations, read
// against it, cannot be read here although every fetch succeeds. The crowd
// sits in the intent store beside the issue ledger the commit keeps as it
// was: a drain plans from origin's default branch (spc-2610161044461448 step
// 1), so its issues must still read there.
func crowdOriginDefault(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	empty := gitIn(t, repo.Root(), "hash-object", "-w", "/dev/null")
	var b strings.Builder
	for i := 0; i <= 20000; i++ {
		fmt.Fprintf(&b, "100644 %s\t%s/drafts/f%05d.md\n", empty, intent.IntentsRelDir, i)
	}
	index := filepath.Join(t.TempDir(), "crowd.index")
	git := func(stdin string, args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo.Root()}, args...)...)
		cmd.Env = append(gittest.Env(t), "GIT_INDEX_FILE="+index)
		cmd.Stdin = strings.NewReader(stdin)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return strings.TrimSpace(string(out))
	}
	git("", "read-tree", "main")
	git(b.String(), "update-index", "--index-info")
	tree := git("", "write-tree")
	c := gitIn(t, repo.Root(), "commit-tree", tree, "-p", "main", "-m", "many records")
	gitIn(t, repo.Root(), "push", "-q", "origin", c+":refs/heads/main")
	gitIn(t, repo.Root(), "fetch", "-q", "origin")
}

// TestUnreadableReservationsStopTheMove: reservations this checkout cannot
// read after a fetch that succeeded stop the move with a refusal that says so
// and why, and pass no issue over as contended: the drain never ends empty
// on them, and a second move refuses the same way without asking. The drain
// reserves an issue before it asks its judgement (spc-2610161044461448 step
// 2), so the first move already refuses, on that reservation's own read of
// origin, and no judgement is asked.
func TestUnreadableReservationsStopTheMove(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	if _, err := claim.Reserve(repo.Root(), claim.Request{Record: "iss-2609010000009999", Footprint: []string{"docs/other.md"}}, claimOptions(time.Now())); err != nil {
		t.Fatal(err)
	}
	crowdOriginDefault(t, repo)
	if _, err := claim.View(repo.Root(), time.Now()); err == nil {
		t.Fatal("the fixture's reservations read")
	}
	for move := range 2 {
		res, err := Drain(repo.Root(), Options{}, DrainOptions{})
		r := mustRefusal(t, err)
		if r.Contention || r.Stage != StageReservation || !strings.Contains(r.Reason, "could not read") ||
			!strings.Contains(r.Reason, "more than 20000 record files") || !strings.Contains(r.Reason, "no reservation was made") {
			t.Fatalf("move %d (%v): the refusal says what could not be read, and why: %+v", move, err, r)
		}
		if len(res.Passed) != 0 || res.Judging != nil || runCount(t, repo) != 0 {
			t.Fatalf("move %d: nothing is passed over, asked or started: %+v", move, res)
		}
		if st, _, err := readDrain(repo.Root()); err != nil || st.Stopped != "" || st.Judging != nil {
			t.Fatalf("move %d: the drain is not ended, and asks nothing: %+v %v", move, st, err)
		}
	}
}

// swapPeerOwnerUID substitutes the owner lookup the remedy reads and returns
// the restore: a second account is not something a test process can create.
func swapPeerOwnerUID(fn func(string) (uint32, error)) (restore func()) {
	prev := peerOwnerUID
	peerOwnerUID = fn
	return func() { peerOwnerUID = prev }
}

// TestALockedReadableWorktreeDoesNotRefuseTheDrain: git's lock on a worktree,
// as another tool takes it, keeps prune and remove away from it and says
// nothing of whether it reads; a locked worktree that reads is an ordinary
// peer, and the drain's move goes on to its judgement.
func TestALockedReadableWorktreeDoesNotRefuseTheDrain(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	locked := filepath.Join(t.TempDir(), "wt-locked")
	repo.Git("worktree", "add", "-q", "-b", "lane-locked", locked)
	repo.Git("worktree", "lock", "--reason", `{"owner":"another-tool"}`, locked)

	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a locked worktree that reads refuses nothing: %v", err)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue {
		t.Fatalf("the drain judges its first issue past a locked worktree: %+v", res)
	}
}
