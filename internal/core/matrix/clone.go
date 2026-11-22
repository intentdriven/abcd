package matrix

// clone.go makes each copy's isolated clone of the frozen commit (decision 4,
// spc-2610161949559050 step 3): one clone per copy in the machine-scoped
// store, outside every working tree, holding no local-tier file and no other
// copy's refs. A copy never sees the person's checkout, another copy, or
// anything past the frozen commit, so what it makes is its model's work from
// the one starting point every copy shares.
//
// Every copy, of a code, a writing or a check task alike, is made under an
// alias minted at random when it starts (makeCopyClone): its clone is
// <store>/<run>/<alias>, so the folder it works in and the folder above it name
// the run and aliases and nothing of any copy's task, cell, route or model.
// Which alias is which copy is recorded in the run's state in its lab, outside
// the clone store, and a copy that reads that state breaks own_copy_only
// (compliance.go).

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/procgroup"
)

// cloneStore is where a repository's matrix copies are cloned: the account's
// abcd home, under matrix/<root-sha>/, beside the worktree store and like it
// outside every checkout (adr-2610031751065746).
func cloneStore(home, rootSHA string) string { return abcdhome.Path(home, "matrix", rootSHA) }

// cloneDir is the clone of run's copy made under alias, in store: the run's id
// and the alias, and nothing of the copy's task or cell.
func cloneDir(store, run, alias string) string { return filepath.Join(store, run, alias) }

// copyAlias is a copy's alias as the run's state records it: the one name its
// clone is made under, and so the one name the copy is shown of which copy it
// is (dcn-2611210903322839). It is minted at random when the copy starts and
// never derived from the copy's key, route, runner or model, so a copy echoing
// it tells its judges nothing.
type copyAlias struct {
	SchemaVersion int    `json:"schema_version"`
	Alias         string `json:"alias"`
}

// copyAliasSchema is the alias record's schema version.
const copyAliasSchema = 1

// aliasRe is an alias: 128 random bits, under a word no task's or cell's id
// can be (an id holds no underscore).
var aliasRe = regexp.MustCompile(`^copy_[0-9a-f]{32}$`)

// aliasRel is copy key's alias record in its run's directory, in the lab.
func aliasRel(run string, key TrialKey) string {
	return run + "/aliases/" + key.Task + "/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".json"
}

// makeCopyClone makes copy key's clone of frozen under the copy's alias. The
// alias is minted and recorded in the run's state before the clone is made,
// or read back where an earlier attempt recorded it, so a clone that was not
// made is made again under the alias its copy was given, and one that is there
// is refused. The run's record must be written first.
func makeCopyClone(labID, repoRoot, store, run, frozen string, key TrialKey) (string, error) {
	if err := cloneArgs(repoRoot, run, frozen, key); err != nil {
		return "", err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return "", err
	}
	defer proc.Close()
	if _, err := proc.ReadState(run + "/run.json"); err != nil {
		return "", fmt.Errorf("copy %s is not made: its run's record is not written: %w", key, err)
	}
	alias, err := readAlias(proc, run, key)
	if errors.Is(err, fs.ErrNotExist) {
		alias, err = mintAlias(proc, run, key)
	}
	if err != nil {
		return "", err
	}
	return makeClone(repoRoot, store, run, frozen, key, alias)
}

// mintAlias mints copy key's alias and records it, once.
func mintAlias(proc *lab.Procedure, run string, key TrialKey) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("copy %s's alias could not be minted: %w", key, err)
	}
	alias := "copy_" + hex.EncodeToString(b[:])
	data, err := json.Marshal(copyAlias{SchemaVersion: copyAliasSchema, Alias: alias})
	if err != nil {
		return "", err
	}
	if err := proc.WriteState(aliasRel(run, key), append(data, '\n')); err != nil {
		return "", fmt.Errorf("copy %s's alias is not recorded: %w", key, err)
	}
	return alias, nil
}

// readAlias is copy key's alias as its run's state records it; an error
// wrapping fs.ErrNotExist where none is recorded.
func readAlias(proc *lab.Procedure, run string, key TrialKey) (string, error) {
	raw, err := proc.ReadState(aliasRel(run, key))
	if err != nil {
		return "", err
	}
	var a copyAlias
	if err := jsonstrict.Decode(raw, &a); err != nil || a.SchemaVersion != copyAliasSchema || !aliasRe.MatchString(a.Alias) {
		return "", fmt.Errorf("copy %s's alias record does not read as one", key)
	}
	return a.Alias, nil
}

// localTier is the checkout's local tier, which no clone holds.
const localTier = ".abcd/.work.local"

