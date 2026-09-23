package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/release"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// emitCut is the front door's half of the release cut: it walks the LIVE cobra
// tree and the live manifests for the current surface, then hands them to the
// core, which owns every judgement. It is the same split as GuardSurface, for
// the same reason — internal/core may not know cobra.
func emitCut(repoRoot string) (release.Cut, error) {
	current, err := SurfaceSnapshot(repoRoot)
	if err != nil {
		return release.Cut{}, err
	}
	return release.Emit(repoRoot, current)
}

// ingestCut is emitCut's write-half twin: the same live-surface walk, plus the
// two things a core must never reach for itself — the untrusted payload, and the
// clock. The date lands in a durable release heading, so it is an argument a test
// can pin rather than a wall-clock read buried in a writer.
func ingestCut(repoRoot string, raw []byte, at time.Time) (release.IngestResult, error) {
	current, err := SurfaceSnapshot(repoRoot)
	if err != nil {
		return release.IngestResult{}, err
	}
	return release.Ingest(repoRoot, current, raw, at)
}

// shipResult is what the ingest step reports: the release record that landed,
// plus the staged payload when one was asked for. The IngestResult is embedded
// so its JSON shape is unchanged for a ship that stages nothing — the payload is
// an addition to the report, not a new dialect of it.
type shipResult struct {
	release.IngestResult
	Payload *launch.PayloadRenderResult `json:"payload,omitempty"`
	// Archive is the release's plugin archive, pinned into the catalog, when the
	// repository publishes a versioned plugin (adr-2609231048308186).
	Archive *shipArchive `json:"archive,omitempty"`
}

// shipArchive is the archive half of a ship's report: the archive the release
// will publish and the address the catalog now pins it at. The archive itself
// is not kept — the release workflow renders it again from the tagged commit and
// refuses to publish unless its digest is this one.
type shipArchive struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	URL     string `json:"url"`
	Files   int    `json:"files"`
}

// releaseRenderRequest assembles the render input for a completed cut.
//
// It is the one place the derived version crosses from the cut into a manifest.
// Everything the core needs is assembled HERE — the version from the cut, the
// tier from the version delta, the date from the same clock the heading was
// dated with, the source commit from git — because internal/core/launch must not
// read a clock or shell out to decide what a durable artefact says.
func releaseRenderRequest(repoRoot, dest string, cut release.Cut, at time.Time) (launch.PayloadRenderRequest, error) {
	next, err := launch.ParseSemver(strings.TrimPrefix(cut.NextTag, "v"))
	if err != nil {
		return launch.PayloadRenderRequest{}, err
	}
	prev, err := launch.ParseSemver(strings.TrimPrefix(cut.BaseTag, "v"))
	if err != nil {
		return launch.PayloadRenderRequest{}, err
	}
	sha, err := gitutil.Run(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return launch.PayloadRenderRequest{}, err
	}
	return launch.PayloadRenderRequest{
		RepoRoot: repoRoot,
		Dest:     dest,
		Version:  next.String(),
		Entry: launch.ChangelogEntry{
			Tier:      launch.BumpTier(prev, next),
			Reason:    bumpReason(cut),
			Date:      at,
			SourceSHA: sha,
		},
	}, nil
}

// renderReleasePayload stages the versioned release payload for a completed cut.
func renderReleasePayload(repoRoot, dest string, cut release.Cut, at time.Time) (launch.PayloadRenderResult, error) {
	req, err := releaseRenderRequest(repoRoot, dest, cut, at)
	if err != nil {
		return launch.PayloadRenderResult{}, err
	}
	return launch.RenderPayload(req)
}

