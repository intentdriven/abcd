package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// statusStore is the noun the checkout resolution names in its refusal.
const statusStore = "the status"

// statusPushers is the forge adapter `abcd status` asks who pushed each
// announcement (C17): the activity record of pushes to the branch.
var statusPushers claim.PusherLookup = func(root, branch string) (map[string]string, error) {
	return github.Pushers(root, claim.StatusRefPrefix+branch)
}

// statusOutput is the --json document of `abcd status`: the status view, the
// announcements, the doubtful branches, what could not be read, and the
// reservations block `abcd peers` lists, in the same fields.
type statusOutput struct {
	StatusView       statusView        `json:"status_view"`
	Announcements    []announcementRow `json:"announcements"`
	Doubtful         []doubtRow        `json:"doubtful"`
	Unread           []string          `json:"unread"`
	SendersUnchecked string            `json:"senders_unchecked,omitempty"`
	ReadError        string            `json:"read_error,omitempty"`
	Grants           string            `json:"grants"`
	reservationsBlock
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
			"each release being cut, which holds every other session's merge.\n\n" +
			"It reads the remote once: it lists the status branches on origin (git\n" +
			"ls-remote), then fetches, in one command, the reservations with origin's\n" +
			"release tags, each status branch whose name is a GitHub username in lower\n" +
			"case, and origin's default branch, and stamps when in the local tier. A\n" +
			"branch under abcd-status/ by any other name is named as doubtful and not\n" +
			"fetched. Then it asks the forge (gh api, the repository's activity record)\n" +
			"who pushed each standing announcement it has not asked about yet: a sender\n" +
			"is verified when the forge names the branch's owner, doubtful when it names\n" +
			"someone else, and claimed when the forge cannot say.\n\n" +
			"An announcement stands for its life from when this checkout first fetched it,\n" +
			"not by the sender's clock. One whose sender is not its branch's owner, one\n" +
			"that does not read as an announcement (never shown but by its branch and the\n" +
			"reason), and a branch that was rewritten, passed a merge or gained more than\n" +
			"one read covers are marked doubtful, and every other branch is still read.\n" +
			"An announcement informs and grants nothing.\n\n" +
			"When the fetch fails, the status is listed as last fetched, with that view's\n" +
			"age and a line saying it is unread since then and why. Exit 0 when the fetch\n" +
			"succeeded and with no remote named origin; exit 1 when it failed; exit 2\n" +
			"outside a git checkout.",
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
			out := statusOutput{Grants: claim.GrantsNothing}
			failed := out.reservationsBlock.fill(root, now, fetched.Gap, ferr)
			out.SendersUnchecked = termsafe.Sanitize(fsutil.RedactHome(fetched.SendersUnchecked))
			if out.View.Origin {
				v, rerr := claim.ReadStatus(context.Background(), root, now, o)
				if rerr != nil {
					out.ReadError = termsafe.Sanitize(fsutil.RedactHome(rerr.Error()))
					if failed == "" {
						failed = out.ReadError
					}
				}
				out.addStatus(v, now)
			}
			out.Announcements = nonNil(out.Announcements)
			out.Doubtful = nonNil(out.Doubtful)
			out.Unread = nonNil(out.Unread)
			if err := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderStatus(w, out) }); err != nil {
				return err
			}
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
		out.Doubtful = append(out.Doubtful, doubtRow{Branch: termsafe.Sanitize(d.Branch), Reason: d.Reason, Tip: d.Tip, Text: claim.DoubtLine(d)})
	}
	for _, u := range []struct {
		what string
		u    *claim.Unread
	}{{"status", v.Unread}, {"reservations", v.ReservationsUnread}} {
		if u.u != nil {
			out.Unread = append(out.Unread, claim.UnreadLine(u.what, *u.u))
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
	if len(out.Announcements) == 0 && out.ReadError == "" {
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
	if out.SendersUnchecked != "" {
		fmt.Fprintln(w, "  not checked against the forge, so shown as claimed: "+out.SendersUnchecked)
	}
	renderReservations(w, out.reservationsBlock)
	fmt.Fprintln(w, out.Grants)
}
