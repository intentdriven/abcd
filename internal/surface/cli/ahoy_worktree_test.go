package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

// worktreeDaysOn sets the door's clock days past now, for one test.
func worktreeDaysOn(t *testing.T, days int) {
	t.Helper()
	prev := worktreeNow
	worktreeNow = func() time.Time { return time.Now().AddDate(0, 0, days) }
	t.Cleanup(func() { worktreeNow = prev })
}

// TestAhoyWorktreePruneListsAndNamesQuietCandidates is criterion 11 at the
// door: fifteen days on, a quiet candidate is listed with its dossier, dirty
// state first, and left (exit 1); `--yes <name>` removes that one alone
// (exit 0); a name not in the lane refuses the run (exit 2), nothing removed.
func TestAhoyWorktreePruneListsAndNamesQuietCandidates(t *testing.T) {
	repo := worktreeFixture(t)
	gone := laneWorktree(t, repo, "gone")
	repo.Git("-C", gone, "commit", "-q", "--allow-empty", "-m", "try the idea (itd-2609091014076309)")
	stay := laneWorktree(t, repo, "stay")
	worktreeDaysOn(t, 15)

	out, code := runWorktree(t, false, "prune")
	if code != 1 || !strings.Contains(out, "quiet  gone") || !strings.Contains(out, "quiet  stay") {
		t.Fatalf("both are quiet candidates, left in place (exit %d):\n%s", code, out)
	}
	at := func(s string) int {
		i := strings.Index(out, s)
		if i < 0 {
			t.Fatalf("the dossier carries %q:\n%s", s, out)
		}
		return i
	}
	if !(at("dirty: none") < at("commits: 1 not on the default branch") && at("commits: 1") < at("try the idea") &&
		at("try the idea") < at("records: itd-2609091014076309") && at("records:") < at("last activity: ") &&
		at("last activity: ") < at("next: open a pull request")) {
		t.Fatalf("the dossier's order, dirty state first:\n%s", out)
	}
	for _, p := range []string{gone, stay} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("a candidate no one named stands: %v", err)
		}
	}

	if out, code := runWorktree(t, false, "prune", "--yes", "gone", "--yes", "nope"); code != 2 || !strings.Contains(out, "nope") {
		t.Fatalf("a name not in the lane refuses the run (exit %d):\n%s", code, out)
	}
	if _, err := os.Stat(gone); err != nil {
		t.Fatalf("a refused run removes nothing: %v", err)
	}

	js, code := runWorktree(t, true, "prune", "--yes", "gone")
	var got worktree.Pruned
	if err := json.Unmarshal([]byte(js), &got); err != nil || code != 0 {
		t.Fatalf("exit %d (%v):\n%s", code, err, js)
	}
	for _, r := range got.Rows {
		switch {
		case r.Name == "gone" && r.Outcome != worktree.OutcomeReclaimed:
			t.Errorf("the named candidate is reclaimed: %+v", r)
		case r.Name == "stay" && (r.Outcome != worktree.OutcomeQuiet || r.Dossier == nil):
			t.Errorf("the other candidate is listed and left: %+v", r)
		}
	}
	if _, err := os.Stat(gone); err == nil {
		t.Fatal("the named candidate is removed")
	}
	if _, err := os.Stat(stay); err != nil {
		t.Fatalf("the other candidate stands: %v", err)
	}
}

