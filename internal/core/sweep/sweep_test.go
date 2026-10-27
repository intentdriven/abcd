package sweep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// sweepRepo is a repository with one commit on main under a fresh HOME, its
// local tier ignored as abcd's own checkouts ignore it, so the tier never
// makes a worktree dirty: the store's prune fixture.
func sweepRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "a\n")
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Commit("one")
	return repo, home
}

// laneAdd makes the worktree name in the repository's lane on a new branch of
// the same name, cut from main, through the store's own Add.
func laneAdd(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	got, err := store.Add(repo.Root(), store.AddOptions{Name: name, Base: repo.Git("rev-parse", "main"),
		Git: func(dir string, args ...string) (string, error) { return gitutil.Run(dir, args...) }})
	if err != nil {
		t.Fatal(err)
	}
	return got.Path
}

// commitAt writes rel in the worktree at dir and commits it there.
func commitAt(t *testing.T, repo *gittest.Repo, dir, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, rel), rel+"\n")
	repo.Git("-C", dir, "add", "-A")
	repo.Git("-C", dir, "commit", "-q", "-m", "change "+rel)
}

// mergedAdd is a lane worktree with one commit of its own that main has
// fast-forwarded over: merged and clean.
func mergedAdd(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	p := laneAdd(t, repo, name)
	commitAt(t, repo, p, name+".txt")
	repo.Git("merge", "-q", "--ff-only", name)
	return p
}

// unmergedAdd is a lane worktree with one commit of its own, at rel, that main
// does not hold.
func unmergedAdd(t *testing.T, repo *gittest.Repo, name, rel string) string {
	t.Helper()
	p := laneAdd(t, repo, name)
	commitAt(t, repo, p, rel)
	return p
}

func nothingHeld(string) (store.Held, error) { return store.Held{}, nil }

// sweepOpts is a dry run that nothing but the test's own setting holds: the
// reservations stamped as fetched, no process, no run, a clock far enough
// ahead that nothing the test wrote is recent, and a working directory no
// worktree holds.
func sweepOpts() Options {
	return Options{
		Probe: ProbeOptions{Reservations: claim.Snapshot{FetchedAt: time.Unix(1, 0)}, Processes: cwds(), Now: stale()},
		Held:  nothingHeld,
		Cwd:   string(filepath.Separator),
	}
}

