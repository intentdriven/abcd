> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 6 — Lifeboat round-trip

## Expectation

By the end of this phase, abcd closes its own loop. A user can point
`/abcd:disembark` at **any** project — including one abcd has never managed, and
one that is already dead — and get a faithful lifeboat without the source repo
being touched: a structured snapshot carrying the project's verbatim artefacts
(specs, ADRs, memory, reviews) *and* the **recorded** why — distilled principles,
a composed brief, a press release, **with every claim citing the source it came
from**. Alongside it comes `coverage.{json,md}`: what could *not* be grounded,
what was searched for it, and the question a human must answer instead. A blank
is a first-class result, never a silent gap. They can run `/abcd:embark` to
unpack that lifeboat into a fresh target, and the new repo's brief, memory, and
ADRs are a faithful subset of the source.

This is the heart of abcd — the surface that makes a project survivable.

The fidelity promise and this phase’s sequencing follow
[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md): the
promise is the **recorded** why, every claim citing its source, and the
coverage experiment (itd-88) is sequenced outside this phase — see
`## Dependency rationale` below.

## Milestone

- `/abcd:disembark <source-repo> to <path>` runs end-to-end on each corpus repo,
  per the acceptance in `04-surfaces/02-disembark.md` — the source repo is read
  only and never written to, and the destination passes the safety gate (absent,
  empty, or carrying a parseable `_provenance.json`).
- The three preview shapes work: `probe <source-repo>` (adapter probes only,
  emitting the **coverage report** — that is the experiment, landed ahead of this
  phase by itd-88), `dry-run` (full plan, no writes), `<source-repo> to <path>
  --no-agents` (verbatim parts only).
- Pass A/B/C all pass their round-trip gates; Pass B retrieval reads the
  **native transcript store** (per
  [adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)) rather than an
  external recorder; each pass's oracle audit is opt-in (per
  [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)).
- `/abcd:embark from <path>` unpacks a lifeboat into an empty target per the
  acceptance in `04-surfaces/03-embark.md`; the round-trip test passes
  (disembark a corpus repo → embark into an empty target → faithful subset).
- Pass C consumes the curated memory substrate built in Phase 2 (itd-36); the
  disembark output is a curated single-repo artifact, not a promotion into a
  second repository (per
  [adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)).

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** a corpus repo abcd has never managed, **when** a user runs
  `/abcd:disembark <source-repo> to <path>`, **then** a lifeboat is produced at
  that destination — the source untouched — carrying both the verbatim artefacts
  AND the **recorded** why (distilled principles, composed brief, press release,
  **with every claim citing its source**), with `coverage.{json,md}` alongside it
  carrying the gaps: what could not be grounded, what was searched for it, and the
  question a human must answer. A journey across adapters, Pass A, Pass B, Pass C,
  and Phase 2's memory substrate that no single stage delivers alone.
- **Given** a project disembarked in this phase, **when** a user runs
  `/abcd:embark` into an empty target and then inspects it, **then** the new
  repo's brief, memory, and ADRs are a faithful subset of the source — the
  round-trip property that only exists because disembark and embark agree on one
  lifeboat schema.
- **Given** a produced lifeboat, **when** a reader with no prior context reads
  it, **then** they can reconstruct not just *what* the project is but **the
  recorded why, with every claim citing its source** — and every gap in that why
  is named in `coverage.{json,md}` as a question for a human, never left as a
  silent omission (per
  [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md), a
  transcript-less lifeboat carries the recorded why, never the unrecorded one).
  Pass B ships as a declared exemption in `_provenance.json`.
- **Given** a `dev-sync` source fails, **when** disembark runs, **then** the
  pipeline degrades gracefully with the failure named in the report rather than
  aborting — an end-to-end resilience property spanning every pass.

## Scope

**Intents:** itd-7 (RepoPrompt workspace pull — carries
`.abcd/rp/workspace.json` in the lifeboat for user-account migration, the opt-in
RepoPrompt-adapter case only; presets and routing are deferred to a later phase).

**Disembark and embark are one round-trip.** Disembark packs a lifeboat; embark
unpacks it into a fresh target. They share one lifeboat schema, and the phase's
milestone is the round-trip proving the schema faithful. The output is a curated
single-repo artifact (per adr-28) — there is no dev→public mirror and no sibling
repository to promote into.

