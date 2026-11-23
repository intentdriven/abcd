package matrix

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Each copy is made in its own clone of the frozen commit (spc-2610161949559050
// step 3, the technical criterion on isolation): outside every working tree,
// holding no local-tier file and no other copy's refs.

const cloneRun = "mx-261108070000-ab12"

// keyAlias is the alias a test makes copy key's clone under where no lab
// records one (makeCopyClone mints it at random): fixed per key, and spelling
// nothing of it.
func keyAlias(key TrialKey) string {
	sum := sha256.Sum256([]byte(key.String()))
	return "copy_" + hex.EncodeToString(sum[:16])
}

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
// at the run and the copy's alias, at the frozen commit with nothing but a
// detached HEAD: no branch, no remote, no other copy's refs, no objects shared
// with the source, and no local-tier file.
func TestACopyIsAnIsolatedCloneOfTheFrozenCommit(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 2}

	dir, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(store, cloneRun, keyAlias(key)); dir != want {
		t.Fatalf("the clone is made under the run and the copy's alias: %s, want %s", dir, want)
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

	dir, err := makeClone(r.Root(), store, cloneRun, frozen, TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}, keyAlias(TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}))
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
			if _, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key)); err == nil ||
				!strings.Contains(err.Error(), "working tree") {
				t.Fatalf("a store inside %s is refused naming the working tree: %v", name, err)
			}
			if _, err := os.Stat(store); !os.IsNotExist(err) {
				t.Fatalf("nothing is made inside %s: %v", name, err)
			}
		})
	}
}

// TestACopyIsNeverMadeWhereGitGivesNoAnswer: a store inside a repository git
// will not answer for (here one of a repository format it does not read) is
// refused, for a clone and for hidden tests alike, and nothing is made there:
// only git's own answer that no repository holds the store lets either be
// made.
func TestACopyIsNeverMadeWhereGitGivesNoAnswer(t *testing.T) {
	src, frozen := cloneSource(t)
	unread := gittest.NewRepo(t)
	unread.Git("config", "core.repositoryformatversion", "99")
	store := filepath.Join(unread.Root(), "deep", "store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	if _, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key)); err == nil ||
		!strings.Contains(err.Error(), "could not be told") {
		t.Fatalf("a store git gives no answer for is refused, saying so: %v", err)
	}
	if _, err := placeHiddenTests(store, cloneRun, "widget", widgetHidden); err == nil ||
		!strings.Contains(err.Error(), "could not be told") {
		t.Fatalf("hidden tests where git gives no answer are refused, saying so: %v", err)
	}
	if _, err := os.Stat(filepath.Join(unread.Root(), "deep")); !os.IsNotExist(err) {
		t.Fatalf("nothing is made where git gives no answer: %v", err)
	}
}

// TestACloneKeepsNoReflog: making the clone writes no reflog, so its git
// directory names no one who checked it out; the frozen commit is the only
// commit it holds.
func TestACloneKeepsNoReflog(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	dir, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "logs")); !os.IsNotExist(err) {
		t.Fatalf("the clone keeps a reflog: %v", err)
	}
	if v := gitOut(t, src.Env(), dir, "config", "--get", "core.logAllRefUpdates"); v != "false" {
		t.Fatalf("the clone writes no reflog later either: core.logAllRefUpdates is %q", v)
	}
}

