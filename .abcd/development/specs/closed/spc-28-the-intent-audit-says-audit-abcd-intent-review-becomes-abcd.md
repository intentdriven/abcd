---
id: spc-28
slug: the-intent-audit-says-audit-abcd-intent-review-becomes-abcd
intent: itd-123
---
# the-intent-audit-says-audit-abcd-intent-review-becomes-abcd

## Summary

Delivers adr-40's first named rename: `abcd intent review` → `abcd intent
audit` (and `review ingest` → `audit ingest`), the `intent-fidelity-reviewer`
agent → `intent-auditor`, the `intent_review` task-class token →
`intent_audit`. Clean break, no aliases. Ships only after spc-27's extended
`surface_coverage` is armed; the table row flips in the same change, so the
gate proves the migration complete.

## Scope — the live sweep (verified inventory, 2026-08-16)

Code and surface:

- `internal/core/intent/review.go` (+ `review_test.go`) → `audit.go` /
  `audit_test.go`; exported `Review*` identifiers → `Audit*`; the
  `review_emit_error` JSON key (`ReviewEmitError`, intent.go) →
  `audit_emit_error`; `create.go`'s references follow.
- `internal/surface/cli/cli.go` — the `review` sub-command under `intent` →
  `audit`; help text; JSON rendering.
- `agents/intent-fidelity-reviewer.md` → `agents/intent-auditor.md` (name,
  frontmatter, prompt body, `capability_scope.task_classes`); `agents/
  README.md` roster row.
- `commands/intent.md` — the Review/ingest section, `argument-hint`.
- `docs/reference/cli/commands.md` — regenerate.

Record (current-state surfaces only):

- Brief: `01-product/01-press-release.md`, `01-product/03-mental-model.md`,
  `02-constraints/04-naming.md` (the task-class enum row: `intent_review` →
  `intent_audit`), `04-surfaces/README.md`, `04-surfaces/05-intent.md`,
  `04-surfaces/06-capture.md`, `04-surfaces/09-reflect.md`,
  `05-internals/README.md`, `01-agents.md`, `02-adapters.md`,
  `03-configuration.md`, `04-universal-patterns.md`, `05-prompt-quality.md`,
  `06-lint.md`, `06-delivery/02-verification-matrix.md`.
- The `intent` sub-verb table row (spc-27's format): `review` → `audit`,
  same change.
- `CHANGELOG.md` — **breaking** entry.

**Not swept** (historical, append-only, or dated — AC 3): ADRs, dated
research notes, dated plans, resolved/wontfix issues, shipped/superseded
intents, `DECISIONS.md`, review-store artefacts, git history. A test or grep
in the PR description demonstrates remaining occurrences are all in these
classes. Amendment (2026-08-26): `intents/disciplines/` appeared in neither
the swept inventory nor this list and was swept late (iss-2608261437043634) —
it is live state with no reachable reconciliation event, since `intent audit`
refuses a non-shipped intent; `drafts/` and `planned/` stay deliberately
unswept under the iss-94 convention (each reconciles when next planned).
Stored artefact formats (`abcd-review:` markers, `review-<ts>` receipt
directories) keep their frozen spellings.

## Approach

1. **Precondition check first**: assert spc-27's sub-verb check is armed
   (the extended rule enabled and the `intent` table present). If not armed,
   STOP — the ordering is the plan's non-negotiable constraint.
2. Rename code path (`Audit`, `AuditIngest`, `audit_emit_error`), watching
   the renamed tests fail under the old spelling first, then pass.
3. Rename the agent file and roster; the task-class token in the agent
   frontmatter and the naming registry move together (the schema cross-check
   test in the binary must be updated in the same commit — the enum is
   PR-to-extend and no shipped check reads stored artefacts, so previously
   ingested verdicts stay valid; assert by re-running the ingest parser over
   an existing stored verdict fixture).
4. Sweep the brief and command surfaces; flip the sub-verb table row.
5. Regenerate `surface.json` (`go generate ./internal/surface/cli`) and the
   CLI reference; `record-lint`, `docs-lint`, and the surface drift test
   exit 0 — that is the proof of completeness.
6. Old spelling behaviour: `abcd intent review` exits as an unknown
   sub-command (cobra default); a surface test pins this.

## Acceptance-criteria mapping

- *New spellings, old gone* — steps 2, 6; surface tests for `audit`,
  `audit ingest`, and the unknown-command refusal of `review`.
- *Live sweep total* — the Scope inventory; step 5's gates prove it.
- *Historical records untouched* — the Not-swept list; PR shows the residual
  grep classified.
- *Armed-gate ordering* — step 1; the run stops if spc-27 is not armed.
- *Stored artefacts still parse* — step 3's fixture assertion.
- *Breaking CHANGELOG* — Scope; version derivation reads it.

Every behaviour change lands with a test watched fail before the change.
