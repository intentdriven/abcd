package worktree_test

import (
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
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newRepo is a repository with one commit on main under the HOME already set,
// its first file named for seed so two fixtures made in one second never
// share a root commit.
func newRepo(t *testing.T, seed string) *gittest.Repo {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(seed+".txt", seed+"\n")
	repo.Commit("root " + seed)
	return repo
}

// addToLane makes the worktree name in repo's lane on a new branch of the
// same name, cut from main, and returns its path.
func addToLane(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	base := strings.TrimSpace(repo.Git("rev-parse", "main"))
	got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: name, Base: base, Git: git})
	if err != nil {
		t.Fatal(err)
	}
	return got.Path
}

// commitIn writes rel in the worktree at dir and commits it there.
func commitIn(t *testing.T, repo *gittest.Repo, dir, rel, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", dir, "add", "-A")
	repo.Git("-C", dir, "commit", "-q", "-m", "change "+rel)
	return strings.TrimSpace(repo.Git("-C", dir, "rev-parse", "HEAD"))
}

func rowNamed(t *testing.T, rows []worktree.Row, name string) worktree.Row {
	t.Helper()
	for _, r := range rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no row named %q in %+v", name, rows)
	return worktree.Row{}
}

// TestListRendersTheLaneAndASibling is criterion 6: a lane of two worktrees and
// a third beside the checkout render three rows, the third outside the store,
// each with its branch, clean or dirty, and merged or unmerged. The main
// checkout is not a row.
func TestListRendersTheLaneAndASibling(t *testing.T) {
	repo, home := repoWithCommit(t)
	a := addToLane(t, repo, "alpha")
	b := addToLane(t, repo, "beta")
	commitIn(t, repo, b, "b.txt", "b\n")
	repo.Git("merge", "-q", "--ff-only", "beta")
	if err := os.WriteFile(filepath.Join(a, "scratch.txt"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(filepath.Dir(repo.Root()), "sibling")
	repo.Git("worktree", "add", "-q", "-b", "sib", sibling)

	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Worktrees) != 3 {
		t.Fatalf("want three rows, got %+v", got.Worktrees)
	}
	l, _ := worktree.LaneFor(repo.Root())
	if got.Lane != fsutil.RedactHome(l.Dir) || strings.Contains(got.Lane, home) {
		t.Fatalf("the lane is named home-redacted: %q", got.Lane)
	}
	for name, want := range map[string]worktree.Row{
		"alpha":   {Branch: "alpha", Where: "store", State: "dirty", Merged: "unmerged"},
		"beta":    {Branch: "beta", Where: "store", State: "clean", Merged: "merged", JudgedBy: "ancestry"},
		"sibling": {Branch: "sib", Where: "outside", State: "clean", Merged: "unmerged"},
	} {
		r := rowNamed(t, got.Worktrees, name)
		if r.Branch != want.Branch || r.Where != want.Where || r.State != want.State || r.Merged != want.Merged || r.JudgedBy != want.JudgedBy || r.Reason != "" {
			t.Errorf("%s: got %+v, want %+v", name, r, want)
		}
	}
	if p := rowNamed(t, got.Worktrees, "alpha").Path; p != fsutil.RedactHome(a) || strings.Contains(p, home) {
		t.Errorf("a store row's path is home-redacted: %q", p)
	}
	if p := rowNamed(t, got.Worktrees, "sibling").Path; p != fsutil.DisplayPath(sibling) {
		t.Errorf("an outside row's path is its display form: %q", p)
	}
}

// TestListNamesWhatTheLaneHoldsThatIsNoWorktree: a directory in the lane git
// does not list, and one directly under the store named by an abbreviation of
// this repository's root commit, each render with their reason in place of
// the state columns.
func TestListNamesWhatTheLaneHoldsThatIsNoWorktree(t *testing.T) {
	repo, _ := repoWithCommit(t)
	addToLane(t, repo, "kept")
	l, _ := worktree.LaneFor(repo.Root())
	if err := os.Mkdir(l.Path("stray"), 0o700); err != nil {
		t.Fatal(err)
	}
	short := filepath.Join(filepath.Dir(l.Dir), l.RootSHA[:12])
	if err := os.Mkdir(short, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	stray := rowNamed(t, got.Worktrees, "stray")
	if stray.Reason != "not a worktree of this repository" || stray.State != "" || stray.Merged != "" {
		t.Fatalf("an unlisted directory in the lane: %+v", stray)
	}
	abbr := rowNamed(t, got.Worktrees, l.RootSHA[:12])
	if !strings.Contains(abbr.Reason, "not a lane") || !strings.Contains(abbr.Reason, l.RootSHA) ||
		!strings.Contains(abbr.Reason, "git worktree remove") || abbr.State != "" {
		t.Fatalf("an abbreviated directory names the full-key lane and the hand retirement: %+v", abbr)
	}
}

// TestListFromAMovedCheckoutFindsTheSameLane is criterion 18: the lane is keyed
// on the root commit, so the moved checkout lists the same lane, and a
// worktree whose .git file still names the old place says how to repair it.
func TestListFromAMovedCheckoutFindsTheSameLane(t *testing.T) {
	repo, _ := repoWithCommit(t)
	addToLane(t, repo, "wt")
	before, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(repo.Root(), moved); err != nil {
		t.Fatal(err)
	}
	after, err := worktree.List(moved, worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if after.Lane != before.Lane {
		t.Fatalf("the moved checkout's lane: %q, want %q", after.Lane, before.Lane)
	}
	r := rowNamed(t, after.Worktrees, "wt")
	if r.Reason != "checkout not found; run `git worktree repair` from the checkout" || r.State != "" || r.Merged != "" {
		t.Fatalf("a worktree pointing at the old place: %+v", r)
	}
}

// TestListShowsTheEmbargoStoreAsAbcdsOwn (iss-2610230014564359): the embargo
// store is a worktree of the repository in the same lane, listed as abcd's
// own sibling store and never judged as a lane worktree.
func TestListShowsTheEmbargoStoreAsAbcdsOwn(t *testing.T) {
	repo, _ := repoWithCommit(t)
	l, _ := worktree.LaneFor(repo.Root())
	if err := l.Ensure(); err != nil {
		t.Fatal(err)
	}
	repo.Git("worktree", "add", "-q", "--detach", l.Path(storepath.Name))

	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := rowNamed(t, got.Worktrees, storepath.Name)
	if r.Where != "sibling" || r.State != "" || r.Merged != "" || !strings.Contains(r.Reason, "embargo store") {
		t.Fatalf("the embargo store is abcd's own sibling store, unjudged: %+v", r)
	}
	all, err := worktree.ListAll(worktree.AllOptions{Here: repo.Root()})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Lanes) != 1 {
		t.Fatalf("one lane: %+v", all.Lanes)
	}
	r = rowNamed(t, all.Lanes[0].Worktrees, storepath.Name)
	if r.Where != "sibling" || r.State != "" || r.Merged != "" {
		t.Fatalf("--all lists the embargo store as abcd's own: %+v", r)
	}
}

// TestListEscapesGitsLockReason: a lock reason is whatever bytes the person
// who locked the worktree gave git, so it reaches the row masked, a control
// byte and a line break included, and the JSON form carries the masked text.
func TestListEscapesGitsLockReason(t *testing.T) {
	repo, _ := repoWithCommit(t)
	p := addToLane(t, repo, "held")
	reason := "paused\x1b[31m red\nforged line\x9b"
	repo.Git("worktree", "lock", "--reason", reason, p)

	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := rowNamed(t, got.Worktrees, "held")
	if !r.Locked || r.LockReason != termsafe.Sanitize(reason) {
		t.Fatalf("the lock reason is masked: %+v", r)
	}
	js, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(js), `\u001b`) || strings.Contains(string(js), `\n`) || strings.Contains(string(js), "\u009b") {
		t.Fatalf("the JSON form carries the masked reason: %s", js)
	}
}

// TestListAllLabelsEveryLane is criterion 7: lanes of two registered
// repositories render under their registry names and an unregistered one under
// its root commit; a registry path label naming a directory that no longer
// exists changes no row; a lane whose checkout moved renders as one "checkout
// not found" row with the repair hint; and a directory named by an abbreviated
// root commit is "not a lane", naming the full-key lane of the repository the
// caller stands in.
func TestListAllLabelsEveryLane(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	alpha, beta, gamma, delta := newRepo(t, "alpha"), newRepo(t, "beta"), newRepo(t, "gamma"), newRepo(t, "delta")
	for _, r := range []*gittest.Repo{alpha, beta, gamma, delta} {
		addToLane(t, r, "wt")
	}
	shas := map[*gittest.Repo]string{}
	for _, r := range []*gittest.Repo{alpha, beta, gamma, delta} {
		shas[r] = gittest.RootCommit(t, r.Root())
	}
	sha := func(r *gittest.Repo) string { return shas[r] }
	if err := os.Rename(delta.Root(), filepath.Join(t.TempDir(), "moved")); err != nil {
		t.Fatal(err)
	}
	store := abcdhome.Path(os.Getenv("HOME"), "worktrees")
	if err := os.Mkdir(filepath.Join(store, sha(alpha)[:12]), 0o700); err != nil {
		t.Fatal(err)
	}

	labels := map[string]worktree.Label{
		sha(alpha): {Name: "alpha-project", Path: alpha.Root()},
		sha(beta):  {Name: "beta-project", Path: beta.Root()},
	}
	got, err := worktree.ListAll(worktree.AllOptions{Labels: labels, Here: alpha.Root()})
	if err != nil {
		t.Fatal(err)
	}
	lanes := map[string]worktree.LaneListing{}
	for _, l := range got.Lanes {
		lanes[l.Key] = l
	}
	if len(lanes) != 5 {
		t.Fatalf("four lanes and one abbreviated directory: %+v", got.Lanes)
	}
	for r, label := range map[*gittest.Repo]string{alpha: "alpha-project", beta: "beta-project", gamma: sha(gamma)} {
		l := lanes[sha(r)]
		if l.Label != label || l.Reason != "" || len(l.Worktrees) != 1 {
			t.Errorf("lane of %s: %+v", label, l)
			continue
		}
		if w := l.Worktrees[0]; w.Name != "wt" || w.Branch != "wt" || w.State != "clean" || w.Merged != "unmerged" {
			t.Errorf("lane of %s's worktree: %+v", label, w)
		}
	}
	moved := lanes[sha(delta)]
	if moved.Reason != "checkout not found; run `git worktree repair` from the checkout" || len(moved.Worktrees) != 0 || moved.Label != sha(delta) {
		t.Errorf("a lane whose checkout moved: %+v", moved)
	}
	abbr := lanes[sha(alpha)[:12]]
	if !strings.Contains(abbr.Reason, "not a lane") || !strings.Contains(abbr.Reason, sha(alpha)) || !strings.Contains(abbr.Reason, "git worktree remove") {
		t.Errorf("an abbreviated directory: %+v", abbr)
	}

	// The registry's path label is never read: one naming a directory that
	// is gone renders the same rows.
	labels[sha(beta)] = worktree.Label{Name: "beta-project", Path: filepath.Join(t.TempDir(), "gone")}
	stale, err := worktree.ListAll(worktree.AllOptions{Labels: labels, Here: alpha.Root()})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(stale)
	if string(a) != string(b) {
		t.Fatalf("a stale registry path changed the rows:\n%s\n%s", a, b)
	}
}

// TestListRefusesASymlinkedStore: the store's levels are held to the tests
// Ensure holds them to before anything under them is read, and a listing
// makes no level that is absent.
func TestListRefusesASymlinkedStore(t *testing.T) {
	repo, home := repoWithCommit(t)
	if _, err := worktree.List(repo.Root(), worktree.ListOptions{}); err != nil {
		t.Fatalf("an absent store lists nothing: %v", err)
	}
	if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("listing makes no store level: %v", err)
	}
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), abcdhome.Path(home, "worktrees")); err != nil {
		t.Fatal(err)
	}
	_, err := worktree.List(repo.Root(), worktree.ListOptions{})
	mustRefusal(t, err)
	_, err = worktree.ListAll(worktree.AllOptions{})
	mustRefusal(t, err)
}

