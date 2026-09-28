package fsutil

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
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
func HomeScopeLink(home, rel string) error {
	dir := path.Dir(path.Clean(filepath.ToSlash(rel)))
	if dir == "." || dir == "/" || strings.HasPrefix(dir, "../") || dir == ".." {
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
func ReadHomeDeclaration(home, rel string, limit int64) ([]byte, DeclarationRefusal, error) {
	p := filepath.Join(home, filepath.FromSlash(rel))
	if _, err := os.Lstat(p); err != nil {
		return nil, DeclarationAbsent, err
	}
	if err := HomeScopeLink(home, rel); err != nil {
		return nil, DeclarationBehindSymlink, err
	}
	return ReadDeclaration(p, limit)
}
