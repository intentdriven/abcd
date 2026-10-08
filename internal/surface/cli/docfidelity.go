package cli

// docfidelity.go — the front door of the doc-fidelity gate over the brief
// (itd-60, spc-2609020903498198). The judgement is core/docfidelity's; this
// file derives the command tree from the live binary (the tree commands.md and
// surface.json are generated from), picks each enforcement point's population,
// and formats the verdict. It invents no word of the verdict.

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// closeShips names the intents a `spec close` of specID would move to
// shipped: each member no other open spec still names. A reference the store
// cannot resolve ships nothing here, and the close reports it itself.
func closeShips(repoRoot, specID string) []string {
	store, err := spec.Load(repoRoot)
	if err != nil {
		return nil
	}
	sp, ok := store.Lookup(specID)
	if !ok || sp.Status != spec.StatusOpen {
		return nil
	}
	var out []string
	for _, m := range sp.Members() {
		others := 0
		for _, o := range store.OpenSpecsForIntent(m) {
			if o.ID != sp.ID {
				others++
			}
		}
		if others == 0 {
			out = append(out, m)
		}
	}
	return out
}

// enforceDocFidelity runs the gate for population and returns the refusal the
// verb exits with, or nil. The gate is armed only in the repository that ships
// the binary its brief describes; elsewhere it judges nothing. An empty
// population still meets layer 1, the coverage floor; only the saved docs
// review waits on a shipped intent (iss-2610020728118137).
func enforceDocFidelity(repoRoot, verb string, population []string) error {
	if !docfidelity.Armed(repoRoot) {
		return nil
	}
	snap, err := SurfaceSnapshot(repoRoot)
	if err != nil {
		return &exitError{Code: 2, Msg: verb + ": the doc-fidelity gate cannot derive the command tree: " + scrubPaths(err)}
	}
	v, _, err := docfidelity.Enforce(repoRoot, snap.Commands, population)
	if err != nil {
		return &exitError{Code: 2, Msg: verb + ": the doc-fidelity gate cannot read its inputs (nothing moved): " + scrubPaths(err)}
	}
	if !v.Refuse {
		return nil
	}
	lags := "the binary it ships"
	if len(population) > 0 {
		lags = "what " + strings.Join(population, ", ") + " delivered"
	}
	var b strings.Builder
	b.WriteString(verb + ": refused by the doc-fidelity gate — the brief lags " + lags + " (nothing moved):")
	for _, r := range v.Reasons {
		b.WriteString("\n  - " + termsafe.Sanitize(r))
	}
	b.WriteString("\n  (`abcd docs fidelity` shows the whole verdict; `abcd docs fidelity record` saves a docs review for HEAD)")
	return &exitError{Code: 1, Msg: b.String()}
}

// docsFidelityReport is `abcd docs fidelity`'s JSON envelope.
type docsFidelityReport struct {
	Armed   bool                 `json:"armed"`
	Verdict docfidelity.Verdict  `json:"verdict"`
	Flagged []docfidelity.Flag   `json:"flagged,omitempty"`
	Request *docfidelity.Request `json:"request,omitempty"`
}

// newDocsFidelityCommand builds `abcd docs fidelity`: the doc-fidelity gate
// run on its own. Bare, it judges and exits 1 on a refusal — what `spec close`
// and `launch ship` would say. --report is the per-task pass: every finding,
// no refusal, exit 0. --apply writes the reviewer's drafted edits into the
// brief and flags each for review; --autonomous does the same for an
// unattended run and also hands it the reviewer's request.
func newDocsFidelityCommand(asJSON *bool) *cobra.Command {
	var report, apply, autonomous bool
	var population []string
	cmd := &cobra.Command{
		Use:  "fidelity",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const verb = "abcd docs fidelity"
			if report && (apply || autonomous) {
				return &exitError{Code: 2, Msg: verb + ": --report blocks nothing and writes nothing, so it takes neither --apply nor --autonomous"}
			}
			root, err := fidelityRoot(verb)
			if err != nil {
				return err
			}
			if !docfidelity.Armed(root) {
				return render(cmd.OutOrStdout(), *asJSON, docsFidelityReport{}, func(w io.Writer) {
					fmt.Fprintf(w, "%s — not armed: this repository does not carry %s and %s/, so its brief does not describe a binary it ships; nothing judged\n",
						verb, docfidelity.SnapshotPath, docfidelity.ChaptersDir)
				})
			}
			snap, err := SurfaceSnapshot(root)
			if err != nil {
				return &exitError{Code: 2, Msg: verb + ": cannot derive the command tree: " + scrubPaths(err)}
			}
			v, _, err := docfidelity.Gate(root, snap.Commands, population, report)
			if err != nil {
				return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
			}
			out := docsFidelityReport{Armed: true}
			if (apply || autonomous) && len(v.Proposed) > 0 && v.Review != nil {
				flags, err := docfidelity.Apply(root, v.Proposed, v.Review.Commit, time.Now())
				if err != nil {
					return &exitError{Code: 2, Msg: verb + ": the drafted edits were not applied (nothing written): " + scrubPaths(err)}
				}
				out.Flagged = flags
				if v, _, err = docfidelity.Gate(root, snap.Commands, population, report); err != nil {
					return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
				}
			}
			out.Verdict = v
			if autonomous && (v.Review == nil || v.Review.Status != docfidelity.ReviewMatch) {
				in, err := docfidelity.ReadInputs(root, snap.Commands, population)
				if err != nil {
					return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
				}
				head, err := gitutil.ResolveCommit(root, "HEAD")
				if err != nil {
					return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
				}
				req := docfidelity.NewRequest(head, in)
				out.Request = &req
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderFidelity(w, verb, out) }); rerr != nil {
				return rerr
			}
			if v.Refuse {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&report, "report", false, "the per-task pass: state every finding, refuse nothing, exit 0")
	cmd.Flags().BoolVar(&apply, "apply", false, "write the reviewer's drafted corrections into the brief and flag each for review")
	cmd.Flags().BoolVar(&autonomous, "autonomous", false,
		"an unattended run: apply the drafted corrections, list every applied edit, and hand the routine the reviewer's request; the refusals stay")
	cmd.Flags().StringSliceVar(&population, "intent", nil, "the intent(s) whose delivery is judged, named in every finding (repeatable)")
	cmd.AddCommand(newDocsFidelityRecordCommand(asJSON))
	return cmd
}

