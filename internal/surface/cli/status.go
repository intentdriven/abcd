package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// statusStore is the noun the checkout resolution names in its refusal.
const statusStore = "the status"

// localTierDir is the local tier, as `abcd status` names it when it is
// missing.
const localTierDir = ".abcd/.work.local/"

// statusPushers is the forge adapter `abcd status` asks who pushed each
// announcement (C17): the activity record of pushes to the branch, answered
// with claim.ErrPartialRecord beside the pushes when its page came back full.
var statusPushers claim.PusherLookup = func(ctx context.Context, root, branch string) ([]claim.Push, error) {
	pushes, partial, err := github.Pushers(ctx, root, claim.StatusRefPrefix+branch)
	if err != nil {
		return nil, err
	}
	out := make([]claim.Push, 0, len(pushes))
	for _, p := range pushes {
		out = append(out, claim.Push{Before: p.Before, After: p.After, Pusher: p.Login, App: p.App})
	}
	if partial {
		return out, claim.ErrPartialRecord
	}
	return out, nil
}

// statusScheduledRun is the forge adapter `abcd status` asks for the latest
// completed scheduled run of the workflow the repository names.
var statusScheduledRun claim.ScheduledRunLookup = func(ctx context.Context, root, workflow, branch string) ([]claim.ScheduledRun, error) {
	runs, err := github.ScheduledRuns(ctx, root, workflow, branch)
	if err != nil {
		return nil, err
	}
	out := make([]claim.ScheduledRun, 0, len(runs))
	for _, r := range runs {
		out = append(out, claim.ScheduledRun{Conclusion: r.Conclusion, URL: r.URL, CompletedAt: r.UpdatedAt})
	}
	return out, nil
}

// statusOutput is the --json document of `abcd status`: the status view, the
// announcements, the doubtful branches, the count of status branches whose
// names are not read (never the names: a pusher chose each, and an agent
// reads this output; null when the count could not be read), the counts of
// status branches and kept announcements past what one read takes, what could
// not be read, and the reservations block
// `abcd peers` lists, in the same fields.
type statusOutput struct {
	StatusView       statusView        `json:"status_view"`
	Announcements    []announcementRow `json:"announcements"`
	Doubtful         []doubtRow        `json:"doubtful"`
	UnreadNames      *int              `json:"unread_names"`
	BranchesNotRead  int               `json:"branches_not_read"`
	BranchesLeft     int               `json:"branches_left_to_read"`
	NotRead          int               `json:"announcements_not_read"`
	Unread           []string          `json:"unread"`
	SendersUnchecked string            `json:"senders_unchecked,omitempty"`
	NotDropped       string            `json:"not_dropped,omitempty"`
	LocalTierMissing bool              `json:"local_tier_missing,omitempty"`
	ReadError        string            `json:"read_error,omitempty"`
	// ScheduledFailure is the repository's last scheduled verdict when it was
	// red, ScheduledDoubt what is said when there is no verdict to name, and
	// ScheduledUnread why the runs could not be read; each absent otherwise.
	ScheduledFailure *scheduledFailureRow `json:"scheduled_failure,omitempty"`
	ScheduledDoubt   *scheduledDoubtRow   `json:"scheduled_doubt,omitempty"`
	ScheduledUnread  string               `json:"scheduled_unread,omitempty"`
	Grants           string               `json:"grants"`
	reservationsBlock
}

// scheduledFailureRow is a failed latest scheduled run as the listing shows it.
type scheduledFailureRow struct {
	Workflow    string `json:"workflow"`
	Branch      string `json:"branch"`
	Conclusion  string `json:"conclusion"`
	URL         string `json:"url"`
	CompletedAt string `json:"completed_at"`
	AgeSeconds  int64  `json:"age_seconds"`
	RunsSince   int    `json:"runs_since,omitempty"`
	Text        string `json:"text"`
}

