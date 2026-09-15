package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newReleaseFetcher is the seam through which `version --check` reaches the
// network. It is a package var so a test can inject a recording stub and prove
// two things: the check path fetches exactly once, and NO other path (version
// bare, ahoy, session-start, install) ever constructs it — the zero-network
// invariant of adr-38 tier 1.
var newReleaseFetcher = vintage.NewGitHubReleaseFetcher

// resolveUpdateTarget is the disk-only install classification the next-step
// line consults; a package var so a test can pin a shape without a PATH fixture.
var resolveUpdateTarget = ahoy.ResolveUpdateTarget

// checkSource names the network source `version --check` consulted, so the
// report says where "latest" came from (AC5).
const checkSource = "github.com/intentdriven/abcd releases"

// versionOutput is `abcd version`'s structured result: identity, install mode,
// vintage, and staleness relative to the on-disk reference. The vintage fields
// come from one comparator (ahoy.Vintage), shared with `abcd ahoy` and the
// session-start notice. Check is populated only by --check.
type versionOutput struct {
	core.VersionInfo
	InstallMode string       `json:"install_mode,omitempty"`
	Vintage     string       `json:"vintage"`
	Staleness   string       `json:"staleness"`
	Check       *checkResult `json:"check,omitempty"`
}

// ahoyOutput is `abcd ahoy`'s bare render: the detection envelope plus the
// shared vintage/staleness, so the JSON surface relays them too (AC3).
type ahoyOutput struct {
	ahoy.DetectionResult
	Vintage   string `json:"vintage"`
	Staleness string `json:"staleness"`
}

// checkResult is the explicit network check's outcome, named source and all.
type checkResult struct {
	Latest  string `json:"latest,omitempty"`
	Source  string `json:"source"`
	Verdict string `json:"verdict"`
	// NextStep names the command that takes the update, rendered through the
	// update verb's own dispatch for this install's shape (`abcd update` for a
	// swappable copy, the host's plugin update for a plugin root, brew for a
	// Cellar install). Set only when an update is available.
	NextStep string `json:"next_step,omitempty"`
}

func newVersionCommand(asJSON *bool) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print abcd's version, install mode, and vintage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			v := core.NewVersion()
			vin := ahoy.Vintage(cwd)
			out := versionOutput{
				VersionInfo: v,
				InstallMode: vin.Mode,
				Vintage:     vin.DisplayVintage(),
				Staleness:   vin.Staleness(),
			}
			// The ONLY network path in itd-111 (adr-38 tier 2): an explicit
			// --check fetches the latest release once and compares through the
			// same comparator as every disk path.
			if check {
				out.Check = runReleaseCheck(v.Version)
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "%s %s\n", v.Name, v.Version)
				if out.InstallMode != "" {
					fmt.Fprintf(w, "  install:   %s\n", out.InstallMode)
				}
				fmt.Fprintf(w, "  vintage:   %s\n", out.Vintage)
				fmt.Fprintf(w, "  staleness: %s\n", out.Staleness)
				if out.Check != nil {
					if out.Check.Latest != "" {
						fmt.Fprintf(w, "  latest:    %s (source: %s)\n", out.Check.Latest, out.Check.Source)
					}
					fmt.Fprintf(w, "  check:     %s\n", out.Check.Verdict)
					if out.Check.NextStep != "" {
						fmt.Fprintf(w, "  next:      %s\n", termsafe.Sanitize(out.Check.NextStep))
					}
				}
			})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "fetch the latest release once and compare (this command's only network touch; abcd never fetches implicitly — adr-38); names its source")
	return cmd
}

// runReleaseCheck fetches the latest release exactly once and compares it
// against the running version through the shared comparator, naming the source.
func runReleaseCheck(currentVersion string) *checkResult {
	res := &checkResult{Source: checkSource}
	// The single network touch: Expected() performs the one fetch.
	exp := vintage.ReleaseProvider(newReleaseFetcher()).Expected()
	if !exp.Known {
		res.Verdict = "the latest release could not be determined (no network, or the source is unreachable)"
		return res
	}
	// The tag is read off an HTTP redirect (shape-checked only by the tag regex);
	// sanitise it before it is rendered, as the retired skew notice does.
	res.Latest = termsafe.Sanitize(exp.Revision)
	cur := vintage.Current{Revision: currentVersion, Known: currentVersion != "" && currentVersion != "dev"}
	if !cur.Known {
		res.Verdict = "latest is " + res.Latest + "; this is an unversioned (dev) build, so there is nothing to compare"
		return res
	}
	// Compare through the shared comparator over the tag already fetched, so the
	// fetch is not repeated.
	if vintage.Compare(cur, vintage.PinnedVersion(exp.Revision)).Outcome == vintage.Fresh {
		res.Verdict = "up to date"
	} else {
		res.Verdict = "update available: " + currentVersion + " -> " + res.Latest
		// The next line is the verb (itd-130): classify the install on disk —
		// no second fetch — and name the command its shape takes.
		res.NextStep = update.NextStep(resolveUpdateTarget())
	}
	return res
}
