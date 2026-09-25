package guard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// RepoRelPath is the per-repo override file, relative to the repo worktree. It
// is a dedicated committed file rather than a rules.json domain: the entry schema
// is structured, and the rules loader's kill switch must never silently disable a
// safety guard (spc-16, "Config home"). Disabling the guard is itself a
// committed, reviewable act.
const RepoRelPath = ".abcd/guard.json"

// maxGuardFileBytes caps the per-repo override (trust boundary).
const maxGuardFileBytes = 256 * 1024

// Load returns the bundled defaults merged with <repoRoot>/.abcd/guard.json when
// that file exists. An absent file yields the defaults unchanged.
//
// A repo override that cannot be read, parsed, or validated is a fail-SAFE
// fallback, not a fail-open one: Load returns the bundled defaults ALONGSIDE the
// error (never an empty registry), so a broken repo layer drops only the repo's
// own overrides while the built-in hazards stay armed. Returning empty here would
// disable the whole guard on one broken committed file — the fail-open the
// bundled hazards never depended on (iss-2608261551087492). The error is still
// returned so the caller (the hook shim) can announce the dropped repo layer
// loudly while continuing to check against the bundled registry.
func Load(repoRoot string) (Registry, error) {
	// Refuse a symlinked .abcd directory component before touching the leaf, so a
	// swapped .abcd cannot redirect the read (trust boundary).
	if di, err := os.Lstat(filepath.Join(repoRoot, ".abcd")); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return Defaults(), fmt.Errorf("%w: .abcd is a symlink (refusing to follow)", ErrMalformedConfig)
	}
	data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, ".abcd", "guard.json"), maxGuardFileBytes)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return Defaults(), nil
		case errors.Is(err, syscall.ELOOP):
			return Defaults(), fmt.Errorf("%w: %s is a symlink (refusing to follow)", ErrMalformedConfig, RepoRelPath)
		case errors.Is(err, fsutil.ErrNotRegular):
			return Defaults(), fmt.Errorf("%w: %s is not a regular file", ErrMalformedConfig, RepoRelPath)
		case errors.Is(err, fsutil.ErrTooBig):
			return Defaults(), fmt.Errorf("%w: %s exceeds the %d-byte cap", ErrMalformedConfig, RepoRelPath, maxGuardFileBytes)
		default:
			return Defaults(), fmt.Errorf("%w: reading %s failed", ErrMalformedConfig, RepoRelPath)
		}
	}
	over, err := parse(data)
	if err != nil {
		return Defaults(), fmt.Errorf("%s: %w", RepoRelPath, err)
	}
	// The override declares its schema version explicitly: an absent version is a
	// truncated or hand-mangled file, not an invitation to guess (a safety config
	// fails closed on unrecognised input).
	if over.SchemaVersion != SchemaVersion {
		return Defaults(), fmt.Errorf("%w: %s must declare schema_version %d, got %d", ErrSchemaVersion, RepoRelPath, SchemaVersion, over.SchemaVersion)
	}
	merged := Merge(Defaults(), over)
	if err := Validate(merged); err != nil {
		return Defaults(), fmt.Errorf("%s: %w", RepoRelPath, err)
	}
	// Weakening the guard is a committed, reviewable act (spc-16). The file is
	// read from the working tree, so without this an agent's uncommitted write
	// of `"disabled": true` — a write the guard itself allows — switched the
	// guard off on the very next command (iss-147). An edit that weakens the
	// registry against what HEAD carries is refused, and the committed registry
	// stays in force until the edit is committed.
	committed := committedRegistry(repoRoot)
	if what := weakening(committed, merged); what != "" {
		return committed, fmt.Errorf("%w: %s %s, and HEAD does not carry that edit; the committed registry is in force until it is committed", ErrUncommittedOverride, RepoRelPath, what)
	}
	return merged, nil
}

// committedRegistry is the registry HEAD's .abcd/guard.json produces: the
// bundled defaults merged with the committed file. Any state in which the committed file cannot be read — no repository, no commit,
// no such file in HEAD, a git that does not answer, a committed file that does
// not load — is the bundled defaults alone, the registry that needs no review.
func committedRegistry(repoRoot string) Registry {
	listed, err := gitutil.Run(repoRoot, "ls-tree", "-z", "--name-only", "HEAD", "--", RepoRelPath)
	if err != nil || strings.Trim(listed, "\x00") == "" {
		return Defaults()
	}
	blob, err := gitutil.RunLimited(repoRoot, maxGuardFileBytes, "cat-file", "blob", "HEAD:./"+RepoRelPath)
	if err != nil {
		return Defaults()
	}
	over, err := parse([]byte(blob))
	if err != nil || over.SchemaVersion != SchemaVersion {
		return Defaults()
	}
	merged := Merge(Defaults(), over)
	if Validate(merged) != nil {
		return Defaults()
	}
	return merged
}

