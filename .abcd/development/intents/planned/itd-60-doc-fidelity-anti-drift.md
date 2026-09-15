---
id: itd-60
slug: doc-fidelity-anti-drift
spec_id: spc-2609020903498198
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: [itd-73, itd-80]
related_adrs: []
prd_path: null
grill_session_id: 60d0f1de-0001-4a60-9c0d-000000000060
glossary_terms_used:
- core/brief
- core/intent
- core/spec
- core/oracle
grilled_intent_hash: bfaa672163edddb5d859bbdfc50169512349da38405c295729ce42c04b894401
prd_grandfathered: false
severity: major
impact: additive
---

# When A Surface Ships, The Brief Describes It — Or The Intent Does Not Reach Shipped

## Press Release

> _Sequenced 2026-09-01 as a delivery rung of
> [Phase 8 — the brief is the shipped state](../../roadmap/phases/phase-8-brief-currency.md);
> the phase records the sync rule and its one legitimate lead._


> **abcd gains a doc-fidelity gate: a change cannot call itself shipped while
> the brief lags the surface it delivered, and a release cannot be cut while any
> intent shipped since the last cut leaves its chapter behind.** The gate stands
> at both moments, deliberately. It runs in two layers. The deterministic layer
> needs no oracle and refuses on its own: every verb, every sub-verb and every
> agent the binary ships must have a brief chapter naming it. The semantic layer
> is host-delegated: it reads what the change actually delivered against that
> chapter and refuses on a confirmed false sentence, and where the reviewer
> cannot be reached it fails closed rather than reporting a clean brief. When the
> gate finds the brief lagging, the pass drafts the brief edit **and applies it**
> in the shipping change, then flags it for the maintainer to read — so the
> change is never held hostage to a paragraph, and the paragraph is never left
> unread.

> "I read the verdict, not the source," said a product thinker shipping with
> abcd. "If the brief is six specs behind the code, my whole picture of what I
> have built is wrong, and nothing tells me. I want the framework to refuse to
> call the work shipped until the brief says what is actually true — and to hand
> me the sentence it wrote so I am reviewing, not writing."

## Why This Matters

abcd's external honesty is part of its safety proposition: a product thinker who
cannot read code trusts the brief to tell them what they have built. The brief
is the shipped state (adr-5), and Phase 8 is what makes that decision
enforceable rather than aspirational — a product thinker opens the brief and
knows what abcd does today, without opening the command reference, the changelog
or the code.

Nothing enforces it today. abcd already grades one artefact against another with
discipline: the intent-fidelity reviewer grades delivery against acceptance
criteria, fails closed, and never infers a pass from absent evidence. The same
discipline applied to the brief closes the loop — built reality against the
chapter that claims to describe it. Without it, the framework that most
rigorously grades *delivery against intention* has no guard on *documentation
against delivery*, which is the one surface a non-expert actually reads.

The enforcement point is not arbitrary. `abcd spec close` is the verb that moves
a planned intent to `shipped/`, and it runs in the change that lands the work —
the one moment at which whoever made the change still has the surface in their
head. A gate anywhere later is archaeology.

This is the **forward** direction: built reality drives the brief. The
**reverse** direction — a human editing the brief, with the implied roadmap
changes drawn out — is a separate, paired intent
([[itd-61-brief-change-derivation]]).

## Decisions (grilled 2026-09-01)

The maintainer resolved the open questions at the planning interview; these are
commitments, not options.

- **The gate refuses at both moments.** An intent's move to shipped is refused
  while the brief lags the surface it delivered, and the release cut refuses
  again over every shipped intent. Belt and braces, deliberately.
- **Built reality is checked in two layers.** A deterministic layer refuses on
  its own with no oracle: every verb, sub-verb and agent the binary ships must
  have a brief chapter naming it. A semantic layer, host-delegated, reads what
  the intent delivered against the chapter and refuses on a confirmed false
  sentence; when the reviewer is unavailable the gate fails closed.
- **Brief only, for Phase 8.** A stale public-doc sentence is reported, never
  refuses; the public docs are a later rung.
- **Draft and apply, review after.** When drift is found the pass edits the
  brief in the shipping change and flags it for the maintainer's review; the
  brief may carry a sentence the maintainer did not write until they read it.
- **Sequenced as Phase 8**, the brief is the shipped state, with itd-147 as the
  other rung; standalone kind, its own spec.
