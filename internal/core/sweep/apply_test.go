package sweep

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/gittest"
)

func apply(t *testing.T, repo *gittest.Repo, o Options) Result {
	t.Helper()
	got, err := Apply(context.Background(), repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if got.Refusal != "" || !got.Applied {
		t.Fatalf("a completed apply carries no refusal and is marked applied: %+v", got)
	}
	return got
}

// recordPrunes makes o's prune the store's own PruneOne, recording each name
// it is asked to reclaim.
func recordPrunes(o *Options) *[]string {
	var asked []string
	o.pruneOne = func(repoRoot, name string, opts store.PruneOptions) (store.Pruned, error) {
		asked = append(asked, name)
		return store.PruneOne(repoRoot, name, opts)
	}
	return &asked
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// TestApplyReclaimsASquashMergedWorktreeThroughThePrune is criterion 2: a
// worktree whose branch reached main only as a squash, so judged merged by
// patch-equivalence, is removed through the store's PruneOne, its branch left
// standing, and the run exits as one that acted.
func TestApplyReclaimsASquashMergedWorktreeThroughThePrune(t *testing.T) {
	repo, _ := sweepRepo(t)
	p := unmergedAdd(t, repo, "sq", "sq.txt")
	repo.Git("merge", "-q", "--squash", "sq")
	repo.Git("commit", "-q", "-m", "squashed")
	o := sweepOpts()
	asked := recordPrunes(&o)

	res := apply(t, repo, o)
	r := rowFor(t, res, "sq")
	if r.Action != ActionRemove || r.Outcome != OutcomeRemoved {
		t.Fatalf("the squash-merged worktree: %+v; want it removed", r)
	}
	if !slices.Contains(r.Evidence, Evidence{Kind: KindMerged, Strength: Proven, Detail: "merged, judged by content"}) {
		t.Fatalf("the merge was judged by patch-equivalence: %+v", r.Evidence)
	}
	if !slices.Equal(*asked, []string{"sq"}) {
		t.Fatalf("the removal goes through the store's PruneOne for that one worktree: asked %q", *asked)
	}
	if exists(p) {
		t.Fatal("the worktree's directory is gone")
	}
	if strings.Contains(repo.Git("worktree", "list", "--porcelain"), p) {
		t.Fatal("git no longer lists the worktree")
	}
	repo.Git("rev-parse", "--verify", "refs/heads/sq")
	if res.ExitCode() != ExitActed {
		t.Fatalf("a run that removed a worktree exits %d; want %d", res.ExitCode(), ExitActed)
	}
}

// TestAWorktreeThatGainsALiveProcessBeforeRemovalIsKept: the dry run's probe
// found the worktree free, but a process starts in it before the removal. The
// last check before removal is a fresh probe of that one worktree, with a new
// process listing, and it keeps the worktree, naming the process.
func TestAWorktreeThatGainsALiveProcessBeforeRemovalIsKept(t *testing.T) {
	check := func(t *testing.T, res Result, p string, calls int) {
		t.Helper()
		r := rowFor(t, res, "busy")
		if r.Action != ActionRemove {
			t.Fatalf("the dry run found it free: %+v", r)
		}
		if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "in use: process") || !hasEvidence(r, KindProcess, Proven) {
			t.Fatalf("the worktree a process entered: %+v; want it kept, naming the process", r)
		}
		if !exists(p) {
			t.Fatal("the worktree stands")
		}
		if calls < 2 {
			t.Fatalf("the check before removal lists the processes afresh: %d listings", calls)
		}
		if res.ExitCode() != ExitNothing {
			t.Fatalf("a run that removed nothing exits %d; want %d", res.ExitCode(), ExitNothing)
		}
	}

	t.Run("listed", func(t *testing.T) {
		repo, _ := sweepRepo(t)
		p := mergedAdd(t, repo, "busy")
		calls := 0
		o := sweepOpts()
		o.Probe.Processes = func(context.Context) ([]Process, error) {
			calls++
			if calls == 1 {
				return nil, nil
			}
			return []Process{{PID: 4321, Cwd: p}}, nil
		}
		check(t, apply(t, repo, o), p, calls)
	})

	t.Run("a real process", func(t *testing.T) {
		if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
			t.Skipf("no process check on %s", runtime.GOOS)
		}
		sleep, err := exec.LookPath("sleep")
		if err != nil {
			t.Skipf("no sleep to start: %v", err)
		}
		repo, _ := sweepRepo(t)
		p := mergedAdd(t, repo, "busy")
		var once sync.Once
		calls := 0
		o := sweepOpts()
		o.Probe.Processes = func(ctx context.Context) ([]Process, error) {
			calls++
			procs, err := HostProcesses(ctx)
			if err != nil {
				t.Skipf("the host probe is unavailable here: %v", err)
			}
			// Started after the dry run's listing, as a session entering the
			// worktree between the two would be.
			once.Do(func() {
				child := exec.Command(sleep, "30")
				child.Dir = p
				if err := child.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = child.Process.Kill()
					_ = child.Wait()
				})
			})
			return procs, nil
		}
		check(t, apply(t, repo, o), p, calls)
	})
}