// scheduledDoubtRow is what is said beside or in place of a verdict: its kind
// (ended_without_verdict, no_verdict, unknown or invalid_name) and its line.
type scheduledDoubtRow struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// statusView dates the announcements: when abcd last fetched the status
// branches here and how long ago, both absent when nothing here stamped a
// fetch, and whether the checkout has a remote named origin.
type statusView struct {
	Origin     bool   `json:"origin"`
	FetchedAt  string `json:"fetched_at,omitempty"`
	AgeSeconds *int64 `json:"age_seconds"`
}

// announcementRow is one announcement as the listing shows it: its validated
// fields, how its sender is judged, and the line it is shown by.
type announcementRow struct {
	Branch       string `json:"branch"`
	Commit       string `json:"commit"`
	Kind         string `json:"kind,omitempty"`
	Item         string `json:"item,omitempty"`
	Sender       string `json:"sender,omitempty"`
	SentAt       string `json:"sent_at,omitempty"`
	FirstFetched string `json:"first_fetched"`
	Until        string `json:"until,omitempty"`
	Check        string `json:"check"`
	Doubtful     string `json:"doubtful,omitempty"`
	Pusher       string `json:"pusher,omitempty"`
	Unreadable   string `json:"unreadable,omitempty"`
	Text         string `json:"text"`
}

// doubtRow is one doubtful status branch: its name, terminal-safe, the
// reason and the line it is shown by.
type doubtRow struct {
	Branch string `json:"branch"`
	Reason string `json:"reason"`
	Tip    string `json:"tip,omitempty"`
	Text   string `json:"text"`
}

