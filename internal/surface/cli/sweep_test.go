package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/sweep"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// `abcd sweep` is the front door of internal/core/sweep (spc-2610230730269295,
// step 5): bare, the dry run's rows; --apply, the same rows with what was done;
// --json, the rows as the core answers them; and the core's exit codes.

// sweepFixture is worktreeFixture with a local tier, an origin holding main, a merged lane
// worktree "done" and a merged local branch "loc" with no worktree: one item a
// sweep removes and one it deletes. The probe sees no process and reads its
// clock a long way ahead, so the fixture's fresh writes are not recent.
func sweepFixture(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo := worktreeFixture(t)
	// The local tier, where the reservations' fetch is stamped: a checkout
	// with none can never show a worktree unreserved.
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	origin := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, origin)
	repo.Git("remote", "add", "origin", origin)
	repo.Git("push", "-q", "origin", "main")
	done := mergedLaneWorktree(t, repo, "done")
	repo.Git("switch", "-q", "-c", "loc")
	repo.Git("commit", "-q", "--allow-empty", "-m", "work on loc")
	repo.Git("switch", "-q", "main")
	repo.Git("merge", "-q", "--ff-only", "loc")
	repo.Git("push", "-q", "origin", "main")
	ahead := time.Now().Add(1000 * time.Hour)
	prev := sweepOptions
	sweepOptions = func(o *sweep.Options) {
		o.Probe.Processes = func(context.Context) ([]sweep.Process, error) { return nil, nil }
		o.Probe.Now = func() time.Time { return ahead }
	}
	t.Cleanup(func() { sweepOptions = prev })
	return repo, done
}

// runSweepCLI runs `abcd sweep` with args and returns what it printed and the
// exit code it asked for.
func runSweepCLI(t *testing.T, asJSON bool, args ...string) (string, int) {
	t.Helper()
	cmd := newSweepCommand(&asJSON)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	code := 0
	if err != nil {
		code = 1
		var coded interface{ ExitCode() int }
		if errors.As(err, &coded) {
			code = coded.ExitCode()
		}
		out.WriteString(err.Error())
	}
	return out.String(), code
}

func sweepRow(t *testing.T, res sweep.Result, kind sweep.RowKind, name string) sweep.Row {
	t.Helper()
	for _, r := range res.Rows {
		if r.Kind == kind && r.Name == name {
			return r
		}
	}
	t.Fatalf("no %s row named %q in %+v", kind, name, res.Rows)
	return sweep.Row{}
}

// TestSweepVerbRowsJSONAndExitCodes: the bare verb lists what --apply would do
// and changes nothing, exiting 0 because apply would act; --json carries the
// same rows; --apply removes the worktree and deletes the branch, naming
// each, and exits 0; a second --apply finds nothing and exits 1.
func TestSweepVerbRowsJSONAndExitCodes(t *testing.T) {
	repo, done := sweepFixture(t)

	text, code := runSweepCLI(t, false)
	if code != sweep.ExitActed {
		t.Fatalf("a dry run with work for --apply: exit %d, want 0:\n%s", code, text)
	}
	for _, want := range []string{"dry run", "remove  worktree  done", "delete  branch  loc"} {
		if !strings.Contains(text, want) {
			t.Errorf("the dry run does not say %q:\n%s", want, text)
		}
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("the dry run removed the worktree: %v", err)
	}
	if _, err := gitutil.Run(repo.Root(), "rev-parse", "--verify", "refs/heads/loc"); err != nil {
		t.Fatalf("the dry run deleted the branch: %v", err)
	}

	js, code := runSweepCLI(t, true)
	var dry sweep.Result
	if err := json.Unmarshal([]byte(js), &dry); err != nil || code != sweep.ExitActed || dry.Applied {
		t.Fatalf("--json dry run: exit %d, %v:\n%s", code, err, js)
	}
	if r := sweepRow(t, dry, sweep.RowWorktree, "done"); r.Action != sweep.ActionRemove {
		t.Fatalf("the worktree's row in --json: %+v", r)
	}
	if r := sweepRow(t, dry, sweep.RowBranch, "loc"); r.Action != sweep.ActionDelete {
		t.Fatalf("the branch's row in --json: %+v", r)
	}

	js, code = runSweepCLI(t, true, "--apply")
	var applied sweep.Result
	if err := json.Unmarshal([]byte(js), &applied); err != nil || code != sweep.ExitActed || !applied.Applied {
		t.Fatalf("--apply: exit %d, %v:\n%s", code, err, js)
	}
	if r := sweepRow(t, applied, sweep.RowWorktree, "done"); r.Outcome != sweep.OutcomeRemoved {
		t.Fatalf("the worktree was not removed: %+v", r)
	}
	if r := sweepRow(t, applied, sweep.RowBranch, "loc"); r.Outcome != sweep.OutcomeDeleted {
		t.Fatalf("the branch was not deleted: %+v", r)
	}
	if _, err := os.Stat(done); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the worktree stands after --apply: %v", err)
	}

	text, code = runSweepCLI(t, false, "--apply")
	if code != sweep.ExitNothing || !strings.Contains(text, "--apply") {
		t.Fatalf("a second --apply with nothing spent: exit %d, want 1:\n%s", code, text)
	}
}

// TestSweepVerbRefusesOutsideACheckout: no checkout, no sweep, exit 2.
func TestSweepVerbRefusesOutsideACheckout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	out, code := runSweepCLI(t, false)
	if code != sweep.ExitRefused || !strings.Contains(out, "not inside a git checkout") {
		t.Fatalf("outside a checkout: exit %d, want 2:\n%s", code, out)
	}
}

// TestSweepVerbRefusesAFailedFetch: an origin that cannot be fetched refuses
// the run before anything is judged, exit 2, the merged branch left standing.
func TestSweepVerbRefusesAFailedFetch(t *testing.T) {
	repo, _ := sweepFixture(t)
	repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	out, code := runSweepCLI(t, false, "--apply")
	if code != sweep.ExitRefused || !strings.Contains(out, "fetching origin") {
		t.Fatalf("a failed fetch: exit %d, want 2:\n%s", code, out)
	}
	if _, err := gitutil.Run(repo.Root(), "rev-parse", "--verify", "refs/heads/loc"); err != nil {
		t.Fatalf("a refused run deleted the branch: %v", err)
	}
}
