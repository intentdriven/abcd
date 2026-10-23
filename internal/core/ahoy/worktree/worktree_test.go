package worktree_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// repoWithCommit is a repository with one commit on main, under a fresh HOME.
func repoWithCommit(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Write("a.txt", "a\n")
	repo.Commit("one")
	return repo, home
}

// git is the runner the tests hand Add: git through gitutil's isolated
// environment, its failure returned.
func git(dir string, args ...string) (string, error) { return gitutil.Run(dir, args...) }

func names(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func mustRefusal(t *testing.T, err error) *worktree.Refusal {
	t.Helper()
	var r *worktree.Refusal
	if !errors.As(err, &r) {
		t.Fatalf("want a *worktree.Refusal, got %T %v", err, err)
	}
	if r.Reason == "" || r.Remedy == "" {
		t.Fatalf("a refusal names its reason and its remedy: %+v", r)
	}
	return r
}

// TestLaneForKeysOnTheFullRootCommit: the lane is
// ~/.abcd.noindex/worktrees/<root-sha>, the full 40-hex root commit the sibling
// stores key on, and a worktree in it is <lane>/<name>. Deriving it makes
// nothing.
func TestLaneForKeysOnTheFullRootCommit(t *testing.T) {
	repo, home := repoWithCommit(t)
	sha := gittest.RootCommit(t, repo.Root())
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if l.RootSHA != sha || !gitutil.IsFullSHA(l.RootSHA) {
		t.Fatalf("RootSHA = %q, want %q", l.RootSHA, sha)
	}
	if l.Home != home || l.Rel != abcdhome.Rel("worktrees", sha) || l.Dir != abcdhome.Path(home, "worktrees", sha) {
		t.Fatalf("lane = %+v", l)
	}
	if got := l.Path("x"); got != filepath.Join(l.Dir, "x") {
		t.Fatalf("Path = %q", got)
	}
	if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deriving the lane makes nothing: %v", err)
	}
}

// TestLaneForRefusesARepositoryWithNoCommits: no root commit, no key, and a
// refusal that says so rather than an error.
func TestLaneForRefusesARepositoryWithNoCommits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	_, err := worktree.LaneFor(repo.Root())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "no root commit") {
		t.Fatalf("the refusal names the missing root commit: %+v", r)
	}
}

// TestEnsureMakesEachLevelTheCallersAlone: every level Ensure makes is 0700, so
// the store it makes passes its own test on the next call.
func TestEnsureMakesEachLevelTheCallersAlone(t *testing.T) {
	repo, home := repoWithCommit(t)
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := l.Ensure(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	for _, level := range []string{abcdhome.Path(home), abcdhome.Path(home, "worktrees"), l.Dir} {
		if fi, err := os.Lstat(level); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
			t.Fatalf("%s is a 0700 directory: %v %v", level, fi, err)
		}
	}
}

// levels are the store's three levels for a lane, outermost first.
func levels(home string, l worktree.Lane) []string {
	return []string{abcdhome.Path(home), abcdhome.Path(home, "worktrees"), l.Dir}
}

// TestEnsureRefusesASymlinkedLevel is criterion 2: a symlink standing in for
// any level refuses the whole creation, nothing is made through it, and
// nothing is made beneath it.
func TestEnsureRefusesASymlinkedLevel(t *testing.T) {
	for i := range 3 {
		t.Run(filepath.Base(levels("/h", worktree.Lane{Dir: "/h/w/sha"})[i]), func(t *testing.T) {
			repo, home := repoWithCommit(t)
			l, err := worktree.LaneFor(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			lv := levels(home, l)
			if i > 0 {
				if err := os.MkdirAll(lv[i-1], 0o700); err != nil {
					t.Fatal(err)
				}
			}
			elsewhere := t.TempDir()
			if err := os.Symlink(elsewhere, lv[i]); err != nil {
				t.Fatal(err)
			}
			r := mustRefusal(t, l.Ensure())
			if !strings.Contains(r.Reason, "real directories") {
				t.Fatalf("the refusal names the store's levels as not real directories: %+v", r)
			}
			if strings.Contains(r.Error(), home) {
				t.Fatalf("the refusal is home-redacted: %q", r.Error())
			}
			if n := names(t, elsewhere); len(n) != 0 {
				t.Fatalf("nothing is made through the symlink: %v", n)
			}
			if fi, err := os.Lstat(lv[i]); err != nil || fi.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("the symlink is left as it was: %v %v", fi, err)
			}
		})
	}
}

