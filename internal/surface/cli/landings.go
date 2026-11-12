package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/landings"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// landingsStore is the noun the checkout resolution names in its refusal.
const landingsStore = "the landings listing"

// landingsForge is the forge client the verb hands the reader; nil is the
// reader's own default (gh, bound to the listing's time). A test sets a fake.
var landingsForge landing.ForgeCall

// The render's own bounds: every value a row prints is cut at
// landingsFieldBytes, and every note at landingsNoteBytes, after the reader's
// own bounds (internal/core/landings), so one line is bounded whatever the
// reader hands it.
const (
	landingsFieldBytes = 200
	landingsNoteBytes  = 600
)

// landingsOutput is the listing's --json: the reader's Listing, its rows kept
// to the unattended ones under --unattended, and under --reverted the groups
// the text prints.
type landingsOutput struct {
	landings.Listing
	Groups []landings.RevertGroup `json:"groups,omitempty"`
}

// newLandingsCommand builds the `landings` verb, the front door onto
// internal/core/landings (itd-2610301850454860, spc-2610301950480609
// approach 8): how each change landed on the default branch, with the rule
// that admitted it. It reads history as last fetched and the forge once per
// pull request, and writes only the reader's forge cache in the local tier.
func newLandingsCommand(asJSON *bool) *cobra.Command {
	var since string
	var unattended, reverted bool
	cmd := &cobra.Command{
		Use: "landings [commit]",
		Long: "List how each change landed on the default branch, newest first: one row per\n" +
			"commit on the default branch's first-parent line, as this checkout last fetched\n" +
			"it, with its pull request, its class (person-approved, drain, automatic or\n" +
			"person-push), the rule that admitted it (a drain rule record by id, or\n" +
			"facilitator.mode as the first parent committed it), the lane kind its branch's\n" +
			"Admitted-by: trailer names, the approval it rests on and where that was read\n" +
			"(forge, local tier, none seen, forge unread), and the issues and intents it\n" +
			"resolves and delivers. After the rows come the notes (how the range was\n" +
			"found, why the forge was not read for some row, why an answer was not kept,\n" +
			"why a first parent's mode could not be read) and the tip read.\n\n" +
			"The range runs from --since, a tag or a date YYYY-MM-DD (taken as the first\n" +
			"first-parent commit at or after it), to the default branch's tip; with no\n" +
			"--since it starts after the last release tag (v*) the tip reaches, and with\n" +
			"no such tag the verb refuses, naming the flag. --unattended keeps the rows no\n" +
			"evidence places with a person. --reverted lists the rows a later commit\n" +
			"reverts, grouped by the rule that admitted them, each group with a count, and\n" +
			"the untraceable reverts in a group of their own.\n\n" +
			"With one commit named, it traces that commit instead: the row that brought it\n" +
			"onto the default branch however far back, with its class and rule, and for a\n" +
			"reverting commit the rows it reverts; a commit the tip does not reach is said\n" +
			"to be not on the default branch. A trace takes none of the three flags.\n\n" +
			"It fetches nothing and pushes nothing: it asks the forge (gh api graphql) at\n" +
			"most once per pull request it has not read before, and keeps each answer it\n" +
			"read in .abcd/.work.local/landings/, the only thing it writes. Every value a\n" +
			"row prints is text someone typed (a subject, a trailer, a login) and is\n" +
			"masked for the terminal. Exit 0 with the listing or the trace; exit 1 when\n" +
			"history or the cache cannot be read; exit 2 outside a git checkout, with no\n" +
			"default branch, with no range, or with a commit named that names none.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return &exitError{Code: 2, Msg: "abcd landings: takes at most one commit to trace (nothing read)"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (since != "" || unattended || reverted) {
				return &exitError{Code: 2, Msg: "abcd landings: a trace of one commit takes none of --since, --unattended and --reverted (nothing read)"}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, landingsStore)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd landings: " + termsafe.Sanitize(err.Error()) + " (nothing read)"}
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			o := landings.Options{Forge: landingsForge}
			if len(args) == 1 {
				return runTrace(ctx, cmd, *asJSON, root, args[0], o)
			}
			r, err := landings.Resolve(ctx, root, since)
			if err != nil {
				return landingsFailed(err)
			}
			l, err := landings.List(ctx, root, r, o)
			if err != nil {
				return landingsFailed(err)
			}
			out := landingsOutput{Listing: l}
			if unattended {
				out.Rows = unattendedRows(l.Rows)
			}
			if reverted {
				out.Groups = landings.RevertedGroups(out.Listing)
				if out.Groups == nil {
					out.Groups = []landings.RevertGroup{}
				}
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderLandings(w, out, reverted) })
		},
	}
	f := cmd.Flags()
	f.StringVar(&since, "since", "", "start the range after this tag, or at the first first-parent commit on this date (YYYY-MM-DD); default: the last release tag (v*)")
	f.BoolVar(&unattended, "unattended", false, "keep only the rows no evidence places with a person")
	f.BoolVar(&reverted, "reverted", false, "list the reverted rows grouped by the rule that admitted them")
	return cmd
}

