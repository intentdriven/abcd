---
id: itd-41
slug: phase-negotiator
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-10]
severity: minor
---

# abcd Tells You, Honestly, What a Phasing Choice Costs

## Press Release

> **abcd proposes how to phase your roadmap and shows you the real trade-offs before you commit.** Ask abcd to phase a set of intents and it works backwards from what each phase should make true, proposes an ordered set of phases, and then — instead of letting your wish-list go unchallenged — it interrogates the plan with you. Pull an intent forward and it asks which phase's expectation now slips; defer one and it names, from the dependency graph, exactly what else waits. Every cost it states is traced to a real dependency edge or a real phase-acceptance bullet — it never invents a trade-off to sound thorough. You still decide; abcd makes sure you decide knowing the price.
>
> "I always want everything in phase one," said Iris, product lead. "abcd asked me — move `itd-3` later and what in phase two stops working? I hadn't thought about the rules loader. It didn't lecture me; it asked the question the dependency graph already implied, and then it showed me the one phase-acceptance bullet I'd quietly broken. I re-scoped in five minutes, with my eyes open."

## Why This Matters

[adr-9](../../decisions/adrs/0009-phase-as-product-layer.md) made the **phase** abcd's sequencing layer — an ordered stretch of work, bundling intents and plumbing, ending in a milestone, with a product-authored `## Expectation` and `## Phase Acceptance`. But adr-9 gave sequencing a *home*, not a *negotiator*. Which intent goes in which phase, and in what order phases run, stays the product thinker's unchallenged judgement.

The product thinker holds the whole product in their head — and so they want everything, soon. abcd has adversarial readers at every other grain: `/abcd:intent grill` interrogates an intent before it is planned; dual-backend review stress-tests a spec; `intent-fidelity-reviewer` checks delivered reality against acceptance. The phase-planning grain has none. This intent adds it.

The danger is a fluent fake. An agent asked "what are the trade-offs?" will invent plausible ones, and a hallucinated trade-off spends the product thinker's trust on a fiction. So this intent's negotiator works the way `intent-fidelity-reviewer` and the grill skill work — **Socratic where it questions, grounded where it asserts** (per [adr-10](../../decisions/adrs/0010-phase-negotiator-grounded-tradeoffs.md)). It surfaces costs the product thinker can verify, and asks — rather than asserts — wherever it cannot ground the claim.

## What's In Scope

- **Phase proposal** — given a set of intents (and the relevant brief plumbing-phases), the negotiator works backwards from candidate `## Expectation`s and proposes an ordered set of phases, each with a draft `## Expectation`, `## Milestone`, and `## Phase Acceptance` for the product thinker to take, edit, or reject.
- **Trade-off discussion — Socratic interface.** The negotiator interrogates the proposed (or product-thinker-edited) phasing with questions, each tagged with a named Socratic move, the same taxonomy `/abcd:intent grill` uses. The product thinker's answers surface the trade-offs.
- **Grounded assertions only.** Where the negotiator states a cost rather than asking, the statement MUST cite a real anchor: a named edge in the dependency DAG (`brief/06-delivery/01-build-sequence.md` + intent/spec dependency declarations), a named `## Phase Acceptance` bullet, or a named `## Expectation` clause. A concern with no such anchor is raised as a question, never as an asserted trade-off.
- **Re-sequencing analysis** — the negotiator can take an existing phase plan and a proposed change (pull intent forward / defer / split a phase) and report, grounded in the DAG, what the change ripples into.
- **Advisory output** — the deliverable is a proposal plus a recorded discussion, not a decision. The product thinker owns the final `## Expectation` and `## Phase Acceptance` per adr-9.

## What's Out of Scope

