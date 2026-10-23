package cli

import (
	"bytes"
	"encoding/json"
	"errors"
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
