package matrix

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// Each copy is made in its own clone of the frozen commit (spc-2610161949559050
// step 3, the technical criterion on isolation): outside every working tree,
// holding no local-tier file and no other copy's refs.

const cloneRun = "mx-261108070000-ab12"

// cloneSource is a repository with a frozen commit, a later commit on another
// branch, and a local tier holding a file, as the person's checkout would.
func cloneSource(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("README.md", "frozen\n")
	r.Write(".gitignore", ".abcd/.work.local/\n")
	r.Commit("frozen")
	frozen := r.Git("rev-parse", "HEAD")
	r.Write(".abcd/.work.local/NEXT.md", "a handover no copy may read\n")
	r.Git("checkout", "-q", "-b", "other-copy")
	r.Write("other.txt", "another copy's work\n")
	r.Commit("other copy")
	r.Git("checkout", "-q", "-")
	return r, frozen
}

func gitOut(t *testing.T, env []string, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestACopyIsAnIsolatedCloneOfTheFrozenCommit: the clone is under the store,
// keyed by task, cell and repeat, at the frozen commit with nothing but a
// detached HEAD: no branch, no remote, no other copy's refs, no objects shared
// with the source, and no local-tier file.
func TestACopyIsAnIsolatedCloneOfTheFrozenCommit(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 2}

	dir, err := makeClone(src.Root(), store, cloneRun, frozen, key)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(store, cloneRun, "widget", "cheap", "2"); dir != want {
		t.Fatalf("the clone is keyed by run, task, cell and repeat: %s, want %s", dir, want)
	}
	env := src.Env()
	if head := gitOut(t, env, dir, "rev-parse", "HEAD"); head != frozen {
		t.Fatalf("the clone is at the frozen commit: %s, want %s", head, frozen)
	}
	if refs := gitOut(t, env, dir, "for-each-ref"); refs != "" {
		t.Fatalf("the clone holds no ref, another copy's or the source's: %q", refs)
	}
	if remotes := gitOut(t, env, dir, "remote"); remotes != "" {
		t.Fatalf("the clone names no remote: %q", remotes)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "objects", "info", "alternates")); !os.IsNotExist(err) {
		t.Fatalf("the clone borrows no objects from the source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".abcd", ".work.local")); !os.IsNotExist(err) {
		t.Fatalf("the clone holds no local tier: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "other.txt")); !os.IsNotExist(err) {
		t.Fatalf("the clone holds nothing past the frozen commit: %v", err)
	}
	probe := exec.Command("git", "-C", dir, "cat-file", "-e", src.Git("rev-parse", "other-copy"))
	probe.Env = env
	if probe.Run() == nil {
		t.Fatal("the clone holds another branch's commit")
	}
	if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err != nil || string(data) != "frozen\n" {
		t.Fatalf("the clone checks out the frozen tree: %q %v", data, err)
	}
}

// TestACloneHoldsTheFrozenCommitAlone: the clone holds the frozen commit and
// its tree, and nothing of the commits before it, so what a copy can read in
// its clone is what the launch bundle's scan reads in its working tree.
func TestACloneHoldsTheFrozenCommitAlone(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("earlier.txt", "content only an earlier commit holds\n")
	r.Commit("earlier")
	parent := r.Git("rev-parse", "HEAD")
	blob := r.Git("rev-parse", "HEAD:earlier.txt")
	r.Git("rm", "-q", "earlier.txt")
	r.Write("README.md", "frozen\n")
	r.Commit("frozen")
	frozen := r.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")

	dir, err := makeClone(r.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "cheap", Repeat: 1})
	if err != nil {
		t.Fatal(err)
	}
	env := r.Env()
	for name, obj := range map[string]string{"the frozen commit's parent": parent, "a file only an earlier commit holds": blob} {
		probe := exec.Command("git", "-C", dir, "cat-file", "-e", obj)
		probe.Env = env
		if probe.Run() == nil {
			t.Fatalf("the clone holds %s (%s)", name, obj)
		}
	}
	if head := gitOut(t, env, dir, "rev-parse", "HEAD"); head != frozen {
		t.Fatalf("the clone is at the frozen commit: %s, want %s", head, frozen)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err != nil || string(data) != "frozen\n" {
		t.Fatalf("the clone checks out the frozen tree: %q %v", data, err)
	}
}

// TestACopyIsNeverMadeInsideAWorkingTree: a store inside the source's working
// tree, inside one of its linked worktrees, or inside any other repository's
// working tree is refused, and nothing is made there.
func TestACopyIsNeverMadeInsideAWorkingTree(t *testing.T) {
	src, frozen := cloneSource(t)
	linked := filepath.Join(t.TempDir(), "linked")
	src.Git("worktree", "add", "-q", "--detach", linked, frozen)
	other := gittest.NewRepo(t)
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	for name, store := range map[string]string{
		"the source":         filepath.Join(src.Root(), "store"),
		"a linked worktree":  filepath.Join(linked, "deep", "store"),
		"another repository": filepath.Join(other.Root(), "store"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := makeClone(src.Root(), store, cloneRun, frozen, key); err == nil ||
				!strings.Contains(err.Error(), "working tree") {
				t.Fatalf("a store inside %s is refused naming the working tree: %v", name, err)
			}
			if _, err := os.Stat(store); !os.IsNotExist(err) {
				t.Fatalf("nothing is made inside %s: %v", name, err)
			}
		})
	}
}

