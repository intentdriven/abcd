// Package banlist is abcd's two-layer banned-names store (itd-74, spc-20). It never
// prints and never exits — front doors under internal/surface/* format its results.
// Its I/O is reads and writes under a caller-supplied repo root, plus two
// subprocesses it must run to tell the truth about a pattern: `git check-ignore`, to
// refuse a private store git would track, and `grep`, to ask the private layer's
// ENFORCING engine whether an expression is usable there.
//
// The two layers differ in visibility, and that difference is the whole design:
//
//   - The PRIVATE layer is a gitignored per-machine file under the local tier
//     (PrivateRelPath). Its FIRST line declares its format — keyed
//     (KEY<space-or-tab>PATTERN) or legacy (one whole-line pattern per line) — and
//     the key is the only part any surface ever renders: a name that must never
//     appear in public content cannot be written into public config to ban it there.
//     The committed .githooks/pre-commit guard is the enforcement point, so this
//     package's job is the store, the format, and validation against that guard's
//     engine — parse, add, remove, list — not the matching. The one read-only
//     matcher it holds, MatchPrivate, asks that same engine which of a few
//     names the layer bans, so the load check can mask them before printing.
//   - The PUBLIC layer is the banned_tokens family of the committed docs-lint
//     config (PublicConfigRelPath), enforced deterministically in CI with a
//     per-line escape. There is exactly one banned-token primitive: the
//     hand-curated families already in that file and the entries these verbs write
//     are the same mechanism, and verb-written entries are namespaced under
//     PublicIDPrefix so ownership is legible.
//
// Neither layer's writer ever rewrites a whole file from a decoded structure: the
// private store is edited line-wise and the public one by byte surgery on the
// located array, so hand-written comments, ordering, and formatting survive an
// edit and a review diff shows only the entry that changed.
package banlist

