package fsutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/text/unicode/norm"
)

// ValidRelPath reports whether p is a safe repo-relative slash path: non-empty,
// not absolute, already clean, free of empty, ".", ".." or control-character
// segments, and containing no backslash. It is the canonical lexical guard for a
// path that arrives as data — a committed configuration value, a packed
// lifeboat's manifest entry — before it is joined onto a trusted root. Lexical
// only: it says nothing about symlinks, so a caller that then follows the path
// still needs the guarded read/write primitives.
//
// A backslash is refused outright, on every host. path.Clean is slash-only, so
// it leaves a backslash untouched: `..\..\x` is "clean" and slash-traversal-free,
// yet on a Windows target (abcd cross-compiles Windows binaries) filepath.FromSlash
// turns those backslashes into separators and the join walks out of the intended
// root. A legitimate committed repo-relative path is always a forward-slash path
// and never contains a backslash, so refusing it everywhere closes the Windows
// escape while keeping the gate identical across platforms (iss-2608280807166510).
func ValidRelPath(p string) bool {
	if p == "" || path.IsAbs(p) || strings.HasPrefix(p, "/") {
		return false
	}
	if strings.ContainsRune(p, '\\') {
		return false
	}
	if p != path.Clean(p) {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for _, r := range seg {
			if r < 0x20 || r == 0x7f {
				return false
			}
		}
	}
	return true
}

// InsideGitDir reports whether the repo-relative path p names the git directory
// or anything under it.
//
// ValidRelPath accepts ".git/config" — it is clean, relative, and inside the
// containment root — so a configuration value that arrives as data and is then
// READ AND PUBLISHED needs this second gate on top of it: .git/config carries a
// credential-bearing remote URL (on a CI runner, the checkout token in an
// `http.…extraheader` line), and nothing under .git is ever a legitimate source
// of rendered text. The first segment is compared case-insensitively, because a
// case-folding filesystem reaches the same .git through ".GIT" (iss-150).
//
// Only the leading segment is examined, so ".github/workflows/ci.yml" and
// ".gitignore" — different directories that merely start with the same bytes —
// are not caught.
func InsideGitDir(p string) bool {
	first, _, _ := strings.Cut(p, "/")
	return strings.EqualFold(first, ".git")
}

// CaseFoldingFS reports whether the platform's default filesystem folds case.
// macOS (APFS/HFS+ default) and Windows do; abcd assumes this default rather
// than probing each volume, and the only cost of a false assumption is a
// stricter comparison.
func CaseFoldingFS() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}

// caseFoldingFS is the package's own view of CaseFoldingFS, held as a var for the
// same reason launch and lifeboat keep one: a redactor's case-folding branch
// cannot be provoked on a case-SENSITIVE host, so substituting the predicate is
// the only way a test can prove a case-variant root is redacted — and, with it
// false, that a case-sensitive host keeps exact-match semantics.
var caseFoldingFS = CaseFoldingFS

// FoldPath returns p in the spelling a path comparison uses: lower-cased AND
// Unicode-normalised (NFC) when fold is set, unchanged otherwise. It is the single
// place a case-folding comparison key is minted, so a containment gate and a
// duplicate-target map fold identically rather than drifting apart.
//
// A case-folding filesystem (APFS/HFS+, Windows) is ALSO normalisation-
// insensitive: the NFC and NFD spellings of one name address the same file, so a
// key that folds only case leaves the NFC/NFD half of the equivalence class open —
// the payload-destination gate, the pack overlap gate and embark's claimed key all
// then read two spellings of one directory as distinct (iss-2608270926030978). NFC
// is applied AFTER the lower-casing so the key is always in composed form. When
// fold is false the host is case-sensitive and normalisation-sensitive (Linux), so
// the key stays byte-exact and two distinct spellings remain distinct.
//
// fold is a parameter rather than a call to CaseFoldingFS because a fail-closed
// gate's case-folding branch cannot be provoked on a case-sensitive host: the
// caller passes its own predicate, and a test passes the branch it means to
// prove.
func FoldPath(p string, fold bool) string {
	if fold {
		return norm.NFC.String(strings.ToLower(p))
	}
	return p
}