// newStatusCommand builds the `status` verb — the people's view of what
// everyone else is doing (itd-2609151838312703 decision 21, C16): the
// announcements every session posted, who holds which reservation and any
// release being cut, after one fetch.
func newStatusCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "status",
		Long: "List everyone's status on the shared remote: the announcements every session\n" +
			"posted (abcd announce), newest first, each with its sender, kind, item, the\n" +
			"time sent and how long it still stands, then who holds which reservation and\n" +
			"each release being cut, which holds every other session's merge, with a live\n" +
			"drain lane's stage and each holder's hand-backs from the last day.\n\n" +
			"It reads the remote once: it lists the status branches on origin (git\n" +
			"ls-remote), then fetches, in one command, the reservations with origin's\n" +
			"release tags, the first 500 status branches whose names are holder ids or\n" +
			"GitHub usernames in lower case (the names before the holder id), and\n" +
			"origin's default branch, and stamps when in\n" +
			"the local tier. A branch under abcd-status/ by any other name is counted,\n" +
			"never named, and not fetched; branches past the 500 are counted and not\n" +
			"fetched. Before the fetch, this checkout's copy of each status branch that\n" +
			"is not among the names it fetches is deleted, so a branch deleted on\n" +
			"origin, one whose name is not read and one past the 500 leave nothing to\n" +
			"read; a copy that cannot be deleted is named, and the fetch goes on. Then\n" +
			"it asks the forge (gh api, the repository's activity record) who pushed\n" +
			"each standing announcement it has not asked about yet, on at most ten\n" +
			"branches named by a username, the one waiting longest first: a sender is\n" +
			"verified when the forge names the branch's owner, doubtful when it names\n" +
			"someone else, and claimed when the forge cannot say or an app pushed it. A\n" +
			"branch named by a holder id names no account the forge's record could\n" +
			"match, so it is not asked about and its senders stay claimed.\n\n" +
			"A repository that names a workflow a schedule runs on its default branch\n" +
			"(status.scheduled_workflow in the .abcd/config.json at that branch's tip, the\n" +
			"file's name, such as nightly.yml) has that workflow's latest scheduled verdict\n" +
			"read too: after a fetch that succeeded it asks the forge (gh api) within five\n" +
			"seconds for the latest completed scheduled runs and keeps what they say in the\n" +
			"machine's store for the repository, which every checkout of it reads; the\n" +
			"prompt hook does the same in the background when that answer is an hour old. A\n" +
			"run that concluded failure, timed_out or startup_failure is named in one line\n" +
			"with its URL and age, here and once to each session at a prompt, until a run\n" +
			"that succeeds replaces it. A latest run that concluded cancelled, skipped or\n" +
			"neutral, which is how a job past its time limit arrives, is not called a failure:\n" +
			"it has a line of its own naming it and the last verdict, until a verdict\n" +
			"replaces it. A green is silent while it is recent for the workflow's own cycle,\n" +
			"and past that one line says no verdict has been reached. An answer that is\n" +
			"missing or more than two days old, or due after a refresh that failed or, some\n" +
			"hours on, kept nothing, says the state is unknown, naming the last successful\n" +
			"ask; an answer that only aged through an idle night says nothing, and the next\n" +
			"prompt carries what the refresh found. None of this changes an exit. With no\n" +
			"such key nothing is asked, kept or shown; a key that could not name a workflow's\n" +
			"file, and a config that could not be read here, each say so.\n\n" +
			"An announcement stands for its life from when this checkout first fetched it,\n" +
			"not by the sender's clock. One whose sender is not its branch's owner, one\n" +
			"that does not read as an announcement (never shown but by its branch and the\n" +
			"reason), and a branch that was rewritten, passed a merge or gained more than\n" +
			"one read covers are marked doubtful, and every other branch is still read.\n" +
			"A running session sees each new line once, at its next prompt, in a\n" +
			"block headed STATUS of at most ten lines and one more that counts the\n" +
			"rest, read from disk alone.\n" +
			"An announcement informs and grants nothing.\n\n" +
			"When " + abcdhome.Display("holder-ids") + " is there and its guarded read refuses it, no\n" +
			"declared id is read as yours, and one line on stderr names the class of the\n" +
			"refusal, never the file's contents (holder_ids_refused in --json).\n\n" +
			"When your holder id is set or kept but refused (ABCD_HOLDER_ID set to\n" +
			"anything but one holder id, or " + abcdhome.Display("holder-id") + " refused by its\n" +
			"guarded read), no hold or withdrawal under it is read as yours, and one line\n" +
			"on stderr names the class of the refusal, never the value or the file's\n" +
			"contents (holder_id_refused in --json).\n\n" +
			"When the fetch fails, the status is listed as last fetched, with that view's\n" +
			"age and a line saying it is unread since then and why. Exit 0 when the fetch\n" +
			"succeeded and with no remote named origin; exit 1 when it failed, when\n" +
			"what it fetched could not be read, or in a checkout without its local\n" +
			"tier; exit 2 outside a git checkout or with an operand.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return &exitError{Code: 2, Msg: "abcd status: takes no operand; it lists everyone's status (nothing read)"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, statusStore)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd status: " + err.Error() + " (nothing read)"}
			}
			now := time.Now().UTC().Truncate(time.Second)
			o := claim.Options{Now: func() time.Time { return now }}
			fetched, ferr := claim.FetchStatus(root, statusPushers, o)
			out := statusOutput{Grants: claim.GrantsNothing, UnreadNames: new(int)}
			if ferr == nil {
				out.ScheduledUnread = claim.RefreshScheduledRun(root, statusScheduledRun, o)
			}
			// One read of the reservations (View, in the block) and one of
			// the holds (ReadStatus's), shared by the listing and the view.
			failed, listed := out.reservationsBlock.fillView(root, now, fetched.Gap, ferr)
			out.SendersUnchecked = termsafe.Sanitize(fsutil.RedactHome(fetched.SendersUnchecked))
			out.NotDropped = termsafe.Sanitize(fsutil.RedactHome(fetched.NotDropped))
			if out.View.Origin {
				v, rerr := claim.ReadStatus(context.Background(), root, now, o)
				if rerr != nil {
					out.ReadError = termsafe.Sanitize(fsutil.RedactHome(rerr.Error()))
					if failed == "" {
						failed = out.ReadError
					}
				}
				if listed {
					out.reservationsBlock.fillHolds("", v.Holds, nil)
				}
				out.addStatus(v, now)
				if v.NoTier && failed == "" {
					failed = "this checkout has no local tier (" + localTierDir + "), so nothing a fetch brings is kept or read here"
				}
			}
			out.Announcements = nonNil(out.Announcements)
			out.Doubtful = nonNil(out.Doubtful)
			out.Unread = nonNil(out.Unread)
			if err := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderStatus(w, out) }); err != nil {
				return err
			}
			noteHolderIDsRefused(cmd.ErrOrStderr(), "abcd status", out.HolderIDsRefused)
			noteHolderIDRefused(cmd.ErrOrStderr(), "abcd status", out.HolderIDRefused)
			if failed != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "abcd status: "+failed)
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

