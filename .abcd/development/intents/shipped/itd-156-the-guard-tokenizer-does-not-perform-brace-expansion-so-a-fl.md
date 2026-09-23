---
id: itd-156
shipped_in: v0.6.8
slug: the-guard-tokenizer-does-not-perform-brace-expansion-so-a-fl
spec_id: spc-49
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608221457227161
impact: fix
---

# The guard tokenizer does not perform brace expansion, so a flag wrapped in a single-element brace group with an empty alternative (git push {--force,} origin main) expands in bash to byte-identical argv --force yet the guard reads the literal token {--force,} and allows it — a Tier-1 blocker miss of the same mutate-the-flag-token shape as the round-6 redirection fix. Distinct from the $'...' quoting gap (this is expansion, not quoting, and breaks no written invariant); recorded for a scoped follow-up because a correct bounded brace-expander is larger than this round's scope.

## Press Release

> _Seeded by promotion from iss-2608221457227161. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2608221457227161`: The guard tokenizer does not perform brace expansion, so a flag wrapped in a single-element brace group with an empty alternative (git push {--force,} origin main) expands in bash to byte-identical argv --force yet the guard reads the literal token {--force,} and allows it — a Tier-1 blocker miss of the same mutate-the-flag-token shape as the round-6 redirection fix. Distinct from the $'...' quoting gap (this is expansion, not quoting, and breaks no written invariant); recorded for a scoped follow-up because a correct bounded brace-expander is larger than this round's scope.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** the command `git push {--force,} origin main`, whose unquoted brace group expands in bash to byte-identical `--force` argv, **when** the guard tokenizes it, **then** the guard refuses the command fail-closed rather than reading the literal token `{--force,}` and allowing it.
- **Given** a brace sequence enclosed in quotes (for example `'{--force,}'`), **when** the guard evaluates the command, **then** it is not treated as a brace expression and is not false-positived.
- **Given** a `${VAR}` parameter expansion in a command, **when** the guard evaluates it, **then** it is not mistaken for a brace group to refuse.
- **Given** an ordinary command that contains no unquoted brace group, **when** the guard evaluates it, **then** its verdict is unaffected by the new brace handling.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-d43ef5189c11 -->
Fidelity review — receipt rcp-d43ef5189c11 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:6e04539356aa74de0d8aada73ec5e5e55e9729aea165c4fe0120befc2de9585e
Input attestations: diff:328a6755^1..328a6755 (PR #555), judged against the tree at bad1c73e@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: a structural unquoted `{` that braceExpansionAt identifies as a group marks the segment, Check folds that into a VerdictBlock signal, and the tests assert Block on the reported shape and that the refusal is fail-closed on the hook path
  evidence: internal/core/guard/tokenize.go:452 — "case c == '{' && braceExpansionAt(line, i, &braceBudget):"
  evidence: internal/core/guard/guard.go:395 — "if s.braceGroup {"
  evidence: internal/core/guard/brace_test.go:16 — "func TestUnquotedBraceGroupIsRefused"
  evidence: internal/core/guard/brace_test.go:73 — "func TestBraceRefusalIsFailClosed"
- ac-2 — MET: quoted bytes never reach the structural switch case, and the shape table asserts `'{--force,}'` and `"{--force,}"` are allowed
  evidence: internal/core/guard/tokenize.go:588 — "braceExpansionAt reports whether the `{` at line[i] — reached as a structural,"
  evidence: internal/core/guard/brace_test.go:110 — "{`git push '{--force,}' origin main`, VerdictAllow},"
- ac-3 — MET: an unescaped `$` immediately before the brace exempts it as parameter expansion, and `${HOME}`, `${x:-a,b}` and `${MSG}` are asserted to keep their allow verdict
  evidence: internal/core/guard/tokenize.go:615 — "if i > 0 && line[i-1] == '$' && !escapedAt(line, i-1) {"
  evidence: internal/core/guard/brace_test.go:114 — "{`echo ${HOME}`, VerdictAllow},"
- ac-4 — MET: a lone `{`, `{a}`, `{}`, `awk {print}`, ordinary commands and a reserved-word `{ …; }` group all keep their prior verdicts in the shape table, and the brace bytes stay in the word so nothing else about tokenisation changes
  evidence: internal/core/guard/brace_test.go:103 — "func TestBraceHandlingLeavesEveryOtherShapeAlone"
  evidence: internal/core/guard/brace_test.go:136 — "{`{ git push --force origin main; }`, VerdictBlock},"
  evidence: internal/core/guard/tokenize.go:473 — "cur = append(cur, c)"

Gap audit:
- honoured:
  - refuse rather than expand: the block rides on the segment so the pre-tool-use hook blocks instead of failing open on a tokenize error
    evidence: internal/core/guard/tokenize.go:467 — "The refusal rides on the segment rather than returning"
    evidence: internal/core/guard/tokenize.go:576 — "func braceExpansionBlockSignal() payloadSignal {"
  - the look-ahead is budgeted so the scan stays linear however many braces a line holds
    evidence: internal/core/guard/tokenize.go:568 — "braceScanBudget = 1 << 16"
    evidence: internal/core/guard/brace_test.go:182 — "func TestBraceScanStaysLinear"
  - variants beyond the reported shape are covered: ranges, nested groups, escaped `$`, substitutions inside an alternative, an inner `}` in the first alternative
    evidence: internal/core/guard/brace_test.go:26 — "`rm -rf dir{1..9}`,"
    evidence: internal/core/guard/brace_test.go:44 — "`git push \${--force,} origin main`,"
- diverged: (none)
- missing: (none)