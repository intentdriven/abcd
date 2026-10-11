---
id: spc-9
slug: an-intent-that-is-not-planned-cannot-be
intent: itd-94
---
# an-intent-that-is-not-planned-cannot-be-implemented-abcd-gai

## Summary

spc-9 delivers itd-94: a machine-checkable implement-readiness gate,
`abcd intent ready <itd-N>`, plus the plugin surface that routes an unready
intent to a human planning interview instead of an improvised implementation.

## Scope

- **Core** (`internal/core/intent/ready.go`): `Ready(repoRoot, intentID)`
  returns a structured `ReadyResult` — always exactly four checks in fixed
  order (`bucket`, `acceptance_criteria`, `spec_link`, `spec_body`), each with
  a detail and, on failure, the exact remedy. "Not ready" is a result;
  `error` is reserved for structural faults (malformed id, unknown intent,
  unreadable record). Read-only: the verb never mutates the store.
- **Supporting** (`internal/core/spec`): the minted placeholder is extracted
  into a shared `stubMarker` constant with an exported `BodyIsStub` detector,
  lockstep-tested so template and detector cannot drift.
- **CLI** (`internal/surface/cli`): `abcd intent ready <itd-N>` renders the
  report (with `--json` emitting the full `ReadyResult`) under the exit-code
  contract 0 ready / 1 not ready (report is the output, empty message) / 2
  fault — the embark-conflicts precedent.
- **Plugin surface** (`commands/abcd/intent.md`, resolving iss-105): the full
  verb family plus THE RULE (gate before implementing any itd-N) and the
  host-run planning interview, whose closing act — the human-confirmed
  `abcd intent plan` — is the maintainer's acceptance-criteria sign-off.
- **Protocol** (`.abcd/development/plans/2026-07-12-abcd-run-protocol.md`):
  step 0 for intent-backed items; nonzero exit is a journaled SKIP, and
  unattended planning is forbidden (the intent-level form of iss-83's
  fail-closed rule).

## Approach

The gate composes existing single-purpose guards into one reporter rather than
adding state: bucket is directory-as-truth, acceptance criteria reuse
`countAcceptanceCriteria` (the same parser `Plan` and the fidelity review
use), the link check mirrors `Reconcile`'s bidirectional-agreement guard as a
report, and the body check keys on the shared stub marker. AC sign-off is
implicit in a human-run `abcd intent plan` (DECISIONS.md 2026-07-18): no new
frontmatter, no forgeable schema; agents are barred from unattended planning
at the protocol layer.

## Acceptance-criteria satisfaction

- **Draft → NOT READY + remedy, exit 1** — `bucketCheck` fails every
  non-planned bucket; the drafts remedy names the confirm-then-plan route (or
  the interview when no AC bullets exist). Tests:
  `TestReadyDraftWithRealAC` (the itd-93 shape), `TestReadyDraftSeededPlaceholder`,
  CLI `TestIntentReadyNotReadyExit1`.
- **Stub spec body fails** — `specBodyCheck` + `spec.BodyIsStub`;
  `TestReadyPlannedStubSpecBody`, `TestBodyIsStubLockstep`.
- **Planned + linked + written passes, exit 0** — `TestReadyGreen`, CLI
  `TestIntentReadyGreenExit0`.
- **Faults exit 2** — `TestReadyFaults`, CLI `TestIntentReadyUnknownExit2`.
- **Surface refuses and offers the interview, never plans unattended** —
  `commands/abcd/intent.md` (THE RULE + interview steps 1–8 + Autonomous runs).
- **Run-protocol step-0 gate** — the skip/stop filter amendment in
  `2026-07-12-abcd-run-protocol.md`.
