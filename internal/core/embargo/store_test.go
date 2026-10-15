package embargo

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// fixture is a checkout with one commit on main and a home of its own, apart
// from the checkout, the way a person's home and their project sit.
type fixture struct {
	repo *gittest.Repo
	home string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/.gitkeep", "")
	r.Write("README.md", "a project\n")
	r.Commit("initial")
	return fixture{repo: r, home: home}
}

// storeGit runs git in the store with the fixture's isolated environment.
func storeGit(t *testing.T, f fixture, store string, args ...string) (string, error) {
	t.Helper()
	full := append([]string{"-C", store, "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = f.repo.Env()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func mustRefuse(t *testing.T, err error, want string) {
	t.Helper()
	var r *RefusalError
	if !errors.As(err, &r) {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if !strings.Contains(r.Error(), want) {
		t.Fatalf("refusal %q does not say %q", r.Error(), want)
	}
}

// TestStoreCreatedDetachedOnce: the first openStore makes the store as a detached
// worktree at the default branch's tip, in the machine-scoped worktree store
// keyed on the repository's root commit; the second finds it and makes nothing.
func TestStoreCreatedDetachedOnce(t *testing.T) {
	f := newFixture(t)
	s, created, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("the first openStore did not report making the store")
	}
	rootSHA := f.repo.Git("rev-list", "--max-parents=0", "HEAD")
	want := abcdhome.Path(f.home, "worktrees", rootSHA, "embargo")
	if s.Root != want {
		t.Fatalf("store at %s, want %s", s.Root, want)
	}
	if _, err := storeGit(t, f, s.Root, "symbolic-ref", "-q", "HEAD"); err == nil {
		t.Fatal("the store's HEAD is on a branch")
	}
	head, err := storeGit(t, f, s.Root, "rev-parse", "HEAD")
	if err != nil || head != f.repo.Git("rev-parse", "main") {
		t.Fatalf("store HEAD %q (%v), want main's tip", head, err)
	}
	s2, created, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if created || s2.Root != s.Root {
		t.Fatalf("the second openStore made a store (%v) or moved it (%s)", created, s2.Root)
	}
	if n := strings.Count(f.repo.Git("worktree", "list", "--porcelain"), "worktree "); n != 2 {
		t.Fatalf("git lists %d worktrees, want the checkout and the store", n)
	}
}

// TestVerifyStoreRefusesBranchHead: a store someone put on a branch has a ref
// that can be pushed, so every sub-verb refuses it.
func TestVerifyStoreRefusesBranchHead(t *testing.T) {
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if out, err := storeGit(t, f, s.Root, "switch", "-c", "held"); err != nil {
		t.Fatalf("switch: %v %s", err, out)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "branch")
	if _, _, err := openStore(f.home, f.repo.Root()); err == nil {
		t.Fatal("openStore accepted a store on a branch")
	}
}

// TestVerifyStoreRefusesTrackedRecord: a private record the store's index
// tracks is one commit from the public history, so every sub-verb refuses.
func TestVerifyStoreRefusesTrackedRecord(t *testing.T) {
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	st := sampleState()
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(s.Root, ".abcd", "work", "issues", "open", "iss-2610120000000001-private-report.md")
	if err := os.WriteFile(rec, []byte("---\nid: iss-2610120000000001\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(); err != nil {
		t.Fatalf("an untracked private record was refused: %v", err)
	}
	if out, err := storeGit(t, f, s.Root, "add", "--", rec); err != nil {
		t.Fatalf("add: %v %s", err, out)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "tracks")
	if strings.Contains(err.Error(), "private-report") {
		t.Fatalf("the refusal names the record's file: %v", err)
	}
}

// TestVerifyStoreRefusesStoreInsideCheckout: a home inside the checkout would
// put the store, and its private records, inside the public working tree. openStore
// refuses before making anything.
func TestVerifyStoreRefusesStoreInsideCheckout(t *testing.T) {
	f := newFixture(t)
	_, _, err := openStore(f.repo.Root(), f.repo.Root())
	mustRefuse(t, err, "inside")
	if _, err := os.Lstat(abcdhome.Path(f.repo.Root())); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused openStore made something inside the checkout: %v", err)
	}
}

// TestVerifyStoreRefusesAPlainDirectory: a directory at the store's path that
// is not this repository's worktree is not the store, and git must not answer
// for it from whatever repository encloses it.
func TestVerifyStoreRefusesAPlainDirectory(t *testing.T) {
	f := newFixture(t)
	p, err := StorePath(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}
	_, _, err = openStore(f.home, f.repo.Root())
	mustRefuse(t, err, "worktree")
}

// TestStoreRootOwnerOnly: git makes a worktree at the umask's 0755, which lets
// every account on the machine list the store and read its private records;
// openStore closes the store's folder to everyone but this account.
func TestStoreRootOwnerOnly(t *testing.T) {
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Fatalf("the store's folder is %#o, want 0700", perm)
	}
}

// TestVerifyStoreRefusesRootOpenToOthers: a store's folder another account
// can read or enter is refused, and the refusal says how to close it.
func TestVerifyStoreRefusesRootOpenToOthers(t *testing.T) {
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(s.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "chmod 700")
	if _, _, err := openStore(f.home, f.repo.Root()); err == nil {
		t.Fatal("openStore accepted a store open to other accounts")
	}
}

// TestVerifyStoreRefusesRecordOpenToOthers: a private record the state names
// that another account could read is refused, without naming the record.
func TestVerifyStoreRefusesRecordOpenToOthers(t *testing.T) {
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveState(sampleState()); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.Root, ".abcd", "work", "issues", "wontfix")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(dir, "iss-2610120000000002-private-report.md")
	if err := os.WriteFile(rec, []byte("---\nid: iss-2610120000000002\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(); err != nil {
		t.Fatalf("an owner-only private record was refused: %v", err)
	}
	if err := os.Chmod(rec, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "go-rwx")
	if strings.Contains(err.Error(), "private-report") || strings.Contains(err.Error(), "iss-2610120000000002") {
		t.Fatalf("the refusal names the record: %v", err)
	}
}
