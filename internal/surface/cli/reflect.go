package cli

// reflect.go is the front door onto internal/core/reflect — the release
// retrospective (itd-24, spc-2609211751376504).
//
// The interview is host-run (commands/reflect.md): the host renders the seed,
// asks the four asked sections one question at a time, and hands the answers to
// `reflect write`. The binary owns the seed, the floor and the only write.
//
// Exit codes: 0 when the seed renders or the retrospective lands; 1 for a
// refusal the person answers (the release shipped nothing, a retrospective
// already exists, targeted intents are unshipped and unconfirmed, an answer is
// thin); 2 for a usage refusal (no such tag, not a release tag, an intent id, a
// malformed answers file) and for a structural fault.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/reflect"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// maxReflectAnswersBytes caps the answers file read; the core refuses anything
// over its own 1 MiB cap, so one byte more is enough to let it say so.
const maxReflectAnswersBytes = 1<<20 + 1

// reflectQuestion is one asked section's opening question, in order, carried in
// the seed view so the host asks exactly the questions the floor is built for.
type reflectQuestion struct {
	Section  reflect.Section `json:"section"`
	Heading  string          `json:"heading"`
	Question string          `json:"question"`
}

// reflectSeedView is the seed as the front door renders it: the core's seed and
// the four questions the interview asks.
type reflectSeedView struct {
	reflect.Seed
	Questions []reflectQuestion `json:"questions"`
}

// reflectRefusal is the --json form of a refusal the person answers, so the
// host can ask the follow-up or the confirmation rather than parse prose.
type reflectRefusal struct {
	Refused   string                   `json:"refused"`
	Message   string                   `json:"message"`
	Path      string                   `json:"path,omitempty"`
	Thin      []reflect.ThinAnswer     `json:"thin,omitempty"`
	Unshipped []reflect.TargetedIntent `json:"unshipped_targets,omitempty"`
}

func newReflectCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "reflect <release-tag>",
		Long: "Open the retrospective for a cut release: render the seed the interview opens from —\n" +
			"the intents the tag shipped, which of them carry audit notes, the intents targeted at\n" +
			"the release that have not shipped, the changelog section and the computed metrics —\n" +
			"and write nothing. `reflect write` writes the retrospective from the answers.\n\n" +
			"The release is the only grain: an intent id is refused, because per-intent\n" +
			"reflection is the intent audit's (`abcd intent audit <itd-N>`).",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return &exitError{Code: 2, Msg: "abcd reflect: one release tag is expected — abcd reflect <release-tag>"}
			}
			if len(args) == 1 {
				return reflectTagOperand(args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			root, err := reflectRoot()
			if err != nil {
				return err
			}
			seed, err := reflect.BuildSeed(root, args[0])
			if err != nil {
				return reflectRefuse(cmd.OutOrStdout(), *asJSON, err)
			}
			view := reflectSeedView{Seed: seed}
			for _, s := range reflect.AskedSections {
				view.Questions = append(view.Questions, reflectQuestion{Section: s, Heading: s.Heading(), Question: s.Question()})
			}
			return render(cmd.OutOrStdout(), *asJSON, view, func(w io.Writer) { renderReflectSeed(w, view) })
		},
	}

	var answersPath string
	var proceed bool
	writeCmd := &cobra.Command{
		Use: "write <release-tag> --answers <file>",
		Long: "Write the retrospective for a cut release from the interview's answers, a JSON\n" +
			"object with one {\"answer\", \"follow_up\"} entry per asked section (went_well,\n" +
			"could_improve, lessons, decisions). It refuses, writing nothing, while an answer is\n" +
			"under the floor and its follow-up is unanswered, while intents targeted at the release\n" +
			"are unshipped and --proceed was not given, and when the retrospective already exists.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &exitError{Code: 2, Msg: "abcd reflect write: one release tag is expected — abcd reflect write <release-tag> --answers <file>"}
			}
			return reflectTagOperand(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if answersPath == "" {
				return &exitError{Code: 2, Msg: "abcd reflect write: --answers <file> is required (nothing written)"}
			}
			data, err := fsutil.ReadGuarded(answersPath, maxReflectAnswersBytes)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd reflect write: reading the answers: " + scrubPaths(err) + " (nothing written)"}
			}
			answers, err := reflect.ParseAnswers(data)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd reflect write: " + termsafe.Sanitize(scrubPaths(err)) + " (nothing written)"}
			}
			root, err := reflectRoot()
			if err != nil {
				return err
			}
			res, err := reflect.Write(root, reflect.WriteRequest{
				Tag: args[0], Answers: answers, ProceedDespiteUnshipped: proceed, Now: time.Now(),
			})
			if err != nil {
				return reflectRefuse(cmd.OutOrStdout(), *asJSON, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "retrospective written — %s\n", res.Path)
				fmt.Fprintf(w, "  release: %s, %d intent(s)\n", res.Seed.Tag, len(res.Seed.Intents))
			})
		},
	}
	writeCmd.Flags().StringVar(&answersPath, "answers", "", "the interview's answers, a JSON file")
	writeCmd.Flags().BoolVar(&proceed, "proceed", false,
		"write although intents targeted at the release are unshipped (the person's confirmation)")
	cmd.AddCommand(writeCmd)
	return cmd
}