// TestEnsureRefusesALevelAnyoneElseCanWrite is criterion 3's mode half: a level
// that already exists and is group- or world-writable refuses, is left as it
// was, and nothing is made beneath it.
func TestEnsureRefusesALevelAnyoneElseCanWrite(t *testing.T) {
	for i := range 3 {
		for _, mode := range []os.FileMode{0o770, 0o722, 0o777} {
			t.Run(mode.String(), func(t *testing.T) {
				repo, home := repoWithCommit(t)
				l, err := worktree.LaneFor(repo.Root())
				if err != nil {
					t.Fatal(err)
				}
				lv := levels(home, l)
				if err := os.MkdirAll(lv[i], 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(lv[i], mode); err != nil {
					t.Fatal(err)
				}
				r := mustRefusal(t, l.Ensure())
				if !strings.Contains(r.Reason, "writable by its group or by every user") {
					t.Fatalf("level %d at %s: want the writable level refused: %+v", i, mode, r)
				}
				if fi, err := os.Lstat(lv[i]); err != nil || fi.Mode().Perm() != mode {
					t.Fatalf("the level is left as it was: %v %v", fi, err)
				}
				if i < 2 {
					if _, err := os.Lstat(lv[i+1]); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("nothing is made beneath the refused level: %v", err)
					}
				} else if n := names(t, lv[i]); len(n) != 0 {
					t.Fatalf("nothing is made in the refused level: %v", n)
				}
			})
		}
	}
}

// TestEnsureRefusesALevelAnotherAccountOwns is criterion 3's owner half. A
// second uid is not something a test can create, so the owner lookup is
// substituted (fsutil.SwapOwnerUIDForTest) to answer another account for one
// level; nothing is made beneath it.
func TestEnsureRefusesALevelAnotherAccountOwns(t *testing.T) {
	for i := range 3 {
		t.Run(filepath.Base(levels("/h", worktree.Lane{Dir: "/h/w/sha"})[i]), func(t *testing.T) {
			repo, home := repoWithCommit(t)
			l, err := worktree.LaneFor(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			lv := levels(home, l)
			foreign := lv[i]
			restore := fsutil.SwapOwnerUIDForTest(func(p string) (uint32, error) {
				if p == foreign {
					return uint32(os.Getuid()) + 1, nil
				}
				return fsutil.OwnerUID(p)
			})
			defer restore()
			r := mustRefusal(t, l.Ensure())
			if !strings.Contains(r.Reason, "owned by another account") {
				t.Fatalf("want the foreign level refused: %+v", r)
			}
			if i < 2 {
				if _, err := os.Lstat(lv[i+1]); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("nothing is made beneath the refused level: %v", err)
				}
			}
		})
	}
}

// TestEnsureRefusesALaneThatIsNotTheStores: a Lane is LaneFor's answer; one
// assembled to point anywhere else under the home makes nothing.
func TestEnsureRefusesALaneThatIsNotTheStores(t *testing.T) {
	repo, home := repoWithCommit(t)
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []worktree.Lane{
		{},
		{Home: home, RootSHA: l.RootSHA, Rel: "projects", Dir: filepath.Join(home, "projects")},
		{Home: home, RootSHA: "0123abcd", Rel: abcdhome.Rel("worktrees", "0123abcd"), Dir: abcdhome.Path(home, "worktrees", "0123abcd")},
		{Home: home, RootSHA: l.RootSHA, Rel: l.Rel, Dir: filepath.Join(home, "elsewhere")},
	} {
		mustRefusal(t, bad.Ensure())
	}
	for _, p := range []string{abcdhome.Path(home), filepath.Join(home, "projects"), filepath.Join(home, "elsewhere")} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("nothing is made for a lane that is not the store's: %s (%v)", p, err)
		}
	}
}

