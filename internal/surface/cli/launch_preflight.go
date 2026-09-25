package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// noLaunchPayloadGuidance is what a repository with no launch payload is told
// (iss-2608270559313719). It is not misconfigured: a repository that ships no
// plugin bundle has no include config, and its releases go through the
// changelog-driven path instead, which the refusal names.
const noLaunchPayloadGuidance = "this repository declares no launch payload (.abcd/config/launch-payload.json), " +
	"so there is no plugin bundle to preview, gate or stage. A repository without one releases through the " +
	"changelog-driven path: `abcd launch scaffold` installs the release workflows, `abcd launch ship` derives the " +
	"version and writes the dated CHANGELOG heading, and the auto-release workflow tags that commit on merge. " +
	"A repository that does ship a plugin declares its payload in that file"

// launchPayloadRefusal turns a launch error into what the operator reads: the
// release-path guidance for a repository with no payload, the scrubbed error
// otherwise.
func launchPayloadRefusal(err error) string {
	if errors.Is(err, launch.ErrNoLaunchPayload) {
		return noLaunchPayloadGuidance
	}
	return scrubPaths(err)
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
