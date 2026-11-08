package matrix

// clone.go makes each copy's isolated clone of the frozen commit (decision 4,
// spc-2610161949559050 step 3): one clone per copy in the machine-scoped
// store, keyed by run, task, cell and repeat, outside every working tree,
// holding no local-tier file and no other copy's refs. A copy never sees the
// person's checkout, another copy, or anything past the frozen commit, so what
// it makes is its model's work from the one starting point every copy shares.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// cloneStore is where a repository's matrix copies are cloned: the account's
// abcd home, under matrix/<root-sha>/, beside the worktree store and like it
// outside every checkout (adr-2610031751065746).
func cloneStore(home, rootSHA string) string { return abcdhome.Path(home, "matrix", rootSHA) }

// cloneDir is copy key's clone of run under store.
func cloneDir(store, run string, key TrialKey) string {
	return filepath.Join(store, run, key.Task, key.Cell, strconv.Itoa(key.Repeat))
}

// localTier is the checkout's local tier, which no clone holds.
const localTier = ".abcd/.work.local"

// makeClone makes copy key's clone of frozen, fetched from the repository at
// repoRoot, at cloneDir(store, run, key), and returns its path. The clone is a
// fresh repository holding the frozen commit's history alone, checked out at
// it on a detached HEAD: no branch, no remote, no tag, no other copy's refs,
// no hooks and no objects borrowed from the source. A store inside any working
// tree (the source's, one of its worktrees, any other repository's) is refused
// before anything is made, as is a key whose clone already exists, so a copy
// never starts from another attempt's leftovers. A frozen commit that tracks a
// local-tier file is refused and its clone removed. A failure past the clone's
// directory removes it.
func makeClone(repoRoot, store, run, frozen string, key TrialKey) (string, error) {
	if !runIDRe.MatchString(run) {
		return "", fmt.Errorf("%q is not a run id", clip(run))
	}
	if err := key.validate(); err != nil {
		return "", err
	}
	if !gitutil.IsFullSHA(frozen) {
		return "", fmt.Errorf("%q is not a full commit id, so it is no frozen commit", clip(frozen))
	}
	if !filepath.IsAbs(store) {
		return "", fmt.Errorf("the clone store %q is not an absolute path", clip(store))
	}
	dir := cloneDir(store, run, key)
	// Asked of the directory the clone goes in: the clone itself, once made, is
	// a repository of its own, and one already there is refused below.
	if err := outsideWorkingTrees(filepath.Dir(dir)); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return "", fmt.Errorf("the clone store could not be made: %w", err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("copy %s already has a clone; a copy is made once, from the frozen commit", key)
		}
		return "", fmt.Errorf("copy %s's clone could not be made: %w", key, err)
	}
	if err := fillClone(repoRoot, dir, frozen); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("copy %s's clone was not made: %w", key, err)
	}
	return dir, nil
}

// fillClone makes dir a repository at frozen, fetched from repoRoot.
func fillClone(repoRoot, dir, frozen string) error {
	steps := [][]string{
		// An empty template: no hooks, and nothing of the person's.
		{"init", "-q", "--template=", dir},
		{"-C", dir, "fetch", "-q", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", repoRoot, frozen},
		{"-C", dir, "-c", "advice.detachedHead=false", "checkout", "-q", "--detach", frozen},
	}
	for _, args := range steps {
		if _, err := git(args...); err != nil {
			return err
		}
	}
	head, err := git("-C", dir, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != frozen {
		return fmt.Errorf("the clone is at %s, not the frozen commit %s", head, frozen)
	}
	if refs, err := git("-C", dir, "for-each-ref"); err != nil || refs != "" {
		return fmt.Errorf("the clone holds refs it should not: %q %v", clip(refs), err)
	}
	if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(localTier))); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("the frozen commit tracks the local tier (%s), which no copy may hold", localTier)
	}
	return nil
}

// outsideWorkingTrees refuses a clone path inside any repository: a working
// tree (the source's, a linked worktree of it, any other's) or a git
// directory. It asks git at the path's nearest existing directory, read
// through its symbolic links, so a path through a link into a checkout is
// refused as the checkout's.
func outsideWorkingTrees(dir string) error {
	at := dir
	for {
		if fi, err := os.Stat(at); err == nil && fi.IsDir() {
			break
		}
		up := filepath.Dir(at)
		if up == at {
			return fmt.Errorf("no directory above %s exists", dir)
		}
		at = up
	}
	real, err := filepath.EvalSymlinks(at)
	if err != nil {
		return err
	}
	out, err := git("-C", real, "rev-parse", "--is-inside-work-tree", "--is-inside-git-dir")
	if err != nil {
		// git names no repository here: the path is outside every one.
		return nil
	}
	if strings.Contains(out, "true") {
		top, _ := git("-C", real, "rev-parse", "--show-toplevel")
		if top == "" {
			top = real
		}
		return fmt.Errorf("a copy's clone is never made inside a working tree or a repository, and %s is inside %s; "+
			"the clone store lives outside every checkout", dir, top)
	}
	return nil
}

// git runs git isolated from the person's configuration and from any
// repository-selecting variable, returning its trimmed output.
func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Env = gitutil.IsolatedEnv()
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, clip(strings.TrimSpace(string(ee.Stderr))))
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}
