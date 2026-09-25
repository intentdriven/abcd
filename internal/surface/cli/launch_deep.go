package cli

// launch_deep.go — the front door's half of itd-66's parity diff and deep
// installability smoke: the baseline a launch measures against, the loud
// release-asset fetcher it reads that baseline through on an explicit ask, and
// the isolated subprocess the deep tier renders every page in.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newReleaseAssetFetcher builds the fetcher a --fetch-baseline run reads the
// previous release's assets through, and names the proxy and CA variables its
// client does not honour. It is a package var so a test can serve the assets
// from memory: no test reaches the network.
var newReleaseAssetFetcher = func(origin string) (launch.ReleaseAssetFetcher, []string, error) {
	a, err := update.NewReleaseAssets(origin)
	if err != nil {
		return nil, nil, err
	}
	return a, a.EnvIgnored(), nil
}

// loudFetcher announces every fetch on stderr before it is made and says what
// came back, so a network read is never silent (adr-38 tier 2: the output names
// its source).
type loudFetcher struct {
	inner  launch.ReleaseAssetFetcher
	w      io.Writer
	origin string
}

func (l loudFetcher) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	fmt.Fprintf(l.w, "abcd launch: fetching %s of release %s from %s\n",
		termsafe.Sanitize(name), termsafe.Sanitize(tag), termsafe.Sanitize(l.origin))
	data, url, found, err := l.inner.FetchReleaseAsset(tag, name)
	switch {
	case err != nil:
		fmt.Fprintf(l.w, "abcd launch:   %s failed: %s\n", termsafe.Sanitize(url), termsafe.Sanitize(err.Error()))
	case !found:
		fmt.Fprintf(l.w, "abcd launch:   %s is not published\n", termsafe.Sanitize(url))
	default:
		fmt.Fprintf(l.w, "abcd launch:   fetched %d bytes from %s\n", len(data), termsafe.Sanitize(url))
	}
	return data, url, found, err
}

// launchParityInput resolves what a launch's parity diff measures against: the
// operator's configured baseline, validated, or else the newest release tag —
// the anchor the cut derives from. With fetch, the baseline is read from the
// tag's release asset first, through the loud fetcher pinned to the plugin's
// own repository.
func launchParityInput(cwd, configured string, fetch bool, stderr io.Writer) (*launch.ParityInput, error) {
	in := &launch.ParityInput{}
	if configured != "" {
		if err := launch.ValidateBaselineTag(cwd, configured); err != nil {
			return nil, err
		}
		in.Baseline = configured
	} else {
		tag, found, err := changelog.LatestReleaseTag(cwd)
		if err != nil {
			in.BaselineError = "the release tags could not be listed: " + scrubPaths(err)
		} else {
			in.Baseline, in.Unanchored, in.BaselineError = unanchoredBaseline(cwd, tag, found)
		}
	}
	if fetch {
		origin, err := launch.ReleaseRepository(cwd)
		if err != nil {
			return nil, err
		}
		f, ignored, err := newReleaseAssetFetcher(origin)
		if err != nil {
			return nil, err
		}
		in.Fetch = loudFetcher{inner: f, w: stderr, origin: origin}
		in.EnvIgnored = ignored
	}
	return in, nil
}

