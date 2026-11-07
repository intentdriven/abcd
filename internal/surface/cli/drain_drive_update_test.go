package cli

// drain_drive_update_test.go holds the rows of spc-2611062151392235 the surface
// answers: no network, no real login, no real installed binary. The installed file
// is a temporary one the seam names, and the harness a stand-in on PATH.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const (
	updFrom = "v0.17.0"
	updTo   = "v0.18.0"
)

// updateWorld is a drive whose installed binary is a temporary file holding the old
// bytes, with every step of the update a counted stand-in over it.
type updateWorld struct {
	startWaits []time.Duration
	*driveWorld
	target string
	// counts of each step, in the order they run.
	asked, staged, rehearsed, proved int
	reexecs                          []updateReexec
	lines                            []string
	log                              strings.Builder
	now                              time.Time
	stop                             chan struct{}
}

type updateReexec struct {
	path string
	args []string
}

func sha(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func newUpdateWorld(t *testing.T) *updateWorld {
	t.Helper()
	w := &updateWorld{driveWorld: newDriveWorld(t), now: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC), stop: make(chan struct{})}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w.target = filepath.Join(dir, "abcd")
	if err := os.WriteFile(w.target, []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	steps, reexec, args, wait := driveSelfUpdateSteps, driveReexec, driveArgs, driveStartWait
	t.Cleanup(func() { driveSelfUpdateSteps, driveReexec, driveArgs, driveStartWait = steps, reexec, args, wait })
	driveStartWait = func(_ context.Context, d time.Duration) error {
		w.startWaits = append(w.startWaits, d)
		return nil
	}
	w.useRelease(updFrom)
	driveSelfUpdateSteps.candidate = func(string) (string, bool, error) {
		w.asked++
		return updTo, true, nil
	}
	driveSelfUpdateSteps.stage = func(target, tag string) (update.Staged, error) {
		w.staged++
		path := filepath.Join(filepath.Dir(target), ".abcd.new")
		if err := os.WriteFile(path, []byte("new-bytes"), 0o755); err != nil {
			return update.Staged{}, err
		}
		return update.Staged{Path: path, SHA256: sha("new-bytes")}, nil
	}
	driveSelfUpdateSteps.rehearse = func(context.Context, driveCandidate, []string) error {
		w.rehearsed++
		return nil
	}
	driveSelfUpdateSteps.prove = func(context.Context, runner.Profile, string) error {
		w.proved++
		return nil
	}
	driveSelfUpdateSteps.commit = update.CommitStaged
	driveSelfUpdateSteps.harnessOwed = func(runner.Profile) bool { return false }
	driveSelfUpdateSteps.harnessMoved = func(runner.Profile) bool { return false }
	driveSelfUpdateSteps.verifyKept, driveSelfUpdateSteps.pinned = keptStubs()
	driveReexec = func(path string, args, _ []string) error {
		w.reexecs = append(w.reexecs, updateReexec{path, slices.Clone(args)})
		return nil
	}
	driveArgs = func() []string { return []string{"abcd", "drain", "--drive", "--max", "3"} }
	return w
}

// useRelease has the running file claim tag.
func (w *updateWorld) useRelease(tag string) {
	driveSelfUpdateSteps.target = func() update.SelfTarget {
		return update.SelfTarget{Path: w.target, Version: tag}
	}
}

// step is the update step as a drive holds it, on walls that pass.
func (w *updateWorld) step(t *testing.T) *driveUpdate {
	t.Helper()
	w.passWalls(t)
	return w.stepOn(runner.Profile{Dir: w.profile}, driveFlags{self: driveSelfUpdateSteps.target()})
}

func (w *updateWorld) stepOn(p runner.Profile, f driveFlags) *driveUpdate {
	u := newDriveUpdate(w.repo, p, f, w.stop, func(line string) { w.lines = append(w.lines, line) }, &w.log)
	u.now = func() time.Time { return w.now }
	return u
}

func (w *updateWorld) body(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// nothingSwapped fails unless the installed file holds the old bytes and the staging
// and kept names are empty.
func (w *updateWorld) nothingSwapped(t *testing.T) {
	t.Helper()
	if got := w.body(t, w.target); got != "old-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	for _, name := range []string{".abcd.new", ".abcd.previous"} {
		if _, err := os.Lstat(filepath.Join(filepath.Dir(w.target), name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s stands beside the installed file: %v", name, err)
		}
	}
	if len(w.reexecs) != 0 {
		t.Fatalf("re-executed: %v", w.reexecs)
	}
}

// oneLine fails unless exactly one line was said, holding each want.
func (w *updateWorld) oneLine(t *testing.T, want ...string) {
	t.Helper()
	if len(w.lines) != 1 {
		t.Fatalf("%d lines, want one: %q", len(w.lines), w.lines)
	}
	for _, s := range want {
		if !strings.Contains(w.lines[0], s) {
			t.Fatalf("the line lacks %q: %s", s, w.lines[0])
		}
	}
}

// pathEntry records the temporary file as this home's install, at digest.
func (w *updateWorld) pathEntry(t *testing.T, digest string) string {
	t.Helper()
	path := abcdhome.Path(w.home, "path-entry")
	if err := os.WriteFile(path, []byte("path="+w.target+"\nbinary_sha256="+digest+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// liveDrain writes a drain in progress with cap.
func (w *updateWorld) liveDrain(t *testing.T, lanes int) {
	t.Helper()
	state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(loop.DrainState{SchemaVersion: loop.DrainSchemaVersion, Max: lanes})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestTheCheckAsksOnceAnHourAtMost is the clock half of the "boundary only" row (the
// loop's half is terminal_boundary_test.go).
func TestTheCheckAsksOnceAnHourAtMost(t *testing.T) {
	w := newUpdateWorld(t)
	driveSelfUpdateSteps.candidate = func(string) (string, bool, error) {
		w.asked++
		return updFrom, false, nil
	}
	u := w.step(t)
	u.atBoundary(context.Background())
	if w.asked != 1 {
		t.Fatalf("asked %d times, want once", w.asked)
	}
	w.now = w.now.Add(59 * time.Minute)
	if u.atBoundary(context.Background()) {
		t.Fatal("a boundary with no request took no time")
	}
	if w.asked != 1 {
		t.Fatalf("a second call within the hour made a request: %d", w.asked)
	}
	w.now = w.now.Add(time.Minute)
	u.atBoundary(context.Background())
	if w.asked != 2 {
		t.Fatalf("an hour on it asks again: %d", w.asked)
	}
}

// TestNoSelfUpdateMakesNoRequest is the "flag" row.
func TestNoSelfUpdateMakesNoRequest(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--no-self-update"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "releases: none (--no-self-update)") {
		t.Fatalf("the consent line says none:\n%s", out.String())
	}
	if w.terminalCalls != 1 {
		t.Fatalf("the loop runs: %d", w.terminalCalls)
	}
	if w.terminal.AtBoundary != nil {
		t.Fatal("a run that takes no release has no check at a boundary")
	}
	// The step itself still asks nothing on such a run, whoever calls it.
	w.stepOn(runner.Profile{Dir: w.profile}, driveFlags{noSelfUpdate: true, self: driveSelfUpdateSteps.target()}).atBoundary(context.Background())
	if w.asked != 0 {
		t.Fatalf("a request was made: %d", w.asked)
	}
	var plain strings.Builder
	if code := Run([]string{"drain", "--no-self-update", "--dry-run"}, &plain, &plain); code != 2 || !strings.Contains(plain.String(), "--no-self-update") {
		t.Fatalf("without --drive it is refused, naming the flag: exit %d: %s", code, plain.String())
	}
}

// TestAnAdmittedBuildsConsentNamesTheCheck: the yes is the ask adr-38's second tier
// requires only if the question names the request.
func TestAnAdmittedBuildsConsentNamesTheCheck(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	for _, want := range []string{"releases:", "release origin", "about once an hour", "--no-self-update"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("the consent lacks %q:\n%s", want, out.String())
		}
	}
}

// TestABuildThatNeverSelfUpdatesSaysWhy is the "source build" and "plugin root, Cellar,
// shim" rows at the surface: the consent line says why, and no request is made.
func TestABuildThatNeverSelfUpdatesSaysWhy(t *testing.T) {
	for _, c := range []struct {
		shape, detail string
	}{
		{update.ShapeSourceBuild, "this abcd is a source build, which never self-updates"},
		{"plugin-root", "the binary belongs to the plugin install"},
		{"package-manager", "the binary resolves into a Cellar, which Homebrew owns"},
		{"dev-shim", "the entry is the track-latest dev shim"},
	} {
		t.Run(c.shape, func(t *testing.T) {
			w := newUpdateWorld(t)
			w.passWalls(t)
			driveSelfUpdateSteps.target = func() update.SelfTarget {
				return update.SelfTarget{Path: w.target, Refusal: &update.Refusal{Shape: c.shape, Detail: c.detail}}
			}
			var out, errb strings.Builder
			if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
				t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
			}
			if !strings.Contains(out.String(), "releases: none: "+c.detail) {
				t.Fatalf("the consent line says why:\n%s", out.String())
			}
			if w.terminal.AtBoundary != nil {
				t.Fatal("a run that takes no release has no check at a boundary")
			}
			// The step itself still asks nothing on such a run, whoever calls it.
			w.stepOn(runner.Profile{Dir: w.profile}, driveFlags{self: driveSelfUpdateSteps.target()}).atBoundary(context.Background())
			if w.asked != 0 {
				t.Fatalf("a request was made: %d", w.asked)
			}
		})
	}
}

// TestALatestThatIsNotNewerFetchesNothing is the "not newer" row: nothing fetched, and
// nothing said beyond the log.
func TestALatestThatIsNotNewerFetchesNothing(t *testing.T) {
	for _, tag := range []string{updFrom, "v0.16.4", "v0.18.0-rc.1"} {
		t.Run(tag, func(t *testing.T) {
			w := newUpdateWorld(t)
			driveSelfUpdateSteps.candidate = func(string) (string, bool, error) {
				w.asked++
				return tag, false, nil
			}
			w.step(t).atBoundary(context.Background())
			if w.asked != 1 || w.staged != 0 {
				t.Fatalf("asked %d, staged %d", w.asked, w.staged)
			}
			if len(w.lines) != 0 {
				t.Fatalf("said %q", w.lines)
			}
			if !strings.Contains(w.log.String(), tag) {
				t.Fatalf("the log names what it read: %q", w.log.String())
			}
			w.nothingSwapped(t)
		})
	}
}

// TestAFetchThatFailsIsSaidOnceAndNotFetchedAgain is the "fetch fails" row.
func TestAFetchThatFailsIsSaidOnceAndNotFetchedAgain(t *testing.T) {
	for _, why := range []string{"release v0.18.0 has no checksums.txt", "downloading the asset: 404", "its build provenance could not be verified", "its build metadata names release v0.17.9"} {
		t.Run(why, func(t *testing.T) {
			w := newUpdateWorld(t)
			driveSelfUpdateSteps.stage = func(string, string) (update.Staged, error) {
				w.staged++
				return update.Staged{}, errors.New(why)
			}
			u := w.step(t)
			if !u.atBoundary(context.Background()) {
				t.Fatal("the check took time")
			}
			w.oneLine(t, updTo, why, "going on with "+updFrom)
			w.nothingSwapped(t)
			w.now = w.now.Add(2 * time.Hour)
			u.atBoundary(context.Background())
			if w.asked != 2 || w.staged != 1 {
				t.Fatalf("asked %d, staged %d: the tag is not fetched again", w.asked, w.staged)
			}
			if len(w.lines) != 1 {
				t.Fatalf("said again: %q", w.lines)
			}
		})
	}
}

// TestAFetchFaultIsTriedAgainAtTheNextCheck is the "fetch fault" row: a fault that says
// nothing of the release writes one line per attempt, costs no more than the one attempt
// per interval, and leaves the release to be taken once the network answers.
func TestAFetchFaultIsTriedAgainAtTheNextCheck(t *testing.T) {
	w := newUpdateWorld(t)
	working := driveSelfUpdateSteps.stage
	fault := errors.New("GET checksums.txt: 503 Service Unavailable")
	driveSelfUpdateSteps.stage = func(string, string) (update.Staged, error) {
		w.staged++
		return update.Staged{}, fmt.Errorf("fetching checksums: %w; a long remedy", &update.NoVerdictError{Err: fault})
	}
	u := w.step(t)
	u.atBoundary(context.Background())
	w.oneLine(t, updTo, "503", "going on with "+updFrom, "next check")
	if strings.Contains(w.lines[0], "long remedy") {
		t.Fatalf("the line is the fault alone: %s", w.lines[0])
	}
	w.nothingSwapped(t)
	if u.atBoundary(context.Background()) || w.asked != 1 || w.staged != 1 {
		t.Fatalf("within the hour nothing more is tried: asked %d, staged %d", w.asked, w.staged)
	}
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if w.asked != 2 || w.staged != 2 || len(w.lines) != 2 {
		t.Fatalf("the next check tries again and says so once: asked %d, staged %d, %q", w.asked, w.staged, w.lines)
	}
	driveSelfUpdateSteps.stage = working
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if w.proved != 1 || len(w.reexecs) != 1 {
		t.Fatalf("once the network answers the release is taken: proved %d, re-executed %d", w.proved, len(w.reexecs))
	}
}

// TestADeadNetworkCostsOneCheckAnHourAndOneLine: with the origin unreachable no fetch is
// attempted, and each check says so once.
func TestADeadNetworkCostsOneCheckAnHourAndOneLine(t *testing.T) {
	w := newUpdateWorld(t)
	driveSelfUpdateSteps.candidate = func(string) (string, bool, error) {
		w.asked++
		return "", false, errors.New("no route to the origin")
	}
	u := w.step(t)
	u.atBoundary(context.Background())
	w.oneLine(t, "no route to the origin")
	u.atBoundary(context.Background())
	if w.asked != 1 || w.staged != 0 || len(w.lines) != 1 {
		t.Fatalf("within the hour nothing more: asked %d, staged %d, %q", w.asked, w.staged, w.lines)
	}
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if w.asked != 2 || w.staged != 0 || len(w.lines) != 2 {
		t.Fatalf("one check and one line each interval: asked %d, staged %d, %q", w.asked, w.staged, w.lines)
	}
}

// TestACandidateThatFailsItsRehearsalIsTriedAgainLater is the "candidate rehearsal" row's
// step (F5): anything but the normal result leaves the drive on the old binary and spends no
// launch, later checks try it again, and each reason is said once.
func TestACandidateThatFailsItsRehearsalIsTriedAgainLater(t *testing.T) {
	w := newUpdateWorld(t)
	why := errors.New("`drain --dry-run` exited 2")
	driveSelfUpdateSteps.rehearse = func(_ context.Context, c driveCandidate, args []string) error {
		w.rehearsed++
		if c.repoRoot != w.repo || filepath.Base(c.path) != ".abcd.new" || !slices.Contains(args, "--after-update") {
			t.Errorf("rehearsed in %s, %s, %v", c.repoRoot, c.path, args)
		}
		return why
	}
	u := w.step(t)
	u.atBoundary(context.Background())
	if w.rehearsed != 1 || w.proved != 0 {
		t.Fatalf("rehearsed %d, proved %d: a rehearsal comes first and spends no launch", w.rehearsed, w.proved)
	}
	w.oneLine(t, updTo, "drain --dry-run", "going on with "+updFrom)
	w.nothingSwapped(t)
	if u.tried[updTo] {
		t.Fatal("a failed rehearsal does not latch the tag")
	}
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if w.rehearsed != 2 || len(w.lines) != 1 {
		t.Fatalf("tried again and the same reason not said again: rehearsed %d, %q", w.rehearsed, w.lines)
	}
	why = errors.New("`drain --dry-run` did not answer")
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if len(w.lines) != 2 || !strings.Contains(w.lines[1], "did not answer") {
		t.Fatalf("a new reason is said: %q", w.lines)
	}
	why = nil
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if w.proved != 1 || len(w.reexecs) != 1 {
		t.Fatalf("a rehearsal that passes goes on: proved %d, re-executed %d", w.proved, len(w.reexecs))
	}
}

// rehearsalScript is a candidate that answers as body says.
func rehearsalScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".abcd.new")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestTheRehearsalRunsTheCandidateReadOnlyWithABound is the "candidate rehearsal" row:
// the real rehearsal over stand-in candidates.
func TestTheRehearsalRunsTheCandidateReadOnlyWithABound(t *testing.T) {
	const normal = `case "$*" in
*--help*) exit 0 ;;
"drain --dry-run --json") printf '{"dry_run": true, "dispositions": []}\n'; exit 0 ;;
esac
exit 9`
	args := []string{"abcd", "drain", "--drive", "--after-update", updFrom + "=" + updTo, "--yes"}
	prev := driveRehearsalTimeout
	t.Cleanup(func() { driveRehearsalTimeout = prev })
	driveRehearsalTimeout = 2 * time.Minute
	dir := t.TempDir()
	if err := driveRehearse(context.Background(), candidateFor(t, dir, rehearsalScript(t, normal)), args); err != nil {
		t.Fatalf("the normal result is accepted: %v", err)
	}
	for _, c := range []struct{ name, body, want string }{
		{"a flag it does not know", `case "$*" in *--help*) echo 'unknown flag: --after-update' >&2; exit 1 ;; esac; printf '{"dry_run":true}\n'`, "unknown flag"},
		{"a refusal", `case "$*" in *--help*) exit 0 ;; esac; echo 'schema too new' >&2; exit 2`, "schema too new"},
		{"not the plan", `case "$*" in *--help*) exit 0 ;; esac; echo 'hello'`, "dry-run"},
		{"a plan that is no dry run", `case "$*" in *--help*) exit 0 ;; esac; printf '{"dry_run":false}\n'`, "dry-run"},
		{"a crash", `kill -SEGV $$`, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := driveRehearse(context.Background(), candidateFor(t, dir, rehearsalScript(t, c.body)), args)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want it to name %q", err, c.want)
			}
		})
	}
	t.Run("a hang", func(t *testing.T) {
		driveRehearsalTimeout = 200 * time.Millisecond
		began := time.Now()
		err := driveRehearse(context.Background(), candidateFor(t, dir, rehearsalScript(t, `sleep 60 & wait`)), args)
		if err == nil {
			t.Fatal("a candidate that hangs is rejected")
		}
		if took := time.Since(began); took > 30*time.Second {
			t.Fatalf("it took %v, past its bound", took)
		}
	})
	t.Run("its working directory", func(t *testing.T) {
		driveRehearsalTimeout = 2 * time.Minute
		mark := filepath.Join(t.TempDir(), "pwd")
		body := `pwd -P > '` + mark + `'; case "$*" in *--help*) exit 0 ;; esac; printf '{"dry_run":true}\n'`
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := driveRehearse(context.Background(), candidateFor(t, dir, rehearsalScript(t, body)), args); err != nil {
			t.Fatal(err)
		}
		if raw, _ := os.ReadFile(mark); strings.TrimSpace(string(raw)) != real {
			t.Fatalf("it ran in %q, not the checkout %q", raw, real)
		}
	})
}

// TestAWallTheCandidateDoesNotHoldChangesNothing is the "proof fails" row, on the real
// proof and a stand-in harness: settings and binary byte-identical, the wall named, the
// staging file gone, and the loop goes on.
func TestAWallTheCandidateDoesNotHoldChangesNothing(t *testing.T) {
	w := newUpdateWorld(t)
	w.recordTerminal(t)
	standInHarness(t, true, wallInit+wallRefused+wallResult)
	driveSelfUpdateSteps.prove = runner.ProveCandidate
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	before := w.body(t, filepath.Join(w.profile, "settings.json"))
	if !strings.Contains(before, runner.SelfBinary()) {
		t.Fatalf("the settings name the drive's binary:\n%s", before)
	}
	// The harness the run proved now reports a guard that refuses nothing.
	harness, err := exec.LookPath("claude")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(harness), "stream"), []byte(wallInit+wallResult), 0o600); err != nil {
		t.Fatal(err)
	}
	if !w.terminal.AtBoundary(context.Background()) {
		t.Fatal("the check took time")
	}
	w.nothingSwapped(t)
	if got := w.body(t, filepath.Join(w.profile, "settings.json")); got != before {
		t.Fatalf("the settings changed:\n%s", got)
	}
	if strings.Count(out.String(), "was not taken") != 1 || !strings.Contains(out.String(), runner.WallGuard) {
		t.Fatalf("one line names the wall:\n%s", out.String())
	}
	if w.terminal.BeforeStage == nil || w.terminal.BeforeStage() != nil {
		t.Fatal("what the check before each stage compares still holds")
	}
}

