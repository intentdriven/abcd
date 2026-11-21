package cli

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/matrix"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// maxDriftLines bounds the drifted fields the text report prints; --json
// carries every one the report lists.
const maxDriftLines = 20

// newLabMatrixReportCommand builds `lab matrix report`: a recorded run's
// report, read from its records in the lab (spc-2610161949559050 step 8).
func newLabMatrixReportCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "report <lab-id> <run-id>",
		Long: "abcd lab matrix report <lab-id> <run-id>\n\n" +
			"Print a recorded run's report from its records in the lab, writing nothing.\n" +
			"A code task gives each builder's pass rate on its hidden tests with a 95%\n" +
			"interval, its pass^k, what the reviewers caught, its cost per passing build\n" +
			"by source and its model time apart from gate waits; each pair of builders is\n" +
			"ranked only at the repeats its power estimate asks for, otherwise it states\n" +
			"the smallest difference it could have detected. A written doc gives each\n" +
			"model's rating from all the judges' rankings with its interval, whether it is\n" +
			"separable, the judges' agreement and each copy's length; below three judges,\n" +
			"each judge's ranking and the good-enough counts only. A check gives the table\n" +
			"the person's locked verdicts unlocked. A re-run's report shows, field by field,\n" +
			"where it drifted from the run it repeats. Every report ends with its\n" +
			"limitations and suggested improvements.\n\n" +
			"Refused, exit 2: a run id that is not one, and a run the lab never recorded.\n" +
			"A halted lab exits 1.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab matrix report")
			if err != nil {
				return err
			}
			rep, err := matrix.BuildReport(root, args[0], args[1])
			if err != nil {
				return matrixReportFailure(err)
			}
			return render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) { renderMatrixReport(w, rep) })
		},
	}
}

// matrixReportFailure is matrixFailure for a verb that writes nothing.
func matrixReportFailure(err error) error {
	e := matrixFailure(err)
	if ee, ok := e.(*exitError); ok {
		ee.Msg = strings.TrimSuffix(strings.Replace(ee.Msg, "abcd lab matrix: ", "abcd lab matrix report: ", 1), "; nothing was written")
	}
	return e
}

// newLabMatrixRerunCommand builds `lab matrix rerun`: a re-run of a recorded
// run from its own declaration, under the person's consent to its own plan.
func newLabMatrixRerunCommand(asJSON *bool) *cobra.Command {
	var approve string
	cmd := &cobra.Command{
		Use: "rerun <lab-id> <run-id|run-folder>",
		Long: "abcd lab matrix rerun <lab-id> <run-id|run-folder> [--approve <digest>]\n\n" +
			"Show the plan for a re-run of a recorded run: the run named by its id in the\n" +
			"lab, or a run's folder carried from elsewhere. The plan is built from that\n" +
			"run's own declaration, as `abcd lab matrix` builds one, and writes nothing.\n\n" +
			"--approve <digest> records the re-run, once the person has seen the plan and\n" +
			"said yes to it, and copies the first run's records beside it, so the re-run's\n" +
			"report shows field by field where it drifted. A model run is re-runnable, not\n" +
			"reproducible: the drift is the finding. Starting the copies is not built yet,\n" +
			"so nothing is launched; the output says so.\n\n" +
			"Refused, exit 2, with nothing written: a run the lab never recorded, a folder\n" +
			"that holds no run's records, a run whose frozen commit is not the lab's pin,\n" +
			"every refusal `abcd lab matrix` gives, and an approval that is not the plan's\n" +
			"digest. A halted lab exits 1.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab matrix rerun")
			if err != nil {
				return err
			}
			cores := runtime.NumCPU()
			if approve == "" {
				plan, notes, err := matrix.PlanRerun(root, args[0], args[1], cores)
				for _, n := range notes {
					fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
				}
				if err != nil {
					return matrixFailure(err)
				}
				view := labMatrixView{Plan: plan, ApproveWith: "abcd lab matrix rerun " + args[0] + " " + args[1] + " --approve " + plan.Digest}
				return render(cmd.OutOrStdout(), *asJSON, view, func(w io.Writer) { renderMatrixPlan(w, view) })
			}
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("abcd lab matrix rerun: the running binary cannot be named: %w", err)
			}
			rec, plan, err := matrix.StartRerun(root, args[0], args[1], approve, cores, exe, core.Version, recordOnly{})
			if err != nil {
				return matrixFailure(err)
			}
			out := labMatrixStarted{RunID: rec.RunID, Digest: plan.Digest, Launched: false,
				Note: "the re-run's record is written in the lab beside a copy of the first run's records; starting its copies is not built yet (spec step 3), so nothing was launched"}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "abcd lab matrix rerun: recorded run %s for the plan %s\n  %s\n", out.RunID, out.Digest[:12], out.Note)
			})
		},
	}
	cmd.Flags().StringVar(&approve, "approve", "", "the digest of the plan the person saw and said yes to; records the re-run")
	return cmd
}

