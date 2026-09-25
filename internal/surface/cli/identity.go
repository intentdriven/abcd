package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newIdentityCommand wires the `abcd identity` family: the repo's canonical
// self-description and the surfaces held to it.
//
//   - bare — read-only status: the block, and every registered surface's verdict.
//   - render — the proposed correction for each drifted surface, as a unified
//     diff on stdout. It writes nothing, and no flag makes it: a deliberate
//     identity change is an edit to the block, after which the same proposal
//     flow chases the surfaces.
//   - init — the write path: scaffold the block and the pointer that records
//     where it lives. It adopts an existing block rather than re-interviewing.
//
// The drift check itself runs on every `abcd lint`; this family is where a
// maintainer looks at the canon and at what a fix would look like.
func newIdentityCommand(asJSON *bool) *cobra.Command {
	identityCmd := &cobra.Command{
		Use:  "identity",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadPositioning()
			if err != nil {
				return err
			}
			rep, err := positioning.Check(root, cfg)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd identity: " + scrubPaths(err)}
			}
			// A status render, not a gate: drift here exits 0 and says so. The
			// gate is `abcd lint`, which carries the tri-state exit code.
			return render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				renderIdentity(w, rep)
			})
		},
	}

	identityCmd.AddCommand(&cobra.Command{
		Use:  "render",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadPositioning()
			if err != nil {
				return err
			}
			prop, err := positioning.Propose(root, cfg)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd identity render: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, prop, func(w io.Writer) {
				renderProposal(w, prop)
			})
		},
	})

	var title, tagline, pitch, file, heading string
	initCmd := &cobra.Command{
		Use:  "init",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := positioning.Init(cwd, positioning.InitRequest{
				Title: title, Tagline: tagline, Pitch: pitch,
				Location: positioning.BlockLocation{File: file, Heading: heading},
			})
			if err != nil {
				// A fault, not a rendered refusal: init prints nothing on this
				// path, so it takes the tree-wide "could not be evaluated" code 2
				// (matching `identity`/`identity render` and loadPositioning), not
				// the code-1 an already-rendered refusal reserves.
				return &exitError{Code: 2, Msg: "abcd identity init: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderIdentityInit(w, res)
			})
		},
	}
	initCmd.Flags().StringVar(&title, "title", "", "the project's title (required unless a block already exists)")
	initCmd.Flags().StringVar(&tagline, "tagline", "", "the project's one-line tagline (required unless a block already exists)")
	initCmd.Flags().StringVar(&pitch, "pitch", "", "the project's short elevator pitch (optional)")
	initCmd.Flags().StringVar(&file, "file", "", "repo-relative file the identity block lives in (default "+positioning.DefaultBlockLocation.File+")")
	initCmd.Flags().StringVar(&heading, "heading", "", "heading the identity block sits under (default \""+positioning.DefaultBlockLocation.Heading+"\")")
	identityCmd.AddCommand(initCmd)

	return identityCmd
}

// loadPositioning resolves the working directory's registry. A repo that has not
// adopted the check is told how to, rather than shown an empty report that reads
// like a clean bill of health.
func loadPositioning() (string, positioning.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", positioning.Config{}, err
	}
	cfg, ok, err := positioning.LoadConfig(cwd)
	if err != nil {
		return "", positioning.Config{}, &exitError{Code: 2, Msg: "abcd identity: " + scrubPaths(err)}
	}
	if !ok {
		return "", positioning.Config{}, &exitError{Code: 2, Msg: "abcd identity: this repo records no identity block (" +
			positioning.ConfigRelPath + " is absent) — run `abcd identity init` to record one"}
	}
	return cwd, cfg, nil
}

// identityGlyph marks a surface's verdict in the human render.
func identityGlyph(s positioning.Status) string {
	switch s {
	case positioning.StatusOK:
		return "✓"
	case positioning.StatusDrifted, positioning.StatusUnlocatable:
		return "⚠"
	default:
		return "•"
	}
}

