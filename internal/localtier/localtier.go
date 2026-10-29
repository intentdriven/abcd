// Package localtier guards reads of a checkout's local tier,
// .abcd/.work.local/: the run, drain and build states, receipts, holds and
// keys abcd keeps for one checkout alone. The tier is ignored by design and
// never committed, so a file in it that git tracks came with a commit, not
// from this checkout, and is not this checkout's state. Every reader of such
// state asks Untracked first.
package localtier

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// RelDir is the local tier, relative to a checkout's root.
const RelDir = ".abcd/.work.local"

// Refused is Untracked's refusal: git tracks something at or beneath Rel, or
// something that reaches the tier (Paths, the first few of them), or git could
// not say whether it does (Cause), which is never read as "nothing tracked".
type Refused struct {
	Rel   string
	Paths []string
	Cause error
}

// Reason says why the read is refused, naming the path.
func (e *Refused) Reason() string {
	if e.Cause != nil {
		return fmt.Sprintf("could not ask git whether it tracks %s (%v), so abcd does not read it as this checkout's state",
			termsafe.Sanitize(e.Rel), e.Cause)
	}
	return fmt.Sprintf("git tracks %s, but abcd keeps %s for this checkout alone and never commits it, so abcd does not read it as this checkout's state",
		strings.Join(termsafe.SanitizeAll(e.Paths), ", "), termsafe.Sanitize(e.Rel))
}

// Remedy says what makes the read possible again.
func (e *Refused) Remedy() string {
	if e.Cause != nil {
		return "repair the repository so git can list what it tracks (`git status` names most faults), then run again"
	}
	return "untrack what git tracks there with `git rm --cached -r --` on each path named (the files stay on disk), and remove them unless this checkout wrote them"
}

func (e *Refused) Error() string { return e.Reason() + "; " + e.Remedy() }

// shown bounds the tracked paths a refusal names.
const shown = 3

// Untracked returns nil when nothing git tracks supplies the file or
// directory at rel, a slash-separated path relative to repoRoot (a file a
// reader opens, or a directory it lists), and otherwise a *Refused: naming
// what git tracks, or, when git cannot answer for a repository-shaped tree,
// saying so. Outside anything repository-shaped no commit can carry the file,
// and it returns nil.
//
// What git tracks is judged by the files it names, not by their spelling: an
// entry is refused when it is rel or lies beneath it, and also when it reaches
// the tier (Carried) and opens rel or a directory above or below it, so a
// spelling the filesystem folds onto the path, a link or a submodule standing
// in for the tier, or one standing in for .abcd, is refused as surely as the
// path's own name. An entry elsewhere in the tier supplies nothing to rel and
// leaves its read alone; record-lint reports it at the change that tracks it.
// repoRoot may be any directory inside the checkout, rel then relative to it,
// as a reader holding an os.Root opened on a run or lane directory has them;
// the refusal names paths from the working tree's top, where its remedy runs.
func Untracked(repoRoot, rel string) error {
	return UntrackedContext(context.Background(), repoRoot, rel)
}

// UntrackedContext is Untracked with its git calls bound to ctx, for a reader
// that holds a budget; a context that ends first refuses, as git not answering
// does.
func UntrackedContext(ctx context.Context, repoRoot, rel string) error {
	return UntrackedEach(ctx, repoRoot, rel)
}

// UntrackedEach is UntrackedContext for every one of rels, asking git once:
// for a reader that lists a directory and reads what it finds there, so a
// file git tracks beside the ones it reads refuses none of them. The refusal
// is the first rel refused.
func UntrackedEach(ctx context.Context, repoRoot string, rels ...string) error {
	if len(rels) == 0 {
		return nil
	}
	dir, err := filepath.Abs(repoRoot)
	if err != nil {
		return &Refused{Rel: rels[0], Cause: err}
	}
	top := gitutil.RepoShapedRoot(dir)
	if top == "" {
		return nil
	}
	tracked, err := gitutil.TrackedInTreeContext(ctx, top)
	if err != nil {
		return &Refused{Rel: rels[0], Cause: err}
	}
	reach, err := carried(top, tracked)
	if err != nil {
		return &Refused{Rel: rels[0], Cause: err}
	}
	for _, rel := range rels {
		full, err := filepath.Rel(top, filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return &Refused{Rel: rel, Cause: err}
		}
		rel = filepath.ToSlash(full)
		var paths []string
		for _, p := range tracked {
			if p == rel || strings.HasPrefix(p, rel+"/") {
				paths = append(paths, p)
			}
		}
		for _, p := range reach {
			if supplies(top, p, rel) && !slices.Contains(paths, p) {
				paths = append(paths, p)
			}
		}
		if len(paths) > 0 {
			return &Refused{Rel: rel, Paths: paths[:min(len(paths), shown)]}
		}
	}
	return nil
}

