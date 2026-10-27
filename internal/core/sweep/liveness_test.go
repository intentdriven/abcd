package sweep

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
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

// none is a worktree that changes nothing, as against nil, which is changes
// the caller could not compute.
var none = []string{}

// probe is newProbe over reservations stamped as fetched, so a test about
// another proof is not held by the unread-reservations rule.
func probe(o ProbeOptions) *Probe {
	if o.Reservations.FetchedAt.IsZero() {
		o.Reservations.FetchedAt = time.Unix(1, 0)
	}
	return newProbe(context.Background(), o)
}

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
	p := probe(ProbeOptions{Processes: cwds("/", "/usr"), Now: stale()})
	frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"internal/a.go"}}))
}

func TestALiveReservationWhoseFootprintCoversTheWorktreesChangesHoldsIt(t *testing.T) {
	wt, _ := worktree(t)
	now := stale()()
	live := claim.Reservation{Record: "iss-2610230000000001", Holder: "alice", State: claim.StateLive,
		Footprint: []string{"internal/core/sweep"}, EffectiveExpiry: now.Add(time.Hour)}
	snap := claim.Snapshot{Reservations: []claim.Reservation{live}}
	p := probe(ProbeOptions{Reservations: snap, Processes: cwds(), Now: at(now)})
	l := p.Judge(Worktree{Path: wt, Changed: []string{"internal/core/sweep/liveness.go"}})
	holdsBy(t, l, KindReservation, Proven)
	if !strings.Contains(l.Reason, live.Record) || !strings.Contains(l.Reason, "alice") {
		t.Fatalf("reason = %q; want it to name the record and its holder", l.Reason)
	}

	// The footprint is matched on a path boundary: internal/core/sweeper is
	// not inside internal/core/sweep.
	frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"internal/core/sweeper/x.go"}}))

	// Only a live reservation is a proof. (An unreadable one within its hold
	// is a hint: TestAnUnreadableReservationWithinItsHoldKeepsEveryWorktree.)
	for _, state := range []string{claim.StateExpired, claim.StateReleased} {
		r := live
		r.State = state
		p := probe(ProbeOptions{Reservations: claim.Snapshot{Reservations: []claim.Reservation{r}},
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

	in := probe(ProbeOptions{Runs: runs, Processes: cwds(), Now: at(t0.Add(lease.Max - time.Second))})
	l := in.Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindRun, Proven)
	if !strings.Contains(l.Reason, "run-1") || !strings.Contains(l.Reason, "lane-1") {
		t.Fatalf("reason = %q; want it to name the run and the lane", l.Reason)
	}

	// The lease lapses a day after the run last wrote its state.
	out := probe(ProbeOptions{Runs: runs, Processes: cwds(), Now: at(t0.Add(lease.Max))})
	frees(t, out.Judge(Worktree{Path: wt, Changed: none}))

	// A complete run holds nothing, within its lease or not.
	done := probe(ProbeOptions{Runs: []loop.State{run(wt, t0, loop.StageDone)},
		Processes: cwds(), Now: at(t0.Add(time.Minute))})
	frees(t, done.Judge(Worktree{Path: wt, Changed: none}))

	// A lane naming a sibling whose name extends this one names another tree.
	other := probe(ProbeOptions{Runs: []loop.State{run(wt+"c", t0, loop.StageImplement)},
		Processes: cwds(), Now: at(t0.Add(time.Minute))})
	frees(t, other.Judge(Worktree{Path: wt, Changed: none}))
}

func TestAProcessWorkingInsideTheWorktreeHoldsIt(t *testing.T) {
	wt, _ := worktree(t)
	inside := probe(ProbeOptions{Processes: cwds("/", filepath.Join(wt, "internal")), Now: stale()})
	l := inside.Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindProcess, Proven)
	if l.Reason != "in use: process 101 works in it" {
		t.Fatalf("reason = %q; want %q", l.Reason, "in use: process 101 works in it")
	}

	onIt := probe(ProbeOptions{Processes: cwds(wt), Now: stale()})
	holdsBy(t, onIt.Judge(Worktree{Path: wt, Changed: none}), KindProcess, Proven)

	// /a/b does not hold /a/bc, nor /a/bc /a/b.
	beside := probe(ProbeOptions{Processes: cwds(wt + "c"), Now: stale()})
	frees(t, beside.Judge(Worktree{Path: wt, Changed: none}))
	above := probe(ProbeOptions{Processes: cwds(filepath.Dir(wt)), Now: stale()})
	frees(t, above.Judge(Worktree{Path: wt, Changed: none}))
}