// TestAProofThatCameToNoVerdictIsNotAVerdictOnTheRelease: a limit or an outage says
// nothing about the release, so the next check tries it again.
func TestAProofThatCameToNoVerdictIsNotAVerdictOnTheRelease(t *testing.T) {
	for name, proofErr := range map[string]error{
		"limit":      &runner.LimitError{},
		"no verdict": &runner.NoVerdictError{Detail: "it did not complete"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newUpdateWorld(t)
			driveSelfUpdateSteps.prove = func(context.Context, runner.Profile, string) error {
				w.proved++
				return proofErr
			}
			u := w.step(t)
			u.atBoundary(context.Background())
			if len(w.lines) != 1 || strings.Contains(w.lines[0], "was not taken") {
				t.Fatalf("said %q", w.lines)
			}
			w.nothingSwapped(t)
			w.now = w.now.Add(time.Hour)
			u.atBoundary(context.Background())
			if w.proved != 2 {
				t.Fatalf("proved %d times: the next check tries it again", w.proved)
			}
		})
	}
}

// TestAStagingFileAlteredAfterTheProofIsNotSwappedIn is the "staging file altered" row.
func TestAStagingFileAlteredAfterTheProofIsNotSwappedIn(t *testing.T) {
	w := newUpdateWorld(t)
	driveSelfUpdateSteps.prove = func(_ context.Context, _ runner.Profile, staged string) error {
		return os.WriteFile(staged, []byte("a stage's bytes"), 0o755)
	}
	w.step(t).atBoundary(context.Background())
	w.nothingSwapped(t)
	w.oneLine(t, updTo, "changed after it was verified", "going on with "+updFrom)
}

