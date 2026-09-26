// Package reviews reads the committed review folders under .abcd/work/reviews/
// and says how stale each has become (itd-28, spc-2609211854150455).
//
// Two kinds of folder live there. A dated review, `<YYYY-MM-DD>-<scope>/`, is a
// commissioned review whose `00-summary.md` names the commit it read in its
// frontmatter (`review_of_commit: <full sha>`); a review of a spec carries the
// spec's id in its scope, `<YYYY-MM-DD>-<spc-N>-<slug>/`. A semantic-gate
// receipt directory, `<40-hex>/<gate>.json`, is keyed by the commit it gates,
// so its name is its pin. The package reads both and counts, per pin, the
// commits the default branch has moved since; it writes nothing.
//
// The pin's one reading is Pin. The reviews-charter gate
// (scripts/check-reviews.sh, RD004) refuses a dated folder whose summary Pin
// would not read, so the board and the gate agree about every folder.
package reviews

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Dir is the committed reviews tree, repo-relative and slash-separated.
const Dir = ".abcd/work/reviews"

// StaleAfter is the threshold the board flags past: a review whose pin the
// default branch has moved more than this many commits beyond is marked for a
// re-run (itd-28 decision 2).
const StaleAfter = 20

// Kind names the two folder classes the tree holds.
const (
	KindReview  = "review"
	KindReceipt = "receipt"
)

// State says whether a folder's staleness could be counted.
const (
	// StatePinned: the pin names a commit this repository holds, and the
	// count is the commits the default branch has moved since.
	StatePinned = "pinned"
	// StateUnreachable: the pin names no commit this repository can count
	// from (a rewritten or squashed-away sha), so the age is unknown.
	StateUnreachable = "unreachable"
	// StateUnpinned: the folder names no commit at all — a review that
	// predates the pin rule.
	StateUnpinned = "unpinned"
)

// summaryFile is the dated review's consolidated summary, the file the pin
// lives in.
const summaryFile = "00-summary.md"

// maxSummaryBytes caps the one read a summary costs; the pin sits in the
// frontmatter at the top, and a summary past this is pathological.
const maxSummaryBytes = 1 << 20

