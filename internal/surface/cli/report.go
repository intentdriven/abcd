package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// reportStore is how a checkout-root refusal names what needed the checkout.
const reportStore = "a report's sender key"

// reportInteractive reports whether the bare verb may open an editor: both ends
// of the session are a terminal. Tests replace it.
var reportInteractive = func() bool { return term.IsTerminal(os.Stdin) && term.IsTerminal(os.Stdout) }

// newReportCommand builds `abcd report` — the front door onto
// internal/core/report's filing half (itd-2609221656361680). It runs in the
// repository abcd manages that has something to tell abcd, and writes only to
// the user account's inbox.
func newReportCommand(asJSON *bool) *cobra.Command {
	var template bool
	cmd := &cobra.Command{
		Use:   "report [<file>|-]",
		Short: "File a defect report or an enhancement proposal about abcd into the inbox in your account",
		Long: "File a written account about abcd itself, from a repository abcd manages, into\n" +
			"the inbox in the user account's machine store (`~/.abcd/inbox/`). Nothing is\n" +
			"written into this repository or into abcd's, and nothing becomes a record until\n" +
			"a person or a session runs `abcd inbox promote`.\n\n" +
			"`abcd report --template` prints the skeleton: a block of fields between `---`\n" +
			"lines (template version, kind, severity, category, title, the abcd version and\n" +
			"surface in play, an optional remedy and evidence pointers) and the prose below\n" +
			"it. `abcd report <file>` validates a filled report and files it; `-` reads it\n" +
			"from stdin. Bare `abcd report` opens the skeleton in $VISUAL or $EDITOR when\n" +
			"the session is a terminal, and files what is saved.\n\n" +
			"The report is held to the template: a missing or malformed field is refused\n" +
			"naming the field, a report over 32 KiB or carrying a control byte is refused,\n" +
			"and a field naming a filesystem path is refused, because a report points at\n" +
			"records, commits and URLs, never at a location on a machine. abcd names the\n" +
			"file from the time and this repository's root-commit key; the verb prints the\n" +
			"report's id and where it landed.\n\n" +
			"Exit 2 on a refusal, with nothing filed.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skeleton := report.Template(core.NewVersion().Version)
			if template {
				if len(args) > 0 {
					return &exitError{Code: 2, Msg: "abcd report: --template takes no file (nothing filed)"}
				}
				return render(cmd.OutOrStdout(), *asJSON, struct {
					Template string `json:"template"`
				}{string(skeleton)}, func(w io.Writer) {
					_, _ = w.Write(skeleton)
				})
			}
			var data []byte
			var err error
			kept := ""
			switch {
			case len(args) == 0:
				data, kept, err = reportFromEditor(skeleton)
			case args[0] == "-":
				data, err = io.ReadAll(io.LimitReader(cmd.InOrStdin(), report.MaxBytes+1))
			default:
				data, err = fsutil.ReadGuarded(args[0], report.MaxBytes)
				if errors.Is(err, fsutil.ErrTooBig) {
					err = fmt.Errorf("%w: the file is over the %d-byte bound a report is held to", report.ErrRefused, report.MaxBytes)
				} else if err != nil {
					err = fmt.Errorf("%w: cannot read the report file: %v", report.ErrRefused, fsutil.RedactHome(err.Error()))
				}
			}
			// refuse says where the editor's text is kept, on every refusal
			// after the editor ran, so a rejected report is never lost.
			refuse := func(err error) error {
				if kept != "" && errors.Is(err, report.ErrRefused) {
					where := fsutil.RedactHome(kept)
					err = fmt.Errorf("%w; what you wrote is kept at %s — fix it and run `abcd report %s`", err, where, where)
				}
				return reportRefusal("report", err, "nothing filed")
			}
			if err != nil {
				return reportRefusal("report", err, "nothing filed")
			}
			r, err := report.Parse(data)
			if err != nil {
				return refuse(err)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, reportStore)
			if err != nil {
				return refuse(fmt.Errorf("%w: %v", report.ErrRefused, err))
			}
			sender, err := report.SenderOf(root)
			if err != nil {
				return refuse(err)
			}
			filed, err := report.File(r, sender)
			if err != nil {
				return refuse(err)
			}
			if kept != "" {
				_ = os.Remove(kept)
			}
			return render(cmd.OutOrStdout(), *asJSON, filed, func(w io.Writer) {
				fmt.Fprintf(w, "filed %s — %s\n", filed.ID, termsafe.Sanitize(filed.Path))
				fmt.Fprintf(w, "  abcd names it at its next start; nothing becomes a record until someone runs `abcd inbox promote %s`\n", filed.ID)
			})
		},
	}
	cmd.Flags().BoolVar(&template, "template", false, "print the report skeleton to fill (writes nothing)")
	return cmd
}

