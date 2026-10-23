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
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/lease"
)

// stale is a clock far enough ahead that nothing a test wrote is recent, so
// the recent-write hint stays out of a test about a proof.
func stale() func() time.Time {
	at := time.Now().Add(1000 * time.Hour)
	return func() time.Time { return at }
}

func at(t time.Time) func() time.Time { return func() time.Time { return t } }

func cwds(paths ...string) ProcessLister {
	var ps []Process
	for i, p := range paths {
		ps = append(ps, Process{PID: 100 + i, Cwd: p})
	}
	return func(context.Context) ([]Process, error) { return ps, nil }
}

// dirs is the working directories of a process listing, in order.
func dirs(ps []Process) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Cwd)
	}
	return out
}

// worktree makes a linked worktree's shape: a tree with a file and a .git file
// pointing at a git dir elsewhere, which holds the index.
func worktree(t *testing.T) (wt, gitDir string) {
	t.Helper()
	base := t.TempDir()
	wt = filepath.Join(base, "wt")
	gitDir = filepath.Join(base, "common", "worktrees", "wt")
	for _, d := range []string{filepath.Join(wt, "internal"), gitDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(wt, "internal", "a.go"), "package a\n")
	write(t, filepath.Join(wt, ".git"), "gitdir: "+gitDir+"\n")
	write(t, filepath.Join(gitDir, "index"), "DIRC")
	return wt, gitDir
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func kinds(l Liveness) []EvidenceKind {
	var out []EvidenceKind
	for _, e := range l.Evidence {
		out = append(out, e.Kind)
	}
	return out
}

// holdsBy asserts the worktree is held by exactly one piece of evidence, of
// the kind and strength given.
func holdsBy(t *testing.T, l Liveness, kind EvidenceKind, s Strength) {
	t.Helper()
	if !l.Held || len(l.Evidence) != 1 || l.Evidence[0].Kind != kind || l.Evidence[0].Strength != s {
		t.Fatalf("liveness = %+v; want held by one %s %s", l, s, kind)
	}
	if l.Reason == "" || l.Evidence[0].Detail == "" {
		t.Fatalf("liveness = %+v; want a reason and a detail a person can read", l)
	}
}

func frees(t *testing.T, l Liveness) {
	t.Helper()
	if l.Held || len(l.Evidence) != 0 {
		t.Fatalf("liveness = %+v; want free, with no evidence", l)
	}
	if l.Reason == "" {
		t.Fatalf("a free worktree carries no reason; want one saying nothing holds it")
	}
}

func TestNoProofAndNoHintFreesTheWorktree(t *testing.T) {
	wt, _ := worktree(t)
	p := NewProbe(context.Background(), ProbeOptions{Processes: cwds("/", "/usr"), Now: stale()})
	frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"internal/a.go"}}))
}

func TestALiveReservationWhoseFootprintCoversTheWorktreesChangesHoldsIt(t *testing.T) {
	wt, _ := worktree(t)
	now := stale()()
	live := claim.Reservation{Record: "iss-2610230000000001", Holder: "alice", State: claim.StateLive,
		Footprint: []string{"internal/core/sweep"}, EffectiveExpiry: now.Add(time.Hour)}
	snap := claim.Snapshot{Reservations: []claim.Reservation{live}}
	p := NewProbe(context.Background(), ProbeOptions{Reservations: snap, Processes: cwds(), Now: at(now)})
	l := p.Judge(Worktree{Path: wt, Changed: []string{"internal/core/sweep/liveness.go"}})
	holdsBy(t, l, KindReservation, Proven)
	if !strings.Contains(l.Reason, live.Record) || !strings.Contains(l.Reason, "alice") {
		t.Fatalf("reason = %q; want it to name the record and its holder", l.Reason)
	}

	// The footprint is matched on a path boundary: internal/core/sweeper is
	// not inside internal/core/sweep.
	frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"internal/core/sweeper/x.go"}}))

	// Only a live reservation is a proof.
	for _, state := range []string{claim.StateExpired, claim.StateReleased, claim.StateUnreadable} {
		r := live
		r.State = state
		p := NewProbe(context.Background(), ProbeOptions{Reservations: claim.Snapshot{Reservations: []claim.Reservation{r}},
			Processes: cwds(), Now: at(now)})
		frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"internal/core/sweep/liveness.go"}}))
	}
}