func dryRun(t *testing.T, repo *gittest.Repo, o Options) Result {
	t.Helper()
	got, err := DryRun(context.Background(), repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if got.Refusal != "" {
		t.Fatalf("a completed dry run carries no refusal: %q", got.Refusal)
	}
	return got
}

func rowFor(t *testing.T, res Result, name string) Row {
	t.Helper()
	for _, r := range res.Rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no row named %q in %+v", name, res.Rows)
	return Row{}
}

// kept asserts the row is a kept worktree whose reason contains want.
func kept(t *testing.T, r Row, want string) {
	t.Helper()
	if r.Action != ActionKeep || !strings.Contains(r.Reason, want) {
		t.Fatalf("row %s = %+v; want it kept, its reason containing %q", r.Name, r, want)
	}
}

// hasEvidence reports whether the row carries evidence of kind and strength.
func hasEvidence(r Row, kind EvidenceKind, s Strength) bool {
	return slices.ContainsFunc(r.Evidence, func(e Evidence) bool { return e.Kind == kind && e.Strength == s })
}

// treeHash is every entry under the roots: its path, mode, size, modification
// time and content (a symlink's target), so any write a dry run made, git's
// own index refresh included, changes it.
func treeHash(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			sum := fmt.Sprintf("%s %d %d", fi.Mode(), fi.Size(), fi.ModTime().UnixNano())
			switch {
			case fi.Mode()&fs.ModeSymlink != 0:
				target, err := os.Readlink(p)
				if err != nil {
					return err
				}
				sum += " -> " + target
			case fi.Mode().IsRegular():
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				h := sha256.Sum256(b)
				sum += " " + hex.EncodeToString(h[:])
			}
			out[p] = sum
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// TestTheDryRunListsEveryKindAndWritesNothing is criterion 1: one row per
// worktree git lists, in the store and outside it, per stray and for the
// embargo store, with every action a worktree can take, and not one byte of
// the checkout, its git dir, the store or the outside worktree changed.
func TestTheDryRunListsEveryKindAndWritesNothing(t *testing.T) {
	repo, home := sweepRepo(t)
	mergedAdd(t, repo, "done")
	dirty := laneAdd(t, repo, "wip")
	write(t, filepath.Join(dirty, "scratch.txt"), "wip\n")
	unmergedAdd(t, repo, "open", "open.txt")
	l, _ := store.LaneFor(repo.Root())
	repo.Git("worktree", "add", "-q", "--detach", l.Path(storepath.Name))
	write(t, l.Path("loose.txt"), "a file in the lane\n")
	outside := filepath.Join(t.TempDir(), "beside")
	repo.Git("worktree", "add", "-q", "-b", "beside", outside)

	roots := []string{repo.Root(), home, filepath.Dir(outside)}
	before := treeHash(t, roots...)
	got := dryRun(t, repo, sweepOpts())
	after := treeHash(t, roots...)
	for p, h := range before {
		if after[p] != h {
			t.Errorf("the dry run changed %s: %q -> %q", p, h, after[p])
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("the dry run made %s", p)
		}
	}

	want := map[string]struct {
		kind   RowKind
		action Action
	}{
		"done":         {RowWorktree, ActionRemove},
		"wip":          {RowWorktree, ActionSavePatch},
		"open":         {RowWorktree, ActionKeep},
		"beside":       {RowWorktree, ActionKeep},
		storepath.Name: {RowEmbargoStore, ActionKeep},
		"loose.txt":    {RowStray, ActionKeep},
	}
	if len(got.Rows) != len(want) {
		t.Fatalf("want %d rows, got %+v", len(want), got.Rows)
	}
	for name, w := range want {
		r := rowFor(t, got, name)
		if r.Kind != w.kind || r.Action != w.action || r.Reason == "" || r.State == "" || r.Path == "" {
			t.Errorf("%s: %+v; want kind %s, action %s, with a state, a path and a reason", name, r, w.kind, w.action)
		}
		if strings.Contains(r.Path, home) {
			t.Errorf("%s: the path is home-redacted: %q", name, r.Path)
		}
	}
	if p := rowFor(t, got, "done").Path; p != fsutil.RedactHome(l.Path("done")) {
		t.Errorf("a store worktree's path is the store's own redacted form: %q", p)
	}
	kept(t, rowFor(t, got, "beside"), "outside the store")
	if got.ExitCode() != ExitActed {
		t.Fatalf("a dry run that would act exits %d; want %d", got.ExitCode(), ExitActed)
	}
}

// TestEachKeepReason is one keep per reason the store's reading gives: dirty
// (a salvage, and kept), unmerged, locked naming a JSON reason's owner, the
// embargo store, a stray, and what the sweep cannot classify.
func TestEachKeepReason(t *testing.T) {
	repo, _ := sweepRepo(t)

	t.Run("dirty", func(t *testing.T) {
		p := laneAdd(t, repo, "dirty")
		write(t, filepath.Join(p, "scratch.txt"), "wip\n")
		r := rowFor(t, dryRun(t, repo, sweepOpts()), "dirty")
		if r.Action != ActionSavePatch || !strings.Contains(r.Reason, "dirty") || !strings.Contains(r.Reason, "kept") || !hasEvidence(r, KindState, Proven) {
			t.Fatalf("a dirty worktree nothing holds: %+v; want its changes saved and the worktree kept", r)
		}
	})

	t.Run("unmerged", func(t *testing.T) {
		unmergedAdd(t, repo, "unmerged", "u.txt")
		r := rowFor(t, dryRun(t, repo, sweepOpts()), "unmerged")
		kept(t, r, "unmerged")
		if !hasEvidence(r, KindMerged, Proven) {
			t.Fatalf("the merged judgement is proven evidence: %+v", r)
		}
	})

	t.Run("locked naming its owner", func(t *testing.T) {
		p := mergedAdd(t, repo, "locked-json")
		repo.Git("worktree", "lock", "--reason", `{"owner":"agent-tool","session":"s-1"}`, p)
		q := mergedAdd(t, repo, "locked-plain")
		repo.Git("worktree", "lock", "--reason", "on a removable disk", q)
		res := dryRun(t, repo, sweepOpts())
		r := rowFor(t, res, "locked-json")
		kept(t, r, "locked by agent-tool")
		if !hasEvidence(r, KindLock, Proven) || !strings.Contains(r.Reason, `"session":"s-1"`) {
			t.Fatalf("the lock's whole reason is shown, its owner named: %+v", r)
		}
		kept(t, rowFor(t, res, "locked-plain"), "locked by git: on a removable disk")
	})

	t.Run("the embargo store", func(t *testing.T) {
		l, _ := store.LaneFor(repo.Root())
		emb := l.Path(storepath.Name)
		repo.Git("worktree", "add", "-q", "--detach", emb)
		write(t, filepath.Join(emb, "advisory.md"), "private\n")
		var asked []string
		o := sweepOpts()
		o.Probe.CopyView = own(&asked)
		o.Probe.Owner = func(p string) (int, bool, error) {
			asked = append(asked, p)
			return 0, false, nil
		}
		r := rowFor(t, dryRun(t, repo, o), storepath.Name)
		if r.Kind != RowEmbargoStore || r.Action != ActionKeep || !strings.Contains(r.Reason, "embargo store") || !hasEvidence(r, KindStore, Proven) {
			t.Fatalf("the embargo store: %+v", r)
		}
		real, _ := filepath.EvalSymlinks(emb)
		if slices.Contains(asked, real) || slices.Contains(asked, emb) {
			t.Fatalf("the embargo store is never read: the probe asked about %q", asked)
		}
	})

	t.Run("a stray", func(t *testing.T) {
		l, _ := store.LaneFor(repo.Root())
		write(t, l.Path("notes.txt"), "left here\n")
		r := rowFor(t, dryRun(t, repo, sweepOpts()), "notes.txt")
		if r.Kind != RowStray || r.Action != ActionKeep || !strings.Contains(r.Reason, "never touched") {
			t.Fatalf("a loose file in the lane: %+v", r)
		}
		if _, err := os.Lstat(l.Path("notes.txt")); err != nil {
			t.Fatal("the stray stands")
		}
	})

	t.Run("unclassifiable", func(t *testing.T) {
		gone := mergedAdd(t, repo, "gone")
		if err := os.RemoveAll(gone); err != nil {
			t.Fatal(err)
		}
		l, _ := store.LaneFor(repo.Root())
		if err := os.Mkdir(l.Path("odd"), 0o700); err != nil {
			t.Fatal(err)
		}
		res := dryRun(t, repo, sweepOpts())
		for name, want := range map[string]string{
			"gone": "directory gone; git still lists it",
			"odd":  "not a worktree of this repository",
		} {
			r := rowFor(t, res, name)
			kept(t, r, want)
			if r.Kind != RowWorktree || !strings.Contains(r.Reason, "cannot classify") || !hasEvidence(r, KindStore, Proven) {
				t.Errorf("%s: %+v; want a worktree row kept as unclassifiable, with the store's reason", name, r)
			}
		}
	})
}

// TestEachLivenessProofKeepsAMergedCleanWorktree: a worktree prune would
// reclaim is kept by each liveness proof, and by the recent-write hint,
// with the probe's reason and its evidence on the row.
func TestEachLivenessProofKeepsAMergedCleanWorktree(t *testing.T) {
	repo, _ := sweepRepo(t)
	now := stale()()
	cases := []struct {
		name  string
		set   func(o *Options, wt string)
		want  string
		kind  EvidenceKind
		proof Strength
	}{
		{"reserved-from", func(o *Options, wt string) {
			r := claim.Reservation{Record: "itd-2610230000000009", Holder: "bob", Copy: claim.CopyThis, State: claim.StateLive, EffectiveExpiry: now.Add(time.Hour)}
			o.Probe.CopyView = func(p string, _ time.Time) (claim.Snapshot, error) {
				if real, _ := filepath.EvalSymlinks(wt); p == real {
					return claim.Snapshot{Reservations: []claim.Reservation{r}}, nil
				}
				return claim.Snapshot{}, nil
			}
		}, "reserved as itd-2610230000000009 by bob from this worktree", KindReservation, Proven},
		{"run-lane", func(o *Options, wt string) {
			o.Probe.Runs = []loop.State{{RunID: "run-7", Key: "itd-2610222214476887", CreatedAt: now, UpdatedAt: now,
				Lanes: []loop.Lane{{ID: "lane-2", Stage: loop.StageImplement, Worktree: wt}}}}
		}, "lane-2 of build run run-7", KindRun, Proven},
		{"in-use", func(o *Options, wt string) {
			o.Probe.Processes = cwds("/", filepath.Join(wt, "sub"))
		}, "in use: process 101 works in it", KindProcess, Proven},
		{"foreign", func(o *Options, wt string) {
			o.Probe.Owner = func(p string) (int, bool, error) {
				if real, _ := filepath.EvalSymlinks(wt); p == real {
					return 4242, true, nil
				}
				return os.Getuid(), false, nil
			}
		}, "owned by another account (uid 4242)", KindOwner, Proven},
		{"recent", func(o *Options, wt string) {
			o.Probe.Now = time.Now
		}, "within the last 24h", KindRecentWrite, Hint},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wt := mergedAdd(t, repo, c.name)
			o := sweepOpts()
			c.set(&o, wt)
			r := rowFor(t, dryRun(t, repo, o), c.name)
			kept(t, r, c.want)
			if !hasEvidence(r, c.kind, c.proof) {
				t.Fatalf("row %+v; want %s %s evidence", r, c.proof, c.kind)
			}
		})
	}

	// A live reservation whose footprint covers a path the worktree's branch
	// changes holds it: the sweep computes the changes the probe compares.
	t.Run("footprint", func(t *testing.T) {
		unmergedAdd(t, repo, "covered", "pkg/feature/a.go")
		o := sweepOpts()
		o.Probe.Reservations.Reservations = []claim.Reservation{{Record: "iss-2610230000000003", Holder: "carol",
			State: claim.StateLive, Footprint: []string{"pkg/feature"}, EffectiveExpiry: now.Add(time.Hour)}}
		r := rowFor(t, dryRun(t, repo, o), "covered")
		kept(t, r, "reserved as iss-2610230000000003 by carol")
		if !strings.Contains(r.Reason, "pkg/feature/a.go") || !hasEvidence(r, KindReservation, Proven) {
			t.Fatalf("the reservation names the changed path it covers: %+v", r)
		}
		// An uncommitted path counts too.
		p := laneAdd(t, repo, "covered-wip")
		if err := os.MkdirAll(filepath.Join(p, "pkg", "feature"), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(p, "pkg", "feature", "b.go"), "package feature\n")
		r = rowFor(t, dryRun(t, repo, o), "covered-wip")
		kept(t, r, "covering pkg/feature/b.go")
	})

	// The build loop's hold as the store's prune reads it, and the sweep's
	// own working directory, keep it as prune would.
	t.Run("held as prune holds", func(t *testing.T) {
		mergedAdd(t, repo, "run-0000000000000001-lane-1")
		here := mergedAdd(t, repo, "here")
		o := sweepOpts()
		o.Held = func(string) (store.Held, error) {
			return store.Held{Names: map[string]string{"run-0000000000000001-lane-1": "run-0000000000000001"}}, nil
		}
		o.Cwd = filepath.Join(here, "sub")
		res := dryRun(t, repo, o)
		kept(t, rowFor(t, res, "run-0000000000000001-lane-1"), "held by the build loop's run run-0000000000000001")
		kept(t, rowFor(t, res, "here"), "the sweep's own working directory")
	})
}

