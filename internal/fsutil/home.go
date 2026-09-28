package fsutil

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
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
// It judges by path, so it answers for the layout and not for a race: a
// caller that must hold the answer through its use — every reader and writer of
// a file there — reaches the file through OpenHomeScope or EnsureHomeScope,
// which judge each level the same way and keep the descriptor of the directory
// judged. A caller may still call HomeScopeLink first to refuse early, before
// any other work, in the same words.
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

// ErrHomeScopeSwapped is the refusal for a directory of a home-scoped path that
// was a real directory when judged and was something else by the time it was
// opened: the descriptor OpenHomeScope obtained is not the directory its Lstat
// vetted, and that directory is not a symlink now either (a symlink found there
// is refused as a *HomeScopeLinkError, which names it).
var ErrHomeScopeSwapped = errors.New("fsutil: a directory of this home-scoped path was replaced while it was opened")

// homeScopeVetted runs between OpenHomeScope's Lstat of one directory level and
// its open of that level. It does nothing in production; it is a var so a
// detector can swap the level for a symlink inside that window, which a real
// race cannot be relied on to hit, and so prove the open refuses what it did
// not vet. dir is the level's full path.
var homeScopeVetted = func(dir string) {}

// SwapHomeScopeVettedForTest substitutes the hook that runs between a home
// scope level's vetting Lstat and its open, and returns the restore. It is
// exported because the writers whose refusal needs proving live in other
// packages (credential, oracle, ahoy). Tests only; never called in production
// code, and never safe to call from a parallel test.
func SwapHomeScopeVettedForTest(fn func(dir string)) (restore func()) {
	prev := homeScopeVetted
	homeScopeVetted = fn
	return func() { homeScopeVetted = prev }
}

// OpenHomeScope opens the directory dir below home (a slash path, ".abcd" or
// ".abcd/history") as an *os.Root, walking it one level at a time relative to
// the descriptor of the level above, so HomeScopeLink's rule holds against a
// race and not only against a layout. It is the descriptor form of that rule:
// HomeScopeLink judges each level by path and the caller then reaches the file
// by path again, so a process running as the same uid that swaps ~/.abcd for a
// symlink between the judgement and the use reads or writes through the link
// (iss-2609281310017733). Here every use goes through the returned root, and
// the root is the very directory each level's judgement was made about.
//
// Each level is judged and then opened: Lstat relative to the level above
// (never following), a symlink refused with a *HomeScopeLinkError naming it, a
// non-directory refused with ErrNotRealDir, then the level opened relative to
// the same descriptor and confirmed with os.SameFile to be the directory the
// Lstat vetted. The confirmation is what os.Root alone cannot give: it follows
// a symlink that stays inside the root, and a dotfiles ~/.abcd usually points
// inside home. A level replaced between its Lstat and its open is refused — as
// a *HomeScopeLinkError when a symlink stands there now, and as
// ErrHomeScopeSwapped otherwise. The guarantee is the one an openat with
// O_NOFOLLOW gives, which the standard library does not expose on every
// platform abcd builds for: the directory held is the one that stood at that
// name, a real directory, when it was vetted; a rename of it afterwards moves
// it without changing what the root refers to.
//
// home itself is opened by path and never judged, for HomeScopeLink's reason
// (decision 2): a home reached through a link is the machine's layout. dir is
// held to ValidRelPath, or is "." for home itself. An absent level returns the
// Lstat's error, which os.IsNotExist recognises. The caller closes the root.
func OpenHomeScope(home, dir string) (*os.Root, error) {
	return openHomeScope(home, dir, false, 0)
}

// EnsureHomeScope is OpenHomeScope for a writer: each missing level is created
// at perm (a single mkdir relative to the level above, which never follows a
// link at the name it creates) before it is judged and opened, so the walk
// that creates ~/.abcd is the same walk that proves it. It is what a writer
// uses in place of os.MkdirAll, which follows a symlinked ~/.abcd and creates
// under its target. A level that already exists keeps its mode.
func EnsureHomeScope(home, dir string, perm os.FileMode) (*os.Root, error) {
	return openHomeScope(home, dir, true, perm)
}

