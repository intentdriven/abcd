package gitutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// isolatedGit builds a git command under root with global and system config
// neutralised, so a developer's environment cannot change what abcd observes —
// and with the repo-local config knobs that can execute code on an
// otherwise read-only command forced off. The probe points git at arbitrary,
// possibly-hostile repositories, and a repo's own .git/config is fully trusted
// by git and cannot be disabled by env; ExecPins forces off the knobs that would
// start a program — a hook, an fsmonitor daemon, a signature verifier. These
// are the defence for read-only commands (log/tag/rev-list/rev-parse); a command
// that honours external-diff/textconv/pager config must not be added to the
// probe without further hardening.
//
// Every isolated command passes lazyFetchGuard first: a refusal is set as the
// command's Err, so Run, Output and Start return it and git never starts.
func isolatedGit(root string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", isolatedArgs(root, args)...)
	cmd.Env = gitEnv()
	guardObjectRead(cmd, root, args)
	return cmd
}

// guardObjectRead applies lazyFetchGuard to an isolated command, setting a
// refusal as cmd.Err (which exec returns from Start before anything runs).
func guardObjectRead(cmd *exec.Cmd, root string, args []string) {
	if cmd.Err != nil {
		return
	}
	if err := lazyFetchGuard(root, args); err != nil {
		cmd.Err = err
	}
}

// contextWaitDelay is how long a context-bound git's Wait waits, once the
// context has ended and git was killed, for its output pipes to close — so a
// process git started that kept a pipe open cannot hold the caller past its
// deadline.
const contextWaitDelay = 50 * time.Millisecond

// isolatedGitContext is isolatedGit bound to ctx: when ctx ends, git is KILLED,
// not abandoned. An abandoned git outlives its caller, and a caller on a
// deadline that runs often — the status verb, on every refresh — would leave
// one behind each time for as long as git hangs.
func isolatedGitContext(ctx context.Context, root string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", isolatedArgs(root, args)...)
	cmd.Env = gitEnv()
	cmd.WaitDelay = contextWaitDelay
	guardObjectRead(cmd, root, args)
	return cmd
}

// isolatedArgs is the isolated command line: the config knobs that can run
// code forced off (ExecPins), verbatim paths, then -C root and the caller's
// arguments.
func isolatedArgs(root string, args []string) []string {
	return append(append(ExecPins(),
		// Emit paths verbatim (UTF-8), not the default C-quoted, double-quoted
		// form for non-ASCII bytes: a caller that matches a git-reported path
		// against a filesystem-derived one (site date history) would never match
		// the quoted key and lose the record's dates.
		"-c", "core.quotePath=false",
		"-C", root,
	), args...)
}