// Carried lists what git tracks in the working tree at repoRoot that reaches
// its local tier: an entry inside the directory the tier's name opens, by
// whatever spelling git holds it (a case or Unicode variant the filesystem
// folds onto it), the tier itself as an entry (a link, a submodule or a file
// standing in for it), or .abcd itself as one. Outside anything
// repository-shaped it lists nothing; a tree git cannot answer for is an error.
func Carried(repoRoot string) ([]string, error) {
	top, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	tracked, err := gitutil.TrackedInTreeContext(context.Background(), top)
	if err != nil {
		return nil, err
	}
	return carried(top, tracked)
}

// carried is Carried over the tracked list already in hand. Each distinct
// first component, and each distinct second component beneath one that is
// .abcd, is judged once, by the file it names (os.SameFile on Lstat, which
// never follows the link a commit may have put there).
func carried(top string, tracked []string) ([]string, error) {
	abcd, err := os.Lstat(filepath.Join(top, ".abcd"))
	if errors.Is(err, fs.ErrNotExist) {
		abcd, err = nil, nil
	}
	if err != nil {
		return nil, err
	}
	tier, err := os.Lstat(filepath.Join(top, filepath.FromSlash(RelDir)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	same := map[string]bool{}
	is := func(rel string, want fs.FileInfo) bool {
		if want == nil {
			return false
		}
		if v, ok := same[rel]; ok {
			return v
		}
		fi, err := os.Lstat(filepath.Join(top, filepath.FromSlash(rel)))
		same[rel] = err == nil && os.SameFile(fi, want)
		return same[rel]
	}
	var out []string
	for _, p := range tracked {
		first, rest, deeper := strings.Cut(p, "/")
		second, _, _ := strings.Cut(rest, "/")
		switch {
		case folds(first, ".abcd") && (!deeper || folds(second, ".work.local")):
			// A spelling a case-folding filesystem opens as the tier, refused on
			// every filesystem: the clone that folds it may be another one.
			out = append(out, p)
		case !is(first, abcd):
		case !deeper || is(first+"/"+second, tier):
			out = append(out, p)
		}
	}
	return out, nil
}

// supplies reports whether p, an entry that reaches the tier (carried), opens
// rel or a directory above or below it: past the components that reach .abcd
// and the tier, p's components and rel's agree as far as the shorter goes,
// each by name under folding or, where the names differ, by the file the
// prefix so far opens under top (a filesystem may fold more than simple case
// folding does, a ligature onto its letters among them). So an entry standing
// in for .abcd or the tier itself supplies every path in the tier. Outside the
// tier rel is supplied only by an entry standing in for .abcd, when rel lies
// under it.
func supplies(top, p, rel string) bool {
	have, want := strings.Split(p, "/"), strings.Split(rel, "/")
	switch {
	case len(want) >= 2 && want[0] == ".abcd" && want[1] == ".work.local":
		have, want = have[min(len(have), 2):], want[2:]
		tier := filepath.Join(top, filepath.FromSlash(RelDir))
		for i := range min(len(have), len(want)) {
			if folds(have[i], want[i]) {
				continue
			}
			a, errA := os.Lstat(filepath.Join(append([]string{tier}, have[:i+1]...)...))
			b, errB := os.Lstat(filepath.Join(append([]string{tier}, want[:i+1]...)...))
			if errA != nil || errB != nil || !os.SameFile(a, b) {
				return false
			}
		}
		return true
	case want[0] == ".abcd":
		return len(have) == 1
	}
	return false
}

// folds reports whether name is want under Unicode case folding, the
// equivalence a case-insensitive filesystem applies (the Kelvin sign folds to
// k as k's own capital does).
func folds(name, want string) bool { return strings.EqualFold(name, want) }

// ReadInRoot is fsutil.ReadGuardedInRoot for local-tier state: rel, relative
// to root, is refused as Untracked refuses it, and its bytes are never handed
// back when it is. Git is asked only once there is a file to refuse, so an
// absent one costs no question. The refusal is never an fs.ErrNotExist, so a
// reader that takes an absent file as "no state" still refuses one a commit
// carries.
func ReadInRoot(root *os.Root, rel string, limit int64) ([]byte, error) {
	data, err := fsutil.ReadGuardedInRoot(root, rel, limit)
	if err != nil {
		return nil, err
	}
	if err := Untracked(root.Name(), rel); err != nil {
		return nil, err
	}
	return data, nil
}

// LstatInRoot is root.Lstat for local-tier state a reader judges by its
// presence alone (a receipt that lets a step pass), refused as ReadInRoot
// refuses once there is something to refuse.
func LstatInRoot(root *os.Root, rel string) (fs.FileInfo, error) {
	fi, err := root.Lstat(rel)
	if err != nil {
		return nil, err
	}
	if err := Untracked(root.Name(), rel); err != nil {
		return nil, err
	}
	return fi, nil
}
