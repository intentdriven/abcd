package worktree

// list.go is what the store says it holds (spc-2609301811532881, step 3):
// List renders every worktree git reports for the repository, the main
// checkout excepted, and every directory in its lane, each judged clean or
// dirty and merged or unmerged; ListAll walks every lane on the machine.
//
// A worktree belongs to the store when the proof of belonging holds
// (adr-2609091248200336): git lists it from this checkout, its real path is
// a direct child of the real path of this repository's lane, and its own
// answer for the common directory is this checkout's. A row that fails the
// proof carries its reason in place of the state columns, and prune reads
// the same rows, so what list shows is what prune acts on.
//
// Every string a row carries is display-ready: a path home-redacted, and a
// branch, a lock or prune reason and a label passed through termsafe.Sanitize,
// because each is another checkout's bytes and reaches a terminal or the JSON
// envelope as it is.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Where a row's worktree sits.
const (
	// WhereStore is a direct child of the repository's lane.
	WhereStore = "store"
	// WhereOutside is anywhere else git lists a worktree: listed, never touched.
	WhereOutside = "outside"
	// WhereSibling is one of abcd's own sibling stores kept in the lane (the
	// embargo store, iss-2610230014564359): never judged, archived or removed.
	WhereSibling = "sibling"
)

// The reasons a row carries in place of its state columns.
const (
	reasonCheckoutGone = "checkout not found; run `git worktree repair` from the checkout"
	reasonDirGone      = "directory gone; git still lists it"
	reasonNotListed    = "not a worktree of this repository"
	reasonOtherCommon  = "another checkout's worktree: its common directory is not this checkout's"
	reasonNoGitFile    = "not a worktree: no .git file names a checkout"
	reasonEmbargo      = "abcd's embargo store (`abcd embargo`), never judged or pruned"
	reasonForeignOwner = "owned by another account; state unknown"
	reasonOwnerUnread  = "its owner could not be read; state unknown"
)

// ownerOf is the uid owning a worktree's root directory: fsutil.OwnerUID, the
// one ownership lookup. It is a variable only so a test can stand in for a
// worktree another account owns, which a test cannot make without root.
var ownerOf = fsutil.OwnerUID

// statusCap bounds one worktree's status listing: a tree whose listing passes
// it is dirty past doubt, and reads as unjudged rather than truncated.
const statusCap = 4 << 20

// minAbbrev is the shortest directory name read as an abbreviation of a root
// commit, git's own shortest abbreviation.
const minAbbrev = 4

// Row is one worktree, or one directory in the store, as list renders it.
type Row struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Where  string `json:"where"`
	// State is clean or dirty, Merged merged or unmerged, and JudgedBy how a
	// merged branch was judged (forge, ancestry, content); all three are empty
	// when Reason says why the row is not judged.
	State    string `json:"state,omitempty"`
	Merged   string `json:"merged,omitempty"`
	JudgedBy string `json:"judged_by,omitempty"`
	// Locked and Prunable are git's attributes, with git's reasons.
	Locked         bool   `json:"locked,omitempty"`
	LockReason     string `json:"lock_reason,omitempty"`
	Prunable       bool   `json:"prunable,omitempty"`
	PrunableReason string `json:"prunable_reason,omitempty"`
	Reason         string `json:"reason,omitempty"`
	// Abs is the worktree's absolute path, as git or the lane names it, for a
	// caller that acts on the row. It is never output: Path is the form that
	// may be shown, so a home path never reaches a terminal or the JSON.
	Abs string `json:"-"`
}

// ListOptions is how List judges the rows.
type ListOptions struct {
	// Forge is the forge client, asked once for the run's pull requests; nil
	// asks none, and merged is judged from git alone.
	Forge GH
}

// Listing is List's answer.
type Listing struct {
	// Lane is the repository's lane, home-redacted.
	Lane      string `json:"lane"`
	Worktrees []Row  `json:"worktrees"`
	// ForgeNote says, once, that no forge answered and why; empty when one did
	// or there was nothing to ask about.
	ForgeNote string `json:"forge_note,omitempty"`
}