func TestPathsAreComparedWithSymlinksResolvedOnBothSides(t *testing.T) {
	wt, _ := worktree(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(wt, link); err != nil {
		t.Skipf("cannot make a symlink here: %v", err)
	}
	// The worktree named through the link, the process reported at the real
	// path, and the reverse.
	p := probe(ProbeOptions{Processes: cwds(filepath.Join(wt, "internal")), Now: stale()})
	holdsBy(t, p.Judge(Worktree{Path: link, Changed: none}), KindProcess, Proven)
	p = probe(ProbeOptions{Processes: cwds(filepath.Join(link, "internal")), Now: stale()})
	holdsBy(t, p.Judge(Worktree{Path: wt, Changed: none}), KindProcess, Proven)

	t0 := stale()()
	p = probe(ProbeOptions{Runs: []loop.State{run(link, t0, loop.StageValidate)},
		Processes: cwds(), Now: at(t0.Add(time.Minute))})
	holdsBy(t, p.Judge(Worktree{Path: wt, Changed: none}), KindRun, Proven)
}

func TestARecentWriteToTheTreeKeepsItAsAHint(t *testing.T) {
	wt, _ := worktree(t)
	p := probe(ProbeOptions{Processes: cwds(), Now: time.Now})
	holdsBy(t, p.Judge(Worktree{Path: wt, Changed: none}), KindRecentWrite, Hint)

	// The window is the caller's: an hour later, an hour's window has passed.
	p = probe(ProbeOptions{Processes: cwds(), Now: at(time.Now().Add(2 * time.Hour)), RecentWindow: time.Hour})
	frees(t, p.Judge(Worktree{Path: wt, Changed: none}))
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
	p := probe(ProbeOptions{Processes: cwds(), Now: time.Now})
	l := p.Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindRecentWrite, Hint)
	if !strings.Contains(l.Reason, "index") {
		t.Fatalf("reason = %q; want it to say the index was written", l.Reason)
	}

	// With the index old as well, nothing keeps it.
	if err := os.Chtimes(filepath.Join(gitDir, "index"), old, old); err != nil {
		t.Fatal(err)
	}
	frees(t, p.Judge(Worktree{Path: wt, Changed: none}))
}

func TestAProofOutranksTheHint(t *testing.T) {
	wt, _ := worktree(t)
	p := probe(ProbeOptions{Processes: cwds(wt), Now: time.Now})
	l := p.Judge(Worktree{Path: wt, Changed: none})
	if !l.Held || l.Evidence[0].Strength != Proven {
		t.Fatalf("liveness = %+v; want a proof listed first", l)
	}
}

func TestAFailedProcessListerFailsClosed(t *testing.T) {
	wt, _ := worktree(t)
	p := probe(ProbeOptions{
		Processes: func(context.Context) ([]Process, error) { return nil, errors.New("lsof timed out after 5s") },
		Now:       stale(),
	})
	if got := p.ProcessCheckUnavailable(); got != "lsof timed out after 5s" {
		t.Fatalf("ProcessCheckUnavailable() = %q; want the lister's why", got)
	}
	l := p.Judge(Worktree{Path: wt, Changed: none})
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
			p := probe(ProbeOptions{Processes: c.cmd(t).list, Now: stale()})
			if took := time.Since(start); took > 3*time.Second {
				t.Fatalf("the probe took %s; want it bounded by its deadline", took)
			}
			why := p.ProcessCheckUnavailable()
			if !strings.Contains(why, c.want) {
				t.Fatalf("ProcessCheckUnavailable() = %q; want it to contain %q", why, c.want)
			}
			l := p.Judge(Worktree{Path: wt, Changed: none})
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
		{"a relative name with an escape in it", "p1\nfcwd\nnrel^Ative\n"},
	}
	for _, c := range bad {
		if got, err := parseLsof([]byte(c.out)); err == nil {
			t.Errorf("%s: parseLsof(%q) = %v, nil; want an error", c.name, c.out, got)
		}
	}
}