// renderIdentity prints the canonical block and one line per surface. The block
// and every surface string is repo content, so it passes through the canonical
// terminal sanitiser.
func renderIdentity(w io.Writer, rep positioning.Report) {
	fmt.Fprintf(w, "abcd identity — %s (%s)\n", termsafe.Sanitize(rep.Block.File), rep.Severity)
	fmt.Fprintf(w, "  title:   %s\n", termsafe.Sanitize(rep.Block.Title))
	fmt.Fprintf(w, "  tagline: %s\n", termsafe.Sanitize(rep.Block.Tagline))
	if rep.Block.Pitch != "" {
		fmt.Fprintf(w, "  pitch:   %s\n", termsafe.Sanitize(rep.Block.Pitch))
	}
	for _, s := range rep.Surfaces {
		loc := termsafe.Sanitize(s.File)
		if loc == "" {
			loc = "(absent)"
		} else if s.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, s.Line)
		}
		fmt.Fprintf(w, "%s [%s] %s — %s\n", identityGlyph(s.Status), termsafe.Sanitize(s.ID), loc, s.Status)
		if s.Status == positioning.StatusDrifted {
			fmt.Fprintf(w, "    says:      %s\n", termsafe.Sanitize(s.Found))
			fmt.Fprintf(w, "    canonical: %s\n", termsafe.Sanitize(s.Canonical))
		}
	}
	fmt.Fprintf(w, "abcd identity — %d surface(s) adrift of the block\n", rep.Drifted())
}

// renderProposal prints the unified diffs. The header states plainly that this
// is a proposal: adopting it is the human's move, and abcd never makes it.
func renderProposal(w io.Writer, p positioning.Proposal) {
	if len(p.Diffs) == 0 {
		fmt.Fprintf(w, "abcd identity render — nothing to propose (%d surface(s) adrift, none with a replaceable region)\n", p.Report.Drifted())
		return
	}
	fmt.Fprintf(w, "abcd identity render — %d proposed change(s), nothing written. Adopt what you agree with.\n\n", len(p.Diffs))
	for _, d := range p.Diffs {
		// The diff body is the repo's own file content, so it is sanitised
		// before it reaches a terminal — per line, because a diff IS lines and
		// masking its newlines would smear it into one unreadable string.
		fmt.Fprint(w, termsafe.SanitizeBlock(d.Unified))
		fmt.Fprintln(w)
	}
}

// renderIdentityInit prints what onboarding did.
func renderIdentityInit(w io.Writer, res positioning.InitResult) {
	switch {
	case res.NoOp:
		fmt.Fprintf(w, "abcd identity init — already recorded at %s (%q); nothing written\n",
			termsafe.Sanitize(res.Location.File), termsafe.Sanitize(res.Location.Heading))
	case res.AdoptedExisting:
		fmt.Fprintf(w, "abcd identity init — adopted the existing block at %s (%q)\n",
			termsafe.Sanitize(res.Location.File), termsafe.Sanitize(res.Location.Heading))
	default:
		fmt.Fprintf(w, "abcd identity init — recorded the identity block at %s (%q)\n",
			termsafe.Sanitize(res.Location.File), termsafe.Sanitize(res.Location.Heading))
	}
	fmt.Fprintf(w, "  title:   %s\n", termsafe.Sanitize(res.Block.Title))
	fmt.Fprintf(w, "  tagline: %s\n", termsafe.Sanitize(res.Block.Tagline))
	if res.Block.Pitch != "" {
		fmt.Fprintf(w, "  pitch:   %s\n", termsafe.Sanitize(res.Block.Pitch))
	}
	for _, p := range res.Wrote {
		fmt.Fprintf(w, "  wrote:   %s\n", p)
	}
	fmt.Fprintln(w, "  `abcd lint` now holds every registered surface to this block.")
}