// entry is one row before it is rendered, with what prune acts on.
type entry struct {
	wt     gitutil.Worktree // git's record; zero for a directory git does not list
	path   string           // absolute, as git or the lane names it
	name   string
	where  string
	reason string // why the state columns are not judged; "" when they are
	clean  bool
	merged judgement
	// status is git's listing of the tree's changes, which the quiet rule
	// reads for the last activity and the dossier's dirty state.
	status []gitutil.StatusEntry
	// openPR is whether the forge lists an open pull request from the
	// branch; false when no forge answered.
	openPR bool
}

// row renders e for display.
func (e entry) row(redact func(string) string) Row {
	r := Row{
		Name:           termsafe.Sanitize(e.name),
		Branch:         termsafe.Sanitize(shortBranch(e.wt)),
		Path:           termsafe.Sanitize(redact(e.path)),
		Where:          e.where,
		Locked:         e.wt.Locked,
		LockReason:     termsafe.Sanitize(e.wt.LockReason),
		Prunable:       e.wt.Prunable,
		PrunableReason: termsafe.Sanitize(e.wt.PrunableReason),
		Reason:         termsafe.Sanitize(fsutil.RedactHome(e.reason)),
		Abs:            e.path,
	}
	if e.reason != "" {
		return r
	}
	r.State, r.Merged = "dirty", "unmerged"
	if e.clean {
		r.State = "clean"
	}
	if e.merged.merged {
		r.Merged, r.JudgedBy = "merged", e.merged.by
	}
	return r
}

// shortBranch is the worktree's branch as a person names it, "detached" for a
// detached HEAD, and "" for a directory git does not list.
func shortBranch(wt gitutil.Worktree) string {
	switch {
	case wt.Branch != "":
		return strings.TrimPrefix(wt.Branch, "refs/heads/")
	case wt.Detached:
		return "detached"
	}
	return ""
}

// displayFor is the path form a row in where takes: a store row's path is the
// one a reader can cd into, an outside row's is recognisable without printing
// an absolute local path.
func displayFor(where string) func(string) string {
	if where == WhereOutside {
		return fsutil.DisplayPath
	}
	return fsutil.RedactHome
}

func render(es []entry) []Row {
	rows := make([]Row, 0, len(es))
	for _, e := range es {
		rows = append(rows, e.row(displayFor(e.where)))
	}
	return rows
}

// List renders the repository at repoRoot's worktrees and its lane. A store
// level that is a symlink, another account's or writable by others is
// refused before anything under it is read, and an absent lane holds no
// rows of its own; nothing is made.
func List(repoRoot string, opts ListOptions) (Listing, error) {
	lane, err := LaneFor(repoRoot)
	if err != nil {
		return Listing{}, err
	}
	es, note, err := scan(repoRoot, lane, opts.Forge, nil)
	if err != nil {
		return Listing{}, err
	}
	return Listing{Lane: fsutil.RedactHome(lane.Dir), Worktrees: render(es), ForgeNote: note}, nil
}

