package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newIntentPrepassCommand builds `abcd intent prepass` (itd-42,
// spc-2609211918551301): the pre-pass before the planning interview. The bare
// form assembles the draft's four inputs (the invariants, the principles, the
// sibling index and the draft) with the rules the host's judgement is held to,
// and writes nothing; `--findings-json <path>` validates the host's findings
// against the input as it stands and writes the planning brief under the local
// tier, the only file the pre-pass writes. Both are front doors onto
// internal/core/intent; every refusal exits 2 with nothing written.
func newIntentPrepassCommand(asJSON *bool) *cobra.Command {
	var findingsJSON string
	cmd := &cobra.Command{
		Use:  "prepass <itd-N> [--findings-json <path>]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := intentStoreRoot(cmd)
			if err != nil {
				return err
			}
			id := args[0]
			if !cmd.Flags().Changed("findings-json") {
				in, err := intent.AssemblePrepass(repoRoot, id)
				if err != nil {
					return prepassRefusal(err, "")
				}
				return render(cmd.OutOrStdout(), *asJSON, in, func(w io.Writer) {
					fmt.Fprintf(w, "abcd intent prepass — %s input assembled (input digest %s); nothing written\n", in.Intent, in.Digest)
					fmt.Fprintf(w, "  draft: %s\n", termsafe.Sanitize(in.DraftPath))
					fmt.Fprintf(w, "  read: %d invariants · %d principles · an index of %d other intents\n",
						len(in.Invariants), len(in.Principles), len(in.Index))
					for _, warn := range in.Warnings {
						fmt.Fprintf(w, "  warning: %s\n", termsafe.Sanitize(warn))
					}
					fmt.Fprintf(w, "  next: judge the input under its rules (`abcd intent prepass %s --json` carries them), then `abcd intent prepass %s --findings-json <path>`\n", in.Intent, in.Intent)
				})
			}
			raw, err := intent.ReadPrepassFindings(findingsJSON)
			if err != nil {
				return prepassRefusal(err, "")
			}
			res, err := intent.WritePrepassBrief(repoRoot, id, raw)
			if err != nil {
				return prepassRefusal(err, id)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent prepass — %s brief written: %s\n", res.Intent, res.BriefPath)
				fmt.Fprintf(w, "  questions: %d conflicts · %d overlaps · %d unanchored (%d demoted from a conflict or an overlap that could not be anchored)\n",
					res.Conflicts, res.Overlaps, res.Unanchored, res.Demoted)
				for _, warn := range res.Warnings {
					fmt.Fprintf(w, "  warning: %s\n", termsafe.Sanitize(warn))
				}
				fmt.Fprintln(w, "  next: the planning interview opens from the brief and asks each question in order")
			})
		},
	}
	cmd.Flags().StringVar(&findingsJSON, "findings-json", "", "path to the host's findings over the input; validates them and writes the planning brief")
	return cmd
}

// prepassRefusal is every refusal of the verb: exit 2, the core's reason with
// the home redacted, and that nothing was written. A refusal of the host's
// findings (id set) also names the contract they are held to.
func prepassRefusal(err error, id string) error {
	msg := "abcd intent prepass: " + strings.TrimPrefix(fsutil.RedactHome(err.Error()), "intent prepass: ")
	if id != "" {
		return &exitError{Code: 2, Msg: fmt.Sprintf("%s (nothing written; the findings are held to the rules `abcd intent prepass %s --json` prints)", msg, termsafe.CleanProseLine(id, 40))}
	}
	return &exitError{Code: 2, Msg: msg + " (nothing written)"}
}