// TestAWorktreeOnTheDefaultBranchWithALiveDirectoryIsKept is criterion 3: a
// worktree whose tip is the default branch, so nothing of its own is
// unmerged, is kept while a process works in it, and the row says so.
func TestAWorktreeOnTheDefaultBranchWithALiveDirectoryIsKept(t *testing.T) {
	repo, _ := sweepRepo(t)
	l, _ := store.LaneFor(repo.Root())
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	wt := l.Path("fresh")
	repo.Git("worktree", "add", "-q", "--detach", wt, "main")
	if repo.Git("-C", wt, "rev-parse", "HEAD") != repo.Git("rev-parse", "main") {
		t.Fatal("the worktree's tip is the default branch")
	}
	o := sweepOpts()
	o.Probe.Processes = cwds(wt)
	r := rowFor(t, dryRun(t, repo, o), "fresh")
	kept(t, r, "in use: process 100 works in it")
	if !hasEvidence(r, KindProcess, Proven) {
		t.Fatalf("the process is the proof: %+v", r)
	}
}

// TestAFailedProcessCheckKeepsEveryWorktreeAndSaysSoOnce: lsof missing,
// timing out or printing what cannot be read leaves every worktree as
// possibly live, each row naming why, and the result saying so once at the
// top. The reservations read failing closed is said once the same way.
func TestAFailedProcessCheckKeepsEveryWorktreeAndSaysSoOnce(t *testing.T) {
	repo, _ := sweepRepo(t)
	mergedAdd(t, repo, "one")
	mergedAdd(t, repo, "two")
	cases := []struct {
		name string
		cmd  func(t *testing.T) lsofCommand
		want string
	}{
		{"missing", func(t *testing.T) lsofCommand {
			return lsofCommand{path: filepath.Join(t.TempDir(), "lsof"), timeout: 5 * time.Second, limit: 1 << 20}
		}, "is missing"},
		{"timeout", func(t *testing.T) lsofCommand {
			return lsofCommand{path: script(t, "exec sleep 30"), timeout: 200 * time.Millisecond, limit: 1 << 20}
		}, "timed out after 200ms"},
		{"unreadable", func(t *testing.T) lsofCommand {
			return lsofCommand{path: script(t, "echo hello"), timeout: 5 * time.Second, limit: 1 << 20}
		}, "unreadable output"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := sweepOpts()
			o.Probe.Processes = c.cmd(t).list
			res := dryRun(t, repo, o)
			if !strings.Contains(res.ProcessCheckUnavailable, c.want) {
				t.Fatalf("ProcessCheckUnavailable = %q; want it to contain %q", res.ProcessCheckUnavailable, c.want)
			}
			if n := countContaining(res.Notes(), "process check unavailable"); n != 1 {
				t.Fatalf("Notes() = %q; want the process check named exactly once", res.Notes())
			}
			for _, name := range []string{"one", "two"} {
				r := rowFor(t, res, name)
				kept(t, r, "process check unavailable: "+res.ProcessCheckUnavailable)
				if !hasEvidence(r, KindProcessCheck, Hint) {
					t.Fatalf("%s: %+v; want the failed check as a hint", name, r)
				}
			}
			if res.ExitCode() != ExitNothing {
				t.Fatalf("nothing would be acted on: exit %d", res.ExitCode())
			}
		})
	}

	t.Run("reservations unread", func(t *testing.T) {
		o := sweepOpts()
		o.Probe.Reservations = claim.Snapshot{}
		res := dryRun(t, repo, o)
		if res.ReservationsUnread == "" || countContaining(res.Notes(), "reservations unread") != 1 {
			t.Fatalf("result %+v; want the unread reservations named once at the top", res)
		}
		kept(t, rowFor(t, res, "one"), "reservations unread")
	})
}

