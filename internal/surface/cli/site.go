package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newSiteCommand builds the `site` verb family: the website as a rendered
// surface of this repository (adr-47).
//
// Bare `abcd site` is a STATUS BOARD and writes nothing — what the repo has
// declared (the composition manifest, the interface-string allowlist, the
// reference baseline) and what the last build left in the output directory.
// `site build` renders: the landing page and record.json from repository text
// and committed assets, into a directory the repository does not track.
func newSiteCommand(asJSON *bool) *cobra.Command {
	siteCmd := &cobra.Command{
		Use:   "site",
		Short: "The website rendered from this repository: what is declared, and what was built (read-only)",
		Args:  cobra.NoArgs,
	}

	var statusOut string
	siteCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		st, err := site.Describe(cwd, statusOut)
		if err != nil {
			return &exitError{Code: 2, Msg: "abcd site: " + scrubPaths(err)}
		}
		return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) {
			renderSiteStatus(w, st)
		})
	}
	siteCmd.Flags().StringVar(&statusOut, "out", site.DefaultOutDir, "output directory to report on")

	var buildOut string
	var version, commit, stampDate string
	var preview bool
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Render the site into the output directory (writes nothing outside it)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := site.Build(site.Request{
				RepoRoot: cwd,
				OutDir:   buildOut,
				Stamp:    site.BuildStamp{Version: version, Commit: commit, GeneratedAt: stampDate, Preview: preview},
			})
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd site build: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderSiteBuild(w, res)
			})
		},
	}
	buildCmd.Flags().StringVar(&buildOut, "out", site.DefaultOutDir, "directory to render into")
	buildCmd.Flags().StringVar(&version, "version", "", "version for the footer and the build stamp (default: the newest dated CHANGELOG heading)")
	buildCmd.Flags().StringVar(&commit, "commit", "", "commit for the footer and the build stamp (default: git HEAD)")
	buildCmd.Flags().StringVar(&stampDate, "date", "", "date for the build stamp (default: the newest release's date)")
	buildCmd.Flags().BoolVar(&preview, "preview", false, "stamp the build as unreleased at this commit, for a preview deployment of an untagged tree")
	buildCmd.MarkFlagsMutuallyExclusive("preview", "version")
	siteCmd.AddCommand(buildCmd)

	var checkOut string
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Gate the built site: provenance, hero drift, banned tokens, snippets, the reference ratchet, mobile and figure labels",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := site.Check(site.CheckRequest{RepoRoot: cwd, OutDir: checkOut})
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd site check: " + scrubPaths(err)}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderSiteCheck(w, res)
			}); rerr != nil {
				return rerr
			}
			if !res.OK() {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	checkCmd.Flags().StringVar(&checkOut, "out", site.DefaultOutDir, "built output directory to check (rendered first if absent)")
	siteCmd.AddCommand(checkCmd)

	return siteCmd
}

// renderSiteCheck prints every failure, grouped by the gate that raised it, and
// the shrink invitations that are news rather than failures.
func renderSiteCheck(w io.Writer, res site.CheckResult) {
	fmt.Fprintf(w, "abcd site check — %s\n", termsafe.Sanitize(res.OutDir))
	if res.Built {
		fmt.Fprintf(w, "  (rendered first: the output directory held no index.html)\n")
	}
	fmt.Fprintf(w, "  pages:   %d (%d composed surfaces)\n", len(res.Pages), len(res.Composed))
	for _, name := range res.Checks {
		n := 0
		for _, f := range res.Findings {
			if f.Check == name {
				n++
			}
		}
		if n == 0 {
			fmt.Fprintf(w, "  ok       %s\n", name)
			continue
		}
		fmt.Fprintf(w, "  FAIL     %s (%d)\n", name, n)
		for _, f := range res.Findings {
			if f.Check != name {
				continue
			}
			where := f.Where
			if f.Source != "" {
				where += " ← " + f.Source
			}
			fmt.Fprintf(w, "    %s: %s\n", termsafe.Sanitize(where), termsafe.Sanitize(f.Detail))
		}
	}
	for _, n := range res.Notes {
		fmt.Fprintf(w, "  note     %s: %s: %s\n", n.Check, termsafe.Sanitize(n.Where), termsafe.Sanitize(n.Detail))
	}
	if res.OK() {
		fmt.Fprintf(w, "every gate passes\n")
		return
	}
	fmt.Fprintf(w, "%d finding(s); the site is not publishable until each is fixed at its source\n", len(res.Findings))
}