// rate prints a rate of nought to one with its interval, or that none counts.
func rate(r *float64, iv *matrix.Interval) string {
	if r == nil {
		return "none counted"
	}
	if iv == nil {
		return fmt.Sprintf("%.2f", *r)
	}
	return fmt.Sprintf("%.2f [%.2f, %.2f]", *r, iv.Low, iv.High)
}

// renderMatrixReport draws a run's report for the person to read.
func renderMatrixReport(w io.Writer, r matrix.Report) {
	s := termsafe.Sanitize
	fmt.Fprintf(w, "abcd lab matrix report — %s (lab %s, frozen %s)\n", s(r.Run), s(r.Lab), s(r.FrozenCommit[:min(12, len(r.FrozenCommit))]))
	fmt.Fprintf(w, "  copies recorded: %d\n", r.Trials)
	if d := r.Drift; d != nil {
		fmt.Fprintf(w, "  drift from %s: %d field(s) drifted, %d the same\n", s(d.Of), len(d.Fields)+d.More, d.Same)
		for i, f := range d.Fields {
			if i == maxDriftLines {
				fmt.Fprintf(w, "    … %d more; --json lists them\n", len(d.Fields)+d.More-maxDriftLines)
				break
			}
			fmt.Fprintf(w, "    %s: %s → %s\n", s(f.Field), s(f.First), s(f.Now))
		}
	}
	for _, c := range r.Code {
		fmt.Fprintf(w, "code %s (%s)\n", s(c.Task), s(c.Intent))
		for _, cell := range c.Cells {
			fmt.Fprintf(w, "  %-14s %s (%s)\n", s(cell.Cell), s(cell.Model), cell.Source)
			fmt.Fprintf(w, "    pass@1 %d of %d: %s", cell.PassAt1.Passed, cell.PassAt1.Attempts, rate(cell.PassAt1.Rate, cell.PassAt1.Interval))
			for _, h := range cell.PassHatK {
				fmt.Fprintf(w, "; pass^%d %.2f", h.K, h.Value)
			}
			fmt.Fprintf(w, "\n    apart: %d never started, %d unexamined, %d excluded\n", cell.NotStarted, cell.Unexamined, cell.Excluded)
			fmt.Fprintf(w, "    reviews: %d confirmed finding(s), %d unconfirmed flag(s)\n", cell.Reviews.Confirmed, cell.Reviews.Unconfirmed)
			for _, k := range cell.Costs {
				switch {
				case k.Unit == "":
					fmt.Fprintf(w, "    cost: none recorded (%s)\n", s(k.Comparable))
				case k.PerPass != nil:
					fmt.Fprintf(w, "    cost: %.2f %s, %.2f a passing build (%s)\n", k.Total, s(k.Unit), *k.PerPass, s(k.Comparable))
				default:
					fmt.Fprintf(w, "    cost: %.2f %s, no passing build (%s)\n", k.Total, s(k.Unit), s(k.Comparable))
				}
			}
			fmt.Fprintf(w, "    time: %.1fs model, %.1fs waiting on gates, each a copy\n", float64(cell.ModelTimeMS)/1000, float64(cell.GateWaitMS)/1000)
		}
		for _, cmp := range c.Comparisons {
			fmt.Fprintf(w, "  %s against %s (%s): %s\n", s(cmp.A), s(cmp.B), cmp.Kind, s(cmp.Note))
		}
		for _, rv := range c.Reviewers {
			o := rv.OtherFamily
			fmt.Fprintf(w, "  reviewer %s: recall %d of %d, on failing copies %d of %d, seeded %d of %d, true negatives %d of %d; %d unanswered\n",
				s(rv.Reviewer), o.Recall.Hits, o.Recall.Of, o.RecallFailing.Hits, o.RecallFailing.Of, o.RecallSeeded.Hits, o.RecallSeeded.Of,
				o.TrueNegative.Hits, o.TrueNegative.Of, rv.Unanswered)
		}
	}
	for _, wr := range r.Writing {
		if !wr.Judged {
			fmt.Fprintf(w, "writing %s: not judged yet\n", s(wr.Task))
			continue
		}
		fmt.Fprintf(w, "writing %s: %d member judge(s); %d guest(s) and %d host self-blinded ranking(s) reported apart\n",
			s(wr.Task), wr.Members, wr.Guests, wr.HostSelfBlinded)
		if wr.Floor != "" {
			fmt.Fprintf(w, "  %s\n", s(wr.Floor))
			for _, jr := range wr.Rankings {
				var order []string
				for _, c := range jr.Ranked {
					order = append(order, c.Key.String())
				}
				fmt.Fprintf(w, "    %s: %s\n", s(jr.Judge), s(strings.Join(order, " > ")))
			}
		}
		for _, br := range wr.Ratings {
			sep := "not separable from any other"
			if br.Separable {
				sep = "separable from " + strings.Join(br.SeparableFrom, ", ")
			}
			lo, hi := br.Rating, br.Rating
			if br.Interval != nil {
				lo, hi = br.Interval.Low, br.Interval.High
			}
			fmt.Fprintf(w, "  %-14s rating %.2f [%.2f, %.2f], %s\n", s(br.Model), br.Rating, lo, hi, s(sep))
		}
		if a := wr.Agreement; a != nil {
			for _, ag := range []struct {
				name string
				g    matrix.Agreement
			}{{"good enough", a.GoodEnough}, {"order", a.Order}} {
				name, g := ag.name, ag.g
				if g.Alpha == nil {
					fmt.Fprintf(w, "  agreement on %s: not estimable (%s)\n", name, s(g.NotEstimable))
					continue
				}
				fmt.Fprintf(w, "  agreement on %s: alpha %s over %d unit(s)\n", name, rate(g.Alpha, g.Interval), g.Units)
			}
		}
		for _, g := range wr.GoodEnough {
			fmt.Fprintf(w, "  %-14s good enough %d of %d mark(s), %d copy(ies)\n", s(g.Cell), g.Good, g.Of, g.GoodCopies)
		}
		for _, c := range wr.Copies {
			fmt.Fprintf(w, "    %s: %d words, mean place %.1f\n", s(c.Key.String()), c.Words, c.MeanPlace)
		}
	}
	for _, c := range r.Check {
		if c.Table == nil {
			fmt.Fprintf(w, "check %s: not judged yet\n", s(c.Task))
			continue
		}
		fmt.Fprintf(w, "check %s: %d row(s), %d seeded\n", s(c.Task), len(c.Table.Rows), c.Table.Seeded)
		for _, sc := range c.Table.Scores {
			fmt.Fprintf(w, "  %-14s precision %d of %d, unique valid %d, recall of seeded %d of %d\n",
				s(sc.Cell), sc.Precision.Hits, sc.Precision.Of, sc.UniqueValid, sc.RecallSeeded.Hits, sc.RecallSeeded.Of)
		}
	}
	fmt.Fprintln(w, "limitations:")
	for _, l := range r.Limitations {
		fmt.Fprintf(w, "  - %s\n", s(l))
	}
	fmt.Fprintln(w, "improvements:")
	for _, l := range r.Improvements {
		fmt.Fprintf(w, "  - %s\n", s(l))
	}
}
