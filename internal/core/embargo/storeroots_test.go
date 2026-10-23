package embargo

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// datedRoot makes a commit with no parent, holding the empty tree and dated
// date, without moving any branch, and returns its object name.
func datedRoot(t *testing.T, r *gittest.Repo, date, msg string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"commit-tree", r.Git("hash-object", "-t", "tree", "/dev/null"), "-m", msg)
	cmd.Env = append(r.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("commit-tree: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// mergeUnrelatedRoot merges, with an empty diff (`merge -s ours`), a root
// commit of an unrelated history dated far ahead, so `rev-list` meets it
// before the repository's own root. It returns the merged root.
func mergeUnrelatedRoot(t *testing.T, r *gittest.Repo) string {
	t.Helper()
	other := datedRoot(t, r, "2090-01-01T00:00:00Z", "an unrelated root")
	r.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", other)
	if gittest.RootCommit(t, r.Root()) != other {
		t.Fatal("the fixture's rev-list does not meet the merged root first")
	}
	return other
}

// holdSample saves the sample state, which holds ghsaOpen unpublished.
func holdSample(t *testing.T, s Store) {
	t.Helper()
	if err := s.SaveState(sampleState()); err != nil {
		t.Fatal(err)
	}
}

// checkStdin checks one stdin text in the repository at repo.
func checkStdin(home, repo, body string) (CheckResult, error) {
	return Check(CheckRequest{Home: home, RepoRoot: repo, Files: []string{"-"}, Stdin: strings.NewReader(body), Now: func() time.Time { return fixedNow }})
}

// wantEmbargoedIDRefused wants the check in repo to read a store and refuse
// ghsaOpen.
func wantEmbargoedIDRefused(t *testing.T, home, repo string) {
	t.Helper()
	res, err := checkStdin(home, repo, "fixes "+ghsaOpen+"\n")
	if err != nil {
		t.Fatalf("the check refused: %v", err)
	}
	if res.NoStore || len(res.Findings) != 1 || res.Clean() {
		t.Fatalf("the check = %+v; want the store read and the id refused once", res)
	}
}

// wantStoreFound wants openStore to find the store at want and make nothing.
func wantStoreFound(t *testing.T, home, repo, want string) {
	t.Helper()
	s, created, err := openStore(home, repo)
	if err != nil || created || s.Root != want {
		t.Fatalf("openStore = %+v, created %v, %v; want the store at %s, found", s, created, err, want)
	}
}

// TestStoreFoundWhicheverRootGitListsFirst: after an empty-diff merge of an
// unrelated root, which `rev-list` now meets first, the store made before it
// is still found: the embargoed record is still refused, the check does
// not report "no store", and opening the store finds it and makes no second
// one under the merged root.
func TestStoreFoundWhicheverRootGitListsFirst(t *testing.T) {
	cf := newCheckFixture(t)
	other := mergeUnrelatedRoot(t, cf.f.repo)
	wantEmbargoedIDRefused(t, cf.f.home, cf.f.repo.Root())
	wantStoreFound(t, cf.f.home, cf.f.repo.Root(), cf.s.Root)
	second := abcdhome.Path(cf.f.home, "worktrees", other, StoreName)
	if _, err := os.Lstat(second); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a second store was made under the merged root (%v)", err)
	}
}

// TestCheckRefusesTheFoundStoresKeyedPath: the store-path form the check
// refuses names the folder the found store is in, not the root `rev-list`
// meets first.
func TestCheckRefusesTheFoundStoresKeyedPath(t *testing.T) {
	cf := newCheckFixture(t)
	key := filepath.Base(filepath.Dir(cf.s.Root))
	mergeUnrelatedRoot(t, cf.f.repo)
	wantKinds(t, cf.text(t, "under worktrees/"+key+"/embargo/.abcd\n"), KindStorePath)
}

// TestStoreFoundAfterOriginHeadMovesToAnUnrelatedBranch: origin's HEAD moved
// to a branch with a history of its own does not hide the store: the check
// still reads it and refuses the embargoed id, and no second store is made.
func TestStoreFoundAfterOriginHeadMovesToAnUnrelatedBranch(t *testing.T) {
	cf := newCheckFixture(t)
	repo := cf.f.repo
	repo.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	repo.Git("update-ref", "refs/remotes/origin/other", datedRoot(t, repo, "2001-01-01T00:00:00Z", "another history"))
	repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/other")
	wantEmbargoedIDRefused(t, cf.f.home, repo.Root())
	wantStoreFound(t, cf.f.home, repo.Root(), cf.s.Root)
}

// TestStoreFoundPastManyMergedRoots: a merge that brings in 1,400 parentless
// commits does not stop the check: the store is still found and the
// embargoed id refused.
func TestStoreFoundPastManyMergedRoots(t *testing.T) {
	cf := newCheckFixture(t)
	cf.f.repo.MergeParentlessRoots(1400)
	wantEmbargoedIDRefused(t, cf.f.home, cf.f.repo.Root())
	wantStoreFound(t, cf.f.home, cf.f.repo.Root(), cf.s.Root)
}

// TestMergedInRepositorysStoreIsReadNotWritten: bar merges in the history
// of foo, whose root is older than bar's, and foo has a store on this
// machine under that root. bar holds no store, yet its check reads foo's
// and refuses foo's embargoed id; its first intake makes bar's own store
// under bar's own root; its check then reads both; and foo's store is never
// changed, so foo's check still refuses foo's embargoed id.
func TestMergedInRepositorysStoreIsReadNotWritten(t *testing.T) {
	bar := newFixture(t)
	foo := gittest.NewRepo(t)
	foo.Write(".abcd/work/issues/open/.gitkeep", "")
	foo.Write("README.md", "another project\n")
	foo.Git("add", "-A")
	fooRoot := datedCommit(t, foo, "2001-01-01T00:00:00Z", "foo's root")
	fooStore, created, err := openStore(bar.home, foo.Root())
	if err != nil || !created {
		t.Fatalf("foo's store: created %v, %v", created, err)
	}
	holdSample(t, fooStore)
	before := listTree(t, fooStore.Root)

	bar.repo.Git("fetch", "-q", foo.Root(), "main:refs/heads/foo")
	bar.repo.Git("merge", "-q", "-s", "ours", "--allow-unrelated-histories", "--no-edit", "foo")
	barRoot := bar.repo.Git("rev-list", "--max-parents=0", "--first-parent", "HEAD")
	if roots, err := gitutil.DefaultBranchRoots(bar.repo.Root()); err != nil || len(roots) != 2 || roots[0] != fooRoot {
		t.Fatalf("bar's roots %q, %v; want foo's older root first", roots, err)
	}

	wantEmbargoedIDRefused(t, bar.home, bar.repo.Root())
	if after := listTree(t, fooStore.Root); after != before {
		t.Fatalf("bar's check changed foo's store:\nbefore %s\nafter  %s", before, after)
	}
	s, created, err := openStore(bar.home, bar.repo.Root())
	if err != nil || !created {
		t.Fatalf("bar's intake store: created %v, %v; want bar's own made", created, err)
	}
	if want := abcdhome.Path(bar.home, "worktrees", barRoot, StoreName); s.Root != want {
		t.Fatalf("bar's store at %s; want it under bar's own root, %s", s.Root, want)
	}
	wantEmbargoedIDRefused(t, bar.home, bar.repo.Root())
	if after := listTree(t, fooStore.Root); after != before {
		t.Fatalf("foo's store changed:\nbefore %s\nafter  %s", before, after)
	}
	wantEmbargoedIDRefused(t, bar.home, foo.Root())
}

// cloneOf clones the fixture's repository into a folder of its own, the way
// a second checkout of the same project is made, and returns its root.
func cloneOf(t *testing.T, f fixture) string {
	t.Helper()
	clone := filepath.Join(t.TempDir(), "clone")
	for _, args := range [][]string{
		{"clone", "-q", "file://" + f.repo.Root(), clone},
		{"-C", clone, "config", "--local", "gc.auto", "0"},
		{"-C", clone, "config", "--local", "core.fsmonitor", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Env = f.repo.Env()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return clone
}

// TestSecondCloneChecksAgainstTheFirstClonesStore: a second clone of the
// same project, with no store of its own, reads the store the first clone
// made under their shared root: its check refuses the first clone's
// embargoed id and the store's own path, reports a store, and leaves that
// store exactly as it was.
func TestSecondCloneChecksAgainstTheFirstClonesStore(t *testing.T) {
	cf := newCheckFixture(t)
	b := cloneOf(t, cf.f)
	before := listTree(t, cf.s.Root)
	wantEmbargoedIDRefused(t, cf.f.home, b)
	key := filepath.Base(filepath.Dir(cf.s.Root))
	res, err := checkStdin(cf.f.home, b, "under worktrees/"+key+"/embargo/.abcd\n")
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindStorePath)
	if res.UnderEmbargo != 3 || res.IntakeAt == nil {
		t.Fatalf("the second clone's check = %+v; want the first clone's store counted", res)
	}
	if after := listTree(t, cf.s.Root); after != before {
		t.Fatalf("the second clone's check changed the first clone's store:\nbefore %s\nafter  %s", before, after)
	}
	if _, _, err := openStore(cf.f.home, b); err == nil {
		t.Fatal("the second clone made or opened a store beside the first clone's")
	}
	if after := listTree(t, cf.s.Root); after != before {
		t.Fatalf("the second clone's intake changed the first clone's store:\nbefore %s\nafter  %s", before, after)
	}
}

// TestCheckRefusesAnUnreadableStoreAtARootsPlace: a store at a root's place
// whose state file cannot be read refuses the second clone's check; it is
// never skipped as though it held nothing.
func TestCheckRefusesAnUnreadableStoreAtARootsPlace(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads any file")
	}
	cf := newCheckFixture(t)
	b := cloneOf(t, cf.f)
	if err := os.Chmod(cf.s.StatePath(), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cf.s.StatePath(), 0o600) })
	res, err := checkStdin(cf.f.home, b, "a text\n")
	if err == nil {
		t.Fatalf("the check = %+v and no refusal", res)
	}
	mustRefuse(t, err, "state file")
}