var (
	receiptRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
	datedRe   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-(.+)$`)
	specRe    = regexp.MustCompile(`(?:^|-)(spc-[0-9]+)(?:-|$)`)
	// pinRe is a full object name, SHA-1 or SHA-256, as git prints it.
	pinRe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
)

// Entry is one review folder as the tree holds it, before any git is asked.
type Entry struct {
	Folder string `json:"folder"`
	Kind   string `json:"kind"`
	// Scope is what the folder reviewed: the dated folder's name after its
	// date, or a receipt directory's gates, comma-separated.
	Scope string `json:"scope"`
	// Spec is the spec id the dated folder's name carries, if any.
	Spec string `json:"spec,omitempty"`
	// ReviewOfCommit is the pin: the summary's frontmatter key, or the
	// receipt directory's own name. Empty when the folder names none.
	ReviewOfCommit string `json:"review_of_commit,omitempty"`
}

// Row is one folder on the board, with its age.
type Row struct {
	Entry
	State string `json:"state"`
	// CommitsSince is the number of commits the default branch holds that
	// the pin does not; absent when the state is not pinned.
	CommitsSince *int `json:"commits_since"`
	// Stale is true when CommitsSince is past the threshold.
	Stale bool `json:"stale"`
}

// Board is the reviews block of the status board.
type Board struct {
	Threshold int `json:"threshold"`
	// DefaultRef is the branch the counts are taken against, as a person
	// names it; HEAD when no default branch resolves.
	DefaultRef string `json:"default_ref"`
	Rows       []Row  `json:"rows"`
}

// StaleCount is the number of rows flagged for a re-run.
func (b Board) StaleCount() int {
	n := 0
	for _, r := range b.Rows {
		if r.Stale {
			n++
		}
	}
	return n
}

// Pin reads `review_of_commit` from a summary's leading frontmatter block.
// The value must be a bare full object name in git's lowercase hex; anything
// else, and a summary with no closed block, is no pin ("").
func Pin(summary []byte) string {
	lines := strings.Split(string(summary), "\n")
	f, ok := frontmatter.Fields(lines)["review_of_commit"]
	if !ok || !pinRe.MatchString(f.Value) {
		return ""
	}
	return f.Value
}

// Read lists the folders under root's reviews tree, sorted by name. A tree
// that does not exist is no folders and no error. Symlinked entries are not
// followed: a review folder is committed content, never a pointer elsewhere.
func Read(root string) ([]Entry, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	dir := filepath.FromSlash(Dir)
	entries, err := fs.ReadDir(r.FS(), Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if receiptRe.MatchString(name) {
			gates, err := receiptGates(r, Dir+"/"+name)
			if err != nil {
				return nil, err
			}
			out = append(out, Entry{Folder: name, Kind: KindReceipt, Scope: strings.Join(gates, ", "), ReviewOfCommit: name})
			continue
		}
		f := Entry{Folder: name, Kind: KindReview, Scope: name}
		if m := datedRe.FindStringSubmatch(name); m != nil {
			f.Scope = m[1]
		}
		if m := specRe.FindStringSubmatch(f.Scope); m != nil {
			f.Spec = m[1]
		}
		data, err := fsutil.ReadGuardedInRoot(r, filepath.Join(dir, name, summaryFile), maxSummaryBytes)
		switch {
		case err == nil:
			f.ReviewOfCommit = Pin(data)
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, fsutil.ErrNotRegular), errors.Is(err, fsutil.ErrTooBig):
			// No readable summary is no pin; RD001 refuses the missing
			// summary, and the board shows the folder unpinned.
		default:
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// receiptGates names the gates a receipt directory holds receipts for.
func receiptGates(r *os.Root, rel string) ([]string, error) {
	entries, err := fs.ReadDir(r.FS(), rel)
	if err != nil {
		return nil, err
	}
	var gates []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".json") {
			gates = append(gates, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return gates, nil
}

// Staleness reads the tree and counts each pin's age against the default
// branch (gitutil.DefaultRef, or HEAD where none resolves), one
// `git rev-list --count <pin>..<ref>` per pinned folder. Rows come stalest
// first: counted rows by descending count, then the unreachable, then the
// unpinned, each run by folder name.
func Staleness(root string) (Board, error) {
	folders, err := Read(root)
	if err != nil {
		return Board{}, err
	}
	b := Board{Threshold: StaleAfter, DefaultRef: "HEAD"}
	ref := gitutil.DefaultRef(root)
	if ref != "" {
		b.DefaultRef = gitutil.ShortRef(ref)
	} else {
		ref = "HEAD"
	}
	for _, f := range folders {
		row := Row{Entry: f, State: StateUnpinned}
		if f.ReviewOfCommit != "" {
			row.State = StateUnreachable
			// The pin is validated hex, never an option, and the ref is
			// git's own name for a commit it resolved.
			if out, err := gitutil.Run(root, "rev-list", "--count", f.ReviewOfCommit+".."+ref, "--"); err == nil {
				if n, err := strconv.Atoi(out); err == nil {
					row.State = StatePinned
					row.CommitsSince = &n
					row.Stale = n > StaleAfter
				}
			}
		}
		b.Rows = append(b.Rows, row)
	}
	rank := map[string]int{StatePinned: 0, StateUnreachable: 1, StateUnpinned: 2}
	sort.SliceStable(b.Rows, func(i, j int) bool {
		a, c := b.Rows[i], b.Rows[j]
		if rank[a.State] != rank[c.State] {
			return rank[a.State] < rank[c.State]
		}
		if a.CommitsSince != nil && c.CommitsSince != nil && *a.CommitsSince != *c.CommitsSince {
			return *a.CommitsSince > *c.CommitsSince
		}
		return a.Folder < c.Folder
	})
	return b, nil
}
