// Package dogfood decides which abcd binary a hook runs in a checkout the
// person has opted in to dogfooding, and builds it (iss-2610070547057816).
//
// In abcd's own source checkout the installed release guards the session,
// so a hazard rule that landed on the default branch after the last release
// is not enforced there until the next one ships. The person opts a checkout
// in by listing it in ~/.abcd.noindex/dogfood-roots, one absolute path per
// line, "#" starting a comment; inside a listed checkout, and inside every
// worktree that checkout records as its own, the hooks run a build of the
// latest reviewed default branch, and everywhere else the installed release,
// unchanged.
//
// Three properties are load-bearing.
//
// The opt-in is the PERSON'S, read from their home and never from a
// repository. Recognising abcd's source by its module path was rejected: a
// repository posing as abcd would get its own code run by the hooks. The list
// is read through fsutil.HomeDeclarationNames, the one reader behind every
// "declare this checkout" opt-in (trusted-roots, local-transcript-roots,
// filter-roots), so it is honoured only as a regular file this account owns
// that no one else can write, within its size cap, reached through no
// symlinked folder; a list failing that is IGNORED out loud and the installed
// release runs everywhere.
//
// The build's SOURCE is the default branch's remote-tracking tip
// (refs/remotes/origin/<default>), never the checkout's HEAD or its working
// tree: the remote tip moves only through review and the merge queue, so no
// agent commit in any checkout changes the guard that judges that agent's
// next command. sourceCommit is the one function that decides it. Nothing
// here fetches; the tip is whatever the checkout last fetched, and a
// `git fetch` moves it.
//
// A failure never weakens the guard below the installed release: a build
// that is missing, running, failed, or not runnable leaves Resolve naming no
// binary, the hook runs the installed release, and a note says why on stderr.
// The PreToolUse shim adds the run-time half: a dogfood guard that answers
// exit 1, which is a warn or "NOT CHECKED", or that dies, hands the same
// command to the installed release, whose deny stands.
//
// The build is the reviewed commit or nothing: the export is checked blob by
// blob against the commit's tree (verifyExport), because `git archive`
// honours repository-local attributes anyone who can commit in the checkout
// can write. What this does not defend is the account itself: the store, like
// the plugin root the installed release lives in, is protected by being this
// uid's alone, and a process running as that uid can replace either binary or
// move refs/remotes/origin/* by hand. The rung is no weaker there than the
// installed release already is.
package dogfood

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

var (
	// RootsRelPath is the person's opt-in list, relative to their home.
	RootsRelPath = abcdhome.Rel("dogfood-roots")

	// RootsDisplay names the list in tilde form, so a message carries no home
	// path and a line a person pastes into a shell works.
	RootsDisplay = abcdhome.Display("dogfood-roots")

	// StoreDisplay names the build store in tilde form.
	StoreDisplay = abcdhome.Display("dogfood")

	// StoreRelPath is the machine-scoped build store, relative to the home:
	// dogfood/<root-sha>/<commit>/abcd. It is keyed on the repository's root
	// commit, as the worktree and transcript stores are, and on the commit the
	// build was made from, so a build is never shared between repositories and
	// a moved tip is a different directory.
	StoreRelPath = abcdhome.Rel("dogfood")
)

const (
	// BinaryName is the build's file name inside its commit directory.
	BinaryName = "abcd"

	// SourceRemote is the remote whose default branch a build is made from.
	SourceRemote = "origin"

	// maxRootsBytes caps the list read: a hand-kept list of checkouts is a
	// few hundred bytes.
	maxRootsBytes = 64 << 10

	// worktreeListCap bounds `git worktree list`.
	worktreeListCap = 1 << 20

	// keepBuilds is how many commit builds a repository keeps; older ones are
	// removed after each successful build.
	keepBuilds = 3

	// retryAfter is how long a failed build of one commit is left alone before
	// a hook starts it again. A tip that moved starts at once.
	retryAfter = 30 * time.Minute

	// buildTimeout bounds one export and build.
	buildTimeout = 15 * time.Minute

	lockName     = ".build.lock"
	failedSuffix = ".failed"
	logSuffix    = ".log"
	tmpPrefix    = ".build-"
)

