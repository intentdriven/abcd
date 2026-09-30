---
id: itd-97
slug: the-facilitator-is-a-mode-not-a-person-abcd-runs-duo-with-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817]
severity: major
---

# The Facilitator Is a Mode, Not a Person

## Press Release

> _Facilitator-seeded draft (2026-07-25 role-ownership review) — the product
> thinker owns the final press-release framing._

> **abcd runs solo or duo: the facilitator is a configuration, not a hire.** A
> project chooses **duo** — a human facilitator working alongside the product
> thinker, as today — or **solo**, where abcd itself performs the facilitator's
> work: shaping intents into plans, curating the backlog, firing reviews,
> triaging captures, and deciding *when* each of those happens. The gates do
> not move: solo and duo run the identical deterministic gates and the
> identical product-thinker decision points; the only thing that changes is
> who operates the machinery between them.
>
> "I don't have a facilitator — and it turned out I didn't need to find one,"
> said Alice, a product thinker. "abcd drafted the plan from my intent, queued
> the work, ran the reviews, and every time a decision was genuinely mine it
> arrived as a clear proposal with the price spelled out. Bob still prefers
> duo, with a human translator beside them — same framework, same gates, their
> choice."

## Why This Matters

The README already records the ambition ("in a later version, abcd aims to
offer an automated facilitator") but nothing owns it. The facilitator's
**what** has shipped kernels — the planning interview, `intent ready`, the
run protocol, the fidelity reviewer — with completions planned (itd-27,
itd-50, itd-53, itd-58, itd-78, itd-82). The facilitator's **when** —
acting unprompted at the right moment — exists only as drafts: the schedule
(itd-13), self-firing reviews (itd-83), stage-aware defaults (itd-19), and
the coordination/escalation contract (itd-33). Solo mode is the named
composition of those parts behind one switch; making it first-class is also
the precondition for testing solo against duo at all (itd-98).

The role boundary is already doctrine and does not change with the mode:
steps owned by the product thinker (adoption, adjudication, dependency
sign-off, irreversible actions) stay human in both modes
(verifier-selects-gates-decide); solo automates only facilitator-owned work.

## What's In Scope

- **A per-repo mode declaration** (duo | solo) with duo the default; the
  status board and every run surface name the active mode — no ambient
  ambiguity about who the facilitator is.
- **Solo = the composed "when"**: scheduled sync, self-firing reviews,
  capture triage, backlog curation, and run initiation composed behind the
  mode switch, over the pluggable run seam (itd-29).
- **Duo unchanged**: the same machinery surfaces queued facilitator work to
  the human instead of acting; nothing is removed from today's flow.
- **Gate parity**: mode changes an operator, never a gate — the deterministic
  gates, review requirements, and STOP conditions are byte-identical across
  modes.
- **The escalation contract**: solo stops at every product-thinker decision
  point with a proposal (options, recommendation, price), exactly where duo
  would hand over.

## What's Out of Scope

- **The solo-vs-duo evaluation** — itd-98 owns the measurement.
- **A team of product thinkers** — itd-99 owns plural thinkers; this intent
  assumes one.
- **Any weakening of human gates in solo** — `--dangerously-unattended`-style
  skips are the run protocol's concern and never cross the product-thinker
  line.
- **Model routing and credit contingency** — operational policy, owned by the
  run PLAN contract.

## Acceptance Criteria

- **Given** a repo configured solo, **when** a new capture lands in the
  ledger, **then** triage runs without a human initiating it, and any routing
  that requires judgement lands as a proposal for the product thinker — never
  a silent action.
- **Given** a repo configured duo, **when** the same events occur, **then**
  abcd surfaces the queued facilitator work to the human facilitator and
  performs none of it unprompted.
- **Given** either mode, **when** the same change is driven to a PR, **then**
  the deterministic gates, reviews, and STOP conditions applied are
  identical (a diff of the gate configuration between modes is empty).
- **Given** solo mode reaches a decision reserved for the product thinker
  (adoption, adjudication, new dependency, irreversible action), **when** it
  gets there, **then** it stops with a clear proposal and recorded reasoning
  and does not proceed until adoption.
- **Given** a session starts in either mode, **when** the status board
  renders, **then** it names the active mode.

## Open Questions

- **Where does the mode live?** A `.abcd/` config key, or a value of the
  itd-19 stage taxonomy (whose Autonomous/Collaborative axis is the nearest
  existing vocabulary)?
- **v1 initiative scope** — which "when" triggers compose into the first solo
  cut (triage and reviews look smallest; scheduling has the itd-13
  launchd/cron design)?
- **Relationship to itd-29's operator verbs** — is `pause` in solo mode
  equivalent to a temporary drop to duo?
- **Does solo v1 require the run seam**, or can it operate in-session
  (host-delegated) first, per script-first MVP?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- README § Roles — the recorded automated-facilitator ambition this intent
  gives an owner.
- itd-29 (run seam operator surface, planned); itd-13, itd-19, itd-33,
  itd-83 (the "when" drafts this composes).
- `.abcd/development/principles/verifier-selects-gates-decide.md` — the role
  boundary both modes preserve.
- itd-98 (the evaluation), itd-99 (plural thinkers) — the sibling drafts
  filed with this one.