// pinReleaseArchive renders the cut's plugin archive from the working tree —
// staging the payload at dest, packing the zip under zipDir — and pins its
// release address and digest into the working tree's catalog. It returns the
// archive, and the staged render for a caller that asked to keep it.
func pinReleaseArchive(repoRoot, dest, zipDir string, cut release.Cut, at time.Time) (shipArchive, launch.PayloadRenderResult, error) {
	req, err := releaseRenderRequest(repoRoot, dest, cut, at)
	if err != nil {
		return shipArchive{}, launch.PayloadRenderResult{}, err
	}
	a, staged, err := launch.RenderPluginArchive(req, zipDir)
	if err != nil {
		return shipArchive{}, staged, err
	}
	url, err := launch.ArchiveReleaseURL(repoRoot, a.Version)
	if err != nil {
		return shipArchive{}, staged, err
	}
	if err := launch.WriteArchivePin(repoRoot, launch.ArchivePin{URL: url, SHA256: a.SHA256}); err != nil {
		return shipArchive{}, staged, err
	}
	// The catalog's shape is part of the committed surface snapshot, so the
	// snapshot is regenerated beside it: the ship's commit must carry a snapshot
	// that matches its own tree, or the next cut's guardrail refuses.
	if err := refreshSurfaceSnapshot(repoRoot); err != nil {
		return shipArchive{}, staged, err
	}
	return shipArchive{Name: a.Name, Version: a.Version, SHA256: a.SHA256, URL: url, Files: a.Files}, staged, nil
}

// refreshSurfaceSnapshot rewrites the committed surface snapshot from the live
// tree, when the repository keeps one.
func refreshSurfaceSnapshot(repoRoot string) error {
	path := filepath.Join(repoRoot, filepath.FromSlash(SurfaceSnapshotPath))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	data, err := GenerateSurface(repoRoot)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicPreserveMode(path, data)
}

// publishesPluginArchive reports whether a ship must pin a plugin archive: the
// repository declares adr-19's version-location contract, which is its
// statement that it publishes a versioned plugin.
func publishesPluginArchive(repoRoot string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, ".abcd", "config", "version-location.json"))
	return err == nil
}

// publishedVersion is the version a launch would publish: the version of the
// newest dated CHANGELOG heading (adr-37), which is the release auto-release.yml
// turns into a tag.
//
// It is resolved at the front door rather than inside internal/core/launch for
// two reasons. adr-19 leaves no version key in the committed manifests, so there
// is nothing there for the launch core to read; and the one reader of that
// heading lives in internal/core/changelog, which imports launch — so the front
// door is the only place the two can meet without a cycle.
//
// An absent or unreadable release record yields the empty string, which the
// retention gate reports as a refusal. Naming no version is the honest answer; a
// launch preview must never invent one.
func publishedVersion(repoRoot string) string {
	v, found, err := changelog.LatestChangelogVersion(repoRoot)
	if err != nil || !found {
		return ""
	}
	return v.String()
}

// changelogPath names the release record the ingest step writes. The ship verb
// holds its pre-ingest bytes so a refused render can put them back.
func changelogPath(repoRoot string) string {
	return filepath.Join(repoRoot, "CHANGELOG.md")
}

// rollbackCut undoes a cut whose payload render refused: it restores the
// pre-ingest release record (and any other file the ship rewrote), and removes
// the staging directory the precheck proved was empty or absent, so nothing
// outside the render's own output is touched.
//
// It returns the sentence appended to the refusal rather than an error, because
// what an operator reading exit 2 most needs to know is whether a durable write
// survived — and a rollback that itself failed is the one case where they must
// recover by hand.
func rollbackCut(repoRoot, dest string, before []byte, others ...savedFile) string {
	var failures []string
	if err := fsutil.WriteFileAtomicPreserveMode(changelogPath(repoRoot), before); err != nil {
		failures = append(failures, "CHANGELOG.md: "+scrubPaths(err))
	}
	for _, f := range others {
		if err := f.restore(repoRoot); err != nil {
			failures = append(failures, f.rel+": "+scrubPaths(err))
		}
	}
	if dest != "" {
		if err := os.RemoveAll(dest); err != nil {
			failures = append(failures, "the payload destination: "+scrubPaths(err))
		}
	}
	if len(failures) > 0 {
		return "\n  THE ROLLBACK FAILED — recover by hand: " + strings.Join(failures, "; ")
	}
	return "\n  the release record was rolled back and nothing was staged"
}