// TestACopyIsMadeOnce: a key whose clone already exists is refused, so a copy
// never starts from another attempt's leftovers.
func TestACopyIsMadeOnce(t *testing.T) {
	src, frozen := cloneSource(t)
	store := filepath.Join(t.TempDir(), "matrix-store")
	key := TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}
	if _, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key)); err != nil {
		t.Fatal(err)
	}
	if _, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key)); err == nil || !strings.Contains(err.Error(), "already") {
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
		if _, err := makeClone(src.Root(), store, c.run, c.commit, c.key, keyAlias(c.key)); err == nil {
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
		_, err := makeClone(source, store, cloneRun, frozen, key, keyAlias(key))
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
	dir, err := makeClone(r.Root(), store, cloneRun, r.Git("rev-parse", "HEAD"), key, keyAlias(key))
	if err == nil || !strings.Contains(err.Error(), "local tier") {
		t.Fatalf("a clone holding a local-tier file is refused: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(store, cloneRun, "widget", "cheap", "1")); !os.IsNotExist(serr) {
		t.Fatalf("the refused clone is removed: %v (%s)", serr, dir)
	}
}

// TestACopysBriefNamesNoPathOnThisMachine: what a copy is asked names where
// it works only as the directory it starts in, never the clone's path, which
// carries the account's home and so its login name.
func TestACopysBriefNamesNoPathOnThisMachine(t *testing.T) {
	home := filepath.Join(string(filepath.Separator)+"Users", "pat-example")
	clone := filepath.Join(abcdhome.Path(home, "matrix"), strings.Repeat("ab", 20), cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "cheap", Repeat: 1}))
	for _, task := range []Task{
		{ID: "widget", Kind: KindCode, Intent: "itd-10"},
		{ID: "notes", Kind: KindWriting, Criteria: "Clear.", Source: []string{"docs/a.md"}},
		{ID: "audit", Kind: KindCheck, Material: []string{"docs/"}},
	} {
		brief := copyBrief(task)
		if strings.Contains(brief, "pat-example") || strings.Contains(brief, clone) || strings.Contains(brief, string(filepath.Separator)+"Users") {
			t.Errorf("%s: the brief names a path on this machine: %s", task.Kind, brief)
		}
		if !strings.Contains(brief, "current directory") {
			t.Errorf("%s: the brief says where the copy works: %s", task.Kind, brief)
		}
	}
}