// PathWithin reports whether child is equal to or nested inside parent. With
// fold set the comparison is case-insensitive: otherwise a destination like
// ".../REPO/dist" computes as an out-of-tree sibling of ".../repo" and slips a
// containment gate, even though the two name the SAME directory on a
// case-folding filesystem — and the write then lands inside the tree the gate
// exists to protect. Erring toward "within" is the safe direction for a
// destructive-write gate (iss-2608270500192615).
//
// It is the canonical containment comparison: the launch payload-destination
// gate and the lifeboat pack overlap gate both route through it rather than
// keeping divergent prefix tests.
func PathWithin(child, parent string, fold bool) bool {
	child = FoldPath(child, fold)
	parent = FoldPath(parent, fold)
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// PathsOverlap reports whether a and b are the same directory or one contains
// the other — either way a write under one touches the other.
func PathsOverlap(a, b string, fold bool) bool {
	return PathWithin(a, b, fold) || PathWithin(b, a, fold)
}

// RealExistingPath returns p made absolute with its longest EXISTING prefix
// symlink-resolved and the absent remainder rejoined lexically. A destination
// that does not exist yet still resolves through its real parent, so a
// containment gate compares real locations rather than lexical strings a
// symlinked ancestor could disguise — and on a platform whose temp root is
// itself a symlink (/var -> /private/var on macOS) an unresolved comparison
// would silently miss an overlap. An empty p is returned empty rather than
// resolved to the working directory.
//
// It is the canonical existing-prefix resolver: the launch payload-destination
// gate, the lifeboat pack destination gate and the site output-directory gate
// all route through it rather than keeping divergent copies. It does not
// case-canonicalise (Go's darwin EvalSymlinks keeps the caller's spelling for
// every non-symlink component), so compare its results with PathWithin under
// CaseFoldingFS.
func RealExistingPath(p string) string {
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	rest := ""
	cur := abs
	for {
		if real, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(real, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// RepoRel renders target as a path relative to base — the repo root, or the
// working directory — so machine output never carries an absolute
// developer-identity path (iss-81). A target outside base yields a "../…" form,
// which is acceptable: the contract is only that the result is never an absolute
// /Users/<name>/… path. It falls back to the base name when a relative form
// cannot be computed (a different volume, a relative target, or an empty base)
// and to target unchanged when target is already relative. A non-path value that
// merely looks absolute-free (e.g. a URL) is returned untouched.
func RepoRel(base, target string) string {
	if target == "" {
		return target
	}
	if base != "" {
		if rel, err := filepath.Rel(base, target); err == nil {
			return rel
		}
	}
	if filepath.IsAbs(target) {
		return filepath.Base(target)
	}
	return target
}

// RedactRoot replaces every occurrence of the absolute directory root in s with
// repl — both a path UNDER the root (root + separator + …) and the BARE root
// itself when it sits at a right boundary (end of string or a non-path
// character). The bare-root case matters because a message that names exactly
// $HOME (e.g. "cannot access /Users/alex") would otherwise leak the developer abcd-audit:allow
// identity — its base segment IS the username. The filesystem root ("/") and
// empty or relative roots are skipped so a message is never mangled.
//
// It is the one statement of what a developer-identity root looks like in
// rendered output: the CLI's error scrub redacts the working directory and the
// home directory out of every command error, and `ahoy install`'s receipt
// redacts the same two roots out of every write it reports (iss-177). Callers
// pass the root and the replacement because the polarity differs — "." for the
// repo, "~" for home — but the boundary rule must not.
//
// On a case-folding filesystem the match is case-insensitive: a message that
// echoes $HOME in a case variant (as a shell or a syscall may on macOS/APFS)
// names the same directory, so the identity must still be scrubbed. The original
// casing of any UNREDACTED text is preserved, and on a case-sensitive host the
// match stays byte-exact so two distinct paths are never merged (iss-2608270908341622).
func RedactRoot(s, root, repl string) string {
	if len(root) <= 1 || !filepath.IsAbs(root) {
		return s
	}
	fold := caseFoldingFS()
	sep := string(os.PathSeparator)
	s = replaceAllFold(s, root+sep, repl+sep, fold)
	return replaceBareRoot(s, root, repl, fold)
}

// indexFold is strings.Index, comparing case-insensitively when fold is set so a
// case-variant root spelling is still located on a case-folding filesystem. The
// scan advances one byte at a time and compares equal-length windows with
// EqualFold, so the returned index always addresses the ORIGINAL string — a
// ToLower of the whole haystack could shift byte offsets when a fold changes a
// rune's width.
func indexFold(s, sub string, fold bool) int {
	if !fold {
		return strings.Index(s, sub)
	}
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if strings.EqualFold(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}

// replaceAllFold replaces every occurrence of old in s with repl, matching
// case-insensitively when fold is set. The matched span is dropped and repl
// written in its place; the surrounding text keeps its original casing.
func replaceAllFold(s, old, repl string, fold bool) string {
	if old == "" {
		return s
	}
	var b strings.Builder
	for {
		i := indexFold(s, old, fold)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		b.WriteString(repl)
		s = s[i+len(old):]
	}
}

// RedactHome replaces the user's home directory in s with "~" wherever it
// appears — the home-polarity shortcut over RedactRoot for a success or receipt
// envelope that carries an absolute path the CLI error scrub (which runs only on
// the error value) never sees. A home-resolution failure or a root-level home
// leaves s unchanged, so the primitive is never worse than the raw string.
func RedactHome(s string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return s
	}
	return RedactRoot(s, filepath.Clean(home), "~")
}

// replaceBareRoot replaces occurrences of root that end at a path boundary (end
// of string or a character that cannot continue a path segment), leaving a longer
// path that merely shares this prefix untouched. With fold set the match is
// case-insensitive; a span that is not redacted is re-emitted in its own original
// casing (the matched bytes), never rewritten to root's spelling.
func replaceBareRoot(s, root, repl string, fold bool) string {
	var b strings.Builder
	for {
		i := indexFold(s, root, fold)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		after := i + len(root)
		matched := s[i:after]
		b.WriteString(s[:i])
		if after >= len(s) || isPathBoundary(s[after]) {
			b.WriteString(repl)
		} else {
			b.WriteString(matched)
		}
		s = s[after:]
	}
}

// isPathBoundary reports whether c cannot be part of a path segment, so a root
// immediately followed by c is a whole path rather than a prefix of a longer one.
func isPathBoundary(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return false
	case c == '/' || c == '.' || c == '-' || c == '_':
		return false
	}
	return true
}

// notPresent reports whether a stat/open error means the path cannot exist: it
// is absent (ErrNotExist), or a component of its prefix is not a directory
// (ENOTDIR, e.g. asking about a/b where a is a regular file). Both are "not
// present", not a filesystem fault, so a fail-closed caller must not abort on
// them.
func notPresent(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// Exists reports whether path exists, following symlinks — so a link to a real
// file exists and a dangling link does not. A stat error other than not-exist is
// returned rather than swallowed, so a caller checking a convention fails closed
// on a permission error instead of silently reporting "absent".
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if notPresent(err) {
		return false, nil
	}
	return false, err
}

// ExistsNoFollow reports whether a filesystem entry occupies path, NEVER
// following a symlink at the leaf — so a dangling symlink still exists: the
// name is taken, and whatever occupies it (including a link whose target
// string is itself sensitive) is what a placement check must see. Ancestor
// components are resolved by the kernel as usual. Error handling mirrors
// Exists: not-present is false with no error, anything else fails closed.
func ExistsNoFollow(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if notPresent(err) {
		return false, nil
	}
	return false, err
}

// IsDir reports whether path exists and is a directory. An absent path is false
// with no error; any other stat error is returned (fail closed).
//
// It follows symlinks: use IsRealDir where a symlinked directory must read as
// false (the owned-store guard).
func IsDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		return fi.IsDir(), nil
	}
	if notPresent(err) {
		return false, nil
	}
	return false, err
}

// DirHasEntries reports whether path is a directory holding at least one entry,
// dotfiles included — a directory kept alive by a lone .gitkeep is not empty.
//
// An absent path is false with no error: "missing" and "empty" are distinct
// conditions, and pairing this with Exists lets a presence rule and a non-empty
// rule report independently rather than one masking the other. A path that
// exists but is not a directory is likewise false with no error.
func DirHasEntries(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if notPresent(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	names, err := f.Readdirnames(1)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil // an empty directory
		}
		// Readdirnames on a non-directory errors; that is "no entries", not a
		// broken filesystem, so it stays a soft false.
		if isDir, dirErr := IsDir(path); dirErr == nil && !isDir {
			return false, nil
		}
		return false, err
	}
	return len(names) > 0, nil
}