func openHomeScope(home, dir string, create bool, perm os.FileMode) (*os.Root, error) {
	if dir != "." && !ValidRelPath(dir) {
		return nil, &os.PathError{Op: "openhomescope", Path: dir, Err: os.ErrInvalid}
	}
	cur, err := os.OpenRoot(home)
	if err != nil {
		return nil, err
	}
	if dir == "." {
		return cur, nil
	}
	full := home
	shown := "~"
	for _, part := range strings.Split(dir, "/") {
		full = filepath.Join(full, part)
		shown += "/" + part
		next, err := openHomeScopeLevel(cur, part, full, shown, create, perm)
		cur.Close()
		if err != nil {
			return nil, err
		}
		cur = next
	}
	return cur, nil
}

// openHomeScopeLevel is one level of openHomeScope: part, inside parent, judged
// and then opened as the directory that was judged.
func openHomeScopeLevel(parent *os.Root, part, full, shown string, create bool, perm os.FileMode) (*os.Root, error) {
	if create {
		if err := parent.Mkdir(part, perm); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
	}
	fi, err := parent.Lstat(part)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, &HomeScopeLinkError{Link: shown}
	}
	if !fi.IsDir() {
		return nil, &os.PathError{Op: "openhomescope", Path: full, Err: ErrNotRealDir}
	}
	homeScopeVetted(full)
	next, err := parent.OpenRoot(part)
	if err != nil {
		return nil, swappedLevel(parent, part, full, shown, err)
	}
	st, err := next.Stat(".")
	if err != nil {
		next.Close()
		return nil, err
	}
	if !os.SameFile(fi, st) {
		next.Close()
		return nil, swappedLevel(parent, part, full, shown, ErrHomeScopeSwapped)
	}
	return next, nil
}

// swappedLevel names a level whose open did not reach the directory that was
// vetted: a symlink standing there now is refused as the link it is, so the
// operator reads the same sentence the unraced refusal gives; anything else is
// reported with err.
func swappedLevel(parent *os.Root, part, full, shown string, err error) error {
	if fi, lerr := parent.Lstat(part); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
		return &HomeScopeLinkError{Link: shown}
	}
	if errors.Is(err, ErrHomeScopeSwapped) {
		return &os.PathError{Op: "openhomescope", Path: full, Err: err}
	}
	return err
}

// ReadHomeDeclaration is ReadDeclaration for a declaration file named by its
// place in the caller's home: home joined with rel (a slash path,
// ".abcd/trusted-roots"). It is the read every home-scoped declaration goes
// through, and it adds the one fact ReadDeclaration cannot see from a path —
// that no directory between home and the file is a symlink — and holds it
// through the read: the directory is opened by OpenHomeScope, and the file is
// judged and read relative to that descriptor, never by its path again
// (iss-2609281310017733).
//
// The order keeps AGENTS.md's rule exactly: a file that is not there is
// DeclarationAbsent whatever the directories are, so a symlinked ~/.abcd
// holding no such file reads as absent and costs its owner nothing; a file
// that IS there behind a symlinked directory is DeclarationBehindSymlink,
// refused before a byte of it is read. The Lstat that decides absence follows
// the directories, which is what lets a file behind a link be found in order to
// be refused; it decides nothing else.
//
// The file's own guards are ReadDeclaration's, applied to the descriptor: a
// leaf that is not a regular file is DeclarationNotRegular, one writable by
// group or other or owned by another uid is DeclarationWritableByOthers or
// DeclarationForeignOwner, and a leaf replaced between its judgement and its
// open is DeclarationUnreadable with ErrDeclarationSwapped. A directory level
// replaced while it was opened is DeclarationBehindSymlink when a symlink
// stands there now and DeclarationUnreadable otherwise.
//
// A rel that is not a clean relative path is DeclarationUnreadable before
// anything is looked at: it names no place in the home to read.
func ReadHomeDeclaration(home, rel string, limit int64) ([]byte, DeclarationRefusal, error) {
	return ReadHomeDeclarationDenying(home, rel, limit, 0)
}