// TestParseLsofKeepsTheLiteralPrefixOfAnEscapedName: every byte before the
// first backslash or caret is the directory's own, so that prefix is kept and
// marked, rather than the name refusing the whole check.
func TestParseLsofKeepsTheLiteralPrefixOfAnEscapedName(t *testing.T) {
	cases := []struct{ name, out, lit string }{
		{"a byte escaped in hex", `p1` + "\nfcwd\n" + `n/tmp/\xc3\xa9-wt` + "\n", "/tmp/"},
		{"a newline escaped", `p1` + "\nfcwd\n" + `n/tmp/a\nb` + "\n", "/tmp/a"},
		{"a backslash doubled", `p1` + "\nfcwd\n" + `n/tmp/wt/a\\b` + "\n", "/tmp/wt/a"},
		{"a control byte in caret form", "p1\nfcwd\nn/tmp/wt/x^Ay\n", "/tmp/wt/x"},
		{"an escape in the first name", "p1\nfcwd\nn/^[x\n", "/"},
	}
	for _, c := range cases {
		got, err := parseLsof([]byte(c.out))
		if want := []Process{{PID: 1, Cwd: c.lit, Escaped: true}}; err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: parseLsof(%q) = %v, %v; want %v", c.name, c.out, got, err, want)
		}
	}
}

// lsofSays is a process lister that reads the lsof output given.
func lsofSays(out string) ProcessLister {
	return func(context.Context) ([]Process, error) { return parseLsof([]byte(out)) }
}

func TestAProcessInAnEscapedDirectoryHoldsWhatItsLiteralPrefixCovers(t *testing.T) {
	wt, _ := worktree(t)
	real, err := filepath.EvalSymlinks(wt)
	if err != nil {
		t.Fatal(err)
	}
	escapedUnder := func(lit string) string {
		return "in use: a process works in a directory lsof printed escaped, under " + lit
	}
	holds := func(t *testing.T, out, lit string, w string) {
		t.Helper()
		p := probe(ProbeOptions{Processes: lsofSays(out), Now: stale()})
		if why := p.ProcessCheckUnavailable(); why != "" {
			t.Fatalf("ProcessCheckUnavailable() = %q; want the check made", why)
		}
		l := p.Judge(Worktree{Path: w, Changed: none})
		holdsBy(t, l, KindProcess, Hint)
		if l.Reason != escapedUnder(lit) {
			t.Fatalf("reason = %q; want %q", l.Reason, escapedUnder(lit))
		}
	}

	t.Run("a caret name under the worktree", func(t *testing.T) {
		holds(t, "p7\nfcwd\nn"+real+"/internal/x^Ay\n", real+"/internal/x", wt)
	})
	t.Run("a backslash name under the worktree", func(t *testing.T) {
		holds(t, "p7\nfcwd\nn"+real+`/internal/a\\b`+"\n", real+"/internal/a", wt)
	})
	t.Run("an escape that cuts the worktree's own name in half", func(t *testing.T) {
		lit := filepath.Join(filepath.Dir(real), "w")
		holds(t, "p7\nfcwd\nn"+lit+"^At/internal\n", lit, wt)
	})
	t.Run("an escape right after the worktree's own name", func(t *testing.T) {
		holds(t, "p7\nfcwd\nn"+real+"/^A\n", real+"/", wt)
		// The slash is part of the prefix: a sibling whose name extends the
		// worktree's is not under it.
		if err := os.Mkdir(real+"c", 0o755); err != nil {
			t.Fatal(err)
		}
		old := time.Now().Add(-72 * time.Hour)
		if err := os.Chtimes(real+"c", old, old); err != nil {
			t.Fatal(err)
		}
		p := probe(ProbeOptions{Processes: lsofSays("p7\nfcwd\nn" + real + "/^A\n"), Now: stale()})
		frees(t, p.Judge(Worktree{Path: real + "c", Changed: none}))
	})
	t.Run("a prefix that runs past the worktree into a sibling", func(t *testing.T) {
		// The bytes before the escape name real+"-2", so the process is in that
		// sibling, not in the worktree its name extends.
		p := probe(ProbeOptions{Processes: lsofSays("p7\nfcwd\nn" + real + "-2/d^A\n"), Now: stale()})
		frees(t, p.Judge(Worktree{Path: wt, Changed: none}))
	})
	t.Run("an escaped name under an unrelated directory", func(t *testing.T) {
		other, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		p := probe(ProbeOptions{Processes: lsofSays("p7\nfcwd\nn" + other + `/x\\y` + "\np8\nfcwd\nn" + real + "\n"), Now: stale()})
		if why := p.ProcessCheckUnavailable(); why != "" {
			t.Fatalf("ProcessCheckUnavailable() = %q; want the check still made", why)
		}
		elsewhere, _ := worktree(t)
		frees(t, p.Judge(Worktree{Path: elsewhere, Changed: none}))
		// The plain name in the same listing still proves its worktree held.
		holdsBy(t, p.Judge(Worktree{Path: wt, Changed: none}), KindProcess, Proven)
	})
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
	self := strconv.Itoa(os.Getpid())
	if err := os.Mkdir(filepath.Join(proc, self), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(proc, self, "cwd")); err != nil {
		t.Fatal(err)
	}
	got, err := procCwds(proc)
	slices.SortFunc(got, func(a, b Process) int { return a.PID - b.PID })
	want := []Process{{PID: 123, Cwd: target}, {PID: os.Getpid(), Cwd: target}}
	slices.SortFunc(want, func(a, b Process) int { return a.PID - b.PID })
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("procCwds = %v, %v; want %v", got, err, want)
	}

	if _, err := procCwds(filepath.Join(proc, "absent")); err == nil || !strings.Contains(err.Error(), "cannot read") {
		t.Fatalf("procCwds over an unreadable /proc = %v; want a refusal saying it cannot read it", err)
	}
}