// gitEnv builds the child environment for an isolated git command: the parent
// environment with every repo-selection and config-injection variable stripped,
// then the config-file neutralisers appended. Neutralising config *files* is not
// enough — an inherited GIT_DIR/GIT_WORK_TREE/GIT_INDEX_FILE takes precedence
// over `-C root` and silently redirects the query to a *different* repository,
// and GIT_CONFIG_COUNT/GIT_CONFIG_PARAMETERS re-inject config that
// GIT_CONFIG_GLOBAL/NOSYSTEM would otherwise suppress. Repo selection and config
// therefore come from the command line alone; see scrubGitVar for the exact set
// dropped (deliberate pass-throughs such as GIT_EXEC_PATH are kept).
func gitEnv() []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+16)
	for _, kv := range base {
		if scrubGitVar(kv) {
			continue
		}
		env = append(env, kv)
	}
	// LC_ALL=C / LANG=C pin git's own chrome to the C locale, appended AFTER
	// os.Environ() so they win over any ambient locale. A translated git would
	// otherwise localise porcelain summaries — e.g. the "N files changed"
	// shortstat the graveyard's wholesale-rewrite signal parses — silently
	// killing the signal on a French/German host and breaking the cross-host
	// determinism of the produced manifest.
	// No detached background work: a git command abcd runs must not leave a
	// gc/maintenance process writing under .git after it returns. In production
	// that is a subprocess outliving a CLI verb inside a user's repository; in a
	// test it is the documented ".git/objects: directory not empty" cleanup race
	// (iss-252 for fixture repos, iss-2609020319494139 for every other isolated
	// call). The keys ride in the environment rather than as -c flags so a caller
	// that builds its own git command from IsolatedEnv — every test helper that
	// inits its own repository — inherits them too. The parent's own
	// GIT_CONFIG_COUNT injection was scrubbed above, so this is the only
	// environment config in effect.
	// GIT_NO_LAZY_FETCH=1: in a partial clone git answers a read of a MISSING
	// object by fetching it, and the fetch runs the transport the repository's
	// own config names (remote.<name>.uploadpack for a local URL,
	// core.sshCommand for ssh://), so an object read abcd made in a copied
	// checkout ran a program that checkout chose (iss-2610090821527948). A
	// missing object is an error instead; a present one still reads. It is
	// appended after the parent's environment, so it wins over an inherited
	// value. git honours it from 2.44; below that floor lazyFetchGuard refuses
	// an object read in a repository that declares a promisor remote.
	return append(env,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_CONFIG_COUNT=4",
		"GIT_CONFIG_KEY_0=gc.auto", "GIT_CONFIG_VALUE_0=0",
		"GIT_CONFIG_KEY_1=gc.autodetach", "GIT_CONFIG_VALUE_1=false",
		"GIT_CONFIG_KEY_2=maintenance.auto", "GIT_CONFIG_VALUE_2=false",
		"GIT_CONFIG_KEY_3=core.fsmonitor", "GIT_CONFIG_VALUE_3=false",
		"LC_ALL=C",
		"LANG=C",
	)
}

// IsolatedEnv returns the child environment an isolated git command runs with:
// the parent environment scrubbed of every repo-selection and config-injection
// variable (GIT_DIR, GIT_WORK_TREE, GIT_CONFIG_*, …), plus the config-file
// neutralisers. A front door that must run a git command this package does not
// already wrap — a probe needing --stdin, say — uses this instead of
// os.Environ() so it inherits the same isolation as Run/RunLimited. Reaching for
// os.Environ() directly lets an inherited GIT_DIR/GIT_WORK_TREE silently
// redirect the command at a DIFFERENT repository, which for a secret-hygiene gate
// is a real vulnerability.
func IsolatedEnv() []string { return gitEnv() }