// TestOnlyACloneTheMatrixMadeIsACopysClone: a directory at a clone's place in
// the store is a copy's clone only when it is a repository of its own (its
// .git a directory, not a linked worktree's file), shallow at exactly the
// frozen commit, borrowing no objects, and holding no ref but its lanes'
// branches. A linked worktree of the source, a full clone of it, a clone at
// another commit, one borrowing the source's objects and one holding another
// ref are refused.
func TestOnlyACloneTheMatrixMadeIsACopysClone(t *testing.T) {
	src, frozen := cloneSource(t)
	home := t.TempDir()
	store := cloneStore(home, gittest.RootCommit(t, src.Root()))
	n := 0
	made := func(t *testing.T) string {
		t.Helper()
		n++
		key := TrialKey{Task: "widget", Cell: "cheap", Repeat: n}
		dir, err := makeClone(src.Root(), store, cloneRun, frozen, key, keyAlias(key))
		if err != nil {
			t.Fatal(err)
		}
		return dir
	}
	place := func(t *testing.T) string {
		t.Helper()
		n++
		dir := cloneDir(store, cloneRun, keyAlias(TrialKey{Task: "widget", Cell: "cheap", Repeat: n}))
		if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	env := src.Env()
	ok := made(t)
	if err := CopyClone(home, ok, frozen, "build/", true); err != nil {
		t.Fatalf("the clone the matrix made is a copy's clone: %v", err)
	}
	gitOut(t, env, ok, "branch", "build/lane-1", frozen)
	if err := CopyClone(home, ok, frozen, "build/", true); err != nil {
		t.Fatalf("a clone holding its lanes' branches is a copy's clone: %v", err)
	}
	if err := CopyClone(home, ok, strings.Repeat("a", len(frozen)), "build/", true); err == nil {
		t.Fatal("a clone is a copy's clone of its own frozen commit only")
	}

	linked := place(t)
	gitOut(t, env, src.Root(), "worktree", "add", "-q", "--detach", linked, frozen)
	full := place(t)
	gitOut(t, env, filepath.Dir(full), "clone", "-q", "--no-checkout", src.Root(), full)
	gitOut(t, env, full, "-c", "advice.detachedHead=false", "checkout", "-q", "--detach", frozen)
	tagged := made(t)
	gitOut(t, env, tagged, "tag", "kept", frozen)
	borrowing := made(t)
	if err := os.MkdirAll(filepath.Join(borrowing, ".git", "objects", "info"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(borrowing, ".git", "objects", "info", "alternates"),
		[]byte(filepath.Join(src.Root(), ".git", "objects")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Built by hand to look like a made clone: every branch of the source
	// fetched, HEAD detached at the frozen commit, every ref deleted and the
	// shallow file written, so its objects still hold the source's history.
	history := place(t)
	if err := os.MkdirAll(history, 0o700); err != nil {
		t.Fatal(err)
	}
	gitOut(t, env, history, "init", "-q", "--template=")
	gitOut(t, env, history, "config", "core.logAllRefUpdates", "false")
	gitOut(t, env, history, "fetch", "-q", "--no-tags", src.Root(), "+refs/heads/*:refs/heads/*")
	gitOut(t, env, history, "-c", "advice.detachedHead=false", "checkout", "-q", "--detach", frozen)
	for _, ref := range strings.Fields(gitOut(t, env, history, "for-each-ref", "--format=%(refname)")) {
		gitOut(t, env, history, "update-ref", "-d", ref)
	}
	if err := os.WriteFile(filepath.Join(history, ".git", "shallow"), []byte(frozen+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A made clone whose object store is a link to the source's.
	linkedObjects := made(t)
	if err := os.RemoveAll(filepath.Join(linkedObjects, ".git", "objects")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(src.Root(), ".git", "objects"), filepath.Join(linkedObjects, ".git", "objects")); err != nil {
		t.Fatal(err)
	}
	// A made clone holding one object nothing in it reaches.
	stray := made(t)
	loose := filepath.Join(t.TempDir(), "loose.txt")
	if err := os.WriteFile(loose, []byte("content no commit of the clone holds\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitOut(t, env, stray, "hash-object", "-w", loose)
	// A made clone of a frozen commit with history behind it, whose .git
	// names a common directory: its own objects emptied and its shallow file
	// left as a decoy, while the common directory holds that whole history
	// fetched, its refs deleted.
	deep := gittest.NewRepo(t)
	deep.Write("earlier.txt", "content only an earlier commit holds\n")
	deep.Commit("earlier")
	deep.Write("README.md", "frozen\n")
	deep.Commit("frozen")
	deepFrozen := deep.Git("rev-parse", "HEAD")
	n++
	deepKey := TrialKey{Task: "widget", Cell: "cheap", Repeat: n}
	redirected, err := makeClone(deep.Root(), store, cloneRun, deepFrozen, deepKey, keyAlias(deepKey))
	if err != nil {
		t.Fatal(err)
	}
	common := filepath.Join(t.TempDir(), "common.git")
	gittest.InitBare(t, common)
	gitOut(t, env, common, "fetch", "-q", "--no-tags", deep.Root(), "+refs/heads/*:refs/heads/*")
	for _, ref := range strings.Fields(gitOut(t, env, common, "for-each-ref", "--format=%(refname)")) {
		gitOut(t, env, common, "update-ref", "-d", ref)
	}
	if err := os.RemoveAll(filepath.Join(redirected, ".git", "objects")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(redirected, ".git", "objects"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(redirected, ".git", "commondir"), []byte(common+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CopyClone(home, redirected, deepFrozen, "build/", true); !errors.Is(err, ErrNotACopyClone) {
		t.Errorf("a clone whose .git names a common directory holding the history is refused: %v", err)
	}
	// A made clone with a lane branch at a commit that does not descend
	// from the frozen commit.
	unrelated := made(t)
	orphan := gitOut(t, env, unrelated, "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture", "commit-tree", "-m", "unrelated", frozen+"^{tree}")
	gitOut(t, env, unrelated, "update-ref", "refs/heads/build/lane-1", orphan)
	for name, dir := range map[string]string{
		"a clone whose .git names a common directory":        redirected,
		"a lane branch not descended from the frozen commit": unrelated,
		"a linked worktree of the source":                    linked,
		"a full clone of the source":                         full,
		"a clone holding another ref":                        tagged,
		"a clone borrowing objects":                          borrowing,
		"a clone holding the source's history":               history,
		"a clone whose objects are the source's":             linkedObjects,
		"a clone holding an object nothing reaches":          stray,
	} {
		err := CopyClone(home, dir, frozen, "build/", true)
		if !errors.Is(err, ErrNotACopyClone) {
			t.Errorf("%s at a clone's place in the store is refused: %v", name, err)
		}
	}
}