// unanchoredBaseline names the baseline when no operator named one. The newest
// release tag is the baseline in a full checkout whose CHANGELOG.md dates no
// newer release. A checkout whose tags cannot be trusted to name the previous
// release is unanchored: a shallow clone, whose listing holds only the tags it
// fetched; a clone with no release tag at all while CHANGELOG.md dates a
// release; and a full clone whose newest tag is older than the release
// CHANGELOG.md dates newest — a fork or mirror whose tags stopped at an older
// release. Its baseline is the newest release either source names, and why is
// said, so the diff refuses rather than reading as a first launch
// (iss-2609251902439938) or measuring against an older release
// (iss-2609252001486609). The last shape is also the window between a cut and
// its tag, where the dated release is the one just cut and the tag is the
// right baseline: the reason says so and names --baseline, the explicit
// escape, and a cut is not blocked by it, because the cut diffs before it
// writes its heading. A tree whose CHANGELOG.md dates no release and that
// holds no tag is a first launch.
func unanchoredBaseline(cwd string, tag launch.Semver, found bool) (baseline, unanchored, failure string) {
	shallow, err := launch.ShallowCheckout(cwd)
	if err != nil {
		return "", "", "whether the checkout is shallow could not be read: " + scrubPaths(err)
	}
	dated, _, err := changelog.DatedReleases(cwd)
	if err != nil {
		return "", "", "CHANGELOG.md could not be read to check the release tags against the releases it dates: " + scrubPaths(err)
	}
	newest, datedOK := launch.Semver{}, false
	if len(dated) > 0 {
		newest, err = launch.ParseSemver(strings.TrimPrefix(dated[0].Version, "v"))
		datedOK = err == nil
	}
	switch {
	case shallow && (found || datedOK):
		if found && (!datedOK || launch.CoreGreater(tag, newest)) {
			newest = tag
		}
		return newest.Tag(), "the checkout is shallow, so its tag listing may hold only the tags that were fetched", ""
	case found && datedOK && launch.CoreGreater(newest, tag):
		return newest.Tag(), fmt.Sprintf("CHANGELOG.md dates release %s, and the newest release tag in this checkout is %s: "+
			"either the %s tag is missing here (a fork or mirror whose tags stop at an older release), "+
			"or %s is cut and not tagged yet, when --baseline %s measures against the release before it",
			newest, tag.Tag(), newest.Tag(), newest, tag.Tag()), ""
	case found:
		return tag.Tag(), "", ""
	case datedOK:
		return newest.Tag(), "CHANGELOG.md dates release " + newest.String() + ", and this checkout holds no release tag, so this is not a first launch", ""
	case len(dated) > 0:
		return "", "", "CHANGELOG.md dates release " + termsafe.Sanitize(dated[0].Version) + ", which is not a release version, and this checkout holds no release tag"
	}
	return "", "", ""
}

// smokePagesTimeout bounds one deep-tier subprocess.
const smokePagesTimeout = 2 * time.Minute

// maxSmokePagesOutput bounds what the parent reads back from the child.
const maxSmokePagesOutput = 16 << 20

// pageRunnerExtraEnv is appended to the child's minimal environment. It is
// empty in the shipped binary; the package's tests set it so the test binary
// can stand in for abcd as the child.
var pageRunnerExtraEnv []string

// smokePagesOutput is what `abcd launch smoke-pages` writes: the directory it
// ran in, so the parent can prove the child was rooted where it was told, and
// one answer per page.
type smokePagesOutput struct {
	Root  string            `json:"root"`
	Pages []launch.PageHelp `json:"pages"`
}

// subprocessPageRunner is the deep tier's isolated runner: the running abcd
// binary, re-executed as `abcd launch smoke-pages` with its working directory,
// HOME and TMPDIR inside the rendered tree and a minimal environment, so
// anything rendering a page could write lands in the throwaway tree. The page
// list goes in on stdin and the answers come back on stdout; a child that ran
// anywhere but root fails the runner.
func subprocessPageRunner(root string, pages []launch.PageRef) ([]launch.PageHelp, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("the abcd binary could not be located: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errors.New("the rendered payload is not a directory")
	}
	input, err := json.Marshal(pages)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), smokePagesTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "launch", "smoke-pages")
	cmd.Dir = root
	cmd.Env = append([]string{"HOME=" + root, "TMPDIR=" + root, "LC_ALL=C", "PATH=/usr/bin:/bin"}, pageRunnerExtraEnv...)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("the page runner failed (%v): %s", err, strings.TrimSpace(stderrHead(stderr.String())))
	}
	if stdout.Len() > maxSmokePagesOutput {
		return nil, errors.New("the page runner wrote more than it was asked for")
	}
	var out smokePagesOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("the page runner's answer is not readable: %w", err)
	}
	if !sameDir(out.Root, root) {
		return nil, errors.New("the page runner did not run rooted at the rendered payload")
	}
	return out.Pages, nil
}

