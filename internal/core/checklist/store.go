package checklist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TierRelPath is the local tier the checklist lives in, repo-relative and
// slash-separated (an os.Root path). As for the mode store, its presence is the
// managed-only property: the writer requires it and never creates it.
const TierRelPath = ".abcd/.work.local"

// FileRelPath is the checklist, one per working copy (dcn-2610180913444971),
// and LastGoodRelPath the copy only a checked write produces
// (spc-2610230805429429, "The write").
const (
	FileRelPath     = TierRelPath + "/checklist.json"
	LastGoodRelPath = TierRelPath + "/checklist.last-good.json"
)

// storePerm is the files' mode: the tier's other stores are written 0600, and
// a session's checklist is no wider than they are.
const storePerm fs.FileMode = 0o600

// maxReadBytes caps the guarded read. It is not a size limit on the checklist,
// which the spec refuses to set ("A write is never refused for the file's
// total size"); it only keeps a device or a runaway file planted at the path
// from being read into memory, and sits far above any list a session writes.
const maxReadBytes = 64 << 20

// tempPrefix is the name fsutil.WriteFileAtomicInRoot gives its temp file, so a
// writer killed before its rename leaves a file of this name in the tier.
const tempPrefix = ".abcd-tmp-"

// tornAfter is how old a temp file must be before the next write removes it.
// The tier is shared with the other local stores, and a write running beside
// this one renames its temp file within moments, so only one older than this
// was cut off (the claim status store's tmpKeep, on the same reasoning).
const tornAfter = time.Minute

// ErrNoLocalTier is the write's refusal when the checkout has no local tier.
var ErrNoLocalTier = errors.New("checklist: no local-ephemeral tier")

// ErrTracked is the write's refusal when git tracks a checklist file: a write
// would change the committed file, and the reader refuses a tracked file, so
// every item written would be lost to the next read.
var ErrTracked = errors.New("checklist: git tracks the checklist in this checkout; untrack it with `git rm --cached` first")

// ownerOf and gitRun are the owner lookup and the git runner Read and the
// writes use, held as vars so a test can provoke another account's file and
// a git that fails inside a checkout.
var (
	ownerOf = fsutil.OwnerUID
	gitRun  = gitutil.Run
)

// ErrScannerUnavailable is the refusal when the native secret scanner cannot be
// built or runs degraded. Both the write and the read fail closed on it: a file
// that could not be checked is never written, and never shown.
var ErrScannerUnavailable = errors.New("checklist: the secret scanner is unavailable")

// SecretError is the write's refusal for a field the native scanner matched
// (adr-2610222026265719): the field's path and the pattern kind, never the
// match.
type SecretError struct {
	Field string
	Kind  string
}

func (e *SecretError) Error() string {
	return fmt.Sprintf("checklist: %s matches the secret pattern %s; nothing was written, and the match is not repeated here", e.Field, e.Kind)
}

// Write validates c, scans every string field with the native secret scanner,
// and writes it atomically to FileRelPath under an already-resolved checkout
// root; then it writes the same bytes as LastGoodRelPath (spc-2610230805429429
// step 1, "The write").
//
// Every refusal comes before any byte is written: a schema violation (a
// *FieldError), an absent tier (ErrNoLocalTier), a scanner that cannot vouch
// for the text (ErrScannerUnavailable), and a secret (a *SecretError). A
// refused write leaves both files as they were.
//
// Write takes no lock: a read-change-write goes through Update, which holds the
// checklist's lock across the read, the change and this write. A caller that
// reads, changes and calls Write itself loses a concurrent writer's item.
//
// Last-good is written from this call's own validated and scanned bytes, never
// copied from the file already on disk: that file may have been changed by
// anything other than this verb, and promoting it would let an unchecked edit
// become the copy the reader falls back to.
func Write(repoRoot string, c Checklist) error {
	if err := c.Validate(); err != nil {
		return err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("checklist: opening the checkout to write: %w", err)
	}
	defer root.Close()
	// Lstat, not Stat: a symlink standing in for the tier is not the tier.
	if fi, err := root.Lstat(TierRelPath); err != nil || !fi.IsDir() {
		return fmt.Errorf("%w: %s/ is not a directory in this checkout, and it is never created on the way to a write",
			ErrNoLocalTier, TierRelPath)
	}
	if tracked, err := trackedFiles(repoRoot); err != nil {
		return fmt.Errorf("checklist: asking git whether the checklist is tracked: %w", err)
	} else if len(tracked) > 0 {
		return ErrTracked
	}
	sc, err := newScanner(repoRoot)
	if err != nil {
		return err
	}
	if field, kind, err := scan(sc, c); err != nil {
		return err
	} else if field != "" {
		return &SecretError{Field: field, Kind: kind}
	}
	data, err := encode(c)
	if err != nil {
		return fmt.Errorf("checklist: encoding: %w", err)
	}
	removeTorn(root, time.Now())
	if err := fsutil.WriteFileAtomicInRoot(root, FileRelPath, data, storePerm); err != nil {
		return fmt.Errorf("checklist: writing %s: %w", FileRelPath, err)
	}
	if err := fsutil.WriteFileAtomicInRoot(root, LastGoodRelPath, data, storePerm); err != nil {
		return fmt.Errorf("checklist: %s was written, but %s was not updated: %w", FileRelPath, LastGoodRelPath, err)
	}
	return nil
}

