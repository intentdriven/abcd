package cli

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/matrix"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// maxDeclarationFile bounds the declaration file the verb reads.
const maxDeclarationFile = 256 << 10

// labMatrixView is the plan as the verb shows it, with the command that
// starts exactly this plan.
type labMatrixView struct {
	matrix.Plan
	ApproveWith string `json:"approve_with"`
}

// labMatrixStarted is a consented run: its record is written; starting copies
// is spec step 3's, so nothing is launched yet, and the verb says so.
type labMatrixStarted struct {
	RunID    string `json:"run_id"`
	Digest   string `json:"digest"`
	Launched bool   `json:"launched"`
	Note     string `json:"note"`
}

// recordOnly is the Starter until the run's trial mode exists (spec step 3):
// the record is written and nothing is launched.
type recordOnly struct{}

func (recordOnly) Start(matrix.Plan, matrix.RunRecord) error { return nil }

// newLabMatrixCommand builds `lab matrix`, the front door onto a model matrix
// run (itd-2610161733316518, spc-2610161949559050 step 2): show the plan, and
// with the person's consent to exactly that plan, record the run.
func newLabMatrixCommand(asJSON *bool) *cobra.Command {
	var approve string
	cmd := &cobra.Command{
		Use: "matrix <lab-id> <declaration.json>",
		Long: "abcd lab matrix <lab-id> <declaration.json> [--approve <digest>]\n\n" +
			"Show the plan for a model matrix run of the declaration in the lab: how many\n" +
			"copies each cell makes, the expected cost by source against the spending cap,\n" +
			"the load cap below this machine's cores, the time, and the judges each task\n" +
			"needs. Building it reads the declaration and this machine's configuration and\n" +
			"nothing else: no network call, no price fetched, nothing written. Its digest\n" +
			"names exactly that plan.\n\n" +
			"--approve <digest> records the run, once the person has seen the plan and said\n" +
			"yes to it: the plan is built again and must carry that digest. A host passes\n" +
			"--approve only after the person, in that session, has seen the plan and said\n" +
			"yes to it. Starting the run's copies is not built yet, so a consented run is\n" +
			"recorded in the lab and nothing is launched; the output says so.\n\n" +
			"Refused, exit 2, with nothing written: a declaration that does not parse, a\n" +
			"cell route the runner rung or this machine's runner configuration refuses, a\n" +
			"cell on a runner that names no model, a model its provider does not list, an\n" +
			"OpenRouter or local cell with no pin, a metered cell with no spending cap, a\n" +
			"local cell with no load cap or one at or above the cores, and an approval that\n" +
			"is not the plan's digest. A halted lab exits 1.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := labRoot("abcd lab matrix")
			if err != nil {
				return err
			}
			decl, err := fsutil.ReadGuarded(args[1], maxDeclarationFile)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd lab matrix: the declaration " + termsafe.Sanitize(fsutil.RedactHome(args[1])) +
					" cannot be read: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; nothing was written"}
			}
			cores := runtime.NumCPU()
			if approve == "" {
				plan, notes, err := matrix.PlanForLab(root, args[0], decl, cores)
				for _, n := range notes {
					fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
				}
				if err != nil {
					return matrixFailure(err)
				}
				view := labMatrixView{Plan: plan, ApproveWith: "abcd lab matrix " + args[0] + " " + args[1] + " --approve " + plan.Digest}
				return render(cmd.OutOrStdout(), *asJSON, view, func(w io.Writer) { renderMatrixPlan(w, view) })
			}
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("abcd lab matrix: the running binary cannot be named: %w", err)
			}
			rec, plan, err := matrix.StartRun(root, args[0], decl, approve, cores, exe, core.Version, recordOnly{})
			if err != nil {
				return matrixFailure(err)
			}
			out := labMatrixStarted{RunID: rec.RunID, Digest: plan.Digest, Launched: false,
				Note: "the run's record is written in the lab; starting its copies is not built yet (spec step 3), so nothing was launched"}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "abcd lab matrix: recorded run %s for the plan %s\n  %s\n", out.RunID, out.Digest[:12], out.Note)
			})
		},
	}
	cmd.Flags().StringVar(&approve, "approve", "", "the digest of the plan the person saw and said yes to; records the run")
	return cmd
}

