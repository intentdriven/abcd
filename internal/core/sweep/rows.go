package sweep

// rows.go is what a sweep renders (spc-2610230730269295, Rows): one Row per
// item, each with the action `--apply` would take or the reason it leaves the
// item, and the exit code a run maps to.

import "strings"

// RowKind is what a row is about: worktrees, strays and the embargo store,
// local branches, branches on origin, reservations and inbox reports.
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
	// RowBranch is a local branch (branches.go).
	RowBranch RowKind = "branch"
	// RowRemoteBranch is a branch on origin, as this checkout last fetched it
	// (remote.go).
	RowRemoteBranch RowKind = "remote-branch"
	// RowReservation is a reservation branch on origin, as the reservation
	// sweep judged it (reservations.go).
	RowReservation RowKind = "reservation"
	// RowInbox is a report waiting in abcd's inbox (inbox.go).
	RowInbox RowKind = "inbox"
)

// Action is what `--apply` does with a row's item: remove, save-patch and keep
// for worktrees, delete for branches, dispatch for spent reservations and
// promote for waiting inbox reports.
type Action string

const (
	// ActionRemove is a merged, clean worktree nothing holds, reclaimed
	// through the store's prune.
	ActionRemove Action = "remove"
	// ActionSavePatch is a dirty worktree nothing holds: its changes are saved
	// as a patch and the worktree is kept.
	ActionSavePatch Action = "save-patch"
	// ActionDelete is a merged branch nothing holds: a local one deleted with
	// `git branch -d`, a branch on origin deleted under a lease on the tip
	// that was judged.
	ActionDelete Action = "delete"
	// ActionDispatch is a spent reservation: apply dispatches the reservation
	// sweep's workflow, which deletes it. The sweep never deletes one itself.
	ActionDispatch Action = "dispatch"
	// ActionPromote is a waiting inbox report: apply files it as an issue
	// through the inbox's promote, with the automatic filer's remedy.
	ActionPromote Action = "promote"
	// ActionKeep leaves the item as it is, its reason saying why.
	ActionKeep Action = "keep"
)

// Outcome is what `--apply` did with a row's item.
type Outcome string

const (
	// OutcomeRemoved is a worktree the store's prune reclaimed.
	OutcomeRemoved Outcome = "removed"
	// OutcomeSaved is a dirty worktree whose changes were saved as a patch;
	// the worktree itself is kept.
	OutcomeSaved Outcome = "saved"
	// OutcomeDeleted is a branch the delete took.
	OutcomeDeleted Outcome = "deleted"
	// OutcomeDispatched is a spent reservation the reservation sweep's
	// workflow was dispatched to delete.
	OutcomeDispatched Outcome = "dispatched"
	// OutcomePromoted is an inbox report filed as an issue.
	OutcomePromoted Outcome = "promoted"
	// OutcomeKept is an item left as it was, its reason saying why: one the
	// dry run kept, or one a check at the moment of acting kept.
	OutcomeKept Outcome = "kept"
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
	// KindProtected is a branch on the never-delete list: the default branch,
	// a release branch, a reservation or a status branch.
	KindProtected EvidenceKind = "protected"
	// KindUpstream is a local branch's standing against its upstream.
	KindUpstream EvidenceKind = "upstream"
	// KindPullRequest is the forge's word on a branch's pull requests: an
	// open one proves it held, and a forge that could not be asked is a hint.
	KindPullRequest EvidenceKind = "pull-request"
	// KindDeleteProof is a check of the shell guard's delete proof that a
	// branch fails: one of abcd's coordination branches, or one a worktree
	// has checked out or a stopped rebase or bisect holds.
	KindDeleteProof EvidenceKind = "delete-proof"
	// KindRemote is origin's own answer at the last check before a delete:
	// its default branch and the branch's tip there.
	KindRemote EvidenceKind = "remote"
	// KindReservationSweep is the reservation sweep's judgement of a
	// reservation branch, and KindWorkflow the dispatch of its workflow.
	KindReservationSweep EvidenceKind = "reservation-sweep"
	KindWorkflow         EvidenceKind = "workflow"
	// KindReport is what the inbox read of a report, and KindPromote the
	// inbox's promote of it.
	KindReport  EvidenceKind = "report"
	KindPromote EvidenceKind = "promote"
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
	// Outcome is what an applied run did with the item; "" in a dry run.
	Outcome Outcome `json:"outcome,omitempty"`
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

// Result is one sweep's answer. The process check, the reservation read and
// the build-run read each fail closed, holding every worktree; when one could
// not be made, the result says so once, here, as well as on every row it holds.
type Result struct {
	// Lane is the repository's lane in the worktree store, home-redacted.
	Lane string `json:"lane,omitempty"`
	Rows []Row  `json:"rows"`
	// ProcessCheckUnavailable is why the process check could not be made.
	ProcessCheckUnavailable string `json:"process_check_unavailable,omitempty"`
	// ReservationsUnread is why the reservations cannot show that nothing is
	// reserved.
	ReservationsUnread string `json:"reservations_unread,omitempty"`
	// RunsUnread is why the build runs could not be read.
	RunsUnread string `json:"runs_unread,omitempty"`
	// ForgeNote is the store's note that no forge answered.
	ForgeNote string `json:"forge_note,omitempty"`
	// Workflow is what became of the reservation sweep's workflow: the run a
	// dispatch started, or why none was dispatched and when the workflow's
	// schedule runs it next.
	Workflow string `json:"workflow,omitempty"`
	// Refusal is why the run refused or faulted; "" when it completed.
	Refusal string `json:"refusal,omitempty"`
	// Applied marks a run that acted, each row carrying its Outcome; a dry
	// run leaves it false.
	Applied bool `json:"applied,omitempty"`
}

// Notes are the lines a rendering puts once at the top: each failed-closed
// check, then the forge note, then the reservation workflow's.
func (r Result) Notes() []string {
	var out []string
	if r.ProcessCheckUnavailable != "" {
		out = append(out, "process check unavailable: "+r.ProcessCheckUnavailable)
	}
	if r.ReservationsUnread != "" {
		out = append(out, "reservations unread: "+r.ReservationsUnread)
	}
	if r.RunsUnread != "" {
		out = append(out, "build runs unread: "+r.RunsUnread)
	}
	if r.ForgeNote != "" {
		out = append(out, r.ForgeNote)
	}
	if r.Workflow != "" {
		out = append(out, r.Workflow)
	}
	return out
}

// ExitCode is the code the front door exits with: ExitRefused on a refusal or
// fault, whatever rows came before it; otherwise, for an applied run,
// ExitActed when any row's outcome is other than kept, and for a dry run when
// any row carries an action apply would take; ExitNothing when none does. A
// save-patch row is one: the dry run has already collected its salvage, and
// one whose changes are saved unchanged by an earlier run, or cannot be
// saved, is a keep, so a dry run never promises what apply skips.
func (r Result) ExitCode() int {
	if strings.TrimSpace(r.Refusal) != "" {
		return ExitRefused
	}
	for _, row := range r.Rows {
		switch {
		case r.Applied && row.Outcome != OutcomeKept && row.Outcome != "":
			return ExitActed
		case !r.Applied && row.Action != ActionKeep && row.Action != "":
			return ExitActed
		}
	}
	return ExitNothing
}
