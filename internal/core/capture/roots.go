package capture

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// resolveRoots resolves (repoRoot, issuesRoot) from the request fields plus git
// discovery, mirroring _issue_lib._resolve_roots (contracts A–D). repoRoot is
// canonicalised; issuesRoot is made absolute without following symlinks so the
// symlink-refusal guards stay effective.
func resolveRoots(repoRoot, issuesRoot string) (string, string, error) {
	var rr string
	switch {
	case repoRoot != "":
		abs, err := filepath.Abs(repoRoot)
		if err != nil {
			return "", "", err
		}
		rr = abs
	case issuesRoot != "":
		// Discover the repo from the explicit issuesRoot's parent.
		absIssues, err := filepath.Abs(issuesRoot)
		if err != nil {
			return "", "", err
		}
		disc := discoverRepoRoot(filepath.Dir(absIssues))
		if disc == "" {
			return "", "", fmt.Errorf("custom issuesRoot requires explicit repoRoot when not in a git repo")
		}
		rr = disc
	default:
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", err
		}
		disc := discoverRepoRoot(cwd)
		if disc == "" {
			return "", "", fmt.Errorf("cannot resolve repoRoot: not in a git repo and no explicit roots given")
		}
		rr = disc
	}

	var ir string
	if issuesRoot != "" {
		abs, err := filepath.Abs(issuesRoot)
		if err != nil {
			return "", "", err
		}
		ir = abs
	} else {
		ir = filepath.Join(rr, LedgerRelPath)
	}
	// Every verb passes through here first, the readers included, so this is
	// where the ledger's directories are judged for a List or Status: a committed
	// symlink at .abcd, .abcd/work, issues/ or a status directory would otherwise
	// make a read serialize a ledger from outside the checkout, under
	// repo-relative paths (GHSA-865x-5m7q-qm79). The write paths judge the same
	// list again, creating, from under ensureLedgerDirs.
	if err := ledgerDirs(rr, ir, false); err != nil {
		return "", "", err
	}
	return rr, ir, nil
}

// ErrNoLedgerRoot is LedgerRoot's refusal: there is no checkout whose ledger the
// caller's working directory belongs to, so there is no ledger to address. A
// surface maps it to an exit code and its own wording; core never prints.
var ErrNoLedgerRoot = errors.New("no ledger root")

// LedgerRoot answers the question every front door has to ask before it touches
// the ledger: which checkout's ledger does a caller standing in cwd address?
//
// It is the ONE resolution for that question, and it exists because the CLI used
// to skip asking it — every capture verb handed its working directory to
// resolveRoots as an explicit repo root, so the discovery below was never
// reached and a verb run from a subdirectory addressed a ledger that was not
// there: a read reported open 0 against a populated checkout, and a write minted
// a second ledger under the subdirectory and reported success with a
// repo-relative path that looked ordinary (iss-2609090951291524).
//
// Three outcomes, and only the first is a root:
//
//   - git names a toplevel: that is the answer, whoever owns the checkout.
//   - git will not answer for a repo-SHAPED tree (git absent from PATH, a
//     corrupt .git, an ownership refusal under the isolated env): REFUSED,
//     naming that git could not answer. This deliberately does not fall
//     through to discoverRepoRoot's marker walk, which accepts any directory
//     merely carrying the name and grew neither the shape check nor the
//     ownership gate its rules-root sibling has (iss-2609090947359464). That
//     walk is unreachable in shipped code, and routing the front doors through
//     it would be the one change that makes it live. A ledger addressed by a
//     guess is the defect this function closes, one directory further out.
//   - nothing repo-shaped anywhere above: REFUSED. Minting a ledger in whatever
//     directory the caller stood in is not a lenient fallback — the records
//     would sit outside any checkout, committed by nothing and read by nothing,
//     which is the same lost trail this resolution exists to prevent. The ledger
//     is per-repository by definition.
func LedgerRoot(cwd string) (string, error) {
	if top, err := gitutil.Run(cwd, "rev-parse", "--show-toplevel"); err == nil && top != "" {
		return top, nil
	}
	// Neither message carries the working directory: an error envelope never
	// leaks an absolute local path (iss-76), and the caller already knows where
	// they are standing.
	if gitutil.RepoShapedRoot(cwd) != "" {
		return "", fmt.Errorf("%w: git could not name the repository root for the working directory (git absent from PATH, the repository unreadable, or its ownership refused), and the ledger is never guessed at",
			ErrNoLedgerRoot)
	}
	return "", fmt.Errorf("%w: the working directory is not inside a git repository, and the issue ledger is per-repository: run this from a checkout",
		ErrNoLedgerRoot)
}

