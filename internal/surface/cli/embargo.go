package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// embargoStore is the noun the checkout resolution names in its refusal.
const embargoStore = "the embargo store"

// newEmbargoCommand builds the `embargo` verb — the front door onto
// internal/core/embargo (itd-2610112239022766, spc-2610120557432880): a
// private security advisory's record kept in the embargo store, never in the
// public ledger, from intake to closing or publication
// (adr-2610120558328615).
func newEmbargoCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "embargo",
		Long: "Keep this repository's private security advisories private until their fix\n" +
			"ships. Each advisory GitHub holds in triage or draft becomes a private record\n" +
			"in the embargo store, a detached worktree in abcd's machine-scoped worktree\n" +
			"store that is never committed; the link from advisory to record lives in the\n" +
			"store's state, never in the record. Every sub-verb verifies the store before\n" +
			"it reads or writes it, and refuses, writing nothing, when it is on a branch,\n" +
			"tracks a private record, or sits inside this checkout.\n\n" +
			"Every remote call is the person's own `gh` against github.com; abcd holds no\n" +
			"token. close and publish each act on GitHub only with --approve naming the\n" +
			"digest of exactly the text or plan they showed, and a host passes it only\n" +
			"after the person, in that session, has seen it and said yes.\n\n" +
			"Exit 2 on a refusal (nothing written) and on a check finding.",
		Args: cobra.NoArgs,
		RunE: helpRunE,
	}
	cmd.AddCommand(newEmbargoIntakeCommand(asJSON), newEmbargoTriageCommand(asJSON), newEmbargoCloseCommand(asJSON),
		newEmbargoCheckCommand(asJSON), newEmbargoPublishCommand(asJSON))
	return cmd
}

// embargoPlace is the checkout root and abcd's home a sub-verb acts on.
func embargoPlace(verb string) (root, home string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	root, err = gitutil.CheckoutRoot(cwd, embargoStore)
	if err != nil {
		return "", "", &exitError{Code: 2, Msg: verb + ": " + err.Error() + " (nothing written)"}
	}
	home, err = os.UserHomeDir()
	if err != nil || home == "" {
		return "", "", &exitError{Code: 2, Msg: verb + ": the home directory cannot be named, so there is no abcd home to keep the embargo store in (nothing written)"}
	}
	return root, home, nil
}

// embargoFailure maps a core error to its exit: a refusal is 2; a close or a
// publication GitHub did not confirm is 1, naming where it stopped; anything
// else is a fault at 1. Every message is home-redacted.
func embargoFailure(verb string, err error) error {
	msg := fsutil.RedactHome(strings.TrimPrefix(err.Error(), "embargo: "))
	var r *embargo.RefusalError
	if errors.As(err, &r) {
		return &exitError{Code: 2, Msg: verb + ": " + msg}
	}
	var cf *embargo.CloseFailedError
	var pf *embargo.PublishFailedError
	if errors.As(err, &cf) || errors.As(err, &pf) {
		return &exitError{Code: 1, Msg: verb + ": " + msg}
	}
	return fmt.Errorf("%s: %s", verb, msg)
}

func newEmbargoIntakeCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "intake",
		Long: "List the repository's security advisories in the triage and draft states\n" +
			"(GitHub's REST API, through your own `gh`) and mint one private record per\n" +
			"advisory not seen before, in the embargo store's own ledger. The store is\n" +
			"made on first use, detached at the default branch's tip. Severity maps\n" +
			"critical to critical, high to major, medium and low to minor; an advisory\n" +
			"with no severity is listed for you to set on GitHub and not recorded. A\n" +
			"re-run records nothing twice. Nothing is written to this checkout's ledger\n" +
			"or committed anywhere.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const verb = "abcd embargo intake"
			root, home, err := embargoPlace(verb)
			if err != nil {
				return err
			}
			res, err := embargo.Intake(embargo.IntakeRequest{Home: home, RepoRoot: root})
			if err != nil {
				return embargoFailure(verb, err)
			}
			res.Store = fsutil.DisplayPath(res.Store)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderEmbargoIntake(w, res) })
		},
	}
}

