---
id: itd-84
slug: intent-decomposition
kind: discipline
kind_notes: "Cross-cutting gate over /abcd:intent capture: a proposed intent is decomposed into its record homes and its interdependencies surfaced BEFORE it is filed, as an advisory analysis a human confirms. No user moment of its own. It is a verdict-rendering agent, so it inherits itd-81. Enforcement is STAGED per the promotion ladder: the principle + hand-run protocol + a deterministic Go pre-pass are the MVP rung (build first); the automated capture-time agent is the discipline rung (build after the protocol is calibrated). Adopted 2026-08-15 at the MVP rung: the decompose-before-filing principle and the hand-run protocol in the /abcd:intent surface page are live, the deterministic Go pre-pass is queued in the predictable-development plan, and the capture-time agent stays deferred until the protocol is calibrated (~50 graded captures)."
suggested_kind: null
spec_id: null
reclassification_history: []
blocked_by: [itd-1]
severity: major
---

# An Intent Is Decomposed Before It Is Filed, And A Reversal Is Only Ever Flagged

## Rule

At `/abcd:intent` capture, before an intent is filed, a validator produces an
**advisory** analysis a human confirms. One proposal is rarely one record;
capture routes the pieces, it never files a monolith.

1. **Decomposition.** Each part of the proposal is routed to its record home —
   user-facing capability → an intent; trust-boundary rule → an ADR (+ a brief
   invariant); standing stance → a principle; plumbing → the brief. The output
   names, per part, its type and its home.

2. **Interdependency.** The existing records the proposal touches, overlaps,
   duplicates, or reverses are surfaced — found by a **deterministic candidate
   pass** (a lexical shortlist over the record corpus), then reasoned over.
   Cross-record links are **typed** — `supersedes` / `reverses` / `duplicates` /
   `refines` — never a vague "related".

3. **A reversal is advisory-only.** "This reverses invariant X" is *flagged for a
   human to confirm* — never auto-classified, never auto-filed. Contradiction
   detection is unreliable even on frontier models and over-flags by default, so
   the most valuable check is the one that must never hold a gate shut on its own.

4. **The verdict has three outcomes, not pass/fail** — e.g. FILE-AS-IS / SPLIT /
   HOLD — with a **sample-size floor** (thin evidence never auto-routes), a
   **safety veto** (a hard structural violation forces HOLD regardless of score),
   and **windowing per `(target, prompt_version)`** so a prompt change cannot
   silently drift the calibration.

5. **Deterministic-first, semantic-host-delegated.** The always-on pre-pass is
   plain Go: a lexical shortlist plus an **atomicity smell** that flags a proposal
   naming several distinct capabilities as a SPLIT candidate — no model. The
   semantic decomposition and reversal reasoning are a host-delegated agent over
   the **small** candidate set. Embeddings are an optional adapter, never a
   runtime dependency.

6. **The validator is a verdict-rendering agent**, so it **inherits itd-81**:
   calibrated against a labelled corpus, a declared true-negative-rate floor,
   every finding carrying a failure scenario. It is independent of the proposer
   (evaluator-outside-the-loop); the human adopts the routing
   (verifier-selects-gates-decide).

**Build path (promotion ladder).** The `decompose-before-filing` principle, the
documented decomposition protocol (the hand-run four-piece table), and the
deterministic Go pre-pass are the **MVP rung — build first**. The automated
capture-time agent is the **discipline rung — build after** the protocol is
calibrated against ~50 real, human-graded captures. Until the agent ships the
gate is the documented protocol, announced as not-yet-automated (loud-staging).

## Why

The record information architecture ([adr-30](../../decisions/adrs/0030-record-information-architecture.md))
already supplies typed homes — intents, ADRs, principles, disciplines, the brief.
The failure mode is filing a monolith into one of them: the 2026-07-13 auto-merge
intent review found one "feature" (`--auto-merge`) was **four** record types —
an experience (intent), a trust rule (ADR + invariant), a stance (principle), and
plumbing (brief) — and only one was an intent. Naming the decomposition as a
capture-time gate makes that routing deliberate instead of accidental.

The mechanism is rooted in SOTA
([`../../research/notes/2026-07-13-intent-decomposition-sota.md`](../../research/notes/2026-07-13-intent-decomposition-sota.md)),
which bifurcates sharply and dictates the shape above:

- **Deterministic candidate-finding is mature.** Linear ships embedding+cosine
  similar-issue detection **advisory, at issue-creation** ([linear]); a plain-Go
  lexical shortlist is the always-on floor, embeddings the optional seam.
- **A deterministic pre-pass carries real load.** Paska hits 89%/89% on
  requirement smells with pure NLP ([paska]) — "is this several records?" is a
  cheap structural check, no LLM.
- **Semantic routing is usable-with-a-human.** Diátaxis classifiers already flag
  a blended artefact so it can be split — the decomposition behaviour, for a
  4-way taxonomy.
- **Contradiction/reversal reasoning is research-only** ([coinflip]; ContraDoc):
  unreliable even on GPT-4, over-flagging is the dominant failure — hence rule 3,
  advisory-only.
- **The advisory-gate pattern is adoptable wholesale** ([gate]): three outcomes,
  sample-size floor, safety veto, per-version windowing — rule 4.

**This subsumes the unbuilt `intent_sota` gate.** itd-29 gestures at a lint that
would enforce that every intent declares its SOTA (the `sota-per-intent`
principle). Enforcing SOTA-rooting is simply **one check this validator runs** —
the reminder that a design decision must be rooted in SOTA and the mechanism that
enforces it are the same family: capture-time automated analysis.

