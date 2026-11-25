package cli

import (
	"cmp"
	"encoding/json"
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

// notSandboxed is what the person is told before they say yes to a plan
// (dcn-2611251934117439): a copy's runner and its code are bounded in their
// environment, working folder and lifetime, and in nothing else.
const notSandboxed = "the runners and each copy's code, its hidden tests included, run as this account on this machine " +
	"and are not sandboxed: only their environment, their working folder and how long they run are bounded"

// MarshalJSON is the view with what the person is told of the sandbox, as
// not_sandboxed, so every plan shown carries it.
func (v labMatrixView) MarshalJSON() ([]byte, error) {
	type plain labMatrixView
	return json.Marshal(struct {
		plain
		NotSandboxed string `json:"not_sandboxed"`
	}{plain(v), notSandboxed})
}

// labMatrixStarted is a consented run: its record is written and every pilot
// copy is handed off, each in its own clone (matrix.Launcher). Launched says
// the copies were handed off; Copies lists each one's hand-off, its clone
// shown with the home directory as ~.
type labMatrixStarted struct {
	RunID    string           `json:"run_id"`
	Digest   string           `json:"digest"`
	Launched bool             `json:"launched"`
	Note     string           `json:"note"`
	Copies   []matrix.HandOff `json:"copies"`
}

// matrixLauncher is the Starter a consented run is started with, from this
// checkout into the clone store under the account's home.
func matrixLauncher(root string) (*matrix.Launcher, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil, fmt.Errorf("the account's home directory, which holds the clone store, cannot be named: %v", err)
	}
	return &matrix.Launcher{RepoRoot: root, Home: home}, nil
}

// startedView is a consented run as the verb shows it.
func startedView(rec matrix.RunRecord, plan matrix.Plan, l *matrix.Launcher, what string) labMatrixStarted {
	out := labMatrixStarted{RunID: rec.RunID, Digest: plan.Digest, Launched: len(l.Handed) > 0, Copies: []matrix.HandOff{}}
	counts := map[matrix.HandOffState]int{}
	for _, h := range l.Handed {
		h.Clone = fsutil.RedactHome(h.Clone)
		h.Brief = fsutil.RedactHome(h.Brief)
		h.Gates = nil
		out.Copies = append(out.Copies, h)
		counts[h.State]++
	}
	out.Note = fmt.Sprintf("%s is written in the lab and its %d pilot copies are handed off, each in its own clone: %d for the host to make and hand back "+
		"with `abcd lab matrix copy`, %d awaiting their runner, which `abcd lab matrix launch` starts, %d refused by the secret scan",
		what, len(l.Handed), counts[matrix.HandOffWaiting], counts[matrix.HandOffAwaitingRunner], counts[matrix.HandOffRefused])
	return out
}

// renderStarted draws a consented run's hand-offs.
func renderStarted(w io.Writer, verb string, out labMatrixStarted) {
	fmt.Fprintf(w, "%s: recorded run %s for the plan %s\n  %s\n", verb, out.RunID, out.Digest[:12], termsafe.Sanitize(out.Note))
	for _, h := range out.Copies {
		fmt.Fprintf(w, "  %-28s %-16s %s\n", termsafe.Sanitize(h.Key.String()), h.State, termsafe.Sanitize(h.Clone))
	}
}

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
			"yes to it. The run then hands off every pilot copy, each in its own clone of the\n" +
			"frozen commit made under a random alias in the clone store outside every\n" +
			"checkout; a clone bound for a provider passes the launch bundle's secret scan\n" +
			"first, and a hit refuses the cell. A host cell's copy is the host's to make in\n" +
			"its clone and hand back with `abcd lab matrix copy`; a runner cell's copy waits\n" +
			"for `abcd lab matrix launch`, which starts the cell's runner in its clone, and\n" +
			"nothing stands in for the runner.\n\n" +
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
			l, err := matrixLauncher(root)
			if err != nil {
				return fmt.Errorf("abcd lab matrix: %w", err)
			}
			rec, plan, err := matrix.StartRun(root, args[0], decl, approve, cores, exe, core.Version, l)
			if err != nil {
				return matrixFailure(err)
			}
			out := startedView(rec, plan, l, "the run's record")
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderStarted(w, "abcd lab matrix", out) })
		},
	}
	cmd.Flags().StringVar(&approve, "approve", "", "the digest of the plan the person saw and said yes to; records the run")
	cmd.AddCommand(newLabMatrixReportCommand(asJSON), newLabMatrixRerunCommand(asJSON), newLabMatrixCopyCommand(asJSON),
		newLabMatrixReviewCommand(asJSON), newLabMatrixJudgeCommand(asJSON), newLabMatrixLaunchCommand(asJSON))
	return cmd
}

// matrixFailure maps a refusal to exit 2 and a halted lab to exit 1, the
// message masked and with no home path. A refusal says nothing was written
// only when it came before any write; one raised after a write says what was
// written in its own words.
func matrixFailure(err error) error {
	msg := "abcd lab matrix: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	if errors.Is(err, lab.ErrHalted) {
		return &exitError{Code: 1, Msg: msg}
	}
	if errors.Is(err, matrix.ErrWrittenInPart) {
		return &exitError{Code: 2, Msg: msg}
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
	fmt.Fprintf(w, "  sandbox:     none: %s\n", notSandboxed)
	fmt.Fprintf(w, "  digest:      %s\n", v.Digest)
	fmt.Fprintf(w, "  to start it: %s\n", strings.TrimSpace(termsafe.Sanitize(v.ApproveWith)))
}