// TestAProcLinkRefusedForThisAccountFailsClosed: a kernel refuses the cwd
// link of a process that is not dumpable, this account's own included, so a
// refused link is passed over only when the status names another account.
func TestAProcLinkRefusedForThisAccountFailsClosed(t *testing.T) {
	proc := t.TempDir()
	self := strconv.Itoa(os.Getpid())
	for _, d := range []string{self, "700", "701"} {
		if err := os.Mkdir(filepath.Join(proc, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(proc, self, "cwd")); err != nil {
		t.Skipf("cannot make a symlink here: %v", err)
	}
	status := func(pid string, uid int) {
		t.Helper()
		body := fmt.Sprintf("Name:\tx\nUid:\t%d\t%d\t%d\t%d\n", uid, uid, uid, uid)
		if err := os.WriteFile(filepath.Join(proc, pid, "status"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	status("700", os.Getuid()+1)
	prev := procReadlink
	t.Cleanup(func() { procReadlink = prev })
	procReadlink = func(path string) (string, error) {
		if strings.HasSuffix(path, "/700/cwd") || strings.HasSuffix(path, "/701/cwd") {
			return "", &fs.PathError{Op: "readlink", Path: path, Err: fs.ErrPermission}
		}
		return os.Readlink(path)
	}

	// 700 is another account's and 701 has exited: both are passed over.
	if got, err := procCwds(proc); err != nil || len(got) != 1 {
		t.Fatalf("procCwds = %v, %v; want this process alone", got, err)
	}
	// 701 is this account's and refuses its link: the check fails closed.
	status("701", os.Getuid())
	if _, err := procCwds(proc); err == nil || !strings.Contains(err.Error(), "process 701") {
		t.Fatalf("procCwds over this account's refused link = %v; want a refusal naming it", err)
	}
	// A status that names no owner fails closed too.
	if err := os.WriteFile(filepath.Join(proc, "701", "status"), []byte("Name:\tx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := procCwds(proc); err == nil {
		t.Fatal("procCwds over a refused link with no readable owner did not refuse")
	}
}

// TestAProcTreeWithoutThisProcessFailsClosed: a /proc that does not list the
// process reading it is empty or another's, as in a chroot, a container or
// under hidepid, so what it lists cannot show that nothing is running.
func TestAProcTreeWithoutThisProcessFailsClosed(t *testing.T) {
	proc := t.TempDir()
	if err := os.Mkdir(filepath.Join(proc, "123"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(proc, "123", "cwd")); err != nil {
		t.Skipf("cannot make a symlink here: %v", err)
	}
	p := probe(ProbeOptions{Processes: func(context.Context) ([]Process, error) { return procCwds(proc) }, Now: stale()})
	if why, want := p.ProcessCheckUnavailable(), "this process is not in "+proc; why != want {
		t.Fatalf("ProcessCheckUnavailable() = %q; want %q", why, want)
	}
	wt, _ := worktree(t)
	l := p.Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindProcessCheck, Hint)
	if want := "process check unavailable: this process is not in " + proc; l.Reason != want {
		t.Fatalf("reason = %q; want %q", l.Reason, want)
	}

	// An empty /proc fails the same way.
	if _, err := procCwds(t.TempDir()); err == nil || !strings.Contains(err.Error(), "this process is not in") {
		t.Fatalf("procCwds over an empty tree = %v; want this process named missing", err)
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
	p := probe(ProbeOptions{Now: stale()})
	if why := p.ProcessCheckUnavailable(); why != "" {
		t.Skipf("the host probe is unavailable here: %s", why)
	}
	l := p.Judge(Worktree{Path: dir, Changed: none})
	if !l.Held || !slices.Contains(kinds(l), KindProcess) {
		t.Fatalf("liveness of the child's working directory = %+v; want held by a process", l)
	}
}

func TestTheLsofProbeRunsUnderAUTF8LocaleAndKeepsAnEscapedNamesPrefix(t *testing.T) {
	// Run with no locale set, lsof escapes a non-ASCII name (\xc3\xa9) and
	// still exits 0; the child must get a UTF-8 locale whatever the parent has.
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "")
	t.Setenv("LC_CTYPE", "")
	cmd := lsofCommand{path: script(t, `[ "$LC_ALL" = en_US.UTF-8 ] || exit 7; printf 'p1\nfcwd\nn/tmp/é\n'`), timeout: 5 * time.Second, limit: 1 << 20}
	got, err := cmd.list(context.Background())
	if err != nil || !slices.Equal(dirs(got), []string{"/tmp/é"}) {
		t.Fatalf("list() = %v, %v; want [/tmp/é], read under a UTF-8 locale", got, err)
	}

	// A name lsof escaped anyway keeps only its literal prefix, marked, and
	// leaves the check standing for every other name.
	escaped := lsofCommand{path: script(t, `printf 'p1\nfcwd\nn/tmp/\\xc3\\xa9-wt\n'`), timeout: 5 * time.Second, limit: 1 << 20}
	got, err = escaped.list(context.Background())
	if want := []Process{{PID: 1, Cwd: "/tmp/", Escaped: true}}; err != nil || !slices.Equal(got, want) {
		t.Fatalf("list() = %v, %v; want %v", got, err, want)
	}
}

func TestTheLsofProbeFailsClosedOnAChildHoldingItsOutput(t *testing.T) {
	// lsof exits at once, but a child it left keeps the pipe open: WaitDelay
	// ends the wait, and the check fails closed rather than reading a part.
	cmd := lsofCommand{path: script(t, `sleep 4 & printf 'p1\nfcwd\nn/\n'; exit 0`), timeout: 10 * time.Second, limit: 1 << 20}
	start := time.Now()
	p := probe(ProbeOptions{Processes: cmd.list, Now: stale()})
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("the probe took %s; want WaitDelay to bound it", took)
	}
	if why := p.ProcessCheckUnavailable(); !strings.Contains(why, "holding its output") {
		t.Fatalf("ProcessCheckUnavailable() = %q; want a child holding lsof's output named", why)
	}
}

// TestTheHostProbeFindsANonASCIIWorkingDirectory proves the host probe finds a
// real process in a directory whose name is not ASCII, with the parent's
// locale cleared, as a session started with no LANG would run it.
func TestTheHostProbeFindsANonASCIIWorkingDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no process check on %s", runtime.GOOS)
	}
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "")
	t.Setenv("LC_CTYPE", "")
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep to start: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "café wt")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Skipf("cannot make a non-ASCII directory here: %v", err)
	}
	child := exec.Command(sleep, "30")
	child.Dir = dir
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = child.Process.Kill()
		_ = child.Wait()
	})
	p := probe(ProbeOptions{Now: stale()})
	if why := p.ProcessCheckUnavailable(); why != "" {
		t.Fatalf("the host probe is unavailable: %s", why)
	}
	if l := p.Judge(Worktree{Path: dir, Changed: none}); !slices.Contains(kinds(l), KindProcess) {
		t.Fatalf("liveness of %q = %+v; want held by its process", dir, l)
	}
}