// weakening names the first way next is weaker than base, or returns "" when
// it is not: switching the guard off, or changing a blocker's tier or pattern.
// A new entry, a stricter tier, and a changed why, successor or fixture set
// strengthen or merely reword, and take effect uncommitted.
func weakening(base, next Registry) string {
	if next.Disabled && !base.Disabled {
		return "switches the guard off"
	}
	ids := make([]string, 0, len(base.Entries))
	for id := range base.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		b := base.Entries[id]
		if b.Tier != TierBlocker {
			continue
		}
		n, ok := next.Entries[id]
		switch {
		case !ok:
			return fmt.Sprintf("removes the blocker %s", id)
		case n.Tier != TierBlocker:
			return fmt.Sprintf("retiers the blocker %s to %s", id, n.Tier)
		case !reflect.DeepEqual(n.Pattern, b.Pattern):
			return fmt.Sprintf("changes the pattern of the blocker %s", id)
		}
	}
	return ""
}

// LoadPosture is what a load produced, and so what a front door owes the
// session. It is decided here, once, so the hook, the check verb, the health
// report and any later surface format the same answer rather than each
// re-deriving it from the registry's size (iss-2608291814576261).
type LoadPosture uint8

const (
	// LoadClean: every layer loaded, or there was no repo layer to load.
	LoadClean LoadPosture = iota
	// LoadRepoDropped: the repo layer was refused — unreadable, invalid, or an
	// uncommitted weakening edit — and the registry holds what could be trusted:
	// the bundled hazards, plus the committed repo layer when only a
	// working-tree edit was refused. It is armed. A session keeps checking
	// against it and is told, loudly, that the repo layer was dropped; a caller
	// that asked a question (the check verb) is told the registry it asked about
	// is not the one in force.
	LoadRepoDropped
	// LoadUnavailable: there is no registry to check against at all. The
	// embedded defaults make it unreachable in practice; a session reaching it
	// runs unguarded, and must say so.
	LoadUnavailable
)

// Loaded is the typed result of loading a repo's hazard registry: the registry
// to check against, the posture the load ended in, and the error behind any
// posture but LoadClean.
type Loaded struct {
	Registry Registry
	Posture  LoadPosture
	Err      error
}

// LoadRepo loads <repoRoot>'s registry and decides the fail-safe posture.
func LoadRepo(repoRoot string) Loaded {
	reg, err := Load(repoRoot)
	return Loaded{Registry: reg, Posture: postureOf(reg, err), Err: err}
}

// postureOf is the fail-safe policy itself: a registry with nothing in it is
// unavailable whatever the error, and an error beside an armed registry is a
// dropped repo layer.
func postureOf(reg Registry, err error) LoadPosture {
	switch {
	case len(reg.Entries) == 0:
		return LoadUnavailable
	case err != nil:
		return LoadRepoDropped
	default:
		return LoadClean
	}
}

// Merge overlays over onto base. Entry fields are per-field: a field set on the
// override wins, an absent field inherits the bundled entry (so {"tier":"warn"}
// retiers an entry while keeping its pattern, successor, and why). New entry keys
// are added. The kill switch is sticky — either side can disable the guard, and
// neither can silently re-enable the other's refusal to run it.
func Merge(base, over Registry) Registry {
	out := cloneRegistry(base)
	if over.SchemaVersion != 0 {
		out.SchemaVersion = over.SchemaVersion
	}
	out.Disabled = base.Disabled || over.Disabled
	if out.Entries == nil && len(over.Entries) > 0 {
		out.Entries = make(map[string]Entry, len(over.Entries))
	}
	for id, oe := range over.Entries {
		out.Entries[id] = mergeEntry(out.Entries[id], oe)
	}
	return out
}

func mergeEntry(base, over Entry) Entry {
	r := cloneEntry(base)
	r.ID = over.ID
	if over.Tier != "" {
		r.Tier = over.Tier
	}
	if over.Successor != "" {
		r.Successor = over.Successor
	}
	if over.Why != "" {
		r.Why = over.Why
	}
	if over.Fixtures.KnownBad != nil {
		r.Fixtures.KnownBad = append([]string(nil), over.Fixtures.KnownBad...)
	}
	if over.Fixtures.KnownGood != nil {
		r.Fixtures.KnownGood = append([]string(nil), over.Fixtures.KnownGood...)
	}
	r.Pattern = mergePattern(r.Pattern, over.Pattern)
	return r
}

func mergePattern(base, over Pattern) Pattern {
	r := base
	if over.Command != "" {
		r.Command = over.Command
	}
	if over.Subcommand != "" {
		r.Subcommand = over.Subcommand
	}
	if over.Subcommand2 != "" {
		r.Subcommand2 = over.Subcommand2
	}
	if over.ValueFlags != nil {
		r.ValueFlags = append([]string(nil), over.ValueFlags...)
	}
	if over.Flags != nil {
		r.Flags = append([]string(nil), over.Flags...)
	}
	if over.ArgPrefixes != nil {
		r.ArgPrefixes = append([]string(nil), over.ArgPrefixes...)
	}
	if over.FlagValues != nil {
		r.FlagValues = cloneFlagValues(over.FlagValues)
	}
	if over.ArgPaths != nil {
		r.ArgPaths = append([]PathArg(nil), over.ArgPaths...)
	}
	// AfterCD is a pointer precisely so an override can set it to false — a
	// bool field could only ever tighten the requirement, never lift it.
	if over.AfterCD != nil {
		v := *over.AfterCD
		r.AfterCD = &v
	}
	return r
}
