package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// labStore is the noun the checkout resolution names in its refusal.
const labStore = "the lab store"

// newLabCommand builds the `lab` verb — the front door onto internal/core/lab
// (itd-2609212137128014, spc-2609212141418943): mint, preflight, record, sweep
// and harvest a lab in the machine-scoped lab store, writing nothing into the
// repository it studies.
func newLabCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lab",
		Short: "Run a lab against a pinned snapshot of this repository: mint, preflight, record, sweep, harvest (bare: list this repository's labs)",
		Long: "A lab is a throwaway world pinned at one commit of this repository, run to\n" +
			"answer one question. Its evidence lives at the operator level, in\n" +
			lab.StoreDisplay + "/<root-sha>/<lab-id>/ (the same root-commit key the transcript\n" +
			"store uses), and its knowledge enters the record only through capture:\n" +
			"nothing any lab verb does writes into the repository.\n\n" +
			"Bare `abcd lab` lists this repository's labs, read-only: each lab's pin,\n" +
			"probe count, and whether a gate holds it halted.\n\n" +
			"A gate refusal halts the lab and is recorded as a finding in its findings\n" +
			"log rather than adapted around: the preflight and the retraction sweep exit 1\n" +
			"on a refusal, write their artefact, and hold the lab halted — no probe is\n" +
			"recorded — until the same gate passes again. The finding stays.\n\n" +
			"Exit 2 on a refusal of the request itself (no checkout, an unknown lab, a bad\n" +
			"name or question); nothing is written on any of them.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := labRoot("abcd lab")
			if err != nil {
				return err
			}
			ls, err := lab.List(root)
			if err != nil {
				return labFailure("abcd lab", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, ls, func(w io.Writer) { renderLabList(w, ls) })
		},
	}
	cmd.AddCommand(newLabMintCommand(asJSON), newLabPreflightCommand(asJSON), newLabRecordCommand(asJSON),
		newLabSweepCommand(asJSON), newLabHarvestCommand(asJSON))
	return cmd
}

func newLabMintCommand(asJSON *bool) *cobra.Command {
	var pin string
	cmd := &cobra.Command{
		Use:   "mint <question>",
		Short: "Mint a lab: its home in the lab store, a registry entry, a standalone snapshot at the pin, and the lifecycle's documents",
		Long: "Mint a lab for one question. The lab home is created under this repository's\n" +
			"lane of the lab store with a registry line; the snapshot is a standalone\n" +
			"clone of this repository detached at the pin (HEAD unless --pin names\n" +
			"another commit), made with no hook firing and with its remote cut, so\n" +
			"nothing done in the lab world reaches this checkout. INTENTION.md,\n" +
			"findings.md, corrections.md and amendments.md are scaffolded, with the\n" +
			"lifecycle mapped onto the home. Nothing is written into the repository.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab mint")
			if err != nil {
				return err
			}
			m, err := lab.Mint(root, strings.Join(args, " "), pin)
			if err != nil {
				return labFailure("abcd lab mint", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, m, func(w io.Writer) {
				fmt.Fprintf(w, "minted %s at pin %s\n", m.ID, m.Pin[:12])
				fmt.Fprintf(w, "  question: %s\n", termsafe.Sanitize(m.Question))
				fmt.Fprintf(w, "  home:     %s\n", m.Home)
				fmt.Fprintf(w, "  snapshot: %s\n", m.Snapshot)
				for _, p := range m.Written {
					fmt.Fprintf(w, "  wrote:    %s\n", p)
				}
				fmt.Fprintln(w, "next:")
				for _, n := range m.Next {
					fmt.Fprintf(w, "  - %s\n", n)
				}
			})
		},
	}
	cmd.Flags().StringVar(&pin, "pin", "", "the commit the snapshot is pinned at (default HEAD)")
	return cmd
}

func newLabPreflightCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "preflight <lab-id>",
		Short: "Run the harness-isolation and dual-binary checks, write them as the lab's preflight artefact, and halt the lab on a failure",
		Long: "Run the preflight against a lab's home and write it to state/preflight.md.\n\n" +
			"Harness isolation: the lab's own HOME holds no link out of the lab; the\n" +
			"snapshot is a standalone clone (not a linked worktree, borrowing no object\n" +
			"store) descending from the pin; it has no remote; and the hooks path a\n" +
			"session in it would run, with the operator's global configuration in force,\n" +
			"resolves inside the lab.\n\n" +
			"Dual binary: bin/abcd is a regular file (never a link to an operator-level\n" +
			"installation) whose embedded vintage — read without running it — is the\n" +
			"pin, unmodified; it is the same binary the first passing preflight pinned,\n" +
			"since the work binary is never rebuilt; and bin/abcd-test, when present, is a\n" +
			"separate file.\n\n" +
			"A failed check halts the lab naming it: exit 1, the refusal recorded as a\n" +
			"gate finding. A preflight that passes lifts that halt.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab preflight")
			if err != nil {
				return err
			}
			res, err := lab.Preflight(root, args[0])
			if err != nil && !errors.Is(err, lab.ErrHalted) {
				return labFailure("abcd lab preflight", err)
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderPreflight(w, res) }); rerr != nil {
				return rerr
			}
			if err != nil {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

func newLabRecordCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "record <lab-id> <probe>",
		Short: "Scaffold one probe record under the lab's state/probes/, naming the artefact it observes",
		Long: "Scaffold the record one probe writes before any harvest may cite it:\n" +
			"input, argv, exit, stdout and stderr, empty, and record.md naming the\n" +
			"artefact observed (the work binary's vintage and sha256, when there is one).\n" +
			"The verb runs nothing: the probe's command is run by whoever runs the lab, with\n" +
			"its output redirected into the scaffold. A probe is recorded once and never\n" +
			"overwritten; a re-run is a new probe. Refused on a halted lab.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab record")
			if err != nil {
				return err
			}
			p, err := lab.Record(root, args[0], args[1])
			if err != nil {
				return labFailure("abcd lab record", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, p, func(w io.Writer) {
				fmt.Fprintf(w, "recorded probe %s in %s\n", p.Name, p.Dir)
				fmt.Fprintf(w, "  artefact: %s\n", p.Artefact)
				fmt.Fprintf(w, "  fill, from that directory: %s\n", p.Fill)
			})
		},
	}
}

func newLabSweepCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "sweep <lab-id>",
		Short: "Sweep the lab's documents for every retracted pattern in corrections.md, list each instance, and halt the lab on an unapplied correction",
		Long: "A retraction is a sweep, not an edit. Every correction line in corrections.md\n" +
			"names a literal that must be absent from the lab's own documents; the sweep\n" +
			"searches them all for it — the pattern, not the instance — and lists every\n" +
			"place it still stands. The snapshot, the lab's HOME and binaries, transcripts\n" +
			"and probe records are not swept: they are the world and the instruments, not\n" +
			"claims. The result is written to state/sweep.md. An unapplied correction, or\n" +
			"one too short to mean anything, fails the sweep: exit 1, the lab halted and\n" +
			"the refusal recorded as a gate finding that names corrections by number, so\n" +
			"it never becomes an instance itself. A sweep that passes lifts that halt.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab sweep")
			if err != nil {
				return err
			}
			res, err := lab.Sweep(root, args[0])
			if err != nil && !errors.Is(err, lab.ErrHalted) {
				return labFailure("abcd lab sweep", err)
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderSweep(w, res) }); rerr != nil {
				return rerr
			}
			if err != nil {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

func newLabHarvestCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "harvest <lab-id>",
		Short: "Assemble the lab's harvest in the lifeboat's section shape, citing its probe records and listing its product findings as capture candidates",
		Long: "Assemble harvest/harvest.md from INTENTION.md, the findings log and the probe\n" +
			"records, in the lifeboat's section shape: intention, method, findings that\n" +
			"worked, findings still open, candidates, coverage. Each finding cites its probe\n" +
			"records by path; each product finding is listed as a capture candidate with\n" +
			"the capture line that files it, found during the lab, and flagged where no\n" +
			"refutation was attempted. Nothing is filed.\n\n" +
			"No claim outlives its input: a finding citing no probe record or evidence\n" +
			"file, or one missing or incomplete, is listed and the harvest refuses (exit 1,\n" +
			"nothing written). A hand-written harvest.md is never overwritten. A halted lab\n" +
			"is still harvested, its gate findings first.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab harvest")
			if err != nil {
				return err
			}
			res, err := lab.Harvest(root, args[0])
			if err != nil && !errors.Is(err, lab.ErrHalted) {
				return labFailure("abcd lab harvest", err)
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderHarvest(w, res) }); rerr != nil {
				return rerr
			}
			if err != nil {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

// labRoot resolves the checkout a lab verb runs for.
func labRoot(verb string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, labStore)
	if err != nil {
		return "", &exitError{Code: 2, Msg: verb + ": " + err.Error() + " (nothing written)"}
	}
	return root, nil
}

// labFailure maps a core error to its exit: a refusal of the request is 2, a
// halt 1, anything else a fault at 1. Every message is scrubbed of paths.
func labFailure(verb string, err error) error {
	switch {
	case errors.Is(err, lab.ErrRefused):
		return &exitError{Code: 2, Msg: verb + ": " + err.Error()}
	case errors.Is(err, lab.ErrHalted):
		return &exitError{Code: 1, Msg: verb + ": " + err.Error()}
	}
	return fmt.Errorf("%s: %w", verb, err)
}

