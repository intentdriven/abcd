package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// `abcd ahoy worktree` is the worktree store's front door
// (spc-2609301811532881): bare and `list` render the repository's worktrees
// and its lane, `list --all` every lane on the machine. The core judges the
// rows and masks every string in them; this door lays them out.

// worktreeFixture is a repository with one commit under a fresh HOME, the
// caller standing in it, and the forge answering nothing.
func worktreeFixture(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "a\n")
	repo.Commit("one")
	t.Chdir(repo.Root())
	prev := worktreeForge
	worktreeForge = func(string, ...string) (string, error) { return "", errors.New("no git remotes found") }
	t.Cleanup(func() { worktreeForge = prev })
	return repo
}

// laneWorktree adds the worktree name to repo's lane on a new branch.
func laneWorktree(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	base := repo.Git("rev-parse", "main")
	got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: name, Base: base,
		Git: func(dir string, args ...string) (string, error) { return gitutil.Run(dir, args...) }})
	if err != nil {
		t.Fatal(err)
	}
	return got.Path
}

// runWorktree runs `ahoy worktree` with args and returns what it printed and
// the exit code it asked for.
func runWorktree(t *testing.T, asJSON bool, args ...string) (string, int) {
	t.Helper()
	cmd := newAhoyWorktreeCommand(&asJSON)
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

// TestAhoyWorktreeListEscapesGitsReasons: a lock reason carrying an escape
// sequence and a line break reaches neither form raw, so it can neither
// recolour the terminal nor forge a row of its own.
func TestAhoyWorktreeListEscapesGitsReasons(t *testing.T) {
	repo := worktreeFixture(t)
	p := laneWorktree(t, repo, "held")
	repo.Git("worktree", "lock", "--reason", "paused\x1b[31m red\nforged  row  clean", p)

	text, code := runWorktree(t, false)
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, text)
	}
	if strings.ContainsAny(text, "\x1b\x9b") || !strings.Contains(text, "locked: paused?[31m red?forged  row  clean") {
		t.Fatalf("the text form masks the reason in its row:\n%q", text)
	}
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if strings.HasPrefix(line, "forged") {
			t.Fatalf("the reason forged a line of its own:\n%s", text)
		}
	}
	if !strings.Contains(text, "no forge answered") {
		t.Fatalf("the run says once that the forge did not answer:\n%s", text)
	}

	js, code := runWorktree(t, true, "list")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, js)
	}
	if strings.Contains(js, `\u001b`) || strings.Contains(js, `\n`) {
		t.Fatalf("the JSON form carries the masked reason:\n%s", js)
	}
	var got worktree.Listing
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("%v\n%s", err, js)
	}
	if len(got.Worktrees) != 1 || got.Worktrees[0].LockReason != "paused?[31m red?forged  row  clean" {
		t.Fatalf("the row: %+v", got.Worktrees)
	}
}

// TestAhoyWorktreeListAllEmitsArrays: --all walks the store, and the JSON
// form's collections are arrays, never null, an empty store included.
func TestAhoyWorktreeListAllEmitsArrays(t *testing.T) {
	repo := worktreeFixture(t)
	js, code := runWorktree(t, true, "list", "--all")
	if code != 0 || !strings.Contains(js, `"lanes": []`) {
		t.Fatalf("an empty store lists no lanes as an array (exit %d):\n%s", code, js)
	}
	laneWorktree(t, repo, "wt")
	js, code = runWorktree(t, true, "list", "--all")
	var got worktree.AllListing
	if err := json.Unmarshal([]byte(js), &got); err != nil || code != 0 {
		t.Fatalf("exit %d, %v\n%s", code, err, js)
	}
	if len(got.Lanes) != 1 || len(got.Lanes[0].Worktrees) != 1 || got.Lanes[0].Worktrees[0].Name != "wt" {
		t.Fatalf("one lane holding wt: %+v", got.Lanes)
	}
	text, code := runWorktree(t, false, "list", "--all")
	if code != 0 || !strings.Contains(text, "wt") || !strings.Contains(text, got.Lanes[0].Key) {
		t.Fatalf("the text form names the lane and its worktree (exit %d):\n%s", code, text)
	}
}

