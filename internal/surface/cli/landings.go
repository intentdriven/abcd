package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf16"
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

// The render's own bounds, applied after the reader's (internal/core/landings)
// and whatever the reader hands it: every value a row prints is cut at
// landingsFieldBytes and every note at landingsNoteBytes; a row prints at most
// landingsMaxIDs of the ids it resolves and delivers, and of its Assisted-by:
// values, and of its revert lines at most landingsMaxUnmarkedLines that mark
// no row, printed first, and landingsMaxRevertLines that mark one, each list
// then saying how many more it holds. So one row's text is bounded however
// many ids and lines the row carries; --json keeps them all. Nothing that says
// why a row is unattended or why the forge's answer was not read is capped:
// those are one line each.
const (
	landingsFieldBytes       = 200
	landingsNoteBytes        = 600
	landingsMaxIDs           = 8
	landingsMaxRevertLines   = 8
	landingsMaxUnmarkedLines = 16
)

// landingsOutput is the listing's --json: the reader's Listing, its rows kept
// to the unattended ones under --unattended, and under --reverted the groups
// the text prints, an empty list when no row is reverted.
type landingsOutput struct {
	landings.Listing
	Groups []landings.RevertGroup `json:"groups,omitzero"`
}

// newLandingsCommand builds the `landings` verb, the front door onto
// internal/core/landings (itd-2610301850454860, spc-2610301950480609
// approach 8): how each change landed on the default branch, with the rule
// that admitted it. It reads history as last fetched, asks the forge as the
// reader does (forge.go), and writes only the reader's forge cache in the
// local tier.
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
			"(forge, local tier, none seen, forge unread) and whether one was seen, up to\n" +
			"eight of the issues and intents it resolves and delivers, its mode, whether it\n" +
			"is unattended, and the commit that reverts it; then when it landed, the head\n" +
			"it merged, where its rule was read and up to eight Assisted-by: values. Under\n" +
			"a row come, never capped, what the forge said against the pull request its\n" +
			"subject names and why the forge's answer was not read; then its revert lines,\n" +
			"each with the row it marks or why it marks none: up to sixteen that mark no\n" +
			"row first (the untraceable ones before the rest), then up to eight that mark\n" +
			"one, then how many more of each. Every capped list says how many more it\n" +
			"holds; --json keeps them all. After the rows come the notes (how the range was\n" +
			"found, why the forge was not read for some row, why an answer was not kept,\n" +
			"why a first parent's mode could not be read) and the tip read.\n\n" +
			"The range runs from --since, a tag or a date YYYY-MM-DD (taken as the first\n" +
			"first-parent commit at or after it), to the default branch's tip; with no\n" +
			"--since it starts after the last release tag (v*) the tip reaches, and with\n" +
			"no such tag, or one git describes by a name no tag holds (an annotated tag\n" +
			"whose ref was renamed), the verb refuses, naming the flag. A tag is looked up\n" +
			"as that exact ref. --unattended keeps the rows no\n" +
			"evidence places with a person. --reverted lists the rows a later commit\n" +
			"reverts, grouped by the rule that admitted them, each group with a count, and\n" +
			"the rows carrying an untraceable revert in a group of their own. The two\n" +
			"filters are not taken together.\n\n" +
			"With one commit named, it traces that commit instead: the row that brought it\n" +
			"onto the default branch however far back, with its class and rule, and for a\n" +
			"reverting commit the rows it reverts; a commit the tip does not reach is said\n" +
			"to be not on the default branch. A trace takes none of the three flags.\n\n" +
			"It fetches nothing and pushes nothing. It asks the forge (gh api graphql)\n" +
			"about each pull request the range names whose answer its cache does not keep,\n" +
			"up to twenty to a document, and keeps in .abcd/.work.local/landings/, the only\n" +
			"thing it writes, the answers that cannot change: one it accepts for its row,\n" +
			"one saying the pull request merged as another commit, and the forge's word\n" +
			"that a number names no pull request. An answer it could not read, one whose\n" +
			"reviews run past the page it reads, one read without the commits its row\n" +
			"asked, and one saying the pull request has not merged or naming no merge\n" +
			"commit are not kept, so every listing asks for them again. A document whose\n" +
			"answer is too large, or carries an error naming no one pull request, is asked\n" +
			"again in halves, within a bound on the calls one listing spends so. A squashed\n" +
			"revert's pull request body is asked apart, on every listing that reads it, and\n" +
			"never kept. The render masks for the terminal every value it prints, and the\n" +
			"committer name, the forge's logins and failure text, the lane, the ids, a\n" +
			"revert line's names and the notes carry text someone typed.\n\n" +
			"Exit 0 with the listing or the trace, a listing whose forge answers were not\n" +
			"read included (those rows are unattended and the forge note says how many);\n" +
			"exit 1 when history or the tags cannot be read; exit 2 outside a git\n" +
			"checkout, with no default branch, with no range, with both filters, or with a\n" +
			"commit named that names none.",
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
			if unattended && reverted {
				return &exitError{Code: 2, Msg: "abcd landings: --unattended and --reverted are not taken together; name one (nothing read)"}
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
			out := landingsView(l, unattended, reverted)
			return landingsRender(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderLandings(w, out, unattended, reverted) })
		},
	}
	f := cmd.Flags()
	f.StringVar(&since, "since", "", "start the range after this tag, or at the first first-parent commit on this date (YYYY-MM-DD); default: the last release tag (v*)")
	f.BoolVar(&unattended, "unattended", false, "keep only the rows no evidence places with a person")
	f.BoolVar(&reverted, "reverted", false, "list the reverted rows grouped by the rule that admitted them")
	return cmd
}