func renderLabList(w io.Writer, ls lab.Listing) {
	if len(ls.Labs) == 0 {
		fmt.Fprintln(w, "abcd lab — no labs for this repository (mint one: abcd lab mint \"<question>\")")
		return
	}
	fmt.Fprintf(w, "abcd lab — %s in %s\n", countOf(len(ls.Labs), "lab"), ls.Store)
	for _, s := range ls.Labs {
		state := countOf(s.Probes, "probe")
		if s.Missing {
			state = "home missing"
		}
		if len(s.Halted) > 0 {
			state += ", HALTED by its " + strings.Join(s.Halted, " and ")
		}
		fmt.Fprintf(w, "  %s  pin %s  %s — %s\n", s.ID, s.Pin[:12], state, termsafe.Sanitize(s.Question))
	}
	if ls.Unparsed > 0 {
		fmt.Fprintf(w, "  (%s in the registry could not be read)\n", countOf(ls.Unparsed, "line"))
	}
}

func verdict(passed bool) string {
	if passed {
		return "PASSED"
	}
	return "HALTED"
}

func renderPreflight(w io.Writer, res lab.Preflighted) {
	fmt.Fprintf(w, "preflight %s: %s\n", res.ID, verdict(res.Passed))
	for _, c := range res.Checks {
		mark := "pass"
		if !c.OK {
			mark = "FAIL"
		}
		fmt.Fprintf(w, "  %s  %-18s %s\n", mark, c.ID, termsafe.Sanitize(c.Detail))
	}
	fmt.Fprintf(w, "artefact: %s\n", res.Artefact)
	if res.Finding != "" {
		fmt.Fprintf(w, "halted: the refusal is recorded as %s in findings.md; the halt lifts only when the preflight passes again\n", res.Finding)
	}
	if res.Lifted != "" {
		fmt.Fprintf(w, "lifted: the halt recorded as %s (the finding stays)\n", res.Lifted)
	}
}

func renderSweep(w io.Writer, res lab.Swept) {
	fmt.Fprintf(w, "sweep %s: %s — %s over %s\n", res.ID, verdict(res.Passed), countOf(len(res.Corrections), "correction"), countOf(res.Files, "document"))
	for _, c := range res.Corrections {
		switch {
		case c.Invalid != "":
			fmt.Fprintf(w, "  correction %d (line %d): UNREADABLE — %s\n", c.N, c.Line, c.Invalid)
		case c.Applied:
			fmt.Fprintf(w, "  correction %d (line %d): applied\n", c.N, c.Line)
		default:
			fmt.Fprintf(w, "  correction %d (line %d) %q: UNAPPLIED — %s\n", c.N, c.Line, termsafe.Sanitize(c.Pattern), countOf(len(c.Instances), "instance"))
			for _, in := range c.Instances {
				fmt.Fprintf(w, "    %s:%d\n", termsafe.Sanitize(in.File), in.Line)
			}
		}
	}
	for _, n := range res.NotSwept {
		fmt.Fprintf(w, "  not swept: %s\n", termsafe.Sanitize(n))
	}
	fmt.Fprintf(w, "artefact: %s\n", res.Artefact)
	if res.Finding != "" {
		fmt.Fprintf(w, "halted: the refusal is recorded as %s in findings.md; the halt lifts only when the sweep passes again\n", res.Finding)
	}
	if res.Lifted != "" {
		fmt.Fprintf(w, "lifted: the halt recorded as %s (the finding stays)\n", res.Lifted)
	}
}

func renderHarvest(w io.Writer, res lab.Harvested) {
	if !res.Written {
		fmt.Fprintf(w, "harvest %s: REFUSED — %s no record can back; nothing written\n", res.ID, countOf(len(res.Gaps), "citation"))
		for _, g := range res.Gaps {
			fmt.Fprintf(w, "  %s: %s\n", g.Finding, termsafe.Sanitize(g.Reason))
		}
		return
	}
	fmt.Fprintf(w, "harvest %s: written %s\n", res.ID, res.Artefact)
	if len(res.Halted) > 0 {
		fmt.Fprintf(w, "  the lab is halted by its %s\n", strings.Join(res.Halted, " and "))
	}
	for _, c := range res.Coverage {
		fmt.Fprintf(w, "  %-24s %s — %s\n", c.Section, c.Status, c.Note)
	}
	if len(res.Candidates) > 0 {
		fmt.Fprintln(w, "capture candidates (nothing is filed):")
		for _, c := range res.Candidates {
			fmt.Fprintf(w, "  %s  %s\n", c.Finding, termsafe.Sanitize(c.Command))
		}
	}
}