// Source says which file a Read used.
type Source string

const (
	// FromFile is the checklist itself.
	FromFile Source = "checklist"
	// FromLastGood is the last copy a checked write produced, used because the
	// checklist itself failed.
	FromLastGood Source = "last-good"
	// FromNothing means neither file passed; the result carries no checklist.
	FromNothing Source = "nothing"
)

// Result is a Read's answer: the checklist (zero for FromNothing), the file it
// came from, and the reason as one sentence a render can print as it stands.
// The reason names files, fields and bundled pattern kinds, never a value.
//
// WHY the reason is fixed text: the session-start hook injects it into every
// session's context, where nothing taken from the repository may go
// (iss-2608241115201044, cli.go's session-start notices). So no clause carries
// an error's text, which can quote the tracked .abcd/config/pii.json or name
// the checkout's absolute path, nor a pattern kind that file declared; each
// names a file abcd defines and a cause in abcd's own words.
type Result struct {
	Checklist Checklist
	Source    Source
	Reason    string
}

// Read reads the checklist under an already-resolved checkout root. The reader
// is the enforcement (the spec's "Settled here"): a file written past the verb,
// by a shell or an editor, reaches it unchecked, so it validates and scans
// FileRelPath, and on any failure validates and scans LastGoodRelPath, and on
// a second failure answers FromNothing. It never writes, so a read never
// promotes a file to last-good. A temp file a killed writer left is never
// read: only the two named paths are.
func Read(repoRoot string) Result {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Result{Source: FromNothing, Reason: "No checklist to show: the checkout could not be opened."}
	}
	defer root.Close()
	switch tierState(root) {
	case tierAbsent:
		return Result{Source: FromNothing, Reason: "No checklist has been written in this working copy yet."}
	case tierNotDir:
		return Result{Source: FromNothing, Reason: "No checklist to show: .abcd/.work.local is not a plain directory in this checkout."}
	}
	tracked, trackedErr := trackedFiles(repoRoot)
	if trackedErr != nil {
		return Result{Source: FromNothing, Reason: "No checklist to show: git could not say whether the checklist files are tracked."}
	}
	sc, scanErr := newScanner(repoRoot)
	judge := func(rel string) (Checklist, string) {
		if tracked[rel] {
			return Checklist{}, "is tracked by git, so it came with the checkout rather than from a checked write in it"
		}
		return load(repoRoot, root, sc, scanErr, rel)
	}

	c, why := judge(FileRelPath)
	if why == "" {
		return Result{Checklist: c, Source: FromFile, Reason: "Read checklist.json, which validated and scanned clean."}
	}
	lg, lgWhy := judge(LastGoodRelPath)
	switch {
	case lgWhy == "":
		return Result{Checklist: lg, Source: FromLastGood, Reason: "Showing the last good copy, because checklist.json " + why +
			"; the copy was written by a checked write and validated and scanned clean again."}
	case why == absent && lgWhy == absent:
		return Result{Source: FromNothing, Reason: "No checklist has been written in this working copy yet."}
	}
	return Result{Source: FromNothing, Reason: "No checklist to show: checklist.json " + why +
		", and checklist.last-good.json " + lgWhy + "."}
}

// absent is load's answer for a file that is not there, compared by Read to
// say "nothing written yet" rather than list two absences.
const absent = "is absent"

// The tier's states, as tierState reads them.
const (
	tierPlain = iota
	tierAbsent
	tierNotDir
)

// tierState is whether each level of the local tier is a plain directory:
// Lstat on .abcd and then on the tier, so a symlink at either level is not
// followed to a file Write never wrote.
func tierState(root *os.Root) int {
	for _, rel := range []string{path.Dir(TierRelPath), TierRelPath} {
		fi, err := root.Lstat(rel)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return tierAbsent
		case err != nil || !fi.IsDir():
			return tierNotDir
		}
	}
	return tierPlain
}

