// Package scaffold renders and writes the changelog-driven release machinery —
// release.yml, auto-release.yml, and the adr-37 runbook — into a managed repo
// that lacks it (itd-93, spc-14).
//
// # Self-scaffold parity (one template, one artifact)
//
// The three files ship as embedded templates. abcd-cli's OWN release workflows
// are regenerated from these same templates with the abcd fact set
// (AbcdSubstitutions): TestSelfScaffoldParity asserts the rendered output is
// byte-identical to the committed .github/workflows/*.yml, so the proven pattern
// and the shipped template can never drift — every abcd release exercises the
// exact machinery a managed repo receives.
//
// A bare managed repo passes a degraded fact set (BareSubstitutions): no semantic
// detectors, a generic Go build, and only the deterministic verify gates. The
// template's abcd-specific regions are guarded so the bare rendering omits them
// cleanly and still parses, arms GITHUB_TOKEN-only, and injects nothing.
//
// # Why custom template delimiters
//
// A GitHub Actions workflow is dense with `${{ … }}` expressions, which collide
// with text/template's default `{{ … }}`. The templates therefore use `<%` / `%>`
// (which never appear in a workflow) as delimiters, so every `${{ … }}` passes
// through as literal text and only the scaffold's own directives are evaluated.
package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/release.yml.tmpl templates/auto-release.yml.tmpl templates/runbook.md.tmpl templates/check-reviews.sh.tmpl
var templatesFS embed.FS

// Gate is one named verify-job step: a display name and the shell it runs. The
// deterministic gates a managed repo inherits beyond the generic Go leg are
// expressed as data so the runbook's numbered list and the workflow's steps are
// rendered from one source (the gate_lockstep invariant, in-template).
type Gate struct {
	Name string
	Run  string
}

// Substitutions is the fact set a rendering binds against. AbcdSubstitutions
// reproduces abcd-cli's live workflows byte-for-byte; BareSubstitutions degrades
// to the deterministic gates alone.
type Substitutions struct {
	// DefaultBranch is the branch auto-release triggers on and the no-branch-commit
	// tripwire guards — derived from the target repo, never hard-coded.
	DefaultBranch string
	// (There is deliberately no Go version here. The rendered workflows point
	// setup-go at go.mod with `go-version-file`, so the go directive is the only
	// place the toolchain is written — see the field's removal in
	// iss-2609090951291799. A substituted literal was a scaffold-time snapshot of
	// that directive, which went stale the moment the adopter bumped it, and it
	// was a value derived from a file that had to be validated against an
	// injection-safe allowlist before it could be written into YAML.)
	//
	// Abcd selects abcd-cli's full rendering: the extra deterministic verify gates,
	// the semantic release gate, and the four-binary cross-compile + attest build.
	// A bare managed repo sets it false and gets a generic Go build with no semantic
	// detectors.
	Abcd bool
	// ExtraGates are the deterministic verify steps beyond the generic Go leg
	// (gofmt/build/vet/test/race). Rendered into both the workflow's verify job and
	// the runbook's numbered gate list, so the two stay in lockstep by construction.
	ExtraGates []Gate
	// SemanticGates are the required host-run detectors the release gate arms
	// (`--require-gate <name>`). Empty means no semantic detector is configured and
	// the deterministic gates alone admit the release (spc-14 clean degradation).
	SemanticGates []string
	// CIChecks are the check names the managed repo's own pull-request CI
	// reports (DeriveCIChecks): the merge gate the release roll passes through.
	// The bare rendering names them in release.yml's verify header and lists them
	// in the runbook as the contexts to require on DefaultBranch. Each is held to
	// an injection-safe allowlist before it gets here. abcd's own rendering
	// leaves this empty; its merge gate is ci.yml, described in its own runbook.
	CIChecks []string
}

// Rendered is the file set a scaffold run produces, keyed by repo-relative path.
type Rendered struct {
	ReleaseYML     []byte
	AutoReleaseYML []byte
	Runbook        []byte
	// CheckReviews is the reviews-charter check (RD001) the bare verify job runs.
	CheckReviews []byte
}

// Repo-relative destinations the scaffold writes. Fixed by the GitHub Actions
// discovery rule (workflows) and mirrored from abcd-cli's own layout (runbook).
const (
	ReleaseYMLPath     = ".github/workflows/release.yml"
	AutoReleaseYMLPath = ".github/workflows/auto-release.yml"
	RunbookPath        = ".abcd/development/release-gate/README.md"
	// CheckReviewsPath is the scaffolded reviews-charter check (RD001, with the
	// sha-keyed receipt directories exempt). It sits beside the runbook, in the
	// release-gate directory the scaffold already owns, rather than in a scripts
	// directory the managed repository may lay out its own way.
	CheckReviewsPath = ".abcd/development/release-gate/check-reviews.sh"
)

// Render binds the four templates against subs and returns their bytes. A
// template parse or execute fault is a programming error in the embedded
// templates, surfaced as an error rather than a panic.
func Render(subs Substitutions) (Rendered, error) {
	rel, err := renderOne("templates/release.yml.tmpl", subs)
	if err != nil {
		return Rendered{}, err
	}
	auto, err := renderOne("templates/auto-release.yml.tmpl", subs)
	if err != nil {
		return Rendered{}, err
	}
	book, err := renderOne("templates/runbook.md.tmpl", subs)
	if err != nil {
		return Rendered{}, err
	}
	charter, err := renderOne("templates/check-reviews.sh.tmpl", subs)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{ReleaseYML: rel, AutoReleaseYML: auto, Runbook: book, CheckReviews: charter}, nil
}

// renderOne parses and executes a single embedded template with the `<%`/`%>`
// delimiters.
func renderOne(name string, subs Substitutions) ([]byte, error) {
	raw, err := templatesFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("scaffold: read embedded template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Delims("<%", "%>").
		Funcs(template.FuncMap{"add": func(a, b int) int { return a + b }}).
		Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("scaffold: parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, subs); err != nil {
		return nil, fmt.Errorf("scaffold: execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