// savedFile is a repository file's pre-ship bytes, held so a refused ship can
// put it back. present is false for a file that did not exist, which the
// restore removes again.
type savedFile struct {
	rel     string
	data    []byte
	present bool
}

// saveFile reads rel's current bytes.
func saveFile(repoRoot, rel string) (savedFile, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	if os.IsNotExist(err) {
		return savedFile{rel: rel}, nil
	}
	if err != nil {
		return savedFile{}, err
	}
	return savedFile{rel: rel, data: data, present: true}, nil
}

func (f savedFile) restore(repoRoot string) error {
	path := filepath.Join(repoRoot, filepath.FromSlash(f.rel))
	if !f.present {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return fsutil.WriteFileAtomicPreserveMode(path, f.data)
}

// bumpReason is the human sentence adr-20's changelog entry records: the impact
// that decided the version and the records that carried it, so a published
// manifest says WHY it is this version and not merely what it is.
func bumpReason(cut release.Cut) string {
	if len(cut.DecidedBy) == 0 {
		return string(cut.Impact)
	}
	return string(cut.Impact) + ": " + strings.Join(cut.DecidedBy, ", ")
}

// newLaunchShipCommand builds `abcd launch ship`, the release-cut write verb.
//
// It has the dual-mode shape of the disembark synthesis verbs, and for the same
// reason: a deterministic step and a host-delegated one are two halves of ONE
// verb, not two commands that could drift apart.
//
//   - WITHOUT --changelog-json it runs the EMIT step: derive the version from
//     the records that shipped, run the surface guardrail, and render the cut a
//     changelog composer works from. This step writes nothing.
//   - WITH --changelog-json it runs the INGEST step: validate the host's composed
//     prose against the cut's record set (the completeness bijection) and write
//     the dated CHANGELOG heading. The payload is untrusted host output and is
//     read behind the shared guarded-operand path.
//
// Exit codes are the machine seam an autonomous run gates on, following `abcd
// intent ready`:
//
//	0  the cut is ready — it may go to the composer; with a payload, the dated
//	   section is written.
//	1  the cut REFUSES. The whole report is rendered first and every refusal
//	   names the specific record, version, or surface that blocks it; the code
//	   is the only extra signal. A refusal is a result, not a crash.
//	2  a structural fault — the repository could not be read, an operand was
//	   unusable, or the composed changelog does not describe the cut. The
//	   diagnostic is path-scrubbed. Nothing is written on this path.
func newLaunchShipCommand(asJSON *bool) *cobra.Command {
	var changelogJSON string
	var payloadDir string
	cmd := &cobra.Command{
		Use:   "ship [--changelog-json <file|->] [--payload-dir <dir>]",
		Short: "Cut a release: derive the version and the record set from what shipped (exit 1 when the cut refuses)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// The payload is rendered by the INGEST step, from a version only a
			// completed cut has. Asking the emit step for one is an operand
			// error, refused before anything is read or staged.
			if payloadDir != "" && changelogJSON == "" {
				return &exitError{Code: 2, Msg: "abcd launch ship: --payload-dir needs --changelog-json — " +
					"the release payload is staged by the ingest step, which the deterministic emit step does not run"}
			}
			// Read the payload before anything else: it is untrusted host input,
			// and reading it through the shared guarded-operand path keeps the
			// ingest seam behind exactly the trust boundary the other delegated
			// verbs use. An empty flag is the sentinel for deterministic mode.
			raw, err := readSynthesisPayload(cmd, changelogJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
			}
			if raw != nil {
				return runShipIngest(cmd, cwd, raw, payloadDir, *asJSON)
			}

			cut, err := emitCut(cwd)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, cut, func(w io.Writer) {
				renderCut(w, "abcd launch ship", cut)
			}); rerr != nil {
				return rerr
			}
			if !cut.Ready {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&changelogJSON, "changelog-json", "",
		"path to the host-composed changelog JSON (or - for stdin); absent runs the deterministic emit step")
	cmd.Flags().StringVar(&payloadDir, "payload-dir", "",
		"stage the versioned release payload in this directory (must be empty and outside the repository)")
	return cmd
}

// runShipIngest is the ingest step of `abcd launch ship`: validate the composed
// prose against the cut, write the dated heading, and — when the repository
// publishes a versioned plugin — render the release's plugin archive and pin it
// into the catalog; with --payload-dir, keep the staged payload too.
func runShipIngest(cmd *cobra.Command, cwd string, raw []byte, payloadDir string, asJSON bool) error {
	archive := publishesPluginArchive(cwd)
	stage := archive || payloadDir != ""

	// Every render refusal that does not need a version is made BEFORE the
	// ingest step writes the dated heading. That heading is a durable release
	// record: a render that refuses after it lands leaves a release in flight,
	// and every retry is then refused for being in flight. The version-free
	// half of the render is therefore run first, and it writes nothing.
	var (
		before  []byte
		saved   []savedFile
		staging = payloadDir
		zipDir  string
	)
	if stage {
		if archive {
			scratch, err := os.MkdirTemp("", "abcd-ship-")
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
			}
			defer func() { _ = os.RemoveAll(scratch) }()
			zipDir = filepath.Join(scratch, "archive")
			if err := os.Mkdir(zipDir, 0o700); err != nil {
				return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
			}
			if staging == "" {
				staging = filepath.Join(scratch, "payload")
			}
		}
		pre, perr := launch.PrecheckPayload(cwd, staging)
		if perr != nil {
			return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(perr)}
		}
		if archive {
			if err := launch.PrecheckPluginArchive(cwd); err != nil {
				return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
			}
			dirty, err := launch.DirtyPayloadFiles(cwd, pre.Bundle)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
			}
			if len(dirty) > 0 {
				return &exitError{Code: 2, Msg: "abcd launch ship: the payload carries uncommitted changes (" +
					termsafe.Sanitize(strings.Join(dirty, ", ")) + ") — the archive pinned from this tree would not be " +
					"the archive the release renders from the commit; commit or discard them first"}
			}
			for _, rel := range []string{".claude-plugin/marketplace.json", SurfaceSnapshotPath} {
				f, err := saveFile(cwd, rel)
				if err != nil {
					return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
				}
				saved = append(saved, f)
			}
		}
		var err error
		before, err = os.ReadFile(changelogPath(cwd))
		if err != nil {
			return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
		}
	}
	at := time.Now()
	ingested, err := ingestCut(cwd, raw, at)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(err)}
	}
	res := shipResult{IngestResult: ingested}
	// Render only behind a written record: a refused cut has no version to
	// stamp, and a refused document must leave the filesystem exactly as it
	// found it.
	if stage && ingested.Written {
		// The staging directory is removed on a refusal only when the operator
		// named it; the scratch one goes with the deferred cleanup regardless.
		var rerr error
		if archive {
			var a shipArchive
			var staged launch.PayloadRenderResult
			a, staged, rerr = pinReleaseArchive(cwd, staging, zipDir, ingested.Cut, at)
			if rerr == nil {
				res.Archive = &a
				if payloadDir != "" {
					res.Payload = &staged
				}
			}
		} else {
			var staged launch.PayloadRenderResult
			staged, rerr = renderReleasePayload(cwd, payloadDir, ingested.Cut, at)
			if rerr == nil {
				res.Payload = &staged
			}
		}
		if rerr != nil {
			// The precheck cleared everything version-free, so a refusal here
			// is version-shaped and rare — but the record is already on disk,
			// so it is rolled back rather than left as an untaggable release in
			// flight.
			return &exitError{Code: 2, Msg: "abcd launch ship: " + scrubPaths(rerr) +
				rollbackCut(cwd, payloadDir, before, saved...)}
		}
	}
	if rerr := render(cmd.OutOrStdout(), asJSON, res, func(w io.Writer) {
		renderIngest(w, res)
	}); rerr != nil {
		return rerr
	}
	if !res.Cut.Ready {
		return &exitError{Code: 1}
	}
	return nil
}

