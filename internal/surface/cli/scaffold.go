package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newLaunchScaffoldCommand builds `abcd launch scaffold` (itd-93, spc-14): it
// writes the changelog-driven release machinery — release.yml, auto-release.yml,
// the adr-37 runbook and the reviews-charter check — into a managed repo that
// lacks it, wired to the repo's own default branch and pull-request CI check
// names, GITHUB_TOKEN-only and injection-safe.
//
// It is idempotent and fail-safe (AC4): a re-run on current machinery is a no-op,
// and a hand-edited file is refused (exit 1) rather than clobbered unless
// --confirm is passed. Exit codes are the machine seam an operator gates on:
//
//   - 0 — every file written or already current (a clean scaffold or a no-op).
//   - 1 — a file exists and differs; the report names it and nothing was written.
//     Re-run with --confirm to overwrite (the rendered report is the output).
//   - 2 — a structural fault (the repository or a template could not be read).
func newLaunchScaffoldCommand(asJSON *bool) *cobra.Command {
	var confirm bool
	cmd := &cobra.Command{
		Use:  "scaffold [--confirm]",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rep, err := scaffold.Scaffold(scaffold.Request{RepoRoot: cwd, Confirm: confirm})
			if err != nil && !errors.Is(err, scaffold.ErrScaffoldBlocked) {
				// A structural fault (unreadable repo/template, failed write): exit 2,
				// scrubbed to one line so no absolute path leaks (iss-81). A mid-write
				// fault still carries a partial per-file report (which files landed
				// before the fault, which were skipped) — render it first so the
				// operator sees the disk state; a pure preflight fault has none.
				if len(rep.Files) > 0 {
					_ = render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
						renderScaffold(w, rep, false)
					})
				}
				return &exitError{Code: 2, Msg: "abcd launch scaffold: " + scrubPaths(err)}
			}
			blocked := errors.Is(err, scaffold.ErrScaffoldBlocked)
			if rerr := render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				renderScaffold(w, rep, blocked)
			}); rerr != nil {
				return rerr
			}
			if blocked {
				// A refusal is an expected outcome, not a crash: the report is the
				// output and exit 1 the only extra signal (the embark-conflicts
				// precedent).
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&confirm, "confirm", false, "overwrite a hand-edited scaffolded file with the current machinery")
	return cmd
}

// renderScaffold prints the human summary of a scaffold run: the derived repo
// facts, each file's disposition, and the overall verdict.
func renderScaffold(w io.Writer, rep scaffold.Report, blocked bool) {
	verdict := "scaffolded"
	switch {
	case blocked:
		verdict = "REFUSED"
	case rep.NoOp:
		verdict = "no-op (already current)"
	}
	fmt.Fprintf(w, "abcd launch scaffold — %s (branch %s, go %s)\n",
		verdict, termsafe.Sanitize(rep.DefaultBranch), termsafe.Sanitize(rep.GoVersion))
	if len(rep.CIChecks) > 0 {
		fmt.Fprintf(w, "  merge gate: %s (require these on %s)\n",
			termsafe.Sanitize(strings.Join(rep.CIChecks, ", ")), termsafe.Sanitize(rep.DefaultBranch))
	} else {
		fmt.Fprintln(w, "  merge gate: no pull-request CI workflow found (the runbook says how to add one)")
	}
	for _, f := range rep.Files {
		// f.Path is a fixed repo-relative constant; f.Detail interpolates a reason.
		fmt.Fprintf(w, "  [%s] %s", f.Status, f.Path)
		if f.Detail != "" {
			fmt.Fprintf(w, " — %s", termsafe.Sanitize(f.Detail))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "  %d written, %d refused\n", rep.Wrote, rep.Refused)
	if blocked {
		fmt.Fprintln(w, "  re-run with --confirm to overwrite the hand-edited file(s) with the machinery.")
	}
}