- **The gate can run fully autonomously (maintainer, 2026-09-02).** A flag
  (the spec names it) lets an unattended run, an autonomous release cut for
  one, execute both layers without waiting for a person: the deterministic
  layer refuses on its own, the host-run reviewer is invoked by the routine
  the way the changelog composer is, the brief edit is drafted and applied,
  and every applied edit is listed for review after in the change the run
  produces. Autonomy changes who waits, never what refuses: the gate still
  refuses on what it can prove and still fails closed when the reviewer is
  unavailable.

## What's In Scope

- **Two layers, one gate.** The pass is two mechanisms stacked. Layer 1 grades
  *coverage* and needs nothing but the binary and the brief. Layer 2 grades
  *meaning* and is host-delegated. A refusal from either is a refusal.
- **Layer 1 — the deterministic coverage floor.** Every verb, every sub-verb and
  every agent the binary ships must have a brief chapter naming it. The check is
  derived from the command tree and the agent set, calls no oracle, and refuses
  on its own. It is cheap enough to run at every enforcement point and in CI.
- **Layer 2 — the semantic doc-fidelity pass.** Host-delegated review of what the
  change delivered against the chapter that claims to describe it, refusing on a
  **confirmed** false sentence and naming the sentence. Meaning cannot be judged
  deterministically, which is why this layer is delegated rather than coded.
- **Fail closed.** No reviewer, no reliable comparison, no verdict — the gate
  refuses. It never reports a clean brief from absent evidence.
- **Two enforcement points, one mechanism.** `abcd spec close`, the verb that
  moves a planned intent to `shipped/`, refuses per intent while the brief lags
  the surface that intent delivered; the release cut refuses again over every
  intent shipped since the last cut. Belt and braces, deliberately: the per-intent
  point catches the change while its author is present, and the cut catches
  whatever reached `main` around it.
- **Draft and apply, review after.** When the gate finds the brief lagging, the
  pass writes the brief edit into the shipping change itself and flags it for the
  maintainer's review. The brief may carry a sentence the maintainer did not
  write until they read it; that trade is the ruling, and the flag is what makes
  it honest.
- **The per-task pass stays a report.** After a task it reports where the brief
  may lag and blocks nothing. It is an advisory surface, not a gate; the gate is
  the refusal at the two enforcement points above.
- **The one legitimate lead is not drift.** A brief edited after a cut is ahead
  of that release until the next cut, and the gate does not report it as wrong
  for being ahead.

## What's Out of Scope

- **The public docs — a later rung, reported and never refusing.** For this
  phase the gate is **brief only**. A public-doc sentence that lags is reported,
  and the move and the cut proceed regardless. Everything the pre-ruling draft
  claimed here is struck to that later rung: grading the brief against the public
  docs, drafting audience-adapted public-doc deltas, and the end-user /
  developer-extending split. The brief has to hold before anything derived from
  it is worth gating.
- **The docs-currency lint that already shipped from this draft.** `abcd docs
  lint` — change-narration, resolvable cross-links, no stray root documents — is
  decidable without reading the code and runs everywhere already. It is the floor
  this gate stands on, not a layer of it, and this rung neither changes nor
  re-claims it.
- **The reverse direction.** Drawing implied intents and principles out of a
  *human-authored brief edit* is [[itd-61-brief-change-derivation]], a separate
  intent that stays adjacent to this phase.
- **Re-grading delivery against intention.** That is the intent-fidelity
  reviewer's job; this pass consumes "what shipped" as an input and does not
  re-derive it.
- **Authoring the brief's prose voice.** The pass drafts a delta against an
  existing chapter; wholesale authorship and information architecture are not its
  concern.
- **Deciding sequencing or dependencies** relative to the paired intents — that
  is `abcd intent plan`'s job.

## Mechanism

> _Facilitator-seeded from the phase's expectation. The maintainer stated no
> mechanism claim at the interview; strike this line and the sentence under it if
> it is not the claim being made._

We expect a product thinker to be able to read the shipped state from the brief
alone because the brief cannot fall behind a surface that has already shipped —
the gate refuses the shipped move and the release cut until the chapter names it
— and a shipped intent found later whose surface no chapter describes is what
would show the expectation wrong.

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** the binary ships a verb, a sub-verb or an agent that no brief chapter
  names, **when** the deterministic layer runs, **then** it refuses and names the
  undocumented surface, with no oracle call and no wait on a reviewer.
