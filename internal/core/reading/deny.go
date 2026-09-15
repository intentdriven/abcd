package reading

import (
	"path"
	"strings"
)

// The default-deny stance, on the shape internal/core/launch carries for the
// release payload: a structural deny no include can promote, binding EVERY path
// component case-insensitively rather than the first segment alone.
//
// The one difference from launch's is where the deny is measured FROM. Here it
// is measured from the include row's own Source downward, which is what makes
// assembler rule 1 mechanical rather than remembered: a row naming the
// repository root cannot reach `.abcd` at any depth, while a row naming a
// record family's leaf bucket individually escapes the deny precisely because
// the denied component is above it, not below.

// denySegments are path components no assembly may descend through. `.abcd`
// covers the whole record — every family, the working tier, and the local tier
// — so a record path a reading legitimately needs must be named individually.
// `agents` holds the reading definitions, `evals` the evals that guard this
// assembler, and `.git` the history no reading receives.
var denySegments = []string{".git", ".abcd", "agents", "evals"}

// denyPrefixes are repo-relative paths, and everything beneath them, that no
// assembly may pass. The assembler's own package is the whole of the list: a
// reading must never receive the include table that decides what it sees
// (itd-183, "the shipped tree" definition; ruling (18)).
var denyPrefixes = []string{"internal/core/reading"}

// segmentDenied reports whether one path component is a denied namespace.
func segmentDenied(seg string) bool {
	for _, denied := range denySegments {
		if strings.EqualFold(seg, denied) {
			return true
		}
	}
	return false
}

// pathContainsDeniedSegment reports whether any component of rel is denied.
func pathContainsDeniedSegment(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if segmentDenied(seg) {
			return true
		}
	}
	return false
}

// prefixDenied reports whether a repo-relative path lies at or beneath a denied
// prefix.
// It folds case, like segmentDenied beside it. It did not, and this file's own
// header claimed the deny "binds EVERY path component case-insensitively" while
// half of it did not — so `Internal/Core/Reading` was accepted where
// `internal/core/reading` was refused (iss-2608312019544150). Nothing was
// admitted by that: the walk emits the on-disk case and trackedSet intersects
// with git. But itd-199 gave the lexical claim a second consumer,
// validPresetPath, whose stated contract is that a path which LOOKS like it
// reaches a denied place is refused at the door rather than silently selecting
// nothing — and on a case-insensitive filesystem that claim was the only thing
// holding.
func prefixDenied(rel string) bool {
	for _, p := range denyPrefixes {
		if strings.EqualFold(rel, p) || strings.HasPrefix(strings.ToLower(rel), strings.ToLower(p)+"/") {
			return true
		}
	}
	return false
}

// matches reports whether a basename satisfies either of the row's two match
// forms. MatchSuffix is a basename suffix, matched case-sensitively; in Match,
// an entry beginning with "." is an extension and any other entry is an exact
// basename. The two are ORed, and only a row declaring NEITHER admits every
// file.
func (r Row) matches(base string) bool {
	// Both forms empty admits every file, which no row uses. A row that
	// declares only MatchSuffix must NOT fall through to that: an empty Match
	// beside a non-empty MatchSuffix means the extension/basename form
	// contributes nothing, not that everything is admitted (spc-68).
	if len(r.Match) == 0 && len(r.MatchSuffix) == 0 {
		return true
	}
	// Case-sensitive by construction: the Go toolchain builds a test only from
	// a lowercase _test.go, so folding case here would label material a test
	// that Go does not build as one. This deliberately differs from the
	// extension form below, which folds. See Row.MatchSuffix for the edges this
	// suffix does not capture; all of them are labelling, never admission.
	for _, s := range r.MatchSuffix {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	ext := path.Ext(base)
	for _, m := range r.Match {
		if strings.HasPrefix(m, ".") {
			if strings.EqualFold(ext, m) {
				return true
			}
			continue
		}
		if base == m {
			return true
		}
	}
	return false
}

// Reaches reports whether the row admits the repo-relative file at rel. It is
// the predicate the walk applies and the predicate the table's own rule tests
// assert against, so the charter cannot claim a bound the walk does not keep.
func (r Row) Reaches(rel string) bool {
	rel = path.Clean(strings.TrimPrefix(rel, "./"))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") {
		return false
	}
	sub := rel
	if r.Source != "." {
		src := path.Clean(r.Source)
		if !strings.HasPrefix(rel, src+"/") {
			return false
		}
		sub = rel[len(src)+1:]
	}
	if pathContainsDeniedSegment(sub) || prefixDenied(rel) {
		return false
	}
	return r.matches(path.Base(rel))
}

// Admits reports whether any row of the table admits rel at position p. It is
// the whole of the assembler's answer to "may a reading see this file".
func Admits(p Position, rel string) bool {
	for _, row := range Table {
		if row.AdmittedAt(p) && row.Reaches(rel) {
			return true
		}
	}
	return false
}
