package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
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

// TestAhoyWorktreeListNeverPrintsTheAbsolutePath: the row's absolute path is
// for a caller that acts on it, and reaches neither the text nor the JSON form
// of list, for a worktree inside HOME or outside it.
func TestAhoyWorktreeListNeverPrintsTheAbsolutePath(t *testing.T) {
	repo := worktreeFixture(t)
	laneWorktree(t, repo, "inside")
	repo.Git("worktree", "add", "-q", "--detach", filepath.Join(filepath.Dir(repo.Root()), "outside"))
	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var abs []string
	for _, r := range got.Worktrees {
		if r.Abs == "" {
			t.Fatalf("%s carries no absolute path", r.Name)
		}
		abs = append(abs, r.Abs, fsutil.RealExistingPath(r.Abs))
	}
	if len(got.Worktrees) != 2 {
		t.Fatalf("two rows: %+v", got.Worktrees)
	}
	for _, asJSON := range []bool{false, true} {
		out, code := runWorktree(t, asJSON, "list")
		if code != 0 {
			t.Fatalf("json=%v: exit %d:\n%s", asJSON, code, out)
		}
		for _, a := range abs {
			if strings.Contains(out, a) {
				t.Fatalf("json=%v: %s reached the output:\n%s", asJSON, a, out)
			}
		}
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

// TestAhoyWorktreePruneRedactsFileNamesInItsReasons: a file the archive
// cannot take is named in the kept row's reason, and that name passes the
// same redaction as the files themselves, in the text and the JSON forms.
func TestAhoyWorktreePruneRedactsFileNamesInItsReasons(t *testing.T) {
	token := "ghp_" + strings.Repeat("a", 40)
	for name, content := range map[string][]byte{
		"binary":    {0, 1, 2, 0xff},
		"oversized": bytes.Repeat([]byte("x"), 4<<20+1),
	} {
		t.Run(name, func(t *testing.T) {
			repo := worktreeFixture(t)
			repo.Write(".gitignore", ".abcd/.work.local/\n")
			repo.Commit("ignore the local tier")
			p := mergedLaneWorktree(t, repo, "kept")
			tier := filepath.Join(p, ".abcd", ".work.local")
			if err := os.MkdirAll(tier, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tier, "note-"+token+".bin"), content, 0o644); err != nil {
				t.Fatal(err)
			}
			for _, asJSON := range []bool{false, true} {
				out, code := runWorktree(t, asJSON, "prune")
				if code != 1 || !strings.Contains(out, "notes could not be archived") {
					t.Fatalf("json=%v: the worktree is kept (exit %d):\n%s", asJSON, code, out)
				}
				if strings.Contains(out, token) {
					t.Fatalf("json=%v: the file name reached the output unredacted:\n%s", asJSON, out)
				}
			}
		})
	}
}

// TestAhoyWorktreeAddPrintsThePath is criteria 1 and 19: add makes the
// worktree in the lane on a new branch cut from the default branch, prints
// its path home-redacted (and as `path` in JSON), leaves the checkout's
// parent as it was, and narrows the worktree to 0700.
func TestAhoyWorktreeAddPrintsThePath(t *testing.T) {
	repo := worktreeFixture(t)
	parent := filepath.Dir(repo.Root())
	before, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	text, code := runWorktree(t, false, "add", "feature")
	if code != 0 || text != fsutil.RedactHome(l.Path("feature"))+"\n" || strings.Contains(text, l.Home) {
		t.Fatalf("exit %d; want the path alone, home-redacted:\n%q", code, text)
	}
	if b := repo.Git("-C", l.Path("feature"), "rev-parse", "--abbrev-ref", "HEAD"); b != "feature" {
		t.Fatalf("the worktree is on its new branch: %q", b)
	}
	if b, m := repo.Git("rev-parse", "feature"), repo.Git("rev-parse", "main"); b != m {
		t.Fatalf("the branch is cut from the default branch: %s, main %s", b, m)
	}
	if fi, err := os.Lstat(l.Path("feature")); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the worktree is 0700: %v %v", fi, err)
	}
	after, err := os.ReadDir(parent)
	if err != nil || len(after) != len(before) {
		t.Fatalf("nothing is made beside the checkout: %d -> %d entries (%v)", len(before), len(after), err)
	}

	js, code := runWorktree(t, true, "add", "second")
	var got struct{ Name, Branch, Path string }
	if err := json.Unmarshal([]byte(js), &got); err != nil || code != 0 {
		t.Fatalf("exit %d, %v\n%s", code, err, js)
	}
	if got.Name != "second" || got.Branch != "second" || got.Path != fsutil.RedactHome(l.Path("second")) {
		t.Fatalf("the JSON form names the worktree: %+v", got)
	}
}