// landingsView is what the listing prints of l under the filters: its rows
// kept to the unattended ones under unattended, and under reverted the
// groups, never nil, so --json carries an empty list when nothing is
// reverted.
func landingsView(l landings.Listing, unattended, reverted bool) landingsOutput {
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
	return out
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
			if err := landingsRender(cmd.OutOrStdout(), true, t, nil); err != nil {
				return err
			}
		}
		return &exitError{Code: 2, Msg: "abcd landings: " + landingsClip(name, landingsFieldBytes) + " " + landingsNote(t.Says)}
	}
	return landingsRender(cmd.OutOrStdout(), asJSON, t, func(w io.Writer) { renderTrace(w, name, t) })
}

// landingsFailed is the verb's refusal for err: exit 2 for a range it was
// not given or a branch it cannot list, exit 1 for anything else, the text
// masked either way.
func landingsFailed(err error) error {
	msg := "abcd landings: " + landingsNote(err.Error())
	if errors.Is(err, landings.ErrNoReleaseTag) || errors.Is(err, landings.ErrReleaseTagUnnamed) || errors.Is(err, landings.ErrNoDefaultBranch) || errors.Is(err, landings.ErrBadSince) {
		return &exitError{Code: 2, Msg: msg + " (nothing read)"}
	}
	return &exitError{Code: 1, Msg: msg}
}

// landingsRender is render for the verb: the text as text, and --json with
// every rune the terminal mask masks above ASCII (DEL, the C1 controls, the
// bidirectional and zero-width runes) written as JSON's own \u escape, which
// encoding/json leaves raw. It is lossless: a JSON reader reads the same
// strings back.
func landingsRender(w io.Writer, asJSON bool, v any, text func(io.Writer)) error {
	if !asJSON {
		text(w)
		return nil
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(landingsJSONEscape(b), '\n'))
	return err
}

// landingsJSONEscape is b, valid UTF-8 as encoding/json writes it, with each
// rune at or above DEL that termsafe.Sanitize masks written as \uXXXX (a
// surrogate pair past the basic plane). Such a rune can stand only inside a
// string, where the escape reads as the rune itself.
func landingsJSONEscape(b []byte) []byte {
	var out []byte
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if r < 0x7f || termsafe.Sanitize(string(r)) == string(r) {
			out = append(out, b[i:i+n]...)
			i += n
			continue
		}
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			out = fmt.Appendf(out, `\u%04x\u%04x`, hi, lo)
		} else {
			out = fmt.Appendf(out, `\u%04x`, r)
		}
		i += n
	}
	return out
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

// landingsClip is s masked for a terminal and then cut to at most n bytes on
// a rune's start. Masking first is what makes the bound exact: an invalid
// byte becomes one '?', and every masked rune one '?', so nothing grows after
// the cut and no cut lands inside a rune it prints.
func landingsClip(s string, n int) string {
	s = termsafe.Sanitize(strings.ToValidUTF8(s, "?"))
	if len(s) > n {
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	return s
}

// landingsNote is a note masked and bounded as one line.
func landingsNote(s string) string { return termsafe.CleanProseLine(s, landingsNoteBytes) }

// landingsShort is a commit's first twelve characters, masked, "-" for none.
func landingsShort(sha string) string {
	if sha == "" {
		return "-"
	}
	return landingsClip(sha, 12)
}

// landingsOrDash is s clipped, "-" for "".
func landingsOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return landingsClip(s, landingsFieldBytes)
}