// ReadHomeDeclarationDenying is ReadHomeDeclaration for a file whose reader
// refuses more of its mode than a declaration's group- and other-write: deny
// holds the permission bits that refuse it (0o077 for a secret no one else
// may read). The bits are judged on the fstat of the file that was opened,
// never on a path, so a file swapped or re-moded after any check by path is
// judged as the file that is read (iss-2609281310017733). A refusal is
// DeclarationExposed with a *DeclarationModeError naming the mode. Every other
// guard, and its order, is ReadHomeDeclaration's.
func ReadHomeDeclarationDenying(home, rel string, limit int64, deny os.FileMode) ([]byte, DeclarationRefusal, error) {
	if !ValidRelPath(rel) {
		return nil, DeclarationUnreadable, &os.PathError{Op: "readhomedeclaration", Path: rel, Err: os.ErrInvalid}
	}
	p := filepath.Join(home, filepath.FromSlash(rel))
	if _, err := os.Lstat(p); err != nil {
		return nil, DeclarationAbsent, err
	}
	root, err := OpenHomeScope(home, path.Dir(rel))
	switch {
	case err == nil:
	case errors.Is(err, ErrHomeScopeSymlinked):
		return nil, DeclarationBehindSymlink, err
	case notPresent(err):
		return nil, DeclarationAbsent, err
	default:
		return nil, DeclarationUnreadable, err
	}
	defer root.Close()
	return readDeclarationIn(root, path.Base(rel), p, limit, deny.Perm())
}

// readDeclarationIn is ReadDeclaration for the file leaf directly inside root:
// the same guards in the same order, with the Lstat, the open and the confirming
// fstat all relative to root's descriptor. p is the file's full path, which the
// owner lookup (ownerUID, the package's test seam) and the vetting hook are
// given; the owner is confirmed again on the opened descriptor, so the lookup
// by path cannot vouch for a file other than the one read. deny is judged on
// that same descriptor (ReadHomeDeclarationDenying).
func readDeclarationIn(root *os.Root, leaf, p string, limit int64, deny os.FileMode) ([]byte, DeclarationRefusal, error) {
	fi, err := root.Lstat(leaf)
	if err != nil {
		return nil, DeclarationAbsent, err
	}
	if !fi.Mode().IsRegular() {
		return nil, DeclarationNotRegular, ErrNotRegular
	}
	if err := CallersAlone(p, fi); err != nil {
		if errors.Is(err, ErrDeclarationWritable) {
			return nil, DeclarationWritableByOthers, err
		}
		return nil, DeclarationForeignOwner, err
	}
	declarationVetted(p)
	f, err := root.OpenFile(leaf, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, DeclarationUnreadable, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, DeclarationUnreadable, err
	}
	if !st.Mode().IsRegular() || !os.SameFile(fi, st) {
		return nil, DeclarationUnreadable, ErrDeclarationSwapped
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); !ok || sys.Uid != uint32(os.Getuid()) {
		return nil, DeclarationForeignOwner, ErrDeclarationForeignOwner
	}
	if perm := st.Mode().Perm(); perm&deny != 0 {
		return nil, DeclarationExposed, &DeclarationModeError{Perm: perm}
	}
	if st.Size() > limit {
		return nil, DeclarationUnreadable, ErrTooBig
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, DeclarationUnreadable, err
	}
	if int64(len(data)) > limit {
		return nil, DeclarationUnreadable, ErrTooBig
	}
	return data, DeclarationOK, nil
}
