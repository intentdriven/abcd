package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// layerfile.go — the guarded reads of the two override layers' files, exported
// so a store that keeps a file beside each rules.json (the memory notes'
// memory.json, spc-2610111627213138) reads it with exactly the checks and the
// refusal wording the rules layers get, rather than a second copy that could
// drift. The rules loader reads its own two files through these.

// ReadRepoLayerFile reads <repoRoot>/<rel>, rel a slash path below .abcd/
// (".abcd/rules.json"). ok is false when the file is absent; every other failure
// is an error prefixed "<what>: " naming the file by rel.
//
// A symlinked .abcd directory component is refused before the leaf is touched,
// so a swapped .abcd cannot redirect the read; the leaf itself is opened once,
// O_NOFOLLOW and O_NONBLOCK, and judged on its descriptor as a regular file
// within limit bytes (readGuarded).
func ReadRepoLayerFile(repoRoot, rel, what string, limit int64) (data []byte, ok bool, err error) {
	if !strings.HasPrefix(rel, ".abcd/") || !fsutil.ValidRelPath(rel) {
		return nil, false, fmt.Errorf("%s: %s is not a file below .abcd/", what, rel)
	}
	if di, err := os.Lstat(filepath.Join(repoRoot, ".abcd")); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return nil, false, fmt.Errorf("%s: .abcd is a symlink (refusing to follow)", what)
	}
	data, err = readGuarded(filepath.Join(repoRoot, filepath.FromSlash(rel)), limit)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return nil, false, nil
		case errors.Is(err, syscall.ELOOP):
			return nil, false, fmt.Errorf("%s: %s is a symlink (refusing to follow)", what, rel)
		case errors.Is(err, errNotRegular):
			return nil, false, fmt.Errorf("%s: %s is not a regular file", what, rel)
		case errors.Is(err, errTooBig):
			return nil, false, fmt.Errorf("%s: %s exceeds the %d-byte cap", what, rel, limit)
		default:
			return nil, false, fmt.Errorf("%s: reading %s: %w", what, rel, err)
		}
	}
	return data, true, nil
}

// ReadUserLayerFile reads the user scope's file rel (a slash path below home,
// abcdhome.Rel("rules.json")). ok is false when there is no such file — or no
// home to hold one — which is the ordinary case and costs nothing: nothing is
// created, nothing is reported. Every refusal is an error prefixed "<what>: "
// naming the file in tilde form; holds names what the file carries ("rules",
// "notes") in the two refusals that say whose they are.
//
// The file injects text into every session on the machine, so it is read as the
// caller's WORD, through the same primitive as the other home-scoped
// declarations (fsutil.ReadDeclaration): a regular file — never a symlink,
// FIFO or device — within the size cap, owned by this session's uid and
// writable by nobody else. That is the ownership rule the repo root's trust
// bound applies to a foreign-uid checkout (root.go, foreignOwnerRefusal): a
// file another account could have written is not this account's convention.
// And the ~/.abcd.noindex directory itself must not be a symlink when a file sits
// behind it, the pre-check the repo's .abcd gets — checked only once a file is
// there, because a machine whose ~/.abcd.noindex is a dotfiles symlink holding no
// such file reads nothing and must keep behaving exactly as it did. That check
// is fsutil.ReadHomeDeclaration's (fsutil.HomeScopeLink), the one every other
// home-scoped reader and writer applies, so the rule cannot drift per file.
//
// Every refusal is an error naming the file in tilde form, never a silent
// fallback: an unreadable user layer that degraded to a partial injection is
// the silent shape loud-staging exists to prevent.
func ReadUserLayerFile(home, rel, what, holds string, limit int64) (data []byte, ok bool, err error) {
	// No home, or a relative one, names no user scope: a relative HOME would
	// resolve ~/.abcd.noindex against whatever directory the session happens to start
	// in, which is not the machine's scope but a guess at one.
	if home == "" || !filepath.IsAbs(home) {
		return nil, false, nil
	}
	shown := "~/" + rel
	data, refusal, err := fsutil.ReadHomeDeclaration(home, rel, limit)
	// Absent is the lstat's answer, so a permission error here is a HOME or
	// ~/.abcd.noindex this uid cannot search, never the file's own mode: that reads as
	// no user layer, as it does for the sibling home-scoped declarations
	// (trusted-roots, local-transcript-roots), so a sandboxed or foreign HOME
	// does not warn on every prompt about a file nobody can see. A file
	// that is there and cannot be opened is DeclarationUnreadable, and loud.
	if refusal == fsutil.DeclarationAbsent && (os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) || errors.Is(err, fs.ErrPermission)) {
		return nil, false, nil
	}
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationBehindSymlink:
		return nil, false, fmt.Errorf("%s: %s is a symlink (refusing to follow it to %s)", what, "~/"+path.Dir(rel), shown)
	case fsutil.DeclarationDirectoryExposed:
		return nil, false, fmt.Errorf("%s: %s is not read: %w", what, shown, err)
	case fsutil.DeclarationNotRegular:
		return nil, false, fmt.Errorf("%s: %s is not a regular file (a symlink, FIFO or device is refused)", what, shown)
	case fsutil.DeclarationWritableByOthers:
		return nil, false, fmt.Errorf("%s: %s is writable by others, so its %s are not necessarily yours (chmod go-w it): %w", what, shown, holds, err)
	case fsutil.DeclarationForeignOwner:
		return nil, false, fmt.Errorf("%s: %s is not owned by this session's uid, so its %s are not this account's: %w", what, shown, holds, err)
	default:
		if errors.Is(err, fsutil.ErrTooBig) {
			return nil, false, fmt.Errorf("%s: %s exceeds the %d-byte cap", what, shown, limit)
		}
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP) {
			return nil, false, fmt.Errorf("%s: %s is not a regular file (a symlink, FIFO or device is refused)", what, shown)
		}
		// The raw error can carry the expanded path; the message names the
		// file in tilde form and keeps only the reason.
		why := "unknown error"
		var pe *os.PathError
		switch {
		case errors.As(err, &pe):
			why = pe.Err.Error()
		case err != nil:
			why = termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		}
		return nil, false, fmt.Errorf("%s: %s could not be read (%s)", what, shown, why)
	}
	return data, true, nil
}