// TestListKeepsTheStartRecordsOutOfTheRows: the start records add keeps in
// the lane are the store's own bookkeeping, never a row of list or --all and
// so never a candidate for prune.
func TestListKeepsTheStartRecordsOutOfTheRows(t *testing.T) {
	repo, _ := repoWithCommit(t)
	addToLane(t, repo, "wt")
	l, _ := worktree.LaneFor(repo.Root())
	if _, err := os.Lstat(filepath.Join(l.Dir, ".starts")); err != nil {
		t.Fatalf("add keeps its start records in the lane: %v", err)
	}
	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Worktrees) != 1 || got.Worktrees[0].Name != "wt" {
		t.Fatalf("one row, the worktree: %+v", got.Worktrees)
	}
	all, err := worktree.ListAll(worktree.AllOptions{Here: repo.Root()})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Lanes) != 1 || len(all.Lanes[0].Worktrees) != 1 || all.Lanes[0].Worktrees[0].Name != "wt" {
		t.Fatalf("--all: one lane of one row: %+v", all.Lanes)
	}
}

// TestListCarriesEachRowsAbsolutePath: a caller that acts on a row, the sweep,
// needs the worktree's own path, which the display path redacts or shortens.
// List fills it for a worktree inside HOME (the lane) and one outside it.
func TestListCarriesEachRowsAbsolutePath(t *testing.T) {
	repo, home := repoWithCommit(t)
	inside := addToLane(t, repo, "inside")
	outside := filepath.Join(filepath.Dir(repo.Root()), "outside")
	repo.Git("worktree", "add", "-q", "--detach", outside)
	if fsutil.PathWithin(fsutil.RealExistingPath(outside), fsutil.RealExistingPath(home), fsutil.CaseFoldingFS()) {
		t.Fatalf("the fixture's outside worktree sits under HOME: %s", outside)
	}

	got, err := worktree.List(repo.Root(), worktree.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"inside": inside, "outside": outside} {
		r := rowNamed(t, got.Worktrees, name)
		if !filepath.IsAbs(r.Abs) || fsutil.RealExistingPath(r.Abs) != fsutil.RealExistingPath(want) {
			t.Errorf("%s: Abs %q, want %q", name, r.Abs, want)
		}
		if r.Path == r.Abs {
			t.Errorf("%s: the display path stays redacted: %q", name, r.Path)
		}
	}
}