// reflectTagOperand refuses an intent id with the reason, before anything else
// reads the operand. Every other shape is judged by the core.
func reflectTagOperand(arg string) error {
	if strings.HasPrefix(arg, "itd-") {
		return &exitError{Code: 2, Msg: fmt.Sprintf(
			"abcd reflect: %s is an intent; reflect takes a release tag (such as v0.11.0). Per-intent reflection is the intent audit: abcd intent audit %s",
			termsafe.Sanitize(arg), termsafe.Sanitize(arg))}
	}
	return nil
}

// reflectRoot is the checkout whose record the seed reads and the retrospective
// lands in, resolved from the working directory.
func reflectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, "the retrospective store")
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd reflect: " + err.Error() + " (nothing written)"}
	}
	return root, nil
}

// reflectRefuse maps a core refusal to its exit and, under --json, to a
// structured refusal on stdout. A refusal the person answers exits 1; anything
// else is a usage refusal or a fault and exits 2.
func reflectRefuse(w io.Writer, asJSON bool, err error) error {
	msg := "abcd reflect: " + termsafe.Sanitize(scrubPaths(err))
	ref := reflectRefusal{Message: termsafe.Sanitize(scrubPaths(err))}
	var (
		nothing   *reflect.NothingShippedError
		exists    *reflect.ExistsError
		unshipped *reflect.UnshippedError
		thin      *reflect.ThinAnswersError
	)
	switch {
	case errors.As(err, &nothing):
		ref.Refused = "nothing_shipped"
	case errors.As(err, &exists):
		ref.Refused, ref.Path = "exists", exists.Path
	case errors.As(err, &unshipped):
		ref.Refused, ref.Unshipped = "unshipped_targets", unshipped.Intents
		msg += "\n  re-run with --proceed once the person confirms"
	case errors.As(err, &thin):
		ref.Refused, ref.Thin = "thin_answers", thin.Thin
		for _, t := range thin.Thin {
			msg += fmt.Sprintf("\n  %s: ask — %s", t.Heading, t.Question)
		}
	default:
		return &exitError{Code: 2, Msg: msg}
	}
	if asJSON {
		if rerr := render(w, true, ref, nil); rerr != nil {
			return rerr
		}
		return &exitError{Code: 1}
	}
	return &exitError{Code: 1, Msg: msg}
}

// renderReflectSeed is the person's view of the seed. Every record-derived
// value is sanitised: a title is text a record author wrote.
func renderReflectSeed(w io.Writer, v reflectSeedView) {
	s := v.Seed
	fmt.Fprintf(w, "retrospective seed for %s", s.Tag)
	if s.Metrics.PreviousTag != "" {
		fmt.Fprintf(w, " (previous release %s)", s.Metrics.PreviousTag)
	}
	fmt.Fprintf(w, " — writes %s\n\n", s.Output)
	fmt.Fprintf(w, "shipped: %d intent(s)\n", len(s.Intents))
	for _, it := range s.Intents {
		notes := "no audit notes"
		if it.Audited {
			notes = fmt.Sprintf("audited: MET %d · MET_WITH_CONCERNS %d · NOT_MET %d · INCONCLUSIVE %d",
				it.Rollup.Met, it.Rollup.MetWithConcerns, it.Rollup.NotMet, it.Rollup.Inconclusive)
		}
		impact := it.Impact
		if impact == "" {
			impact = "none declared"
		}
		fmt.Fprintf(w, "  %-8s %s [%s] — %s\n", termsafe.Sanitize(it.ID), termsafe.Sanitize(it.Title), termsafe.Sanitize(impact), notes)
	}
	if len(s.Unaudited) > 0 {
		fmt.Fprintf(w, "\nno audit notes on %d intent(s); offer the audit first, then continue either way:\n", len(s.Unaudited))
		for _, o := range s.Unaudited {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(o.Command))
		}
	}
	if len(s.Unshipped) > 0 {
		fmt.Fprintf(w, "\nWARNING: %d intent(s) targeted at %s have not shipped; ask before proceeding (reflect write --proceed):\n", len(s.Unshipped), s.Tag)
		for _, u := range s.Unshipped {
			fmt.Fprintf(w, "  %s  %s\n", termsafe.Sanitize(u.ID), termsafe.Sanitize(u.Path))
		}
	}
	if s.Changelog.Found {
		fmt.Fprintf(w, "\nchangelog: %s\n", termsafe.Sanitize(s.Changelog.Heading))
	} else {
		fmt.Fprintf(w, "\nchangelog: no dated section for %s\n", s.Tag)
	}
	m := s.Metrics
	fmt.Fprintf(w, "metrics (computed, not asked): %d shipped, %d audited, %d without audit notes; tagged %s\n",
		m.IntentsShipped, m.Audited, m.Unaudited, orDash(m.TagDate))
	fmt.Fprintln(w, "\nthe interview asks, one at a time:")
	for i, q := range v.Questions {
		fmt.Fprintf(w, "  %d. %s — %s\n", i+1, q.Heading, q.Question)
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