// TestASwapKeepsThePreviousAndReexecutesWithTheSameArguments is the "swap and re-exec"
// row.
func TestASwapKeepsThePreviousAndReexecutesWithTheSameArguments(t *testing.T) {
	w := newUpdateWorld(t)
	entry := w.pathEntry(t, sha("old-bytes"))
	w.step(t).atBoundary(context.Background())
	if got := w.body(t, w.target); got != "new-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	if got := w.body(t, filepath.Join(filepath.Dir(w.target), ".abcd.previous")); got != "old-bytes" {
		t.Fatalf("the kept copy holds %q", got)
	}
	if got := w.body(t, entry); !strings.Contains(got, sha("new-bytes")) {
		t.Fatalf("the install record's digest:\n%s", got)
	}
	if tag, ok := ahoy.ReadReleaseMark(); !ok || tag != updTo {
		t.Fatalf("the release mark reads %q, %v", tag, ok)
	}
	if len(w.reexecs) != 1 {
		t.Fatalf("re-executed %d times", len(w.reexecs))
	}
	got := w.reexecs[0]
	if got.path != w.target {
		t.Fatalf("it re-executes the installed name it has just checked: %s", got.path)
	}
	want := []string{"abcd", "drain", "--drive", "--after-update", updFrom + "=" + updTo, "--yes", "--profile", "walled"}
	if !slices.Equal(got.args, want) {
		t.Fatalf("args %q, want %q", got.args, want)
	}
}