// newChangelogCommand builds `abcd changelog`, the DETERMINISTIC-ONLY preview of
// the next release cut.
//
// It renders exactly what `abcd launch ship` would emit — the derived version,
// the deciding impact, the record list, and the guardrail status — and no prose:
// the changelog text is composed once, at the reviewed ship, never twice with a
// chance of disagreeing. It performs ZERO writes.
//
// It always exits 0 (2 only on a structural fault), which is the deliberate
// difference from `launch ship`. This is a status render in the shape of `abcd
// capture` bare and `abcd launch --dry-run`: a refused cut is information a
// reader asked for, not a gate they tripped. The gate is the ship verb.
func newChangelogCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "changelog",
		Short: "Preview the next release cut — derived version, records, guardrail (read-only, no prose)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			cut, err := emitCut(cwd)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd changelog: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, cut, func(w io.Writer) {
				renderCut(w, "abcd changelog", cut)
			})
		},
	}
}

// renderCut writes the human rendering of a cut, shared by the preview and the
// ship verb so the two can never describe the same repository differently.
//
// Every value that came out of a record — a title, a path, a refusal quoting
// either — is sanitised: record frontmatter and prose are author-supplied text
// that reaches a terminal here.
func renderCut(w io.Writer, verb string, cut release.Cut) {
	base := cut.BaseTag
	if base == "" {
		base = "(no release tag)"
	}
	if cut.Ready {
		fmt.Fprintf(w, "%s — %s -> %s (%s)\n", verb, base, cut.NextTag, cut.Impact)
	} else {
		fmt.Fprintf(w, "%s — REFUSED (base %s)\n", verb, base)
	}
	if len(cut.DecidedBy) > 0 {
		fmt.Fprintf(w, "  decided by: %s\n", termsafe.Sanitize(strings.Join(cut.DecidedBy, ", ")))
	}
	fmt.Fprintf(w, "  guard:      %s\n", guardLine(cut.Guard))
	fmt.Fprintf(w, "  findings:   %s\n", findingsLine(cut.Findings))
	// A waived finding is named in the terminal render, not only in --json. A
	// deferral an operator cannot see in the report they actually read is
	// indistinguishable from having ignored the finding, which is the thing the
	// waiver exists to be the opposite of.
	for _, f := range cut.Findings.Waived {
		fmt.Fprintf(w, "    deferred: %s [%s] — %s\n",
			termsafe.Sanitize(f.ID), termsafe.Sanitize(f.Severity), termsafe.Sanitize(f.Reason))
	}
	renderEntries(w, "added", cut.Added)
	renderEntries(w, "removed", cut.Removed)
	for _, refusal := range cut.Refusals {
		fmt.Fprintf(w, "  refused (%s):\n", refusal.Kind)
		// SanitizeBlock, not Sanitize: a refusal reason IS lines — the surface
		// guard names one break per line, the stale-intent refusal one intent,
		// this gate one finding — and Sanitize masks a newline to '?', so the
		// split that follows found nothing to split and printed the whole list
		// as one unreadable line. The line breaks here are the render's own, not
		// injected by an untrusted value, which is exactly the case
		// SanitizeBlock exists for.
		for _, line := range strings.Split(termsafe.SanitizeBlock(refusal.Reason), "\n") {
			fmt.Fprintf(w, "    %s\n", strings.TrimRight(line, " "))
		}
	}
}

