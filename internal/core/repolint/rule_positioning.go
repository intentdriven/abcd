package repolint

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// identityPositioning holds every registered surface — the README strapline, the
// manifest description, the conventions file's opening — to the repo's one
// canonical identity block. A project's positioning fragments silently when each
// surface is edited at a different moment (iss-143's recorded three-variant
// tagline drift is what this rule exists to catch), so the check names the file,
// the exact drifted line, and the canonical line it should carry.
//
// It is warn-tier by default: it highlights, it never gates, and it never
// rewrites a surface — `abcd identity render` proposes the correction as a diff
// a human adopts. A repo that wants a hard gate sets "severity": "blocker" in
// its registry, which promotes every finding in the family to error.
type identityPositioning struct{}

// findingMaxLen caps the quoted surface text in a message. A whole drifted
// paragraph would swamp the report; the head of it is what identifies the line.
const findingMaxLen = 200

func (identityPositioning) Meta() RuleMeta {
	return RuleMeta{
		ID:       "identity-positioning",
		Severity: SeverityWarn,
		Fix:      "run `abcd identity render` and adopt the proposed diff, or change the identity block deliberately",
		PolicyInfo: "one canonical identity block is the single home for what the project says about itself; " +
			"every rendered surface derives from it, and a surface that says something else is drift",
	}
}

// Where: only a repo that has committed a positioning registry has adopted the
// check. An absent registry is opting out, never a silent violation.
func (identityPositioning) Where(ctx Context) bool {
	present, err := fsutil.Exists(filepath.Join(ctx.RepoRoot, filepath.FromSlash(positioning.ConfigRelPath)))
	return err == nil && present
}

func (r identityPositioning) Eval(ctx Context) ([]Finding, error) {
	cfg, ok, err := positioning.LoadConfig(ctx.RepoRoot)
	if err != nil {
		// A registry that cannot be loaded must not read as "no drift": report
		// it as a finding of this family rather than aborting the whole audit,
		// which would take the other rules' results down with it.
		// `abcd lint identity` refuses the same registry, so this is an error.
		return []Finding{targetRefusal("identity-positioning", "identity", positioning.ConfigRelPath,
			"positioning registry could not be loaded: "+cleanErr(err),
			"correct "+positioning.ConfigRelPath+" (or remove it to opt out of the positioning check)")}, nil
	}
	if !ok {
		// Raced away between Where and Eval; nothing to check.
		return nil, nil
	}

	rep, err := positioning.Check(ctx.RepoRoot, cfg)
	if err != nil {
		// A block that cannot be read checked no surface: the target's
		// refusal, an error whatever severity the family's drift takes.
		return []Finding{targetRefusal("identity-positioning", "identity", cfg.Block.File,
			"the canonical identity block could not be read: "+cleanErr(err),
			"add the identity block, or point "+positioning.ConfigRelPath+" at where it lives")}, nil
	}

	sev := severityFor(cfg)
	var out []Finding
	for _, s := range rep.Surfaces {
		switch s.Status {
		case positioning.StatusDrifted:
			out = append(out, Finding{
				RuleID:   "identity-positioning",
				Severity: sev,
				File:     s.File,
				Line:     s.Line,
				Message: fmt.Sprintf("surface %q no longer carries the canonical %s — it says %q; the identity block (%s) says %q",
					s.ID, strings.Join(s.Missing, ", "), truncate(s.Found), rep.Block.File, s.Canonical),
			})
		case positioning.StatusUnlocatable:
			out = append(out, Finding{
				RuleID:   "identity-positioning",
				Severity: sev,
				File:     s.File,
				Message: fmt.Sprintf("surface %q is registered but could not be located in this file — its locator matches nothing, so drift there would go unseen",
					s.ID),
				Fix: "update the surface's locator in " + positioning.ConfigRelPath + ", or unregister the surface",
			})
		}
	}
	return out, nil
}

// severityFor maps the registry's family severity onto the audit vocabulary. The
// mapping happens here, at this rule's boundary, so the audit engine keeps one
// severity vocabulary (the docs-currency precedent).
func severityFor(cfg positioning.Config) Severity {
	if strings.EqualFold(strings.TrimSpace(cfg.Severity), positioning.SeverityBlocker) {
		return SeverityError
	}
	return SeverityWarn
}

// truncate shortens quoted surface text for a one-line finding, collapsing the
// wrapping so a paragraph reads as the line it drifted on. The cut backs off to
// a rune boundary: slicing bytes mid-rune would put a replacement glyph in the
// message, and the surface text is arbitrary UTF-8 from the audited repo.
func truncate(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= findingMaxLen {
		return s
	}
	cut := findingMaxLen
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
