> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 3 — Intent, brief, and review

## Expectation

By the end of this phase, a user can take a vague intent and harden it into a
spec-ready one, and the second front door — MCP — is live. `/abcd:intent`
manages the intent lifecycle; `/abcd:intent grill` runs a two-phase Socratic
challenge — interrogating the intent for vagueness, then silently synthesising a
Pocock-shaped PRD frozen at promotion — and emits an emergent glossary. The
review methodology runs through the **host-delegated oracle** (per
[adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)): plan- and
impl-review are oracle passes, with RepoPrompt available as an opt-in backend,
and their output lands as durable, spec-tied review artefacts in the native
store. The **MCP front door** opens here (per
[adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)), so the same core
that backs the CLI is reachable by an MCP client — and by the companion harness as a peer
(per [adr-24](../../decisions/adrs/0024-companion-harness-peer-via-conventions-and-mcp.md)).
This is the phase where an idea stops being a sentence and becomes something the
native spec engine (Phase 4) can turn into a spec.

This phase is the **end of intent authoring**, not the start of implementation.
Grill's job is to make the hand-off to the spec engine clean: a grilled intent
is one the planner can consume without re-interrogating it.

## Milestone

- `/abcd:intent grill <itd-N>` runs both phases per `04-surfaces/05-intent.md`:
  Phase 1 Socratic vagueness interrogation, Phase 2 silent PRD synthesis. The
  PRD is frozen at `.abcd/intents/<itd-N>/prd.md` on promotion.
- The emergent glossary is written under
  `.abcd/development/brief/glossary/` (the bounded-context glossary home, per
  [adr-30](../../decisions/adrs/0030-record-information-architecture.md));
  grill's lint codes (GL001–GL005, GR001–GR005) are live as Go lints.
- The coherence-aware grill tiers work: Tier 2 (brief-coherence) and Tier 3
  (sibling-intent index) run; light vs. full grill is selected by lifecycle
  position.
- Review passes run through the host-delegated oracle and land spec-tied in the
  native review store with a `review.json` sidecar and two-stage redaction, per
  `04-surfaces/05-intent.md`. RepoPrompt is an opt-in oracle backend, not a
  requirement.
- The MCP front door is enabled: `internal/core`'s intent, brief, and review
  operations are reachable over MCP as well as the CLI (per adr-23), which is
  the surface the companion harness composes against as a peer (per adr-24).

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** a vague draft intent, **when** a user runs `/abcd:intent grill`
  and then `/abcd:intent plan`, **then** the intent reaches `planned/` carrying
  a frozen PRD, a populated glossary, and acceptance criteria that survive the
  Phase 0 itd-1 gate — a journey across itd-27, itd-42, and the Phase 0
  disciplines that no single intent delivers alone.
- **Given** a grilled intent whose PRD is frozen, **when** the native spec
  engine (Phase 4) consumes it, **then** the planner has enough specificity to
  produce a spec without re-interrogating the user — the emergent "clean
  hand-off" property that is the whole point of the phase.
- **Given** a plan-review or impl-review runs on a spec, **when** it produces
  review output, **then** that output lands spec-tied in the native review store
  with redaction applied — itd-28 making the review trail a durable, pinned
  artefact rather than transient chat scrollback, whichever oracle backend
  produced it.
- **Given** the MCP front door is enabled, **when** an MCP client (or the companion harness as
  a peer) calls an intent or review operation, **then** it reaches the same
  `internal/core` the CLI does and gets the same result — the transport-agnostic
  property adr-23 guarantees.

## Scope

**Intents:** itd-27 (`/abcd:intent grill` sub-verb + emergent glossary —
two-phase Socratic challenger producing a frozen Pocock PRD), itd-42
(coherence-aware grill — Tier 2 brief-coherence and Tier 3 sibling-intent
index, light vs. full grill by lifecycle), itd-28 (spec-tied reviews land in the
native review store — the review artefacts the review methodology produces need
a durable home).

**Review methodology via the oracle.** Plan- and impl-review are oracle passes
run through the host-delegated default; RepoPrompt is an opt-in oracle backend.
Their output is post-processed by itd-28 into spec-tied artefacts in the native
store — the review trail is engine-neutral and does not depend on any external
review tool being installed.

**The MCP front door opens here.** Phase 1 shipped the CLI front door; this phase
enables the MCP front door over the same core (adr-23), which is what makes
the companion harness a composable peer rather than a code dependency (adr-24).

**Why grill is here and not in an "implementation" phase.** Grill's Phase 2
output is a PRD *frozen at intent promotion* — the last step of intent
authoring, the thing that turns a vague intent into a spec-ready one. The build
itself is the native spec engine and its optional ccpm backend (Phase 4), which
consumes the grilled intent. There is no separate abcd "implementation phase";
there is this intent-hardening phase, and grill is its centrepiece.

**Brief plumbing-phases:** the `/abcd:intent` surface flow, covered by
`04-surfaces/05-intent.md`. The probe-only bare `/abcd:intent` render stub from
Phase 1 is joined here by the real `grill` sub-verb.

## Maps against

- **Brief:** `04-surfaces/05-intent.md` (the intent surface, grill, the
  intent-auditor's roles); `05-internals/06-lint.md` (the GL/GR lint
  families); `05-internals/08-skills.md` (the abcdGrill skill).
- **Intents deliver the expectation:** itd-27 delivers the grill sub-verb and
  glossary; itd-42 delivers the coherence-aware tiers; itd-28 delivers the
  spec-tied review trail the phase's reviews write into.
- **ADRs realised:** adr-9 (phase-as-product-layer — grill's PRD and the phase's
  `## Expectation` mirror at intent grain); adr-23 (transport-agnostic core — the
  MCP front door); adr-24 (the companion harness as a peer over MCP); adr-25 (host-delegated
  oracle — the review methodology's backend); adr-8 (dual-backend review — the
  review artefacts itd-28 lands).

## Dependency rationale

- **Runs after Phase 0** — every spec planned in this phase inherits the Phase 0
  disciplines, plugs into the oracle seam, and grill's own acceptance criteria
  must pass the itd-1 gate. Grill is in part an *application* of the
  acceptance-gate discipline at authoring time, so the discipline must already
  be in force.
- **Runs after Phase 1** — `/abcd:intent` is a command; a command needs the ahoy
  install flow and the rules loader live before its sub-verbs are wired, and the
  MCP front door builds on the same core the CLI front door already exposes.
- **itd-42 depends on itd-27** — itd-27 lands the `/abcd:intent grill` sub-verb
  and the glossary tier; itd-42 extends that grill with the coherence-aware
  tiers (and renames `--with-docs` to `--glossary`). Both intents declare this
  direction (itd-42 `blocked_by: [itd-27]`; itd-27 "Extended by: itd-42").
  itd-27 also depends on itd-1 (its own acceptance criteria pass the gate).
- **itd-28 depends on the oracle seam** (Phase 0's itd-6) — spec-tied reviews are
  post-processed from what the oracle emits; itd-28 lands them in the native
  store. It is grouped here because the reviews this phase runs are the first
  heavy producers of review artefacts that need a durable home.
- **Before Phase 4** — the native spec engine consumes grilled intents; intent
  hardening must precede the engine that turns them into specs.

## Open questions

- Confirm the native review store's on-disk shape is settled here so Phase 4's
  spec engine and Phase 6's disembark can both read review artefacts without a
  later migration.
- itd-27 must be planned (and likely grilled) before itd-42's spec can depend on
  it. Confirm the itd-27 → itd-42 ordering holds once both have specs.