// TestAResumedDriveRewritesItsOwnFlagAndKeepsThePersons: a second update does not stack
// the flag, and a --yes or a --profile the person typed is not added twice.
func TestAResumedDriveRewritesItsOwnFlagAndKeepsThePersons(t *testing.T) {
	w := newUpdateWorld(t)
	driveArgs = func() []string {
		return []string{"abcd", "drain", "--drive", "--yes", "--profile=walled", "--after-update", "v0.16.0=" + updFrom, "--update-refused", "v0.17.5"}
	}
	w.step(t).atBoundary(context.Background())
	want := []string{"abcd", "drain", "--drive", "--yes", "--profile=walled", "--update-refused", "v0.17.5", "--after-update", updFrom + "=" + updTo}
	if len(w.reexecs) != 1 || !slices.Equal(w.reexecs[0].args, want) {
		t.Fatalf("got %v, want %q", w.reexecs, want)
	}
}

// TestARefusedReexecutionPutsThePreviousBack is the "exec refused" row.
func TestARefusedReexecutionPutsThePreviousBack(t *testing.T) {
	w := newUpdateWorld(t)
	entry := w.pathEntry(t, sha("old-bytes"))
	driveReexec = func(string, []string, []string) error { return errors.New("exec format error") }
	u := w.step(t)
	if !u.atBoundary(context.Background()) {
		t.Fatal("the step took time")
	}
	if got := w.body(t, w.target); got != "old-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	if got := w.body(t, entry); !strings.Contains(got, sha("old-bytes")) {
		t.Fatalf("the install record is back on the previous bytes:\n%s", got)
	}
	w.oneLine(t, updTo, "exec format error", updFrom)
	w.now = w.now.Add(2 * time.Hour)
	u.atBoundary(context.Background())
	if w.staged != 1 {
		t.Fatalf("that version was tried again: staged %d", w.staged)
	}
}

