// Package peers is the read-only peer reader (itd-2609091416295622,
// spc-2609202056480020): given one checkout, which records do the other
// checkouts and branches sharing its git common dir hold that this tree does
// not, or hold in a different state?
//
// Every collision on record between two sessions on one machine was a failure
// of visibility: the peer's record sat on the same disk, and the verb that
// collided answered "not found". This package makes the holding visible and
// changes nothing. It opens no file for writing, runs only read-only git
// commands under gitutil's isolated environment, takes no lock, and stamps
// nothing — the claim, the lease and the register are other records' work.
//
// A PEER is one of two things, read through one reader:
//
//   - a linked worktree git lists for this repository (`git worktree list`),
//     confirmed from its own `--git-common-dir` and read off its disk, so an
//     uncommitted capture in it is seen;
//   - a local branch no worktree has checked out, read from the common dir with
//     `git ls-tree`, so a paused branch is seen with no checkout at all.
//
// A third source, the register (itd-2609150819440345), is where holdings live
// across accounts and machines. Its slot is the Sources list: this package ships
// with the two local sources only, and says so in every Report.
//
// A holding is an id by FILENAME, under the issue ledger's status folders and
// the intent store's buckets, read by the same filename grammar the record
// resolver reads (recordid.FilenameNumRe). A record file is opened only for its
// title, through the guarded read (fsutil.ReadGuarded: no symlink followed, no
// device blocked on, a size cap), and a title that cannot be read leaves the id
// with no title rather than failing the listing.
//
// This package is a library: it returns values and errors, and never prints.
package peers

// Source names where a peer was read from.
type Source string

const (
	// SourceWorktree is a linked worktree sharing this checkout's common dir,
	// read off its disk.
	SourceWorktree Source = "worktree"
	// SourceBranch is a local branch no worktree has checked out, read from
	// the common dir.
	SourceBranch Source = "branch"
	// SourceRegister is the slot the register intent (itd-2609150819440345)
	// fills: holdings across accounts and machines. Nothing reads it yet.
	SourceRegister Source = "register"
)

// Sources is the set of sources this build reads. The register's slot is
// deliberately absent: a consumer reading a Report can tell "no peer holds it
// anywhere" from "no LOCAL peer holds it", which is all this build can say.
var Sources = []Source{SourceWorktree, SourceBranch}

// Kind is which of the three diffs a row belongs to.
type Kind string

const (
	// KindOpenThere is a record open in the peer and absent from every status
	// folder here: the peer captured something this tree has not seen.
	KindOpenThere Kind = "open-there"
	// KindTerminalThere is a record open here and resolved or won't-fixed in the
	// peer: the peer closed something this tree still thinks is open.
	KindTerminalThere Kind = "terminal-there"
	// KindDraftThere is an intent drafted in the peer and absent from every
	// bucket here.
	KindDraftThere Kind = "draft-there"
)

// Row is one record a peer holds that this tree does not, or holds in another
// state.
type Row struct {
	ID   string `json:"id"`
	Kind Kind   `json:"kind"`
	// Folder is the folder that holds the record in the PEER: open, resolved,
	// wontfix or drafts.
	Folder string `json:"folder"`
	// Title is the record's title, empty when the file could not be read or is
	// malformed; the id is listed either way.
	Title string `json:"title,omitempty"`
}

// Peer is one live peer and what it holds.
type Peer struct {
	Source Source `json:"source"`
	// Branch is the peer's short branch name; empty for a detached worktree.
	Branch string `json:"branch,omitempty"`
	// Path is the worktree's absolute path as git reports it; empty for a
	// branch peer. A surface redacts it before it reaches a stream.
	Path string `json:"path,omitempty"`
	// NotRead is set, with the reason, when the peer was named and not read:
	// git refused to answer for it, its common dir is not this checkout's, its
	// ledger holds one id twice, or it holds no ledger at the committed layout.
	// A peer that was not read carries no rows.
	NotRead string `json:"not_read,omitempty"`
	Rows    []Row  `json:"rows"`

	holdings holdings
}

// Skip reasons: a spent peer contributes no rows and is counted instead.
const (
	SkipGone   = "gone"
	SkipMerged = "merged"
)

// Skipped is a spent peer: its worktree directory is gone, or its branch is
// merged into the default branch (and, for a worktree, its record folders hold
// no uncommitted change).
type Skipped struct {
	Source Source `json:"source"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

// Report is the whole picture for one checkout.
type Report struct {
	// Sources is the set of sources read (see Sources).
	Sources []Source `json:"sources"`
	// DefaultRef is the ref a branch is judged merged into, empty when none
	// could be resolved (then nothing is skipped as merged).
	DefaultRef string    `json:"default_ref,omitempty"`
	Peers      []Peer    `json:"peers"`
	Skipped    []Skipped `json:"skipped"`

	here holdings
	root string
}

// Live is the number of live peers: those read, and those named but not read.
func (r Report) Live() int { return len(r.Peers) }

// IDCount is the number of distinct record ids across every peer's rows.
func (r Report) IDCount() int {
	seen := map[string]bool{}
	for _, p := range r.Peers {
		for _, row := range p.Rows {
			seen[row.ID] = true
		}
	}
	return len(seen)
}