// discoverRepoRoot returns the git worktree root containing start, or "".
func discoverRepoRoot(start string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = start
	// Isolate: `rev-parse --show-toplevel` honours an inherited GIT_WORK_TREE/GIT_DIR
	// over cmd.Dir, so without scrubbing an inherited value redirects repo-root
	// discovery at a DIFFERENT tree — and the derived issuesRoot then reads and
	// writes the ledger under an attacker-chosen path. Repo discovery needs no
	// global config, so full isolation is safe.
	cmd.Env = gitutil.IsolatedEnv()
	out, err := cmd.Output()
	if err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

var reNonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// deriveSlug produces a filename-safe slug from free text: lowercase, collapse
// every non-[a-z0-9] run to a single hyphen, trim, then truncate to 60 chars.
//
// It lives in core, not on the CLI, precisely so it runs on the ALREADY-REDACTED
// text (Capture redacts its inputs before it calls this). A caller that
// kebab-cased the raw text first — which the CLI used to do — hands the ledger
// redactor "users-alice-local-bin-abcd", where nothing looks like a path any
// more, so the username survives into the committed filename even though the body
// is redacted (gh-485). Deriving here, after redaction, closes that seam and
// mirrors the intent engine's deriveIntentSlug, which likewise keeps derivation
// in core rather than trusting a pre-kebab'd slug.
func deriveSlug(text string) string {
	collapsed := strings.Trim(reNonSlug.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if len(collapsed) > 60 {
		collapsed = strings.Trim(collapsed[:60], "-")
	}
	return collapsed
}

// normaliseSlug lowercases, collapses non-alphanumeric runs to a single hyphen,
// and trims hyphens, mirroring _normalise_slug. Empty result is an error.
func normaliseSlug(slug string) (string, error) {
	candidate := strings.Trim(reNonSlug.ReplaceAllString(strings.ToLower(slug), "-"), "-")
	if candidate == "" {
		return "", fmt.Errorf("slug normalises to empty: %q", slug)
	}
	if !reSlug.MatchString(candidate) {
		return "", fmt.Errorf("slug %q is not kebab-case", candidate)
	}
	return candidate, nil
}

var emptyChecksum = sha256Hex(nil)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// readWithChecksum reads a record once, through the guarded reader, and hashes
// that same buffer. It is the read behind Promote, transition and the capture
// commit, so the guard below is what those verbs stand on: a symlinked or
// non-regular leaf is refused before a stamp could be written through it.
func readWithChecksum(path string) (string, string, error) {
	content, err := readRecordGuarded(path)
	if err != nil {
		return "", "", err
	}
	return content, sha256Hex([]byte(content)), nil
}

// readRecordGuarded reads one record file through the shared trust-boundary
// primitive: fsutil.ReadGuarded opens once with O_NOFOLLOW and O_NONBLOCK and
// validates on the SAME descriptor, so no symlink swap fits between a check and
// the read, and a FIFO or device at a record name cannot block the open. The cap
// is issueschema.RecordReadLimit, the ONE cap the ledger's families share —
// core/lint applies the same value, because a cap the board applies loosely and
// the verb applies tightly makes the ledger say two things about one file.
//
// It is the reader for EVERY record family in this package. The reading and
// disposition families used it from the start; the issue family read through a
// bare os.ReadFile until GHSA-fh9j-8xmg-m33f, so a committed FIFO hung `list`,
// an oversize record was serialized unbounded, and a committed symlink read an
// out-of-tree file into `list --json` (iss-2609012036271396). A record store a
// clone carries is a trust boundary whichever family the record belongs to.
//
// The sentinels are mapped to this package's own: a non-regular leaf, or a
// symlink refused by O_NOFOLLOW, is ErrPathUnsafe, which is what every caller
// here already tests for.
func readRecordGuarded(path string) (string, error) {
	data, err := fsutil.ReadGuarded(path, issueschema.RecordReadLimit)
	if err == nil {
		return string(data), nil
	}
	if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP) {
		return "", fmt.Errorf("%w: record path is not a regular file: %s", ErrPathUnsafe, path)
	}
	if errors.Is(err, fsutil.ErrTooBig) {
		return "", fmt.Errorf("%w: record is larger than the %d-byte size cap and was left unread: %s", ErrPathUnsafe, issueschema.RecordReadLimit, path)
	}
	// A record the process may not open is a refusal with a name, not a raw open
	// error surfacing through a verb: the caller is told the ledger could not be
	// read and which file, rather than a syscall's own wording.
	if errors.Is(err, fs.ErrPermission) {
		return "", fmt.Errorf("%w: record is unreadable (permission denied): %s", ErrPathUnsafe, path)
	}
	return "", err
}
