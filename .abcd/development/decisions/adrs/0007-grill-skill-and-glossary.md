---
id: adr-7
slug: grill-skill-and-glossary
status: accepted
date: 2026-05-11
supersedes: null
superseded_by: null
related_intents: [itd-27]
related_rfcs: []
related_adrs: []
---

# ADR-7: `/abcd:intent grill` — One Sub-verb with Two Inseparable Phases

## Context

`itd-27` adds adversarial stress-testing to the intent stage. The design space had several
candidate shapes: five separate sub-verbs (`/abcd:intent stress-test`, `/abcd:intent
brief-gaps`, etc.), a standalone top-level skill `/abcd:grill`, a single sub-verb with a
grill-only output, or a single sub-verb with two inseparable phases (grill → PRD synthesis).

Three design questions needed explicit decisions:

1. **One sub-verb with two phases vs five separate sub-verbs**
2. **Cite-or-fail lint enforcement vs advisory glossary**
3. **Bounded-context glossary structure vs flat term list**

Each decision passes Pocock's three-clause ADR test: (1) hard to reverse, (2) surprising,
(3) real trade-off. See the test per decision below.

## Decisions

### Decision 1 — One sub-verb, two inseparable phases

`/abcd:intent grill` runs a two-phase lifecycle:
- **Phase 1 (interactive):** Socratic interrogation of the intent, producing a grill report at `.abcd/logbook/grill/<ts>-<itd>/grill-report.{json,md}`. The interrogation is host-delegated to the agent harness ([adr-25](0025-host-delegated-llm-default.md)); abcd owns the prompts and consumes the structured result.
- **Phase 2 (silent synthesis):** Consumes the sharpened intent + grill findings + glossary citations + ADRs in scope → writes a Pocock-shaped PRD to `.abcd/intents/<itd-N>/prd.md`.

Phase 2 cannot be skipped or separated from Phase 1. Once synthesis runs, the session is sealed.

**Rejected alternative:** five separate sub-verbs. Rejected because (a) they lose the shared
context the grill just built — each sub-verb would have to re-prime from the intent body, not
from the live interrogation, (b) they let users skip PRD synthesis (the handoff artefact),
producing reports without the contract that justifies the session, (c) five sub-verbs with
overlapping scope create naming and sequencing confusion with no benefit over one coherent verb.

**Rejected alternative:** separate `/abcd:intent to-prd` sub-verb. Rejected because it lets
users skip Phase 2 (producing grill reports without the PRD handoff artefact), exactly the
failure mode this design rules out. Extension path if PRD regen is needed post-grill:
`/abcd:intent grill --resynthesise`, not a new sub-verb.

**Three-clause test:**
- Hard to reverse? **Yes** — once consumers (native specs, [adr-26](0026-native-spec-layer-ccpm-backend.md)) depend on the `prd_path` field
  set by Phase 2, splitting the phases would break every downstream spec.
- Surprising? **Yes** — most intent-review tools are pure interrogation; the silent synthesis
  phase with its seven Pocock sections is unexpected. New contributors will ask "why does grill
  write a PRD?"
- Real trade-off? **Yes** — tight coupling of Phase 1 → Phase 2 trades user flexibility
  (can't grill without getting a PRD) for contract completeness (every grill session produces
  the handoff artefact, no silent skip path).

### Decision 2 — Cite-or-fail lint enforcement

The intent lint (a Go implementation) enforces glossary discipline as a promotion blocker, not advisory:
- `GL001` undefined term — warn
- `GL002` forbidden synonym — blocker
- `GL003` cross-context collision without `contexts:` — warn
- `GL004` non-stable term (`draft` OR `deprecated`) in promoted intent — blocker
- `GR001` intent has acceptance but never grilled — warn
- `GR002` intent in `planned/` without PRD on disk — blocker
- `GR003` PRD modified post-promotion — blocker
- `GR004` stale planning attempt (`planning_attempt_id` set but no matching spec and >24h old) — warn
- `GR005` provenance mismatch (`source_intent_hash` or `grill_report_hash` does not match on-disk state) — blocker
- `GL005` body uses a recognised glossary term but `glossary_terms_used` omits its qualified ID — blocker (promoted) / warn (draft)

**Rejected alternative:** advisory-only glossary (no blockers). Rejected because advisory
lint produces "glossary drift" within two sprints: team members stop fixing warnings once
the list grows beyond 5 unread items. The whole point of the glossary is to prevent silent
synonym proliferation; a non-blocking lint cannot prevent it.

**Three-clause test:**
- Hard to reverse? **Yes** — once a release ships with the blocker lint, removing it would
  require auditing all existing intents for silent violations the advisory run missed.
- Surprising? **Yes** — most glossary tools are advisory; a lint that blocks promotion is
  counter-intuitive and will generate resistance from contributors used to style guides.
