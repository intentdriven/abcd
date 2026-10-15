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
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ideate"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newIdeateCommand builds the `ideate` sub-tree. The parent takes no arguments
// and prints help: unlike capture or intent, ideate has no read-only status to
// render — a gauntlet is a run, not a store.
func newIdeateCommand(asJSON *bool) *cobra.Command {
	ideateCmd := &cobra.Command{
		Use: "ideate",
		Long: "Record the verdict of abcd's idea-admission protocol — primary-source research, a grill\n" +
			"against the existing record, and an independent adversarial review.\n\n" +
			"The legs are host work; `/abcd:ideate` orchestrates them. This verb validates what they\n" +
			"produced and writes the durable verdict. Ideate is OPTIONAL and never a gate: no other\n" +
			"verb requires it, and skipping it is never warned about.",
		Args: cobra.NoArgs,
		RunE: helpRunE,
	}

	var verdictJSON string
	var private, public bool
	recordCmd := &cobra.Command{
		Use: "record <idea-slug> --private|--public --verdict-json <file|->",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &exitError{Code: 2, Msg: "ideate record: <idea-slug> is required"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := ideateStoreRoot(cmd)
			if err != nil {
				return err
			}
			store, err := ideateStore(repoRoot, private, public)
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
			res, err := ideate.Record(repoRoot, args[0], store, raw, time.Now())
			if err != nil {
				return &exitError{Code: 2, Msg: "ideate record: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, renderIdeateResult(res))
			})
		},
	}
	recordCmd.Flags().BoolVar(&private, "private", false,
		"the idea is private: record the verdict in "+abcdhome.Display("ideas")+"/<root-sha>/, owner-only, with no decision pointing at it (exclusive with --public)")
	recordCmd.Flags().BoolVar(&public, "public", false,
		"the idea is public: commit the verdict under "+ideate.ResearchRelDir+"/<root-sha>/ and record a decision pointing at it (exclusive with --private)")
	recordCmd.Flags().StringVar(&verdictJSON, "verdict-json", "",
		"path to the host-composed verdict JSON (or - for stdin)")

	ideateCmd.AddCommand(recordCmd)
	return ideateCmd
}

// ideateStoreRoot is the front door's first step: the checkout whose research
// store the verdict is written into and whose decisions folder its pointer is
// recorded in, resolved from the working directory rather than taken to BE it.
//
// The verb used to hand os.Getwd() straight to ideate.Record, which joins both
// relative paths onto whatever it is given. Run from a subdirectory the grill
// then read a record that was not there and reported the cited ids as records
// that "do not exist in this repository" — a plausible wrong answer that blames
// the operator's grill for the verb's own misaddressing — and where the caller's
// directory happened to carry a decisions store, the verdict and its pointer landed
// in a store below the checkout root. Outside every repository it did the same
// in a plain directory (iss-2609091729516940). A verdict filed that way reaches
// no gate, no release cut and no reader, which for this family is the whole
// point of writing it: a killed idea nobody can find is an idea that gets
// proposed again.
//
// gitutil.CheckoutRoot owns the resolution and both refusals — the same one the
// capture verbs resolve their ledger through and `decide` its decision store,
// with only the store's noun differing. Nothing is written on a refusal, because
// the core is never reached.
//
// The stray-store note rides the same step, on stderr, exactly as the ledger's
// and the decision store's do: a resolution that silently steps over a store the
// defect already laid would leave those verdicts where nothing will ever look
// again. It REPORTS and moves nothing.
func ideateStoreRoot(cmd *cobra.Command) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, "the research store")
	if err != nil {
		return "", &exitError{Code: 2, Msg: "ideate record: " + err.Error() + " (nothing written)"}
	}
	for _, note := range strayStoreNotes(cwd, root, ideate.ResearchRelDir, "research store") {
		fmt.Fprintf(cmd.ErrOrStderr(), "ideate record: %s\n", termsafe.Sanitize(note))
	}
	return root, nil
}

// ideateStore turns the person's answer into the store the verdict is written
// to (iss-2610090740139804). The answer is required and has no default: the
// host asks "is this idea private or public?" before it records, and the flag
// carries the answer, so neither flag, or both, refuses before anything is read
// or written. The store is keyed on the repository's root commit, so a
// repository with no commit yet is refused here, naming why; HOME is resolved
// only for a private verdict, and the core judges it.
func ideateStore(repoRoot string, private, public bool) (ideate.Store, error) {
	var vis ideate.Visibility
	switch {
	case private && public:
		return ideate.Store{}, &exitError{Code: 2, Msg: "ideate record: --private and --public are exclusive; " +
			"pass the one answer the person gave (nothing written)"}
	case private:
		vis = ideate.VisibilityPrivate
	case public:
		vis = ideate.VisibilityPublic
	default:
		return ideate.Store{}, &exitError{Code: 2, Msg: "ideate record: one of --private or --public is required; " +
			"ask the person whether the idea is private or public, then record it with that answer (nothing written)"}
	}
	// A shallow clone's `rev-list --max-parents=0 HEAD` names its graft, not the
	// repository's root, so a verdict keyed on it would land in a folder no full
	// clone of the same repository ever reads. The probe is fail-closed: a git
	// that cannot say whether the checkout is shallow refuses too.
	if err := gitutil.RequireFullHistory(repoRoot); err != nil {
		if errors.Is(err, gitutil.ErrShallowCheckout) {
			return ideate.Store{}, &exitError{Code: 2, Msg: "ideate record: this checkout is shallow, and a verdict is filed under " +
				"the repository's root commit, which a shallow clone does not hold; fetch the full history (git fetch --unshallow), " +
				"then record (nothing written)"}
		}
		return ideate.Store{}, &exitError{Code: 2, Msg: "ideate record: whether this checkout is shallow could not be read (" +
			scrubPaths(err) + "), and a verdict is filed under the repository's root commit, which a shallow clone does not hold " +
			"(nothing written)"}
	}
	sha := gitutil.RootCommit(repoRoot)
	if sha == "" {
		return ideate.Store{}, &exitError{Code: 2, Msg: "ideate record: this repository has no commit yet, " +
			"and a verdict is filed under the repository's root commit; commit once, then record (nothing written)"}
	}
	store := ideate.Store{Visibility: vis, RootSHA: sha}
	if vis == ideate.VisibilityPrivate {
		// The error is not checked here: an unresolved home reaches the core as
		// "", and the core's refusal names it in the same words for every caller.
		store.Home, _ = os.UserHomeDir()
	}
	return store, nil
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
	out += fmt.Sprintf("  stored:    %s\n", res.Visibility)
	out += fmt.Sprintf("  record:    %s\n", res.Path)
	if res.DecisionPath == "" {
		out += "  pointer:   none — a private verdict records no decision\n"
	} else {
		out += fmt.Sprintf("  pointer:   %s\n", res.DecisionPath)
	}
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
	// One line per verdict: a reframed idea survives in another shape, so its
	// line must not read as the killed one's (iss-2609100508573400).
	switch {
	case res.Graduates:
		out += "  next:      the idea may graduate to a draft intent — `abcd intent \"<text>\"`\n"
	case res.Verdict == ideate.VerdictReframed:
		out += "  next:      the idea as posed does not graduate, but its reframing may — " +
			"`abcd intent \"<text>\"` with the reframing the record carries, not the original wording\n"
	default:
		out += "  next:      the idea does not graduate; the record is why\n"
	}
	return out
}
