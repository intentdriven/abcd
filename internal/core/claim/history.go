package claim

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
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
//
// The walk reads first-parent lines only, each from its own start and never
// more than limit+1 commits of it: the tip's, and, when the mark is not in
// that window, the mark's. It never asks git for the commits between two
// commits (`tip ^mark`) or whether one is an ancestor of the other: either
// reads all the history between them, however little it prints, and a
// writer may push a million commits in one fast-forward. What that leaves
// unsaid is in Added's, More's and Merged's documentation, and one reading
// more: a tip moved back more than limit commits below the mark, with more
// than limit commits beneath it, reads as More with Rewritten false, and
// Added then holds commits already read.
//
// Every commit is read as stored, never through a replace ref or a graft
// file (storedGit).
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
	// Added is the commits on the tip's first-parent line above the mark,
	// newest first, at most the walk's limit. With the branch rewritten, it is
	// the tip's line above where it meets the mark's line (nothing, when the
	// tip is on the mark's line: the branch moved back), or, with the mark
	// gone or no meeting found, the tip's line as though never read. A tip's
	// line that reaches its root and meets the mark's line only more than
	// limit+1 commits below the mark is that last case: Added then carries
	// commits already read, and Rewritten says so.
	Added []string
	// More reports that the tip's line runs past the limit before reaching
	// the mark or the mark's line: Added holds the newest limit commits of it,
	// and the rest are not read. A rewrite whose new line runs past the limit
	// before it meets the old one reads as More, not Rewritten: telling the
	// two apart reads the whole history between.
	More bool
	// Rewritten reports that the mark is not on the tip's first-parent line:
	// the mark names a commit this checkout no longer holds, the tip's line
	// meets the mark's line below the mark, or the tip's line reaches its root
	// without meeting it. A mark reachable from the tip only through a merge's
	// second parent is not on the line, and reads as rewritten.
	Rewritten bool
	// Merged reports that a commit in Added, every one of them a commit the
	// walk passed between the tip and the mark, has more than one parent:
	// commits reached only through a merge are not listed; treat the window
	// as incomplete. A window whose parents pass the listing's bound (an
	// octopus with more parents than the window has commits) is read without
	// them, and reads as Merged whenever Added is not empty.
	Merged bool
}

// since walks the branch at ref, a full tracking ref under ns.tracking, from
// its tip back to the commit this checkout last marked it read at, returning
// at most limit commits (1 to maxWalk) and reading at most limit+1 commits of
// each of at most two first-parent lines (history). It reads only this
// checkout's disk and writes nothing: marking the walk read is markRead's,
// once its caller has consumed it. The walk follows first parents only, so a
// merge onto the branch cannot pull another line's history into it; it says
// when one was passed (Merged).
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
	line, merges, err := tipLine(root, ref, h.Tip, limit)
	if err != nil {
		return history{}, err
	}
	if err := h.place(root, ref, line, limit); err != nil {
		return history{}, err
	}
	// Added is always a prefix of line.
	h.Merged = len(h.Added) > 0 && (merges == nil || slices.Contains(merges[:len(h.Added)], true))
	return h, nil
}

// place sets Added, More and Rewritten from the tip's line, reading the
// mark's own line when the mark is not in the tip's window.
func (h *history) place(root, ref string, line []string, limit int) error {
	if h.Since == "" {
		h.Added, h.More = upTo(line, limit)
		return nil
	}
	if i := slices.Index(line, h.Since); i >= 0 {
		h.Added = line[:i]
		return nil
	}
	held, err := hasCommit(root, h.Since)
	if err != nil {
		return fmt.Errorf("reading %s's mark: %w", ref, err)
	}
	if !held {
		h.Rewritten = true
		h.Added, h.More = upTo(line, limit)
		return nil
	}
	// The mark is not on the tip's line as far as it was read: the branch was
	// rewritten if the tip's line meets the mark's below the mark, or ends
	// without meeting it; otherwise it only ran past the window.
	old, err := firstParents(root, ref, h.Since, limit)
	if err != nil {
		return err
	}
	for i, c := range line {
		if slices.Contains(old, c) {
			h.Added, h.Rewritten = line[:i], true
			return nil
		}
	}
	h.Rewritten = len(line) <= limit
	h.Added, h.More = upTo(line, limit)
	return nil
}

// upTo is line's first limit commits, and whether it held more.
func upTo(line []string, limit int) ([]string, bool) {
	if len(line) > limit {
		return line[:limit], true
	}
	return line, false
}

// tipLine is firstParents from the tip, read in the same one bounded walk
// with each commit's parents (--parents), and whether each commit is a merge.
// The listing has room for every commit of the window to name two parents; a
// window naming more (an octopus) overflows it, and the line is then read
// again without parents and merges is nil: git cannot say how many parents a
// commit has in fewer bytes than naming them all, and the bound is what keeps
// a writer's commit from growing this read.
func tipLine(root, ref, tip string, limit int) (line []string, merges []bool, err error) {
	out, err := storedGit(root, (limit+1)*3*(len(tip)+1)+1,
		"rev-list", "--first-parent", "--parents", "--max-count="+strconv.Itoa(limit+1), tip, "--")
	if err != nil {
		if !strings.Contains(err.Error(), "exceeded") {
			return nil, nil, fmt.Errorf("reading %s: %w", ref, err)
		}
		line, err := firstParents(root, ref, tip, limit)
		return line, nil, err
	}
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 0 {
			continue
		}
		for _, sha := range f {
			if !gitutil.IsFullSHA(sha) {
				return nil, nil, fmt.Errorf("reading %s: git named a commit %s", ref, echo(sha))
			}
		}
		line, merges = append(line, f[0]), append(merges, len(f) > 2)
	}
	return line, merges, nil
}

// firstParents is the first-parent line from commit, newest first, at most
// limit+1 commits of it. With nothing excluded, git stops at the count, so
// the walk reads no more than it prints. The `--` ends the revisions, so a
// file named as the commit is never read as a path.
func firstParents(root, ref, commit string, limit int) ([]string, error) {
	out, err := storedGit(root, (limit+1)*(len(commit)+1)+1,
		"rev-list", "--first-parent", "--max-count="+strconv.Itoa(limit+1), commit, "--")
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ref, err)
	}
	var line []string
	for _, sha := range strings.Fields(out) {
		if !gitutil.IsFullSHA(sha) {
			return nil, fmt.Errorf("reading %s: git named a commit %s", ref, echo(sha))
		}
		line = append(line, sha)
	}
	return line, nil
}

// hasCommit reports whether this checkout's object store holds commit as a
// commit. It reads the object's type, which git takes from the object's
// header; peeling it (`^{commit}`) reads and hashes the whole object, however
// big a writer made it. A missing object is the answer no: `cat-file -t`
// cannot tell it from a failure, so `cat-file -e`, whose exit 1 is "no such
// object", tells them apart.
func hasCommit(root, commit string) (bool, error) {
	if !gitutil.IsFullSHA(commit) {
		return false, nil
	}
	typ, err := storedGit(root, 64, "cat-file", "-t", commit)
	if err == nil {
		return strings.TrimSpace(typ) == "commit", nil
	}
	args := []string{"cat-file", "-e", commit}
	trace(args...)
	if _, missing, eerr := gitutil.RunCappedExitOneAsStored(root, 64, args...); eerr == nil && missing {
		return false, nil
	}
	return false, err
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
