package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// The lazy-fetch floor (iss-2610090821527948). In a partial clone git answers
// a read of a MISSING object by fetching it, and the fetch runs the transport
// the repository's own config names (remote.<name>.uploadpack for a local URL,
// core.sshCommand for ssh://). gitEnv sets GIT_NO_LAZY_FETCH=1 so such a read
// is an error instead, but git honours the variable only from 2.44: an older
// git (Apple's Command Line Tools ship 2.39) ignores it and fetches. So below
// the floor an isolated command that reads objects is refused outright in a
// repository that declares a promisor remote, before git starts; a repository
// that declares none cannot lazy-fetch and reads as before.
const (
	lazyFetchFloorMajor = 2
	lazyFetchFloorMinor = 44
)

// ErrLazyFetchFloor is the refusal of an object read in a partial clone on a
// git too old to honour GIT_NO_LAZY_FETCH.
var ErrLazyFetchFloor = fmt.Errorf("the repository declares a promisor remote (a partial clone), and git older than %d.%d ignores GIT_NO_LAZY_FETCH, so reading a missing object would run the transport the repository configures: abcd reads no objects here until git is %d.%d or later",
	lazyFetchFloorMajor, lazyFetchFloorMinor, lazyFetchFloorMajor, lazyFetchFloorMinor)

// versionCache holds the `git version` answer, probed once per process. The
// source is a field so a test can inject a version on either side of the
// floor whatever git is installed.
type versionCache struct {
	mu     sync.Mutex
	src    func() (string, error)
	done   bool
	honour bool
	raw    string
}

var gitVersion = &versionCache{src: probeGitVersion}

// SwapGitVersionForTest replaces the `git version` probe the lazy-fetch floor
// reads, so a test in any package can put git on either side of the floor
// whatever is installed. It returns the restore, which puts the real probe
// back; the cached answer is cleared both ways.
func SwapGitVersionForTest(src func() (string, error)) (restore func()) {
	gitVersion.set(src)
	return func() { gitVersion.set(probeGitVersion) }
}

// set replaces the source and clears the cached answer.
func (v *versionCache) set(src func() (string, error)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.src, v.done, v.honour, v.raw = src, false, false, ""
}

// honoursNoLazyFetch reports whether the git on PATH honours GIT_NO_LAZY_FETCH,
// and the version line it read. A probe that fails, or a line that does not
// parse, is below the floor: the refusal it leads to touches only a
// repository that declares a promisor remote, so failing closed costs nothing
// elsewhere.
func (v *versionCache) honoursNoLazyFetch() (bool, string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.done {
		raw, err := v.src()
		v.raw = strings.TrimSpace(raw)
		v.honour = err == nil && versionAtLeast(v.raw, lazyFetchFloorMajor, lazyFetchFloorMinor)
		v.done = true
	}
	return v.honour, v.raw
}

// gitVersionRe reads major and minor from `git version 2.39.5 (Apple Git-154)`
// or `git version 2.45.1.windows.1`.
var gitVersionRe = regexp.MustCompile(`^git version (\d+)\.(\d+)`)

func versionAtLeast(line string, major, minor int) bool {
	m := gitVersionRe.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	ma, err1 := strconv.Atoi(m[1])
	mi, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return false
	}
	return ma > major || (ma == major && mi >= minor)
}

// probeGitVersion asks the git on PATH for its version under the isolated
// environment.
func probeGitVersion() (string, error) {
	cmd := exec.Command("git", "version")
	cmd.Env = gitEnv()
	w := &capWriter{remaining: 4096}
	cmd.Stdout = w
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return string(w.buf), nil
}