// TestTheHostProbeHoldsAWorktreeUnderADirectoryLsofEscapes proves a real
// process in a directory whose name lsof escapes (a backslash) still holds the
// worktree it works in, and leaves the process check standing.
func TestTheHostProbeHoldsAWorktreeUnderADirectoryLsofEscapes(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no process check on %s", runtime.GOOS)
	}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep to start: %v", err)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	dir := filepath.Join(wt, `back\slash`)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skipf("cannot make a directory with a backslash here: %v", err)
	}
	child := exec.Command(sleep, "30")
	child.Dir = dir
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = child.Process.Kill()
		_ = child.Wait()
	})
	p := probe(ProbeOptions{Now: stale()})
	if why := p.ProcessCheckUnavailable(); why != "" {
		t.Fatalf("the host probe is unavailable: %s", why)
	}
	l := p.Judge(Worktree{Path: wt, Changed: none})
	if !slices.Contains(kinds(l), KindProcess) {
		t.Fatalf("liveness of %q = %+v; want held by its process", wt, l)
	}
	// lsof doubles the backslash, so on macOS the hold comes from the prefix.
	if runtime.GOOS == "darwin" && !strings.Contains(l.Reason, "printed escaped") {
		t.Fatalf("reason = %q; want the escaped directory named", l.Reason)
	}
}