// nonNil is s, or an empty slice for nil, so --json lists none as [].
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// addStatus adds v's announcements, doubts and unread lines to out.
func (out *statusOutput) addStatus(v claim.StatusView, now time.Time) {
	out.StatusView.Origin = v.Origin
	out.LocalTierMissing = v.NoTier
	if !v.FetchedAt.IsZero() {
		out.StatusView.FetchedAt = rfc3339(v.FetchedAt)
		age := int64(now.Sub(v.FetchedAt).Seconds())
		out.StatusView.AgeSeconds = &age
	}
	for _, a := range v.Announcements {
		row := announcementRow{Branch: "abcd-status/" + a.Branch, Commit: a.Commit, Kind: a.Kind, Item: a.Item, Sender: a.Sender,
			FirstFetched: rfc3339(a.FirstFetched), Check: a.Check, Doubtful: a.Doubtful, Pusher: a.Pusher,
			Unreadable: a.Unreadable, Text: a.Line(now)}
		if a.Unreadable == "" {
			row.SentAt, row.Until = rfc3339(a.SentAt), rfc3339(a.Until)
		}
		out.Announcements = append(out.Announcements, row)
	}
	for _, d := range v.Doubts {
		out.Doubtful = append(out.Doubtful, doubtRow{Branch: "abcd-status/" + d.Branch, Reason: d.Reason, Tip: d.Tip, Text: claim.DoubtLine(d)})
	}
	if f := v.ScheduledFailure; f != nil {
		out.ScheduledFailure = &scheduledFailureRow{Workflow: f.Workflow, Branch: f.Branch, Conclusion: f.Conclusion, URL: f.URL,
			CompletedAt: rfc3339(f.CompletedAt), AgeSeconds: max(int64(now.Sub(f.CompletedAt).Seconds()), 0), RunsSince: f.RunsSince, Text: f.Line(now)}
	}
	if d := v.ScheduledDoubt; d != nil {
		out.ScheduledDoubt = &scheduledDoubtRow{Kind: d.Kind, Text: d.Text}
	}
	out.BranchesNotRead, out.NotRead, out.BranchesLeft = v.BranchesNotRead, v.AnnouncementsNotRead, v.BranchesLeft
	n := v.UnreadNames
	out.UnreadNames = &n
	if v.UnreadNamesUnknown {
		out.UnreadNames = nil
	}
	for _, u := range []struct {
		what string
		u    *claim.Unread
	}{{"status", v.Unread}, {"reservations", v.ReservationsUnread}} {
		if u.u != nil {
			out.Unread = append(out.Unread, claim.UnreadReasonLine(u.what, *u.u))
		}
	}
}