func fidelityRoot(verb string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
	}
	root, err := gitutil.CheckoutRoot(cwd, "the brief")
	if err != nil {
		return "", &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
	}
	return root, nil
}

// newDocsFidelityRecordCommand builds `abcd docs fidelity record`: it saves the
// delegated reviewer's verdict as the receipt for the commit the checkout
// stands at. The binary labels the receipt; the reviewer never does.
func newDocsFidelityRecordCommand(asJSON *bool) *cobra.Command {
	var verdictJSON string
	cmd := &cobra.Command{
		Use:  "record --verdict-json <file|->",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const verb = "abcd docs fidelity record"
			if verdictJSON == "" {
				return &exitError{Code: 2, Msg: verb + ": --verdict-json is required (a file, or - for stdin)"}
			}
			root, err := fidelityRoot(verb)
			if err != nil {
				return err
			}
			raw, err := readSynthesisPayload(cmd, verdictJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
			}
			rel, review, err := docfidelity.Record(root, raw, time.Now())
			if err != nil {
				return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err) + " (nothing written)"}
			}
			res := struct {
				Receipt string             `json:"receipt"`
				Review  docfidelity.Review `json:"review"`
			}{rel, review}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "%s — saved %s (verdict %s for %s)\n", verb, rel, review.Verdict, review.Commit)
				fmt.Fprintf(w, "  `spec close` and `launch ship` find it while HEAD stays at that commit; `abcd docs fidelity` shows the verdict\n")
			})
		},
	}
	cmd.Flags().StringVar(&verdictJSON, "verdict-json", "", "the reviewer's verdict JSON (a file, or - for stdin)")
	return cmd
}

func renderFidelity(w io.Writer, verb string, out docsFidelityReport) {
	v := out.Verdict
	mode := "gate"
	if v.Report {
		mode = "report"
	}
	fmt.Fprintf(w, "%s — %s mode", verb, mode)
	if len(v.Population) > 0 {
		fmt.Fprintf(w, " · %s", strings.Join(v.Population, ", "))
	}
	fmt.Fprintln(w)
	covered := 0
	for _, r := range v.Coverage {
		if r.Chapter != "" {
			covered++
		}
	}
	fmt.Fprintf(w, "  layer 1: %d of %d shipped surfaces named by a chapter; %d uncovered\n",
		covered, len(v.Coverage), len(v.Uncovered))
	for _, s := range v.Uncovered {
		fmt.Fprintf(w, "    uncovered %s `%s`\n", s.Kind, s.Name)
	}
	if v.Review == nil {
		fmt.Fprintln(w, "  layer 2: not run — layer 1 refused, and an undocumented surface needs no reviewer")
	} else {
		fmt.Fprintf(w, "  layer 2: the saved docs review is %s for %s\n", v.Review.Status, termsafe.Sanitize(v.Review.Commit))
	}
	for _, f := range out.Flagged {
		fmt.Fprintf(w, "  applied  %s: %q -> %q (flagged for review in %s)\n", f.Chapter,
			termsafe.Sanitize(f.Sentence), termsafe.Sanitize(f.Replacement), docfidelity.FlagsPath)
	}
	for _, f := range v.Applied {
		fmt.Fprintf(w, "  drafted and applied, awaiting review: %s: %q\n", f.Chapter, termsafe.Sanitize(f.Replacement))
	}
	for _, f := range v.Public {
		fmt.Fprintf(w, "  public doc (reported, never refuses): %s: %q (%s)\n", termsafe.Sanitize(f.Chapter),
			termsafe.Sanitize(f.Sentence), termsafe.Sanitize(f.Evidence))
	}
	for _, r := range v.Reasons {
		fmt.Fprintf(w, "  - %s\n", termsafe.Sanitize(r))
	}
	if out.Request != nil {
		fmt.Fprintf(w, "  request: review %s at %s; save the verdict with `%s`\n",
			strings.Join(out.Request.Chapters, ", "), out.Request.Commit, out.Request.RecordWith)
	}
	switch {
	case v.Report:
		fmt.Fprintln(w, "  (report only: nothing is refused)")
	case v.Refuse:
		fmt.Fprintln(w, "  REFUSED")
	default:
		fmt.Fprintln(w, "  allowed")
	}
}