func countContaining(lines []string, sub string) int {
	n := 0
	for _, l := range lines {
		if strings.Contains(l, sub) {
			n++
		}
	}
	return n
}

// TestTheDryRunRefusesWithoutAReaderOfTheLoopsHold: like the store's prune, a
// sweep that cannot tell what the build loop holds acts on nothing, and the
// result maps to the refusal's exit code.
func TestTheDryRunRefusesWithoutAReaderOfTheLoopsHold(t *testing.T) {
	repo, _ := sweepRepo(t)
	o := sweepOpts()
	o.Held = nil
	res, err := DryRun(context.Background(), repo.Root(), o)
	if err == nil || res.ExitCode() != ExitRefused || res.Refusal == "" {
		t.Fatalf("DryRun with no reader = %+v, %v; want a refusal exiting %d", res, err, ExitRefused)
	}
	o = sweepOpts()
	o.Held = func(string) (store.Held, error) { return store.Held{}, errors.New("the run state is unreadable") }
	res, err = DryRun(context.Background(), repo.Root(), o)
	if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "the run state is unreadable") {
		t.Fatalf("DryRun with a failing reader = %+v, %v; want a refusal naming why", res, err)
	}
}

// TestTheDryRunRefusesWithoutItsWorkingDirectory: the worktree holding the
// sweep's own working directory is kept, so a caller that forgets to say
// where the sweep runs is refused, as one that forgets the loop's hold is.
func TestTheDryRunRefusesWithoutItsWorkingDirectory(t *testing.T) {
	repo, _ := sweepRepo(t)
	mergedAdd(t, repo, "here")
	o := sweepOpts()
	o.Cwd = ""
	res, err := DryRun(context.Background(), repo.Root(), o)
	if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "working directory") || len(res.Rows) != 0 {
		t.Fatalf("DryRun with no working directory = %+v, %v; want a refusal naming it, exiting %d", res, err, ExitRefused)
	}
}