// scan reads the repository's worktrees and its lane into entries, judged,
// and the forge's note. It is the one reading list and prune share. A pick
// keeps only the entries it accepts, before any is judged; nil keeps every
// one.
func scan(repoRoot string, lane Lane, forge GH, pick func(entry) bool) ([]entry, string, error) {
	inLane, err := lane.present()
	if err != nil {
		return nil, "", err
	}
	storeLevel, err := levels(lane.Home, StoreRel, "worktree store", false)
	if err != nil {
		return nil, "", err
	}
	wts, err := gitutil.ListWorktrees(repoRoot, gitutil.MaxWorktreeListing)
	if err != nil {
		return nil, "", fmt.Errorf("listing the repository's worktrees: %w", err)
	}
	own, err := gitutil.CommonDir(repoRoot)
	if err != nil {
		return nil, "", fmt.Errorf("reading the checkout's common directory: %w", err)
	}
	own = fsutil.RealExistingPath(own)
	realLane := fsutil.RealExistingPath(lane.Dir)

	var es []entry
	seen := map[string]bool{}
	for i, wt := range wts {
		// git lists the main working tree first; it is the checkout, not a row.
		if i == 0 || wt.Bare {
			continue
		}
		real := fsutil.RealExistingPath(wt.Path)
		seen[real] = true
		e := entry{wt: wt, path: wt.Path, name: filepath.Base(filepath.Clean(wt.Path)), where: WhereOutside}
		switch {
		case storepath.HasStoreShape(wt.Path):
			e.where, e.reason = WhereSibling, reasonEmbargo
		case filepath.Dir(real) == realLane:
			e.where = WhereStore
		}
		if e.reason == "" {
			e.reason = ownedElsewhere(wt.Path)
		}
		if e.reason == "" {
			e.reason = belongs(wt, own)
		}
		es = append(es, e)
	}
	if inLane {
		names, err := dirNames(lane.Dir)
		if err != nil {
			return nil, "", err
		}
		for _, n := range names {
			if n == startsDir || seen[filepath.Join(realLane, n)] {
				continue
			}
			e := entry{path: lane.Path(n), name: n, where: WhereStore, reason: reasonNotListed}
			if n == storepath.Name {
				e.where, e.reason = WhereSibling, reasonEmbargo
			}
			es = append(es, e)
		}
	}
	if storeLevel {
		store := filepath.Dir(lane.Dir)
		names, err := dirNames(store)
		if err != nil {
			return nil, "", err
		}
		for _, n := range names {
			if n != lane.RootSHA && len(n) >= minAbbrev && strings.HasPrefix(lane.RootSHA, n) {
				es = append(es, entry{path: filepath.Join(store, n), name: n, where: WhereStore, reason: notALane(lane, true)})
			}
		}
	}
	if pick != nil {
		es = slices.DeleteFunc(es, func(e entry) bool { return !pick(e) })
	}
	note := judgeAll(repoRoot, lane, unjudged(es), forge)
	return es, note, nil
}

// ownedElsewhere is why no git may run in the worktree at p, read before any
// does: its root directory is another account's, or its owner could not be
// read, which fails closed. The store judges only the caller's own
// worktrees, so it leaves another account's unjudged and never runs git
// there. "" when the caller owns it, or when it is gone, which the proof's
// own reasons then name.
func ownedElsewhere(p string) string {
	uid, err := ownerOf(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ""
	case err != nil:
		return reasonOwnerUnread
	case uid != uint32(os.Getuid()):
		return reasonForeignOwner
	}
	return ""
}

// belongs is the proof of belonging's third test, and git's own word on a
// listed worktree: "" when the worktree's own answer for the common directory
// is this checkout's, otherwise the reason it is not judged.
func belongs(wt gitutil.Worktree, own string) string {
	if wt.Prunable {
		return reasonDirGone
	}
	common, err := gitutil.CommonDir(wt.Path)
	switch {
	case err != nil:
		return reasonCheckoutGone
	case fsutil.RealExistingPath(common) != own:
		return reasonOtherCommon
	}
	return ""
}

// unjudged is every entry with no reason not to judge it.
func unjudged(es []entry) []*entry {
	var out []*entry
	for i := range es {
		if es[i].reason == "" {
			out = append(out, &es[i])
		}
	}
	return out
}

// judgeAll judges each entry against the default branch of the repository at
// dir, asking the forge once for every branch among them. A store entry on
// the branch its start record in lane names is judged from that start.
func judgeAll(dir string, lane Lane, judged []*entry, forge GH) string {
	if len(judged) == 0 {
		return ""
	}
	prs, note := askForge(forge, dir)
	def := gitutil.DefaultRef(dir)
	for _, e := range judged {
		st, err := gitutil.Status(e.path, statusCap, gitutil.StatusOptions{})
		if err != nil {
			e.reason = "git could not read its status: " + firstLine(err.Error())
			continue
		}
		e.clean, e.status = len(st) == 0, st
		e.openPR = openFrom(prs, e.wt)
		start := ""
		if e.where == WhereStore {
			if rec, ok := readStart(lane, e.name); ok && rec.Branch == e.wt.Branch {
				start = rec.Start
			}
		}
		e.merged = judge(e.path, def, e.wt, start, prs)
	}
	return note
}