// renderStatus is the text form: the announcements under a heading that
// dates them, the doubtful branches, then the reservations block.
func renderStatus(w io.Writer, out statusOutput) {
	if !out.View.Origin {
		fmt.Fprintln(w, "abcd status — this checkout has no remote named origin, so there is no status to read")
		return
	}
	switch {
	case out.StatusView.AgeSeconds == nil:
		fmt.Fprintln(w, "abcd status — announcements; nothing has fetched them here, so the age of this view is unknown")
	default:
		fmt.Fprintf(w, "abcd status — announcements as of the fetch %s ago (%s)\n", ageText(*out.StatusView.AgeSeconds), out.StatusView.FetchedAt)
	}
	for _, u := range out.Unread {
		fmt.Fprintln(w, "  "+u)
	}
	if out.ReadError != "" {
		fmt.Fprintln(w, "  not read: "+out.ReadError)
	}
	if out.LocalTierMissing {
		fmt.Fprintln(w, "  this checkout has no local tier ("+localTierDir+"), so nothing a fetch brings is kept or read here; `abcd ahoy install` lays it")
	}
	if len(out.Announcements) == 0 && out.ReadError == "" && !out.LocalTierMissing {
		// An empty list is said only after a fetch that succeeded (C10).
		if out.StatusView.AgeSeconds != nil && len(out.Unread) == 0 {
			fmt.Fprintln(w, "  none standing")
		} else {
			fmt.Fprintln(w, "  none read")
		}
	}
	for _, a := range out.Announcements {
		line := "  " + a.Text
		if a.SentAt != "" {
			line += "  sent " + a.SentAt
		}
		fmt.Fprintln(w, line)
	}
	for _, d := range out.Doubtful {
		fmt.Fprintln(w, "  "+d.Text)
	}
	// Counted, never named: git itself lists the names, outside this
	// output, for a person who wants them.
	const listNames = "`git ls-remote origin 'refs/heads/abcd-status/*'` lists every name"
	switch {
	case out.UnreadNames == nil:
		fmt.Fprintln(w, "  "+claim.UnreadNamesUnknownLine+": "+listNames)
	case out.UnreadNames != nil && *out.UnreadNames > 0:
		fmt.Fprintln(w, "  "+claim.UnreadNamesLine(*out.UnreadNames)+", so not shown: "+listNames)
	}
	if out.BranchesNotRead > 0 {
		fmt.Fprintln(w, "  "+claim.BranchesNotReadLine(out.BranchesNotRead))
	}
	if out.NotRead > 0 {
		fmt.Fprintln(w, "  "+claim.AnnouncementsNotReadLine(out.NotRead))
	}
	if out.BranchesLeft > 0 {
		fmt.Fprintln(w, "  "+claim.BranchesLeftLine(out.BranchesLeft))
	}
	if out.NotDropped != "" {
		fmt.Fprintln(w, "  not deleted: "+out.NotDropped)
	}
	if out.SendersUnchecked != "" {
		fmt.Fprintln(w, "  not checked against the forge, so shown as claimed: "+out.SendersUnchecked)
	}
	if out.ScheduledFailure != nil {
		fmt.Fprintln(w, "  "+out.ScheduledFailure.Text)
	}
	if d := out.ScheduledDoubt; d != nil {
		fmt.Fprintln(w, "  "+d.Text)
	}
	// An invalid key is one fact, said by both.
	if out.ScheduledUnread != "" && (out.ScheduledDoubt == nil || out.ScheduledUnread != out.ScheduledDoubt.Text) {
		fmt.Fprintln(w, "  "+out.ScheduledUnread)
	}
	renderReservations(w, out.reservationsBlock)
	fmt.Fprintln(w, out.Grants)
}

// hookBudget is the prompt hook's whole budget (C12), counted from the hook's
// start: the status read takes what the drain leaves of it, before the rules
// and the memory notes, at most 600 ms (claim.PromptNotes), and leaves the hook time to
// write the block and mark it shown. A test lowers it.
var hookBudget = time.Second

// routerStatus is the prompt hook's status block for the checkout at root,
// read, rendered and kept shown before deadline, the hook's one deadline
// (claim.PromptNotes). A read that could not be made is named on stderr, and
// its block says the status is unavailable. The caller marks the block shown
// once it has written it. A refresh the block says is due is started here, in
// the background, and never waited for.
func routerStatus(cmd *cobra.Command, root, session string, rootCommit func() (string, error), deadline time.Time) claim.Notes {
	if root == "" {
		return claim.Notes{}
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Second)
	o := claim.Options{Now: func() time.Time { return now }, Session: session, RootCommit: rootCommit}
	n, err := claim.PromptNotes(ctx, root, now, o)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd status: %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	}
	if n.RefreshDue {
		startDueRefresh(cmd, n, root, now, deadline)
	}
	return n
}

// joinBlocks is the injected text with the status block after it, one blank
// line between them.
func joinBlocks(text, block string) string {
	switch {
	case block == "":
		return text
	case text == "":
		return block
	}
	return strings.TrimRight(text, "\n") + "\n\n" + block
}
