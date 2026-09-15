---
id: itd-156
slug: the-guard-tokenizer-does-not-perform-brace-expansion-so-a-fl
spec_id: spc-49
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2608221457227161
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

_Empty. Populated by intent-auditor when intent moves to shipped/._
