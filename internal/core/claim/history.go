package claim

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxWalk bounds every history walk, whatever its caller asks: the most
// commits one read of one branch returns.
const maxWalk = 1000

// history is what one branch gained since this checkout last read it, as the
// walk (namespace.since) reads it from this checkout's disk.
type history struct {
	// Tip is the branch's commit as last fetched; "" when this checkout holds
	// no such ref.
	Tip string
	// Unreadable is a fixed reason the tip cannot be walked (it is not a
	// commit), "" when it can.
	Unreadable string
	// Since is the commit this checkout last read the branch at (its mark);
	// "" when it never read it, or the mark does not read.
	Since string
	// Added is the commits the branch gained after Since, along its
	// first-parent line, newest first, at most the walk's limit.
	Added []string
	// More reports that the branch gained more than the limit: Added holds the
	// newest of them, and the rest are not read.
	More bool
	// Rewritten reports that Since is no longer in the branch's history: the
	// branch was rewritten (a non-fast-forward) relative to the last read, or
	// the mark names a commit this checkout no longer holds. Added is then
	// what the tip holds that Since does not, or, with Since gone, the walk
	// from the tip as though never read.
	Rewritten bool
}

// since walks the branch at ref, a full tracking ref under ns.tracking, from
// its tip back to the commit this checkout last marked it read at, reading at
// most limit commits (1 to maxWalk). It reads only this checkout's disk and
// writes nothing: marking the walk read is markRead's, once its caller has
// consumed it. The walk follows first parents only, so a merge onto the branch
// cannot pull another line's history into it.
func (ns namespace) since(root, ref string, limit int) (history, error) {
	if limit < 1 || limit > maxWalk {
		return history{}, fmt.Errorf("a walk reads 1 to %d commits, not %d", maxWalk, limit)
	}
	if err := ns.owns(ref); err != nil {
		return history{}, err
	}
	h := history{Since: ns.lastRead(root, ref)}
	name, typ, ok, err := resolveRef(root, ref)
	if err != nil || !ok {
		return h, err
	}
	h.Tip = name
	if typ != "commit" {
		h.Unreadable = reasonNoObject
		return h, nil
	}
	if h.Since == h.Tip {
		return h, nil
	}
	walk := []string{"rev-list", "--first-parent", "--max-count=" + strconv.Itoa(limit+1), h.Tip}
	if h.Since != "" {
		held, err := hasCommit(root, h.Since)
		if err != nil {
			return history{}, fmt.Errorf("reading %s's mark: %w", ref, err)
		}
		if !held {
			h.Rewritten = true
		} else {
			// The mark is in the tip's history exactly when nothing it holds
			// is missing from the tip's.
			out, err := localGit(root, 128, "rev-list", "--max-count=1", h.Since, "^"+h.Tip)
			if err != nil {
				return history{}, fmt.Errorf("reading %s: %w", ref, err)
			}
			h.Rewritten = strings.TrimSpace(out) != ""
			walk = append(walk, "^"+h.Since)
		}
	}
	out, err := localGit(root, (limit+1)*(len(h.Tip)+1)+1, walk...)
	if err != nil {
		return history{}, fmt.Errorf("reading %s: %w", ref, err)
	}
	for _, sha := range strings.Fields(out) {
		if !gitutil.IsFullSHA(sha) {
			return history{}, fmt.Errorf("reading %s: git named a commit %s", ref, echo(sha))
		}
		h.Added = append(h.Added, sha)
	}
	if len(h.Added) > limit {
		h.Added, h.More = h.Added[:limit], true
	}
	return h, nil
}

// hasCommit reports whether this checkout's object store holds commit as a
// commit; git's exit 1 is the answer no.
func hasCommit(root, commit string) (bool, error) {
	args := []string{"rev-parse", "--verify", "--quiet", commit + "^{commit}"}
	trace(args...)
	out, missing, err := gitutil.RunCappedExitOne(root, 128, args...)
	if err != nil {
		return false, err
	}
	return !missing && strings.TrimSpace(string(out)) == commit, nil
}

// markRead records that this checkout has read the branch at ref up to
// commit, the mark the next walk starts from. A checkout without its local
// tier keeps no mark (the tier is never created here), so its every walk
// reads as never read.
func (ns namespace) markRead(root, ref, commit string) error {
	if err := ns.owns(ref); err != nil {
		return err
	}
	if !gitutil.IsFullSHA(commit) {
		return fmt.Errorf("a read mark names a commit, not %s", echo(commit))
	}
	if tierPresent(root) != nil {
		return nil
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := mkStateDir(r, ns.readDir()); err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(r, ns.markRel(ref), []byte(commit+"\n"), stateMode)
}

// lastRead is the commit ref's mark names; "" when there is none or it does
// not read as a commit's name.
func (ns namespace) lastRead(root, ref string) string {
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, ns.markRel(ref), 128)
	if err != nil {
		return ""
	}
	if sha := strings.TrimSpace(string(data)); gitutil.IsFullSHA(sha) {
		return sha
	}
	return ""
}

// readDir is the directory the namespace keeps its read marks in.
func (ns namespace) readDir() string { return ns.stateDir + "/read" }

// markRel is ref's mark, named by the hash of the ref so no branch name, which
// a writer chose, ever becomes a path.
func (ns namespace) markRel(ref string) string {
	sum := sha256.Sum256([]byte(ref))
	return ns.readDir() + "/" + hex.EncodeToString(sum[:])
}

// owns refuses a ref outside the namespace's tracking prefix, or one naming
// nothing under it.
func (ns namespace) owns(ref string) error {
	if name, ok := strings.CutPrefix(ref, ns.tracking); !ok || name == "" {
		return fmt.Errorf("%s is not a branch of %s", echo(ref), ns.noun)
	}
	return nil
}
