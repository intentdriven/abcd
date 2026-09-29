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
// writes the changelog-driven release machinery into a managed repo that lacks
// it, wired to the repo's own default branch and pull-request CI check names,
// GITHUB_TOKEN-only and injection-safe. The file set follows the artefact kind
// the repository declares (itd-2609150819432059): a plugin receives release.yml,
// auto-release.yml, the adr-37 runbook and the reviews-charter check; any other
// kind receives the gate workflow abcd-release-gate.yml with a named empty build
// job, the runbook, the charter check, the empty [Unreleased] anchor when it has
// no changelog, and auto-release.yml unless its own release workflow — left
// byte-for-byte — stays in charge. --dependency-reauthor (or a declaration already
// present) adds the dependency-bump re-authoring workflow, its script and the
// seeded declaration (itd-2609221842494980). A repository that has declared no kind, or a
// kind abcd does not know, is refused before anything is written (exit 2).
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
	var confirm, reauthor bool
	cmd := &cobra.Command{
		Use:  "scaffold [--confirm] [--dependency-reauthor]",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rep, err := scaffold.Scaffold(scaffold.Request{RepoRoot: cwd, Confirm: confirm,
				DependencyReauthor: reauthor})
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
	cmd.Flags().BoolVar(&reauthor, "dependency-reauthor", false,
		"opt in to re-authoring bot-opened dependency bumps as the repository owner (seeds "+scaffold.ReauthorConfPath+")")
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
	// The toolchain is named only for a Go module; a repository with no go.mod
	// resolves none (the report leaves GoVersion empty).
	toolchain := ""
	if rep.GoVersion != "" {
		toolchain = ", go " + termsafe.Sanitize(rep.GoVersion)
	}
	fmt.Fprintf(w, "abcd launch scaffold — %s (kind %s, branch %s%s)\n",
		verdict, termsafe.Sanitize(string(rep.Kind)), termsafe.Sanitize(rep.DefaultBranch), toolchain)
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
	// The repository's own release workflow is left alone, so what to add to it
	// is printed rather than written: the stanza is abcd's own text.
	if rep.CallStanza != "" {
		fmt.Fprintln(w, "  to call the gate, add this to your own release workflow:")
		for _, line := range strings.Split(strings.TrimRight(rep.CallStanza, "\n"), "\n") {
			fmt.Fprintf(w, "    %s\n", line)
		}
	}
	if rep.DependencyReauthor {
		fmt.Fprintf(w, "  dependency re-authoring: opted in; the owner in %s and the Dependabot secrets\n", scaffold.ReauthorConfPath)
		fmt.Fprintln(w, "    DEPENDENCY_REAUTHOR_APP_ID and DEPENDENCY_REAUTHOR_APP_KEY are the person's to set;")
		fmt.Fprintln(w, "    until they are, an in-bound bump is refused and left for a person to land.")
	}
	if blocked {
		fmt.Fprintln(w, "  re-run with --confirm to overwrite the hand-edited file(s) with the machinery.")
	}
}