- **Given** a change that delivered a surface, **when** the intent moves to
  shipped through `abcd spec close`, **then** the semantic layer reads what the
  change delivered against that surface's chapter and refuses the move on a
  confirmed false sentence, naming the sentence.
- **Given** the semantic reviewer cannot be reached or returns no usable verdict,
  **when** either enforcement point runs, **then** the gate refuses, rather than
  reporting the brief clean.
- **Given** a release cut, **when** the gate runs over every intent shipped since
  the last cut, **then** the cut is refused while any of their chapters lags,
  naming the intent and the lagging sentence.
- **Given** the gate finds the brief lagging, **when** the shipping change is
  prepared, **then** the brief edit is drafted and applied in that same change
  and flagged for the maintainer's review, so no change lands with the brief left
  lagging and unflagged.
- **Given** a public-doc sentence that lags the brief, **when** either
  enforcement point runs, **then** it is reported and the move or the cut
  proceeds — in this rung the public docs never refuse.
- **Given** a task has completed, **when** the per-task pass runs, **then** it
  reports where the brief may lag, with evidence pointing at the divergence, and
  blocks nothing.
- **Given** a brief edited after a release was cut, **when** the next cut's gate
  runs, **then** the edit is treated as the legitimate lead and nothing between
  the cuts reported the brief as wrong for being ahead.
- **Given** the gate is invoked with its autonomous flag inside an unattended
  run, **when** it finds a lagging chapter, **then** it drafts and applies the
  edit, lists every applied edit in the run's own output for review after, and
  waits for nobody; it still refuses on an undocumented surface, on a confirmed
  false sentence, and when the reviewer is unavailable.

## Open Questions

> _Every question below was put to the maintainer at the planning interview on
> 2026-09-01 and answered there; the answers are the rulings in the Decisions
> section above. The section is kept so a later reader sees what was asked._

- **Resolved — where the pass hooks.** Two points, one mechanism: `abcd spec
  close`, which moves the intent to shipped, and the release cut. Not a
  composition with the intent-fidelity reviewer's surface; a distinct gate at
  both moments.
- **Resolved — what "built reality" is as a concrete input.** Two layers: the
  command tree and the agent set for the deterministic layer, and what the change
  delivered for the semantic layer.
- **Resolved — the developer-extending public-doc view.** Out of scope. The
  public docs as a whole are a later rung; neither audience view ships here.
- **Resolved — how the per-task tier avoids noise.** By not being a gate. The
  per-task pass is kept as a report only; the refusal lives at the shipped move
  and the cut, where a change has a surface to be judged against.
- **Open, and not gating scope** — whether this pass becomes a framework-provided
  discipline (see [[itd-62-pluggable-safety-gate]]'s two-discipline-kinds
  question) or stays a reviewer surface like fidelity. Deferred: the answer
  changes where the pass is configured, not what this rung builds.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Sequenced by
  [Phase 8 — the brief is the shipped state](../../roadmap/phases/phase-8-brief-currency.md),
  whose expectation this intent's mechanism claim is derived from, and whose
  milestone names this pass as the hard gate at the shipped move.
- Rests on adr-5: the brief is the current state of the project. This intent is
  what makes that decision enforceable.
- Originating assessment: `~/Desktop/abcd-assessment.html` (2026-06-26) — the
  README-over-promises / brief-honest finding that motivates the forward
  doc-fidelity loop. Its public-docs half is the later rung.
- Builds on itd-80, the intent-fidelity reviewer (delivery-against-intention):
  this pass is the documentation-against-delivery analogue, it consumes that
  reviewer's notion of what a change delivered, and it reuses its fail-closed,
  deterministic-shell, never-silent posture.
- Builds on itd-73, derived versioning, whose structural surface diff already
  runs at the release cut — the second of this gate's two enforcement points.
- Paired with: [[itd-61-brief-change-derivation]] (the reverse direction) and
  [[itd-62-pluggable-safety-gate]] (whose brief change this pass would govern);
  the other Phase 8 rung is
  [[itd-147-the-brief-s-surface-chapters-are-a-generated-reflection-of-t]].
- Governing principle: single source of truth — the brief is canonical
  (`.abcd/development/brief/` and the engineering conventions in `AGENTS.md`).

## Grounds

- pursued: the brief is the shipped state, so a product thinker reads the shipped state from the brief alone; a shipped intent whose surface the brief does not describe, found after the gate exists, shows the gate was wrong
