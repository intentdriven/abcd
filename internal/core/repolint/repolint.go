// Package audit is abcd's read-only repo-conformance engine: it evaluates a set
// of declarative rules against a repository and returns findings. It performs no
// I/O beyond reading under a caller-supplied working directory — no printing, no
// os.Exit, no mutation — so it is fully testable and reusable across surfaces.
//
// The design adapts three external patterns behind seams, copying vocabulary not
// code (see .abcd/development/plans/2026-07-13-abcd-audit-verb.md): repolinter's
// rule-object schema (id/severity/where/fix/policyInfo), Conftest's
// severity-to-exit-code semantics (0 clean, 1 warnings, 2 any error), and SARIF
// as a future optional export behind the Serializer seam. Net new dependencies:
// zero.
package repolint

import (
	"fmt"
	"sort"
)

// Severity is a rule's weight. The vocabulary is repolinter/Conftest's
// error|warn|off, chosen over the record-lint engine's blocker|warn because it
// maps directly onto the tri-state exit code and reads correctly in the human
// render. A reused docs-lint finding (blocker|warn) is mapped at that rule's
// boundary, not here.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityOff   Severity = "off"
)

// RuleMeta is a rule's declarative header — data, separate from the evaluator.
// It mirrors repolinter's rule object: a stable ID (the machine key), a
// Severity, a Fix (the remediation message shown to a human), and PolicyInfo
// (why the rule exists).
type RuleMeta struct {
	ID         string
	Severity   Severity
	Fix        string
	PolicyInfo string
}

// Rule is one conformance check. Meta is its declarative header; Where gates
// conditional enablement (a false Where skips the rule rather than failing it —
// e.g. docs-currency when docs/ is absent); Eval runs the check and returns
// findings. Eval returns an error only when the check itself cannot run (a
// malformed input), never to signal a violation — a violation is a Finding. This
// interface is the rule-loader seam: rules are values the evaluator ranges over,
// not branches wired into it.
type Rule interface {
	Meta() RuleMeta
	Where(ctx Context) bool
	Eval(ctx Context) ([]Finding, error)
}

// Context is what every rule is evaluated against: the repository under audit. It
// is passed by value; a rule must not mutate it.
type Context struct {
	// RepoRoot is the absolute path to the repository being audited.
	RepoRoot string
}

// Finding is one conformance violation. File is repo-relative; Line is 1-based
// (0 when the finding is not tied to a specific line, e.g. a missing directory).
// Fix and Policy are copied from the rule's Meta so a serialized finding is
// self-describing.
type Finding struct {
	RuleID   string   `json:"ruleId"`
	Severity Severity `json:"severity"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix,omitempty"`
	Policy   string   `json:"policyInfo,omitempty"`
}

// Result is the outcome of an audit. Findings are sorted deterministically.
// Blockers and Warnings count error- and warn-severity findings; ExitCode is the
// Conftest tri-state derived from them. Skipped names the rules whose Where
// predicate was false, so a caller can show "skipped (not applicable)" rather
// than silently omitting them.
type Result struct {
	Findings []Finding `json:"findings"`
	Skipped  []string  `json:"skipped"`
	Blockers int       `json:"-"`
	Warnings int       `json:"-"`
	ExitCode int       `json:"-"`
}

// Evaluate runs every rule against ctx and returns the aggregated, sorted result.
// A rule with SeverityOff is skipped without evaluation. A rule whose Where
// predicate is false is recorded in Skipped and not evaluated. If any rule's Eval
// returns an error, Evaluate aborts and returns it — a check that cannot run must
// not be silently reported as passing.
func Evaluate(rules []Rule, ctx Context) (Result, error) {
	var res Result
	for _, r := range rules {
		m := r.Meta()
		if m.Severity == SeverityOff {
			continue
		}
		if !r.Where(ctx) {
			res.Skipped = append(res.Skipped, m.ID)
			continue
		}
		fs, err := r.Eval(ctx)
		if err != nil {
			return Result{}, err
		}
		for _, f := range fs {
			// A finding must carry error or warn severity; anything else (empty,
			// off, a typo) would serialize yet count toward no exit code,
			// yielding a clean exit beside a non-empty findings list. That is a
			// rule bug, so fail closed rather than emit the contradiction.
			if f.Severity != SeverityError && f.Severity != SeverityWarn {
				return Result{}, fmt.Errorf("rule %q returned a finding with severity %q; findings must be %q or %q",
					m.ID, f.Severity, SeverityError, SeverityWarn)
			}
			// A finding inherits its rule's remediation metadata unless it set
			// its own, so every serialized finding is self-describing.
			if f.Fix == "" {
				f.Fix = m.Fix
			}
			if f.Policy == "" {
				f.Policy = m.PolicyInfo
			}
			res.Findings = append(res.Findings, f)
		}
	}
	sortFindings(res.Findings)
	sort.Strings(res.Skipped)

	for _, f := range res.Findings {
		switch f.Severity {
		case SeverityError:
			res.Blockers++
		case SeverityWarn:
			res.Warnings++
		}
	}
	res.ExitCode = exitCode(res.Blockers, res.Warnings)
	return res, nil
}

// exitCode is the Conftest tri-state: 2 if any error, else 1 if any warning,
// else 0. An error dominates a warning.
func exitCode(blockers, warnings int) int {
	switch {
	case blockers > 0:
		return 2
	case warnings > 0:
		return 1
	default:
		return 0
	}
}

// sortFindings orders findings by rule id, then file, then line — a stable order
// so JSON output and the human render are deterministic across runs.
func sortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].RuleID != fs[j].RuleID {
			return fs[i].RuleID < fs[j].RuleID
		}
		if fs[i].File != fs[j].File {
			return fs[i].File < fs[j].File
		}
		return fs[i].Line < fs[j].Line
	})
}

// targetRefusal is the finding bare `abcd lint` raises when one of its targets
// refuses rather than reports: the target checked nothing, so a warning, or
// worse a skip, would let the aggregate read clean over a check that never
// ran (iss-2610100649479892). It is error severity whatever the rule's own
// findings weigh, because the target alone exits 2 on the same refusal; the
// message names the target and carries the refusal's own words, and fix says
// where the next edit goes.
func targetRefusal(ruleID, target, file, refusal, fix string) Finding {
	return Finding{
		RuleID:   ruleID,
		Severity: SeverityError,
		File:     file,
		Message:  "`abcd lint " + target + "` refuses, so nothing in it was checked: " + refusal,
		Fix:      fix,
	}
}
