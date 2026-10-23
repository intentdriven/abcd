package worktree

// prune.go reclaims what has merged (spc-2609301811532881, step 4): a
// worktree that passes the proof of belonging, whose branch has merged and
// whose tree is clean, has its local tier archived and is then removed by
// `git worktree remove`, never forced. Everything else stays and is named with
// why. The store deletes only what passes its proof of belonging
// (adr-2609091248200336), and nothing git did not delete.

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// A prune row's outcome.
const (
	OutcomeReclaimed    = "reclaimed"
	OutcomeWouldReclaim = "would reclaim"
	OutcomeKept         = "kept"
)

// Held is what the build loop holds in the lane: the worktrees a run whose
// state is not complete names, by their name in the lane, and the runs whose
// state could not be read, each of which holds every worktree named after it.
type Held struct {
	// Names maps a worktree's name in the lane to the run holding it.
	Names map[string]string
	// Runs are the run ids whose state could not be read.
	Runs []string
}

// By is the run holding the worktree called name, or "".
func (h Held) By(name string) string {
	if run := h.Names[name]; run != "" {
		return run
	}
	for _, run := range h.Runs {
		if strings.HasPrefix(name, run+"-") {
			return run
		}
	}
	return ""
}

// HeldReader reads what the build loop holds for the repository at
// repoRoot. The loop imports this package, so it is handed in, the shape the
// status block's lane reader takes.
type HeldReader func(repoRoot string) (Held, error)

// PruneOptions is one prune run.
type PruneOptions struct {
	// DryRun archives and removes nothing, and reports the rows a real run
	// would.
	DryRun bool
	// Forge is asked once for the merged judgement; nil asks none.
	Forge GH
	// Held reads what the build loop holds. It is required: a run that cannot
	// tell what the loop holds removes nothing.
	Held HeldReader
	// Cwd is prune's own working directory; the worktree holding it is kept.
	Cwd string
	// Now stamps the archive entry; nil is the clock.
	Now func() time.Time
	// PreRemove is the caller's own check on a worktree prune is about to
	// remove, given its absolute path: asked last, after every refusal, the
	// merged judgement and the archive's flush, immediately before `git
	// worktree remove`. A keep keeps the worktree with reason and takes back
	// the archive entry just written for it. A dry run asks it at the same
	// point, so it reports the row a real run would. nil checks nothing more.
	PreRemove func(abs string) (keep bool, reason string)
}

// PruneRow is one worktree's outcome.
type PruneRow struct {
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	Path    string `json:"path"`
	Where   string `json:"where"`
	Outcome string `json:"outcome"`
	// Reason says why a kept worktree was kept.
	Reason string `json:"reason,omitempty"`
	// Notes is how much of the local tier a reclaimed worktree had archived,
	// "none" for an absent or empty tier.
	Notes string `json:"notes,omitempty"`
	// Archive names the notes entry written, home-redacted.
	Archive string `json:"archive,omitempty"`
}

// Pruned is Prune's answer.
type Pruned struct {
	Lane      string     `json:"lane"`
	DryRun    bool       `json:"dry_run"`
	Rows      []PruneRow `json:"rows"`
	ForgeNote string     `json:"forge_note,omitempty"`
}

// Reclaimed counts the rows reclaimed, or that a dry run would reclaim.
func (p Pruned) Reclaimed() int {
	n := 0
	for _, r := range p.Rows {
		if r.Outcome == OutcomeReclaimed || r.Outcome == OutcomeWouldReclaim {
			n++
		}
	}
	return n
}

// Prune reclaims the merged, clean worktrees of the repository at repoRoot's
// lane. A store refusal (a symlinked level, one another account owns or
// others can write) is returned before anything is judged, and so is a
// failure to read what the build loop holds; otherwise every worktree is a
// row.
func Prune(repoRoot string, opts PruneOptions) (Pruned, error) {
	return prune(repoRoot, "", opts)
}

// PruneOne is Prune for the one worktree called name in the repository's
// lane: the same proof, refusals, judgement, archive and removal, its answer
// the one row. A name the lane does not hold, a worktree outside the store
// among them, is refused before anything is archived or removed. Only that
// worktree is judged, so a caller walking the lane one name at a time does
// not judge every worktree once a call.
func PruneOne(repoRoot, name string, opts PruneOptions) (Pruned, error) {
	if name == "" {
		return Pruned{}, refuse("no worktree is named", "name a worktree of this repository's lane")
	}
	return prune(repoRoot, name, opts)
}