// TestApplyNeverActsOnALockedAnEmbargoOrAnOutsideWorktree: what the dry run
// keeps, the apply keeps, and the store's prune is never asked about it: a
// locked worktree, the embargo store, a merged worktree outside the store,
// and a dirty one, which is saved and kept, never removed.
func TestApplyNeverActsOnALockedAnEmbargoOrAnOutsideWorktree(t *testing.T) {
	repo, _ := sweepRepo(t)
	locked := mergedAdd(t, repo, "locked")
	repo.Git("worktree", "lock", "--reason", "on a removable disk", locked)
	l, _ := store.LaneFor(repo.Root())
	emb := l.Path(storepath.Name)
	repo.Git("worktree", "add", "-q", "--detach", emb)
	outside := filepath.Join(t.TempDir(), "beside")
	repo.Git("worktree", "add", "-q", "-b", "beside", outside)
	dirty := laneAdd(t, repo, "dirty")
	write(t, filepath.Join(dirty, "scratch.txt"), "wip\n")
	o := sweepOpts()
	asked := recordPrunes(&o)

	res := apply(t, repo, o)
	for name, p := range map[string]string{"locked": locked, storepath.Name: emb, "beside": outside, "dirty": dirty} {
		r := rowFor(t, res, name)
		if r.Outcome != OutcomeKept || r.Reason == "" {
			t.Errorf("%s: %+v; want it kept, saying why", name, r)
		}
		if !exists(p) {
			t.Errorf("%s: the worktree stands", name)
		}
	}
	kept(t, rowFor(t, res, "locked"), "locked by git: on a removable disk")
	kept(t, rowFor(t, res, storepath.Name), "embargo store")
	kept(t, rowFor(t, res, "beside"), "outside the store")
	if len(*asked) != 0 {
		t.Fatalf("the store's prune is never asked about a kept worktree: asked %q", *asked)
	}
	if res.ExitCode() != ExitNothing {
		t.Fatalf("a run that acted on nothing exits %d; want %d", res.ExitCode(), ExitNothing)
	}
}

// TestApplyExitCodes: an applied run reads what was done, not what the dry
// run would do, and a prune that refuses or faults maps to the refusal's
// code, whatever was removed before it, with the worktree kept.
func TestApplyExitCodes(t *testing.T) {
	keptRemove := Row{Kind: RowWorktree, Action: ActionRemove, Outcome: OutcomeKept}
	removed := Row{Kind: RowWorktree, Action: ActionRemove, Outcome: OutcomeRemoved}
	for _, c := range []struct {
		name string
		res  Result
		want int
	}{
		{"a removal kept at the last check is no act", Result{Applied: true, Rows: []Row{keptRemove}}, ExitNothing},
		{"a removal", Result{Applied: true, Rows: []Row{keptRemove, removed}}, ExitActed},
		{"a fault outranks a removal", Result{Applied: true, Rows: []Row{removed}, Refusal: "fault"}, ExitRefused},
	} {
		if got := c.res.ExitCode(); got != c.want {
			t.Errorf("%s: ExitCode() = %d; want %d", c.name, got, c.want)
		}
	}

	t.Run("a prune that faults", func(t *testing.T) {
		repo, _ := sweepRepo(t)
		p := mergedAdd(t, repo, "done")
		o := sweepOpts()
		reads := 0
		o.Held = func(string) (store.Held, error) {
			if reads++; reads > 1 {
				return store.Held{}, errors.New("the run state is unreadable")
			}
			return store.Held{}, nil
		}
		res, err := Apply(context.Background(), repo.Root(), o)
		if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "the run state is unreadable") {
			t.Fatalf("Apply with a prune that faults = %+v, %v; want a fault naming why, exiting %d", res, err, ExitRefused)
		}
		r := rowFor(t, res, "done")
		if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the run state is unreadable") || !exists(p) {
			t.Fatalf("the worktree the prune faulted on: %+v; want it kept and standing, naming why", r)
		}
	})

	t.Run("a dry run that refuses", func(t *testing.T) {
		repo, _ := sweepRepo(t)
		o := sweepOpts()
		o.Held = nil
		res, err := Apply(context.Background(), repo.Root(), o)
		if err == nil || res.ExitCode() != ExitRefused {
			t.Fatalf("Apply with no reader = %+v, %v; want a refusal exiting %d", res, err, ExitRefused)
		}
	})
}