// landingsList is the first max of vals clipped and joined by sep, then
// "and N more" for the rest.
func landingsList(vals []string, max int, sep string) string {
	var parts []string
	for _, v := range vals[:min(max, len(vals))] {
		parts = append(parts, landingsClip(v, landingsFieldBytes))
	}
	if len(vals) > max {
		parts = append(parts, fmt.Sprintf("and %d more", len(vals)-max))
	}
	return strings.Join(parts, sep)
}

// landingRow is one row as its lines. The first is the row: commit, pull
// request, class, rule (or a person's push's committer name, as git holds
// it), lane kind, approval, where it was read and whether one was seen, the
// ids it resolves and delivers, its mode, the unattended mark, and the commit
// that reverts it. The second says when it landed and what it was read from:
// its committer time, the head it merged, where its rule was read, and its
// Assisted-by: values. Then, each on its own line, what the forge said against
// the pull request its subject names, why the forge's answer was not read,
// and the revert lines its commits carry with the row each marks or why it
// marks none (landingsRevertLines). Every value is masked and clipped, and the
// lists capped.
func landingRow(r landings.Row) []string {
	pr := "-"
	if r.PR > 0 {
		pr = fmt.Sprintf("#%d", r.PR)
	}
	approval := landingsOrDash(r.Approval.Source)
	if r.Approval.By != "" {
		approval += ": " + landingsClip(r.Approval.By, landingsFieldBytes)
	}
	if !r.Approval.At.IsZero() {
		approval += ", " + r.Approval.At.UTC().Format(time.RFC3339)
	}
	if r.Approval.Seen {
		approval += ", seen"
	}
	rule := landingsOrDash(r.Rule.String())
	committerIsRule := r.Rule.String() == "" && r.Committer != ""
	if committerIsRule {
		// A person's push names no rule: the committer name git holds, which
		// whoever made the commit set; nothing read says who pushed it.
		rule = "committer " + landingsClip(r.Committer, landingsFieldBytes)
	}
	head := []string{landingsShort(r.Commit), pr, landingsOrDash(r.Class), rule, "lane " + landingsOrDash(r.Lane), "approval " + approval}
	if ids := append(append([]string{}, r.Resolves...), r.Delivers...); len(ids) > 0 {
		head = append(head, landingsList(ids, landingsMaxIDs, " "))
	}
	head = append(head, "mode "+landingsOrDash(r.Mode))
	if r.Unattended {
		head = append(head, "unattended")
	}
	if m := r.Reverted; m != nil {
		rv := fmt.Sprintf("reverted by %s (%s, in row %s)", landingsShort(m.By), landingsOrDash(m.Kind), landingsShort(m.ByRow))
		if m.WithdrawnBy != "" {
			rv += ", withdrawn by " + landingsShort(m.WithdrawnBy)
		}
		head = append(head, rv)
	}
	read := []string{"landed " + r.When.UTC().Format(time.RFC3339)}
	if r.MergedHead != "" {
		read = append(read, "merged head "+landingsShort(r.MergedHead))
	}
	if r.Rule.Path != "" || r.Rule.At != "" {
		read = append(read, "rule read from "+landingsOrDash(r.Rule.Path)+" at "+landingsShort(r.Rule.At))
	}
	if r.Committer != "" && !committerIsRule {
		read = append(read, "committer "+landingsClip(r.Committer, landingsFieldBytes))
	}
	if len(r.AssistedBy) > 0 {
		read = append(read, "assisted by "+landingsList(r.AssistedBy, landingsMaxIDs, ", "))
	}
	lines := []string{strings.Join(head, "  "), "  " + strings.Join(read, ", ")}
	if r.ForgeSays != "" {
		lines = append(lines, "  forge says: "+landingsNote(r.ForgeSays))
	}
	if r.ForgeUnreadable != "" {
		lines = append(lines, "  forge unread: "+landingsNote(r.ForgeUnreadable))
	}
	for _, l := range landingsRevertLines(r.Reverts) {
		lines = append(lines, "  "+l)
	}
	return lines
}