func run(wt string, updated time.Time, stage loop.Stage) loop.State {
	return loop.State{RunID: "run-1", Key: "itd-2610222214476887", CreatedAt: updated, UpdatedAt: updated,
		Lanes: []loop.Lane{{ID: "lane-1", Stage: stage, Worktree: wt}}}
}

func TestAnUnfinishedRunNamingTheWorktreeAsALaneHoldsItWithinTheLease(t *testing.T) {
	wt, _ := worktree(t)
	t0 := stale()()
	runs := []loop.State{run(wt, t0, loop.StageImplement)}

	in := NewProbe(context.Background(), ProbeOptions{Runs: runs, Processes: cwds(), Now: at(t0.Add(lease.Max - time.Second))})
	l := in.Judge(Worktree{Path: wt})
	holdsBy(t, l, KindRun, Proven)
	if !strings.Contains(l.Reason, "run-1") || !strings.Contains(l.Reason, "lane-1") {
		t.Fatalf("reason = %q; want it to name the run and the lane", l.Reason)
	}

	// The lease lapses a day after the run last wrote its state.
	out := NewProbe(context.Background(), ProbeOptions{Runs: runs, Processes: cwds(), Now: at(t0.Add(lease.Max))})
	frees(t, out.Judge(Worktree{Path: wt}))

	// A complete run holds nothing, within its lease or not.
	done := NewProbe(context.Background(), ProbeOptions{Runs: []loop.State{run(wt, t0, loop.StageDone)},
		Processes: cwds(), Now: at(t0.Add(time.Minute))})
	frees(t, done.Judge(Worktree{Path: wt}))

	// A lane naming a sibling whose name extends this one names another tree.
	other := NewProbe(context.Background(), ProbeOptions{Runs: []loop.State{run(wt+"c", t0, loop.StageImplement)},
		Processes: cwds(), Now: at(t0.Add(time.Minute))})
	frees(t, other.Judge(Worktree{Path: wt}))
}

func TestAProcessWorkingInsideTheWorktreeHoldsIt(t *testing.T) {
	wt, _ := worktree(t)
	inside := NewProbe(context.Background(), ProbeOptions{Processes: cwds("/", filepath.Join(wt, "internal")), Now: stale()})
	l := inside.Judge(Worktree{Path: wt})
	holdsBy(t, l, KindProcess, Proven)
	if !strings.Contains(l.Reason, "process 101") {
		t.Fatalf("reason = %q; want it to name the process by its pid", l.Reason)
	}

	onIt := NewProbe(context.Background(), ProbeOptions{Processes: cwds(wt), Now: stale()})
	holdsBy(t, onIt.Judge(Worktree{Path: wt}), KindProcess, Proven)

	// /a/b does not hold /a/bc, nor /a/bc /a/b.
	beside := NewProbe(context.Background(), ProbeOptions{Processes: cwds(wt + "c"), Now: stale()})
	frees(t, beside.Judge(Worktree{Path: wt}))
	above := NewProbe(context.Background(), ProbeOptions{Processes: cwds(filepath.Dir(wt)), Now: stale()})
	frees(t, above.Judge(Worktree{Path: wt}))
}

