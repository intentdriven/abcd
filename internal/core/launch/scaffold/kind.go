package scaffold

// kind.go — the kind-shaped scaffold (itd-2609150819432059, decisions 4–7).
//
// A plugin keeps the shipped set: release.yml, auto-release.yml, the runbook and
// the reviews-charter check. Any other declared kind receives the gate workflow
// as a file of its own — the release template rendered as gate plumbing plus a
// named empty build job — the runbook and the charter check, the empty
// [Unreleased] anchor when it has no changelog, and auto-release.yml only when
// no release workflow of its own is already in charge. An existing release
// workflow is never opened; the report names it and prints what to add to it.

import (
	"os"
	"path/filepath"
)

// The non-plugin kind's files.
const (
	// GateWorkflowName is the gate workflow's file name.
	GateWorkflowName = "abcd-release-gate.yml"
	// GateWorkflowPath is where the gate workflow is written.
	GateWorkflowPath = ".github/workflows/" + GateWorkflowName
	// ChangelogPath is the release record the changelog-driven path reads.
	ChangelogPath = "CHANGELOG.md"
	// ChangelogAnchor is the changelog a repository with none receives: the
	// empty [Unreleased] anchor and nothing else (decision 7). History before
	// adoption is not represented.
	ChangelogAnchor = "# Changelog\n\n## [Unreleased]\n"
)

// ownReleaseWorkflows are the paths a repository's own release workflow is
// found at. A non-plugin kind is never written one of them, so a file there is
// the repository's, not abcd's.
var ownReleaseWorkflows = []string{".github/workflows/release.yml", ".github/workflows/release.yaml"}

// gateCallStanza is the job a repository's own release workflow adds to call
// the gate before it builds. The gate's tag and release jobs declare
// contents: write, and a called workflow's jobs may not ask for more than the
// calling job grants, so the stanza grants it; with create_tag and publish
// false those jobs are skipped and write nothing.
const gateCallStanza = `jobs:
  abcd-release-gate:
    uses: ./.github/workflows/` + GateWorkflowName + `
    with:
      tag: ${{ github.ref_name }}
      publish: false
    permissions:
      contents: write
  # ...and on the job that builds and publishes:
  #   needs: abcd-release-gate
`

// GateSubstitutions is the fact set a non-plugin kind's gate renders with: the
// bare profile rendered as the gate workflow. own names the repository's own
// release workflow when it has one, which the runbook then describes as the
// gate's caller.
func GateSubstitutions(defaultBranch, own string) Substitutions {
	subs := BareSubstitutions(defaultBranch)
	subs.Gate = true
	subs.ReleaseWorkflow = GateWorkflowName
	subs.OwnReleaseWorkflow = own
	if own != "" {
		subs.CallStanza = gateCallStanza
	}
	return subs
}

// findOwnReleaseWorkflow returns the repository's own release workflow,
// repo-relative, or "" when it has none. Lstat, so a link there counts as
// present: it is never opened, only left alone.
func findOwnReleaseWorkflow(repoRoot string) string {
	for _, rel := range ownReleaseWorkflows {
		if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(rel))); err == nil {
			return rel
		}
	}
	return ""
}

// isGoModule reports whether the repository carries a go.mod.
func isGoModule(repoRoot string) bool {
	info, err := os.Lstat(filepath.Join(repoRoot, "go.mod"))
	return err == nil && info.Mode().IsRegular()
}

// plannedFile is one file a scaffold run writes or leaves current.
type plannedFile struct {
	rel  string
	data []byte
}
