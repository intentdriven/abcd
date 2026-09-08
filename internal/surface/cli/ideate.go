package cli

// ideate.go is the front door onto internal/core/ideate — the write step of the
// idea-admission protocol (itd-104, spc-18).
//
// The verb is deliberately SMALL. Everything the protocol is actually made of —
// the primary-source research, the record grill, the adversarial review — is host
// work orchestrated by `commands/ideate.md`; the binary's whole job is to
// validate what those legs produced and write the durable verdict. So this file
// reads a payload, calls one core function, and formats the outcome.
//
// Exit codes follow the launch-ship / disembark-synthesis shape: 0 when the record
// landed, 2 for every structural refusal (a payload fault, an unresolvable
// citation, an unusable slug, an existing record). There is no exit 1 here,
// because ideate has no "refused cut" middle state — either the verdict is
// recordable or it is not.

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/ideate"
	"github.com/spf13/cobra"
)

// newIdeateCommand builds the `ideate` sub-tree. The parent takes no arguments
// and prints help: unlike capture or intent, ideate has no read-only status to
// render — a gauntlet is a run, not a store.
func newIdeateCommand(asJSON *bool) *cobra.Command {
	ideateCmd := &cobra.Command{
		Use:   "ideate",
		Short: "Idea-admission protocol: record the verdict of the three-leg gauntlet",
		Long: "Record the verdict of abcd's idea-admission protocol — primary-source research, a grill\n" +
			"against the existing record, and an independent adversarial review.\n\n" +
			"The legs are host work; `/abcd:ideate` orchestrates them. This verb validates what they\n" +
			"produced and writes the durable verdict. Ideate is OPTIONAL and never a gate: no other\n" +
			"verb requires it, and skipping it is never warned about.",
		Args: cobra.NoArgs,
		RunE: helpRunE,
	}

	var verdictJSON string
	recordCmd := &cobra.Command{
		Use:   "record <idea-slug> --verdict-json <file|->",
		Short: "Validate a host-composed verdict and write the dated research record",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &exitError{Code: 2, Msg: "ideate record: <idea-slug> is required"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// The flag is REQUIRED, not optional-with-a-deterministic-fallback. The
			// disembark synthesis verbs can fall back because a lifeboat's own files
			// carry evidence; there is no evidence-only verdict an idea could have,
			// and inventing one would be the binary doing the judging.
			if verdictJSON == "" {
				return &exitError{Code: 2, Msg: "ideate record: --verdict-json <file|-> is required; " +
					"the three legs are host work, and the binary records only what they produced"}
			}
			raw, err := readIdeatePayload(cmd, verdictJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "ideate record: " + scrubPaths(err)}
			}
			// The clock is read HERE, at the front door, and passed in: the date lands
			// in a durable filename and a durable log line, so the core takes it as an
			// argument a test can pin rather than reading it from under a writer.
			res, err := ideate.Record(cwd, args[0], raw, time.Now())
			if err != nil {
				return &exitError{Code: 2, Msg: "ideate record: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, renderIdeateResult(res))
			})
		},
	}
	recordCmd.Flags().StringVar(&verdictJSON, "verdict-json", "",
		"path to the host-composed verdict JSON (or - for stdin)")

	ideateCmd.AddCommand(recordCmd)
	return ideateCmd
}

// readIdeatePayload reads the untrusted verdict document behind the same trust
// guards every other host-delegated ingest uses: a file must be regular,
// non-symlink, and under the cap the core enforces; "-" reads stdin bounded to the
// same ceiling and refused whole when over-cap (not truncated).
func readIdeatePayload(cmd *cobra.Command, spec string) ([]byte, error) {
	if spec == "-" {
		return readCappedStdin(cmd, ideate.MaxPayloadBytes)
	}
	return readGuardedOperand(spec, ideate.MaxPayloadBytes)
}

// renderIdeateResult is the human view of a recorded verdict. Every value in it is
// core-owned (a validated slug, a registered verdict, counts, and paths this
// binary built), so nothing here needs sanitising — the untrusted prose stayed in
// the record, where the core's redactor scanned it and the renderer escaped it.
func renderIdeateResult(res ideate.Result) string {
	out := fmt.Sprintf("ideate verdict recorded — %s\n", res.Slug)
	out += fmt.Sprintf("  verdict:   %s\n", res.Verdict)
	out += fmt.Sprintf("  record:    %s\n", res.Path)
	out += fmt.Sprintf("  pointer:   %s\n", res.DecisionPath)
	out += fmt.Sprintf("  claims:    %d (verified %d · falsified %d · unverifiable %d)\n",
		res.Claims.Total(), res.Claims.Verified, res.Claims.Falsified, res.Claims.Unverifiable)
	if len(res.CitedRecords) == 0 {
		out += "  grill:     no entry in the record covers, contradicts, or supersedes this\n"
	} else {
		out += fmt.Sprintf("  grill:     %d hit(s), all cited ids resolved: %v\n", res.GrillHits, res.CitedRecords)
	}
	out += fmt.Sprintf("  adversary: %d kill attempt(s)\n", res.KillAttempts)
	if res.RejectedAlternatives == 0 {
		out += "  rejected:  none, recorded explicitly\n"
	} else {
		out += fmt.Sprintf("  rejected:  %d alternative(s) recorded\n", res.RejectedAlternatives)
	}
	// Loud-staging: the record was rewritten before it was written, and a composer
	// that is not told cannot know it just pasted a credential into a durable
	// record. The count is core-owned; the spans themselves are never echoed.
	if res.Redactions > 0 {
		out += fmt.Sprintf("  redacted:  %d secret/PII span(s) rewritten out of the verdict text before it was recorded\n",
			res.Redactions)
	}
	if res.Graduates {
		out += "  next:      the idea may graduate to a draft intent — `abcd intent \"<text>\"`\n"
	} else {
		out += "  next:      the idea does not graduate; the record is why\n"
	}
	return out
}
