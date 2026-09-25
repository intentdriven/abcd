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
// Where-gated on docs/ existing (a repo with no user-facing docs cannot drift),
// so an absent docs/ skips the rule rather than failing it.
//
// Every finding is emitted at warn severity regardless of the underlying
// docs-lint severity: audit is an advisory conformance surface, and the
// authoritative docs gate is `abcd lint docs` itself (which still exits 2 on a
// blocker). Re-raising a docs blocker as an audit error would double-gate the
// same check. (Recorded in DECISIONS.md.)
type docsCurrency struct{}

func (docsCurrency) Meta() RuleMeta {
	return RuleMeta{
		ID:         "docs-currency",
		Severity:   SeverityWarn,
		Fix:        "run `abcd lint docs` and resolve the drift it reports",
		PolicyInfo: "docs describe what IS; change-narration, broken links, and stray root markdown are the drift signals the docs-lint engine already checks",
	}
}

// Where: only when docs/ exists.
func (docsCurrency) Where(ctx Context) bool {
	isDir, err := fsutil.IsDir(filepath.Join(ctx.RepoRoot, "docs"))
	return err == nil && isDir
}

func (docsCurrency) Eval(ctx Context) ([]Finding, error) {
	cfgPath := filepath.Join(ctx.RepoRoot, ".abcd", "docs-lint.json")
	// The engine reuse needs the repo's docs-lint config. A repo with docs/ but
	// no committed config cannot be linted — but the rule must not then read as a
	// silent pass. Surface a warn that the check could not run (a
	// prepare-this-repo gap), not an absence of drift.
	if present, err := fsutil.Exists(cfgPath); err != nil {
		return nil, err
	} else if !present {
		return []Finding{{
			RuleID:   "docs-currency",
			Severity: SeverityWarn,
			File:     ".abcd/docs-lint.json",
			Message:  "docs/ is present but .abcd/docs-lint.json is missing — docs currency cannot be checked; run prepare-this-repo",
		}}, nil
	}

	cfg, err := lint.LoadConfig(cfgPath)
	if err != nil {
		// A malformed config is a real problem, but it is the docs-lint surface's
		// to report, not audit's — surface it as a single warn pointer without
		// leaking the underlying path error.
		return []Finding{{
			RuleID:   "docs-currency",
			Severity: SeverityWarn,
			File:     ".abcd/docs-lint.json",
			Message:  "docs-lint config could not be loaded: " + cleanErr(err),
		}}, nil
	}

	findings, err := lint.Lint(cfg, ctx.RepoRoot)
	if err != nil {
		return nil, err
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