// makeClone makes copy key's clone of frozen, fetched from the repository at
// repoRoot, an absolute path, at cloneDir(store, run, alias), and returns its
// path; makeCopyClone mints and records the alias. The clone is a
// fresh repository holding the frozen commit alone, fetched one commit deep,
// so no commit before it, nor anything only such a commit holds, is in the
// clone: what a copy can read there is the frozen tree, which is what the
// launch bundle's scan reads (admitClone). It is checked out at the frozen
// commit on a detached HEAD: no branch, no remote, no tag, no other copy's
// refs, no hooks and no objects borrowed from the source. A store inside any working
// tree (the source's, one of its worktrees, any other repository's) is refused
// before anything is made, as is a key whose clone already exists, so a copy
// never starts from another attempt's leftovers. A frozen commit that tracks a
// local-tier file is refused and its clone removed. A failure past the clone's
// directory removes it.
func makeClone(repoRoot, store, run, frozen string, key TrialKey, alias string) (string, error) {
	if err := cloneArgs(repoRoot, run, frozen, key); err != nil {
		return "", err
	}
	if !aliasRe.MatchString(alias) {
		return "", fmt.Errorf("copy %s's alias %q is not one (copy_ and 32 hex digits)", key, clip(alias))
	}
	return makeCloneAt(repoRoot, store, cloneDir(store, run, alias), frozen, "copy "+key.String())
}

// cloneArgs refuses what no clone is made from: a run id, a key, a frozen
// commit or a source that is not one.
func cloneArgs(repoRoot, run, frozen string, key TrialKey) error {
	if !runIDRe.MatchString(run) {
		return fmt.Errorf("%q is not a run id", clip(run))
	}
	if err := key.validate(); err != nil {
		return err
	}
	if !gitutil.IsFullSHA(frozen) {
		return fmt.Errorf("%q is not a full commit id, so it is no frozen commit", clip(frozen))
	}
	if !filepath.IsAbs(repoRoot) {
		// The source is handed to git fetch by path: an absolute one can never
		// be read as an option.
		return fmt.Errorf("the clone's source %q is not an absolute path", clip(repoRoot))
	}
	return nil
}

// makeCloneAt is makeClone at dir, a directory under store, for the copy
// named what, its arguments already checked (cloneArgs).
func makeCloneAt(repoRoot, store, dir, frozen, what string) (string, error) {
	if !filepath.IsAbs(store) {
		return "", fmt.Errorf("the clone store %q is not an absolute path", clip(store))
	}
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
			return "", fmt.Errorf("%s already has a clone; a copy is made once, from the frozen commit", what)
		}
		return "", fmt.Errorf("%s's clone could not be made: %w", what, err)
	}
	if err := fillClone(repoRoot, dir, frozen); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("%s's clone was not made: %w", what, err)
	}
	return dir, nil
}

// fillClone makes dir a repository at frozen, fetched from repoRoot.
func fillClone(repoRoot, dir, frozen string) error {
	steps := [][]string{
		// An empty template: no hooks, and nothing of the person's.
		{"init", "-q", "--template=", dir},
		// One commit deep: the frozen commit and its tree, and none of its
		// ancestors.
		{"-C", dir, "fetch", "-q", "--depth=1", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", repoRoot, frozen},
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
		// The checkout is named only for the refusal: its toplevel is asked
		// through the one shape-checked question, and an unshaped answer
		// names the directory itself.
		top, terr := gitutil.Toplevel(real)
		if terr != nil || top == "" {
			top = real
		}
		return fmt.Errorf("a copy's clone is never made inside a working tree or a repository, and %s is inside %s; "+
			"the clone store lives outside every checkout", dir, top)
	}
	return nil
}

// gitBound bounds one git command a clone is made with: a fetch of the frozen
// commit and its checkout scale with the tree, so the bound is there only so a
// git that waits never holds the run forever.
const gitBound = 30 * time.Minute

// git runs git isolated from the person's configuration and from any
// repository-selecting variable, returning its trimmed output.
func git(args ...string) (string, error) {
	out, err := gitRaw(args...)
	return strings.TrimSpace(string(out)), err
}

// gitRaw is git returning its output as git wrote it, for a NUL-separated
// listing whose names trimming would change.
func gitRaw(args ...string) ([]byte, error) { return gitRawWithin(gitBound, args...) }

// gitRawWithin is gitRaw bounded by limit: past it git's process group is
// killed, its output read to its end before the wait (procgroup.RunToEOF), and
// the call refused, naming the bound, never answered with what git wrote
// before it stopped.
func gitRawWithin(limit time.Duration, args ...string) ([]byte, error) {
	return gitRawCtx(context.Background(), limit, args...)
}

// gitRawCtx is gitRawWithin under parent too: git is killed when parent ends
// or past limit, whichever comes first, and the refusal names which.
func gitRawCtx(parent context.Context, limit time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = gitutil.IsolatedEnv()
	// git leads its own group and the bound kills the whole of it, so a
	// helper git started (a remote helper, a pager) ends with it.
	procgroup.Bound(cmd, time.Second)
	var stdout, stderr bytes.Buffer
	err := procgroup.RunToEOF(ctx, cmd, &stdout, &stderr)
	if parent.Err() != nil {
		return nil, fmt.Errorf("git %s was stopped at the deadline of the work it was part of", args[0])
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("git %s did not finish within %s, so it was stopped", args[0], limit)
	}
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("git %s: %w: %s", args[0], err, clip(msg))
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return stdout.Bytes(), nil
}