// notALane is the reason a directory directly under the store whose name is
// not a full root commit renders with: never a lane, retired by hand, and,
// when its name abbreviates the root commit of the repository the caller
// stands in, naming that repository's full-key lane.
func notALane(here Lane, prefix bool) string {
	if prefix {
		return fmt.Sprintf("not a lane: its name abbreviates this repository's root commit, whose lane is %s; its worktrees are retired by hand with `git worktree remove`",
			fsutil.RedactHome(here.Dir))
	}
	return "not a lane: its name is not a full root commit; its worktrees are retired by hand with `git worktree remove`"
}

// dirNames lists dir's entries by name, sorted, without following any of
// them.
func dirNames(dir string) ([]string, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", fsutil.RedactHome(dir), err)
	}
	names := make([]string, 0, len(des))
	for _, d := range des {
		names = append(names, d.Name())
	}
	slices.Sort(names)
	return names, nil
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}

// Label is a repository's entry in abcd's registry, keyed on its root commit:
// the name a lane is shown under. Path is the registry's label for where the
// checkout was, which ListAll never reads: which checkout a lane belongs to is
// each worktree's own .git file's answer.
type Label struct {
	Name string
	Path string
}

// AllOptions is how ListAll labels and judges the lanes.
type AllOptions struct {
	// Labels are the registry's labels by root commit; a lane with none is
	// shown under its root commit.
	Labels map[string]Label
	// Here is the checkout the caller stands in, "" for none: a directory
	// abbreviating its root commit names its full-key lane.
	Here string
	// Forge is asked once per lane's checkout; nil asks none.
	Forge GH
}

// LaneListing is one directory directly under the store.
type LaneListing struct {
	// Key is the directory's name, a root commit for a lane.
	Key string `json:"key"`
	// Label is the repository's registry name, or Key where it has none.
	Label     string `json:"label"`
	Path      string `json:"path"`
	Reason    string `json:"reason,omitempty"`
	Worktrees []Row  `json:"worktrees"`
}

// AllListing is ListAll's answer.
type AllListing struct {
	Store     string        `json:"store"`
	Lanes     []LaneListing `json:"lanes"`
	ForgeNote string        `json:"forge_note,omitempty"`
}

// ListAll walks every directory under ~/.abcd.noindex/worktrees/ without following
// symlinks. A directory named by a full root commit is a lane, labelled from
// opts.Labels; its worktrees are judged from their own checkout, found
// through each one's .git file. Anything else is not a lane.
func ListAll(opts AllOptions) (AllListing, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return AllListing{}, fmt.Errorf("cannot resolve the caller's home directory for the worktree store: %v", err)
	}
	store := filepath.Join(home, filepath.FromSlash(StoreRel))
	out := AllListing{Store: fsutil.RedactHome(store), Lanes: []LaneListing{}}
	present, err := levels(home, StoreRel, "worktree store", false)
	if err != nil || !present {
		return out, err
	}
	var here *Lane
	if opts.Here != "" {
		if l, err := LaneFor(opts.Here); err == nil {
			here = &l
		}
	}
	names, err := dirNames(store)
	if err != nil {
		return AllListing{}, err
	}
	for _, n := range names {
		ll := LaneListing{Key: termsafe.Sanitize(n), Label: termsafe.Sanitize(n), Path: termsafe.Sanitize(fsutil.RedactHome(filepath.Join(store, n))), Worktrees: []Row{}}
		if !gitutil.IsFullSHA(n) {
			prefix := here != nil && n != here.RootSHA && len(n) >= minAbbrev && strings.HasPrefix(here.RootSHA, n)
			if prefix {
				ll.Reason = notALane(*here, true)
			} else {
				ll.Reason = notALane(Lane{}, false)
			}
			out.Lanes = append(out.Lanes, ll)
			continue
		}
		if l, ok := opts.Labels[n]; ok && l.Name != "" {
			ll.Label = termsafe.Sanitize(l.Name)
		}
		lane := Lane{Home: home, RootSHA: n, Rel: StoreRel + "/" + n, Dir: filepath.Join(store, n)}
		if _, err := lane.present(); err != nil {
			var r *Refusal
			if !errors.As(err, &r) {
				return AllListing{}, err
			}
			ll.Reason = r.Error()
			out.Lanes = append(out.Lanes, ll)
			continue
		}
		es, reason, note, err := scanLane(lane, opts.Forge)
		if err != nil {
			return AllListing{}, err
		}
		if note != "" && out.ForgeNote == "" {
			out.ForgeNote = note
		}
		ll.Reason, ll.Worktrees = termsafe.Sanitize(reason), render(es)
		out.Lanes = append(out.Lanes, ll)
	}
	return out, nil
}

