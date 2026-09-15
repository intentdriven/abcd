package mode

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// storePerm is the stored file's mode. The tier is per-machine and gitignored,
// and its other store — the private banlist — is written 0600; the state is not a
// secret, but there is no reason for it to be wider than the tier it lives in.
const storePerm fs.FileMode = 0o600

// Root answers which checkout's mode store a caller standing in cwd addresses.
// It holds no resolution of its own: the three-state answer — git's toplevel, a
// refusal for a repository-shaped tree git cannot answer for, and a refusal
// outside a repository — is resolved once in gitutil.CheckoutRoot, and all this
// store supplies is the noun those refusals carry.
//
// Both refusals wrap gitutil.ErrNoCheckoutRoot, so a front door maps one error to
// one exit code. Neither falls back to the working directory: a mode store minted
// under a subdirectory would read `managed` against a repository whose loop is
// parked, and report success while doing it.
func Root(cwd string) (string, error) {
	return gitutil.CheckoutRoot(cwd, StoreName)
}

// Read returns the waiting-on state of the repository containing cwd, resolving
// the checkout root first. An absent store reads as Managed.
func Read(cwd string) (State, error) {
	root, err := Root(cwd)
	if err != nil {
		return "", err
	}
	return ReadAt(root)
}

// Set records the waiting-on state of the repository containing cwd, resolving
// the checkout root first. An unknown state is refused before anything is
// resolved or written.
func Set(cwd string, s State) error {
	if !s.Valid() {
		return unknown(s)
	}
	root, err := Root(cwd)
	if err != nil {
		return err
	}
	return SetAt(root, s)
}

// ReadAt returns the waiting-on state stored under an already-resolved checkout
// root. Use Read where the caller holds only a working directory.
//
// An absent store — an absent file, an absent tier, an absent `.abcd/` — reads as
// Managed, which is the whole of the managed-only rule on this side: a repository
// abcd does not manage has no tier, so it has no file, so it reads as the same
// "nobody is waiting" a managed repository with no parked stop reads as.
//
// Everything else fails closed. A store that is a symlink, a directory, a device
// or oversize is an error rather than a fallback to Managed, and so is a word
// outside the vocabulary: each of those is somebody having written something
// here, and reporting "nobody is waiting" over it would hide exactly the parked
// stop this store exists to make visible.
func ReadAt(repoRoot string) (State, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", fmt.Errorf("opening the checkout to read %s: %w", StoreName, err)
	}
	defer root.Close()

	// Resolved inside the os.Root so a symlinked ANCESTOR — `.abcd` or
	// `.work.local` committed as a symlink out of the tree — cannot walk the read
	// outside the checkout. A path that escapes the root is not an fs.ErrNotExist,
	// so the absent-means-managed branch below fails closed on the escape rather
	// than reporting it as "the file is not there".
	data, err := fsutil.ReadGuardedInRoot(root, FileRelPath, maxStoreBytes)
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist):
		return Managed, nil
	default:
		return "", fmt.Errorf("reading %s at %s: %w", StoreName, FileRelPath, err)
	}
	return ParseState(string(data))
}

// SetAt records the waiting-on state under an already-resolved checkout root. Use
// Set where the caller holds only a working directory.
//
// Two refusals, and both happen before any byte is written. An unknown state is
// refused first, so an invalid word leaves the store exactly as it was rather
// than truncating it. An absent local-ephemeral tier is refused second, and the
// tier is never created: it is present only in a repository abcd manages, which
// is what makes this state managed-only without a separate managed check to drift
// from. A tier that is a symlink rather than a real directory is refused by the
// same test.
//
// The write itself is a whole-file replacement through fsutil's atomic writer, so
// the rename is the commit point and a concurrent reader sees either the old
// state or the new one, never a half-written word. It needs no lock because it
// reads nothing first: two writers race to a well-formed file, and the later
// rename wins.
func SetAt(repoRoot string, s State) error {
	if !s.Valid() {
		return unknown(s)
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("opening the checkout to write %s: %w", StoreName, err)
	}
	defer root.Close()

	// Lstat, not Stat: a symlink standing in for the tier reports its target's
	// IsDir through Stat, and the point of the test is that the tier is really
	// here, in this checkout.
	fi, err := root.Lstat(TierRelPath)
	if err != nil || !fi.IsDir() {
		return fmt.Errorf("%w: %s/ is not a directory in this checkout, so there is nowhere for %s to live — only a repository abcd manages has that tier, and it is never created on the way to a write",
			ErrNoLocalTier, TierRelPath, StoreName)
	}

	if err := fsutil.WriteFileAtomicInRoot(root, FileRelPath, []byte(string(s)+"\n"), storePerm); err != nil {
		return fmt.Errorf("writing %s at %s: %w", StoreName, FileRelPath, err)
	}
	return nil
}

// unknown phrases the closed vocabulary's refusal for a State the caller supplied
// (as against one ParseState read out of the store). The word reaches a terminal
// from a front door's argument, so it is cleaned on the same terms as a word read
// out of the file.
func unknown(s State) error {
	return fmt.Errorf("%w: %q is not one of %s",
		ErrUnknownState, termsafe.CleanProseLine(string(s), echoCap), stateList())
}
