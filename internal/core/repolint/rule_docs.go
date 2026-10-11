package repolint

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// docsCurrency reuses the docs-lint engine to surface documentation drift —
// change-narration tense, broken relative links, stray root markdown. It is
// Where-gated on docs/ or a committed .abcd/docs-lint.json existing: a repo
// with neither has no docs target, so the rule is skipped rather than failed.
// A config alone is enough, because its roots need not include docs/ (a
// README, a brief, a conventions file), and skipping it there hid the docs
// target's refusal from bare `abcd lint` (iss-2610100649479892).
//
// Every finding is emitted at warn severity regardless of the underlying
// docs-lint severity: audit is an advisory conformance surface, and the
// authoritative docs gate is `abcd lint docs` itself (which still exits 2 on a
// blocker). Re-raising a docs blocker as an audit error would double-gate the
// same check. (Recorded in DECISIONS.md.) A REFUSAL is not a finding: a config
// the target cannot load, or one it refuses to lint (a roots entry that does
// not resolve), checked nothing, so it is an error (targetRefusal).
type docsCurrency struct{}

// docsRefusalFix is where the docs target's refusal is fixed.
const docsRefusalFix = "correct .abcd/docs-lint.json where the refusal says (a roots entry that does not exist is " +
	"removed from roots, or the file it names created), then run `abcd lint docs`"

func (docsCurrency) Meta() RuleMeta {
	return RuleMeta{
		ID:         "docs-currency",
		Severity:   SeverityWarn,
		Fix:        "run `abcd lint docs` and resolve the drift it reports",
		PolicyInfo: "docs describe what IS; change-narration, broken links, and stray root markdown are the drift signals the docs-lint engine already checks",
	}
}

// Where: only when docs/ exists or a docs-lint config is present.
func (docsCurrency) Where(ctx Context) bool {
	if isDir, err := fsutil.IsDir(filepath.Join(ctx.RepoRoot, "docs")); err == nil && isDir {
		return true
	}
	present, err := fsutil.Exists(filepath.Join(ctx.RepoRoot, ".abcd", "docs-lint.json"))
	return err == nil && present
}

func (docsCurrency) Eval(ctx Context) ([]Finding, error) {
	cfgPath := filepath.Join(ctx.RepoRoot, ".abcd", "docs-lint.json")
	// The engine reuse needs the repo's docs-lint config. A repo with docs/ but
	// no committed config cannot be linted — but the rule must not then read as a
	// silent pass. Surface a warn that the check could not run (a gap the
	// install closes, since it seeds the config), not an absence of drift.
	if present, err := fsutil.Exists(cfgPath); err != nil {
		return nil, err
	} else if !present {
		return []Finding{{
			RuleID:   "docs-currency",
			Severity: SeverityWarn,
			File:     ".abcd/docs-lint.json",
			Message:  "docs/ is present but .abcd/docs-lint.json is missing — docs currency cannot be checked; run abcd ahoy install, which seeds it",
		}}, nil
	}

	cfg, err := lint.LoadConfig(cfgPath)
	if err != nil {
		// A config the docs target cannot load is its refusal: one error
		// pointer, without leaking the underlying path error.
		return []Finding{targetRefusal("docs-currency", "docs", ".abcd/docs-lint.json",
			"docs-lint config could not be loaded: "+cleanErr(err), docsRefusalFix)}, nil
	}

	findings, err := lint.Lint(cfg, ctx.RepoRoot)
	if err != nil {
		// The engine refused the tree (a roots entry that does not resolve, a
		// path outside the repository) or could not read it: the same refusal
		// `abcd lint docs` exits 2 on, reported beside the other rules' results
		// rather than taking them down with an aborted lint.
		return []Finding{targetRefusal("docs-currency", "docs", ".abcd/docs-lint.json", cleanErr(err), docsRefusalFix)}, nil
	}
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		out = append(out, Finding{
			RuleID:   "docs-currency",
			Severity: SeverityWarn, // advisory; the docs gate is `abcd lint docs`
			File:     f.File,
			Line:     f.Line,
			Message:  f.Message,
		})
	}
	return out, nil
}

// cleanErr returns an error's message with any *os.PathError path stripped, so a
// config-load failure never leaks an absolute path into a finding.
func cleanErr(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}