// trackedFiles is which of the two files git's index holds, staged or
// committed, as paths relative to repoRoot. A file git tracks came with the
// clone, written by whoever committed it, and Read injects nothing a
// repository supplies. A directory with no .git at or above it has no index,
// so it tracks nothing; inside a checkout, any failure of git (absent, or
// refusing a checkout another account owns) is an error, since the files
// cannot then be vouched for.
func trackedFiles(repoRoot string) (map[string]bool, error) {
	out, err := gitRun(repoRoot, "ls-files", "-z", "--cached", "--", FileRelPath, LastGoodRelPath)
	if err != nil {
		if !underGit(repoRoot) {
			return nil, nil
		}
		return nil, err
	}
	tracked := map[string]bool{}
	for _, rel := range strings.Split(out, "\x00") {
		if rel != "" {
			tracked[rel] = true
		}
	}
	return tracked, nil
}

// underGit is whether repoRoot or a directory above it holds a .git entry,
// a directory or a gitfile alike.
func underGit(repoRoot string) bool {
	dir := filepath.Clean(repoRoot)
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// load reads one of the two files and answers the checklist, or why it cannot
// be used as a clause that follows the file's name.
func load(repoRoot string, root *os.Root, sc *scanner.Scanner, scanErr error, rel string) (Checklist, string) {
	// Write makes both files storePerm; a file group or others can read was
	// made some other way, most often checked out of a commit at the umask's
	// 0644, and is refused before its bytes are read.
	if fi, err := root.Lstat(rel); err == nil && fi.Mode().IsRegular() {
		if fi.Mode().Perm()&^storePerm != 0 {
			return Checklist{}, "is readable by others than its owner, so a checked write did not make it"
		}
		if uid, err := ownerOf(filepath.Join(repoRoot, filepath.FromSlash(rel))); err != nil || uid != uint32(os.Getuid()) {
			return Checklist{}, "is owned by another account, or its owner could not be read"
		}
	}
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxReadBytes)
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist):
		return Checklist{}, absent
	case errors.Is(err, fsutil.ErrNotRegular):
		return Checklist{}, "is not a regular file"
	case errors.Is(err, fsutil.ErrTooBig):
		return Checklist{}, fmt.Sprintf("is larger than the %d-byte read cap", maxReadBytes)
	default:
		return Checklist{}, "could not be read"
	}
	c, err := Decode(data)
	if err != nil {
		// WHY a fixed sentence: the decoder names an unknown or repeated key by
		// the file's own spelling, a key is as able to hold a token as a value
		// is, and a redaction still keeps a fingerprint of it (its head, tail
		// and length), which invariant 24 counts as a part of the secret.
		return Checklist{}, "does not decode as one v1 checklist (malformed JSON, or an unknown or repeated key)"
	}
	if err := c.Validate(); err != nil {
		var fe *FieldError
		if errors.As(err, &fe) {
			return Checklist{}, "fails the schema: " + fe.Field + " " + fe.Problem
		}
		return Checklist{}, "fails the schema"
	}
	if scanErr != nil {
		return Checklist{}, scannerDown
	}
	field, kind, err := scan(sc, c)
	switch {
	case err != nil:
		return Checklist{}, scannerDown
	case field != "" && bundledKinds()[kind]:
		return Checklist{}, "carries a secret in " + field + " (" + kind + ")"
	case field != "":
		return Checklist{}, "carries a secret in " + field + " (a pattern abcd does not ship)"
	}
	return c, ""
}

// scannerDown is load's clause for a scanner that cannot vouch for the text.
// The scanner's own reason quotes .abcd/config/pii.json, a tracked file, so
// it stays with `abcd lint`, which a person runs.
const scannerDown = "could not be checked for secrets, because the secret scanner is unavailable; `abcd lint` says why"

// bundledKinds is the set of pattern kinds abcd ships. A finding's kind
// outside it was named by the repository's scanner config (or an augmenter's
// rules), which the reason never repeats.
var bundledKinds = sync.OnceValue(func() map[string]bool {
	kinds := map[string]bool{}
	for _, p := range scanner.DefaultPatterns() {
		kinds[p.Kind] = true
	}
	return kinds
})

// newScanner builds the native scanner for repoRoot, with the repository's
// pattern overrides folded in, and refuses one that is degraded: the
// decide and decisions stores' fail-closed stance, since a checklist is
// re-injected into every later session.
func newScanner(repoRoot string) (*scanner.Scanner, error) {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScannerUnavailable, err)
	}
	if unavail, why := sc.Unavailable(); unavail {
		return nil, fmt.Errorf("%w: %s", ErrScannerUnavailable, why)
	}
	return sc, nil
}