// ScrubbedEnv is IsolatedEnv without the global/system config-file neutralisers:
// the parent environment with every repo-selection and config-injection variable
// stripped (GIT_DIR, GIT_WORK_TREE, GIT_INDEX_FILE, GIT_CONFIG_COUNT/PARAMETERS/
// KEY_*/VALUE_*, …) but with ~/.gitconfig and the system config still in effect.
// It is for a git command that MUST honour the developer's real global config —
// the identity probe reads user.name/user.email to redact the caller's OWN
// identity, and those overwhelmingly live in global config, so suppressing it
// (as IsolatedEnv does) would blind the probe and silently stop redacting a real
// identity leak. Scrubbing still defeats the two attacks that matter here: an
// inherited GIT_DIR redirecting the probe at a different repository, and an
// injected GIT_CONFIG_* forging a fake identity that displaces the real one.
func ScrubbedEnv() []string {
	base := os.Environ()
	env := make([]string, 0, len(base))
	for _, kv := range base {
		if scrubGitVar(kv) {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// CommandLineConfig is the `git -c` configuration the parent environment
// carries — GIT_CONFIG_PARAMETERS and the GIT_CONFIG_COUNT/GIT_CONFIG_KEY_n/
// GIT_CONFIG_VALUE_n form, the entries ScrubbedEnv strips — as "KEY=value"
// entries, in environment order. git hands exactly these to a command it runs
// (a hook, an alias), so a reader that must see the configuration git itself
// is running with appends them to ScrubbedEnv for its own command; no other
// caller's scrub changes.
func CommandLineConfig() []string {
	var out []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if key == "GIT_CONFIG_PARAMETERS" || key == "GIT_CONFIG_COUNT" ||
			strings.HasPrefix(key, "GIT_CONFIG_KEY_") || strings.HasPrefix(key, "GIT_CONFIG_VALUE_") {
			out = append(out, kv)
		}
	}
	return out
}

// scrubGitVar reports whether an "KEY=value" environment entry names a git
// repo-selection or config-injection variable that must not leak into an
// isolated command. It is deliberately a denylist: unrelated GIT_* pass-throughs
// (GIT_EXEC_PATH, GIT_SSH, …) and the config-file neutralisers gitEnv appends
// are kept intact.
func scrubGitVar(kv string) bool {
	key := kv
	if i := strings.IndexByte(kv, '='); i >= 0 {
		key = kv[:i]
	}
	switch key {
	case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE",
		"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
		"GIT_NAMESPACE", "GIT_COMMON_DIR",
		"GIT_CEILING_DIRECTORIES", "GIT_DISCOVERY_ACROSS_FILESYSTEM",
		"GIT_CONFIG", "GIT_CONFIG_COUNT", "GIT_CONFIG_PARAMETERS":
		return true
	}
	return strings.HasPrefix(key, "GIT_CONFIG_KEY_") ||
		strings.HasPrefix(key, "GIT_CONFIG_VALUE_")
}

// capWriter buffers at most a fixed number of bytes, discarding the rest, and
// never errors — so a git process writing far more than the cap is not blocked
// (no SIGPIPE) yet cannot grow abcd's memory past the cap. It RECORDS having
// discarded anything, so a caller that cannot use a partial answer can tell the
// difference; a caller that can (the lifeboat probe) ignores the flag.
type capWriter struct {
	buf       []byte
	remaining int
	// overflowed is true once any byte has been dropped.
	overflowed bool
}

func (w *capWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n > w.remaining {
		n = w.remaining
		w.overflowed = true
	}
	if n > 0 {
		w.buf = append(w.buf, p[:n]...)
		w.remaining -= n
	}
	return len(p), nil
}

// InRepo reports whether root is inside a git working tree. A convention rule
// uses it to tell "git says this path is not ignored" apart from "git cannot
// answer" (git absent, or not a repo) — the latter is "cannot tell", never
// "compliant".
func InRepo(root string) bool {
	out, err := isolatedGit(root, "rev-parse", "--is-inside-work-tree").Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// IsAncestor reports whether ancestor is an ancestor of (or equal to) descendant
// under root, via `git merge-base --is-ancestor`. git encodes the answer in the
// exit status — 0 is "yes", 1 is "no", and anything else (128 for a bad object, a
// missing repository) is a real failure — so a bare Run cannot be used: it folds
// the informative 1 into a generic error. A caller deriving a release's content
// commit from the receipts directory needs the three outcomes kept apart: "yes"
// selects a candidate, "no" skips it, and only a genuine git failure is fatal.
func IsAncestor(root, ancestor, descendant string) (bool, error) {
	cmd := isolatedGit(root, "merge-base", "--is-ancestor", ancestor, descendant)
	e := &capWriter{remaining: 4096}
	cmd.Stderr = e
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
}

// RootCommit returns the repository's canonical (first) root-commit SHA, or ""
// when it cannot be derived — git absent, not a repository, no commits. It is
// total: a caller keying a store or a marker on the repository's identity gets
// "" rather than an error, and decides for itself what an unidentified
// repository means. A repository can have several root commits (an octopus of
// unrelated histories); the first `rev-list` reports is the canonical one, the
// same choice the history registry and the lifeboat probe make.
//
// The output is bounded (one object name, so the cap is generous) rather than
// buffered whole: a hostile repository must not be able to make an identity
// probe allocate.
func RootCommit(root string) string {
	sha, _ := RootCommitContext(context.Background(), root)
	return sha
}

// RootCommitContext is RootCommit bound to ctx: the same answer, total in the
// same way, except that a context that ends before git answers is returned as
// its own error (wrapped) with git killed — so a caller on a deadline can tell
// "this repository has no root commit" from "git did not answer in time".
func RootCommitContext(ctx context.Context, root string) (string, error) {
	out, err := RunLimitedContext(ctx, root, 4096, "rev-list", "-n", "1", "--max-parents=0", "HEAD")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if err != nil {
		return "", nil
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}

// fullSHARe is a full object name: forty hex digits under SHA-1, sixty-four
// under SHA-256, lower case as git prints them.
var fullSHARe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// IsFullSHA reports whether s is a full object name as RootCommit returns one.
// The machine-scoped stores key a directory on the root commit, so the check is
// what keeps a value that arrived any other way from becoming a path segment.
func IsFullSHA(s string) bool { return fullSHARe.MatchString(s) }

// RepoShaped reports whether root sits anywhere inside a tree carrying a .git
// entry — a directory, or the file a worktree or submodule leaves. It is
// deliberately cruder than InRepo: its job is to tell "not a repository" apart
// from "a repository git would not answer for" (git absent from PATH, a corrupt
// .git, an ownership refusal under the isolated env). Only the first is safe to
// treat as "nothing is tracked here"; the others can commit, and nothing can
// say what git would answer.
//
// It walks to the filesystem root, because git does: checking root alone
// answers "not a repository" for every SUBDIRECTORY of one.
func RepoShaped(root string) bool { return RepoShapedRoot(root) != "" }

// RepoShapedRoot is RepoShaped's walk with its answer kept: the nearest
// directory at or above root carrying a .git entry, or "" when there is none.
// It is the toplevel a caller can still bound work at when git itself will not
// name one — the ownership refusal under the isolated env, git absent from
// PATH, a corrupt .git — where the alternative is treating a real working tree
// as an unbounded directory.
//
// It is a MARKER, not git's answer: it does not read .git, so it cannot tell a
// valid repository from a directory that merely holds the name, and for a
// submodule or linked worktree it reports the tree the .git file sits in
// (which is the working-tree root a config walk wants). A caller that needs
// git's own answer must ask git.
func RepoShapedRoot(root string) string {
	dir := root
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// ErrNoCheckoutRoot is CheckoutRoot's refusal: there is no checkout whose
// record store the caller's working directory belongs to, so there is no store
// to address. A surface maps it to an exit code and its own wording; this
// package never prints.
var ErrNoCheckoutRoot = errors.New("no checkout root")

// CheckoutRoot answers the question every front door onto a repository-scoped
// record store has to ask before it builds a request: which checkout's store
// does a caller standing in cwd address?
//
// It is the ONE resolution for that question, and it exists because two front
// doors skipped asking it and reached the same failure independently. The
// capture verbs handed their working directory to the ledger core as an explicit
// repo root, so a verb run from a subdirectory addressed a ledger that was not
// there: a read reported open 0 against a populated checkout, and a write minted
// a second ledger under the subdirectory and reported success with a
// repo-relative path that looked ordinary (iss-2609090951291524). `decide` did
// the same to the decision store, and one directory further out: run outside
// every repository it exited 0 and laid a full ADR store in whatever plain
// directory the caller stood in (iss-2609091707224329). The resolution is
// store-agnostic, so it lives here rather than in either store's package, and
// `store` — a noun phrase naming what is being addressed, "the issue ledger",
// "the decision store" — is the ONLY thing that varies between callers.
//
// Three outcomes, and only the first is a root:
//
//   - git names a toplevel: that is the answer, whoever owns the checkout.
//   - git will not answer for a repo-SHAPED tree (git absent from PATH, a
//     corrupt .git, an ownership refusal under the isolated env, or an answer
//     Toplevel refuses for its shape, which a core.worktree setting naming a
//     tree that does not contain cwd, or an ancestor of the checkout that does
//     not hold its .git (iss-2610090821543020), produces): REFUSED,
//     naming that git could not answer. RepoShapedRoot is read here as a
//     CLASSIFIER and never as a root: it is a marker walk, which accepts any
//     directory merely carrying the name and has neither the shape check nor
//     the ownership gate the rules-root resolver grew (iss-2609090947359464),
//     so returning its answer is the one change that would make that walk live.
//     A store addressed by a guess is the defect this function closes, one
//     directory further out.
//   - nothing repo-shaped anywhere above: REFUSED. Laying a store in whatever
//     directory the caller stood in is not a lenient fallback — the records
//     would sit outside any checkout, committed by nothing and read by nothing,
//     which is the same lost trail this resolution exists to prevent. Every
//     store this resolves for is per-repository by definition.
func CheckoutRoot(cwd, store string) (string, error) {
	if top, err := Toplevel(cwd); err == nil {
		return top, nil
	}
	// Neither message carries the working directory: an error envelope never
	// leaks an absolute local path (iss-76), and the caller already knows where
	// they are standing.
	if RepoShapedRoot(cwd) != "" {
		return "", fmt.Errorf("%w: git could not name the repository root for the working directory (git absent from PATH, the repository unreadable, its ownership refused, or a core.worktree setting naming a working tree other than this checkout), and %s is never guessed at",
			ErrNoCheckoutRoot, store)
	}
	return "", fmt.Errorf("%w: the working directory is not inside a git repository, and %s is per-repository: run this from a checkout",
		ErrNoCheckoutRoot, store)
}

// TrackedFiles returns the repo-relative paths git tracks under root, NUL-safe
// so a filename with a newline cannot desync the list. Outside anything
// repo-shaped it returns no files and no error — a scan over committed files
// then degrades to "nothing to scan" rather than failing. In a repo-shaped tree
// git cannot answer for, and inside a repo on any other ls-files failure (a
// corrupt index, say), it returns an error, so a caller cannot mistake "could
// not read the repository" for "nothing tracked" and report a scanning rule
// compliant after reading zero files.
func TrackedFiles(root string) ([]string, error) {
	if !InRepo(root) {
		if RepoShaped(root) {
			// Repo-shaped, but git could not answer: absent from PATH, an
			// unreadable or corrupt repository, or an ownership refusal under
			// the isolated env. Content here CAN be committed, so "nothing
			// tracked" would be a false clean.
			return nil, errors.New("git could not read the repository here (absent, unreadable, or refused), so tracked files cannot be listed")
		}
		// Not a repository → nothing tracked, not an error.
		return nil, nil
	}
	out, err := isolatedGit(root, "ls-files", "-z").Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(out), "\x00")
	files := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// Run executes a read-only git command under root with the developer's global
// and system config neutralised, returning trimmed stdout. It is the shared
// primitive for tooling that reads git history (the lifeboat probe's Tier-0
// adapters); centralising it keeps every caller on the same isolated
// environment rather than re-deriving the exec plumbing. An error (git absent,
// not a repo, a failing subcommand) is returned verbatim so the caller can
// decide whether "git cannot answer" degrades to empty or is fatal.
func Run(root string, args ...string) (string, error) {
	out, err := isolatedGit(root, args...).Output()
	if err != nil {
		return "", withStderr(err)
	}
	return strings.TrimSpace(string(out)), nil
}

// withStderr surfaces a failed git command's own (bounded) stderr in the
// returned error. A bare "exit status 128" hides git's reason ("bad object",
// "not a git repository", an object-store race) and makes a probe failure over
// an archived or hostile repo undiagnosable; the reason is the diagnosis.
func withStderr(err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		msg := ee.Stderr
		if len(msg) > 4096 {
			msg = msg[:4096]
		}
		return fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(msg)))
	}
	return err
}

// RunLimited is Run with a hard cap on how much stdout is buffered. A hostile or
// archived repository can make a read-only command (a full `git log`) emit
// arbitrarily much output; the unbounded `Output()` would buffer all of it. The
// probe uses this so a giant history cannot exhaust memory — output past
// maxBytes is discarded (the last retained line may be truncated, which degrades
// a single probe rather than crashing it). On failure the error carries git's
// bounded stderr — the reason, not just the exit status.
func RunLimited(root string, maxBytes int, args ...string) (string, error) {
	out, _, err := runBounded(root, maxBytes, args...)
	return out, err
}

// RunLimitedContext is RunLimited bound to ctx: when ctx ends before git
// answers, git is killed (isolatedGitContext) and the error wraps ctx's own,
// so errors.Is(err, context.DeadlineExceeded) tells a caller that git did not
// answer in time rather than that it said no. It is the one primitive every
// deadline-bound git question goes through.
func RunLimitedContext(ctx context.Context, root string, maxBytes int, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	out, _, err := runBoundedCmd(isolatedGitContext(ctx, root, args...), maxBytes)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), ctxErr)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), " \t\r\n"), nil
}