// lazyFetchGuard is the one check every isolated git command passes before it
// starts: nil when git honours GIT_NO_LAZY_FETCH, when the command reads no
// objects, or when the repository under root declares no promisor remote;
// ErrLazyFetchFloor otherwise. On a git at or past the floor it costs nothing
// after the first call.
//
// ls-files is index-only for the flags readsObjects admits, unless the index
// is sparse: with core.sparseCheckout or index.sparse enabled, git can expand
// a sparse index by reading tree objects, so in a partial clone below the
// floor EVERY ls-files is refused (coordinator ruling on
// iss-2610091935324732). The settings are read through the same isolated
// config view, and from any -c the command line carries.
func lazyFetchGuard(root string, args []string) error {
	if ok, _ := gitVersion.honoursNoLazyFetch(); ok {
		return nil
	}
	reads := readsObjects(args)
	listing := subcommand(args) == "ls-files"
	if !reads && !listing {
		return nil
	}
	cfg, err := readFloorConfig(root)
	if err != nil {
		return err
	}
	cmdPromisor, cmdSparse := commandLineFloorConfig(args)
	if !cfg.promisor && !cmdPromisor {
		return nil
	}
	_, raw := gitVersion.honoursNoLazyFetch()
	if reads {
		return fmt.Errorf("%w (git on PATH: %q)", ErrLazyFetchFloor, raw)
	}
	if cfg.sparse || cmdSparse {
		return fmt.Errorf("%w (git on PATH: %q; the repository enables a sparse checkout or sparse index, so ls-files can read tree objects to expand it)", ErrLazyFetchFloor, raw)
	}
	return nil
}

// subcommand is the git subcommand an isolated command line names, past any
// leading -c pairs; "" when there is none.
func subcommand(args []string) string {
	i := 0
	for i+1 < len(args) && args[i] == "-c" {
		i += 2
	}
	if i >= len(args) {
		return ""
	}
	return args[i]
}

// commandLineFloorConfig reads the floor's settings from the leading -c pairs
// of a command line: a promisor declaration (extensions.partialClone, or a
// remote.<name>.promisor git reads as true) and a sparse setting
// (core.sparseCheckout or index.sparse read as true). A -c key with no "=" is
// true, as git reads it.
func commandLineFloorConfig(args []string) (promisor, sparse bool) {
	for i := 0; i+1 < len(args) && args[i] == "-c"; i += 2 {
		key, val, hasVal := strings.Cut(args[i+1], "=")
		on := !hasVal || !isGitFalse(val)
		switch k := strings.ToLower(key); {
		case k == "extensions.partialclone":
			promisor = promisor || (hasVal && strings.TrimSpace(val) != "")
		case strings.HasPrefix(k, "remote.") && strings.HasSuffix(k, ".promisor"):
			promisor = promisor || on
		case k == "core.sparsecheckout", k == "index.sparse":
			sparse = sparse || on
		}
	}
	return promisor, sparse
}

// readsObjects reports whether an isolated command line can read an object,
// and so lazy-fetch one. The exemptions are the commands that read only the
// config, refs, the index or the filesystem, which root discovery and the
// worktree probes use; anything else is assumed to read objects, so a command
// added later is guarded until it is shown not to need it.
//
// Reading the ignore or attribute rules is NOT an index-only read: for a
// .gitignore or .gitattributes marked skip-worktree and missing from disk, git
// reads the blob the index names out of the object store
// (iss-2610091935324732). So check-ignore is exempt only with --no-index, and
// ls-files only when every flag is one of lsFilesIndexOnly and no pathspec
// carries attr magic: the exclude flags (-o, -i, --exclude-standard, ...),
// --with-tree, --eol, --format and -m all count as reading objects. config
// --blob reads its config out of a blob, so it counts too.
func readsObjects(args []string) bool {
	i := 0
	for i+1 < len(args) && args[i] == "-c" {
		i += 2
	}
	if i >= len(args) {
		return true
	}
	sub, rest := args[i], args[i+1:]
	switch sub {
	case "symbolic-ref":
		return false
	case "config":
		// --blob reads the config out of a blob, not a file.
		for _, a := range rest {
			if a == "--blob" || strings.HasPrefix(a, "--blob=") {
				return true
			}
		}
		return false
	case "check-ignore":
		for _, a := range rest {
			if a == "--" {
				break
			}
			if a == "--no-index" {
				return false
			}
		}
		return true
	case "worktree":
		return len(rest) == 0 || rest[0] != "list"
	case "ls-files":
		operands := false
		for _, a := range rest {
			switch {
			case operands || !strings.HasPrefix(a, "-"):
				if pathspecReadsAttributes(a) {
					return true
				}
			case a == "--":
				operands = true
			case !lsFilesIndexOnly[a]:
				return true
			}
		}
		return false
	case "rev-parse":
		// Flags alone (--show-toplevel, --git-dir, --is-inside-work-tree)
		// read no object; a revision operand does.
		for _, a := range rest {
			if !strings.HasPrefix(a, "--") || a == "--" {
				return true
			}
		}
		return false
	}
	return true
}