func renderEmbargoIntake(w io.Writer, res embargo.IntakeResult) {
	fmt.Fprintf(w, "abcd embargo intake — %s: %s, %d already held, %d needing a severity\n",
		termsafe.Sanitize(res.Repo), countOf(len(res.Minted), "new private record"), len(res.Held), len(res.NeedsSeverity))
	for _, m := range res.Minted {
		fmt.Fprintf(w, "  minted %s for %s (%s)\n", m.IssueID, m.GHSAID, m.Severity)
	}
	for _, id := range res.NeedsSeverity {
		fmt.Fprintf(w, "  not recorded: %s has no severity this abcd maps; set it on GitHub, then run intake again\n", id)
	}
	created := ""
	if res.StoreCreated {
		created = " (made now)"
	}
	fmt.Fprintf(w, "store: %s%s\n", termsafe.Sanitize(res.Store), created)
	if len(res.Minted) > 0 {
		fmt.Fprintln(w, "next: abcd embargo triage <iss-N|GHSA-id> <verdict> --by \"<who decided>\"")
	}
}

func newEmbargoTriageCommand(asJSON *bool) *cobra.Command {
	var by, reason, existsWithout, cannotHelp string
	cmd := &cobra.Command{
		Use: "triage <iss-N|GHSA-id> <verdict>",
		Long: "Record your verdict on one private advisory, with who decided it (--by) and\n" +
			"when (the clock, never typed). The verdicts:\n\n" +
			"  vulnerability        the record stays open in the store, to be fixed\n" +
			"  not-abcds            needs --exists-without and --cannot-help, the two\n" +
			"                       halves of abcd-fixes-what-it-adds\n" +
			"  not-a-vulnerability  needs --reason\n" +
			"  public-track         needs --reason; prints the `abcd capture` command to\n" +
			"                       run in the public checkout\n\n" +
			"The two closing verdicts move the record to the store's wontfix/ and draft\n" +
			"the closing explanation, printed here, for `abcd embargo close`. A second\n" +
			"triage of the same advisory keeps the earlier verdict in its history. An\n" +
			"unknown verdict, or one missing its reason, is refused with nothing written.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd embargo triage"
			root, home, err := embargoPlace(verb)
			if err != nil {
				return err
			}
			res, err := embargo.Triage(embargo.TriageRequest{Home: home, RepoRoot: root, Target: args[0], Verdict: args[1],
				By: by, Reason: reason, ExistsWithout: existsWithout, CannotHelp: cannotHelp})
			if err != nil {
				return embargoFailure(verb, err)
			}
			res.DraftPath = fsutil.DisplayPath(res.DraftPath)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderEmbargoTriage(w, res) })
		},
	}
	cmd.Flags().StringVar(&by, "by", "", "who decided the verdict (required)")
	cmd.Flags().StringVar(&reason, "reason", "", "why: required for not-a-vulnerability and public-track")
	cmd.Flags().StringVar(&existsWithout, "exists-without", "", "not-abcds: why the problem would exist without abcd")
	cmd.Flags().StringVar(&cannotHelp, "cannot-help", "", "not-abcds: why abcd cannot help without changing how the repository works")
	return cmd
}

func renderEmbargoTriage(w io.Writer, res embargo.TriageResult) {
	fmt.Fprintf(w, "abcd embargo triage — %s (%s): %s, decided by %s at %s; the record is %s in the store\n",
		res.GHSAID, res.IssueID, res.Verdict, termsafe.Sanitize(res.DecidedBy), res.DecidedAt.Format("2006-01-02T15:04:05Z"), res.RecordStatus)
	if res.Draft != "" {
		fmt.Fprintf(w, "closing draft (%s):\n", termsafe.Sanitize(res.DraftPath))
		for _, line := range strings.Split(strings.TrimRight(res.Draft, "\n"), "\n") {
			fmt.Fprintf(w, "  | %s\n", termsafe.Sanitize(line))
		}
		fmt.Fprintf(w, "next: abcd embargo close %s (shows the text and its digest; nothing is sent)\n", res.GHSAID)
	}
	if res.CaptureCommand != "" {
		fmt.Fprintf(w, "run in the public checkout:\n  %s\n", termsafe.Sanitize(res.CaptureCommand))
	}
}

func newEmbargoCloseCommand(asJSON *bool) *cobra.Command {
	var approve string
	cmd := &cobra.Command{
		Use: "close <iss-N|GHSA-id>",
		Long: "Close one advisory whose verdict is not-abcds or not-a-vulnerability on\n" +
			"GitHub, with the closing explanation its triage drafted. GitHub offers no\n" +
			"comment on a repository advisory, so the explanation is posted at the head of\n" +
			"the advisory's description, with the reporter's own text kept whole below a\n" +
			"separator line.\n\n" +
			"Without --approve it reads the advisory and shows exactly the text that would\n" +
			"be posted and the digest that approves it; nothing but that read is sent. With\n" +
			"--approve <digest> it re-reads both and closes only when the digest is the\n" +
			"one over exactly that text and the description as read, then reads the\n" +
			"advisory back to confirm it is closed with the text. A digest for any other\n" +
			"text sends nothing. A host passes --approve only after the person, in that\n" +
			"session, has seen the text and said yes to it.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd embargo close"
			root, home, err := embargoPlace(verb)
			if err != nil {
				return err
			}
			res, err := embargo.Close(embargo.CloseRequest{Home: home, RepoRoot: root, Target: args[0], Approve: approve})
			if err != nil {
				return embargoFailure(verb, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderEmbargoClose(w, res) })
		},
	}
	cmd.Flags().StringVar(&approve, "approve", "", "the digest shown for the text; only on the person's yes to exactly that text")
	return cmd
}