// landingsRevertLines is a row's revert lines as the text prints them. The
// lines that mark no row come first, the untraceable ones (why the row is in
// the untraceable group) before the rest, up to landingsMaxUnmarkedLines;
// then the lines that mark a row, up to landingsMaxRevertLines; then, when
// either cap left lines out, how many, and how many of those mark a row and
// how many do not. A row can carry many lines that mark no row (one for each
// of its commits whose subject reads as git's revert, and up to 64 for each
// commit's or body's revert lines), so they have a cap of their own, and no
// line that marks one is printed while a line that marks none is held back.
func landingsRevertLines(xs []landings.RevertLine) []string {
	var untraceable, unmarked, marked []landings.RevertLine
	for _, x := range xs {
		switch {
		case x.Row != "":
			marked = append(marked, x)
		case x.Kind == landings.RevertUntraceable:
			untraceable = append(untraceable, x)
		default:
			unmarked = append(unmarked, x)
		}
	}
	none := append(untraceable, unmarked...)
	shownNone, shownMarked := min(len(none), landingsMaxUnmarkedLines), min(len(marked), landingsMaxRevertLines)
	var out []string
	for _, x := range append(none[:shownNone:shownNone], marked[:shownMarked]...) {
		out = append(out, landingsRevertLine(x))
	}
	if hiddenNone, hiddenMarked := len(none)-shownNone, len(marked)-shownMarked; hiddenNone+hiddenMarked > 0 {
		out = append(out, fmt.Sprintf("and %d more revert lines: %d marking a row, %d with no row marked", hiddenNone+hiddenMarked, hiddenMarked, hiddenNone))
	}
	return out
}

// landingsRevertLine is one revert line: its kind, the commit carrying it,
// what it names and the commit that resolved to, then the row it marks or
// why it marks none.
func landingsRevertLine(x landings.RevertLine) string {
	s := "reverts (" + landingsOrDash(x.Kind) + " line in " + landingsShort(x.Commit) + ")"
	if x.Names != "" {
		s += " names " + landingsClip(x.Names, landingsFieldBytes)
	}
	if x.Target != "" {
		s += ", resolved to " + landingsShort(x.Target)
	}
	switch {
	case x.Row != "":
		s += ": marks row " + landingsShort(x.Row)
		if x.Unmarked != "" {
			s += "; " + landingsNote(x.Unmarked)
		}
	case x.Unmarked != "":
		s += ": marks no row: " + landingsNote(x.Unmarked)
	default:
		s += ": marks no row"
	}
	return s
}

// printLandingRow prints r's lines, each after indent.
func printLandingRow(w io.Writer, indent string, r landings.Row) {
	for _, l := range landingRow(r) {
		fmt.Fprintln(w, indent+l)
	}
}

// renderLandings is the listing's text: each row's lines (or, under
// --reverted, each group's rule and count and then its rows), then the notes
// and the tip read.
func renderLandings(w io.Writer, out landingsOutput, unattended, reverted bool) {
	switch {
	case reverted && len(out.Groups) == 0:
		fmt.Fprintln(w, "no row in the range is reverted")
	case reverted:
		for _, g := range out.Groups {
			fmt.Fprintf(w, "%s  %d\n", landingsOrDash(g.Rule), g.Count)
			for _, r := range g.Rows {
				printLandingRow(w, "  ", r)
			}
		}
	case unattended && len(out.Rows) == 0:
		fmt.Fprintln(w, "no row in the range is unattended")
	case len(out.Rows) == 0:
		fmt.Fprintln(w, "no landings in the range")
	default:
		for _, r := range out.Rows {
			printLandingRow(w, "", r)
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

// renderTrace is the trace's text: the row that brought the commit, with its
// revert lines, the rows they mark, the notes (what the range could not
// place among them) and the tip read with where the listing began; or what
// stops it having a row.
func renderTrace(w io.Writer, name string, t landings.Trace) {
	commit := landingsClip(name, landingsFieldBytes)
	if t.Commit != "" {
		commit = landingsShort(t.Commit)
	}
	switch {
	case t.Says != "":
		fmt.Fprintf(w, "%s: %s\n", commit, landingsNote(t.Says))
	case t.Row != nil:
		fmt.Fprintf(w, "%s landed in:\n", commit)
		printLandingRow(w, "  ", *t.Row)
		if len(t.Reverts) > 0 {
			fmt.Fprintln(w, "it reverts:")
			for _, r := range t.Reverts {
				printLandingRow(w, "  ", r)
			}
		}
	}
	renderLandingsNotes(w, [][2]string{
		{"range", t.RangeNote}, {"forge", t.ForgeUnread}, {"cache", t.CacheUnwritten}, {"mode", t.ModeUnread},
	})
	tip := fmt.Sprintf("read %s at %s", landingsOrDash(t.Tip), landingsShort(t.To))
	if t.From != "" {
		tip += ", listed after " + landingsShort(t.From)
	}
	fmt.Fprintln(w, tip)
}