// ErrBuildRunning is Build's answer when another build holds the store's lock.
var ErrBuildRunning = errors.New("another dogfood build is running")

// Checkout is a listed checkout a session runs in, and the commit its build
// is made from.
type Checkout struct {
	// Toplevel is the session's working tree.
	Toplevel string
	// Repo is the listed checkout that vouches for Toplevel: Toplevel itself,
	// or the main working tree that records Toplevel as one of its worktrees.
	// Every git read and the export run there.
	Repo    string
	RootSHA string
	// Ref is the remote-tracking ref the build is made from.
	Ref    string
	Commit string
}

// Short is the commit's abbreviation as a message shows it.
func (c Checkout) Short() string { return short(c.Commit) }

// LogDisplay is the build log's path in tilde form.
func (c Checkout) LogDisplay() string {
	return abcdhome.Display("dogfood", c.RootSHA, c.Commit+logSuffix)
}

// Plan is what a hook runs.
type Plan struct {
	// Binary is the build to run, "" for the installed release.
	Binary string
	// Checkout is set when the session's checkout is listed and its source
	// commit resolved.
	Checkout *Checkout
	// StartBuild asks the caller to start a build of Checkout in the
	// background; the installed release runs meanwhile.
	StartBuild bool
	// Notes are whole lines for stderr, each starting "abcd dogfood:".
	Notes []string
}

// caseFolding is the case-folding predicate the list match uses, a variable so
// a test can force either branch.
var caseFolding = fsutil.CaseFoldingFS

// now is the clock the retry window reads, a variable for tests.
var now = time.Now

// Resolve decides what a hook standing in cwd runs. home is the person's home
// directory. It writes nothing; a build it asks for is the caller's to start.
func Resolve(home, cwd string) Plan {
	if home == "" {
		return Plan{}
	}
	// No list is the ordinary case and costs one Lstat. The Lstat follows the
	// folders, so a list behind a symlinked ~/.abcd.noindex is found here and
	// refused by the reader below.
	if _, err := os.Lstat(filepath.Join(home, filepath.FromSlash(RootsRelPath))); err != nil {
		return Plan{}
	}
	c, note := listedCheckout(home, cwd)
	if c == nil {
		if note != "" {
			return Plan{Notes: []string{note}}
		}
		return Plan{}
	}
	ref, commit, err := sourceCommit(c.Repo)
	if err != nil {
		return Plan{Notes: []string{"abcd dogfood: " + fsutil.DisplayPath(c.Toplevel) + " is listed in " + RootsDisplay +
			", but " + err.Error() + "; the installed release runs"}}
	}
	root, err := gitutil.RootCommit(c.Repo)
	if err != nil || !gitutil.IsFullSHA(root) {
		return Plan{Notes: []string{"abcd dogfood: " + fsutil.DisplayPath(c.Toplevel) + " is listed in " + RootsDisplay +
			", but its root commit could not be read; the installed release runs"}}
	}
	c.Ref, c.Commit, c.RootSHA = ref, commit, root
	if bin, ok := ReadyBinary(home, root, commit); ok {
		return Plan{Binary: bin, Checkout: c}
	}
	what := gitutil.ShortRef(ref) + " at " + c.Short()
	logShown := c.LogDisplay()
	storeAbs := filepath.Join(home, filepath.FromSlash(path.Join(StoreRelPath, root)))
	if building(storeAbs) {
		return Plan{Checkout: c, Notes: []string{"abcd dogfood: the build of " + what +
			" is still running (log: " + logShown + "); the installed release runs until it is ready"}}
	}
	if fi, err := os.Lstat(filepath.Join(storeAbs, commit+failedSuffix)); err == nil && now().Sub(fi.ModTime()) < retryAfter {
		return Plan{Checkout: c, Notes: []string{"abcd dogfood: the build of " + what +
			" FAILED (log: " + logShown + "); the installed release runs instead, and the build is tried again after " +
			retryAfter.String() + " or when " + gitutil.ShortRef(ref) + " moves"}}
	}
	return Plan{Checkout: c, StartBuild: true, Notes: []string{"abcd dogfood: no build yet of " + what +
		"; building it in the background (log: " + logShown + "), and the installed release runs until it is ready"}}
}