// own is a per-worktree view holding the reservations given, for the worktree
// named, and recording which worktrees it was asked about.
func own(asked *[]string, rs ...claim.Reservation) func(string, time.Time) (claim.Snapshot, error) {
	return func(path string, _ time.Time) (claim.Snapshot, error) {
		*asked = append(*asked, path)
		return claim.Snapshot{Reservations: rs}, nil
	}
}

func TestAReservationMadeFromTheWorktreeHoldsItBeforeItChangesAnything(t *testing.T) {
	wt, _ := worktree(t)
	now := stale()()
	mine := claim.Reservation{Record: "itd-2610230000000002", Holder: "bob", Copy: claim.CopyThis,
		State: claim.StateLive, EffectiveExpiry: now.Add(time.Hour)}
	var asked []string
	p := probe(ProbeOptions{CopyView: own(&asked, mine), Processes: cwds(), Now: at(now)})
	l := p.Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindReservation, Proven)
	if !strings.Contains(l.Reason, mine.Record) || !strings.Contains(l.Reason, "bob") {
		t.Fatalf("reason = %q; want it to name the record and its holder", l.Reason)
	}
	if real, _ := filepath.EvalSymlinks(wt); len(asked) != 1 || asked[0] != real {
		t.Fatalf("the per-worktree view was asked about %q; want [%s]", asked, real)
	}

	// Another copy's reservation with no overlap, or this copy's once it is
	// no longer live, holds nothing.
	other := mine
	other.Copy = ""
	frees(t, probe(ProbeOptions{CopyView: own(&asked, other), Processes: cwds(), Now: at(now)}).Judge(Worktree{Path: wt, Changed: none}))
	gone := mine
	gone.State = claim.StateExpired
	frees(t, probe(ProbeOptions{CopyView: own(&asked, gone), Processes: cwds(), Now: at(now)}).Judge(Worktree{Path: wt, Changed: none}))

	// The same reservation found both ways is named once.
	mine.Footprint = []string{"internal"}
	l = probe(ProbeOptions{Reservations: claim.Snapshot{Reservations: []claim.Reservation{mine}}, CopyView: own(&asked, mine),
		Processes: cwds(), Now: at(now)}).Judge(Worktree{Path: wt, Changed: []string{"internal/a.go"}})
	holdsBy(t, l, KindReservation, Proven)

	// A per-worktree view that cannot be read keeps the worktree.
	broken := func(string, time.Time) (claim.Snapshot, error) { return claim.Snapshot{}, errors.New("git failed") }
	l = probe(ProbeOptions{CopyView: broken, Processes: cwds(), Now: at(now)}).Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindReservationCheck, Hint)
	if !strings.Contains(l.Reason, "git failed") {
		t.Fatalf("reason = %q; want the view's failure named", l.Reason)
	}
}