// runTrace is the positional trace of name.
func runTrace(ctx context.Context, cmd *cobra.Command, asJSON bool, root, name string, o landings.Options) error {
	t, err := landings.TraceCommit(ctx, root, name, o)
	if err != nil {
		return landingsFailed(err)
	}
	if t.Says != "" && t.Says != landings.SaysNotOnDefault {
		// The name names no one commit: the operand is refused, as one the
		// verb was not given.
		if asJSON {
			if err := render(cmd.OutOrStdout(), true, t, nil); err != nil {
				return err
			}
		}
		return &exitError{Code: 2, Msg: "abcd landings: " + landingsClip(name, landingsFieldBytes) + " " + landingsNote(t.Says)}
	}
	return render(cmd.OutOrStdout(), asJSON, t, func(w io.Writer) { renderTrace(w, name, t) })
}

// landingsFailed is the verb's refusal for err: exit 2 for a range it was
// not given or a branch it cannot list, exit 1 for anything else, the text
// masked either way.
func landingsFailed(err error) error {
	msg := "abcd landings: " + landingsNote(err.Error())
	if errors.Is(err, landings.ErrNoReleaseTag) || errors.Is(err, landings.ErrNoDefaultBranch) || errors.Is(err, landings.ErrBadSince) {
		return &exitError{Code: 2, Msg: msg + " (nothing read)"}
	}
	return &exitError{Code: 1, Msg: msg}
}

// unattendedRows is the rows with Unattended set, never nil.
func unattendedRows(rows []landings.Row) []landings.Row {
	out := []landings.Row{}
	for _, r := range rows {
		if r.Unattended {
			out = append(out, r)
		}
	}
	return out
}

// landingsClip is s masked for a terminal and cut at n bytes, on a rune's start.
func landingsClip(s string, n int) string {
	if len(s) > n {
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	return termsafe.Sanitize(s)
}

// landingsNote is a note masked and bounded as one line.
func landingsNote(s string) string { return termsafe.CleanProseLine(s, landingsNoteBytes) }

// landingsShort is a commit's first twelve characters, "-" for none.
func landingsShort(sha string) string {
	if sha == "" {
		return "-"
	}
	return landingsClip(sha[:min(12, len(sha))], 12)
}

// landingsOrDash is s clipped, "-" for "".
func landingsOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return landingsClip(s, landingsFieldBytes)
}

