package sweep

// rows.go is what a sweep renders (spc-2610230730269295, Rows): one Row per
// item, each with the action `--apply` would take or the reason it leaves the
// item, and the exit code a run maps to.

import "strings"

// RowKind is what a row is about. This step renders worktrees, strays and the
// embargo store; the branch, remote-branch, reservation and inbox kinds join
// them as the steps that judge those items land.
type RowKind string

const (
	// RowWorktree is a worktree git lists, in the store or outside it, or a
	// directory in the store's lane that fails its proof of belonging.
	RowWorktree RowKind = "worktree"
	// RowStray is an entry in the store's lane that is not a directory: a
	// loose file, say. It is listed and never touched.
	RowStray RowKind = "stray"
	// RowEmbargoStore is abcd's embargo store, never read, saved or removed.
	RowEmbargoStore RowKind = "embargo-store"
)

// Action is what `--apply` does with a row's item. This step judges
// worktrees, so it renders remove, save-patch and keep; delete, dispatch and
// promote join them with the kinds they act on.
type Action string

const (
	// ActionRemove is a merged, clean worktree nothing holds, reclaimed
	// through the store's prune.
	ActionRemove Action = "remove"
	// ActionSavePatch is a dirty worktree nothing holds: its changes are saved
	// as a patch and the worktree is kept.
	ActionSavePatch Action = "save-patch"
	// ActionKeep leaves the item as it is, its reason saying why.
	ActionKeep Action = "keep"
)

// The evidence kinds a row adds to the liveness probe's: what the store read
// of the worktree, each a proof, since git said it.
const (
	// KindStore is the store's own word on an item it did not judge: the
	// embargo store, a stray, or a worktree failing the proof of belonging.
	KindStore EvidenceKind = "store"
	// KindLock is git's lock on the worktree, with its reason.
	KindLock EvidenceKind = "lock"
	// KindState is clean or dirty, and KindMerged the merged judgement.
	KindState  EvidenceKind = "state"
	KindMerged EvidenceKind = "merged"
	// KindLoop is the build loop's hold as the store's prune reads it, and
	// KindCwd the sweep's own working directory.
	KindLoop EvidenceKind = "loop"
	KindCwd  EvidenceKind = "cwd"
)

// Row is one item a sweep judged. Every string in it is display-ready: a path
// home-redacted the way the store's own rows are, and any text another
// checkout or a reservation's writer chose passed through termsafe.Sanitize.
type Row struct {
	Kind RowKind `json:"kind"`
	Name string  `json:"name"`
	Path string  `json:"path"`
	// State is what the sweep read of the item, in words.
	State  string `json:"state"`
	Action Action `json:"action"`
	// Reason says why the item is kept, or why the action is safe.
	Reason string `json:"reason"`
	// Evidence is each thing read that bears on the action, every piece
	// marked proven or hint.
	Evidence []Evidence `json:"evidence,omitempty"`
}

// The exit codes a sweep maps to (spc-2610230730269295, Rows).
const (
	// ExitActed is a run that acted on something, or a dry run that would.
	ExitActed = 0
	// ExitNothing is a run that completed and acted, or would act, on nothing.
	ExitNothing = 1
	// ExitRefused is a refusal or a fault.
	ExitRefused = 2
)

// Result is one sweep's answer. The process check and the reservation read
// each fail closed, holding every worktree; when either could not be made, the
// result says so once, here, as well as on every row it holds.
type Result struct {
	// Lane is the repository's lane in the worktree store, home-redacted.
	Lane string `json:"lane,omitempty"`
	Rows []Row  `json:"rows"`
	// ProcessCheckUnavailable is why the process check could not be made.
	ProcessCheckUnavailable string `json:"process_check_unavailable,omitempty"`
	// ReservationsUnread is why the reservations cannot show that nothing is
	// reserved.
	ReservationsUnread string `json:"reservations_unread,omitempty"`
	// ForgeNote is the store's note that no forge answered.
	ForgeNote string `json:"forge_note,omitempty"`
	// Refusal is why the run refused or faulted; "" when it completed.
	Refusal string `json:"refusal,omitempty"`
}

// Notes are the lines a rendering puts once at the top: each failed-closed
// check, then the forge note.
func (r Result) Notes() []string {
	var out []string
	if r.ProcessCheckUnavailable != "" {
		out = append(out, "process check unavailable: "+r.ProcessCheckUnavailable)
	}
	if r.ReservationsUnread != "" {
		out = append(out, "reservations unread: "+r.ReservationsUnread)
	}
	if r.ForgeNote != "" {
		out = append(out, r.ForgeNote)
	}
	return out
}

// ExitCode is the code the front door exits with: ExitRefused on a refusal or
// fault, whatever rows came before it; otherwise ExitActed when any row
// carries an action other than keep, and ExitNothing when none does.
func (r Result) ExitCode() int {
	if strings.TrimSpace(r.Refusal) != "" {
		return ExitRefused
	}
	for _, row := range r.Rows {
		if row.Action != ActionKeep && row.Action != "" {
			return ExitActed
		}
	}
	return ExitNothing
}