// TestTheRefusalErrorIsRedacted: the error a refusal returns carries the same
// cleaned text as its Refusal, so a front door printing the error leaks no
// home path, and the cause is still reachable through errors.Is.
func TestTheRefusalErrorIsRedacted(t *testing.T) {
	repo, home := sweepRepo(t)
	cause := fmt.Errorf("cannot open %s", filepath.Join(home, "private", "state.json"))
	o := sweepOpts()
	o.Held = func(string) (store.Held, error) { return store.Held{}, cause }
	res, err := DryRun(context.Background(), repo.Root(), o)
	if err == nil || res.ExitCode() != ExitRefused {
		t.Fatalf("DryRun with a failing reader = %+v, %v; want a refusal", res, err)
	}
	if strings.Contains(err.Error(), home) || !strings.Contains(err.Error(), "~/private/state.json") {
		t.Fatalf("the refusal error is home-redacted: %q", err)
	}
	if err.Error() != res.Refusal {
		t.Fatalf("the error %q and the Refusal %q say the same", err, res.Refusal)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("the cause is reachable through the error: %v", err)
	}
}

// TestAForeignOwnedWorktreeHasNoGitRunInIt: the owner is judged before any git
// command, so a worktree another account owns is kept on the owner alone and
// this account runs no git in it, while its own worktree is read as before.
func TestAForeignOwnedWorktreeHasNoGitRunInIt(t *testing.T) {
	repo, _ := sweepRepo(t)
	mine := fsutil.RealExistingPath(mergedAdd(t, repo, "mine"))
	theirs := fsutil.RealExistingPath(mergedAdd(t, repo, "theirs"))
	o := sweepOpts()
	o.Probe.Owner = func(p string) (int, bool, error) {
		if p == theirs {
			return 4242, true, nil
		}
		return os.Getuid(), false, nil
	}
	var ran []string
	o.changes = func(abs, def string) []string {
		ran = append(ran, fsutil.RealExistingPath(abs))
		return changedPaths(abs, def)
	}
	res := dryRun(t, repo, o)
	r := rowFor(t, res, "theirs")
	kept(t, r, "owned by another account (uid 4242)")
	if !hasEvidence(r, KindOwner, Proven) {
		t.Fatalf("the owner is the proof: %+v", r)
	}
	if slices.Contains(ran, theirs) {
		t.Fatalf("git ran in the foreign-owned worktree: %q", ran)
	}
	if !slices.Contains(ran, mine) || rowFor(t, res, "mine").Action != ActionRemove {
		t.Fatalf("this account's own worktree is read and judged as before: ran %q, row %+v", ran, rowFor(t, res, "mine"))
	}
}

