---
id: itd-24
slug: reflect-command
spec_id: spc-2609211751376504
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
builds_on: []
impact: additive
---

# Completed Releases Get A Retrospective

## Press Release

> **abcd ships `/abcd:reflect` for release retrospectives.** Run `/abcd:reflect v0.11.0` and abcd walks an interview-driven retrospective: what went well, what could improve, lessons learned, decisions made, metrics. The interview is *seeded* by what the release shipped — the intents whose `shipped_in` names the tag, each with the audit notes the intent auditor wrote on it, and the changelog section the cut composed — so the conversation opens from what actually passed and failed. Output is a structured `.abcd/development/retrospectives/v0.11.0/README.md`, committed as part of the permanent record. Future lifeboats carry the retrospective forward; future intents reference past lessons. Reflection becomes a first-class abcd primitive, not an afterthought.
>
> "abcd's brief and intents captured *what* I'd done," said Henry, a junior-developer persona. "Reflect captures *what I learned* — and because it starts from what the release shipped and how each audit went, it doesn't ask me to re-remember the work, it asks me what the verdicts *mean*. When I started a new voyage six months later, embark surfaced past retrospectives in the lifeboat unpack — the lessons came with the work. I didn't re-make the mistakes."

## Why This Matters

abcd ships strong post-implementation transparency: shipped intents have audit notes (per `itd-1` acceptance criteria); the native spec store's completion records capture what was built; and each release cut composes a changelog from the records that shipped in it. What's missing is **post-release reflection** — the structured "what did we learn" doc that's bigger than per-intent audit notes, bigger than a pass/fail audit verdict, and smaller than a brief rewrite.

The legacy `~/.claude/templates/retrospective.md.template` had the right prompt structure (what went well, what could improve, lessons learned, decisions made, metrics) but lived as a manual template that rarely got used. It was deferred (see [`research/legacy-harvest.md`](../../research/legacy-harvest.md) Pass 4 retrospective decision); `/abcd:reflect` promotes it to a first-class command with structured interview + structured output.

`/abcd:reflect` is the *release-level* reflection surface — broader than per-intent audit notes (which the intent auditor already produces on ship per the itd-1 discipline) and narrower than a brief rewrite. It spans the intents a release shipped and captures what's transferable to future voyages. It was first written at the grain of a roadmap phase (adr-9); phases are retired (adr-2609212115255771), and the release is the unit that replaced them (decisions 4 and 5).

An audit and a retrospective are **distinct activities**. The audit asks *did each intent deliver its criteria* (a per-criterion verdict). The retrospective asks *what did we learn* (transferable insight, interview-driven). `/abcd:reflect` does not replace the audit — it **consumes** it: the release's audit verdicts are the seed material the retrospective interview opens from.

## Mechanism

We expect the value of a retrospective to be in the lifeboat: a new project that starts from an old one's lessons avoids a repeat, because the lessons arrive with the work rather than being re-derived; shown wrong if projects embarked from a lifeboat carrying retrospectives never open a surfaced lesson.

## What's In Scope

- **`/abcd:reflect <release-tag>`** — retrospective for a cut release, and the command's *only* argument form. Examples: `/abcd:reflect v0.10.0`, `/abcd:reflect v0.11.1`. Per-intent reflection is out of scope (see below) — `/abcd:reflect` operates at the release grain only.
- **Seeded from the release** (decision 5). The seed is the intents a tag shipped (`shipped_in` names the tag), with their `## Audit Notes` (per-criterion verdicts, honoured / diverged / missing), their `impact`, and the changelog section the cut composed; the interview opens from them rather than from a blank prompt.
- **Audit-missing handling.** A shipped intent in the release with no audit notes is named, and the command offers `abcd intent audit <itd-N>` for it first; the interview continues either way.
- **Empty-release refusal.** A tag whose release shipped no intent refuses — there is no delivered work to reflect on.
- **Interview-driven structure**:
  - What went well (successes and strengths, with specific examples)
  - What could improve (issues and gaps, ranked)
  - Lessons learned (transferable insights, framed for future-you)
  - Decisions made (architectural / design choices crystallised during the release)
  - Metrics (intents shipped, audit-note severity distribution, time-to-ship if measurable)
