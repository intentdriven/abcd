> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 4 — Native spec and task engine

## Expectation

By the end of this phase, abcd owns its spec and task layer. A grilled intent
flows through `intent → plan → ship`: `/abcd:intent plan` turns a frozen PRD
into a **native minimal spec** in the native store, and `/abcd:intent ship`
drives the build against that spec. The engine is native by default — a small,
self-contained spec/task model with no external planner required — and the companion harness's
`ccpm` is the **primary deeper backend** when a project wants richer task
management (per
[adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)). Status —
"what is planned, what is in flight, what is done" — is read live from the native
store, and from ccpm when that backend is attached. This is the phase where an
intent stops being a spec-ready PRD and becomes tracked, executable work.

## Milestone

- `/abcd:intent plan <itd-N>` writes a native minimal spec into the native spec
  store from the intent's frozen PRD, with the discipline-inherited
  Given-When-Then acceptance carried through (per adr-26).
- `/abcd:intent ship` drives the build against a planned spec and records task
  progress in the native store.
- The native store reports true per-spec, per-task status live — no
  hand-maintained counts (the stale-proof pattern the roadmap dashboard reads).
- The companion harness `ccpm` adapter is wired as the opt-in deeper backend: when
  attached, plan/ship route through ccpm and status reflects it; when absent,
  the native engine handles the whole flow (per adr-26 and
  [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)).

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** a grilled intent with a frozen PRD, **when** a user runs
  `/abcd:intent plan` and then `/abcd:intent ship`, **then** the intent becomes a
  native spec that is planned, worked, and closed against the native store — the
  whole intent→plan→ship spine walkable with no external planner installed.
- **Given** a project with the `ccpm` backend attached, **when** the same
  plan→ship flow runs, **then** it routes through ccpm and status reflects the
  richer backend — the pluggable-deeper-backend property adr-26 guarantees,
  without changing the user-facing verbs.
- **Given** a spec is planned and worked, **when** a contributor asks "what is
  planned and what is done", **then** the native store answers live — the
  stale-proof status property every later dashboard read depends on.

## Scope

**Native spec/task engine** (per adr-26): the minimal native spec model, the
`intent → plan → ship` flow over it, and the companion harness `ccpm` adapter as the
primary deeper backend. The engine is engine-neutral in the sense that no
external planner is a hard dependency; ccpm attaches at the spec adapter seam.

**The plan and ship sub-verbs.** `/abcd:intent plan` and `/abcd:intent ship`
are the hand-off from Phase 3's intent authoring into tracked work. Plan
consumes the frozen PRD; ship drives the build. Both read and write the native
spec store, or route to ccpm when that backend is present.

**No separate planner surface.** There is no external plan/work tool in the
critical path — the spec engine is abcd's own, with ccpm as an optional
deepening rather than a dependency.

## Maps against

- **Brief:** `04-surfaces/05-intent.md` (the `plan`/`ship` sub-verbs);
  `05-internals/01-agents.md` (the plan and ship agents);
  `06-delivery/01-build-sequence.md`.
- **Intents deliver the expectation:** itd-27's grilled-intent PRD is the input
  the spec engine consumes; the plan/ship flow turns it into tracked work.
- **ADRs realised:** adr-26 (native minimal spec layer with ccpm as the primary
  deeper backend); adr-22 (ccpm as a pluggable adapter); adr-24 (the companion harness as a
  peer — ccpm is composed, not depended on).

## Dependency rationale

- **Runs after Phase 3** — the spec engine consumes grilled intents with frozen
  PRDs; intent hardening must precede the engine that turns them into specs.
- **Runs after Phase 0** — the spec adapter seam and the native store the engine
  writes into are Phase 0 substrate; the disciplines the spec inherits are in
  force from Phase 0.
- **Before Phase 5** — the autonomous run seam drives specs to completion; the
  native spec/task model it runs against must exist first.
- **ccpm attaches, it does not gate** — the native engine is whole on its own;
  the ccpm adapter can land in parallel and be wired when a project opts into
  the deeper backend.

## Open questions

- Confirm the native spec store's schema is a superset the ccpm adapter can
  round-trip, so attaching or detaching ccpm does not lose task state.
- Confirm the `intent → plan → ship` verbs read exactly the frozen-PRD shape
  Phase 3 produces, so no PRD re-shaping is needed at plan time.