// TestAhoyWorktreeRefusesOutsideARepository: a fault exits 2.
func TestAhoyWorktreeRefusesOutsideARepository(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	if out, code := runWorktree(t, false, "list"); code != 2 {
		t.Fatalf("exit %d, want 2:\n%s", code, out)
	}
}

// mergedLaneWorktree adds name to the lane with a commit of its own and
// fast-forwards main over it: merged and clean.
func mergedLaneWorktree(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	p := laneWorktree(t, repo, name)
	repo.Git("-C", p, "commit", "-q", "--allow-empty", "-m", "work on "+name)
	repo.Git("merge", "-q", "--ff-only", name)
	return p
}

// TestAhoyWorktreePruneExitCodes is decision 8: 1 when the run reclaimed
// nothing, an empty lane included; 0 once one worktree is reclaimed; a dry
// run answers the code the real run would, and --json carries the rows.
func TestAhoyWorktreePruneExitCodes(t *testing.T) {
	repo := worktreeFixture(t)
	if out, code := runWorktree(t, false, "prune"); code != 1 {
		t.Fatalf("an empty lane: exit %d, want 1:\n%s", code, out)
	}
	work := laneWorktree(t, repo, "work")
	repo.Git("-C", work, "commit", "-q", "--allow-empty", "-m", "unmerged work")
	if out, code := runWorktree(t, false, "prune"); code != 1 || !strings.Contains(out, "work") || !strings.Contains(out, "unmerged") {
		t.Fatalf("an unmerged worktree is kept: exit %d:\n%s", code, out)
	}
	done := mergedLaneWorktree(t, repo, "done")

	js, code := runWorktree(t, true, "prune", "--dry-run")
	if code != 0 {
		t.Fatalf("a dry run answers the real run's code: exit %d:\n%s", code, js)
	}
	var dry worktree.Pruned
	if err := json.Unmarshal([]byte(js), &dry); err != nil || !dry.DryRun || len(dry.Rows) != 2 {
		t.Fatalf("the dry run's rows (%v):\n%s", err, js)
	}
	if _, err := gitutil.Run(done, "rev-parse", "HEAD"); err != nil {
		t.Fatalf("a dry run removes nothing: %v", err)
	}
	text, code := runWorktree(t, false, "prune")
	if code != 0 || !strings.Contains(text, "done") || !strings.Contains(text, "reclaimed") {
		t.Fatalf("a merged clean worktree is reclaimed: exit %d:\n%s", code, text)
	}
}

// TestAhoyWorktreePruneRefusesAStoreItCannotTrust is criterion 12 at the
// door: exit 2, nothing removed.
func TestAhoyWorktreePruneRefusesAStoreItCannotTrust(t *testing.T) {
	repo := worktreeFixture(t)
	done := mergedLaneWorktree(t, repo, "done")
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(l.Dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if out, code := runWorktree(t, false, "prune"); code != 2 || !strings.Contains(out, "writable by its group or by every user") {
		t.Fatalf("exit %d, want 2:\n%s", code, out)
	}
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("nothing is removed: %v", err)
	}
}

// TestAhoyWorktreePruneKeepsWhatTheLoopHolds: the door hands prune the build
// loop's own reader, and a run whose state cannot be read holds every
// worktree named after it.
func TestAhoyWorktreePruneKeepsWhatTheLoopHolds(t *testing.T) {
	repo := worktreeFixture(t)
	busy := mergedLaneWorktree(t, repo, "run-0000000000000007-lane-1")
	repo.Write(".abcd/.work.local/run/run-0000000000000007/state.json", "not a state file\n")
	out, code := runWorktree(t, false, "prune")
	if code != 1 || !strings.Contains(out, "held by the build loop's run run-0000000000000007") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if _, err := os.Stat(busy); err != nil {
		t.Fatalf("the held worktree stands: %v", err)
	}
}