// TestValidNameIsOnePlainPathSegment: a name cannot walk out of the lane or be
// read as an option.
func TestValidNameIsOnePlainPathSegment(t *testing.T) {
	for name, want := range map[string]bool{
		"ok-name": true, "run-1-lane-2": true, "a.b_c": true,
		"": false, ".": false, "..": false, "../x": false, "a/b": false, "-x": false,
		".hidden": false, "a..b": false, "x\x00y": false, "tab\tname": false, "é": false,
	} {
		if got := worktree.ValidName(name); got != want {
			t.Errorf("ValidName(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestAddMakesTheWorktreeInTheLane: a new branch cut from the base, the
// worktree at <lane>/<name> with that branch checked out, its directory 0700,
// and nothing made beside the checkout.
func TestAddMakesTheWorktreeInTheLane(t *testing.T) {
	repo, _ := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	parent := filepath.Dir(repo.Root())
	before := names(t, parent)

	got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "feature", Base: base, Git: git})
	if err != nil {
		t.Fatal(err)
	}
	l, _ := worktree.LaneFor(repo.Root())
	if got.Path != l.Path("feature") || got.Branch != "feature" || got.Lane != l {
		t.Fatalf("Added = %+v, lane %+v", got, l)
	}
	if b := strings.TrimSpace(repo.Git("-C", got.Path, "rev-parse", "--abbrev-ref", "HEAD")); b != "feature" {
		t.Fatalf("the worktree has its branch checked out: %q", b)
	}
	if h := strings.TrimSpace(repo.Git("rev-parse", "feature")); h != base {
		t.Fatalf("the branch is cut from the base: %s", h)
	}
	if fi, err := os.Lstat(got.Path); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the worktree directory is 0700: %v %v", fi, err)
	}
	if after := names(t, parent); !slices.Equal(before, after) {
		t.Fatalf("nothing is made beside the checkout: %v -> %v", before, after)
	}
	// Adopt finds what Add made, on its branch.
	if ok, err := worktree.Adopt(repo.Root(), l, "feature", "feature"); err != nil || !ok {
		t.Fatalf("Adopt finds the worktree Add made: %v %v", ok, err)
	}
}

// TestAddChecksOutAnExistingBranch: Existing checks the named branch out and
// makes no new one.
func TestAddChecksOutAnExistingBranch(t *testing.T) {
	repo, _ := repoWithCommit(t)
	repo.Git("branch", "topic")
	refs := repo.Git("for-each-ref", "refs/heads/")
	got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "wt", Branch: "topic", Existing: true, Git: git})
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != "topic" {
		t.Fatalf("branch = %q", got.Branch)
	}
	if b := strings.TrimSpace(repo.Git("-C", got.Path, "rev-parse", "--abbrev-ref", "HEAD")); b != "topic" {
		t.Fatalf("the existing branch is checked out: %q", b)
	}
	if after := repo.Git("for-each-ref", "refs/heads/"); after != refs {
		t.Fatalf("no branch is made: %q -> %q", refs, after)
	}
}

