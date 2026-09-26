package cli

// scribe.go is the front door onto internal/core/scribe — the ledger scribe's
// context assembler and output ingest (itd-2609020625402599,
// spc-2609020626045177).
//
// `scribe` is a top-level verb and never a sub-verb of `reading`, because the
// two contexts must never share a front door: a reading is handed a slice of
// the shipped repository and no ledger, and the scribe the ledger and no
// shipped tree (brief invariant 15; adr-2609021016275803).
//
// Nothing here runs the scribe. `assemble` produces the context a scribe session
// is handed and the manifest an auditor checks it by; dispatching it is host
// work, and the host obligation to grant that session nothing else is stated on
// the plugin surface, never claimed as an enforcement this binary performs.
// `ingest` validates what the session returned and writes it through the
// capture verbs.
//
// Exit codes follow `reading`'s shape: 0 when the verb landed, 2 for every
// refusal, with a result rendered first whenever it has something to disclose.

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/scribe"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newScribeCommand builds the `scribe` sub-tree.
func newScribeCommand(asJSON *bool) *cobra.Command {
	scribeCmd := &cobra.Command{
		Use:   "scribe",
		Short: "Ledger scribe: assemble its context from the ledger alone, and ingest what it transcribed",
		Long: "Build the ledger scribe's context and ingest what the scribe returns.\n\n" +
			"The scribe transcribes a reading run's records and the researcher's dispositions into the\n" +
			"ledger's declared shapes, and authors nothing. Its context is the reading assembler's exact\n" +
			"inverse: ledger content only, drawn from the issue ledger's own directories, and the\n" +
			"researcher's supplied text. `assemble` builds it with a manifest of every path passed;\n" +
			"`ingest` validates the scribe's output and refuses anything the scribe authored.",
		Args: cobra.NoArgs,
		RunE: helpRunE,
	}

	var run, dispositions, outDir string
	var dryRun bool
	assembleCmd := &cobra.Command{
		Use:   "assemble --run <rdg-N> --dispositions <path>",
		Short: "Build a scribe session's context from the ledger and the supplied dispositions",
		Long: "Build the context one scribe session is handed, for one ingested reading run.\n\n" +
			"The context is positive inclusion at directory grain: the issue ledger's own directories\n" +
			"(its reading records, dispositions, admissions, surprises and reframes, and its three status\n" +
			"directories), derived from the ledger's directory list, and the researcher's dispositions\n" +
			"text read whole. Nothing else is walked, and an item outside that list is refused whatever\n" +
			"route it arrived by. The run must be ingested: its records come from the store, never from\n" +
			"a raw reading output handed over again.\n\n" +
			"The context and a manifest naming every path passed, by hash, are parked in the local tier\n" +
			"(or under --out, which may not be a directory a reading's include table reaches). Nothing in\n" +
			"the durable record is touched. Both carry the scribe's per-run context stamp.",
		Example: "  abcd scribe assemble --run rdg-2609250000000001 --dispositions ./dispositions.md --json",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return &exitError{Code: 2, Msg: "scribe assemble: this verb takes no positional argument; " +
					"the invocation is --run and --dispositions, and the context is the ledger's own"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if run == "" {
				return &exitError{Code: 2, Msg: "scribe assemble: --run <rdg-N> is required: the ingested " +
					"reading run the session transcribes dispositions for"}
			}
			if dispositions == "" {
				return &exitError{Code: 2, Msg: "scribe assemble: --dispositions <path> is required: the " +
					"researcher's dispositions text, which the scribe transcribes and never writes"}
			}
			cwd := mustCwd()
			resolvedOut := resolveAgainst(cwd, outDir)
			res, err := scribe.Assemble(scribe.AssembleRequest{
				RepoRoot:         captureRoot(cwd),
				Run:              run,
				DispositionsPath: resolveAgainst(cwd, dispositions),
				OutDir:           resolvedOut,
				OutDirLabel:      outDir,
				DryRun:           dryRun,
			})
			if err != nil {
				return scribeRefusal("scribe assemble", err)
			}
			// The core was handed the resolved path; the operator is shown the
			// string they typed.
			if outDir != "" {
				res.OutDir = outDir
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderScribeAssemble(w, res)
			})
		},
	}
	assembleCmd.Flags().StringVar(&run, "run", "", "the ingested reading run the session transcribes for (rdg-N)")
	assembleCmd.Flags().StringVar(&dispositions, "dispositions", "",
		"the researcher's dispositions text, read whole and carried verbatim")
	assembleCmd.Flags().StringVar(&outDir, "out", "",
		"an empty or absent directory the context and the manifest are written to\n"+
			"(default: the local-tier scribe run directory)")
	assembleCmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"write nothing; with --out the two artefacts still land in that directory")

	var scribeJSON, contextPath, ingestDispositions string
	ingestCmd := &cobra.Command{
		Use:   "ingest --scribe-json <path> --dispositions <path>",
		Short: "Validate a scribe session's output and write what it transcribed",
		Long: "Validate the JSON a scribe session returned and write its records through the capture verbs.\n\n" +
			"The context the session was handed is proven first: it must hash to its parked manifest, and\n" +
			"the output must cite that hash. The pair is parked where a scribe session could rewrite it, so\n" +
			"--dispositions names the researcher's own text again, the file assemble was handed: the\n" +
			"manifest's supplied hash and the context's supplied copy must both equal it, and every check\n" +
			"below reads it. Then the output is refused if the scribe authored anything —\n" +
			"a field outside the declared shapes, an item the supplied dispositions never name, a state or an\n" +
			"admission the item's own line of the supplied text does not carry, or a ground, exit condition\n" +
			"or surprise that does not stand verbatim in the supplied text once whitespace is folded — or if\n" +
			"it passes over an unanswered item of the run in silence. Nothing is written until all of that\n" +
			"holds.\n\n" +
			"Dispositions, admissions and surprises are then written in that order through the capture verbs,\n" +
			"which apply their own redaction and refusals, the ordering gate included; the first refusal stops\n" +
			"the ingest and names what landed before it. Fidelity flags and refusals are reported and never\n" +
			"written. Once every write has landed the manifest is promoted beside the run, write-once.",
		Example: "  abcd scribe ingest --scribe-json ./scribe-output.json --dispositions ./dispositions.md --json",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return &exitError{Code: 2, Msg: "scribe ingest: this verb takes no positional argument; " +
					"the output names its own run"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if scribeJSON == "" {
				return &exitError{Code: 2, Msg: "scribe ingest: --scribe-json <path> is required: the JSON " +
					"the scribe session returned"}
			}
			if ingestDispositions == "" {
				return &exitError{Code: 2, Msg: "scribe ingest: --dispositions <path> is required: the " +
					"researcher's dispositions text assemble was handed, which every word the scribe carries is " +
					"held to"}
			}
			cwd := mustCwd()
			res, err := scribe.Ingest(scribe.IngestRequest{
				RepoRoot:         captureRoot(cwd),
				ScribeJSONPath:   resolveAgainst(cwd, scribeJSON),
				ContextPath:      resolveAgainst(cwd, contextPath),
				DispositionsPath: resolveAgainst(cwd, ingestDispositions),
			})
			if err != nil {
				// A refusal after something landed discloses what landed, before it
				// exits: the operator's handle on the partial state is the render.
				if len(res.Landed()) > 0 {
					_ = render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
						renderScribeIngest(w, res)
					})
				}
				return scribeRefusal("scribe ingest", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderScribeIngest(w, res)
			})
		},
	}
	ingestCmd.Flags().StringVar(&scribeJSON, "scribe-json", "", "path to the JSON the scribe session returned")
	ingestCmd.Flags().StringVar(&ingestDispositions, "dispositions", "",
		"the researcher's dispositions text, the file assemble was handed")
	ingestCmd.Flags().StringVar(&contextPath, "context", "",
		"the context the session was handed, when assemble wrote it under --out\n"+
			"(default: the local-tier scribe run directory of the output's run)")

	scribeCmd.AddCommand(assembleCmd)
	scribeCmd.AddCommand(ingestCmd)
	return scribeCmd
}