// TestACopyIsMadeOnce: a key whose clone already exists is refused, so a copy
// never starts from another attempt's leftovers.
func TestACopyIsMadeOnce(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	if _, err := makeClone(src.Root(), store, cloneRun, frozen, key); err != nil {
		t.Fatal(err)
	}
	if _, err := makeClone(src.Root(), store, cloneRun, frozen, key); err == nil || !strings.Contains(err.Error(), "already") {
		t.Fatalf("a second clone of one key is refused: %v", err)
	}
}

// TestACloneRefusesABadKeyOrCommit: the run, the key and the commit are
// checked before anything is made, so none can steer the path or the fetch.
func TestACloneRefusesABadKeyOrCommit(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	for name, c := range map[string]struct {
		run, commit string
		key         TrialKey
	}{
		"a run id":       {"../up", frozen, TrialKey{Task: "w", Cell: "c", Repeat: 1}},
		"a task":         {cloneRun, frozen, TrialKey{Task: "../w", Cell: "c", Repeat: 1}},
		"a short commit": {cloneRun, frozen[:12], TrialKey{Task: "w", Cell: "c", Repeat: 1}},
		"an option":      {cloneRun, "--upload-pack=x", TrialKey{Task: "w", Cell: "c", Repeat: 1}},
	} {
		if _, err := makeClone(src.Root(), store, c.run, c.commit, c.key); err == nil {
			t.Fatalf("%s that is not one is refused", name)
		}
	}
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Fatalf("nothing is made: %v", err)
	}
}

// TestACloneTakesItsSourceByAbsolutePath: the source repository is handed to
// git fetch by path, so one that is not absolute is refused before any git
// call, and an option-shaped one can never reach git as an option.
func TestACloneTakesItsSourceByAbsolutePath(t *testing.T) {
	_, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	for _, source := range []string{"src", "./src", "--upload-pack=x", ""} {
		_, err := makeClone(source, store, cloneRun, frozen, key)
		if err == nil || !strings.Contains(err.Error(), "source") || !strings.Contains(err.Error(), "absolute") {
			t.Fatalf("a source of %q is refused as not absolute: %v", source, err)
		}
	}
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Fatalf("nothing is made before the source is refused: %v", err)
	}
}

// TestAFrozenCommitTrackingALocalTierFileIsRefused: a frozen commit that
// tracks a file under the local tier would hand every copy that file, so the
// clone is refused and removed.
func TestAFrozenCommitTrackingALocalTierFileIsRefused(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".abcd/.work.local/NEXT.md", "tracked by mistake\n")
	r.Commit("tracked local tier")
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	dir, err := makeClone(r.Root(), store, cloneRun, r.Git("rev-parse", "HEAD"), key)
	if err == nil || !strings.Contains(err.Error(), "local tier") {
		t.Fatalf("a clone holding a local-tier file is refused: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(store, cloneRun, "widget", "cheap", "1")); !os.IsNotExist(serr) {
		t.Fatalf("the refused clone is removed: %v (%s)", serr, dir)
	}
}