// TestAddRefusesAndWritesNothing: a name that is not one plain segment, a path
// already occupied in the lane, a new branch whose name is taken, an existing
// branch that is not there, and an invalid branch name each refuse, naming the
// reason, and leave the lane as it was.
func TestAddRefusesAndWritesNothing(t *testing.T) {
	repo, home := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Git("branch", "taken")
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		opts worktree.AddOptions
		want string
	}{
		"traversal":       {worktree.AddOptions{Name: "../x", Base: base}, "plain path segment"},
		"option":          {worktree.AddOptions{Name: "-x", Base: base}, "plain path segment"},
		"branch taken":    {worktree.AddOptions{Name: "taken", Base: base}, "already exists"},
		"no such branch":  {worktree.AddOptions{Name: "n", Branch: "absent", Existing: true}, "no local branch"},
		"bad branch name": {worktree.AddOptions{Name: "n", Branch: "a..b", Base: base}, "not a valid branch name"},
	} {
		tc.opts.Git = git
		_, err := worktree.Add(repo.Root(), tc.opts)
		if r := mustRefusal(t, err); !strings.Contains(r.Reason, tc.want) {
			t.Errorf("%s: want %q in the reason: %+v", name, tc.want, r)
		}
	}
	if _, err := os.Lstat(l.Dir); err == nil {
		if n := names(t, l.Dir); len(n) != 0 {
			t.Fatalf("nothing is made in the lane: %v", n)
		}
	}
	if !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "worktree ") || strings.Count(repo.Git("worktree", "list", "--porcelain"), "worktree ") != 1 {
		t.Fatal("git lists no new worktree")
	}

	// An occupied path is refused and left as it was.
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	squat := l.Path("squat")
	if err := os.Mkdir(squat, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(squat, "mine.txt"), []byte("theirs\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = worktree.Add(repo.Root(), worktree.AddOptions{Name: "squat", Base: base, Git: git})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "occupied") || strings.Contains(r.Error(), home) {
		t.Fatalf("want the occupied path refused, home-redacted: %+v", r)
	}
	if n := names(t, squat); !slices.Equal(n, []string{"mine.txt"}) {
		t.Fatalf("the squatter is left as it was: %v", n)
	}
	if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "n", Base: base}); err == nil {
		t.Fatal("Add with no git runner is an error, never a silent default")
	}
}

// TestAddKeepsGitsFailureAsTheCause: a git that fails is a refusal whose cause
// unwraps to git's own error, so a caller can tell its own runner's failures
// (a checkout past its bound) from the rest.
func TestAddKeepsGitsFailureAsTheCause(t *testing.T) {
	repo, _ := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	boom := errors.New("the checkout did not finish")
	_, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "n", Base: base,
		Git: func(string, ...string) (string, error) { return "", boom }})
	r := mustRefusal(t, err)
	if !errors.Is(err, boom) || r.Err != boom || !strings.Contains(r.Reason, "git could not add the worktree") {
		t.Fatalf("want git's failure kept as the cause: %+v", r)
	}
}

// TestAdoptRefusesAWorktreeOnAnotherBranch: git listing a worktree at the
// path on any branch but the one asked for is refused, never adopted.
func TestAdoptRefusesAWorktreeOnAnotherBranch(t *testing.T) {
	repo, _ := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "wt", Base: base, Git: git}); err != nil {
		t.Fatal(err)
	}
	l, _ := worktree.LaneFor(repo.Root())
	ok, err := worktree.Adopt(repo.Root(), l, "wt", "other")
	if r := mustRefusal(t, err); ok || !strings.Contains(r.Reason, "not on the branch") {
		t.Fatalf("want a worktree on another branch refused: %v %+v", ok, r)
	}
	if ok, err := worktree.Adopt(repo.Root(), l, "absent", "absent"); ok || err != nil {
		t.Fatalf("nothing at the path is nothing to adopt: %v %v", ok, err)
	}
}

// TestAddRefusesBeforeWritingAnything is criterion 4: each refusal comes
// before the store's levels are made, so a refused add leaves the home as it
// was: a name that is not one plain segment, the embargo store's name in any
// case (iss-2610230014564359, read from storepath.Name), a new branch whose
// name is taken, an existing branch that is not there, and an invalid branch
// name.
func TestAddRefusesBeforeWritingAnything(t *testing.T) {
	repo, home := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	repo.Git("branch", "taken")
	for name, tc := range map[string]struct {
		opts worktree.AddOptions
		want string
	}{
		"traversal":       {worktree.AddOptions{Name: "../x", Base: base}, "plain path segment"},
		"option":          {worktree.AddOptions{Name: "-x", Base: base}, "plain path segment"},
		"embargo":         {worktree.AddOptions{Name: storepath.Name, Base: base}, "embargo"},
		"embargo folded":  {worktree.AddOptions{Name: strings.ToUpper(storepath.Name), Base: base}, "embargo"},
		"branch taken":    {worktree.AddOptions{Name: "taken", Base: base}, "already exists"},
		"no such branch":  {worktree.AddOptions{Name: "n", Branch: "absent", Existing: true}, "no local branch"},
		"bad branch name": {worktree.AddOptions{Name: "n", Branch: "a..b", Base: base}, "not a valid branch name"},
	} {
		tc.opts.Git = git
		_, err := worktree.Add(repo.Root(), tc.opts)
		if r := mustRefusal(t, err); !strings.Contains(r.Reason, tc.want) {
			t.Errorf("%s: want %q in the reason: %+v", name, tc.want, r)
		}
		if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s: a refused add writes nothing under the home: %v", name, err)
		}
	}
	if worktree.ValidName(storepath.Name) {
		t.Fatal("the embargo store's name is never a worktree's name")
	}
}

