---
id: itd-128
slug: one-canonical-yaml-scalar-resolver
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# One canonical YAML scalar resolver: frontmatter exports the single null/bool/quote resolution helper and the capture, lint, lifeboat, and memory decoders all delegate to it, so a cross-package verdict split on any scalar axis becomes unrepresentable

## Press Release

Alice writes `impact: NULL` in an issue record and every abcd surface reads
it the same way: capture accepts what lint accepts, the release cut
diagnoses what lint diagnoses, and the lifeboat graveyard sees the same null
the memory dumper writes. There is exactly one place in the codebase that
knows what a YAML scalar means — an exported resolution helper in
`internal/core/frontmatter` covering nulls, booleans, and quoting — and the
capture, lint, lifeboat, and memory decoders all delegate to it. When Bob
extends the scalar rules (a new spelling, whitespace trimming), they edit one
function and every consumer moves together; the class of bug where two
hand-synced copies drift apart is no longer expressible.

## Why This Matters

The uppercase-null bug (iss-287) happened because a widening landed in some
predicate copies and not others; its fix (PR #294) widened two of four
independent scalar decoders and left the quoting axis split (iss-285) and
the boolean axis untouched (`memory/parseScalar` accepts true/True/TRUE
while `memory/vintage.go` tests `== "true"` exactly). Each remaining
asymmetry is a future issue of the same shape. This is the
one-canonical-primitive principle applied to scalar resolution: fix the
class, not the instance. Evidence:
`.abcd/work/reviews/2026-08-19-pr-294-null-predicate/` (F6).

## Acceptance Criteria

- Given a YAML scalar value, When any of the capture, lint, lifeboat, or
  memory decoders resolves its null/bool/quoting meaning, Then the answer
  comes from one exported frontmatter helper and no package-local literal
  set or hand-synced copy remains (grep proves zero duplicate null/bool
  tables outside the helper and its tests).
- Given the frontmatter package, When its scalar rules are widened in one
  edit, Then an equivalence/consumer test suite proves every delegating
  package observes the change without further edits.
- Given a record carrying a quoted null (`impact: "NULL"`), When capture
  validation and record-lint both judge it, Then their verdicts agree
  (closes iss-285), and the two currently-contradictory tests
  (`frontmatter_test.go` vs `graveyard_abandoned_test.go`) are reconciled to
  the same stance.

## Open Questions

- Where does quoting normalisation live — inside the shared helper (callers
  pass raw bytes) or as a documented pre-step every caller must apply? The
  helper-owns-it answer makes the drift unrepresentable; the pre-step answer
  preserves callers that need the raw form (lint's "not a record handle"
  message wants the original bytes).
- Does the write path (`memory/yaml.go` dumper's own null-literal list) fold
  into the same helper, or is emit a separate concern?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