// renderIngest writes the human rendering of a completed ship: the same cut
// report the emit step renders, then what landed in the release record.
//
// The cut is rendered FIRST and unchanged, so the two halves of one verb read
// identically — an operator comparing a dry preview with the ship that followed
// is comparing the same lines, not two dialects of the same report.
func renderIngest(w io.Writer, res shipResult) {
	renderCut(w, "abcd launch ship", res.Cut)
	if !res.Written {
		return
	}
	fmt.Fprintf(w, "  wrote:      %s\n", res.Path)
	fmt.Fprintf(w, "    %s\n", res.Heading)
	fmt.Fprintf(w, "    %d line(s), citing %s\n", res.Lines, termsafe.Sanitize(strings.Join(res.Cited, ", ")))
	if res.Archive != nil {
		fmt.Fprintf(w, "  archive:    %s (%d file(s)), pinned in .claude-plugin/marketplace.json\n", res.Archive.Name, res.Archive.Files)
		fmt.Fprintf(w, "    url:    %s\n", termsafe.Sanitize(res.Archive.URL))
		fmt.Fprintf(w, "    sha256: %s\n", res.Archive.SHA256)
	}
	if res.Payload == nil {
		return
	}
	// The staged path is an operator-supplied absolute location, so it is
	// reported through the same sanitiser every other outside string uses.
	fmt.Fprintf(w, "  payload:    %s\n", termsafe.Sanitize(res.Payload.Dest))
	fmt.Fprintf(w, "    %d file(s), version %s in %s\n",
		res.Payload.Files, res.Payload.Version, strings.Join(res.Payload.Manifests, ", "))
}