// ModuleRoot walks up from start until it finds the directory holding go.mod —
// the module root.
//
// It exists for the repo's code generators, which must write to the same file
// whether they are invoked by `go generate` (the working directory is the package
// being generated) or run directly from the repo root. Anchoring on go.mod rather
// than on .git means a generator also works inside a worktree or an export where
// the git directory is not where the walk expects it.
//
// A start outside any module is an error rather than a fallback to the working
// directory: a generator that silently wrote its artefact into an unrelated
// directory would be worse than one that refuses.
func ModuleRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("fsutil: go.mod not found at or above %s", start)
		}
		dir = parent
	}
}

// ErrOwnerUnknown is OwnerUID's sentinel for a path that exists but whose owning
// uid the platform did not report. abcd builds only darwin and linux, where
// os.FileInfo.Sys() always carries a *syscall.Stat_t, so it cannot arise there —
// it exists so a fail-closed caller has a named condition to refuse on rather
// than an "ok" it never earned.
var ErrOwnerUnknown = errors.New("fsutil: owning uid unavailable")

// OwnerUID reports the uid that owns path, following symlinks (the caller has
// usually resolved the path already, and a link's own uid is not the uid that
// controls its content). It is the canonical ownership lookup: code asking "is
// this directory mine?" compares this against os.Getuid() instead of reaching
// for syscall.Stat_t itself, so exactly one place knows the platform's stat
// shape.
//
// The error is returned rather than folded into a boolean because ownership is
// a fail-closed question: a caller that cannot learn the owner must be able to
// tell that from learning the owner is itself, and a (uid, bool) shape makes the
// two look alike at the call site.
func OwnerUID(path string) (uint32, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, ErrOwnerUnknown
	}
	return uint32(st.Uid), nil
}
