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
// address a release publishes it at, whether — with --verify — the committed
// catalog pins exactly this archive, and whether — with --repository — that
// address is the named repository's release.
type archiveReport struct {
	Archive    launch.PluginArchive `json:"archive"`
	URL        string               `json:"url"`
	Pin        archiveCheck         `json:"pin"`
	Repository archiveCheck         `json:"repository"`
}

// archiveCheck is one gate's verdict. Checked is false when the caller did not
// ask, so a render that proved nothing never reads as one that passed.
type archiveCheck struct {
	Checked bool   `json:"checked"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
}

// newLaunchArchiveCommand builds `abcd launch archive`, the release gate's
// half of the pinned plugin archive (adr-2609231048308186).
//
// It renders the plugin archive of the release the newest dated CHANGELOG
// heading names — the heading auto-release.yml tags — from the checked-out tree,
// writes it into --out, and with --verify refuses unless the committed catalog
// pins exactly that archive's address and digest. With --repository <owner/name>
// it also refuses unless that address sits under the named repository's
// download path for the release's tag: the address derives from plugin.json's
// repository, which a rename, a transfer or a fork leaves naming another
// repository, so --verify alone passes on a pin every install 404s on. The
// release workflows run it with the repository they release from, against the
// tagged commit: in auto-release's detect job before the tag, in verify before
// anything is built, and in the publish job, where the verified archive is the
// file checksummed, attested and uploaded.
//
// Exit codes:
//
//	0  the archive was written (and, with --verify and --repository, matches
//	   the pin and sits under the repository's release).
//	1  a gate refused: the committed pin names another archive, or none, or the
//	   address is not the named repository's release. The rendered archive is
//	   removed from --out, so no later step can publish it.
//	2  a structural fault — no dated release, a --tag naming another release, a
//	   --repository that is not owner/name, an unusable --out, or a render
//	   refusal. Nothing is left in --out.
func newLaunchArchiveCommand(asJSON *bool) *cobra.Command {
	var outDir, tag, repository string
	var verify bool
	cmd := &cobra.Command{
		Use: "archive --out <dir> [--tag <vX.Y.Z>] [--verify] [--repository <owner/name>]",
		Long: `Render the release's plugin archive into --out and, with --verify, prove the
committed catalog pins it (exit 1 on a mismatch).

With --verify the pin judges the working tree: a payload file that differs
from the commit changes the archive's digest, and the pin refuses it. Without
--verify nothing else judges the tree, so an uncommitted change, tracked or
untracked and outside the local tier, refuses the render (exit 2) and nothing
is written to --out.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if outDir == "" {
				return &exitError{Code: 2, Msg: "abcd launch archive: --out names no directory"}
			}
			// A malformed --repository is refused before anything is rendered,
			// so an operand error never leaves an archive behind.
			if cmd.Flags().Changed("repository") {
				if err := launch.ValidateGitHubRepository(repository); err != nil {
					return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(err)}
				}
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
			// The dirty-tree policy is stated, never inherited. With --verify
			// the committed pin is the dirt gate: a payload file that differs
			// from the commit changes the archive's digest and refuses, while
			// the release workflows' own outputs beside the checkout are not
			// the payload. Without --verify nothing else judges the tree, so
			// a dirty one refuses.
			req.Dirty = launch.DirtyRefuse
			if verify {
				req.Dirty = launch.DirtySkip
			}

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
				default:
					_ = os.Remove(a.Path)
					return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(verr)}
				}
			}
			// Both gates run before either decides, so a refusal names every
			// gate the release failed rather than only the first.
			if cmd.Flags().Changed("repository") {
				rep.Repository.Checked = true
				rerr := launch.CheckArchiveRepository(rep.URL, repository, "v"+a.Version)
				switch {
				case rerr == nil:
					rep.Repository.OK = true
				case errors.Is(rerr, launch.ErrArchiveRepositoryMismatch):
					rep.Repository.Detail = rerr.Error()
				default:
					_ = os.Remove(a.Path)
					return &exitError{Code: 2, Msg: "abcd launch archive: " + scrubPaths(rerr)}
				}
			}
			// Fail closed: an archive a gate refused must not sit where a later
			// upload step's glob would find it.
			if rep.refused() {
				_ = os.Remove(a.Path)
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				renderArchive(w, rep)
			}); rerr != nil {
				return rerr
			}
			if rep.refused() {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outDir, "out", "", "existing directory to write <plugin>-plugin-v<version>.zip into")
	cmd.Flags().StringVar(&tag, "tag", "", "refuse unless the newest dated CHANGELOG version is this tag")
	cmd.Flags().BoolVar(&verify, "verify", false, "refuse (exit 1) unless the committed catalog pins this archive's address and digest; without it, a tree with an uncommitted change refuses (exit 2)")
	cmd.Flags().StringVar(&repository, "repository", "",
		"refuse (exit 1) unless the archive's address is this GitHub owner/name's release download for the tag")
	return cmd
}

// refused reports whether a gate the caller asked for refused the archive.
func (rep archiveReport) refused() bool {
	return (rep.Pin.Checked && !rep.Pin.OK) || (rep.Repository.Checked && !rep.Repository.OK)
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
	if rep.refused() {
		fmt.Fprintln(w, "  written: nothing (the archive was removed)")
	} else {
		fmt.Fprintf(w, "  written: %s\n", termsafe.Sanitize(a.Path))
	}
	switch {
	case !rep.Pin.Checked:
		fmt.Fprintln(w, "  pin:     not checked (pass --verify to prove the committed catalog pins this archive)")
	case rep.Pin.OK:
		fmt.Fprintln(w, "  pin:     matches the committed catalog")
	default:
		fmt.Fprintf(w, "  pin:     MISMATCH — %s\n", termsafe.Sanitize(rep.Pin.Detail))
	}
	switch {
	case !rep.Repository.Checked:
		fmt.Fprintln(w, "  repo:    not checked (pass --repository to prove the address is that repository's release)")
	case rep.Repository.OK:
		fmt.Fprintln(w, "  repo:    the address is the named repository's release")
	default:
		fmt.Fprintf(w, "  repo:    MISMATCH — %s\n", termsafe.Sanitize(rep.Repository.Detail))
	}
}