func renderEmbargoClose(w io.Writer, res embargo.CloseResult) {
	if res.Closed {
		at := ""
		if res.ClosedAt != nil {
			at = " at " + res.ClosedAt.Format("2006-01-02T15:04:05Z")
		}
		fmt.Fprintf(w, "abcd embargo close — %s (%s) is closed on GitHub with the approved text%s, confirmed by reading it back\n",
			res.GHSAID, res.IssueID, at)
		return
	}
	fmt.Fprintf(w, "abcd embargo close — %s (%s), verdict %s, %s on GitHub; nothing was sent\n",
		res.GHSAID, res.IssueID, res.Verdict, res.ForgeState)
	fmt.Fprintln(w, "the text that would be posted at the head of the advisory's description:")
	for _, line := range strings.Split(strings.TrimRight(res.Text, "\n"), "\n") {
		fmt.Fprintf(w, "  | %s\n", termsafe.Sanitize(line))
	}
	fmt.Fprintf(w, "digest: %s\n", res.Digest)
	fmt.Fprintf(w, "approve with: %s\n", res.ApproveWith)
}

func newEmbargoCheckCommand(asJSON *bool) *cobra.Command {
	var rng string
	var files, allow []string
	cmd := &cobra.Command{
		Use: "check",
		Long: "The outbound embargo check: before anything is made public, read a commit\n" +
			"range (--range <a>..<b>: each commit's message and the lines it adds, a\n" +
			"merge's against its first parent) and any texts (--file <path>, repeatable;\n" +
			"- reads stdin), and refuse every unpublished advisory's id, a CVE id\n" +
			"reserved for one, its title, the store's location on this machine, and any\n" +
			"other advisory id --allow does not name. A finding names where the match is\n" +
			"and never repeats it.\n\n" +
			"Limits: a title shorter than 12 characters is not matched on its own (its\n" +
			"id still is); a paraphrase or a count (\"fixes several private reports\") is\n" +
			"not recognised, and stays the author's to avoid. The check warns when the\n" +
			"last intake is more than a day old.\n\n" +
			"Exit 0 clean, or when no embargo store exists on this machine; exit 2 on any\n" +
			"finding.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const verb = "abcd embargo check"
			root, home, err := embargoPlace(verb)
			if err != nil {
				return err
			}
			res, err := embargo.Check(embargo.CheckRequest{Home: home, RepoRoot: root, Range: rng, Files: files,
				Stdin: cmd.InOrStdin(), Allow: allow})
			if err != nil {
				return embargoFailure(verb, err)
			}
			for i := range res.Findings {
				res.Findings[i].Source = fsutil.RedactHome(res.Findings[i].Source)
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderEmbargoCheck(w, res) }); rerr != nil {
				return rerr
			}
			if !res.Clean() {
				return &exitError{Code: 2}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&rng, "range", "", "a commit range <a>..<b> whose messages and added lines are read")
	cmd.Flags().StringArrayVar(&files, "file", nil, "a text to read (repeatable; - reads stdin)")
	cmd.Flags().StringArrayVar(&allow, "allow", nil, "an advisory id of another project, already public, that may appear (repeatable)")
	return cmd
}

func renderEmbargoCheck(w io.Writer, res embargo.CheckResult) {
	read := fmt.Sprintf("read %s and %s", countOf(res.Commits, "commit"), countOf(res.Files, "text"))
	if res.NoStore {
		fmt.Fprintf(w, "abcd embargo check — clean: %s; %s\n", res.Note, read)
		return
	}
	verdict := "clean"
	if !res.Clean() {
		verdict = "REFUSED, " + countOf(len(res.Findings), "finding")
	}
	fmt.Fprintf(w, "abcd embargo check — %s: %s under embargo; %s\n", verdict, advisoryCount(res.UnderEmbargo), read)
	if len(res.Allowed) > 0 {
		fmt.Fprintf(w, "  allowed: %s\n", strings.Join(res.Allowed, ", "))
	}
	for _, f := range res.Findings {
		where := f.Source
		if f.Where != "" {
			where += ", " + f.Where
		}
		if f.Line > 0 {
			where += fmt.Sprintf(", line %d", f.Line)
		}
		line := fmt.Sprintf("  %s  %s", f.Kind, where)
		if f.Snippet != "" {
			line += ": " + f.Snippet
		}
		fmt.Fprintln(w, termsafe.Sanitize(line))
	}
	switch {
	case res.IntakeAt == nil:
		fmt.Fprintln(w, "warning: intake has never run here, so an advisory reported on GitHub may be missing from the check; run `abcd embargo intake`")
	case res.IntakeStale:
		fmt.Fprintf(w, "warning: the last intake was at %s, more than a day ago; run `abcd embargo intake` so a newer advisory is in the check\n",
			res.IntakeAt.Format("2006-01-02T15:04:05Z"))
	}
}