// listedCheckout resolves the session's working tree and reports it when the
// person's list covers it: the tree itself is listed, or the main working tree
// of its repository is listed AND that main working tree records the session's
// tree among its own worktrees. The second half is read from the listed
// checkout, never from the session's tree, so a directory whose .git merely
// points into a listed repository is not covered by it. note is a refusal to
// say out loud, empty when the tree is simply not listed.
func listedCheckout(home, cwd string) (*Checkout, string) {
	top, err := gitutil.Toplevel(cwd)
	if err != nil {
		return nil, ""
	}
	declared := func(p string) (bool, string) {
		return fsutil.HomeDeclarationNames(home, RootsRelPath, maxRootsBytes, p, caseFolding())
	}
	on, why := declared(top)
	if why != "" {
		return nil, "abcd dogfood: IGNORED " + RootsDisplay + " — " + why + "; the installed release runs in every checkout"
	}
	repo := top
	if !on {
		wts, err := gitutil.ListWorktrees(top, worktreeListCap)
		if err != nil || len(wts) == 0 || wts[0].Bare || wts[0].Path == "" {
			return nil, ""
		}
		main := wts[0].Path
		if same(main, top) {
			return nil, ""
		}
		if on, _ = declared(main); !on {
			return nil, ""
		}
		repo = main
	}
	if !recordsWorktree(repo, top) {
		return nil, "abcd dogfood: REFUSED — " + fsutil.DisplayPath(top) + " points at the listed checkout " +
			fsutil.DisplayPath(repo) + " as its repository, but that checkout does not record it as one of its worktrees; the installed release runs"
	}
	return &Checkout{Toplevel: top, Repo: repo}, ""
}

// recordsWorktree reports whether the checkout at repo lists tree among its
// working trees, compared by file identity.
func recordsWorktree(repo, tree string) bool {
	if same(repo, tree) {
		return true
	}
	wts, err := gitutil.ListWorktrees(repo, worktreeListCap)
	if err != nil {
		return false
	}
	for _, w := range wts {
		if !w.Bare && w.Path != "" && same(w.Path, tree) {
			return true
		}
	}
	return false
}

func same(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	return err == nil && os.SameFile(fa, fb)
}