// TestAhoyWorktreeAddChecksOutAnExistingBranch is criterion 4's last half:
// --branch checks an existing branch out and makes none.
func TestAhoyWorktreeAddChecksOutAnExistingBranch(t *testing.T) {
	repo := worktreeFixture(t)
	repo.Git("branch", "topic")
	refs := repo.Git("for-each-ref", "refs/heads/")
	text, code := runWorktree(t, false, "add", "wt", "--branch", "topic")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, text)
	}
	l, _ := worktree.LaneFor(repo.Root())
	if b := repo.Git("-C", l.Path("wt"), "rev-parse", "--abbrev-ref", "HEAD"); b != "topic" {
		t.Fatalf("the existing branch is checked out: %q", b)
	}
	if after := repo.Git("for-each-ref", "refs/heads/"); after != refs {
		t.Fatalf("no branch is made: %q -> %q", refs, after)
	}
}

// TestAhoyWorktreeAddRefusesAndWritesNothing is criterion 4: `../x`, `-x`, a
// taken name, the embargo store's name, a branch that already exists without
// --branch, and a repository with no commits each exit 2 naming the reason,
// and write nothing.
func TestAhoyWorktreeAddRefusesAndWritesNothing(t *testing.T) {
	repo := worktreeFixture(t)
	home, _ := os.UserHomeDir()
	repo.Git("branch", "exists")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"add", "../x"}, "plain path segment"},
		{[]string{"add", "--", "-x"}, "plain path segment"},
		{[]string{"add", storepath.Name}, "embargo"},
		{[]string{"add", "exists"}, "--branch exists"},
		{[]string{"add", "wt", "--branch", "absent"}, "no local branch"},
	} {
		text, code := runWorktree(t, false, tc.args...)
		if code != 2 || !strings.Contains(text, tc.want) {
			t.Errorf("%v: exit %d, want 2 naming %q:\n%s", tc.args, code, tc.want, text)
		}
		if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%v: a refused add writes nothing under the home: %v", tc.args, err)
		}
	}

	if _, code := runWorktree(t, false, "add", "taken"); code != 0 {
		t.Fatal("the first add of a name succeeds")
	}
	l, _ := worktree.LaneFor(repo.Root())
	lane, err := os.ReadDir(l.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if text, code := runWorktree(t, false, "add", "taken"); code != 2 || !strings.Contains(text, "occupied") {
		t.Fatalf("a taken name: exit %d:\n%s", code, text)
	}
	if after, err := os.ReadDir(l.Dir); err != nil || len(after) != len(lane) {
		t.Fatalf("the lane is left as it was: %d -> %d (%v)", len(lane), len(after), err)
	}

	t.Setenv("HOME", t.TempDir())
	t.Chdir(gittest.NewRepo(t).Root())
	home, _ = os.UserHomeDir()
	if text, code := runWorktree(t, false, "add", "wt"); code != 2 || !strings.Contains(text, "no root commit") {
		t.Fatalf("a repository with no commits: exit %d:\n%s", code, text)
	}
	if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a repository with no commits writes nothing under the home: %v", err)
	}
}