- **Output**: `.abcd/development/retrospectives/<release-tag>/README.md` — a peer of `.abcd/development/intents/`, committed as part of the permanent record.
- **Lifeboat integration**: `/abcd:disembark pack <repo> <path>` packs *all* of the voyage's retrospectives into the lifeboat — the full reflection arc travels. `/abcd:embark from <path>` surfaces predecessor retrospectives during the press-release interview ("here's what the previous voyage learned about X — does that apply here?").
- **Reference back to intents and the audit**: the retrospective links to the release's changelog section, to the intents it shipped, and to their audit notes; the audit notes are referenced, not duplicated.
- **`reflection-composer` agent** — runs the interview, drafts the structured output, asks clarifying questions when answers feel thin.

## What's Out of Scope

- **The audit itself** — judging delivered reality against each intent's criteria is the intent auditor's job. `/abcd:reflect` consumes those verdicts; it does not produce them.
- **Per-intent reflection** — the intent auditor already produces per-criterion verdicts and a three-bucket prose audit on every shipped intent (per the itd-1 discipline). That *is* per-intent reflection; `/abcd:reflect` does not duplicate it. There is no `/abcd:reflect <itd-N>` form — the command takes a release tag only.
- **A date or tag range** as the seed, and **phase documents** as the seed: the alternatives decision 5 did not take.
- **Quantitative retrospective metrics** (DORA, velocity, etc.) — abcd doesn't gather the underlying telemetry. Metrics section is qualitative + simple counts only.
- **Team retrospectives** — abcd is single-developer-shaped (or pair-shaped); team retrospective patterns belong elsewhere.
- **Automatic triggering** — reflection requires the persona's deliberate engagement; beyond the one-line nudge of decision 1, nothing prompts for it.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an abcd repo with a cut release (`v0.11.0` is tagged and at least one intent's `shipped_in` names it), **when** the persona runs `/abcd:reflect v0.11.0`, **then** the reflection-composer agent runs an interview *seeded by that release's shipped intents and their audit notes* and writes `.abcd/development/retrospectives/v0.11.0/README.md` with all five required sections populated.
- **Given** a release one of whose shipped intents carries no audit notes, **when** the persona runs `/abcd:reflect <release-tag>`, **then** the command names that intent and offers `abcd intent audit <itd-N>` before continuing into the retrospective.
- **Given** a release tag that shipped no intent, **when** the persona runs `/abcd:reflect <release-tag>`, **then** the command refuses with "no intent shipped in `<release-tag>` — nothing shipped to reflect on" and writes no output.
- **Given** a draft retrospective with thin answers (e.g. "what went well: it worked"), **when** the agent drafts the output, **then** the agent surfaces the thinness as a clarifying question rather than committing the thin answer.
- **Given** the same repo's lifeboat is then packed via `/abcd:disembark pack <repo> <path>`, **when** the lifeboat is inspected, **then** every `.abcd/development/retrospectives/<release-tag>/README.md` the voyage produced is included in the lifeboat artefact.
- **Given** a target repo embarked from a lifeboat that includes retrospectives, **when** `/abcd:embark from <path>` runs the press-release interview, **then** the persona is shown the few predecessor lessons ranked most like the new voyage's brief, with the rest as a list, and asked which apply.
- **Given** a release with intents still unshipped whose `target_release` names it, **when** `/abcd:reflect <release-tag>` runs, **then** the command warns the persona, lists those intents, and asks for confirmation to proceed anyway.
- **Given** a release cut is written, **when** that cut completes, **then** abcd says once that a retrospective for the release is owed and names the command, and says nothing further about it.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **Nudge once.** When a phase's last piece of work closes, abcd says once that a retrospective is owed; it is not repeated and it is not a gate.
2. **A ranked few on embark.** Predecessor lessons most like the new voyage's brief are shown; the rest are a list opened on request.
3. **Layout.** The retrospective lives under the durable record tier, `.abcd/development/retrospectives/<phase-id>/README.md`; the paths this record was written against predate the three-tier layout and are read as that.
4. **The unit is the release** (ruled 2026-09-21, adr-2609212115255771): phases are retired, so `<phase-id>` reads as the release tag (`v0.10.0`), the seed is the release's shipped intents and their audit notes with the derived changelog, the empty case is a release that shipped no intent, the nudge fires once when the cut is written, and criterion 7's warning names intents targeted at the release (`target_release`) still unshipped. Every criterion below is read with "phase" meaning "release".
5. **Seed from a release** (ruled by the product thinker on 2026-09-29, ruling AD of autonomous run A's owed list): a retrospective starts from the intents a tag shipped. Seeding from a date or tag range the person names, and keeping phase documents as the anchor, were the alternatives not taken. The press release, scope and criteria were rewritten to the release on 2026-09-29, so decision 4's reading rule is now the text itself; the Blocking Dependency and v1 notes below record the phase-grain design this replaces.

## Open Questions

_None open; decisions 1 and 2 settle the two this record carried (the reflection cadence and the lifeboat surfacing on embark)._

## Blocking Dependency

_Superseded by decisions 4 and 5: the seed is the per-intent audit notes, which exist, so no phase-fidelity output is awaited. The text below is the phase-grain history._

`/abcd:reflect` **cannot be planned until the phase-fidelity-reviewer ships** and its output artefact is stable and machine-readable. The reviewer is deferred in adr-9. Because `/abcd:reflect`'s core design is to *consume* the audit's per-bullet verdicts as interview seed material, and the audit-missing AC depends on running the reviewer inline, the command has no buildable contract until the reviewer's output format exists. `/abcd:intent plan itd-24` must not proceed while the phase-fidelity-reviewer remains unbuilt.

**Satisfied:** the stable machine-readable phase-fidelity output is provided by spc-66 (predecessor store) (`phase_review_report.schema.json` + `.abcd/logbook/audit/phase-<ts>/report.{json,md}`), so this dependency is met and `/abcd:reflect` is planned under spc-83. V1 keys empty-phase detection off the spc-66 (predecessor store) receipts (not a `phase:` anchor) and refuses on a missing/empty-audited receipt rather than offering the inline reviewer — see the `### Implementation notes (spc-83.3 — v1 scope)` block below.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Implementation notes (v1 scope)

_Phase-grain history of a thin V1 that is not in the tree; superseded by decision 5, which seeds from a release._

`/abcd:reflect` (thin V1) refines two acceptance behaviours from
their originally-drafted form; both are recorded here so the fidelity review
reads them as deliberate v1 scope, not gaps:

- **Missing-audit inline-reviewer offer → refusal (deferred).** The draft AC
  had the command "offer to run the phase-fidelity-reviewer inline" when no
  audit exists. V1 instead **refuses** when no spc-66 (predecessor store) phase-audit receipt exists
  for the named phase (and when the latest matching receipt is empty-audited).
  Empty-phase detection keys off the spc-66 (predecessor store) receipts, not a `phase:` spec anchor
  (that anchor is deferred; phase membership is editorial). Running the reviewer
  inline from reflect is a recorded future extension.
- **Open-spec warn/confirm (deferred).** The draft AC had reflect warn and ask
  for confirmation when a phase's specs are not all closed. V1 does not parse
  editorial Scope membership, so this warn/confirm path is deferred; the refusal
  semantics above are the v1 gate.
- **Phase-only grain, source links, lifeboat.** `/abcd:reflect <itd-N>` is
  refused (phase-only grain). V1 links to the phase doc + audit report + member
  specs only (no intent links — the spc-66 (predecessor store) receipt carries no intent ids;
  recorded future extension). The lifeboat-packs-all-retrospectives requirement
  is a DOCUMENTED forward requirement on the future disembark spec (spc-17 (predecessor store) stubs),
  not a behaviour this surface implements — recorded in the surface doc.

The interview is a single seeded pass (per-bullet verdicts → five questions);
multi-turn depth is a recorded future extension. Full surface record:
[`../../brief/04-surfaces/09-reflect.md`](../../brief/04-surfaces/09-reflect.md).

### Linkage note (2026-09-30)

`builds_on` named itd-27 (the grill sub-verb), which is superseded by itd-94.
The edge is dropped rather than relinked: itd-94 carries the grill forward only
as the planning interview behind the implement-readiness gate, and a
retrospective seeded from a release's shipped intents needs neither that
interview nor that gate. The retrospective interview is its own, so nothing
itd-94 delivers is an input to this record.

### Linkage note (spc-83.5)

Ships as one of FOUR intents sharing spec
`spc-83-operator-surfaces-manifest-lockstep`. abcd represents "N intents, one
spec" as a bundle (`kind: bundle-member` + shared `bundle: spc-83-operator-surfaces`)
— the representation the doc_fidelity intent-resolution + spec-close preflight
require. Bundle member by delivery relationship, not a scope change. This intent
keeps its real grill linkage (`grill_session_id`); GR002 is handled via
`prd_grandfathered`. Full record in the spec's process-exception note.

## Grounds

- pursued: nine phases have closed with their lessons living only in session handovers; we expect the first retrospectives to hold what the handovers do not, and a new voyage to open a carried lesson; shown wrong if the first retrospectives say nothing the handovers did not, or if no embarked project opens one