// RunCapped is RunLimited for callers whose answer is only correct if it is
// COMPLETE. It returns an error rather than a truncated string when git's output
// exceeds maxBytes.
//
// The distinction is the whole reason both exist. A truncated `git log` is not a
// shorter history; it is a wrong one, and a caller that cannot tell the two
// apart will publish the wrong one with no sign that anything was dropped.
// RunLimited's callers (the lifeboat probe over an archived repository) would
// genuinely rather have a partial answer than none, and keep that behaviour.
func RunCapped(root string, maxBytes int, args ...string) (string, error) {
	out, overflowed, err := runBounded(root, maxBytes, args...)
	if err != nil {
		return "", err
	}
	if overflowed {
		return "", fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated history is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}

// runBounded is the shared body: run git with stdout and stderr bounded, and
// report whether stdout was cut short.
func runBounded(root string, maxBytes int, args ...string) (string, bool, error) {
	out, overflowed, err := runBoundedBytes(root, maxBytes, args...)
	if err != nil {
		return "", overflowed, err
	}
	// Trim the trailing side only. Leading bytes are content: a NUL-separated
	// listing (-z) starts with its first entry, and a whole-buffer TrimSpace
	// silently stripped leading whitespace off that entry's filename — the -z
	// form exists precisely so such names survive. Every consumer parses
	// per-line, per-field, or per-NUL and tolerates a leading space; none may
	// lose one.
	return strings.TrimRight(string(out), " \t\r\n"), overflowed, nil
}

// runBoundedBytes is the untrimmed body runBounded and RunCappedBytes share:
// run git with stdout and stderr bounded, return stdout verbatim, and report
// whether it was cut short.
func runBoundedBytes(root string, maxBytes int, args ...string) ([]byte, bool, error) {
	return runBoundedCmd(isolatedGit(root, args...), maxBytes)
}

// runBoundedCmd runs an isolated git command with stdout and stderr bounded.
func runBoundedCmd(cmd *exec.Cmd, maxBytes int) ([]byte, bool, error) {
	w := &capWriter{remaining: maxBytes}
	e := &capWriter{remaining: 4096}
	cmd.Stdout = w
	cmd.Stderr = e
	if err := cmd.Run(); err != nil {
		return nil, w.overflowed, fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	return w.buf, w.overflowed, nil
}

// ErrToplevelShape is Toplevel's refusal of an answer git would never give.
var ErrToplevelShape = errors.New("git's toplevel answer is not one absolute path containing the directory asked about and holding the git directory git found")

// Toplevel asks git for the working-tree root that contains dir, and holds the
// answer to the one shape git ever gives: a single absolute line naming a
// directory at or above dir. Anything else (relative, several lines, empty, a
// directory elsewhere) is refused with ErrToplevelShape rather than handed on
// as a root, because every caller bounds work at the answer: it is where a
// ledger, a rules file or a site is read and written. It is the one call every
// `rev-parse --show-toplevel` in the module goes through, so the rule is kept
// once (iss-2608292038186663).
//
// A git failure (not a repository, git absent, an ownership refusal under the
// isolated environment) is returned as git's own error; a caller that must
// tell "no repository" from "a repository git will not answer for" follows up
// with RepoShapedRoot.
func Toplevel(dir string) (string, error) {
	return ToplevelContext(context.Background(), dir)
}

// toplevelCap bounds the toplevel answer. One path is a few hundred bytes at
// most; an answer that overflows the cap cannot be one path, and the shape
// check refuses whatever is left of it.
const toplevelCap = 64 << 10

// ToplevelContext is Toplevel bound to ctx, for a caller on a deadline (the
// status verb): the same question and the same shape check, with git killed
// when ctx ends and the context's own error returned (RunLimitedContext), so
// the caller can tell a slow git from "not a repository".
//
// Containment is not enough on its own: a repo-local core.worktree is resolved
// by git relative to the .git directory, so `core.worktree=../..` in a copied
// checkout names the checkout's PARENT, which contains dir and passed the shape
// check, widening every record store to the parent (iss-2610090821543020). So
// git is asked, in the same invocation, for the git directory it discovered,
// and the toplevel is accepted only when it holds that directory: its .git is
// the directory itself, or a gitfile naming it (a linked worktree, a
// submodule, a --separate-git-dir checkout). An ancestor that core.worktree
// alone selected holds no such entry and is refused with ErrToplevelShape.
func ToplevelContext(ctx context.Context, dir string) (string, error) {
	out, err := RunLimitedContext(ctx, dir, toplevelCap, "rev-parse", "--show-toplevel", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	top, gitDir, ok := strings.Cut(out, "\n")
	if !ok || !ToplevelShaped(dir, top) || !holdsGitDir(top, gitDir) {
		return "", ErrToplevelShape
	}
	return top, nil
}

// gitfileCap bounds the gitfile holdsGitDir reads: one "gitdir: <path>" line.
const gitfileCap = 4 << 10

// holdsGitDir reports whether top's own .git entry is gitDir: the directory
// itself, a symlink to it, or a regular file whose "gitdir: " line names it (relative to top
// when the path is relative). Identity is compared by file, never by spelling,
// so a symlinked temp root or a case variant on a case-insensitive volume does
// not refuse a real checkout. A gitDir that is not one absolute line is no
// answer.
func holdsGitDir(top, gitDir string) bool {
	if gitDir == "" || !filepath.IsAbs(gitDir) || strings.ContainsAny(gitDir, "\n\r") {
		return false
	}
	want, err := os.Stat(gitDir)
	if err != nil || !want.IsDir() {
		return false
	}
	entry := filepath.Join(top, ".git")
	fi, err := os.Lstat(entry)
	if err != nil {
		return false
	}
	switch {
	case fi.IsDir():
		return os.SameFile(fi, want)
	case fi.Mode()&os.ModeSymlink != 0:
		// git follows a symlinked .git to the directory it names; the
		// toplevel holds that directory as it holds one a gitfile names.
		got, err := os.Stat(entry)
		return err == nil && got.IsDir() && os.SameFile(got, want)
	case fi.Mode().IsRegular():
		f, err := os.Open(entry)
		if err != nil {
			return false
		}
		defer f.Close()
		buf := make([]byte, gitfileCap)
		n, _ := io.ReadFull(f, buf)
		line, _, _ := strings.Cut(string(buf[:n]), "\n")
		target, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "gitdir: ")
		if !ok || target == "" {
			return false
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(top, target)
		}
		got, err := os.Stat(target)
		return err == nil && os.SameFile(got, want)
	}
	return false
}

// RevParseAbsPath asks `git rev-parse --path-format=absolute <flag>` for one
// path (--git-dir, --git-common-dir, --show-toplevel) and refuses any answer
// that is not exactly one absolute line. --path-format arrived in git 2.31: an
// older rev-parse echoes the option it does not know to stdout and exits 0, so
// its answer is the flag's text and a path on two lines, which a caller that
// compared it would read as a path. Refusing it keeps a resolution on an old
// git failing closed where it compares, never on a string that only looks
// like an answer.
func RevParseAbsPath(dir, flag string) (string, error) {
	out, err := Run(dir, "rev-parse", "--path-format=absolute", flag)
	if err != nil {
		return "", err
	}
	if out == "" || strings.ContainsAny(out, "\r\n") || !filepath.IsAbs(out) {
		return "", fmt.Errorf("git rev-parse %s named no single absolute path (--path-format needs git 2.31 or later)", flag)
	}
	return out, nil
}

// ToplevelShaped reports whether top has the shape of git's toplevel answer
// for dir: one absolute line naming a directory that contains dir. It is
// Toplevel's check, for the callers that must run git themselves (a command
// that pins the git binary it runs, or asks for the path in another format).
func ToplevelShaped(dir, top string) bool {
	if top == "" || !filepath.IsAbs(top) || strings.ContainsAny(top, "\n\r") {
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return fsutil.PathWithin(fsutil.RealExistingPath(abs), fsutil.RealExistingPath(top), fsutil.CaseFoldingFS())
}