// TestAddRefusesATakenName: a name a worktree already holds in the lane is
// refused, and the lane and git's worktrees are left as they were.
func TestAddRefusesATakenName(t *testing.T) {
	repo, _ := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "wt", Base: base, Git: git}); err != nil {
		t.Fatal(err)
	}
	l, _ := worktree.LaneFor(repo.Root())
	lane, listing := names(t, l.Dir), repo.Git("worktree", "list", "--porcelain")
	_, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "wt", Branch: "other", Base: base, Git: git})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "occupied") {
		t.Fatalf("want the taken name refused: %+v", r)
	}
	if after := names(t, l.Dir); !slices.Equal(lane, after) {
		t.Fatalf("the lane is left as it was: %v -> %v", lane, after)
	}
	if after := repo.Git("worktree", "list", "--porcelain"); after != listing {
		t.Fatalf("git lists no new worktree:\n%s", after)
	}
	if _, err := gitutil.Run(repo.Root(), "rev-parse", "--verify", "--quiet", "refs/heads/other"); err == nil {
		t.Fatal("no branch is made for a refused add")
	}
}

// TestAddRefusesARepositoryWithNoCommits is criterion 4's last case: no root
// commit, no lane, a refusal saying so, and nothing written.
func TestAddRefusesARepositoryWithNoCommits(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	_, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "wt", Git: git})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "no root commit") {
		t.Fatalf("the refusal names the missing root commit: %+v", r)
	}
	if n := names(t, home); slices.Contains(n, filepath.Base(abcdhome.Path(home))) {
		t.Fatalf("nothing is written under the home: %v", n)
	}
}

// TestAddLaysTheLocalTier (iss-2610222022157889): a worktree the store adds
// has its own .abcd/.work.local/, the caller's alone, so a reservation can
// keep its key there at once; a .abcd/ the branch carries keeps its mode.
func TestAddLaysTheLocalTier(t *testing.T) {
	repo, _ := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "bare", Base: base, Git: git})
	if err != nil {
		t.Fatal(err)
	}
	tier := filepath.Join(got.Path, ".abcd", ".work.local")
	if fi, err := os.Lstat(tier); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the local tier is a 0700 directory: %v %v", fi, err)
	}
	if st := repo.Git("-C", got.Path, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("an empty tier leaves the tree clean: %q", st)
	}

	repo.Write(".abcd/config.json", "{}\n")
	repo.Commit("carry .abcd")
	base = strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	got, err = worktree.Add(repo.Root(), worktree.AddOptions{Name: "carried", Base: base, Git: git})
	if err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(filepath.Join(got.Path, ".abcd", ".work.local")); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the local tier is a 0700 directory beside the branch's .abcd: %v %v", fi, err)
	}
	if fi, err := os.Lstat(filepath.Join(got.Path, ".abcd")); err != nil || fi.Mode().Perm() == 0o700 {
		t.Fatalf("the branch's own .abcd keeps the mode git gave it: %v %v", fi, err)
	}
}