// TestAnUnreadableOwnerHasNoGitRunInIt: a worktree whose owner cannot be read
// is kept on the owner as a hint, and this account runs no git in it.
func TestAnUnreadableOwnerHasNoGitRunInIt(t *testing.T) {
	repo, _ := sweepRepo(t)
	unknown := fsutil.RealExistingPath(mergedAdd(t, repo, "unknown"))
	o := sweepOpts()
	o.Probe.Owner = func(p string) (int, bool, error) {
		if p == unknown {
			return 0, false, errors.New("stat failed")
		}
		return os.Getuid(), false, nil
	}
	var ran []string
	o.changes = func(abs, def string) []string {
		ran = append(ran, fsutil.RealExistingPath(abs))
		return changedPaths(abs, def)
	}
	r := rowFor(t, dryRun(t, repo, o), "unknown")
	if r.Action != ActionKeep || !hasEvidence(r, KindOwner, Hint) || !strings.Contains(r.Reason, "stat failed") {
		t.Fatalf("the unreadable owner keeps it, named: %+v", r)
	}
	if slices.Contains(ran, unknown) {
		t.Fatalf("git ran in the worktree whose owner could not be read: %q", ran)
	}
}

// TestADirtyWorktreeThatIsHeldIsKept: a hold outranks the salvage, so a dirty
// worktree a process works in is kept with the hold as its reason, and its
// changes are not saved.
func TestADirtyWorktreeThatIsHeldIsKept(t *testing.T) {
	repo, _ := sweepRepo(t)
	p := laneAdd(t, repo, "dirty-held")
	write(t, filepath.Join(p, "scratch.txt"), "wip\n")
	o := sweepOpts()
	o.Probe.Processes = cwds(p)
	r := rowFor(t, dryRun(t, repo, o), "dirty-held")
	kept(t, r, "in use: process 100 works in it")
	if !slices.Contains(r.Evidence, Evidence{Kind: KindState, Strength: Proven, Detail: "dirty"}) {
		t.Fatalf("the row still says it is dirty: %+v", r)
	}
}