// resumeArgs are the arguments a re-execution would pass.
func resumeArgs(extra ...string) []string {
	return append([]string{"drain", "--drive", "--after-update", updFrom + "=" + updTo, "--yes"}, extra...)
}

// TestAResumedDriveGoesOnFromTheDrainsState is the "resume" row: real static walls, a
// harness whose live launch would fail, so a proof run at the resume would show.
func TestAResumedDriveGoesOnFromTheDrainsState(t *testing.T) {
	w := newUpdateWorld(t)
	w.useRelease(updTo)
	w.recordTerminal(t)
	standInHarness(t, true, wallInit+wallResult)
	w.liveDrain(t, 3)
	until := time.Now().Add(240 * time.Hour).UTC().Truncate(time.Second)
	var out, errb strings.Builder
	code := Run(resumeArgs("--until", until.Format(time.RFC3339)), &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	text := out.String() + errb.String()
	if strings.Contains(text, "works through this repository") || strings.Contains(text, "Start?") {
		t.Fatalf("the consent screen showed:\n%s", text)
	}
	if got := strings.Count(text, "abcd drain --drive: abcd updated from "+updFrom+" to "+updTo); got != 1 {
		t.Fatalf("%d lines name the move:\n%s", got, text)
	}
	if w.terminalCalls != 1 {
		t.Fatalf("the loop ran %d times:\n%s", w.terminalCalls, text)
	}
	progress, err := loop.ReadDrainProgress(w.repo)
	if err != nil || progress.Max != 3 || !progress.Live {
		t.Fatalf("the cap: %+v, %v", progress, err)
	}
	if w.terminal.Drain.Max != 0 || !w.terminal.Until.Equal(until) || len(w.terminal.Drain.Skip) != 0 {
		t.Fatalf("the cap and the skips stand in the state, and the bound in the arguments: %+v until %v", w.terminal.Drain, w.terminal.Until)
	}
	if got := w.body(t, filepath.Join(w.profile, "settings.json")); !strings.Contains(got, runner.SelfBinary()) {
		t.Fatalf("the settings name the same path:\n%s", got)
	}
	if err := w.terminal.BeforeStage(); err != nil {
		t.Fatalf("the baseline is what is there now: %v", err)
	}
}

// TestABoundThatPassedDuringTheUpdateEndsTheRunInTheLoop: the bound was in the future
// when the check began; a refusal at the resume would read as a release that cannot
// start and take the previous binary back.
func TestABoundThatPassedDuringTheUpdateEndsTheRunInTheLoop(t *testing.T) {
	w := newUpdateWorld(t)
	w.useRelease(updTo)
	w.passWalls(t)
	w.liveDrain(t, 3)
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	if code := Run(resumeArgs("--until", past), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminalCalls != 1 || len(w.reexecs) != 0 {
		t.Fatalf("loop %d, re-executions %v:\n%s%s", w.terminalCalls, w.reexecs, out.String(), errb.String())
	}
}

// TestTheResumeFlagsTypedByHandAreRefused is the "--after-update by hand" row.
func TestTheResumeFlagsTypedByHandAreRefused(t *testing.T) {
	keep := func(w *updateWorld) {
		if err := os.WriteFile(filepath.Join(filepath.Dir(w.target), ".abcd.previous"), []byte("kept"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	unmoved := func(t *testing.T, w *updateWorld) {
		t.Helper()
		if got := w.body(t, w.target); got != "old-bytes" {
			t.Fatalf("the installed file holds %q", got)
		}
		if w.terminalCalls != 0 || len(w.reexecs) != 0 {
			t.Fatalf("loop %d, re-executions %v", w.terminalCalls, w.reexecs)
		}
	}
	t.Run("no live drain", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.useRelease(updTo)
		w.passWalls(t)
		keep(w)
		if text := refusedDrive(t, resumeArgs()...); !strings.Contains(text, "--after-update") {
			t.Fatalf("the refusal names the flag: %s", text)
		}
		unmoved(t, w)
	})
	t.Run("a release this build does not claim", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		w.liveDrain(t, 3)
		keep(w)
		if text := refusedDrive(t, resumeArgs()...); !strings.Contains(text, "--after-update") {
			t.Fatalf("the refusal names the flag: %s", text)
		}
		unmoved(t, w)
	})
	t.Run("not the shape", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		w.liveDrain(t, 3)
		if text := refusedDrive(t, "drain", "--drive", "--after-update", "v0.18.0"); !strings.Contains(text, "--after-update") {
			t.Fatalf("the refusal names the flag: %s", text)
		}
		unmoved(t, w)
	})
	t.Run("update-refused with no live drain", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		if text := refusedDrive(t, "drain", "--drive", "--update-refused", updTo); !strings.Contains(text, "--update-refused") {
			t.Fatalf("the refusal names the flag: %s", text)
		}
		unmoved(t, w)
	})
	t.Run("without a drive", func(t *testing.T) {
		w := newUpdateWorld(t)
		for _, flag := range []string{"--after-update", "--update-refused"} {
			if text := refusedDrive(t, "drain", "--dry-run", flag, updTo); !strings.Contains(text, flag) {
				t.Fatalf("the refusal names the flag: %s", text)
			}
		}
		unmoved(t, w)
	})
}

// TestAStageCannotDecideTheUpdate is the surface's half of the "stage cannot decide"
// row: a release mark and an install record a stage wrote, naming anything, change
// nothing about what the check asks or takes. (A stage writing the binary or the
// settings ends the run: wall_candidate_test.go and terminal_boundary_test.go.)
func TestAStageCannotDecideTheUpdate(t *testing.T) {
	w := newUpdateWorld(t)
	if err := os.WriteFile(abcdhome.Path(w.home, "release-mark"), []byte("release_tag=v9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.pathEntry(t, sha("something a stage wrote"))
	w.step(t).atBoundary(context.Background())
	if w.asked != 1 || w.proved != 1 || len(w.reexecs) != 1 {
		t.Fatalf("asked %d, proved %d, re-executed %v", w.asked, w.proved, w.reexecs)
	}
	args := w.reexecs[0].args
	if i := slices.Index(args, driveAfterUpdateFlag); i < 0 || i+1 >= len(args) || args[i+1] != updFrom+"="+updTo {
		t.Fatalf("the candidate is the origin's: %q", args)
	}
}

// TestAStopDuringTheStepSwapsNothing is the "stop during the step" row.
func TestAStopDuringTheStepSwapsNothing(t *testing.T) {
	for _, at := range []string{"stage", "rehearse", "prove"} {
		t.Run(at, func(t *testing.T) {
			w := newUpdateWorld(t)
			stage, rehearse, prove := driveSelfUpdateSteps.stage, driveSelfUpdateSteps.rehearse, driveSelfUpdateSteps.prove
			switch at {
			case "stage":
				driveSelfUpdateSteps.stage = func(target, tag string) (update.Staged, error) {
					close(w.stop)
					return stage(target, tag)
				}
			case "rehearse":
				driveSelfUpdateSteps.rehearse = func(ctx context.Context, c driveCandidate, args []string) error {
					close(w.stop)
					return rehearse(ctx, c, args)
				}
			case "prove":
				driveSelfUpdateSteps.prove = func(ctx context.Context, p runner.Profile, staged string) error {
					close(w.stop)
					return prove(ctx, p, staged)
				}
			}
			if !w.step(t).atBoundary(context.Background()) {
				t.Fatal("the step took time, so the loop's own stop check runs next")
			}
			w.nothingSwapped(t)
			if len(w.lines) != 0 {
				t.Fatalf("a stop is no failed release: %q", w.lines)
			}
		})
	}
	t.Run("before anything", func(t *testing.T) {
		w := newUpdateWorld(t)
		close(w.stop)
		w.step(t).atBoundary(context.Background())
		if w.asked != 0 {
			t.Fatalf("asked %d times under a stop", w.asked)
		}
	})
	t.Run("after the swap", func(t *testing.T) {
		w := newUpdateWorld(t)
		commit := driveSelfUpdateSteps.commit
		driveSelfUpdateSteps.commit = func(target string, st update.Staged, keep string) error {
			err := commit(target, st, keep)
			close(w.stop)
			return err
		}
		w.step(t).atBoundary(context.Background())
		if len(w.reexecs) != 0 {
			t.Fatalf("a re-execution would lose the stop: %v", w.reexecs)
		}
		w.oneLine(t, updTo, "stop")
	})
}

// goBackWorld is a re-executed drive whose start fails on its static walls.
func goBackWorld(t *testing.T) *updateWorld {
	t.Helper()
	w := newUpdateWorld(t)
	w.useRelease(updTo)
	w.passWalls(t)
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		return nil, errors.New("this release cannot read the settings")
	}
	w.liveDrain(t, 3)
	driveArgs = func() []string {
		return append([]string{"abcd"}, resumeArgs()...)
	}
	if err := os.WriteFile(w.target, []byte("new-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(w.target), ".abcd.previous"), []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	return w
}

// TestAReleaseThatCannotStartGoesBackToTheKeptCopy is the "re-executed start fails" row:
// the kept copy is restored, the version recorded as refused in its arguments, and the
// kept copy carries on.
func TestAReleaseThatCannotStartGoesBackToTheKeptCopy(t *testing.T) {
	w := goBackWorld(t)
	entry := w.pathEntry(t, sha("new-bytes"))
	var out, errb strings.Builder
	if code := Run(resumeArgs("--max", "3"), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminalCalls != 0 {
		t.Fatalf("the loop ran on the release that could not start")
	}
	if got := w.body(t, w.target); got != "old-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(w.target), ".abcd.previous")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the kept name is empty once restored: %v", err)
	}
	if got := w.body(t, entry); !strings.Contains(got, sha("old-bytes")) {
		t.Fatalf("the install record is back on the previous bytes:\n%s", got)
	}
	want := []string{"abcd", "drain", "--drive", "--yes", "--update-refused", updTo}
	if len(w.reexecs) != 1 || !slices.Equal(w.reexecs[0].args, want) {
		t.Fatalf("got %v, want %q", w.reexecs, want)
	}
	if w.reexecs[0].path != w.target {
		t.Fatalf("it starts the installed name it has just checked: %s", w.reexecs[0].path)
	}
	text := out.String() + errb.String()
	for _, s := range []string{updTo + " could not start", "cannot read the settings", updFrom, "not tried again"} {
		if !strings.Contains(text, s) {
			t.Fatalf("the line lacks %q:\n%s", s, text)
		}
	}
}

// TestTheKeptCopyCarriesOnAndDoesNotTryThatVersion is that row's other half: the kept
// copy, started with the flag, asks nothing, proves itself live (F4), and does not fetch
// the release it was restored from.
func TestTheKeptCopyCarriesOnAndDoesNotTryThatVersion(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	w.liveDrain(t, 3)
	proved := false
	driveWalls.live = func(context.Context, runner.Profile) error {
		proved = true
		return nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--update-refused", updTo}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	text := out.String() + errb.String()
	if strings.Contains(text, "works through this repository") || !proved {
		t.Fatalf("proved %v, consent shown:\n%s", proved, text)
	}
	if w.terminalCalls != 1 || w.terminal.AtBoundary == nil {
		t.Fatalf("the loop ran %d times", w.terminalCalls)
	}
	if !strings.Contains(text, updTo) || !strings.Contains(text, "going on with "+updFrom) {
		t.Fatalf("the line says what became of it:\n%s", text)
	}
	// The check waits its hour from the restart, and the tag is not fetched then.
	u := w.stepOn(runner.Profile{Dir: w.profile}, driveFlags{self: driveSelfUpdateSteps.target(), refused: []string{updTo}})
	u.last = time.Time{}
	u.atBoundary(context.Background())
	if w.asked != 1 || w.staged != 0 {
		t.Fatalf("asked %d, staged %d", w.asked, w.staged)
	}
}

// TestAStartThatCannotGoBackNamesTheKeptCopy: restoring or starting the kept copy fails,
// so the run ends, exit 2, naming it.
func TestAStartThatCannotGoBackNamesTheKeptCopy(t *testing.T) {
	t.Run("restore", func(t *testing.T) {
		w := goBackWorld(t)
		keep := filepath.Join(filepath.Dir(w.target), ".abcd.previous")
		if err := os.Remove(keep); err != nil {
			t.Fatal(err)
		}
		text := refusedDrive(t, resumeArgs("--max", "3")...)
		if !strings.Contains(text, ".abcd.previous") || !strings.Contains(text, "abcd update "+updFrom) || strings.Contains(text, "move that copy") {
			t.Fatalf("the exit names the kept copy and the verb that verifies: %s", text)
		}
		if len(w.reexecs) != 0 {
			t.Fatalf("started: %v", w.reexecs)
		}
	})
	t.Run("exec", func(t *testing.T) {
		w := goBackWorld(t)
		driveReexec = func(string, []string, []string) error { return errors.New("exec format error") }
		text := refusedDrive(t, resumeArgs("--max", "3")...)
		if !strings.Contains(text, "exec format error") || !strings.Contains(text, filepath.Base(w.target)) {
			t.Fatalf("the exit names the copy: %s", text)
		}
		if got := w.body(t, w.target); got != "old-bytes" {
			t.Fatalf("the installed file holds %q", got)
		}
	})
}

// TestAnotherDrivesHoldIsNoReasonToGoBack: a drive that lost the lock did not fail to
// start because of the release, and the holder may be running it.
func TestAnotherDrivesHoldIsNoReasonToGoBack(t *testing.T) {
	w := goBackWorld(t)
	held := make(chan struct{})
	release := make(chan struct{})
	lock := filepath.Join(w.repo, filepath.FromSlash(driveLockRel))
	if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(lock, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	defer func() {
		close(release)
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	var out, errb strings.Builder
	if code := Run(resumeArgs("--max", "3"), &out, &errb); code != 3 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if got := w.body(t, w.target); got != "new-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	if len(w.reexecs) != 0 {
		t.Fatalf("started: %v", w.reexecs)
	}
}

// TestARunThatEndsOnAFailureAfterAnUpdateNamesTheKeptCopy: past the start, restoring it
// is the person's act.
func TestARunThatEndsOnAFailureAfterAnUpdateNamesTheKeptCopy(t *testing.T) {
	w := newUpdateWorld(t)
	w.useRelease(updTo)
	w.passWalls(t)
	w.liveDrain(t, 3)
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		o.Moved()
		return loop.TerminalResult{Ended: loop.TerminalWall, Why: "a check failed"}, nil
	}
	if err := os.WriteFile(w.target, []byte("new-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(w.target), ".abcd.previous"), []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	text := refusedDrive(t, resumeArgs()...)
	if !strings.Contains(text, ".abcd.previous") {
		t.Fatalf("the exit names the kept copy and the way to restore it: %s", text)
	}
	if got := w.body(t, w.target); got != "new-bytes" {
		t.Fatalf("restoring it is the person's act: %q", got)
	}
	if len(w.reexecs) != 0 {
		t.Fatalf("started: %v", w.reexecs)
	}
}

// TestAReexecutionCarriesTheArgumentsAndReleasesTheDriveLock runs the real exec: the
// process that holds the checkout's drive lock replaces itself, and the process it
// becomes takes the lock inside the wait a drive allows.
func TestAReexecutionCarriesTheArgumentsAndReleasesTheDriveLock(t *testing.T) {
	const stageEnv, lockEnv = "ABCD_TEST_REEXEC_STAGE", "ABCD_TEST_REEXEC_LOCK"
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{self, "-test.run=^TestAReexecutionCarriesTheArgumentsAndReleasesTheDriveLock$", "-test.v"}
	// The stage's variable is replaced, not added: the first of two wins, and a process
	// that read "before" again would replace itself for ever.
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool { return strings.HasPrefix(kv, stageEnv+"=") })
	switch os.Getenv(stageEnv) {
	case "before":
		err := fsutil.WithFileLock(os.Getenv(lockEnv), time.Second, func() error {
			return driveReexec(self, args, append(env, stageEnv+"=after"))
		})
		t.Fatalf("the exec returned: %v", err)
	case "after":
		if err := fsutil.WithFileLock(os.Getenv(lockEnv), driveLockWait, func() error { return nil }); err != nil {
			t.Fatalf("the lock survived the exec: %v", err)
		}
		t.Log("took the lock after the exec")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, self, args[1:]...)
	cmd.Env = append(env, stageEnv+"=before", lockEnv+"="+filepath.Join(t.TempDir(), "drive.lock"))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "took the lock after the exec") {
		t.Fatalf("%v:\n%s", err, out)
	}
}