// resolveAgainst takes an operator's relative path against the working
// directory, which is a transport fact the core does not hold.
func resolveAgainst(cwd, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, filepath.FromSlash(p))
}

// scribeRefusal is every scribe refusal: exit 2, paths scrubbed, the core's own
// tag replaced by the verb's.
func scribeRefusal(verb string, err error) *exitError {
	return &exitError{Code: 2, Msg: verb + ": " + strings.TrimPrefix(scrubPaths(err), "scribe: ")}
}

// renderScribeAssemble writes one assembly's text render.
func renderScribeAssemble(w io.Writer, res scribe.AssembleResult) {
	fmt.Fprintf(w, "abcd scribe assemble — %s: %d ledger record(s) and the supplied dispositions\n",
		res.Run, res.ItemCount)
	fmt.Fprintf(w, "  context stamp:  %s\n", res.ContextStamp)
	fmt.Fprintf(w, "  context sha256: %s (the output cites this)\n", res.ContextSHA256)
	if res.Written {
		fmt.Fprintf(w, "  parked in %s: %s\n", termsafe.Sanitize(res.OutDir), strings.Join(res.Artefacts, ", "))
	} else {
		fmt.Fprintln(w, "  dry run: nothing written")
	}
	fmt.Fprintln(w, "Hand the context, and nothing else, to a scribe session that is not a reading session.")
}

// renderScribeIngest writes one ingest's text render. Every payload-derived
// string is neutralised before it reaches the terminal.
func renderScribeIngest(w io.Writer, res scribe.IngestResult) {
	fmt.Fprintf(w, "abcd scribe ingest — %s\n", res.Run)
	for _, d := range res.Dispositions {
		fmt.Fprintf(w, "  disposition %s  %s  %s\n", d.ID, d.Item, d.State)
	}
	for _, a := range res.Admissions {
		fmt.Fprintf(w, "  admission   %s  %s (disposition %s)\n", a.Admission, a.Item, a.Disposition)
	}
	for _, s := range res.Surprises {
		fmt.Fprintf(w, "  surprise    %s  occasioned by %s\n", s.ID, s.OccasionedBy)
	}
	if len(res.Outstanding) > 0 {
		fmt.Fprintf(w, "  outstanding: %s\n", termsafe.Sanitize(strings.Join(res.Outstanding, ", ")))
	}
	for _, f := range res.FidelityFlags {
		fmt.Fprintf(w, "  FIDELITY FLAG (unresolved): %q against %q\n",
			termsafe.Sanitize(f.First), termsafe.Sanitize(f.Second))
	}
	for _, r := range res.Refusals {
		fmt.Fprintf(w, "  scribe refused %q: %s\n", termsafe.Sanitize(r.Subject), termsafe.Sanitize(r.Reason))
	}
	if res.Manifest != "" {
		fmt.Fprintf(w, "  manifest promoted: %s\n", res.Manifest)
	}
}