func TestTheDefaultCopyViewReadsNothingWhereNoCopyKeyIsKept(t *testing.T) {
	snap, err := worktreeView(t.TempDir(), time.Now())
	if err != nil || len(snap.Reservations) != 0 {
		t.Fatalf("worktreeView over a tree with no copy key = %+v, %v; want nothing, no error", snap, err)
	}
}

func TestChangesTheCallerCouldNotComputeKeepTheWorktree(t *testing.T) {
	wt, _ := worktree(t)
	p := probe(ProbeOptions{Processes: cwds(), Now: stale()})
	l := p.Judge(Worktree{Path: wt})
	holdsBy(t, l, KindChanges, Hint)
	if l.Reason != "changes unknown" {
		t.Fatalf("reason = %q; want %q", l.Reason, "changes unknown")
	}
	frees(t, p.Judge(Worktree{Path: wt, Changed: none}))
}

func TestAChangedPathThatCannotBeReadKeepsTheWorktree(t *testing.T) {
	wt, _ := worktree(t)
	p := probe(ProbeOptions{Processes: cwds(), Now: stale()})
	l := p.Judge(Worktree{Path: wt, Changed: []string{"internal/a.go", `docs/a\b.md`}})
	holdsBy(t, l, KindChanges, Hint)
	if want := `changes unreadable: docs/a\b.md`; l.Reason != want {
		t.Fatalf("reason = %q; want %q", l.Reason, want)
	}
	for _, bad := range []string{"../x", "/abs", "a//b"} {
		holdsBy(t, p.Judge(Worktree{Path: wt, Changed: []string{bad}}), KindChanges, Hint)
	}
	// "" and "." name no path at all, and stay passed over.
	frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"", "."}}))
}

func TestReservationsNoFetchStampedKeepEveryWorktree(t *testing.T) {
	wt, _ := worktree(t)
	p := newProbe(context.Background(), ProbeOptions{Processes: cwds(), Now: stale()})
	why := p.ReservationsUnread()
	if why == "" {
		t.Fatal("ReservationsUnread() = \"\"; want why, with no fetch stamped")
	}
	l := p.Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindReservationCheck, Hint)
	if l.Reason != "reservations unread: "+why {
		t.Fatalf("reason = %q; want %q", l.Reason, "reservations unread: "+why)
	}
	if got := probe(ProbeOptions{Processes: cwds(), Now: stale()}).ReservationsUnread(); got != "" {
		t.Fatalf("ReservationsUnread() after a fetch = %q; want \"\"", got)
	}
}

func TestAnUnreadableReservationWithinItsHoldKeepsEveryWorktree(t *testing.T) {
	wt, _ := worktree(t)
	now := stale()()
	r := claim.Reservation{Record: "iss-2610230000000003", State: claim.StateUnreadable, Unreadable: "its message cannot be parsed",
		EffectiveExpiry: now.Add(time.Hour)}
	p := probe(ProbeOptions{Reservations: claim.Snapshot{Reservations: []claim.Reservation{r}}, Processes: cwds(), Now: at(now)})
	l := p.Judge(Worktree{Path: wt, Changed: []string{"docs/x.md"}})
	holdsBy(t, l, KindReservation, Hint)
	if !strings.Contains(l.Reason, r.Record) || !strings.Contains(l.Reason, "holder") {
		t.Fatalf("reason = %q; want it to name the record and its holder", l.Reason)
	}
	r.EffectiveExpiry = now.Add(-time.Second)
	p = probe(ProbeOptions{Reservations: claim.Snapshot{Reservations: []claim.Reservation{r}}, Processes: cwds(), Now: at(now)})
	frees(t, p.Judge(Worktree{Path: wt, Changed: []string{"docs/x.md"}}))
}