// prune is Prune and PruneOne's one path; only, when set, is the name of the
// one worktree in the lane it judges.
func prune(repoRoot, only string, opts PruneOptions) (Pruned, error) {
	if opts.Held == nil {
		return Pruned{}, errors.New("worktree: Prune needs a reader of what the build loop holds")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	lane, err := LaneFor(repoRoot)
	if err != nil {
		return Pruned{}, err
	}
	var pick func(entry) bool
	if only != "" {
		pick = func(e entry) bool { return inLane(lane, only, e) }
	}
	es, note, err := scan(repoRoot, lane, opts.Forge, pick)
	if err != nil {
		return Pruned{}, err
	}
	if only != "" && len(es) == 0 {
		return Pruned{}, refuse(fmt.Sprintf("this repository's lane holds no worktree named %q", only),
			"name one `abcd ahoy worktree list` shows in the store")
	}
	held, err := opts.Held(repoRoot)
	if err != nil {
		return Pruned{}, fmt.Errorf("reading what the build loop holds: %w", err)
	}
	cwd := ""
	if opts.Cwd != "" {
		cwd = fsutil.RealExistingPath(opts.Cwd)
	}
	out := Pruned{Lane: fsutil.RedactHome(lane.Dir), DryRun: opts.DryRun, Rows: []PruneRow{}, ForgeNote: note}
	for _, e := range es {
		r := e.row(displayFor(e.where))
		row := PruneRow{Name: r.Name, Branch: r.Branch, Path: r.Path, Where: r.Where, Outcome: OutcomeKept}
		row.Reason = keptFor(e, held, cwd)
		if row.Reason == "" {
			reclaim(repoRoot, lane, e, opts, &row)
		}
		row.Reason = termsafe.Sanitize(fsutil.RedactHome(row.Reason))
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// inLane reports whether e is the directory called name directly in lane,
// listed by git or not. The name is matched exactly and the real paths
// compared, so neither a path for a name nor an outside worktree sharing the
// base name selects anything.
func inLane(lane Lane, name string, e entry) bool {
	return e.name == name && e.where != WhereOutside &&
		fsutil.RealExistingPath(e.path) == fsutil.RealExistingPath(lane.Path(name))
}

// keptFor is why e is kept before its notes are read, in the spec's order, or
// "" for a worktree prune may reclaim.
func keptFor(e entry, held Held, cwd string) string {
	switch {
	case e.where == WhereOutside:
		return "outside the store; never touched"
	case e.reason != "":
		return e.reason
	case e.wt.Locked && e.wt.LockReason != "":
		return "locked by git: " + e.wt.LockReason
	case e.wt.Locked:
		return "locked by git"
	case cwd != "" && fsutil.PathWithin(cwd, fsutil.RealExistingPath(e.path), fsutil.CaseFoldingFS()):
		return "prune's own working directory"
	}
	if run := held.By(e.name); run != "" {
		return "held by the build loop's run " + run
	}
	switch {
	case !e.clean:
		return "dirty: uncommitted changes"
	case !e.merged.merged:
		return "unmerged"
	}
	return ""
}

// removeWorktree is `git worktree remove` without --force, run from the
// checkout. It is a variable only so a test can see when it is asked for.
var removeWorktree = func(repoRoot, path string) error {
	_, err := gitutil.Run(repoRoot, "worktree", "remove", "--", path)
	return err
}

// reclaim archives e's notes and removes it, or under a dry run reads the
// notes the archive would write and stops there. A failed archive or a git
// that refuses the removal keeps the worktree, naming why.
func reclaim(repoRoot string, lane Lane, e entry, opts PruneOptions, row *PruneRow) {
	notes, err := collectNotes(repoRoot, e.path)
	if err != nil {
		row.Reason = "the notes could not be archived: " + err.Error()
		return
	}
	row.Notes = notesCount(notes)
	if opts.DryRun {
		if keptByCaller(opts, e, row) {
			return
		}
		row.Outcome = OutcomeWouldReclaim
		return
	}
	entryDir := ""
	if len(notes) > 0 {
		entryDir, err = writeNotes(repoRoot, lane, e, notes, opts.Now())
		if entryDir != "" {
			row.Archive = termsafe.Sanitize(fsutil.RedactHome(entryDir))
		}
		if err != nil {
			row.Notes = ""
			row.Reason = "the notes could not be archived: " + err.Error()
			return
		}
	}
	if keptByCaller(opts, e, row) {
		// The tier still stands in the kept worktree, so the entry is a
		// second copy no removal will ever claim: taken back, it leaves no
		// orphan in the archive. The entry was made by this run alone.
		if entryDir != "" {
			if err := os.RemoveAll(entryDir); err != nil {
				row.Reason += "; its notes archive entry could not be taken back: " + causeOf(err)
				return
			}
			row.Archive = ""
		}
		return
	}
	// Without --force: git re-checks the tree and the lock as it removes, and
	// refuses a tree that changed since it was judged. The archive written
	// stays, since the tier still stands in the kept worktree.
	if err := removeWorktree(repoRoot, e.path); err != nil {
		row.Reason = "git refused the removal: " + firstLine(err.Error())
		return
	}
	row.Outcome = OutcomeReclaimed
}

// keptByCaller asks the caller's PreRemove about e and, on a keep, marks row
// kept with the caller's reason. The reason is caller text, and is masked and
// home-redacted with every other reason as the row is finished.
func keptByCaller(opts PruneOptions, e entry, row *PruneRow) bool {
	if opts.PreRemove == nil {
		return false
	}
	keep, reason := opts.PreRemove(e.path)
	if !keep {
		return false
	}
	if reason == "" {
		reason = "kept by the caller's check before removal"
	}
	row.Notes, row.Reason = "", reason
	return true
}

func notesCount(notes []note) string {
	switch len(notes) {
	case 0:
		return "none"
	case 1:
		return "1 file"
	}
	return fmt.Sprintf("%d files", len(notes))
}
