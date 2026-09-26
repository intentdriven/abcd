> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 0 — Foundations

## Expectation

By the end of this phase, abcd has a Go substrate an honest account of its own
state can be built on. The Go binary scaffold is in place; `internal/core` holds
the transport-agnostic engine and `internal/adapter/*` holds the seam skeletons
— oracle, history, spec, run, scanner — each with a working **native default**
so the tool functions with no external tool installed. The plugin surface is
scaffolded so later phases have somewhere to register commands. The three
cross-cutting disciplines are in force, so every spec planned after this point
inherits Given-When-Then acceptance, prompt-quality gates, and a
modification-grammar section from the moment it is written — no retrofitting.
The three-intent-kinds taxonomy is codified, and the spec-terminology is settled
before any further doc is written against another word.

This phase ships **no product-capability user moment** — no press-release-worthy
capability a user reaches for — and that is why it is numbered 0. It is the
floor the capability phases stand on: the core the front doors call, the
adapters the later stores plug into, the disciplines the later specs are
measured against, and the vocabulary every later doc is written in. Numbering it
0 rather than 1 states honestly that no *product capability* lands here; the
first such moment is `/abcd:ahoy` and `/abcd:launch` in Phase 1. The qualifier
is "product capability", not "user-typed surface": the `/abcd:intent audit`
discipline-audit *sub-verb* is a substrate/maintenance surface — the substrate
inspecting itself, not a product capability. So Phase 0 does carry a user-typed
verb; what it does not carry is a product-capability user moment.

## Milestone

- The Go binary scaffold builds and runs: `internal/core` compiles as the
  transport-agnostic engine (per [adr-21](../../decisions/adrs/0021-rebuild-in-go.md)
  and [adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)), with the
  CLI as its first thin front door.
- The adapter seams exist under `internal/adapter/*` — oracle, history, spec,
  run, scanner — each a skeleton with a **native default** that runs with no
  external tool present (per [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)).
  External tools (a transcript recorder, RepoPrompt, a ccpm backend, a workflow
  engine, codex) attach at these seams as opt-in adapters, never as hard
  dependencies.
- The oracle seam's native default is **host-delegated** — the LLM is provided
  by the host that runs abcd; RepoPrompt and codex are opt-in oracle adapters,
  not a required cascade (per [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)).
- The plugin surface is scaffolded: the command registry and dispatch skeleton
  exist so Phase 1's `ahoy`/`launch` and later surfaces have a place to land.
- The phase's three disciplines (`itd-1`, `itd-5`, `itd-37`) are registered as
  active gates in `disciplines/` (later phases add more), and `itd-34`'s three-kinds taxonomy is the schema
  they are registered under.
- The `itd-43` spec-terminology is settled: schema, reviews, and prose use
  "spec", with `terminology/core/spec.md` canonical. No half-renamed state
  remains in the corpus.

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** the three disciplines are registered, **when** any spec is planned
  in a later phase, **then** that spec inherits all three discipline
  gates at once — Given-When-Then acceptance, `prompt_version`, and a
  `## Modification Grammar` section — with no per-spec retrofit. (Emergent: the
  *every later spec is born correctly-shaped* guarantee is a property of the
  three disciplines being in force together, owned by no single discipline.)
- **Given** a discipline gate has a mechanical half (section/field presence,
  well-formedness) and a judgement half (is the criterion *actually met*? is
  the Modification Grammar boilerplate?), **when** Phase 0 closes, **then** the
  mechanical half is hard-enforced by the Go lints (the `IL`/`MG`/`VR` lint
  families, `prompt_version`/`capability_scope` checks) and the judgement half
  is formalised by the dedicated `intent-auditor`, run through the
  host-delegated oracle. Before the reviewer lands, the judgement half is
  covered by the oracle's plan-review and impl-review passes; once it ships,
  "the disciplines are in force" is true at full strength — **lint-enforced AND
  judgement-enforced by the dedicated reviewer**.
- **Given** the adapter seams carry native defaults, **when** any core code path
  runs with no external tool installed, **then** it resolves through the native
  default and never hard-fails for lack of a backend — the no-external-tool-as-
  hard-dependency guarantee that is a property of the seam design, owned by no
  single adapter.
- **Given** the `itd-43` terminology and the `itd-34` taxonomy have both landed,
  **when** any later phase's spec or intent is authored, **then** it is written
  in the settled vocabulary (one kind from three; the settled term) from the
  first draft — no later phase inherits a terminology migration.

## Scope

**Intents:** itd-6 (oracle adapter — RepoPrompt is one opt-in oracle backend
behind the seam; no user moment, so it lives in Phase 0), itd-1 (acceptance
gates — discipline), itd-5 (prompt-quality additions — discipline), itd-37
(modification grammar — discipline), itd-34 (three intent kinds — the taxonomy
the disciplines are registered under), itd-43 (spec-terminology).

**Substrate work** (no intent): the Go binary scaffold, `internal/core`, the
`internal/adapter/*` seam skeletons with their native defaults, and the plugin
surface scaffolding. This is the arch-neutral floor the disciplines are enforced
on and the later stores plug into.

The `intent-auditor` ships as the last Phase 0 spec (see the
discipline-gate Phase Acceptance bullet above). It runs through the
host-delegated oracle and also exposes the manual `/abcd:intent audit` surface.

## Maps against

- **Brief:** `06-delivery/01-build-sequence.md` (Phase 0 foundation, itd-1/5/6
  in the execution order); `05-internals/01-agents.md` (oracle backend
  resolution); `05-internals/05-prompt-quality.md` (itd-5's home);
  `01-product/03-mental-model.md` (itd-34's three-kinds taxonomy).
- **Intents deliver the expectation:** itd-6 delivers the oracle adapter seam;
  itd-1/5/37 deliver the discipline gates that make every later spec auditable;
  itd-34 and itd-43 settle the taxonomy and vocabulary every later phase writes
  in.
- **ADRs realised:** adr-21 (rebuild in Go); adr-22 (bundled deps as pluggable
  adapters); adr-23 (transport-agnostic core); adr-25 (host-delegated LLM);
  adr-8 (dual-backend review, exercised by the oracle's plan review).

## Dependency rationale

- **The Go core and adapter seams first** — every later phase's store (history,
  spec, run) plugs into a seam skeleton, and every front door calls
  `internal/core`. The substrate must compile and expose its seams before any
  capability phase builds on them.
- **Disciplines before Phase 1** — disciplines impose inherited acceptance
  gates on every *other* spec. Registering them before any later capability
  phase's surface spec is planned means those specs land correctly-shaped from
  day one. This is the single most important ordering constraint in the whole
  plan, and the reason this work is a phase of its own rather than folded into
  the install phase.
- **itd-43 before the corpus grows** — a terminology settlement is cheapest
  while the corpus is small; every later phase's docs are written in the
  settled vocabulary, so it must precede them.

## Open questions

- The `phase:` spec anchor is a native-store field validated by the `PA001`
  verify-exists lint (a `phase:` naming a non-existent phase is an error; a
  missing anchor stays legal). The valid-phase set is derived live from the
  phase docs, so the seven phases 0–6 need no manual update. What stays deferred
  is the corpus anchor *backfill* (making `phase:` a standing convention) — a
  separate planning act.
- `IL011` (the cross-phase bundle check) is a plan-time check rather than a
  frontmatter-field lint, since phase membership lives editorially in phase
  docs. Confirm the planner has access to target-phase context when `IL011` is
  eventually implemented.
- Confirm itd-34 still ships *with* the disciplines (its taxonomy is exercised
  by them) rather than as a separable unit.