// sourceCommit is the ONE place that decides which commit a listed checkout's
// build is made from: the default branch's remote-tracking tip, resolved as
// the implement loop's landing resolves it (gitutil.DefaultRef, held to the
// SourceRemote remote). The tip moves only through review and the merge queue,
// so neither a commit on HEAD nor anything uncommitted reaches the build; no
// local branch is a fallback, because a local branch is what an agent commits
// to. Nothing is fetched: a hook never touches the network.
func sourceCommit(repo string) (ref, commit string, err error) {
	ref = gitutil.DefaultRef(repo)
	name, ok := strings.CutPrefix(ref, "refs/remotes/"+SourceRemote+"/")
	if !ok || name == "" || name == "HEAD" {
		return "", "", fmt.Errorf("it has no default branch on the remote %s to build from (run `git fetch %s` in it)", SourceRemote, SourceRemote)
	}
	commit, err = gitutil.Run(repo, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(commit) {
		return "", "", fmt.Errorf("%s does not name a commit", gitutil.ShortRef(ref))
	}
	return ref, commit, nil
}

// ReadyBinary is the finished build of commit for the repository rooted at
// rootSHA, when there is one this account alone can change: every store level
// a real directory owned by this uid that no one else can write, and the
// binary a regular, executable file of the same kind.
func ReadyBinary(home, rootSHA, commit string) (string, bool) {
	if !gitutil.IsFullSHA(rootSHA) || !gitutil.IsFullSHA(commit) {
		return "", false
	}
	dir := path.Join(StoreRelPath, rootSHA, commit)
	r, err := fsutil.OpenHomeScope(home, dir)
	if err != nil {
		return "", false
	}
	defer r.Close()
	for _, level := range []string{StoreRelPath, path.Join(StoreRelPath, rootSHA), dir} {
		p := filepath.Join(home, filepath.FromSlash(level))
		fi, err := os.Lstat(p)
		if err != nil || !fi.IsDir() || fsutil.CallersAlone(p, fi) != nil {
			return "", false
		}
	}
	fi, err := r.Lstat(BinaryName)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o100 == 0 {
		return "", false
	}
	bin := filepath.Join(home, filepath.FromSlash(dir), BinaryName)
	if fsutil.CallersAlone(bin, fi) != nil {
		return "", false
	}
	return bin, true
}

// building reports whether a build holds the store's lock.
func building(storeAbs string) bool {
	if _, err := os.Lstat(storeAbs); err != nil {
		return false
	}
	err := fsutil.WithFileLock(filepath.Join(storeAbs, lockName), 0, func() error { return nil })
	return errors.Is(err, fsutil.ErrLockContention)
}

// Build makes the build of c.Commit, unless one is already there, and returns
// its path. It holds the store's lock for the whole export and build, so two
// hooks firing at once cannot race it: the second gets ErrBuildRunning at
// once. The source is a clean export of the commit (`git archive`, checked
// against the commit's tree) into a fresh directory under the store, never the
// working tree, and the finished
// binary is moved into place by one rename, so a reader sees no build or a
// whole one. A failure leaves a marker beside the log that Resolve reads.
func Build(home string, c Checkout) (string, error) {
	if !gitutil.IsFullSHA(c.RootSHA) || !gitutil.IsFullSHA(c.Commit) {
		return "", errors.New("the checkout's root commit or source commit is not a full object name")
	}
	storeRel := path.Join(StoreRelPath, c.RootSHA)
	r, err := fsutil.EnsureHomeScope(home, storeRel, abcdhome.DirMode)
	if err != nil {
		return "", fmt.Errorf("preparing the build store: %w", err)
	}
	r.Close()
	storeAbs := filepath.Join(home, filepath.FromSlash(storeRel))
	var bin string
	err = fsutil.WithFileLock(filepath.Join(storeAbs, lockName), 0, func() error {
		if p, ok := ReadyBinary(home, c.RootSHA, c.Commit); ok {
			bin = p
			return nil
		}
		p, err := build(storeAbs, c)
		if err != nil {
			_ = os.WriteFile(filepath.Join(storeAbs, c.Commit+failedSuffix), []byte(err.Error()+"\n"), 0o600)
			return err
		}
		_ = os.Remove(filepath.Join(storeAbs, c.Commit+failedSuffix))
		prune(storeAbs, c.Commit)
		bin = p
		return nil
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return "", ErrBuildRunning
	}
	return bin, err
}

// build exports c.Commit and builds ./cmd/abcd from it, logging both steps.
func build(storeAbs string, c Checkout) (string, error) {
	logf, err := os.OpenFile(filepath.Join(storeAbs, c.Commit+logSuffix), os.O_CREATE|os.O_TRUNC|os.O_WRONLY|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return "", fmt.Errorf("opening the build log: %w", err)
	}
	defer logf.Close()
	fmt.Fprintf(logf, "abcd dogfood: building %s at %s from %s\n", gitutil.ShortRef(c.Ref), c.Commit, c.Repo)
	tmp, err := os.MkdirTemp(storeAbs, tmpPrefix)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	src, out := filepath.Join(tmp, "src"), filepath.Join(tmp, "out")
	for _, d := range []string{src, out} {
		if err := os.Mkdir(d, 0o700); err != nil {
			return "", err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
	if err := export(ctx, c.Repo, c.Commit, src, logf); err != nil {
		return "", fmt.Errorf("exporting %s: %w", short(c.Commit), err)
	}
	if err := verifyExport(c.Repo, c.Commit, src); err != nil {
		return "", fmt.Errorf("exporting %s: %w", short(c.Commit), err)
	}
	if err := goBuild(ctx, src, filepath.Join(out, BinaryName), logf); err != nil {
		return "", fmt.Errorf("building %s: %w", short(c.Commit), err)
	}
	final := filepath.Join(storeAbs, c.Commit)
	if err := os.Rename(out, final); err != nil {
		return "", fmt.Errorf("moving the build into place: %w", err)
	}
	fmt.Fprintf(logf, "abcd dogfood: built %s\n", filepath.Join(final, BinaryName))
	return filepath.Join(final, BinaryName), nil
}

// export writes the tree of commit, read from the repository at repo, into
// dst: `git archive` piped into `tar`, under the isolated git environment,
// with every content filter the repository configures blanked
// (gitutil.FilterOverrides), so no program the repository names runs.
func export(ctx context.Context, repo, commit, dst string, log io.Writer) error {
	filters, err := gitutil.FilterOverrides(repo)
	if err != nil {
		return fmt.Errorf("reading the repository's content filters: %w", err)
	}
	args := append(append(gitutil.ExecPins(), filters...), "-C", repo, "archive", "--format=tar", commit)
	git := exec.CommandContext(ctx, "git", args...)
	git.Env = gitutil.IsolatedEnv()
	git.Stderr = log
	tar := exec.CommandContext(ctx, "tar", "-x", "-f", "-", "-C", dst)
	tar.Stdout, tar.Stderr = log, log
	pipe, err := git.StdoutPipe()
	if err != nil {
		return err
	}
	tar.Stdin = pipe
	if err := git.Start(); err != nil {
		return err
	}
	terr := tar.Run()
	gerr := git.Wait()
	if gerr != nil {
		return fmt.Errorf("git archive: %w", gerr)
	}
	if terr != nil {
		return fmt.Errorf("tar: %w", terr)
	}
	return nil
}

// maxTreeListing caps `git ls-tree -r` of the commit being verified.
const maxTreeListing = 64 << 20

// verifyExport proves the export in dst is the tree of commit and nothing
// else: every blob of the commit is present with the content the commit
// records (its object id recomputed from the exported bytes, a symlink's from
// its target), and nothing is present that the commit does not hold. `git
// archive` honours attributes, and repository-local ones
// (.git/info/attributes, core.attributesFile) are writable by anyone who can
// commit in the checkout: export-ignore drops a file, export-subst and eol
// rewrite one. The reviewed commit is what is built, or nothing is.
func verifyExport(repo, commit, dst string) error {
	newHash, ok := objectHash(commit)
	if !ok {
		return errors.New("the commit is not a full object name")
	}
	out, err := gitutil.RunCapped(repo, maxTreeListing, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return fmt.Errorf("listing the commit's tree: %w", err)
	}
	type entry struct{ mode, oid string }
	want := map[string]entry{}
	for _, rec := range strings.Split(out, "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 {
			return fmt.Errorf("unreadable tree entry %q", rec)
		}
		if f[1] != "blob" {
			continue // a submodule's commit: archive writes no content for it
		}
		want[p] = entry{mode: f[0], oid: f[2]}
	}
	seen := 0
	err = filepath.WalkDir(dst, func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dst, abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		e, ok := want[rel]
		if !ok {
			return fmt.Errorf("the export holds %s, which the commit does not", rel)
		}
		var body []byte
		if e.mode == "120000" {
			if d.Type()&os.ModeSymlink == 0 {
				return fmt.Errorf("the export's %s is not the symlink the commit records", rel)
			}
			target, err := os.Readlink(abs)
			if err != nil {
				return err
			}
			body = []byte(target)
		} else {
			if !d.Type().IsRegular() {
				return fmt.Errorf("the export's %s is not the regular file the commit records", rel)
			}
			if body, err = os.ReadFile(abs); err != nil {
				return err
			}
		}
		h := newHash()
		fmt.Fprintf(h, "blob %d\x00", len(body))
		h.Write(body)
		if hex.EncodeToString(h.Sum(nil)) != e.oid {
			return fmt.Errorf("the export's %s differs from the commit (a repository attribute or filter changed it)", rel)
		}
		seen++
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(want) {
		var missing []string
		for p := range want {
			if _, err := os.Lstat(filepath.Join(dst, filepath.FromSlash(p))); err != nil {
				missing = append(missing, p)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("the export is missing %s, which the commit holds (a repository attribute dropped it)", strings.Join(missing, ", "))
	}
	return nil
}

// objectHash is the hash git names objects with in a repository whose
// object names are as long as name: SHA-1 for 40 hex digits, SHA-256 for 64.
func objectHash(name string) (func() hash.Hash, bool) {
	switch len(name) {
	case 40:
		return sha1.New, true
	case 64:
		return sha256.New, true
	}
	return nil, false
}

// goModVersionRe reads the `go <version>` line of go.mod, as the Makefile's
// GO_TOOLCHAIN_VERSION and scripts/pinned-toolchain.sh's caller do.
var goModVersionRe = regexp.MustCompile(`(?m)^go ([0-9][0-9.]*)\s*$`)

// goBuild builds ./cmd/abcd in src into dst with the toolchain src's go.mod
// declares (GOTOOLCHAIN=go<version>, the pin `make preflight` builds with),
// path-clean, with no VCS stamp (an export carries no .git, and a home that
// is itself a repository must not be stamped), and with the ambient
// workspace and flags neutralised so the build is the commit's alone.
func goBuild(ctx context.Context, src, dst string, log io.Writer) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return errors.New("no go toolchain is on PATH")
	}
	env := make([]string, 0, len(os.Environ())+4)
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch k {
		case "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "GOOS", "GOARCH":
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "GOWORK=off", "GOFLAGS=-mod=readonly")
	if mod, err := os.ReadFile(filepath.Join(src, "go.mod")); err == nil {
		if m := goModVersionRe.FindSubmatch(mod); m != nil {
			env = append(env, "GOTOOLCHAIN=go"+string(m[1]))
		}
	}
	cmd := exec.CommandContext(ctx, goBin, "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w", "-o", dst, "./cmd/abcd")
	cmd.Dir = src
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

// prune keeps the keepBuilds most recent commit builds in the store and
// removes the rest with their logs and markers, and any export left behind by
// a build that was killed. keep is never removed.
func prune(storeAbs, keep string) {
	entries, err := os.ReadDir(storeAbs)
	if err != nil {
		return
	}
	type build struct {
		name string
		mod  time.Time
	}
	var builds []build
	for _, e := range entries {
		name := e.Name()
		fi, err := e.Info()
		if err != nil {
			continue
		}
		switch {
		case e.IsDir() && gitutil.IsFullSHA(name):
			builds = append(builds, build{name, fi.ModTime()})
		case e.IsDir() && strings.HasPrefix(name, tmpPrefix) && now().Sub(fi.ModTime()) > buildTimeout:
			_ = os.RemoveAll(filepath.Join(storeAbs, name))
		}
	}
	sort.Slice(builds, func(i, j int) bool { return builds[i].mod.After(builds[j].mod) })
	kept := 0
	for _, b := range builds {
		if b.name == keep || kept < keepBuilds-1 {
			if b.name != keep {
				kept++
			}
			continue
		}
		_ = os.RemoveAll(filepath.Join(storeAbs, b.name))
		_ = os.Remove(filepath.Join(storeAbs, b.name+logSuffix))
		_ = os.Remove(filepath.Join(storeAbs, b.name+failedSuffix))
	}
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
