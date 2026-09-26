package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/repolint"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newLintCommand wires the read-only `abcd lint` verb (renamed from `abcd
// audit` per adr-40/spc-29: it applies rules about form, so it is the lint;
// `/abcd:audit` returns to itd-16's reserved hash-chain fidelity surface): it evaluates the
// bundled conformance rules against the working directory and reports them, human
// text by default or machine JSON with --json. It never writes. The exit code is
// Conftest's tri-state — 0 clean, 1 warnings only, 2 any error — so `abcd lint`
// can gate a repo's CI.
func newLintCommand(asJSON *bool) *cobra.Command {
	var rootDir string
	cmd := &cobra.Command{
		Use:  "lint",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir := rootDir
			if dir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return &exitError{Code: 2, Msg: scrubPaths(err)}
				}
				dir = cwd
			}
			dir, err := filepath.Abs(dir)
			if err != nil {
				return &exitError{Code: 2, Msg: scrubPaths(err)}
			}
			// Validate the root before evaluating: the rules read ENOENT as
			// "artifact missing" and would emit confident, fabricated convention
			// violations against a directory that is not there (B41). Fail with a
			// usage error instead, matching the disembark probe guard.
			if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
				shown := rootDir
				if shown == "" {
					shown = dir
				}
				return &exitError{Code: 2, Msg: fmt.Sprintf("lint: %s is not a directory", shown)}
			}

			result, err := repolint.Evaluate(repolint.DefaultRules(), repolint.Context{RepoRoot: dir})
			if err != nil {
				// A rule that cannot run (a stat error, an unreadable tracked file, a
				// git-index fault) is a check that never happened, not a clean-ish
				// warnings-only pass: map it to the tri-state's "any error" code (2),
				// so a CI gate keying on >=2 fails closed instead of reading a
				// lint-that-never-ran as advisory. scrubPaths keeps the developer
				// identity path out of the rendered error.
				return &exitError{Code: 2, Msg: scrubPaths(err)}
			}

			if *asJSON {
				out, err := repolint.JSONSerializer{}.Serialize(result)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				renderLintHuman(cmd.OutOrStdout(), result)
			}

			// Tri-state exit: the output is already rendered, so a non-zero code
			// propagates with an empty message (main prints nothing more).
			if result.ExitCode != 0 {
				return &exitError{Code: result.ExitCode}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&rootDir, "root", "", "repo root to lint (default: current working directory)")
	// `lint outbound` judges one piece of TEXT the caller hands it rather than
	// this repository, and it lives here because adr-40's vocabulary puts a verb
	// that applies rules about form in the lint bucket. See lint_outbound.go for
	// why the outbound policy's session-URL half cannot be gated in shell.
	cmd.AddCommand(newLintOutboundCommand(asJSON))
	// The targets (itd-2609212130136102): each runs one check on its own, as its
	// old spelling did — `docs lint`, `site check`, bare `identity` — and bare
	// `lint` runs every one that judges the repository through its rules.
	cmd.AddCommand(newLintDocsCommand(asJSON))
	cmd.AddCommand(newLintSiteCommand(asJSON))
	cmd.AddCommand(newLintIdentityCommand(asJSON))
	return cmd
}

// severityGlyph returns the doctor-style marker for a severity.
func severityGlyph(sev repolint.Severity) string {
	switch sev {
	case repolint.SeverityError:
		return "✗"
	case repolint.SeverityWarn:
		return "⚠"
	default:
		return "•"
	}
}

// renderLintHuman writes the grouped, doctor-style report: a line per finding
// with a severity glyph, the rule id, the citation, the message, and an indented
// fix; skipped (not-applicable) rules and a summary tail. A clean repo gets a
// single green line.
func renderLintHuman(w io.Writer, res repolint.Result) {
	if len(res.Findings) == 0 {
		fmt.Fprintln(w, "abcd lint — ✓ conforms to the working conventions")
	}
	for _, f := range res.Findings {
		// File, Message and Fix are built from the linted repo's own file paths and
		// content (`abcd lint` runs over any repo), so they are untrusted terminal
		// output and pass through the canonical sanitiser; RuleID/Severity are enum
		// constants and need none.
		loc := termsafe.Sanitize(f.File)
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", termsafe.Sanitize(f.File), f.Line)
		}
		fmt.Fprintf(w, "%s [%s] %s — %s\n", severityGlyph(f.Severity), f.RuleID, loc, termsafe.Sanitize(f.Message))
		if f.Fix != "" {
			fmt.Fprintf(w, "    fix: %s\n", termsafe.Sanitize(f.Fix))
		}
	}
	for _, id := range res.Skipped {
		fmt.Fprintf(w, "• [%s] skipped (not applicable)\n", id)
	}
	fmt.Fprintf(w, "abcd lint — %d error(s), %d warning(s)\n", res.Blockers, res.Warnings)
}
