package fsutil

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// ErrHomeScopeSymlinked is the refusal for a home-scoped path whose DIRECTORY
// is a symbolic link: ~/.abcd itself, or a directory below it on the way to the
// file. It is the one rule every reader and writer of the caller's machine
// scope applies, stated for the rules loader in AGENTS.md: a dotfiles-symlinked
// ~/.abcd can never host a file abcd trusts. HomeScopeLink returns it wrapped
// in a *HomeScopeLinkError, so errors.Is finds it.
var ErrHomeScopeSymlinked = errors.New("fsutil: a directory of this home-scoped path is a symlink")

// HomeScopeLinkError names the symlinked directory in tilde form. Its message
// is the whole operator-facing sentence, the remedy included, so every reader
// and writer that refuses says the same thing.
type HomeScopeLinkError struct {
	// Link is the symlinked directory in tilde form ("~/.abcd").
	Link string
}

func (e *HomeScopeLinkError) Error() string {
	return e.Link + " is a symlink, which abcd refuses rather than follows; replace the link with a real directory to keep abcd's files there"
}

func (e *HomeScopeLinkError) Unwrap() error { return ErrHomeScopeSymlinked }

// HomeScopeLink is the check behind that rule. rel is a slash path relative to
// home (".abcd/path-entry", ".abcd/credentials.json"); every DIRECTORY
// component of it — ~/.abcd first, never home itself and never the leaf, whose
// own guard is the reader's or the writer's — is Lstat'd, and the first that
// is a symlink is refused with a *HomeScopeLinkError naming it in tilde form
// (wrapping ErrHomeScopeSymlinked). A component that is absent, or that this
// uid cannot search, is not a link and ends the walk: nothing below it can be
// reached through a link that is not there.
//
// It exists because every home-scoped primitive guards the LEAF (O_NOFOLLOW,
// an Lstat of the file) and resolves the directories above it through the
// kernel, which follows a link without comment. A ~/.abcd symlinked into a
// dotfiles checkout therefore hosted a path-entry that decides which binary the
// hook shims execute, a cache attestation that decides which binary is promoted
// onto PATH, and a credentials.json written into a repository — while the rules
// loader beside them refused its rules.json there (iss-2609281017573862,
// iss-2609260958587561). One check, called by all of them, is what keeps the
// rule from drifting per reader.
//
// home itself is deliberately not judged: a home directory reached through a
// link (/home -> /usr/home, a relocated account) is the machine's layout, not a
// declaration the caller could have made somewhere else, and refusing it would
// refuse every account laid out that way.
//
// rel must be a clean relative path (ValidRelPath); anything else is refused
// with an *os.PathError wrapping os.ErrInvalid rather than judged. An escaping
// rel ("../x/f") leaves home before any directory below it is reached, and an
// absolute one would judge home itself, so passing either would vouch for a
// path the walk never covered.
func HomeScopeLink(home, rel string) error {
	if !ValidRelPath(rel) {
		return &os.PathError{Op: "homescopelink", Path: rel, Err: os.ErrInvalid}
	}
	dir := path.Dir(rel)
	if dir == "." {
		return nil
	}
	cur := home
	shown := "~"
	for _, part := range strings.Split(dir, "/") {
		cur = filepath.Join(cur, part)
		shown += "/" + part
		fi, err := os.Lstat(cur)
		if err != nil {
			return nil
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return &HomeScopeLinkError{Link: shown}
		}
	}
	return nil
}

// ReadHomeDeclaration is ReadDeclaration for a declaration file named by its
// place in the caller's home: home joined with rel (a slash path,
// ".abcd/trusted-roots"). It is the read every home-scoped declaration goes
// through, and it adds the one fact ReadDeclaration cannot see from a path —
// that no directory between home and the file is a symlink (HomeScopeLink).
//
// The order keeps AGENTS.md's rule exactly: a file that is not there is
// DeclarationAbsent whatever the directories are, so a symlinked ~/.abcd
// holding no such file reads as absent and costs its owner nothing; a file
// that IS there behind a symlinked directory is DeclarationBehindSymlink,
// refused before a byte of it is read. The Lstat that decides absence follows
// the directories, which is what lets a file behind a link be found in order to
// be refused.
//
// A rel that is not a clean relative path is DeclarationUnreadable before
// anything is looked at: it names no place in the home to read.
func ReadHomeDeclaration(home, rel string, limit int64) ([]byte, DeclarationRefusal, error) {
	if !ValidRelPath(rel) {
		return nil, DeclarationUnreadable, &os.PathError{Op: "readhomedeclaration", Path: rel, Err: os.ErrInvalid}
	}
	p := filepath.Join(home, filepath.FromSlash(rel))
	if _, err := os.Lstat(p); err != nil {
		return nil, DeclarationAbsent, err
	}
	if err := HomeScopeLink(home, rel); err != nil {
		return nil, DeclarationBehindSymlink, err
	}
	return ReadDeclaration(p, limit)
}

// HomeDeclarationNames reads the line-oriented path declaration at rel under
// home through ReadHomeDeclaration and reports whether one of its entries names
// target. It is the one reader behind every "declare this checkout" opt-in —
// ~/.abcd/trusted-roots for the rules resolver, ~/.abcd/local-transcript-roots
// for the transcript store — so a hardening of what a declaration must be, or
// of how an entry is matched, lands once and reaches every caller
// (iss-2609090951283654).
//
// An entry is a trimmed line that is neither blank nor a "#" comment and is
// absolute: a relative entry names a different directory per caller and names
// nothing here. Each entry is compared in two spellings, as written and
// symlink-resolved, against target in the same two spellings, because a
// declared path need not be resolved and a caller's target may be either.
// The comparison key is FoldPath under fold; fold is a parameter, not a call to
// CaseFoldingFS, so each caller holds the predicate in a variable a test can
// force and the case-folding branch is provable on a case-sensitive host
// (iss-2609090951297149).
//
// ignored is empty when there is no declaration (the ordinary case, never a
// diagnostic) or when it was read; otherwise it is the terminal-safe clause
// saying why a present declaration was not honoured, for the caller to render
// in its own voice. A declaration that was not honoured names nothing.
func HomeDeclarationNames(home, rel string, limit int64, target string, fold bool) (declared bool, ignored string) {
	raw, refusal, err := ReadHomeDeclaration(home, rel, limit)
	switch refusal {
	case DeclarationOK:
	case DeclarationAbsent:
		return false, ""
	case DeclarationBehindSymlink:
		return false, termsafe.Sanitize(err.Error())
	case DeclarationNotRegular:
		return false, "it is not a regular file"
	case DeclarationWritableByOthers:
		return false, "it is writable by others, so its contents are not necessarily yours"
	case DeclarationForeignOwner:
		return false, "it is not owned by this session's uid"
	default:
		return false, "it could not be read (" + termsafe.Sanitize(err.Error()) + ")"
	}
	want := []string{FoldPath(filepath.Clean(target), fold), FoldPath(resolveOrClean(target), fold)}
	for _, line := range strings.Split(string(raw), "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") || !filepath.IsAbs(entry) {
			continue
		}
		for _, cand := range []string{filepath.Clean(entry), resolveOrClean(entry)} {
			key := FoldPath(cand, fold)
			if key == want[0] || key == want[1] {
				return true, ""
			}
		}
	}
	return false, ""
}

// resolveOrClean is EvalSymlinks with a lexical fallback, so a declared path
// that does not currently exist (a bind mount not mounted) still compares.
func resolveOrClean(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}
