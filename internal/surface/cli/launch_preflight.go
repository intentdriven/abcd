package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/release"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// launchPreview is what `abcd launch --dry-run` reports: the bundle report and,
// beside it, the release cut the next `launch ship` would make — its derived
// version, deciding impact, records, guard verdict and findings
// (spc-2610100613109045, decision 5; the preview that was `abcd changelog`).
// The report is embedded so its JSON shape is unchanged; the cut is an
// addition to it. The cut is the deterministic emit and nothing more: no prose
// is composed and no agent runs in a preview.
type launchPreview struct {
	launch.DryRunReport
	// Cut is the cut the release would make, present whenever it could be
	// read. A refused cut is information here, as it was in the changelog
	// verb, and changes no exit code the dry run gives.
	Cut *release.Cut `json:"cut,omitempty"`
	// CutError says why the cut could not be read, in place of Cut. It
	// refuses nothing either: the dry run's verdict is the bundle's.
	CutError string `json:"cut_error,omitempty"`
}

// previewCut reads the cut the dry run renders. It is read from the checkout
// root, as `launch ship` reads it, so a dry run from a subdirectory reports
// the same records, baseline and anchor tag as one from the root
// (iss-2609251713073532). A cut that cannot be read is returned as the reason,
// already path-scrubbed, never as an error.
func previewCut(cwd string) (*release.Cut, string) {
	root, err := gitutil.CheckoutRoot(cwd, "the release record")
	if err != nil {
		return nil, scrubPaths(err)
	}
	cut, err := emitCut(root)
	if err != nil {
		return nil, scrubPaths(err)
	}
	return &cut, ""
}

// noLaunchPayloadGuidance is what a repository that declares kind plugin but
// no launch payload is told (iss-2608270559313719). A plugin's bundle is its
// include set, so there is nothing to preview, gate or stage until it declares
// one; a repository that ships something other than a plugin says so in its
// artefact declaration instead, and its preview scans the tree the release tag
// would archive (itd-2609150819432059).
const noLaunchPayloadGuidance = "this repository declares kind plugin (.abcd/config/artefact.json) but no launch payload " +
	"(.abcd/config/launch-payload.json), so there is no plugin bundle to preview, gate or stage. A plugin declares its " +
	"payload in that file. A repository that ships no plugin declares its kind as binary or application instead, and " +
	"its preview scans the tree the release tag would archive; its releases go through the changelog-driven path: " +
	"`abcd launch scaffold` installs the release workflows, `abcd launch ship` derives the version and writes the dated " +
	"CHANGELOG heading, and the auto-release workflow tags that commit on merge"

// launchPayloadRefusal turns a launch error into what the operator reads: the
// release-path guidance for a repository with no payload, the scrubbed error
// otherwise.
func launchPayloadRefusal(err error) string {
	if errors.Is(err, launch.ErrNoLaunchPayload) {
		return noLaunchPayloadGuidance
	}
	return scrubPaths(err)
}

// launchArtefact reads the artefact declaration a launch verb runs against,
// through the one reader every verb and ahoy share (itd-2609150819432059). A
// declaration that is present and wrong — an unknown kind, a malformed file —
// refuses the verb before it reads or writes anything else. Its absence is the
// plugin shape the verbs that predate it assume; the preview and the scaffold,
// which choose what to read and write by the kind, refuse the absence in the
// core instead.
func launchArtefact(verb, cwd string) (launch.Artefact, error) {
	art, err := launch.LoadArtefactOrPlugin(cwd)
	if err != nil {
		return art, &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err) + " (nothing was written)"}
	}
	return art, nil
}

// docAuditPreflight measures the documentation audit a launch's
// documentation-auditor row reports: the docs-lint engine over the
// repository's configured doc roots. It is measured HERE, at the front door,
// because the engine lives in internal/core/lint, which imports launch — the
// citationPreflight shape. No docs-lint configuration returns nil: the row then
// says the audit is not armed rather than reading as a pass.
func docAuditPreflight(repoRoot string) *launch.DocAuditPreflight {
	cfg, err := lint.LoadConfig(filepath.Join(repoRoot, ".abcd", "docs-lint.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &launch.DocAuditPreflight{Unreadable: scrubPaths(err)}
	}
	findings, err := lint.Lint(cfg, repoRoot)
	if err != nil {
		return &launch.DocAuditPreflight{Unreadable: scrubPaths(err)}
	}
	audit := &launch.DocAuditPreflight{Findings: []launch.GateFinding{}}
	for _, f := range findings {
		audit.Findings = append(audit.Findings, launch.GateFinding{
			File: f.File, Line: f.Line, Detail: "[" + f.Severity + " " + f.RuleID + "] " + f.Message,
		})
	}
	return audit
}

// writePreflight writes one run's pre-flight report and returns the directory
// it landed in, or why it could not be written. A report that cannot be written
// never changes the run's verdict — it is the run's record, not one of its
// gates — but the failure is reported where the path would have been.
func writePreflight(repoRoot string, rep launch.PreflightReport) (path, failure string) {
	dir, err := launch.WritePreflightReport(repoRoot, rep)
	if err != nil {
		return "", scrubPaths(err)
	}
	return dir, ""
}