func TestPathsAreComparedWithSymlinksResolvedOnBothSides(t *testing.T) {
	wt, _ := worktree(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(wt, link); err != nil {
		t.Skipf("cannot make a symlink here: %v", err)
	}
	// The worktree named through the link, the process reported at the real
	// path, and the reverse.
	p := NewProbe(context.Background(), ProbeOptions{Processes: cwds(filepath.Join(wt, "internal")), Now: stale()})
	holdsBy(t, p.Judge(Worktree{Path: link}), KindProcess, Proven)
	p = NewProbe(context.Background(), ProbeOptions{Processes: cwds(filepath.Join(link, "internal")), Now: stale()})
	holdsBy(t, p.Judge(Worktree{Path: wt}), KindProcess, Proven)

	t0 := stale()()
	p = NewProbe(context.Background(), ProbeOptions{Runs: []loop.State{run(link, t0, loop.StageValidate)},
		Processes: cwds(), Now: at(t0.Add(time.Minute))})
	holdsBy(t, p.Judge(Worktree{Path: wt}), KindRun, Proven)
}

func TestARecentWriteToTheTreeKeepsItAsAHint(t *testing.T) {
	wt, _ := worktree(t)
	p := NewProbe(context.Background(), ProbeOptions{Processes: cwds(), Now: time.Now})
	holdsBy(t, p.Judge(Worktree{Path: wt}), KindRecentWrite, Hint)

	// The window is the caller's: an hour later, an hour's window has passed.
	p = NewProbe(context.Background(), ProbeOptions{Processes: cwds(), Now: at(time.Now().Add(2 * time.Hour)), RecentWindow: time.Hour})
	frees(t, p.Judge(Worktree{Path: wt}))
	if DefaultRecentWindow != 24*time.Hour {
		t.Fatalf("DefaultRecentWindow = %s; want the spec's 24h", DefaultRecentWindow)
	}
}

func TestARecentWriteToALinkedWorktreesIndexKeepsItAsAHint(t *testing.T) {
	wt, gitDir := worktree(t)
	old := time.Now().Add(-72 * time.Hour)
	for _, p := range []string{filepath.Join(wt, "internal", "a.go"), filepath.Join(wt, "internal"), filepath.Join(wt, ".git"), wt} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	p := NewProbe(context.Background(), ProbeOptions{Processes: cwds(), Now: time.Now})
	l := p.Judge(Worktree{Path: wt})
	holdsBy(t, l, KindRecentWrite, Hint)
	if !strings.Contains(l.Reason, "index") {
		t.Fatalf("reason = %q; want it to say the index was written", l.Reason)
	}

	// With the index old as well, nothing keeps it.
	if err := os.Chtimes(filepath.Join(gitDir, "index"), old, old); err != nil {
		t.Fatal(err)
	}
	frees(t, p.Judge(Worktree{Path: wt}))
}

func TestAProofOutranksTheHint(t *testing.T) {
	wt, _ := worktree(t)
	p := NewProbe(context.Background(), ProbeOptions{Processes: cwds(wt), Now: time.Now})
	l := p.Judge(Worktree{Path: wt})
	if !l.Held || l.Evidence[0].Strength != Proven {
		t.Fatalf("liveness = %+v; want a proof listed first", l)
	}
}

func TestAFailedProcessListerFailsClosed(t *testing.T) {
	wt, _ := worktree(t)
	p := NewProbe(context.Background(), ProbeOptions{
		Processes: func(context.Context) ([]Process, error) { return nil, errors.New("lsof timed out after 5s") },
		Now:       stale(),
	})
	if got := p.ProcessCheckUnavailable(); got != "lsof timed out after 5s" {
		t.Fatalf("ProcessCheckUnavailable() = %q; want the lister's why", got)
	}
	l := p.Judge(Worktree{Path: wt})
	holdsBy(t, l, KindProcessCheck, Hint)
	if l.Reason != "process check unavailable: lsof timed out after 5s" {
		t.Fatalf("reason = %q; want %q", l.Reason, "process check unavailable: lsof timed out after 5s")
	}
}

// script writes an executable shell script standing in for lsof.
func script(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in lsof is a shell script")
	}
	path := filepath.Join(t.TempDir(), "lsof")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheLsofProbeFailsClosed(t *testing.T) {
	wt, _ := worktree(t)
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
		{"non-zero exit", func(t *testing.T) lsofCommand {
			return lsofCommand{path: script(t, "printf 'p1\\nfcwd\\nn/\\n'; exit 3"), timeout: 5 * time.Second, limit: 1 << 20}
		}, "exited with status 3"},
		{"garbage", func(t *testing.T) lsofCommand {
			return lsofCommand{path: script(t, "echo hello"), timeout: 5 * time.Second, limit: 1 << 20}
		}, "unreadable output"},
		{"over the read bound", func(t *testing.T) lsofCommand {
			return lsofCommand{path: script(t, "i=0; while [ $i -lt 100 ]; do printf 'p%d\\nfcwd\\nn/\\n' $i; i=$((i+1)); done"),
				timeout: 5 * time.Second, limit: 64}
		}, "more than 64 bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start := time.Now()
			p := NewProbe(context.Background(), ProbeOptions{Processes: c.cmd(t).list, Now: stale()})
			if took := time.Since(start); took > 3*time.Second {
				t.Fatalf("the probe took %s; want it bounded by its deadline", took)
			}
			why := p.ProcessCheckUnavailable()
			if !strings.Contains(why, c.want) {
				t.Fatalf("ProcessCheckUnavailable() = %q; want it to contain %q", why, c.want)
			}
			l := p.Judge(Worktree{Path: wt})
			holdsBy(t, l, KindProcessCheck, Hint)
			if l.Reason != "process check unavailable: "+why {
				t.Fatalf("reason = %q; want %q", l.Reason, "process check unavailable: "+why)
			}
		})
	}
}