// TestAddNeverFollowsWhatStandsAtTheLocalTier: a branch that carries a
// symlink or a file where the tier, or the .abcd above it, would be is
// refused, naming the tier and the worktree git made; nothing is made
// through the link, and what stands there is left as it was.
func TestAddNeverFollowsWhatStandsAtTheLocalTier(t *testing.T) {
	for name, plant := range map[string]func(t *testing.T, repo *gittest.Repo, elsewhere string){
		"tier is a symlink": func(t *testing.T, repo *gittest.Repo, elsewhere string) {
			repo.Write(".abcd/config.json", "{}\n")
			if err := os.Symlink(elsewhere, filepath.Join(repo.Root(), ".abcd", ".work.local")); err != nil {
				t.Fatal(err)
			}
		},
		".abcd is a symlink": func(t *testing.T, repo *gittest.Repo, elsewhere string) {
			if err := os.Symlink(elsewhere, filepath.Join(repo.Root(), ".abcd")); err != nil {
				t.Fatal(err)
			}
		},
		"tier is a file": func(t *testing.T, repo *gittest.Repo, _ string) {
			repo.Write(".abcd/.work.local", "not a directory\n")
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, home := repoWithCommit(t)
			elsewhere := t.TempDir()
			plant(t, repo, elsewhere)
			repo.Git("add", "-A", "-f")
			repo.Git("commit", "-q", "-m", "plant")
			base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
			got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "wt", Base: base, Git: git})
			r := mustRefusal(t, err)
			if !strings.Contains(r.Reason, ".abcd/.work.local") || !strings.Contains(r.Reason, "made") || strings.Contains(r.Error(), home) {
				t.Fatalf("the refusal names the tier and the worktree git made, home-redacted: %+v", r)
			}
			if n := names(t, elsewhere); len(n) != 0 {
				t.Fatalf("nothing is made through the link: %v", n)
			}
			l, _ := worktree.LaneFor(repo.Root())
			if got.Path != l.Path("wt") {
				t.Fatalf("the worktree git made is named: %+v", got)
			}
			if fi, err := os.Lstat(filepath.Join(got.Path, ".abcd")); err != nil {
				t.Fatalf("what the branch carries is left in place: %v %v", fi, err)
			}
		})
	}
}

// TestAddRecordsTheStartCommit (iss-2610231011465325): add records where the
// worktree's branch starts, in the lane beside the worktrees, owner-only: a
// new branch's base, or an existing branch's tip as add checks it out. A
// later add at the same name records its own start over the earlier one.
func TestAddRecordsTheStartCommit(t *testing.T) {
	repo, _ := repoWithCommit(t)
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "fresh", Base: base, Git: git})
	if err != nil {
		t.Fatal(err)
	}
	branch, start, ok := worktree.StartForTest(got.Lane, "fresh")
	if !ok || branch != "refs/heads/fresh" || start != base {
		t.Fatalf("a new branch starts at its base: %q %q %v", branch, start, ok)
	}
	dir := filepath.Join(got.Lane.Dir, ".starts")
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the start records' directory is 0700: %v %v", fi, err)
	}
	if fi, err := os.Lstat(filepath.Join(dir, "fresh")); err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
		t.Fatalf("a start record is a 0600 file: %v %v", fi, err)
	}

	repo.Git("checkout", "-q", "-b", "topic")
	repo.Write("t.txt", "t\n")
	repo.Commit("topic work")
	repo.Git("checkout", "-q", "main")
	tip := strings.TrimSpace(repo.Git("rev-parse", "topic"))
	if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "existing", Branch: "topic", Existing: true, Git: git}); err != nil {
		t.Fatal(err)
	}
	if branch, start, ok := worktree.StartForTest(got.Lane, "existing"); !ok || branch != "refs/heads/topic" || start != tip {
		t.Fatalf("an existing branch starts at its tip as add checks it out: %q %q %v", branch, start, ok)
	}

	repo.Git("worktree", "remove", got.Path)
	if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "fresh", Branch: "again", Base: tip, Git: git}); err != nil {
		t.Fatal(err)
	}
	if branch, start, ok := worktree.StartForTest(got.Lane, "fresh"); !ok || branch != "refs/heads/again" || start != tip {
		t.Fatalf("a later add records its own start: %q %q %v", branch, start, ok)
	}
}