**The memory substrate is Phase 2's.** Pass C's `principle-distiller` and
`press-release-composer` consume the curated substrate `/abcd:memory` built in
Phase 2 (itd-36); this phase draws on it rather than building it. Pass B's
retrieval reads the native transcript store (adr-29).

This is the most plumbing-heavy phase — settled-artefact adapters, Pass A spine
agents, Pass B targeted retrieval over the native transcript store, and Pass C
principles/compose/audit — plus the embark unpack half of the round-trip.

## Maps against

- **Brief:** `04-surfaces/02-disembark.md` (disembark);
  `04-surfaces/03-embark.md` (embark); `05-internals/02-adapters.md` (the
  adapters); `05-internals/01-agents.md` (the Pass A/B/C agents);
  `05-internals/07-memory.md` (Phase 2's itd-36 substrate this phase consumes);
  `06-delivery/01-build-sequence.md`.
- **Intents deliver the expectation:** itd-7 delivers the RepoPrompt-workspace
  portability that makes a lifeboat carry an opt-in RepoPrompt setup across
  machines; Phase 2's itd-36 supplies the memory the composed lifeboat draws its
  **recorded** why from — every claim citing the source it came from.
- **ADRs realised:** [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)
  (the lifeboat is a coverage experiment — read-only, out-of-tree, proven before
  it is packed; supersedes adr-4's regenerable-output model and moves `voyage/`
  to the operator level); adr-28 (single-repo curated release — the disembark
  output is a curated artifact, not a promotion); adr-29 (native transcript
  corpus — Pass B's source); adr-1's "phase audit" feedback loop becomes
  exercisable here once the pipeline produces auditable output.

## Dependency rationale

> Per [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md):
> the **coverage experiment (itd-88) lives outside this phase** and is
> separately sequenced; what remains here is the packer and the round-trip,
> built to whatever section list the experiment's cross-repo aggregate leaves
> standing. No prior phase gates this work.

- **Phases 3, 4 and 5 do not gate this work.** The native spec engine
  (`spec.Load` / `Create` / `Close` / `Validate`) ships; reviews are already
  committed markdown under `.abcd/work/reviews/`; backgrounding is a host
  affordance and abcd ships no `resume` verb by its own design. The
  host-delegation seam itd-2 was said to block on already ships **twice** —
  `memory.Distiller` fed by `--pages-json`, and `intent audit ingest
  --verdict-json` with a dead-letter path.
- **The one real dependency is data, not code.** Phase 2's history and memory
  *packages* are built; their **stores are empty**. Pass B — mining chat for the
  rationale nobody wrote down — has no corpus, and **cannot get one
  retroactively**. That is why the transcript-capture hook ships ahead of any
  lifeboat code, and why it is the only genuinely irreversible item on the board.
- **Runs after Phase 1** — disembark's `dev-sync` foundation and adapter dispatch
  build on the ahoy install flow and the rules loader.
- **Adapters degrade rather than block.** Tier 0 (git) is present in every
  repository, so a lifeboat is always producible; richer tiers raise coverage
  rather than gate the run. A missing source is a `blank` in the coverage
  report — a first-class result carrying the question a human must answer — not
  a failure.
- **Adapters → Pass A → Pass B → Pass C → embark** is a hard internal chain:
  each pass consumes the previous pass's validated output through a round-trip
  gate, and embark unpacks what Pass C composes.
- **itd-7 attaches at the RepoPrompt adapter** — it carries `workspace.json` only
  when the opt-in RepoPrompt adapter is in use; it is otherwise independent of
  the disembark/embark chain and can run in parallel.

## Open questions

- Pass B's signal-density gating threshold (15%) was set from early transcript
  sampling — confirm it holds against the full validation corpus, read from the
  native transcript store, before Pass B is considered done.
- Degraded-mode behaviour when `dev-sync` fails on a source: the brief specifies
  Pass C runs degraded with notes — verify this is exercised by a test, not just
  specified.
- itd-7 ships `workspace.json` only; presets and routing are explicitly
  deferred — confirm which later phase picks them up when this phase plans.