// scanLane reads one lane under --all into entries. Each directory with a
// .git file is a worktree of whichever checkout that file names; its checkout
// lists it and judges it. A lane whose every worktree's .git file names a
// checkout that is gone is one reason, with no rows for those worktrees.
func scanLane(lane Lane, forge GH) ([]entry, string, string, error) {
	names, err := dirNames(lane.Dir)
	if err != nil {
		return nil, "", "", err
	}
	var es []entry
	// The worktrees of each checkout, by its real common directory, so a
	// checkout is listed and asked once.
	byCommon := map[string][]int{}
	var commons []string
	gone, live := 0, 0
	for _, n := range names {
		if n == startsDir {
			continue
		}
		p := lane.Path(n)
		e := entry{path: p, name: n, where: WhereStore}
		fi, err := os.Lstat(p)
		switch {
		case n == storepath.Name:
			e.where, e.reason = WhereSibling, reasonEmbargo
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return nil, "", "", fmt.Errorf("reading %s: %w", fsutil.RedactHome(p), err)
		case err != nil || !fi.IsDir():
			e.reason = reasonNoGitFile
		default:
			if _, ok := gitutil.GitfileTarget(p); !ok {
				e.reason = reasonNoGitFile
				break
			}
			if r := ownedElsewhere(p); r != "" {
				e.reason = r
				break
			}
			common, err := gitutil.CommonDir(p)
			if err != nil {
				gone++
				e.reason = reasonCheckoutGone
				break
			}
			live++
			c := fsutil.RealExistingPath(common)
			if _, ok := byCommon[c]; !ok {
				commons = append(commons, c)
			}
			byCommon[c] = append(byCommon[c], len(es))
		}
		es = append(es, e)
	}
	note := ""
	for _, c := range commons {
		idx := byCommon[c]
		first := es[idx[0]].path
		wts, err := gitutil.ListWorktrees(first, gitutil.MaxWorktreeListing)
		if err != nil {
			return nil, "", "", fmt.Errorf("listing the worktrees of %s's checkout: %w", fsutil.RedactHome(first), err)
		}
		listed := map[string]gitutil.Worktree{}
		for _, wt := range wts {
			listed[fsutil.RealExistingPath(wt.Path)] = wt
		}
		group := make([]*entry, 0, len(idx))
		for _, i := range idx {
			wt, ok := listed[fsutil.RealExistingPath(es[i].path)]
			if !ok {
				es[i].reason = "not listed by the checkout its .git file names"
				continue
			}
			es[i].wt = wt
			if wt.Prunable {
				es[i].reason = reasonDirGone
				continue
			}
			group = append(group, &es[i])
		}
		if n := judgeAll(first, lane, group, forge); n != "" && note == "" {
			note = n
		}
	}
	if gone > 0 && live == 0 {
		kept := es[:0]
		for _, e := range es {
			if e.reason != reasonCheckoutGone {
				kept = append(kept, e)
			}
		}
		return kept, reasonCheckoutGone, note, nil
	}
	return es, "", note, nil
}