// reportFromEditor writes the skeleton to a private temporary file outside both
// repositories, opens it in the reporter's editor, and returns what they saved.
// kept names the file while it still holds their text, so a refusal can say
// where it is; the caller removes it once the report is filed.
func reportFromEditor(skeleton []byte) (data []byte, kept string, err error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" || !reportInteractive() {
		return nil, "", fmt.Errorf("%w: with no file, the report opens in $VISUAL or $EDITOR on a terminal, and neither is available here; "+
			"run `abcd report --template`, fill it, and file it with `abcd report <file>` or `abcd report -`", report.ErrRefused)
	}
	f, err := os.CreateTemp("", "abcd-report-*.md")
	if err != nil {
		return nil, "", err
	}
	name := f.Name()
	_, werr := f.Write(skeleton)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(name)
		return nil, "", werr
	}
	// The editor is the reporter's own setting and may carry arguments, so it is
	// run by the shell; the file travels as a positional parameter, never
	// spliced into the command text.
	ed := exec.Command("sh", "-c", editor+` "$1"`, "abcd-report", name)
	ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := ed.Run(); err != nil {
		return nil, name, fmt.Errorf("%w: the editor exited with %v; what you wrote is kept at %s", report.ErrRefused, err, fsutil.RedactHome(name))
	}
	data, err = fsutil.ReadGuarded(name, report.MaxBytes)
	if err != nil {
		return nil, name, fmt.Errorf("%w: cannot read what the editor saved (%v); it is kept at %s", report.ErrRefused, err, fsutil.RedactHome(name))
	}
	if bytes.Equal(data, skeleton) {
		_ = os.Remove(name)
		return nil, "", fmt.Errorf("%w: the skeleton came back unchanged", report.ErrRefused)
	}
	return data, name, nil
}

// reportRefusal maps a refusal to exit 2, sanitised for the terminal: the text
// may quote a field of a report that came from another repository.
func reportRefusal(verb string, err error, what string) error {
	if errors.Is(err, report.ErrRefused) {
		msg := strings.TrimPrefix(err.Error(), report.ErrRefused.Error()+": ")
		return &exitError{Code: 2, Msg: "abcd " + verb + ": " + termsafe.Sanitize(msg) + " (" + what + ")"}
	}
	return fmt.Errorf("abcd %s: %s", verb, termsafe.Sanitize(fsutil.RedactHome(err.Error())))
}

// managedRepos renders a count of sending repositories.
func managedRepos(n int) string {
	if n == 1 {
		return "1 managed repository"
	}
	return fmt.Sprintf("%d managed repositories", n)
}

// inboxTallyText is the greeting's words and the board row's: counts only. It
// carries no sender name and no report text, because the session-start line is
// injected into a session's context and a report is another repository's
// untrusted words.
func inboxTallyText(t report.Tally) string {
	return fmt.Sprintf("%d report(s) from %s", t.Reports, managedRepos(t.Senders))
}

// inboxGreeting is the session-start line, or "" when nothing waits.
func inboxGreeting() string {
	t, err := report.Count()
	if err != nil || t.Reports == 0 {
		return ""
	}
	return "abcd: " + inboxTallyText(t) + " wait in the inbox; `abcd inbox` lists them."
}

// boardInbox is the board's inbox row, or nil when nothing waits.
func boardInbox() *report.Tally {
	t, err := report.Count()
	if err != nil || t.Reports == 0 {
		return nil
	}
	return &t
}

// inboxListOutput is `abcd inbox --json`.
type inboxListOutput struct {
	Tally   report.Tally   `json:"tally"`
	Reports []report.Entry `json:"reports"`
}

