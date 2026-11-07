package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/update"
)

// keptName is the kept copy's path in w.
func (w *updateWorld) keptName() string { return update.KeptPath(w.target) }

// aForeignKeptCopy puts bytes that are no release at the kept name.
func (w *updateWorld) aForeignKeptCopy(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(w.keptName(), []byte("not-a-release"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// restoredNothing holds the world to the installed file as it stands, with nothing run.
func (w *updateWorld) restoredNothing(t *testing.T, installed string) {
	t.Helper()
	if got := w.body(t, w.target); got != installed {
		t.Fatalf("the installed file holds %q", got)
	}
	if len(w.reexecs) != 0 {
		t.Fatalf("started: %v", w.reexecs)
	}
}

// TestAKeptCopyThatDoesNotVerifyIsNotRestored (F1): a start that fails goes back only
// to a kept copy verified as the release it replaced; anything else renames nothing and
// exits 2 naming the copy, offline included.
func TestAKeptCopyThatDoesNotVerifyIsNotRestored(t *testing.T) {
	t.Run("bytes that are no release", func(t *testing.T) {
		w := goBackWorld(t)
		w.aForeignKeptCopy(t)
		text := refusedDrive(t, resumeArgs("--max", "3")...)
		w.restoredNothing(t, "new-bytes")
		if !strings.Contains(text, ".abcd.previous") || !strings.Contains(text, "update "+updFrom) {
			t.Fatalf("the exit names the copy and the way on: %s", text)
		}
		if got := w.body(t, w.keptName()); got != "not-a-release" {
			t.Fatalf("the kept name holds %q", got)
		}
	})
	t.Run("offline", func(t *testing.T) {
		w := goBackWorld(t)
		driveSelfUpdateSteps.verifyKept = func(string, string) ([]byte, error) {
			return nil, &update.NoVerdictError{Err: errors.New("no route to the attestations API")}
		}
		text := refusedDrive(t, resumeArgs("--max", "3")...)
		w.restoredNothing(t, "new-bytes")
		if !strings.Contains(text, "no route") {
			t.Fatalf("the exit says why: %s", text)
		}
	})
}

// TestAnAgentsSessionCannotReachTheGoBack (F1): with a session's marker set the hidden
// flags refuse before anything is read or renamed.
func TestAnAgentsSessionCannotReachTheGoBack(t *testing.T) {
	w := newUpdateWorld(t)
	w.useRelease(updTo)
	w.passWalls(t)
	w.liveDrain(t, 3)
	w.aForeignKeptCopy(t)
	if err := os.WriteFile(w.target, []byte("new-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	verified := 0
	driveSelfUpdateSteps.verifyKept = func(string, string) ([]byte, error) {
		verified++
		return []byte("not-a-release"), nil
	}
	t.Setenv("CLAUDECODE", "1")
	text := refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if verified != 0 || !strings.Contains(text, "CLAUDECODE") {
		t.Fatalf("verified %d: %s", verified, text)
	}
}

// TestADrainStateThatCannotBeReadRestoresNothingUnverified (F1): a state that cannot be
// read is left to the start, and what guards the rename is the kept copy's verification.
func TestADrainStateThatCannotBeReadRestoresNothingUnverified(t *testing.T) {
	w := goBackWorld(t)
	w.aForeignKeptCopy(t)
	state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
	if err := os.WriteFile(state, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	text := refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if !strings.Contains(text, ".abcd.previous") {
		t.Fatalf("the exit names the copy: %s", text)
	}
}

// TestTheHiddenFlagsRefuseOnABuildThatNeverSelfUpdates (F1).
func TestTheHiddenFlagsRefuseOnABuildThatNeverSelfUpdates(t *testing.T) {
	for _, args := range [][]string{
		resumeArgs(),
		{"drain", "--drive", "--yes", "--update-refused", updTo},
	} {
		w := newUpdateWorld(t)
		driveSelfUpdateSteps.target = func() update.SelfTarget {
			return update.SelfTarget{Path: w.target, Version: updTo, Refusal: &update.Refusal{Shape: update.ShapeSourceBuild, Detail: "a source build"}}
		}
		w.passWalls(t)
		w.liveDrain(t, 3)
		text := refusedDrive(t, args...)
		if w.terminalCalls != 0 || !strings.Contains(text, "never self-updates") {
			t.Fatalf("the loop ran %d: %s", w.terminalCalls, text)
		}
	}
}

// TestARefusedExecRestoresOnlyTheBytesTheRunStartedOn (F1): the old process already holds
// the digest of the binary it runs, so nothing else goes back in place.
func TestARefusedExecRestoresOnlyTheBytesTheRunStartedOn(t *testing.T) {
	w := newUpdateWorld(t)
	driveReexec = func(string, []string, []string) error {
		if err := os.WriteFile(w.keptName(), []byte("not-a-release"), 0o755); err != nil {
			t.Fatal(err)
		}
		return errors.New("exec format error")
	}
	w.step(t).atBoundary(context.Background())
	if got := w.body(t, w.target); got != "new-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	w.oneLine(t, updTo, "exec format error", "does not match", ".abcd.previous")
}

// TestTheRecoveryLineNamesTheMoveOnlyForACopyThatVerifies (F2).
func TestTheRecoveryLineNamesTheMoveOnlyForACopyThatVerifies(t *testing.T) {
	note := func(t *testing.T, w *updateWorld) string {
		t.Helper()
		var b strings.Builder
		driveKeptNote(&b, driveFlags{self: driveSelfUpdateSteps.target(), afterUpdate: updFrom + "=" + updTo})
		return b.String()
	}
	t.Run("verified", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.useRelease(updTo)
		if err := os.WriteFile(w.keptName(), []byte("old-bytes"), 0o755); err != nil {
			t.Fatal(err)
		}
		if text := note(t, w); !strings.Contains(text, "move it over") {
			t.Fatalf("the line names the move: %s", text)
		}
	})
	for name, stub := range map[string]func(string, string) ([]byte, error){
		"does not verify": func(string, string) ([]byte, error) { return nil, errors.New("names another release") },
		"cannot run": func(string, string) ([]byte, error) {
			return nil, &update.NoVerdictError{Err: errors.New("no route")}
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := newUpdateWorld(t)
			w.useRelease(updTo)
			w.aForeignKeptCopy(t)
			driveSelfUpdateSteps.verifyKept = stub
			text := note(t, w)
			if strings.Contains(text, "move it over") || !strings.Contains(text, "abcd update "+updFrom) {
				t.Fatalf("the line names the update verb and the tag instead: %s", text)
			}
		})
	}
}

// keptStubs are the two checks on the kept copy in a world with no network: the bytes the
// world installed are the release, and the digest the run holds is theirs.
func keptStubs() (func(string, string) ([]byte, error), func(runner.Profile, string) ([sha256.Size]byte, bool)) {
	return func(keep, _ string) ([]byte, error) {
			raw, err := os.ReadFile(keep)
			if err != nil {
				return nil, err
			}
			if string(raw) != "old-bytes" {
				return nil, errors.New("its build metadata names no release")
			}
			return raw, nil
		}, func(runner.Profile, string) ([sha256.Size]byte, bool) {
			return sha256.Sum256([]byte("old-bytes")), true
		}
}

// candidateFor is what a rehearsal holds for a script standing in for the staged binary.
func candidateFor(t *testing.T, dir, path string) driveCandidate {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return driveCandidate{repoRoot: dir, path: path, sha256: hex.EncodeToString(sum[:]), env: os.Environ()}
}

// TestAStagedFileThatChangedOrBecameALinkIsNotRun (F3): the file is held to its verified
// digest, and to being a regular file, immediately before it is run.
func TestAStagedFileThatChangedOrBecameALinkIsNotRun(t *testing.T) {
	const script = "#!/bin/sh\necho ran >> \"$0.ran\"\nprintf '{\"dry_run\":true}\\n'\n"
	args := []string{"abcd", "drain", "--drive"}
	mk := func(t *testing.T) (string, driveCandidate) {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, ".abcd.new")
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		return dir, candidateFor(t, dir, path)
	}
	t.Run("the digest", func(t *testing.T) {
		dir, c := mk(t)
		if err := os.WriteFile(c.path, []byte(script+"# and more\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := driveRehearse(context.Background(), c, args); err == nil {
			t.Fatal("what changed is not run")
		}
		if _, err := os.Stat(c.path + ".ran"); err == nil {
			t.Fatalf("it ran in %s", dir)
		}
	})
	t.Run("a link to the same bytes", func(t *testing.T) {
		_, c := mk(t)
		real := filepath.Join(t.TempDir(), "elsewhere")
		if err := os.Rename(c.path, real); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, c.path); err != nil {
			t.Fatal(err)
		}
		if err := driveRehearse(context.Background(), c, args); err == nil {
			t.Fatal("a link is not run")
		}
		if _, err := os.Stat(real + ".ran"); err == nil {
			t.Fatal("it ran")
		}
	})
}

// TestAStagedFileThatChangesBeforeTheProofIsNotProved (F3).
func TestAStagedFileThatChangesBeforeTheProofIsNotProved(t *testing.T) {
	w := newUpdateWorld(t)
	driveSelfUpdateSteps.rehearse = func(_ context.Context, c driveCandidate, _ []string) error {
		return os.WriteFile(c.path, []byte("other bytes"), 0o755)
	}
	u := w.step(t)
	u.atBoundary(context.Background())
	if w.proved != 0 {
		t.Fatal("the proof ran on bytes that were not verified")
	}
	w.nothingSwapped(t)
	w.oneLine(t, updTo, "changed after it was verified")
	if u.tried[updTo] {
		t.Fatal("a changed staging file is no verdict on the release")
	}
}

// TestAnInstalledFileThatChangesBeforeTheReexecutionIsNotStarted (F3): the swap is held to
// the verified digest once more immediately before the exec, and what the run holds goes
// back in place.
func TestAnInstalledFileThatChangesBeforeTheReexecutionIsNotStarted(t *testing.T) {
	w := newUpdateWorld(t)
	commit := driveSelfUpdateSteps.commit
	driveSelfUpdateSteps.commit = func(target string, st update.Staged, keepAs string) error {
		if err := commit(target, st, keepAs); err != nil {
			return err
		}
		return os.WriteFile(target, []byte("other bytes"), 0o755)
	}
	u := w.step(t)
	u.atBoundary(context.Background())
	if len(w.reexecs) != 0 {
		t.Fatalf("started: %v", w.reexecs)
	}
	if got := w.body(t, w.target); got != "old-bytes" {
		t.Fatalf("the installed file holds %q", got)
	}
	if u.tried[updTo] {
		t.Fatal("a changed file is no verdict on the release")
	}
}

// TestAKeptCopyIsHeldToItsDigestBeforeItIsStarted (F3).
func TestAKeptCopyIsHeldToItsDigestBeforeItIsStarted(t *testing.T) {
	w := goBackWorld(t)
	verify := driveSelfUpdateSteps.verifyKept
	checked := []string{}
	prev := driveCheckBeforeExec
	t.Cleanup(func() { driveCheckBeforeExec = prev })
	driveCheckBeforeExec = func(path, sum string) error {
		checked = append(checked, path)
		return errors.New("the file changed")
	}
	driveSelfUpdateSteps.verifyKept = verify
	text := refusedDrive(t, resumeArgs()...)
	if len(checked) != 1 || checked[0] != w.target || len(w.reexecs) != 0 {
		t.Fatalf("checked %v, started %v", checked, w.reexecs)
	}
	if !strings.Contains(text, "installed binary") {
		t.Fatalf("the exit says why: %s", text)
	}
}

// TestTheRehearsalRunsInTheSealedEnvironment (F3): the candidate sees what a stage sees,
// not the drive's own environment.
func TestTheRehearsalRunsInTheSealedEnvironment(t *testing.T) {
	w := newUpdateWorld(t)
	t.Setenv("ABCD_A_PERSONS_SECRET", "kept out")
	seen := filepath.Join(t.TempDir(), "env")
	script := "#!/bin/sh\nenv > '" + seen + "'\ncase \"$*\" in *--help*) exit 0 ;; esac\nprintf '{\"dry_run\":true}\\n'\n"
	driveSelfUpdateSteps.stage = func(target, _ string) (update.Staged, error) {
		path := filepath.Join(filepath.Dir(target), ".abcd.new")
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			return update.Staged{}, err
		}
		return update.Staged{Path: path, SHA256: sha(script)}, nil
	}
	driveSelfUpdateSteps.rehearse = driveRehearse
	w.passWalls(t)
	home := t.TempDir()
	u := w.stepOn(runner.Profile{Dir: w.profile, ShellHome: home}, driveFlags{self: driveSelfUpdateSteps.target()})
	u.atBoundary(context.Background())
	raw, err := os.ReadFile(seen)
	if err != nil {
		t.Fatalf("the candidate did not run: %v, %q", err, w.lines)
	}
	if strings.Contains(string(raw), "ABCD_A_PERSONS_SECRET") || !strings.Contains(string(raw), "HOME="+home) {
		t.Fatalf("the candidate's environment:\n%s", raw)
	}
}

// TestAKeptCopyThatDoesNotHoldItsWallsEndsTheRun (F4): the kept copy proves itself live, and
// a wall it does not hold is said as any drive says one.
func TestAKeptCopyThatDoesNotHoldItsWallsEndsTheRun(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	w.liveDrain(t, 3)
	driveWalls.live = func(context.Context, runner.Profile) error {
		return &runner.WallError{Wall: runner.WallTools, Detail: "a tool too many"}
	}
	text := refusedDrive(t, "drain", "--drive", "--yes", "--update-refused", updTo)
	if w.terminalCalls != 0 || !strings.Contains(text, runner.WallTools) {
		t.Fatalf("the loop ran %d: %s", w.terminalCalls, text)
	}
}