// TestCheckRefusesAMergedRootsPlaceThatCannotBeExamined: with this
// repository's own store registered, a plain folder at the place under a
// merged root is read as a store holding nothing, so the check still reads
// its own; that place unable to be examined refuses the check: it may hide
// a store whose records the check would otherwise miss.
func TestCheckRefusesAMergedRootsPlaceThatCannotBeExamined(t *testing.T) {
	cf := newCheckFixture(t)
	other := mergeUnrelatedRoot(t, cf.f.repo)
	place := abcdhome.Path(cf.f.home, "worktrees", other, StoreName)
	if err := os.MkdirAll(place, 0o700); err != nil {
		t.Fatal(err)
	}
	wantEmbargoedIDRefused(t, cf.f.home, cf.f.repo.Root())
	if os.Geteuid() == 0 {
		return
	}
	if err := os.Chmod(filepath.Dir(place), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(place), 0o700) })
	_, err := checkStdin(cf.f.home, cf.f.repo.Root(), "a text\n")
	mustRefuse(t, err, "examined")
}

// datedCommit commits what is staged in r, dated date, and returns the
// commit's object name.
func datedCommit(t *testing.T, r *gittest.Repo, date, msg string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", msg)
	cmd.Env = append(r.Env(), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
	return r.Git("rev-parse", "HEAD")
}

// listTree is every path under dir with its size and mode, one per line.
func listTree(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		b.WriteString(rel + " " + fi.Mode().String() + " " + fi.ModTime().String())
		if fi.Mode().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			b.WriteString(" " + string(data))
		}
		b.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestStoreFoundAfterAShallowCloneDeepens: a store made in a shallow clone,
// whose only root is the clone's boundary commit, is still found once the
// clone is deepened and the boundary is gone.
func TestStoreFoundAfterAShallowCloneDeepens(t *testing.T) {
	src := newFixture(t)
	src.repo.Write("a.txt", "one\n")
	src.repo.Commit("c1")
	src.repo.Write("a.txt", "two\n")
	src.repo.Commit("c2")
	clone := filepath.Join(t.TempDir(), "clone")
	cloneGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Env = src.repo.Env()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	cloneGit("clone", "-q", "--depth", "1", "file://"+src.repo.Root(), clone)
	for _, kv := range [][2]string{{"gc.auto", "0"}, {"gc.autodetach", "false"}, {"maintenance.auto", "false"}, {"core.fsmonitor", "false"}} {
		cloneGit("-C", clone, "config", "--local", kv[0], kv[1])
	}
	boundary := cloneGit("-C", clone, "rev-parse", "HEAD")
	s, created, err := openStore(src.home, clone)
	if err != nil || !created {
		t.Fatalf("openStore: created %v, %v", created, err)
	}
	if want := abcdhome.Path(src.home, "worktrees", boundary, StoreName); s.Root != want {
		t.Fatalf("the shallow clone's store is at %s; want it keyed on the boundary, %s", s.Root, want)
	}
	holdSample(t, s)
	cloneGit("-C", clone, "fetch", "-q", "--unshallow")
	if gittest.RootCommit(t, clone) == boundary {
		t.Fatal("the deepened clone still has the boundary for its root")
	}
	wantEmbargoedIDRefused(t, src.home, clone)
	wantStoreFound(t, src.home, clone, s.Root)
}

// TestStoreMadeUnderAReplacedRootIsFound: a store made by an earlier abcd,
// which keyed it on the root it saw through a replace ref, is found, though
// the history read as stored has another root.
func TestStoreMadeUnderAReplacedRootIsFound(t *testing.T) {
	f := newFixture(t)
	f.repo.Write("NEXT.md", "more\n")
	f.repo.Commit("c1")
	tip := f.repo.Git("rev-parse", "HEAD")
	standIn := f.repo.Git("commit-tree", f.repo.Git("rev-parse", "HEAD^{tree}"), "-m", "a stand-in root")
	f.repo.Git("replace", tip, standIn)
	stored := f.repo.Git("-c", "core.useReplaceRefs=false", "rev-list", "--max-parents=0", "HEAD")
	oldKey := gittest.RootCommit(t, f.repo.Root())
	if !gitutil.IsFullSHA(oldKey) || oldKey == stored {
		t.Fatalf("the replace ref does not move the root an earlier abcd keyed on (%q, stored %q)", oldKey, stored)
	}
	s := Store{Root: abcdhome.Path(f.home, "worktrees", oldKey, StoreName), Checkout: f.repo.Root()}
	if err := s.create(f.home); err != nil {
		t.Fatal(err)
	}
	holdSample(t, s)
	wantEmbargoedIDRefused(t, f.home, f.repo.Root())
	wantStoreFound(t, f.home, f.repo.Root(), s.Root)
}

// TestTwoRegisteredStoresAreRefused: with two embargo stores registered as
// worktrees of this repository, which one holds the records cannot be told,
// so the check, the opening of an existing store and the making of one all
// refuse, saying how to inspect and remove one and naming neither store,
// and nothing is written to either.
func TestTwoRegisteredStoresAreRefused(t *testing.T) {
	cf := newCheckFixture(t)
	key := strings.Repeat("ab", 20)
	second := Store{Root: abcdhome.Path(cf.f.home, "worktrees", key, StoreName), Checkout: cf.f.repo.Root()}
	if err := second.create(cf.f.home); err != nil {
		t.Fatal(err)
	}
	before := listTree(t, second.Root)
	_, err1 := checkStdin(cf.f.home, cf.f.repo.Root(), "a text\n")
	_, _, err2 := openExisting(cf.f.home, cf.f.repo.Root())
	_, _, err3 := openStore(cf.f.home, cf.f.repo.Root())
	for _, err := range []error{err1, err2, err3} {
		mustRefuse(t, err, "git worktree list")
		mustRefuse(t, err, "git worktree remove")
		for _, private := range []string{key, filepath.Base(filepath.Dir(cf.s.Root)), cf.f.home, "aside"} {
			if strings.Contains(err.Error(), private) {
				t.Fatalf("the refusal %q says %q", err, private)
			}
		}
	}
	if listTree(t, second.Root) != before {
		t.Fatal("the second store was written to")
	}
}

// TestCheckRefusesWhenTheWorktreeListingFails: when this repository's
// worktrees cannot be listed (here a lock reason longer than any field the
// listing reads), the store cannot be looked for, and the check, the opening
// of an existing store and the making of one all refuse rather than report
// no store.
func TestCheckRefusesWhenTheWorktreeListingFails(t *testing.T) {
	cf := newCheckFixture(t)
	lockWithReason(t, cf.s.Root, strings.Repeat("r", 2<<20))
	_, err1 := checkStdin(cf.f.home, cf.f.repo.Root(), "a text\n")
	_, _, err2 := openExisting(cf.f.home, cf.f.repo.Root())
	_, _, err3 := openStore(cf.f.home, cf.f.repo.Root())
	for _, err := range []error{err1, err2, err3} {
		mustRefuse(t, err, "could not list this repository's worktrees")
	}
}

// lockWithReason writes reason as the lock reason of the linked worktree at
// wt, into its administrative folder, where `git worktree lock` keeps it.
func lockWithReason(t *testing.T, wt, reason string) {
	t.Helper()
	gitfile, err := os.ReadFile(filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	admin, ok := strings.CutPrefix(strings.TrimSpace(string(gitfile)), "gitdir: ")
	if !ok {
		t.Fatalf("%s/.git is not a gitfile", wt)
	}
	if err := os.WriteFile(filepath.Join(admin, "locked"), []byte(reason), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckRefusesWhenTheStoresPlaceCannotBeExamined: with no store
// registered and the place this repository's store would be made unable to
// be examined (its folder's parent cannot be searched), the check refuses
// rather than report no store: "no store" is what lets a check pass.
func TestCheckRefusesWhenTheStoresPlaceCannotBeExamined(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches any folder")
	}
	f := newFixture(t)
	parent := abcdhome.Path(f.home, "worktrees")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	res, err := checkStdin(f.home, f.repo.Root(), "a text\n")
	if err == nil {
		t.Fatalf("the check = %+v and no refusal", res)
	}
	mustRefuse(t, err, "examined")
}

// TestOrdinaryCloneKeepsItsStorePath: in a clone with one root, the store is
// made where it always was, under that root.
func TestOrdinaryCloneKeepsItsStorePath(t *testing.T) {
	f := newFixture(t)
	f.repo.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	f.repo.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	s, created, err := openStore(f.home, f.repo.Root())
	if err != nil || !created {
		t.Fatalf("openStore: created %v, %v", created, err)
	}
	if want := abcdhome.Path(f.home, "worktrees", gittest.RootCommit(t, f.repo.Root()), StoreName); s.Root != want {
		t.Fatalf("store at %s; want %s", s.Root, want)
	}
}
