---
id: itd-24
slug: reflect-command
spec_id: null
kind: bundle-member
bundle: spc-83-operator-surfaces
suggested_kind: null
reclassification_history: []
glossary_terms_used: [core/phase, core/intent, core/voyage, core/persona, core/brief, core/lifeboat, core/oracle, core/spec, interview/embark, distribution/release]
grill_session_id: e6a24d86-e133-495f-8dec-94dec21449ea
grilled_at: 2026-05-16T15:37:17Z
grilled_intent_hash: 8412a59b575df882fc4a370ab01404796cad4dd9e120d0519e9918d3ea891c61
prd_path: null
prd_grandfathered: true
severity: minor
builds_on: [itd-27]
---

# Completed Phases Get A Retrospective

## Press Release

> **abcd ships `/abcd:reflect` for phase retrospectives.** Run `/abcd:reflect phase-1-substrate` and abcd walks an interview-driven retrospective: what went well, what could improve, lessons learned, decisions made, metrics. The interview is *seeded* by the phase audit — abcd reads the phase-fidelity-reviewer's per-bullet verdicts and opens the conversation from what actually passed and failed. Output is a structured `.abcd/retrospectives/phase-1-substrate/README.md` — committed as part of the phase's permanent record. Future lifeboats carry the retrospective forward; future intents reference past lessons. Reflection becomes a first-class abcd primitive, not an afterthought.
>
> "abcd's brief and intents captured *what* I'd done," said Henry, a junior-developer persona. "Reflect captures *what I learned* — and because it starts from the phase audit, it doesn't ask me to re-remember the work, it asks me what the verdicts *mean*. When I started a new voyage six months later, embark surfaced past retrospectives in the lifeboat unpack — the lessons came with the work. I didn't re-make the mistakes."

## Why This Matters

abcd ships strong post-implementation transparency: shipped intents have audit notes (per `itd-1` acceptance criteria); the native spec store's completion records capture what was built; and per [adr-9](../../decisions/adrs/0009-phase-as-product-layer.md) a completed phase gets a **phase audit** — the phase-fidelity-reviewer comparing delivered reality against the phase's `## Phase Acceptance`. What's missing is **post-phase reflection** — the structured "what did we learn" doc that's bigger than per-intent audit notes, bigger than a pass/fail audit verdict, and smaller than a brief rewrite.

The legacy `~/.claude/templates/retrospective.md.template` had the right prompt structure (what went well, what could improve, lessons learned, decisions made, metrics) but lived as a manual template that rarely got used. It was deferred (see [`research/legacy-harvest.md`](../../research/legacy-harvest.md) Pass 4 retrospective decision); `/abcd:reflect` promotes it to a first-class command with structured interview + structured output.