- Real trade-off? **Yes** — strict enforcement trades contributor friction (must fix synonyms
  before promoting) for contract integrity (no intent ships with a term the codebase doesn't
  agree on). The alternative (advisory) trades no friction for eventual synonym drift.

### Decision 3 — Bounded-context glossary structure

Each term file in `.abcd/development/foundation/terminology/` carries `bounded_context:`
frontmatter. Terms are organised under context subdirectories (`core/`, `interview/`, etc.).
The same English noun (`session`) may have different definitions under different bounded contexts
with no collision — the context disambiguates.

This is DDD ubiquitous-language applied to abcd's domain: the glossary is bounded, not global.

**Rejected alternative:** flat term list (single definition per noun). Rejected because the
abcd domain already has demonstrated collision: `session` means a grill interview session in
the intent stage and an agent-runtime session in the oracle layer — two different concepts
sharing one word. A flat glossary would either block legitimate polysemy or silently conflate
the two meanings.

**Three-clause test:**
- Hard to reverse? **Yes** — once term files exist with `bounded_context:`, flattening to a
  single namespace would require auditing every existing intent for which context each usage
  belongs to.
- Surprising? **Yes** — DDD ubiquitous-language in a developer-tool domain glossary is unusual;
  most teams use a flat style guide. The bounded-context pattern will be unfamiliar to
  contributors without DDD background.
- Real trade-off? **Yes** — bounded contexts trade added complexity (contributors must learn
  the DDD pattern) for precision (no silent polysemy, no single-definition tyranny over
  multi-context terms). The alternative (flat list) trades simplicity for eventual definitional
  collision.

## Phase 1 → Phase 2 Inseparability

Phase 1 and Phase 2 are inseparable by design. The PRD synthesises from the **live interrogation
context** — the sharpened press release, the accepted glossary additions, the Toulmin warrants
surfaced during questions, and the ADR decisions offered during the session. None of this is
reconstructable from the intent file alone after the session ends.

A user who runs Phase 1 and skips Phase 2 has a grill report but no handoff artefact — the
session produces no contract that `/abcd:intent plan` can consume. This is the design failure the
inseparability constraint prevents. The one exception to "no skip" is `--lite` mode, which
still runs Phase 2 (the flag controls glossary loading, not the two-phase lifecycle).

## Freeze Contract

At `/abcd:intent plan` time, the PRD is frozen: `frozen_at`, `frozen_content_hash`
(SHA-256 of all PRD content except the self-referential freeze fields `frozen_at`,
`frozen_content_hash`, `spec`, and `planning_attempt_id`; provenance fields such as
`source_intent_hash`, `grill_report_hash`, and `grill_report_path` ARE included to prevent
post-freeze tampering), and `planning_attempt_id` (UUIDv4 linking to the plan run in the native spec
layer, [adr-26](0026-native-spec-layer-ccpm-backend.md))
are written to PRD frontmatter. After freeze:
- Any modification to the PRD body triggers `GR003` blocker
- The press-release intent and frozen PRD are both immutable input artefacts for the native spec layer
- The press release is the elevator pitch; the PRD is the AI-consumption contract
- v1 refuses regrill on frozen PRDs immediately via `FrozenPRDError` (not a GR003 check — GR003
  fires only on frozen-PRD body mutation); regrill-after-freeze (`--resynthesise`) is reserved for v2

## Acknowledgements

- PRD template adapted from [mattpocock/skills `to-prd`](https://github.com/mattpocock/skills/blob/main/skills/engineering/to-prd/SKILL.md) (MIT). Seven sections: Problem Statement, Solution, User Stories, Implementation Decisions, Testing Decisions, Out of Scope, Further Notes.
- Capture-while-grilling pattern adapted from [mattpocock/skills `/grill-with-docs`](https://github.com/mattpocock/skills/blob/main/skills/engineering/grill-with-docs/SKILL.md) (MIT).
- Verb name `grill` borrowed from [mattpocock/skills `/grill-me`](https://github.com/mattpocock/skills/blob/main/skills/productivity/grill-me/SKILL.md) (MIT).
- Bounded-context glossary structure: Evans, E. (2003). *Domain-Driven Design*. Fowler, M. [BoundedContext](https://martinfowler.com/bliki/BoundedContext.html).
- Socratic moves (six named): Chang et al. (2023). arXiv:2303.08769.
- EARS acceptance criteria notation: Mavin et al. (2009). *EARS: The Easy Approach to Requirements Syntax*.
- Cite-or-fail mechanism: GitHub Spec Kit `/speckit.clarify` + `/speckit.checklist` patterns.

## Related

- [itd-27](../../intents/superseded/itd-27-grill-skill-and-glossary.md) — source intent
- [05-internals/01-agents.md](../../brief/05-internals/01-agents.md) — `intent-fidelity-reviewer` auditor contract
- tests/fixtures/grill/ — fixture corpus