// renderSiteStatus prints the read-only board.
func renderSiteStatus(w io.Writer, st site.Status) {
	fmt.Fprintf(w, "abcd site — %s\n", mark(st.Manifest, site.ManifestRelPath))
	if !st.Manifest {
		fmt.Fprintf(w, "  this repo declares no site composition; nothing to build\n")
		return
	}
	fmt.Fprintf(w, "  chapters:     %d\n", st.Chapters)
	fmt.Fprintf(w, "  issue ledger: %s\n", publishedWord(st.IssueLedge))
	fmt.Fprintf(w, "  ui strings:   %s\n", mark(st.UIStrings, termsafe.Sanitize(st.UIPath)))
	if st.Baseline {
		fmt.Fprintf(w, "  baseline:     %s (%d unresolved references admitted)\n",
			termsafe.Sanitize(st.BaselinePath), st.BaselineN)
	} else {
		fmt.Fprintf(w, "  baseline:     absent (%s)\n", termsafe.Sanitize(st.BaselinePath))
	}
	if st.Version != "" {
		fmt.Fprintf(w, "  release:      v%s\n", termsafe.Sanitize(st.Version))
	}
	if st.Commit != "" {
		fmt.Fprintf(w, "  commit:       %s\n", termsafe.Sanitize(st.Commit))
	}
	switch {
	case st.OutRefused != "":
		fmt.Fprintf(w, "  output:       %s (refused: %s)\n", termsafe.Sanitize(st.OutDir), termsafe.Sanitize(scrubPaths(errors.New(st.OutRefused))))
	case st.OutExists:
		fmt.Fprintf(w, "  output:       %s (%d entries)\n", termsafe.Sanitize(st.OutDir), st.OutFiles)
	default:
		fmt.Fprintf(w, "  output:       %s (not built)\n", termsafe.Sanitize(st.OutDir))
	}
	fmt.Fprintf(w, "run `abcd site build` to render\n")
}

// renderSiteBuild prints what a build wrote and what it measured.
func renderSiteBuild(w io.Writer, res site.Result) {
	fmt.Fprintf(w, "abcd site build — %s\n", termsafe.Sanitize(res.OutDir))
	// The explorer writes a page per record, so a full listing is hundreds of
	// lines of scroll. The head of it still says what shape the tree took, and
	// the count says how much of it is not shown.
	const listed = 12
	for i, f := range res.Files {
		if i == listed && len(res.Files) > listed+1 {
			fmt.Fprintf(w, "  … %d more\n", len(res.Files)-listed)
			break
		}
		fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(f))
	}
	fmt.Fprintf(w, "  pages:   %d rendered from the record\n", res.Pages)
	fmt.Fprintf(w, "  record:  %d records · %d links · %d mentions\n", res.Records, res.Links, res.Mentions)
	fmt.Fprintf(w, "  refs:    %d unresolved (baseline %d)\n", res.Unresolved, res.Baseline)
	fmt.Fprintf(w, "  layout:  %d overlapping bubbles across both arrangements\n", res.Overlaps)
	stamp := res.Version
	if stamp != "" {
		stamp = "v" + stamp
	}
	if res.Commit != "" {
		if stamp != "" {
			stamp += " · "
		}
		stamp += res.Commit
	}
	if stamp != "" {
		fmt.Fprintf(w, "  built:   %s\n", termsafe.Sanitize(stamp))
	}
	fmt.Fprintf(w, "wrote %d files (%d bytes)\n", len(res.Files), res.Bytes)
}

// mark renders a present/absent line for one declared input.
func mark(present bool, label string) string {
	if present {
		return label
	}
	return label + " (absent)"
}

// publishedWord says whether the working-tier issue ledger is opted in.
func publishedWord(on bool) string {
	if on {
		return "published"
	}
	return "not published"
}