// TestAWorktreeTheStoreGaveAnUnknownStateOrMergeIsKept: the store reads clean
// or dirty and merged or unmerged; a row carrying any other word, and no
// reason, is not one the sweep can classify, and it is kept.
func TestAWorktreeTheStoreGaveAnUnknownStateOrMergeIsKept(t *testing.T) {
	dir := t.TempDir()
	o := sweepOpts()
	o.Probe.CopyView = func(string, time.Time) (claim.Snapshot, error) { return claim.Snapshot{}, nil }
	o.Probe.Owner = func(string) (int, bool, error) { return os.Getuid(), false, nil }
	j := judge{probe: NewProbe(context.Background(), o.Probe), changes: func(string, string) []string { return []string{} }}
	for _, sr := range []store.Row{
		{Name: "odd-state", Branch: "odd-state", Path: "odd-state", Abs: dir, Where: store.WhereStore, State: "half", Merged: "merged", JudgedBy: "ancestry"},
		{Name: "odd-merge", Branch: "odd-merge", Path: "odd-merge", Abs: dir, Where: store.WhereStore, State: "clean", Merged: "maybe"},
	} {
		r := j.row(sr)
		kept(t, r, fmt.Sprintf("cannot classify: the store read its state as %q and its merge as %q", sr.State, sr.Merged))
		if !hasEvidence(r, KindStore, Hint) {
			t.Fatalf("%s: %+v; want the store's unknown word as a hint", sr.Name, r)
		}
	}
}

// TestTwoDetachedWorktreesOutsideTheStoreWithOneNameAreEachJudged: outside
// HOME the store shows a worktree by its base name, so two detached worktrees
// with one base name give two rows alike in name, branch and path. The
// store's Row.Abs ties each row to its own worktree, so each is judged, and
// probed, at its own path, and kept as outside the store.
func TestTwoDetachedWorktreesOutsideTheStoreWithOneNameAreEachJudged(t *testing.T) {
	repo, _ := sweepRepo(t)
	a := filepath.Join(t.TempDir(), "twin")
	b := filepath.Join(t.TempDir(), "twin")
	repo.Git("worktree", "add", "-q", "--detach", a)
	repo.Git("worktree", "add", "-q", "--detach", b)
	o := sweepOpts()
	var ran []string
	o.changes = func(abs, def string) []string {
		ran = append(ran, fsutil.RealExistingPath(abs))
		return changedPaths(abs, def)
	}
	res := dryRun(t, repo, o)
	n := 0
	for _, r := range res.Rows {
		if r.Name != "twin" {
			continue
		}
		n++
		kept(t, r, "outside the store; never touched")
		if !hasEvidence(r, KindState, Proven) {
			t.Fatalf("each twin is judged: %+v", r)
		}
	}
	if n != 2 {
		t.Fatalf("want two rows named twin, got %+v", res.Rows)
	}
	for _, p := range []string{a, b} {
		if !slices.Contains(ran, fsutil.RealExistingPath(p)) {
			t.Fatalf("each twin is probed at its own path: ran %q, missing %s", ran, p)
		}
	}
}

// TestTheDryRunActsOnNothingInAnEmptyLane: a repository with no worktrees
// renders no rows and maps to "nothing to act on".
func TestTheDryRunActsOnNothingInAnEmptyLane(t *testing.T) {
	repo, _ := sweepRepo(t)
	res := dryRun(t, repo, sweepOpts())
	if len(res.Rows) != 0 || res.ExitCode() != ExitNothing {
		t.Fatalf("an empty lane: %+v, exit %d", res, res.ExitCode())
	}
}