// guardLine renders the guardrail verdict, keeping a REFUSED guard visually
// distinct from a passing one: conflating "nothing broke" with "I could not
// tell" is the whole risk the guardrail exists to close.
func guardLine(g changelog.SurfaceGuard) string {
	line := string(g.Status)
	if line == "" {
		line = "(not run)"
	}
	if len(g.Breaks) > 0 {
		line += fmt.Sprintf(" (%d surface change(s))", len(g.Breaks))
	}
	return line
}

// findingsLine renders the unfixed-findings verdict, counting both what blocks
// the cut and what was consciously deferred past it. The waived count is shown
// on a PASS: a release that carries deferrals is a different fact from one that
// carries none, and a line that reported only "passed" would hide it.
func findingsLine(g changelog.FindingGuard) string {
	line := string(g.Status)
	if line == "" {
		line = "(not run)"
	}
	if len(g.Unfixed) > 0 {
		line += fmt.Sprintf(" (%d unfixed finding(s) captured since %s)", len(g.Unfixed), g.BaseTag)
	}
	// A deletion is counted on its own line-fragment rather than folded into the
	// unfixed count: the two are different defects with different remedies, and a
	// single number would let a removed record read as one still sitting in open/
	// — which is the confusion the deletion check exists to end.
	if len(g.Deleted) > 0 {
		line += fmt.Sprintf(" (%d record(s) deleted from the ledger since %s)", len(g.Deleted), g.BaseTag)
	}
	if len(g.Waived) > 0 {
		line += fmt.Sprintf(" (%d deferred)", len(g.Waived))
	}
	return line
}

// renderEntries lists one side of the cut, marking the records the changelog
// prose must NOT cite so a reader can see why a record is present but silent.
func renderEntries(w io.Writer, label string, entries []release.Entry) {
	fmt.Fprintf(w, "  %-11s %d record(s)\n", label+":", len(entries))
	for _, e := range entries {
		note := ""
		if !e.InChangelog {
			note = "  (excluded from the changelog)"
		}
		fmt.Fprintf(w, "    [%-9s] %-8s %s%s\n", e.Impact, termsafe.Sanitize(e.ID), termsafe.Sanitize(e.Title), note)
		// A record that TRIED to say it shipped elsewhere and failed says so here.
		// It is in the cut precisely because the value could not be read, so
		// without this line the operator sees an ordinary entry and never learns
		// the exclusion they wrote did not take. The field was reachable only in
		// --json before, which made "reported, not silent" true for a machine and
		// false for the person running the documented ship flow.
		if e.ShippedInErr != "" {
			fmt.Fprintf(w, "      ! %s — still in this cut\n", termsafe.Sanitize(e.ShippedInErr))
		}
	}
}
