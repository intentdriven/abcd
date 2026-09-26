package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
// and committed assets, into a directory the repository does not track. The
// gate over what it rendered is `abcd lint site`.
func newSiteCommand(asJSON *bool) *cobra.Command {
	siteCmd := &cobra.Command{
		Use:  "site",
		Args: cobra.NoArgs,
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
		Use:  "build",
		Args: cobra.NoArgs,
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

	siteCmd.AddCommand(newSiteSetupCommand(asJSON))

	// The gate over the built site is `abcd lint site` (itd-2609212130136102);
	// `site check` answers with it for one release.
	siteCmd.AddCommand(movedStub("check", "abcd lint site"))

	return siteCmd
}

// newLintSiteCommand builds `lint site`: the gates adr-47 decision 3 arms, run
// over a built output directory, rendering it first when it holds no
// index.html. It exits 1 when any gate fails, so a release job can stop on it.
func newLintSiteCommand(asJSON *bool) *cobra.Command {
	var checkOut string
	checkCmd := &cobra.Command{
		Use:  "site",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := site.Check(site.CheckRequest{RepoRoot: cwd, OutDir: checkOut})
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd lint site: " + scrubPaths(err)}
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

	return checkCmd
}

// renderSiteCheck prints every failure, grouped by the gate that raised it, and
// the shrink invitations that are news rather than failures.
func renderSiteCheck(w io.Writer, res site.CheckResult) {
	fmt.Fprintf(w, "abcd lint site — %s\n", termsafe.Sanitize(res.OutDir))
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

// newSiteSetupCommand builds `site setup` (itd-2609061543533170): the verb that
// takes a managed repository's site from the checkout to a live address. It
// writes the repository half, and — each only once confirmed (adr-44) — the
// forge's deployment environments and, with a hosting credential on this
// machine, the host. An unanswered run declines both remote writes, so a script
// that pipes nothing changes nothing remote; --yes says yes in advance.
func newSiteSetupCommand(asJSON *bool) *cobra.Command {
	var name, domain string
	var confirm, yes bool
	cmd := &cobra.Command{
		Use:  "setup",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			var asker site.Asker = newPrompter(cmd)
			if yes {
				asker = alwaysConfirm{}
			}
			res, err := site.Setup(site.SetupRequest{
				RepoRoot: cwd, Name: name, Domain: domain, Confirm: confirm, Asker: asker,
				Context: cmd.Context(),
			})
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd site setup: " + scrubPaths(err)}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderSiteSetup(w, res)
			}); rerr != nil {
				return rerr
			}
			// A run that did not do what it set out to exits non-zero, the reason
			// on stdout above: a declined confirmation reads EOF in a script, and
			// exiting 0 there would be indistinguishable from a write that landed.
			if res.Status == site.StatusRefused || res.Status == site.StatusDeclined {
				return &exitError{Code: 1, Msg: "abcd site setup: " + res.Status + " — see the result above"}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "host name when the composition names none (default: the repository's name)")
	cmd.Flags().StringVar(&domain, "domain", "", "custom domain to route to the host when the composition names none")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "replace a workflow or host configuration that differs from what setup writes")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm the forge and host changes without being asked; without it an unanswered run declines them")
	return cmd
}

// renderSiteSetup prints the three stages and what remains.
func renderSiteSetup(w io.Writer, res site.SetupResult) {
	fmt.Fprintf(w, "abcd site setup — %s\n", termsafe.Sanitize(res.Status))
	// One status column for files and environments, as wide as the longest
	// status in this result (`unrestricted` is twelve characters), so every
	// name starts in one column and a change line sits under its name.
	width := 8
	for _, f := range res.Files {
		width = max(width, len(f.Status))
	}
	for _, e := range res.Environments {
		width = max(width, len(e.Status))
	}
	under := strings.Repeat(" ", 4+width+1)
	fmt.Fprintf(w, "  repository\n")
	for _, f := range res.Files {
		line := fmt.Sprintf("    %-*s %s", width, f.Status, f.Path)
		if f.Detail != "" && f.Status != "kept" {
			line += " (" + f.Detail + ")"
		}
		fmt.Fprintln(w, termsafe.Sanitize(line))
	}
	repo := res.Repo
	if repo == "" {
		repo = "no forge"
	}
	fmt.Fprintf(w, "  forge (%s)\n", termsafe.Sanitize(repo))
	for _, e := range res.Environments {
		fmt.Fprintf(w, "    %-*s %s\n", width, termsafe.Sanitize(e.Status), termsafe.Sanitize(e.Name))
		for _, c := range e.Changes {
			fmt.Fprintf(w, "%s%s\n", under, termsafe.Sanitize(c))
		}
	}
	h := res.Host
	fmt.Fprintf(w, "  host (%s: %s)\n", termsafe.Sanitize(h.Provider), termsafe.Sanitize(h.Name))
	fmt.Fprintf(w, "    %s\n", termsafe.Sanitize(h.Status))
	for _, c := range h.Changes {
		fmt.Fprintf(w, "             %s\n", termsafe.Sanitize(c))
	}
	if h.Address != "" {
		fmt.Fprintf(w, "    live at %s\n", termsafe.Sanitize(h.Address))
	}
	if h.Detail != "" {
		fmt.Fprintf(w, "    %s\n", termsafe.Sanitize(h.Detail))
	}
	for _, n := range res.Notes {
		fmt.Fprintf(w, "  note: %s\n", termsafe.Sanitize(n))
	}
	if len(res.Remaining) == 0 {
		fmt.Fprintf(w, "nothing remains for you to do\n")
		return
	}
	fmt.Fprintf(w, "remaining:\n")
	for i, r := range res.Remaining {
		fmt.Fprintf(w, "  %d. %s\n", i+1, termsafe.Sanitize(r))
	}
}