// TestADirtyWorktreeIsKeptAndSaidToBeUnsaved: patch salvage is not built, so
// the dry run says a dirty worktree is kept with its changes unsaved, never
// that they are saved, and the apply says the same.
func TestADirtyWorktreeIsKeptAndSaidToBeUnsaved(t *testing.T) {
	repo, _ := sweepRepo(t)
	p := laneAdd(t, repo, "dirty")
	write(t, filepath.Join(p, "scratch.txt"), "wip\n")

	dry := rowFor(t, dryRun(t, repo, sweepOpts()), "dirty")
	if dry.Action != ActionSavePatch || strings.Contains(dry.Reason, "saved as a patch") ||
		!strings.Contains(dry.Reason, "kept") || !strings.Contains(dry.Reason, "does not yet save") {
		t.Fatalf("the dry run's dirty row: %+v; want it kept, saying its changes are not saved yet", dry)
	}
	applied := rowFor(t, apply(t, repo, sweepOpts()), "dirty")
	if applied.Outcome != OutcomeKept || applied.Reason != dry.Reason {
		t.Fatalf("the apply's dirty row: %+v; want it kept with the dry run's reason %q", applied, dry.Reason)
	}
}

// failedApply is Apply on a run whose reclaim of the worktree name fails:
// the run faults, naming want, exits ExitRefused, and the worktree at p is
// kept and stands.
func failedApply(t *testing.T, repo *gittest.Repo, o Options, name, p, want string) {
	t.Helper()
	res, err := Apply(context.Background(), repo.Root(), o)
	if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, want) {
		t.Fatalf("Apply over a failed reclaim = %+v, %v; want a fault naming %q, exiting %d", res, err, want, ExitRefused)
	}
	r := rowFor(t, res, name)
	if r.Action != ActionRemove || r.Outcome != OutcomeKept || !strings.Contains(r.Reason, want) {
		t.Fatalf("the worktree whose reclaim failed: %+v; want it kept, naming %q", r, want)
	}
	if !exists(p) {
		t.Fatal("the worktree stands")
	}
}

// TestAFailedReclaimIsAFaultNotAKeep: the store keeps a worktree whose
// reclaim failed, with no error, as it keeps one it chose to; the sweep tells
// the two apart, and a failure is a fault exiting ExitRefused, the worktree
// kept.
func TestAFailedReclaimIsAFaultNotAKeep(t *testing.T) {
	t.Run("git refuses the removal", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root removes a directory it cannot write")
		}
		repo, _ := sweepRepo(t)
		p := laneAdd(t, repo, "stuck")
		commitAt(t, repo, p, "sub/f.txt")
		repo.Git("merge", "-q", "--ff-only", "stuck")
		sub := filepath.Join(p, "sub")
		if err := os.Chmod(sub, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })
		failedApply(t, repo, sweepOpts(), "stuck", p, "git refused the removal")
	})

	t.Run("the notes cannot be archived", func(t *testing.T) {
		repo, _ := sweepRepo(t)
		p := mergedAdd(t, repo, "binary")
		blob := filepath.Join(p, ".abcd", ".work.local", "blob.bin")
		if err := os.MkdirAll(filepath.Dir(blob), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(blob, []byte{0, 1, 2, 0xff}, 0o644); err != nil {
			t.Fatal(err)
		}
		failedApply(t, repo, sweepOpts(), "binary", p, "the notes could not be archived")
	})

	t.Run("a held worktree's archive entry cannot be taken back", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root removes a directory it cannot write")
		}
		repo, home := sweepRepo(t)
		p := mergedAdd(t, repo, "held")
		write(t, filepath.Join(p, ".abcd", ".work.local", "NEXT.md"), "next\n")
		calls := 0
		o := sweepOpts()
		o.Probe.Processes = func(context.Context) ([]Process, error) {
			if calls++; calls == 1 {
				return nil, nil
			}
			// The last check before removal: the entry is written, and its
			// parent is made read-only so taking it back fails.
			dirs, _ := filepath.Glob(filepath.Join(home, ".abcd.noindex", "notes", "*"))
			for _, d := range dirs {
				if err := os.Chmod(d, 0o500); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(d, 0o700) })
			}
			return []Process{{PID: 4321, Cwd: p}}, nil
		}
		failedApply(t, repo, o, "held", p, "could not be taken back")
	})
}