// lsFilesIndexOnly are the ls-files flags that list the index (or stat the
// files it names) without consulting the ignore rules or reading blob
// content. Any other flag, a combined short form like -oi included, counts as
// reading objects.
var lsFilesIndexOnly = map[string]bool{
	"-z": true, "-c": true, "--cached": true, "-s": true, "--stage": true,
	"-d": true, "--deleted": true, "-u": true, "--unmerged": true,
	"-t": true, "-v": true, "-f": true, "--full-name": true,
	"--error-unmatch": true, "--deduplicate": true, "--sparse": true,
}

// pathspecReadsAttributes reports whether a pathspec carries attr magic
// (":(attr:...)"), which matches against the attribute rules and so reads a
// skip-worktree .gitattributes from the object store.
func pathspecReadsAttributes(p string) bool {
	if !strings.HasPrefix(p, ":(") {
		return false
	}
	magic, _, _ := strings.Cut(p[2:], ")")
	for _, m := range strings.Split(magic, ",") {
		if strings.HasPrefix(strings.TrimSpace(m), "attr") {
			return true
		}
	}
	return false
}

// floorConfig is what the lazy-fetch floor reads from a repository's config.
type floorConfig struct {
	promisor bool // extensions.partialClone set, or a remote.<name>.promisor true
	sparse   bool // core.sparseCheckout or index.sparse true
}

// readFloorConfig reads the repository's promisor and sparse settings: a
// promisor remote is extensions.partialClone set or any remote.<name>.promisor
// true; sparse is core.sparseCheckout or index.sparse true. It reads through
// the same isolated config view the guarded command would (global and system
// config neutralised, includes followed, the worktree config where git reads
// it), so a key git would act on is a key it sees. A config read git cannot
// answer (exit other than 1, which is "no such key") is returned as an error,
// and a listing past the cap counts as both: the guard fails closed.
func readFloorConfig(root string) (floorConfig, error) {
	cmd := exec.Command("git", isolatedArgs(root, []string{"config", "-z", "--get-regexp",
		`^(extensions\.partialclone|remote\..*\.promisor|core\.sparsecheckout|index\.sparse)$`})...)
	cmd.Env = gitEnv()
	w := &capWriter{remaining: 64 << 10}
	e := &capWriter{remaining: 4096}
	cmd.Stdout, cmd.Stderr = w, e
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return floorConfig{}, nil
		}
		return floorConfig{}, fmt.Errorf("reading the repository's promisor config before an object read: %w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	if w.overflowed {
		return floorConfig{promisor: true, sparse: true}, nil
	}
	var cfg floorConfig
	for _, entry := range bytes.Split(w.buf, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		key, val, hasVal := strings.Cut(string(entry), "\n")
		// A bare key is true; only a value git reads as false clears it, and
		// an unparseable one counts as true.
		on := !hasVal || !isGitFalse(val)
		switch key {
		case "extensions.partialclone":
			cfg.promisor = true
		case "core.sparsecheckout", "index.sparse":
			cfg.sparse = cfg.sparse || on
		default: // remote.<name>.promisor
			cfg.promisor = cfg.promisor || on
		}
	}
	return cfg, nil
}

// isGitFalse reports whether git reads a boolean config value as false.
func isGitFalse(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "no", "off", "":
		return true
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 0, 64)
	return err == nil && n == 0
}