It also **embodies `facilitator-default-thinker-optional`** (landing in the
auto-merge groundwork change): the validator's default output is the facilitator
decomposition table + interdependency list; the press-release framing is the
optional layer, never the headline.

## What's In Scope

- The record-home taxonomy and the per-part routing.
- The deterministic Go pre-pass: lexical candidate-finder + atomicity smell.
- Typed cross-record links (`supersedes` / `reverses` / `duplicates` / `refines`).
- The advisory three-outcome verdict with a false-positive budget, sample-size
  floor, safety veto, and per-`(target, prompt_version)` windowing.
- SOTA-rooting as one check the validator runs (the `sota-per-intent` enforcement
  the unbuilt `intent_sota` gate wanted).

## What's Out of Scope

- **A bespoke contradiction/reversal detector as a reliable gate.** SOTA:
  research-only. It stays strictly advisory and human-confirmed.
- **Any blocking gate or auto-filing.** The verdict is a proposal; the human
  adopts it.
- **Reinventing vector storage or semantic similarity.** Lexical is the default;
  embeddings are an adapter to an existing service, not Go we write.
- **Formal RTM / traceability tooling, DSM/goal impact-analysis machinery,
  Controlled-Natural-Language authoring, and off-the-shelf NLP4RE libraries** —
  anti-fit or non-existent-as-a-drop-in (the field is 44.6% unlicensed).
- **Training or fine-tuning a classifier model** — a heavy dependency for a task
  an agent does zero-shot.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](itd-1-acceptance-gates.md)._

- **Given** a proposed intent at capture, **when** the validator runs, **then** it
  emits a per-part routing (each part → its record home), and nothing is filed
  until a human confirms it.
- **Given** a proposal naming several distinct capabilities, **when** the
  deterministic pre-pass runs, **then** the atomicity smell flags it as a SPLIT
  candidate — without invoking an LLM.
- **Given** a proposal that touches existing records, **when** the candidate-finder
  runs, **then** related records are surfaced by lexical shortlist and presented
  with typed-link suggestions.
- **Given** a possible reversal of an existing invariant, **when** the validator
  raises it, **then** the flag is advisory and requires human confirmation — never
  auto-classified, never auto-filed.
- **Given** the validator's verdict, **when** it renders, **then** it is one of
  three outcomes (not pass/fail), respects a sample-size floor, and a hard
  structural violation forces the negative outcome regardless of score.
- **Given** the validator is a verdict-rendering agent, **when** its spec closes,
  **then** it is calibrated per itd-81 (corpus, TNR floor, findings carry a
  failure scenario).

## Open Questions

- **Where the deterministic/semantic boundary sits.** How much the atomicity and
  overlap smells catch cheaply before the agent is invoked — resolve empirically
  once the pre-pass runs on real captures.
- **The taxonomy enum.** Is `discipline` a routing target distinct from
  `principle`? Where exactly does a trust rule split between an ADR and a brief
  invariant? Derive from graded real captures, not front-loaded.
- **Does the pre-pass run in `preflight`?** Leaning: the deterministic pre-pass is
  cheap enough for capture-time; the semantic agent is the expensive async tier
  (the itd-81 harness question, same shape).
- **Corpus construction.** Inject from our own history — past intents that should
  have been split (the auto-merge review is case one) — mirroring itd-81's
  harvest-from-`fix:`-commits recipe.

## Dependencies

- **Hard prerequisite:** [itd-1](itd-1-acceptance-gates.md) — this discipline's
  acceptance gates use its Given-When-Then shape.
- **Inherits:** [itd-81](itd-81-judge-calibration.md) — the validator renders
  verdicts, so it is calibrated (corpus, TNR floor, failure-scenario admissibility)
  under the judge-calibration discipline; it must never become a blocking gate.
- **Enforces:** the `sota-per-intent` principle — this validator subsumes the
  unbuilt `intent_sota` gate [itd-29](../superseded/itd-29-autonomous-run-resilience.md)
  gestures at.
- **Embodies:** the `facilitator-default-thinker-optional` principle (landing in
  the auto-merge groundwork change), plus `evaluator-outside-the-loop`,
  `verifier-selects-gates-decide`, and `script-first-mvp`.
- **Routes into:** [adr-30](../../decisions/adrs/0030-record-information-architecture.md)
  — the record IA is the taxonomy the decomposition routes each part into.
- **Related:** [itd-79](itd-79-persona-registry.md) — the two-role framing the
  facilitator/thinker principle generalises.

## Audit Notes

_Empty. Populated by `intent-auditor`'s single-document role when this
discipline is first audited. Like itd-1, itd-5, and itd-81, audited continuously
via rule-applies-to-every-capture semantics rather than a planned→shipped
transition._

## References

- [`../../research/notes/2026-07-13-intent-decomposition-sota.md`](../../research/notes/2026-07-13-intent-decomposition-sota.md)
  — the SOTA survey this discipline is drawn from, including its citation-confidence caveat.

[linear]: https://linear.app/now/using-ai-to-detect-similar-issues "Linear — Using AI to detect similar issues"
[paska]: https://arxiv.org/html/2305.07097 "Paska/Rimay — Automated Smell Detection in NL Requirements"
[gate]: https://vadim.blog/evidence-driven-release-gates-llm-sales-agents/ "Evidence-Driven Release Gates for LLM Agents"
[coinflip]: https://arxiv.org/pdf/2603.06594 "A Coin Flip for Safety — LLM Judges Fail to Reliably Measure"