// TestAhoyWorktreePruneReadsTheQuietWindow: the door reads
// worktrees.quiet_days through the layered resolver, so a window wider than
// the quiet spell lists no candidate, and a bad value refuses the run naming
// its file before anything is judged.
func TestAhoyWorktreePruneReadsTheQuietWindow(t *testing.T) {
	repo := worktreeFixture(t)
	idle := laneWorktree(t, repo, "idle")
	worktreeDaysOn(t, 15)

	repo.Write(".abcd/config.json", `{"worktrees":{"quiet_days":20}}`)
	if out, code := runWorktree(t, false, "prune"); code != 1 || strings.Contains(out, "quiet  idle") || !strings.Contains(out, "unmerged") {
		t.Fatalf("a twenty-day window lists no candidate (exit %d):\n%s", code, out)
	}
	for _, bad := range []string{`{"worktrees":{"quiet_days":0}}`, `{"worktrees":{"quiet_dayz":3}}`} {
		repo.Write(".abcd/config.json", bad)
		out, code := runWorktree(t, false, "prune", "--yes", "idle")
		if code != 2 || !strings.Contains(out, ".abcd/config.json") {
			t.Fatalf("%s refuses the run naming its file (exit %d):\n%s", bad, code, out)
		}
	}
	if _, err := os.Stat(idle); err != nil {
		t.Fatalf("nothing is removed: %v", err)
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

// TestAhoyWorktreeAddChecksOutWhatTheGlobalFilterSmudges is
// iss-2610111729375108 at the store's door: a content filter defined in the
// person's global config (as a global `git lfs install` defines LFS's) applies
// to the worktree add makes, as it does to the person's own `git worktree
// add`, so the file holds the smudged bytes, not the committed pointer text.
func TestAhoyWorktreeAddChecksOutWhatTheGlobalFilterSmudges(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sed filter")
	}
	repo := worktreeFixture(t)
	repo.Write(".gitattributes", "*.bin filter=fake\n")
	repo.Write("asset.bin", "PTR\n")
	repo.Commit("an asset behind a filter")
	// Defined after the commit and in the global config alone, so only a git
	// that reads the person's global config smudges the checkout.
	cfg := filepath.Join(os.Getenv("HOME"), ".gitconfig")
	if err := os.WriteFile(cfg, []byte("[filter \"fake\"]\n\tsmudge = sed s/PTR/REAL/\n\tclean = sed s/REAL/PTR/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)

	out, code := runWorktree(t, true, "add", "assets")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(l.Path("assets"), "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "REAL\n" {
		t.Fatalf("the worktree holds what the person's own git checks out: asset.bin = %q, want %q", got, "REAL\n")
	}
}

// writeHook installs the repository's post-checkout hook with body.
func writeHook(t *testing.T, repo *gittest.Repo, body string) {
	t.Helper()
	hook := filepath.Join(repo.Root(), ".git", "hooks", "post-checkout")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestAhoyWorktreeAddRunsTheRepositorysPostCheckoutHook (iss-2610111729399920):
// the person's own `git worktree add` runs the repository's post-checkout hook
// (the one `git lfs install` installs) in the new worktree, so the store's add
// runs it too, as the build loop's lanes do.
func TestAhoyWorktreeAddRunsTheRepositorysPostCheckoutHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	repo := worktreeFixture(t)
	mark := filepath.Join(t.TempDir(), "post-checkout")
	writeHook(t, repo, "pwd -P > "+mark)

	if out, code := runWorktree(t, false, "add", "hooked"); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	got, err := os.ReadFile(mark)
	if err != nil {
		t.Fatal("adding the worktree did not run the repository's post-checkout hook")
	}
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if want := fsutil.RealExistingPath(l.Path("hooked")); strings.TrimSpace(string(got)) != want {
		t.Fatalf("the hook ran in %q, want the new worktree %q", strings.TrimSpace(string(got)), want)
	}
}

// TestAhoyWorktreeAddStopsACheckoutPastItsBound: a post-checkout hook that
// never finishes cannot hold add forever. Past the checkout's bound git and
// the hook are stopped, so git removes its lock files, and add is refused
// naming the checkout and the bound.
func TestAhoyWorktreeAddStopsACheckoutPastItsBound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	repo := worktreeFixture(t)
	writeHook(t, repo, "exec sleep 60")
	prev := worktreeAddTimeout
	worktreeAddTimeout = time.Second
	t.Cleanup(func() { worktreeAddTimeout = prev })

	began := time.Now()
	out, code := runWorktree(t, false, "add", "stalled")
	if took := time.Since(began); took > 20*time.Second {
		t.Fatalf("add took %s; the checkout's bound did not stop it", took)
	}
	if code != 2 || !strings.Contains(out, "did not finish") || !strings.Contains(out, "within 1s") {
		t.Fatalf("add is refused naming the checkout and its bound (exit %d):\n%s", code, out)
	}
	// git runs the hook after it has made and registered the worktree, so
	// the worktree stands, and add finishes it and says so.
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	p := l.Path("stalled")
	if !strings.Contains(out, "stands") || !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "/stalled\nHEAD ") {
		t.Fatalf("the worktree git made stands, and add says so:\n%s", out)
	}
	if fi, err := os.Lstat(filepath.Join(l.Dir, ".starts", "stalled")); err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("its start record is written: %v", err)
	}
	if fi, err := os.Lstat(filepath.Join(p, ".abcd", ".work.local")); err != nil || !fi.IsDir() {
		t.Fatalf("its local tier is laid: %v", err)
	}
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("it is the caller's alone: %v %v", fi.Mode(), err)
	}
	if again, code := runWorktree(t, false, "add", "stalled"); code != 2 || !strings.Contains(again, "occupied") || strings.Contains(again, "does not list") {
		t.Fatalf("a second add is refused as a taken name (exit %d):\n%s", code, again)
	}
	err = filepath.WalkDir(filepath.Join(repo.Root(), ".git"), func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Name() == "index.lock" {
			t.Errorf("git left a lock behind: %s", p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAhoyWorktreePruneDossierShowsIgnoredFiles: the dossier's dirty line
// reads "none" for a tree whose only files are ignored, so a line of its own
// says what a removal deletes unarchived.
func TestAhoyWorktreePruneDossierShowsIgnoredFiles(t *testing.T) {
	repo := worktreeFixture(t)
	idle := laneWorktree(t, repo, "idle")
	exclude := filepath.Join(repo.Root(), ".git", "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exclude, []byte("*.local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(idle, "env.local"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	worktreeDaysOn(t, 15)

	out, _ := runWorktree(t, false, "prune")
	if !strings.Contains(out, "ignored: 1 removed with it, unarchived: env.local") {
		t.Fatalf("the dossier does not name the ignored file a removal deletes:\n%s", out)
	}
}