func TestTheLsofProbeReadsWellFormedOutput(t *testing.T) {
	cmd := lsofCommand{path: script(t, "printf 'p12\\nfcwd\\nn/tmp/a\\np13\\nfcwd\\nn/tmp/b c\\n'"), timeout: 5 * time.Second, limit: 1 << 20}
	got, err := cmd.list(context.Background())
	if err != nil || !slices.Equal(got, []Process{{PID: 12, Cwd: "/tmp/a"}, {PID: 13, Cwd: "/tmp/b c"}}) {
		t.Fatalf("list() = %v, %v; want [/tmp/a /tmp/b c]", got, err)
	}
}

func TestParseLsof(t *testing.T) {
	good := []struct {
		name, out string
		want      []string
	}{
		{"one process", "p1\nfcwd\nn/\n", []string{"/"}},
		{"several", "p1\nfcwd\nn/\np22\nfcwd\nn/Users/x/wt\n", []string{"/", "/Users/x/wt"}},
		{"no trailing newline", "p1\nfcwd\nn/a", []string{"/a"}},
		{"a process with no name", "p1\nfcwd\np2\nfcwd\nn/b\n", []string{"/b"}},
		{"a descriptor other than cwd", "p1\nftxt\nn/bin/sh\nfcwd\nn/c\n", []string{"/c"}},
	}
	for _, c := range good {
		got, err := parseLsof([]byte(c.out))
		if err != nil || !slices.Equal(dirs(got), c.want) {
			t.Errorf("%s: parseLsof(%q) = %v, %v; want %q", c.name, c.out, got, err, c.want)
		}
	}
	bad := []struct{ name, out string }{
		{"empty: lsof always lists itself", ""},
		{"an unknown field", "p1\nfcwd\nxwhat\n"},
		{"a pid that is not a number", "pabc\nfcwd\nn/\n"},
		{"a name before any process", "n/\n"},
		{"a descriptor before any process", "fcwd\nn/\n"},
		{"a name before its descriptor", "p1\nn/\n"},
		{"a relative name", "p1\nfcwd\nnrelative\n"},
		{"a blank line", "p1\n\nfcwd\nn/\n"},
		{"a NUL", "p1\nfcwd\nn/a\x00b\n"},
	}
	for _, c := range bad {
		if got, err := parseLsof([]byte(c.out)); err == nil {
			t.Errorf("%s: parseLsof(%q) = %v, nil; want an error", c.name, c.out, got)
		}
	}
}

func TestProcCwdsReadsAProcTree(t *testing.T) {
	proc := t.TempDir()
	target := t.TempDir()
	for _, d := range []string{"123", "456", "self", "net"} {
		if err := os.Mkdir(filepath.Join(proc, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(target, filepath.Join(proc, "123", "cwd")); err != nil {
		t.Skipf("cannot make a symlink here: %v", err)
	}
	// 456 has no cwd link, as a process that exited mid-listing; self and net
	// are not processes.
	if err := os.Symlink("/elsewhere", filepath.Join(proc, "self", "cwd")); err != nil {
		t.Fatal(err)
	}
	got, err := procCwds(proc)
	if err != nil || !slices.Equal(got, []Process{{PID: 123, Cwd: target}}) {
		t.Fatalf("procCwds = %v, %v; want [%s]", got, err, target)
	}

	if _, err := procCwds(filepath.Join(proc, "absent")); err == nil || !strings.Contains(err.Error(), "cannot read") {
		t.Fatalf("procCwds over an unreadable /proc = %v; want a refusal saying it cannot read it", err)
	}
}

// TestTheHostProbeFindsARealChildProcess starts a real process with its
// working directory in a temp dir and proves the host's own probe finds it.
func TestTheHostProbeFindsARealChildProcess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no process check on %s", runtime.GOOS)
	}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep to start: %v", err)
	}
	dir := t.TempDir()
	child := exec.Command(sleep, "30")
	child.Dir = dir
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = child.Process.Kill()
		_ = child.Wait()
	})
	p := NewProbe(context.Background(), ProbeOptions{Now: stale()})
	if why := p.ProcessCheckUnavailable(); why != "" {
		t.Skipf("the host probe is unavailable here: %s", why)
	}
	l := p.Judge(Worktree{Path: dir})
	if !l.Held || !slices.Contains(kinds(l), KindProcess) {
		t.Fatalf("liveness of the child's working directory = %+v; want held by a process", l)
	}
}
