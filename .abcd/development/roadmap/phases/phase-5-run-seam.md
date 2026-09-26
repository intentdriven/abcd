> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 5 — Autonomous run seam

## Expectation

By the end of this phase, abcd can drive work autonomously through a **pluggable
run seam** rather than a single hard-wired loop. A user can hand abcd a set of
planned specs and have them worked to completion unattended, and the engine
behind that run is a choice: a host **Workflows** backend, the **companion harness's loop**,
or abcd's own **native loop** as the always-available fallback (per
[adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)). The seam
presents one contract — take specs, make progress, report honestly, stop on a
declared condition — and the backend behind it is swappable. This is the phase
where abcd runs itself forward without a human in the loop, on whichever run
engine a project chooses.

## Milestone

- The run adapter seam exposes one contract, with the **native loop** as the
  default backend that runs with no external run engine installed (per adr-27
  and [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)).
- A run drives planned specs (Phase 4's native spec/task engine) to completion,
  recording progress and stopping on a declared STOP condition rather than
  pushing through it.
- The Workflows and companion-harness-loop backends attach at the seam as opt-in
  alternatives: selecting one changes the run engine without changing the
  user-facing surface.
- A run's output is auditable — each pass's progress is recorded so a
  host-delegated oracle audit can run over it when opted in (per
  [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)).

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** a set of planned specs, **when** a user starts an autonomous run with
  no external run engine installed, **then** the native loop drives the specs to
  completion and stops on a declared STOP condition — the run seam is whole on
  its own backend, per adr-27.
- **Given** a project that selects the Workflows or companion-harness-loop backend, **when**
  the same run starts, **then** it drives the same specs through the chosen
  engine with the same user-facing surface — the swappable-backend property
  adr-27 guarantees.
- **Given** a completed run, **when** an oracle audit is opted in, **then** the
  recorded per-pass progress is auditable end to end — an emergent honesty
  property spanning the whole run.

## Scope

**Autonomous run seam** (per adr-27): the run adapter contract, the native loop
as the default backend, and the Workflows and companion-harness-loop backends as opt-in
alternatives at the seam. The seam is engine-neutral by construction — no single
run engine is a hard dependency, and the run is not a port of any one upstream
loop.

**Native loop is the floor.** The default backend is abcd's own loop, so a
project can run autonomously with nothing external installed. Workflows and the
companion harness's loop are deepenings a project opts into.

**STOP conditions are first-class.** A run declares its STOP conditions up front;
hitting one stops the run and reports, rather than pushing through — the same
discipline the playbook applies to human-driven work.

## Maps against

- **Brief:** `04-surfaces/05-intent.md` (the ship/run surface);
  `05-internals/01-agents.md` (the run agents);
  `06-delivery/01-build-sequence.md`.
- **Intents deliver the expectation:** the run seam drives Phase 4's native
  spec/task engine to completion; it has no dedicated intent of its own and is
  substrate work at the run adapter seam.
- **ADRs realised:** adr-27 (autonomous run as a pluggable seam, not a Ralph
  port); adr-22 (run backends as pluggable adapters); adr-25 (host-delegated
  oracle for the opt-in run audit).

## Dependency rationale

- **Runs after Phase 4** — the run seam drives planned specs to completion; the
  native spec/task engine those specs live in must exist first.
- **Runs after Phase 0** — the run adapter seam is Phase 0 substrate; the native
  loop backend plugs into it.
- **Before Phase 6** — a survivable project needs its work driven forward before
  the lifeboat round-trip captures the result; the run seam is the last engine
  substrate the lifeboat depends on being native.
- **Backends attach, they do not gate** — the native loop is whole on its own;
  Workflows and the companion harness's loop can land in parallel and be wired when a
  project opts into them.

## Open questions

- Confirm the run seam's STOP-condition contract is uniform across the native,
  Workflows, and companion-harness-loop backends, so a STOP means the same thing whichever
  engine runs.
- Confirm the recorded per-pass progress shape is what the opt-in oracle audit
  reads, so no run-specific audit adapter is needed.