// scan runs every string field through the scanner and answers the first
// field a secret pattern matched, with the pattern's kind, or "" when none did.
//
// The fields are scanned as one text, one field to a line, so a repository's
// opt-in augmenter runs once per checklist rather than once per field; a
// finding's line names its field. That mapping holds because Validate, which
// both callers run first, refuses a line break in every field.
func scan(sc *scanner.Scanner, c Checklist) (field, kind string, err error) {
	fields := c.stringFields()
	lines := make([]string, len(fields))
	for i, f := range fields {
		lines[i] = f.value
	}
	findings := sc.ScanText(strings.Join(lines, "\n"), "checklist")
	// A failed augmenter run degrades the scanner during the call.
	if unavail, why := sc.Unavailable(); unavail {
		return "", "", fmt.Errorf("%w: %s", ErrScannerUnavailable, why)
	}
	for _, f := range findings {
		if !isSecret(f) {
			continue
		}
		name := "a field"
		if f.Line >= 1 && f.Line <= len(fields) {
			name = fields[f.Line-1].name
		}
		return name, f.Kind, nil
	}
	return "", "", nil
}

// isSecret is the class the write refuses: a hard-fail finding or a token of
// any severity, but not an identity finding (a name, a home path, an address),
// which adr-2610222026265719 does not reach. It is the scanner's own line
// between a secret span and the rest (its hasSecret), drawn from the exported
// kind predicates.
func isSecret(f scanner.Finding) bool {
	return (f.Severity == scanner.SeverityHardFail || scanner.IsTokenKind(f.Kind)) && !scanner.IsIdentityKind(f.Kind)
}

// encode is the file's one spelling: indented, with a trailing newline, and
// without encoding/json's HTML escaping, which would turn a promise's "=>"
// into "=>" in a file a person may read.
func encode(c Checklist) ([]byte, error) {
	if c.Items == nil {
		c.Items = []Item{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// removeTorn removes, best effort, every temp file in the tier a killed writer
// left behind: the atomic writer's prefix, a regular file, and older than
// tornAfter. A younger one may belong to a write still running.
func removeTorn(root *os.Root, now time.Time) {
	dir, err := root.Open(TierRelPath)
	if err != nil {
		return
	}
	entries, err := dir.ReadDir(-1)
	dir.Close()
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), tempPrefix) {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() || now.Sub(info.ModTime()) <= tornAfter {
			continue
		}
		_ = root.Remove(path.Join(TierRelPath, e.Name()))
	}
}

// LockRelPath is the lock every read-change-write of the checklist holds.
const LockRelPath = TierRelPath + "/checklist.lock"

// lockTimeout bounds the wait for another writer's cycle, which is one read,
// one scan and two atomic writes. A var only so a test can shorten it.
var lockTimeout = 10 * time.Second

// ErrLockHeld is Update's refusal when another writer held the checklist's
// lock past the wait, or the lock file is not a plain file: nothing was
// written.
var ErrLockHeld = errors.New("checklist: another writer holds the checklist's lock, or its lock file is not a plain file; nothing was written")

// Update is the one read-change-write cycle: it reads the checklist as Read
// does, hands the result to change, and writes what change returns through
// Write, holding the checklist's lock across all three. Every writer that
// builds on what is there goes through it.
//
// WHY a lock: a session's verb, its stop hook and sub-agents sharing the
// worktree write one checklist, and two unlocked cycles that read the same
// list each write back their own item over the other's, losing one; a lost
// item is the failure the checklist exists to prevent. The lock is fsutil's
// one inter-process primitive (iss-129), the one `abcd decide` and the memory
// store take, never a lock of this package's own.
func Update(repoRoot string, change func(Result) (Checklist, error)) error {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("checklist: opening the checkout to write: %w", err)
	}
	defer root.Close()
	// Through the checkout's os.Root, so a .abcd swapped for a symlink cannot
	// carry the tier check or the lock file out of the checkout; Lstat, not
	// Stat, as Write judges the tier, and the lock file is never created in a
	// tier that is not there.
	if fi, err := root.Lstat(TierRelPath); err != nil || !fi.IsDir() {
		return fmt.Errorf("%w: %s/ is not a directory in this checkout, and it is never created on the way to a write",
			ErrNoLocalTier, TierRelPath)
	}
	if tracked, err := trackedFiles(repoRoot); err != nil {
		return fmt.Errorf("checklist: asking git whether the checklist is tracked: %w", err)
	} else if len(tracked) > 0 {
		return ErrTracked
	}
	ran := false
	err = fsutil.WithFileLockIn(root, LockRelPath, lockTimeout, func() error {
		ran = true
		c, err := change(Read(repoRoot))
		if err != nil {
			return err
		}
		return Write(repoRoot, c)
	})
	// fn's own error comes back unwrapped, so only an error before fn ran is
	// the lock's own.
	if !ran && (errors.Is(err, fsutil.ErrLockContention) || errors.Is(err, fsutil.ErrLockPathUnsafe)) {
		return ErrLockHeld
	}
	return err
}