// landingRow is one row as one line: commit, pull request, class, rule (or a
// person's push's committer), lane kind, approval and where it was read, the ids it resolves and delivers,
// and the commit that reverts it where one does. Every value is clipped.
func landingRow(r landings.Row) string {
	pr := "-"
	if r.PR > 0 {
		pr = fmt.Sprintf("#%d", r.PR)
	}
	approval := landingsOrDash(r.Approval.Source)
	if r.Approval.By != "" {
		approval += ": " + landingsClip(r.Approval.By, landingsFieldBytes)
		if !r.Approval.At.IsZero() {
			approval += ", " + r.Approval.At.UTC().Format(time.RFC3339)
		}
	}
	rule := landingsOrDash(r.Rule.String())
	if r.Rule.String() == "" && r.Committer != "" {
		// A person's push names no rule: its committer, as git holds it.
		rule = "pushed by " + landingsClip(r.Committer, landingsFieldBytes)
	}
	parts := []string{landingsShort(r.Commit), pr, landingsOrDash(r.Class), rule, "lane " + landingsOrDash(r.Lane), "approval " + approval}
	var ids []string
	for _, id := range append(append([]string{}, r.Resolves...), r.Delivers...) {
		ids = append(ids, landingsClip(id, landingsFieldBytes))
	}
	if len(ids) > 0 {
		parts = append(parts, strings.Join(ids, " "))
	}
	if r.Reverted != nil {
		rv := "reverted by " + landingsShort(r.Reverted.By)
		if r.Reverted.WithdrawnBy != "" {
			rv += ", withdrawn by " + landingsShort(r.Reverted.WithdrawnBy)
		}
		parts = append(parts, rv)
	}
	return strings.Join(parts, "  ")
}

// renderLandings is the listing's text: one line per row (or, under
// --reverted, per row of each group under its rule and count), then the
// notes and the tip read.
func renderLandings(w io.Writer, out landingsOutput, reverted bool) {
	switch {
	case reverted && len(out.Groups) == 0:
		fmt.Fprintln(w, "no row in the range is reverted")
	case reverted:
		for _, g := range out.Groups {
			fmt.Fprintf(w, "%s  %d\n", landingsOrDash(g.Rule), g.Count)
			for _, r := range g.Rows {
				fmt.Fprintln(w, "  "+landingRow(r))
			}
		}
	case len(out.Rows) == 0:
		fmt.Fprintln(w, "no landings in the range")
	default:
		for _, r := range out.Rows {
			fmt.Fprintln(w, landingRow(r))
		}
	}
	renderLandingsNotes(w, [][2]string{
		{"range", out.SinceNote}, {"forge", out.ForgeUnread}, {"cache", out.CacheUnwritten}, {"mode", out.ModeUnread},
	})
	from := "the whole history"
	if out.From != "" {
		from = "after " + landingsShort(out.From)
	}
	fmt.Fprintf(w, "read %s at %s, since %s (%s)\n", landingsOrDash(out.Tip), landingsShort(out.To), landingsOrDash(out.Since), from)
}

// renderLandingsNotes prints each note that says something, one masked line each.
func renderLandingsNotes(w io.Writer, notes [][2]string) {
	for _, n := range notes {
		if n[1] != "" {
			fmt.Fprintf(w, "%s: %s\n", n[0], landingsNote(n[1]))
		}
	}
}

// renderTrace is the trace's text: the row that brought the commit, the rows
// it reverts, the notes and the tip read; or that it is not on the default
// branch.
func renderTrace(w io.Writer, name string, t landings.Trace) {
	commit := landingsClip(name, landingsFieldBytes)
	if t.Commit != "" {
		commit = landingsShort(t.Commit)
	}
	switch {
	case t.Says != "":
		fmt.Fprintf(w, "%s: %s\n", commit, landingsNote(t.Says))
	case t.Row != nil:
		fmt.Fprintf(w, "%s landed in:\n  %s\n", commit, landingRow(*t.Row))
		if len(t.Reverts) > 0 {
			fmt.Fprintln(w, "it reverts:")
			for _, r := range t.Reverts {
				fmt.Fprintln(w, "  "+landingRow(r))
			}
		}
	}
	renderLandingsNotes(w, [][2]string{
		{"range", t.RangeNote}, {"forge", t.ForgeUnread}, {"cache", t.CacheUnwritten}, {"mode", t.ModeUnread},
	})
	fmt.Fprintf(w, "read %s at %s\n", landingsOrDash(t.Tip), landingsShort(t.To))
}