// sameDir reports whether two paths name one directory once symlinks resolve
// (a temporary directory on macOS is reached through /var and /private/var).
func sameDir(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

// stderrHead is the first line of a child's stderr, what a failure quotes.
func stderrHead(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// newLaunchSmokePagesCommand builds `abcd launch smoke-pages`, the deep tier's
// child. It is operator-internal and hidden: it reads a JSON page list on
// stdin, renders each page's help from the working directory, and writes the
// answers as JSON. It writes nothing to disk.
func newLaunchSmokePagesCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "smoke-pages",
		Short:  "Render declared pages' help for the deep installability smoke (operator-internal; JSON on stdin and stdout)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch smoke-pages: " + scrubPaths(err)}
			}
			var pages []launch.PageRef
			if err := json.NewDecoder(io.LimitReader(cmd.InOrStdin(), maxSmokePagesOutput)).Decode(&pages); err != nil {
				return &exitError{Code: 2, Msg: "abcd launch smoke-pages: the page list on stdin is not readable: " + scrubPaths(err)}
			}
			out := smokePagesOutput{Root: cwd, Pages: make([]launch.PageHelp, 0, len(pages))}
			for _, p := range pages {
				out.Pages = append(out.Pages, launch.RenderPageHelp(cwd, p))
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		},
	}
}

// renderParity writes a parity diff for a person: the baseline and where it
// came from, then every path that differs with its digest. The preview and the
// cut share it, so the two never describe one diff differently.
func renderParity(w io.Writer, p *launch.ParityReport) {
	if p == nil {
		return
	}
	baseline := p.Baseline
	if baseline == "" {
		baseline = "(no previous release)"
	}
	// The ignored variables are named on a refusal too: a direct connection is
	// the likeliest reason a fetch behind a mandatory proxy fails.
	ignored := func() {
		if len(p.EnvIgnored) > 0 {
			fmt.Fprintf(w, "    ignored from the environment: %s\n", termsafe.Sanitize(strings.Join(p.EnvIgnored, ", ")))
		}
	}
	if p.Refused {
		fmt.Fprintf(w, "  parity:         refused against %s — %s\n", termsafe.Sanitize(baseline), termsafe.Sanitize(p.RefusalReason))
		ignored()
		return
	}
	fmt.Fprintf(w, "  parity:         against %s (%s): %d added, %d changed, %d removed, %d unchanged\n",
		termsafe.Sanitize(baseline), p.Source, p.Added, p.Changed, p.Removed, p.Unchanged)
	for _, u := range p.Fetched {
		fmt.Fprintf(w, "    fetched %s\n", termsafe.Sanitize(u))
	}
	ignored()
	if p.Note != "" {
		fmt.Fprintf(w, "    note: %s\n", termsafe.Sanitize(p.Note))
	}
	if len(p.NotCompared) > 0 {
		fmt.Fprintf(w, "    not compared (the release archive omits it): %s\n", termsafe.Sanitize(strings.Join(p.NotCompared, ", ")))
	}
	for _, e := range p.Entries {
		switch e.Change {
		case launch.ParityAdded:
			fmt.Fprintf(w, "    added %s %s\n", termsafe.Sanitize(e.Path), e.Digest)
		case launch.ParityRemoved:
			fmt.Fprintf(w, "    removed %s (was %s)\n", termsafe.Sanitize(e.Path), e.BaselineDigest)
		default:
			fmt.Fprintf(w, "    changed %s %s (was %s)\n", termsafe.Sanitize(e.Path), e.Digest, e.BaselineDigest)
		}
	}
}

// renderDeepSmoke writes the deep tier's verdict line.
func renderDeepSmoke(w io.Writer, d *launch.DeepSmokeReport) {
	if d == nil {
		return
	}
	verdict := "every page loads"
	if !d.OK {
		verdict = fmt.Sprintf("%d finding(s)", len(d.Findings))
	}
	fmt.Fprintf(w, "  deep smoke:     rendered %d page(s) in an isolated subprocess, %s\n", d.Checked, verdict)
}