// newInboxCommand builds `abcd inbox` — the reading and filing half of
// internal/core/report. The list and show forms are read-only; promote is the
// one act that files anything.
func newInboxCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Read the reports managed repositories filed back to abcd, and promote one to a capture",
		Long: "Read the reports repositories abcd manages filed with `abcd report`, from the\n" +
			"inbox in the user account's machine store (`~/.abcd/inbox/`).\n\n" +
			"Bare `abcd inbox` lists the waiting reports newest first, naming each sender\n" +
			"repository plainly; `abcd inbox show <id>` renders one whole. Both are\n" +
			"read-only and file nothing. A report written to a template version this abcd\n" +
			"does not know is listed as unreadable, naming the version, and is never\n" +
			"dropped. Everything a report says is another repository's words and is\n" +
			"sanitised before it reaches the terminal.\n\n" +
			"`abcd inbox promote <id>` is the one act that files anything: it files the\n" +
			"report as a capture in the ledger of the repository you stand in, through the\n" +
			"capture verb's own path and redactor, with source `managed-repo`. The capture\n" +
			"carries the sender's root-commit key and the words \"a managed repository\",\n" +
			"never the sender's name, and the report's id as its evidence. The report is\n" +
			"kept, marked promoted.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			list, err := report.List()
			if err != nil {
				return reportRefusal("inbox", err, "nothing read")
			}
			senders := map[string]bool{}
			for _, e := range list {
				senders[e.SenderKey] = true
			}
			out := inboxListOutput{Tally: report.Tally{Reports: len(list), Senders: len(senders)}, Reports: list}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if len(list) == 0 {
					fmt.Fprintln(w, "abcd inbox — nothing waits")
					return
				}
				where := "repositories"
				if out.Tally.Senders == 1 {
					where = "repository"
				}
				fmt.Fprintf(w, "abcd inbox — %d waiting from %d %s\n", out.Tally.Reports, out.Tally.Senders, where)
				for _, e := range list {
					if e.State == report.StateUnreadable {
						fmt.Fprintf(w, "  %s  %s  key %s  UNREADABLE: %s\n", e.ID, e.ReceivedAt, e.SenderKey[:12], termsafe.Sanitize(e.Unreadable))
						continue
					}
					fmt.Fprintf(w, "  %s  %s  %s  %s/%s  %s\n", e.ID, e.ReceivedAt, termsafe.Sanitize(e.SenderName),
						e.Kind, e.Severity, termsafe.Sanitize(e.Title))
				}
			})
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show <id>",
		Short: "Render one report whole (read-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := report.Show(args[0])
			if err != nil {
				return reportRefusal("inbox show", err, "nothing read")
			}
			return render(cmd.OutOrStdout(), *asJSON, e, func(w io.Writer) { renderReport(w, e) })
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "promote <id>",
		Short: "File one report as a capture in this repository's ledger, fingerprinted, never named",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			ledger, err := capture.LedgerRoot(cwd)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd inbox promote: " + termsafe.Sanitize(err.Error()) + " (nothing written)"}
			}
			p, err := report.Promote(ledger, args[0])
			if err != nil {
				return reportRefusal("inbox promote", err, "nothing written")
			}
			return render(cmd.OutOrStdout(), *asJSON, p, func(w io.Writer) {
				fmt.Fprintf(w, "promoted %s to %s — %s\n", p.Report, p.Capture, termsafe.Sanitize(p.Path))
				if p.Resumed {
					fmt.Fprintln(w, "  finished an earlier promotion that filed this capture; nothing new was filed")
				}
				fmt.Fprintln(w, "  the report is kept in the inbox, marked promoted")
				if p.Redacted > 0 {
					fmt.Fprintf(w, "  redacted %d span(s) before writing (home paths and identifiers are never committed)\n", p.Redacted)
				}
				if p.Degraded != "" {
					fmt.Fprintf(w, "  WARNING: %s\n", termsafe.Sanitize(p.Degraded))
				}
			})
		},
	})
	return cmd
}

// renderReport is the text form of `abcd inbox show`. Every value came from
// another repository, so every value is sanitised.
func renderReport(w io.Writer, e report.Entry) {
	who := termsafe.Sanitize(e.SenderName)
	if who == "" {
		who = "an unnamed repository"
	}
	fmt.Fprintf(w, "%s (%s) from %s, root commit %s\n", e.ID, e.State, who, e.SenderKey)
	fmt.Fprintf(w, "  received:  %s\n", termsafe.Sanitize(e.ReceivedAt))
	if e.PromotedTo != "" {
		fmt.Fprintf(w, "  promoted:  %s\n", termsafe.Sanitize(e.PromotedTo))
	}
	if e.Unreadable != "" {
		fmt.Fprintf(w, "  UNREADABLE: %s\n", termsafe.Sanitize(e.Unreadable))
		return
	}
	r := e.Report
	if r == nil {
		return
	}
	fmt.Fprintf(w, "  kind:      %s / %s / %s\n", r.Kind, r.Severity, r.Category)
	fmt.Fprintf(w, "  title:     %s\n", termsafe.Sanitize(r.Title))
	fmt.Fprintf(w, "  abcd:      %s\n", termsafe.Sanitize(r.AbcdVersion))
	fmt.Fprintf(w, "  surface:   %s\n", termsafe.Sanitize(r.Surface))
	if r.Remedy != "" {
		fmt.Fprintf(w, "  remedy:    %s\n", termsafe.Sanitize(r.Remedy))
	}
	for _, ev := range r.Evidence {
		fmt.Fprintf(w, "  evidence:  %s\n", termsafe.Sanitize(ev))
	}
	fmt.Fprintf(w, "\n%s\n", termsafe.SanitizeBlock(r.Prose))
}