// TestAFreshCopyOnTheDefaultBranchIsKeptOnlyWhileInUse is the coordinator's
// case: a copy sitting exactly on the default branch, changing nothing and
// reserving nothing, is kept while a process works in it and freed once
// nothing does and nothing in it is recent.
func TestAFreshCopyOnTheDefaultBranchIsKeptOnlyWhileInUse(t *testing.T) {
	wt, gitDir := worktree(t)
	old := time.Now().Add(-72 * time.Hour)
	for _, p := range []string{filepath.Join(wt, "internal", "a.go"), filepath.Join(wt, "internal"), filepath.Join(wt, ".git"), wt, filepath.Join(gitDir, "index")} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	l := probe(ProbeOptions{Processes: cwds(wt), Now: time.Now}).Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindProcess, Proven)
	if !strings.HasPrefix(l.Reason, "in use: ") {
		t.Fatalf("reason = %q; want it to read as in use", l.Reason)
	}
	frees(t, probe(ProbeOptions{Processes: cwds(), Now: time.Now}).Judge(Worktree{Path: wt, Changed: none}))
}

func TestAWorktreeAnotherAccountOwnsIsKept(t *testing.T) {
	wt, _ := worktree(t)
	foreign := func(string) (int, bool, error) { return 4242, true, nil }
	l := probe(ProbeOptions{Owner: foreign, Processes: cwds(), Now: stale()}).Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindOwner, Proven)
	if l.Reason != "owned by another account (uid 4242)" {
		t.Fatalf("reason = %q; want %q", l.Reason, "owned by another account (uid 4242)")
	}

	// Reported as this account's, it falls through to the other proofs.
	mine := func(string) (int, bool, error) { return 0, false, nil }
	frees(t, probe(ProbeOptions{Owner: mine, Processes: cwds(), Now: stale()}).Judge(Worktree{Path: wt, Changed: none}))
	holdsBy(t, probe(ProbeOptions{Owner: mine, Processes: cwds(wt), Now: stale()}).Judge(Worktree{Path: wt, Changed: none}), KindProcess, Proven)

	// An owner that cannot be read keeps it, and the worktree's own view of
	// the reservations, which runs git in it, is not read.
	unread := func(string) (int, bool, error) { return 0, false, errors.New("stat failed") }
	viewed := 0
	view := func(string, time.Time) (claim.Snapshot, error) { viewed++; return claim.Snapshot{}, nil }
	l = probe(ProbeOptions{Owner: unread, CopyView: view, Processes: cwds(), Now: stale()}).Judge(Worktree{Path: wt, Changed: none})
	holdsBy(t, l, KindOwner, Hint)
	if !strings.Contains(l.Reason, "stat failed") {
		t.Fatalf("reason = %q; want the failure named", l.Reason)
	}
	if viewed != 0 {
		t.Fatalf("the worktree's own reservation view ran %d time(s) with its owner unread", viewed)
	}
	probe(ProbeOptions{Owner: mine, CopyView: view, Processes: cwds(), Now: stale()}).Judge(Worktree{Path: wt, Changed: none})
	if viewed != 1 {
		t.Fatalf("the worktree's own reservation view ran %d time(s) with its owner read; want 1", viewed)
	}
}

func TestTheDefaultOwnerCheckReadsThisAccountsTreeAsItsOwn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no uid on windows")
	}
	wt, _ := worktree(t)
	if uid, foreign, err := foreignOwner(wt); err != nil || foreign {
		t.Fatalf("foreignOwner(own temp tree) = %d, %t, %v; want this account's", uid, foreign, err)
	}
	if _, _, err := foreignOwner(filepath.Join(wt, "absent")); err == nil {
		t.Fatal("foreignOwner(absent) = nil error; want the stat failure")
	}
}