func newEmbargoPublishCommand(asJSON *bool) *cobra.Command {
	var release, approve string
	cmd := &cobra.Command{
		Use: "publish",
		Long: "Show the publication plan for the release tag (--release, default the highest\n" +
			"vMAJOR.MINOR.PATCH tag): for each advisory judged a vulnerability and not yet\n" +
			"published, its record resolved in that release, its fix commit, the first\n" +
			"release tag holding the fix, the patched version that gives, and the affected\n" +
			"range in GitHub's form. An advisory whose record is not resolved there, whose\n" +
			"fix is in no release tag, whose package is not set on GitHub, or whose text\n" +
			"carries a secret, a private name or another advisory still under embargo is\n" +
			"left out, saying why. Without --approve nothing but reads is sent.\n\n" +
			"With --approve <digest> of exactly the plan shown it publishes one advisory as\n" +
			"a canary and reads it back, then the rest the same way, stopping at the first\n" +
			"mismatch, and confirms none is left in draft. A host passes --approve only\n" +
			"after the person, in that session, has seen the plan and said yes to it.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const verb = "abcd embargo publish"
			root, home, err := embargoPlace(verb)
			if err != nil {
				return err
			}
			res, err := embargo.Publish(embargo.PublishRequest{Home: home, RepoRoot: root, Release: release, Approve: approve})
			if err != nil {
				return embargoFailure(verb, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderEmbargoPublish(w, res) })
		},
	}
	cmd.Flags().StringVar(&release, "release", "", "the release tag vMAJOR.MINOR.PATCH to publish from (default the highest)")
	cmd.Flags().StringVar(&approve, "approve", "", "the plan's digest; only on the person's yes to exactly that plan")
	return cmd
}

func renderEmbargoPublish(w io.Writer, res embargo.PublishResult) {
	p := res.Plan
	if res.Approved {
		fmt.Fprintf(w, "abcd embargo publish — %s from %s published, the first as the canary", advisoryCount(len(res.Published)), p.Release)
		if res.NoDraftLeft {
			fmt.Fprint(w, "; GitHub lists none of them in draft or triage")
		}
		fmt.Fprintln(w)
		for _, a := range res.Published {
			fmt.Fprintf(w, "  published %s (%s) at %s\n", a.GHSAID, a.IssueID, a.PublishedAt.Format("2006-01-02T15:04:05Z"))
		}
		return
	}
	fmt.Fprintf(w, "abcd embargo publish — plan for %s in %s: %s, %s left out; nothing was sent\n",
		termsafe.Sanitize(p.Repo), p.Release, advisoryCount(len(p.Entries)), advisoryCount(len(p.LeftOut)))
	for _, e := range p.Entries {
		fmt.Fprintf(w, "  %s (%s): fix %s, first released in %s, patched %s\n", e.GHSAID, e.IssueID, e.FixCommit[:12], e.FirstTag, e.PatchedVersion)
		for _, v := range e.Vulnerabilities {
			fmt.Fprintf(w, "    %s %s: affected %q, patched %q\n", termsafe.Sanitize(v.Package.Ecosystem), termsafe.Sanitize(v.Package.Name),
				v.VulnerableVersionRange, v.PatchedVersions)
		}
	}
	for _, l := range p.LeftOut {
		fmt.Fprintf(w, "  left out %s (%s): %s\n", l.GHSAID, l.IssueID, termsafe.Sanitize(l.Reason))
	}
	if p.ApproveWith != "" {
		fmt.Fprintf(w, "digest: %s\napprove with: %s\n", p.Digest, p.ApproveWith)
	}
}

// advisoryCount is n advisories, spelled as English spells it.
func advisoryCount(n int) string {
	if n == 1 {
		return "1 advisory"
	}
	return fmt.Sprintf("%d advisories", n)
}