`/abcd:reflect` is the *phase-level* reflection surface — broader than per-intent audit notes (which `intent-fidelity-reviewer`'s Role 1 already produces on ship per the itd-1 discipline) and narrower than a brief rewrite. It spans the intents and plumbing a phase bundled and captures what's transferable to future voyages.

A phase audit and a phase retrospective are **distinct activities**, the same split `intent-fidelity-reviewer` Role 1 draws at the intent grain — one grain up. The audit asks *did the phase's `## Phase Acceptance` pass* (a per-bullet verdict). The retrospective asks *what did we learn* (transferable insight, interview-driven). `/abcd:reflect` does not replace the audit — it **consumes** it: the audit's verdicts are the seed material the retrospective interview opens from.

## What's In Scope

- **`/abcd:reflect <phase-id>`** — retrospective for a completed phase, and the command's *only* argument form. Examples: `/abcd:reflect phase-1-substrate`, `/abcd:reflect phase-2-ahoy`. Per-intent reflection is out of scope (see below) — `/abcd:reflect` operates at the phase grain only.
- **Audit-seeded interview.** `/abcd:reflect` reads the phase-fidelity-reviewer's per-bullet `## Phase Acceptance` verdicts and opens the interview from them — the seed grounds the conversation in delivered reality rather than starting from a blank prompt.
- **Audit-missing handling.** If `/abcd:reflect <phase-id>` is run before the phase audit exists, the command detects the gap and **offers to run the phase-fidelity-reviewer inline** first, then continues into the retrospective.
- **Empty-phase refusal.** If no spec carries the named phase's `phase:` anchor, the command refuses — there is no delivered work to reflect on.
- **Interview-driven structure**:
  - What went well (successes and strengths, with specific examples)
  - What could improve (issues and gaps, ranked)
  - Lessons learned (transferable insights, framed for future-you)
  - Decisions made (architectural / design choices crystallised during the phase)
  - Metrics (intents shipped, audit-note severity distribution, time-to-ship if measurable)
- **Output**: `.abcd/retrospectives/<phase-id>/README.md` — a peer of `.abcd/intents/` and `.abcd/logbook/`, committed as part of the phase's permanent record.
- **Lifeboat integration**: `/abcd:disembark to <path>` packs *all* of the voyage's phase retrospectives into the lifeboat — the full reflection arc travels. `/abcd:embark from <path>` surfaces predecessor retrospectives during the press-release interview ("here's what the previous voyage learned about X — does that apply here?").
- **Reference back to intents and the audit**: the retrospective links to the phase doc, to the intents the phase bundled, and to the phase audit; per-intent reviewer notes are referenced (not duplicated).
- **`reflection-composer` agent** — runs the interview, drafts the structured output, asks clarifying questions when answers feel thin.

## What's Out of Scope

- **The phase audit itself** — comparing delivered reality against `## Phase Acceptance` is the phase-fidelity-reviewer's job (per adr-9). `/abcd:reflect` consumes that verdict; it does not produce it.
- **Per-intent reflection** — `intent-fidelity-reviewer`'s Role 1 already produces per-criterion verdicts and a three-bucket prose audit on every shipped intent (per the itd-1 discipline). That *is* per-intent reflection; `/abcd:reflect` does not duplicate it. There is no `/abcd:reflect <itd-N>` form — the command takes a phase ID only.
- **Quantitative retrospective metrics** (DORA, velocity, etc.) — abcd doesn't gather the underlying telemetry. Metrics section is qualitative + simple counts only.
- **Team retrospectives** — abcd is single-developer-shaped (or pair-shaped); team retrospective patterns belong elsewhere.
- **Automatic triggering** — reflection requires the persona's deliberate engagement; no auto-prompting after phase close.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an abcd repo with a completed phase (`phase-1-substrate.md` exists, every spec carrying `phase: phase-1-substrate` is closed, and a phase audit has been recorded), **when** the persona runs `/abcd:reflect phase-1-substrate`, **then** the reflection-composer agent runs an interview *seeded by the phase audit's per-bullet verdicts* and writes `.abcd/retrospectives/phase-1-substrate/README.md` with all five required sections populated.
- **Given** a completed phase with no phase audit yet recorded, **when** the persona runs `/abcd:reflect <phase-id>`, **then** the command reports the missing audit and offers to run the phase-fidelity-reviewer inline before continuing into the retrospective.
- **Given** a phase doc that exists but has no spec carrying its `phase:` anchor, **when** the persona runs `/abcd:reflect <phase-id>`, **then** the command refuses with "no specs anchored to `<phase-id>` — nothing shipped to reflect on" and writes no output.
- **Given** a draft retrospective with thin answers (e.g. "what went well: it worked"), **when** the agent drafts the output, **then** the agent surfaces the thinness as a clarifying question rather than committing the thin answer.
- **Given** the same repo's lifeboat is then packed via `/abcd:disembark to <path>`, **when** the lifeboat is inspected, **then** every `.abcd/retrospectives/<phase-id>/README.md` the voyage produced is included in the lifeboat artefact.
- **Given** a target repo embarked from a lifeboat that includes retrospectives, **when** `/abcd:embark from <path>` runs the press-release interview, **then** the persona is shown predecessor retrospective lessons and asked which apply to the new voyage.
- **Given** an attempt to reflect on a phase whose specs are not all closed, **when** `/abcd:reflect <phase-id>` runs, **then** the command warns the persona, lists the open specs anchored to that phase, and asks for confirmation to proceed anyway.

## Open Questions

- **Reflection cadence** — is there a soft prompt to encourage running it (e.g. when the last spec of a phase closes), or fully on-demand?
- **Lifeboat surfacing on embark** — how intrusive? The lifeboat carries all phase retrospectives; how should embark present them — show every one, or rank by relevance to the new voyage? Risk of "previous-voyage lessons" feeling like noise on a brand-new voyage.

## Blocking Dependency

`/abcd:reflect` **cannot be planned until the phase-fidelity-reviewer ships** and its output artefact is stable and machine-readable. The reviewer is deferred in adr-9. Because `/abcd:reflect`'s core design is to *consume* the audit's per-bullet verdicts as interview seed material, and the audit-missing AC depends on running the reviewer inline, the command has no buildable contract until the reviewer's output format exists. `/abcd:intent plan itd-24` must not proceed while the phase-fidelity-reviewer remains unbuilt.

**Satisfied:** the stable machine-readable phase-fidelity output is provided by spc-66 (`phase_review_report.schema.json` + `.abcd/logbook/audit/phase-<ts>/report.{json,md}`), so this dependency is met and `/abcd:reflect` is planned under spc-83. V1 keys empty-phase detection off the spc-66 receipts (not a `phase:` anchor) and refuses on a missing/empty-audited receipt rather than offering the inline reviewer — see the `### Implementation notes (spc-83.3 — v1 scope)` block below.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Implementation notes (v1 scope)

`/abcd:reflect` (thin V1) refines two acceptance behaviours from
their originally-drafted form; both are recorded here so the fidelity review
reads them as deliberate v1 scope, not gaps:

- **Missing-audit inline-reviewer offer → refusal (deferred).** The draft AC
  had the command "offer to run the phase-fidelity-reviewer inline" when no
  audit exists. V1 instead **refuses** when no spc-66 phase-audit receipt exists
  for the named phase (and when the latest matching receipt is empty-audited).
  Empty-phase detection keys off the spc-66 receipts, not a `phase:` spec anchor
  (that anchor is deferred; phase membership is editorial). Running the reviewer
  inline from reflect is a recorded future extension.
- **Open-spec warn/confirm (deferred).** The draft AC had reflect warn and ask
  for confirmation when a phase's specs are not all closed. V1 does not parse
  editorial Scope membership, so this warn/confirm path is deferred; the refusal
  semantics above are the v1 gate.
- **Phase-only grain, source links, lifeboat.** `/abcd:reflect <itd-N>` is
  refused (phase-only grain). V1 links to the phase doc + audit report + member
  specs only (no intent links — the spc-66 receipt carries no intent ids;
  recorded future extension). The lifeboat-packs-all-retrospectives requirement
  is a DOCUMENTED forward requirement on the future disembark spec (spc-17 stubs),
  not a behaviour this surface implements — recorded in the surface doc.

The interview is a single seeded pass (per-bullet verdicts → five questions);
multi-turn depth is a recorded future extension. Full surface record:
[`../../brief/04-surfaces/09-reflect.md`](../../brief/04-surfaces/09-reflect.md).

### Linkage note (spc-83.5)

Ships as one of FOUR intents sharing spec
`spc-83-operator-surfaces-manifest-lockstep`. abcd represents "N intents, one
spec" as a bundle (`kind: bundle-member` + shared `bundle: spc-83-operator-surfaces`)
— the representation the doc_fidelity intent-resolution + spec-close preflight
require. Bundle member by delivery relationship, not a scope change. This intent
keeps its real grill linkage (`grill_session_id`); GR002 is handled via
`prd_grandfathered`. Full record in the spec's process-exception note.