- **Deciding the phase plan** — the negotiator never commits a phasing. It proposes and discusses; the product thinker decides and authors the canonical phase docs.
- **Ungrounded trade-offs** — if a cost cannot be tied to a DAG edge or an acceptance/expectation clause, it is not asserted. "Sounds thorough" is not a licence to invent.
- **Replacing the grill skill** — grill operates on a single intent pre-plan; the negotiator operates on phases and their ordering. Shared Socratic lineage, different surface (per adr-10 alternative 4).
- **Estimating effort or dates** — the negotiator reasons about dependencies and acceptance, not durations. No time estimates (per the workspace no-estimates rule).
- **Auto-authoring phase docs** — it drafts proposals; writing the canonical `roadmap/phases/phase-N-*.md` files is the product thinker's act.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** a set of intents and the relevant plumbing-phases, **when** the product thinker asks abcd to phase them, **then** abcd proposes an ordered set of phases, each with a draft `## Expectation` and `## Phase Acceptance`, and presents it as a proposal to edit or reject — not as a committed plan.
- **Given** a proposed phasing, **when** the negotiator raises a trade-off as an assertion, **then** the assertion cites a specific dependency-DAG edge, `## Phase Acceptance` bullet, or `## Expectation` clause; an assertion with no such citation is a defect.
- **Given** a concern the negotiator cannot ground in a DAG edge or an acceptance/expectation clause, **when** it surfaces that concern, **then** it is phrased as a Socratic question tagged with a named move — never as an asserted trade-off.
- **Given** the product thinker proposes pulling an intent into an earlier phase, **when** the negotiator analyses the change, **then** it names, from the dependency graph, every item whose dependency chain is affected, and names which `## Phase Acceptance` bullet of the affected phase is put at risk.
- **Given** the negotiator has finished, **when** the product thinker reviews the output, **then** the phase plan is unchanged on disk until the product thinker authors it — the negotiator's output is advisory.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [traceability-governance SOTA note](../../research/notes/2026-10-23-traceability-governance-sota.md).

- **The record this draft rests on is superseded.** adr-9 (phases) gave way to adr-2609212115255771, which retires the phase, the milestone and the word roadmap: "Sequencing is dependencies plus the shelves", Now/Next/Later "is a rendered status, never stored", and the derived release is the checkpoint. The phase plan the negotiator proposes no longer exists in the record; the re-sequencing analysis over the dependency graph does. Nothing ruled here is contradicted by the research; the plan-time interview owes the product thinker this fact first.
- **The field's sequencing conversations are about what a move costs, not about phases.** Shape Up bets "only one cycle ahead", drops unchosen pitches, revives an idea only when "brought back with a context, by a person, with a purpose", and holds that "There is nothing special about bugs that makes them automatically more important than everything else". Now-Next-Later exists because "Putting features in dated columns turns every date into a commitment".
- **Cost of delay is the one quantitative "what does waiting cost"**, and it needs a duration per item; the no-estimates rule excludes that term, and the relative job size SAFe substitutes is the part its critics reject. Without durations it degenerates to severity weighted by what waits behind an item, which is itd-78's derived order.
- **Story mapping is the "work backwards from what each slice makes true" instrument** without phases: a backbone that "just is", a walking skeleton built first, release slices drawn across the map so "we never release a car without brakes" (Patton).
- **Grounded-or-asked is a shipped shape:** Spec Kit's `analyze` gives every finding a location and a severity and offers remediation the user approves; a cost with no location is the defect this draft names.
- **Path 2, a proposed re-scope for the plan-time interview to confirm** (the product thinker has not ruled it, and the superseded-record fact above comes first in that interview): native, shaped to cost of delay's question without its numbers. Given a proposed move (pull forward, hold, defer, mark to revisit), the asserted costs come from itd-78's transitive closure and the affected intents' acceptance criteria, each with a location; everything else is a tagged Socratic question, as adr-10 holds. The seam is itd-78's ordering function, which the negotiator reads and never writes; the product thinker's own order (itd-2610070550516046) and the revisit mark (itd-2610070549060530) are the "phasing choice" it now interrogates.

The independent fit-challenge runs at plan time.

## Open Questions

- Surface shape — is this a sub-verb of a phase-oriented command, a sub-verb under `/abcd:intent` (alongside `grill`), or its own command? adr-10 leaves the surface to this intent; it needs deciding at plan time.
- Should the negotiator's discussion be persisted (a logbook artefact, like grill reports) so a phase doc can cite the negotiation that shaped it?
- How does the negotiator behave when the dependency DAG is prose-only for some area — degrade to pure Socratic questioning (adr-10's stated limit), or prompt the product thinker to formalise that DAG slice first?
- Does the negotiator share a Socratic-move taxonomy module with the grill skill, or keep its own copy until the two demonstrably converge?
- (SOTA 2026-10-23) With phases retired (adr-2609212115255771), what the negotiator negotiates: a proposed move in the product thinker's order, a revisit mark, or a hold; and whether the draft is re-scoped, superseded by a narrower one, or folded into itd-78's report.
- (SOTA 2026-10-23) Whether the retired words (phase, milestone, roadmap) are replaced throughout the draft at plan time or the draft is rewritten against the rendered Now/Next/Later.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Shares the grounded-adversary pattern with: `itd-42` (coherence-aware grill) — both are *Socratic where they question, grounded where they assert*; a concern with no anchor is asked, never asserted.