// matrixFailure maps a refusal to exit 2 and a halted lab to exit 1, the
// message masked and with no home path.
func matrixFailure(err error) error {
	msg := "abcd lab matrix: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	if errors.Is(err, lab.ErrHalted) {
		return &exitError{Code: 1, Msg: msg}
	}
	return &exitError{Code: 2, Msg: msg + "; nothing was written"}
}

// onOff names a switch.
func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

// renderMatrixPlan draws the plan for the person to read before consenting.
func renderMatrixPlan(w io.Writer, v labMatrixView) {
	fmt.Fprintf(w, "abcd lab matrix — plan for %s at %s\n", v.Lab, v.FrozenCommit[:12])
	fmt.Fprintf(w, "  copies:      %d in the pilot, at most %d\n", v.Copies.PilotTotal, v.Copies.MaxTotal)
	for _, c := range v.Cells {
		on := ""
		if c.Runner != "" {
			on = " on " + termsafe.Sanitize(c.Runner)
		}
		fmt.Fprintf(w, "    %-14s %-9s %s (%s)%s\n", termsafe.Sanitize(c.Cell), c.Role, termsafe.Sanitize(c.Model), c.Source, on)
		// The digest covers each cell's pin, every field of it, so the person
		// sees every field too.
		if p := c.ProviderPin; p != nil {
			fmt.Fprintf(w, "      provider pin: order %s; fallbacks %s; parameters required %s; quantizations %s; data collection %s; slug %s\n",
				termsafe.Sanitize(cmp.Or(strings.Join(p.Order, ", "), "none")), onOff(p.AllowFallbacks), onOff(p.RequireParameters),
				termsafe.Sanitize(cmp.Or(strings.Join(p.Quantizations, ", "), "any")), termsafe.Sanitize(cmp.Or(p.DataCollection, "the provider's default")),
				termsafe.Sanitize(cmp.Or(p.CanonicalSlug, "none")))
		}
		if p := c.LocalPin; p != nil {
			file := ""
			if p.WeightsFile != "" {
				file = " (" + termsafe.Sanitize(p.WeightsFile) + ")"
			}
			fmt.Fprintf(w, "      local pin: %s %s, weights %s%s, %s, context %d, kv cache %s\n", termsafe.Sanitize(p.Runtime),
				termsafe.Sanitize(p.RuntimeVersion), termsafe.Sanitize(p.WeightsSHA256), file, termsafe.Sanitize(p.Quantisation),
				p.ContextLength, termsafe.Sanitize(p.KVCacheType))
		}
	}
	cap := "none named"
	if v.SpendCap != nil {
		cap = fmt.Sprintf("%.2f %s (%s)", v.SpendCap.Amount, v.SpendCap.Unit, v.SpendCapOrigin)
	}
	fmt.Fprintf(w, "  spend cap:   %s\n", cap)
	for _, c := range v.Cost {
		switch {
		case c.Expected != nil:
			fmt.Fprintf(w, "    %-12s expected %.2f %s at most\n", c.Source, c.Expected.Amount, c.Expected.Unit)
		default:
			fmt.Fprintf(w, "    %-12s %s\n", c.Source, c.Note)
		}
	}
	load := "not used (no local cell)"
	if v.LoadCap != nil {
		load = fmt.Sprintf("%d of %d cores", *v.LoadCap, v.Cores)
	}
	fmt.Fprintf(w, "  load cap:    %s\n", load)
	if v.Time.BoundMinutes != nil {
		fmt.Fprintf(w, "  time:        at most %.0f minutes (%s)\n", *v.Time.BoundMinutes, v.Time.Note)
	} else {
		fmt.Fprintf(w, "  time:        %s\n", v.Time.Note)
	}
	for _, j := range v.Judges {
		fmt.Fprintf(w, "  judges:      %s (%s): %d, %s\n", termsafe.Sanitize(j.Task), j.Kind, j.Count, j.Who)
	}
	fmt.Fprintf(w, "  digest:      %s\n", v.Digest)
	fmt.Fprintf(w, "  to start it: %s\n", strings.TrimSpace(termsafe.Sanitize(v.ApproveWith)))
}
