package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// archiveReport is what `abcd launch archive` reports: the archive it wrote, the
// address a release publishes it at, and — with --verify — whether the
// committed catalog pins exactly this archive.
type archiveReport struct {
	Archive launch.PluginArchive `json:"archive"`
	URL     string               `json:"url"`
	Pin     archivePinCheck      `json:"pin"`
}

// archivePinCheck is the --verify verdict. Checked is false when the caller did
// not ask, so a render that proved nothing never reads as one that passed.
type archivePinCheck struct {
	Checked bool   `json:"checked"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
}

// newLaunchArchiveCommand builds `abcd launch archive`, the release gate's
// half of the pinned plugin archive (adr-2609231200000000).
//
// It renders the plugin archive of the release the newest dated CHANGELOG
// heading names — the heading auto-release.yml tags — from the checked-out tree,
// writes it into --out, and with --verify refuses unless the committed catalog
// pins exactly that archive's address and digest. The release workflow runs it
// against the tagged commit twice: in verify, before anything is built, and in
// the publish job, where the verified archive is the file checksummed, attested
// and uploaded.
//
// Exit codes:
//
//	0  the archive was written (and, with --verify, matches the pin).
//	1  --verify refused: the committed pin names another archive, or none. The
//	   rendered archive is removed from --out, so no later step can publish it.
//	2  a structural fault — no dated release, a --tag naming another release, an
//	   unusable --out, or a render refusal. Nothing is left in --out.
func newLaunchArchiveCommand(asJSON *bool) *cobra.Command {
	var outDir, tag string
	var verify bool
	cmd := &cobra.Command{
		Use:   "archive --out <dir> [--tag <vX.Y.Z>] [--verify]",
		Short: "Render the release's plugin archive and (--verify) prove the committed catalog pins it (exit 1 on a mismatch)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if outDir == "" {
				return &exitError{Code: 2, Msg: "abcd launch archive: --out names no directory"}
			}
			out, err := filepath.Abs(outDir)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(err)}
			}
			req, err := archiveRenderRequest(cwd, tag)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(err)}
			}
			scratch, err := os.MkdirTemp("", "abcd-archive-")
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(err)}
			}
			defer func() { _ = os.RemoveAll(scratch) }()
			req.Dest = filepath.Join(scratch, "payload")

			a, _, err := launch.RenderPluginArchive(req, out)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(err)}
			}
			rep := archiveReport{Archive: a}
			if rep.URL, err = launch.ArchiveReleaseURL(cwd, a.Version); err != nil {
				_ = os.Remove(a.Path)
				return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(err)}
			}
			if verify {
				rep.Pin.Checked = true
				verr := launch.VerifyArchivePin(cwd, a)
				switch {
				case verr == nil:
					rep.Pin.OK = true
				case errors.Is(verr, launch.ErrArchivePinMismatch):
					rep.Pin.Detail = verr.Error()
					// Fail closed: an archive the catalog does not pin must not
					// sit where a later upload step's glob would find it.
					_ = os.Remove(a.Path)
				default:
					_ = os.Remove(a.Path)
					return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(verr)}
				}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				renderArchive(w, rep)
			}); rerr != nil {
				return rerr
			}
			if rep.Pin.Checked && !rep.Pin.OK {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outDir, "out", "", "existing directory to write <plugin>-plugin-v<version>.zip into")
	cmd.Flags().StringVar(&tag, "tag", "", "refuse unless the newest dated CHANGELOG version is this tag")
	cmd.Flags().BoolVar(&verify, "verify", false, "refuse (exit 1) unless the committed catalog pins this archive's address and digest")
	return cmd
}

// archiveRenderRequest assembles the render input for the release the newest
// dated CHANGELOG heading names.
//
// Only the version and the plugin payload reach the archive: the changelog
// entry lands in the staged catalog, which the archive leaves out. The entry is
// still assembled from real facts — the heading's date, the tier from the
// previous dated release, the checked-out commit — because the render validates
// it and a placeholder would be a fabricated record, however briefly it lives.
func archiveRenderRequest(repoRoot, tag string) (launch.PayloadRenderRequest, error) {
	releases, found, err := changelog.DatedReleases(repoRoot)
	if err != nil {
		return launch.PayloadRenderRequest{}, err
	}
	if !found || len(releases) == 0 {
		return launch.PayloadRenderRequest{}, errors.New("CHANGELOG.md dates no release, so there is no version to render the archive for")
	}
	newest := releases[0]
	next, err := launch.ParseSemver(strings.TrimPrefix(newest.Version, "v"))
	if err != nil {
		return launch.PayloadRenderRequest{}, err
	}
	if tag != "" && tag != next.Tag() {
		return launch.PayloadRenderRequest{}, fmt.Errorf("--tag %s is not the release CHANGELOG.md dates newest (%s)", termsafe.Sanitize(tag), next.Tag())
	}
	prev := launch.Semver{}
	if len(releases) > 1 {
		if prev, err = launch.ParseSemver(strings.TrimPrefix(releases[1].Version, "v")); err != nil {
			return launch.PayloadRenderRequest{}, err
		}
	}
	date, err := time.Parse("2006-01-02", firstField(newest.Date))
	if err != nil {
		return launch.PayloadRenderRequest{}, fmt.Errorf("the dated heading for %s carries no YYYY-MM-DD date", next.Tag())
	}
	sha, err := gitutil.Run(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return launch.PayloadRenderRequest{}, err
	}
	return launch.PayloadRenderRequest{
		RepoRoot: repoRoot,
		Version:  next.String(),
		Entry: launch.ChangelogEntry{
			Tier:      launch.BumpTier(prev, next),
			Reason:    "the release CHANGELOG.md dates " + next.Tag(),
			Date:      date,
			SourceSHA: sha,
		},
	}, nil
}

// firstField is the first whitespace-separated field of s.
func firstField(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

// renderArchive writes the human rendering of `abcd launch archive`.
func renderArchive(w io.Writer, rep archiveReport) {
	a := rep.Archive
	fmt.Fprintf(w, "abcd launch archive — %s (%d file(s), %d bytes)\n", a.Name, a.Files, a.Bytes)
	fmt.Fprintf(w, "  sha256:  %s\n", a.SHA256)
	fmt.Fprintf(w, "  url:     %s\n", termsafe.Sanitize(rep.URL))
	switch {
	case !rep.Pin.Checked:
		fmt.Fprintf(w, "  written: %s\n", termsafe.Sanitize(a.Path))
		fmt.Fprintln(w, "  pin:     not checked (pass --verify to prove the committed catalog pins this archive)")
	case rep.Pin.OK:
		fmt.Fprintf(w, "  written: %s\n", termsafe.Sanitize(a.Path))
		fmt.Fprintln(w, "  pin:     matches the committed catalog")
	default:
		fmt.Fprintln(w, "  written: nothing (the archive was removed)")
		fmt.Fprintf(w, "  pin:     MISMATCH — %s\n", termsafe.Sanitize(rep.Pin.Detail))
	}
}