import (
	"errors"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// PrivateRelPath is the gitignored per-machine private banlist, repo-relative. It
// sits in the local-ephemeral tier, which the three-tier layout gitignores as a
// whole — the one placement where a private pattern is safe from a `git add -A`.
const PrivateRelPath = ".abcd/.work.local/private-names.txt"

// PrivateDirRelPath is the local-ephemeral tier that holds the private store and
// its lock, repo-relative and slash-separated (an os.Root path).
const PrivateDirRelPath = ".abcd/.work.local"

// privateLockFilename and publicLockFilename name the load-modify-write locks.
// BOTH live in the local-ephemeral tier: a lock file beside the committed
// docs-lint config would be untracked litter in a committed directory, and the
// local tier is gitignored as a whole.
const (
	privateLockFilename = "banlist-private.lock"
	publicLockFilename  = "banlist-public.lock"
)

// storeLockTimeout bounds the wait for either store's lock, following the capture
// allocator's precedent: long enough to outlast a concurrent verb, short enough
// that a stale holder is reported rather than waited on for ever.
const storeLockTimeout = 5 * time.Second

// PublicConfigRelPath is the committed docs-lint config that holds the public
// layer, repo-relative.
const PublicConfigRelPath = ".abcd/docs-lint.json"

// PublicIDPrefix namespaces the banned_tokens entries these verbs own. Entries
// outside it (the hand-curated harness/* and present_tense/* families) are listed
// but never edited or removed by a verb: the prefix is the ownership boundary.
const PublicIDPrefix = "names/"

// PrivateReachNote is the one sentence every surface that describes the private
// layer must state (AC7). It lives here, once, because the statement is a property
// of the layer rather than of any one renderer: a status board, a verb, and a JSON
// envelope that worded it differently would let a reader believe one of them meant
// something weaker. It is not a limitation awaiting a fix — a pattern written into
// CI config is a published pattern, which is the whole reason this layer is local.
//
// It names the SECOND limit too, because "machines that have opted in" is necessary
// and not sufficient. A pre-commit hook sees the commits git asks it about, and a
// fast-forward `git pull` creates no commit at all — the commonest uncovered path of
// the lot, and the one a reader is least likely to think of. git runs no hook for a
// rebase, `git am`, `git revert` or a cherry-pick either, `--no-verify` switches it
// off, and a merge commit needs the separate pre-merge-commit half.
//
// The list is explicitly non-exhaustive ("e.g.") and carries nothing that is
// actually covered: `git stash pop` followed by a commit runs pre-commit like any
// other commit, and a bypass list naming a covered path teaches a reader to distrust
// the rest of it. A sentence that stopped at opt-in would leave them believing an
// opted-in machine is fully covered, which is the belief that gets a name committed.
const PrivateReachNote = "CI cannot enforce this layer — it protects only machines that have opted in, " +
	"and only the commits git runs a hook for (e.g. a fast-forward `git pull`, a rebase, `git am`, " +
	"`git revert` or a cherry-pick bypasses it, as does --no-verify)"

// maxStoreBytes caps every banlist read (trust boundary), following the guard
// registry's precedent.
const maxStoreBytes = 256 * 1024

// Severity values a public entry may carry.
const (
	SeverityBlocker = "blocker"
	SeverityWarn    = "warn"
)

// Sentinel errors. A message never contains a pattern value: on the private layer
// the pattern is the secret, and an error is output like any other.
var (
	// ErrNoStore reports that the layer's file does not exist. It is never
	// flattened into an empty success: "no store" and "no entries" are different
	// states, and conflating them makes silence look like protection.
	ErrNoStore = errors.New("banlist store is absent")
	// ErrInvalidKey rejects a key outside the portable key charset.
	ErrInvalidKey = errors.New("invalid banlist key")
	// ErrInvalidPattern rejects an empty or uncompilable pattern.
	ErrInvalidPattern = errors.New("invalid banlist pattern")
	// ErrInvalidSeverity rejects a severity outside blocker|warn.
	ErrInvalidSeverity = errors.New("invalid banlist severity")
	// ErrDuplicateKey rejects a key the layer already carries.
	ErrDuplicateKey = errors.New("banlist key already exists")
	// ErrUnknownKey reports a key the layer does not carry.
	ErrUnknownKey = errors.New("unknown banlist key")
	// ErrNotManaged reports an entry outside the verb-owned id namespace.
	ErrNotManaged = errors.New("banlist entry is not verb-managed")
	// ErrMalformedStore reports a store whose bytes cannot be read as this layer's
	// format at all (as opposed to one unusable line, which is reported by number).
	ErrMalformedStore = errors.New("banlist store is malformed")
	// ErrLegacyStore reports a private store with no format declaration and at
	// least one entry. Its lines are whole-line patterns, so a verb may not write a
	// keyed line into it: doing so would change what every other line means.
	ErrLegacyStore = errors.New("banlist store predates the keyed format")
	// ErrStoreNotIgnored reports a private store at a path git would track. The
	// layer's whole safety rests on the file being untracked.
	ErrStoreNotIgnored = errors.New("banlist private store is not gitignored")
	// ErrStoreLocked reports that another process holds the store's lock.
	ErrStoreLocked = errors.New("banlist store is locked by another process")
)

// keyRe is the portable key charset, shared by both layers and by the shell hook's
// own parser. It excludes every regular-expression metacharacter so a key can
// never be mistaken for the head of a pattern, and it excludes CR and LF so a key
// can never write a second line into a line-oriented store. What makes the split
// safe is NOT the charset, though — it is the store's format declaration: a keyed
// store splits every line and refuses one that will not, a legacy store splits
// none. Nothing is decided per line by how a field looks.
var keyRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

// validKey reports whether key is usable as a banlist key.
func validKey(key string) bool { return keyRe.MatchString(key) }

// validPublicPattern checks a PUBLIC pattern through the linter's own compile path,
// against the exact string that will be stored — the public layer's engine is Go's
// regexp, because `abcd lint docs` is what enforces it, and a check against
// anything else would be a check of a different thing. (The private layer's engine
// is grep; see checkPattern.) The compile error is DISCARDED: Go's regexp errors
// quote the expression, and one error path for both layers must never be the leak.
func validPublicPattern(stored string) bool {
	if stored == "" {
		return false
	}
	return lint.ValidateBannedToken(lint.BannedToken{Pattern: stored, AllowContext: []string{docsLintAllowEscape}}) == nil
}
