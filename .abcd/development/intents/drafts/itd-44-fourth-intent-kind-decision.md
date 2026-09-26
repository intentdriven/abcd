---
id: itd-44
slug: fourth-intent-kind-decision
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
blocked_by: [itd-34]
severity: minor
builds_on: [itd-39]
---

# A Capture-Time `decision` Verdict That Routes Infrastructure Choices Into The Existing ADR Store

## Operator deviation (thin adoption — recorded)

> **This intent is scoped to an operator-elected THIN adoption** (interview
> decision; ratified at plan-review r1 of `spc-56-fourth-intent-kind-decision-thin`
> in the predecessor store, a spec the Go rebuild did not carry — the live
> `spc-56` is an unrelated record). Two design choices in the original
> draft (preserved below the line, struck through where superseded) are
> **rejected**, and the rejection is recorded here so a future reader does not
> re-derive the heavy design:
>
> 1. **`decision` is NOT a fourth persisted intent `kind`.** It is a
>    **capture-time classifier VERDICT** only. Every persisted `kind` in the
>    codebase maps to a lifecycle directory and a routing/lint matrix (`IL003`,
>    `IL005`, `IL006`, `_PLAN_CANDIDATE_KINDS`, `reclassify`). A `decision` has
>    no lifecycle directory under thin adoption, so a committed `kind: decision`
>    would be an enum value with no legal home. The schema `kind` and
>    `kind_at_supersession` enums stay **three-valued**; only `suggested_kind`
>    (the advisory capture hint) and the classifier admit `decision`.
> 2. **NO new `intents/decisions/{active,superseded}/` store.** This repo
>    already runs a thriving ADR practice
>    (`.abcd/development/decisions/adrs/`, `adr-N-<slug>.md`, next-free
>    numbering, a README-defined template, supersession via the `status`
>    field). Two stores for one artifact class is the Single-Source-of-Truth
>    violation this repo's conventions exist to prevent. A `decision` verdict
>    therefore routes capture to the **existing ADR store**, not to a new
>    intent lifecycle.
>
> To the facilitator it is an ADR; to the product thinker it was just another
> `/abcd:intent`. The captured artefact is an `adr-N-<slug>.md` with the
> store's `## Context` / `## Decision` / `## Alternatives Considered` /
> `## Consequences` body — **no** `## Rationale` section (rationale folds into
> `## Context`, matching the store convention).

> **Delivery state (amended 2026-09-26, iss-239).** Nothing here is delivered.
> The thin adoption was ratified against a spec of the predecessor store; the Go
> binary has no capture-time classifier verdict, no `decision` route into the ADR
> store from `abcd intent`, and no "decisions are not plannable" refusal — the
> brief states the same (`brief/04-surfaces/05-intent.md`, the fourth capture
> verdict). This draft carries the design for a later plan; `abcd decide` mints an
> ADR directly and is not this capture-time route.

## Why This Matters

The three-intent-kinds taxonomy (itd-34) covers `standalone`, `bundle-member`,
and `discipline`. It omits a fourth shape the corpus produces in practice:
**infrastructure choices that aren't user-facing and aren't rules**.

Examples from real projects (these are the classifier's `decision`-verdict
fixtures):

- "We use Postgres for the audit trail" — not a user moment, not a rule every
  artefact must clear; a one-time standing choice with consequences.
- "The project is Python 3.12" — same shape.
- "The build system is Bazel, not Make" — same shape.
- "We deploy to Fly.io, not Heroku" — same shape.

None of these fit the existing kinds cleanly:

- **`standalone`** forces a press release for a thing with no user moment. The
  press release reads as fiction.
- **`bundle-member`** is for intents coupled by shared delivery. A Postgres
  choice isn't coupled to a user-facing intent in that sense.
- **`discipline`** is for cross-cutting *rules* (every X must Y). "We use
  Postgres" isn't structurally that shape — it's a standing choice, not a rule.
  There's no per-artefact enforcement, just a standing fact.

In abcd-cli today these decisions fall through three gaps: the brief (meant to
stay high-level), individual native specs (the choice leaks in as a
side-effect of the feature that needed it), or nowhere at all (genuinely
invisible plumbing). The cost: no canonical record; the decision and its
rationale are entangled with the code that consumed them — the wrong shape,
because code rots and gets refactored while the *decision* is more stable than
any one place that touches it.

The framing trick that resolves this without adding command-surface: **what an
industry engineer calls an ADR, a product thinker calls an intent.** Both
vocabularies point at the same artefact. The product thinker keeps writing
`/abcd:intent "<text>"` for every idea; the capture classifier recognises the
standing-choice signature and emits a `decision` verdict; the system routes the
artefact into the ADR store; the facilitator reads ADRs.

This intent is **project-agnostic** in the same sense itd-34 is: abcd-cli
produces decisions about lint orchestration and Codex wiring; an application
project produces decisions about its database, its deployment target, its
language version. The verdict generalises.

## What's In Scope (thin adoption)

### A capture-time `decision` verdict (not a persisted kind)

- **The capture-time kind classifier gains a fourth verdict, `decision`,** for
  the no-user-moment / not-a-per-artefact-rule / standing-choice signature.
- The verdict is exposed as a CLI primitive, `intent classify-capture-kind`,
  that reads the seed text on stdin and returns a STABLE JSON verdict
  (`{"suggested_kind": "decision"|"standalone"|"bundle-member"|"discipline"}`),
  because the `abcd-intent-new` skill is wrapper-only and has no Python
  classifier seam. The skill calls the CLI, then routes. Existing kind
  suggestions are unchanged (regression fixtures). This is distinct from the
  Role-3 `ShapeSuggestion` reclassify path.
- `suggested_kind: decision` is accepted by the intent schema; `kind:
  decision` and `kind_at_supersession: decision` remain **rejected**.

### Routing a `decision` verdict to the existing ADR store

- A confirmed `decision` verdict DIVERTS capture: instead of
  `reserve_and_write_intent` (which writes `drafts/itd-N-*.md`), capture writes
  `adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`, matching
  the store's README template EXACTLY — frontmatter (`id, slug, status, date,
  supersedes, superseded_by, related_intents, related_rfcs, related_adrs`) plus
  body `## Context` / `## Decision` / `## Alternatives Considered` /
  `## Consequences`. No `## Rationale`; no `## Press Release`; no
  `## Acceptance Criteria`.
- Because a false-positive `decision` would silently write an ADR instead of an
  intent draft, capture CONFIRMS: on a `decision` verdict the operator confirms
  "capture as an ADR?" or OVERRIDES to a normal intent draft. On override the
  draft persists a PLANNABLE `suggested_kind` (`null`, or `standalone` if the
  operator picks one) — never `decision` (an unplannable value, refused by the
  plan/reclassify guard).
- `decision` is therefore NEVER written to the schema `kind` or
  `kind_at_supersession`, NEVER enters the `drafts/→planned/→shipped/`
  lifecycle, NEVER gets a native spec, and is referenced downstream as
  `adr-N` (which the predecessor store's spc-48 RC linkage lint, matching only
  `itd-N` tokens, ignored).

### Plan/reclassify refusal guard

- `_choose_candidate_kind` reads `suggested_kind` but only routes values in
  `_PLAN_CANDIDATE_KINDS`; an unknown value (incl. `decision`) would fall
  through to the interactive prompt and could plan a `suggested_kind: decision`
  draft as `standalone`. `plan_single`/`reclassify` therefore gain an EXPLICIT
  refusal: a `decision` candidate (via `--kind decision`, `suggested_kind:
  decision`, or `reclassify --kind decision`) raises "route this to ADR
  capture; decisions are not plannable". `_PLAN_CANDIDATE_KINDS` stays
  three-valued.

## What's Out Of Scope

- **A new `intents/decisions/` store and its `active/`/`superseded/`
  lifecycle.** Rejected (operator deviation above) — the existing flat
  `adrs/` + supersession-via-`status` is the convention.
- **A persisted `kind: decision`.** Rejected — `decision` is a verdict, never a
  committed kind.
- **Per-domain decision templates** (database/deployment/language-specific
  fields). The abcd-framework shape is structurally identical across domains;
  per-domain templates belong to the project that needs them.
- **Automated decision detection from code** (inferring "uses Postgres" from
  `pyproject.toml`). A separate follow-up intent.
- **Decision-aware code review** (catching a PR that violates a standing
  decision). Belongs with the disciplines machinery, not here.
- **Migrating existing scattered plumbing** (Python 3.12, flow-next, the lint
  orchestrator, the Codex backend) into the ADR store. A separate downstream
  task.
- **Planner-context pull of active decisions / a decisions fidelity reviewer.**
  The original draft's load-bearing "pull active decisions into the planner"
  claim rode on itd-39 (scope-aware memory retrieval); under thin adoption the
  artefact is an ADR the facilitator reads directly, and the
  pull-into-planner mechanism is deferred to itd-39's substrate, not built here.

## Acceptance Criteria

The original draft's full-store acceptance criteria are **superseded-by-decision**
(the thin-adoption operator deviation recorded above). They are struck through in
the preserved draft below the line and are NOT requirements of any spec. The
thin-adoption criteria were written as R1–R5 of
`spc-56-fourth-intent-kind-decision-thin` in the predecessor store; the live spec
store holds no spec for this intent (`spec_id: null`), so the criteria below are
this draft's own, in summary:

- *Given* a product thinker records a standing choice, *when* they run
  `/abcd:intent "we use Postgres for the audit trail"`, *then* the capture
  classifier emits a `decision` verdict, the operator confirms, and the system
  writes an `adr-N-<slug>.md` into the existing ADR store (README-exact
  template) — no intent draft, no spec, no `intents/decisions/` directory.
- *Given* a false-positive `decision` verdict, *when* the operator OVERRIDES,
  *then* a normal intent draft is written carrying a PLANNABLE `suggested_kind`
  (never `decision`), and it routes cleanly through `plan_single`.
- *Given* a `decision` candidate reaches `plan_single`/`reclassify` (via
  `--kind decision`, `suggested_kind: decision`, or `reclassify --kind
  decision`), *then* it is REFUSED with "decisions are not plannable";
  `_PLAN_CANDIDATE_KINDS` stays three-valued.
- *Given* the schema, *then* `suggested_kind: decision` validates while `kind:
  decision` and `kind_at_supersession: decision` are rejected.

## Related

- **itd-34** (three-intent-kinds) — this intent extends the capture taxonomy
  with a fourth CAPTURE-VERDICT value; the persisted three-`kind` enum is
  unchanged. `adr-2-three-intent-kinds` is the ADR that records itd-34.
- **`.abcd/development/decisions/adrs/README.md`** — the store template,
  filename convention (`adr-N-<slug>.md`), and `status`-field supersession the
  ADR writer matches.
- **itd-39** (scope-aware memory retrieval) — the deferred substrate for
  pulling active decisions into planner context (out of scope here).
- **a dated working-log entry (2026-05-22)** — the discussion that produced this intent.
- **`.abcd/logbook/grill/20260522T192535Z-itd-44/`** — first grill pass (lite
  mode); recommends re-grilling against the thin scope (this body rewrite is its
  prerequisite).

---

## Superseded draft (full-store design — REJECTED, preserved for the record)

> The sections below were the original draft's full-store proposal. They are
> **superseded-by-decision** under the thin-adoption operator deviation recorded
> at the top of this file: `decision` is NOT a persisted kind, and NO
> `intents/decisions/` store is created. Preserved so the rejected design and
> its rationale are not silently lost.

### ~~A fourth intent kind: `decision`~~ (superseded)

- ~~**Frontmatter `kind: decision`**, set at plan time, suggested by the classifier at capture time.~~
- ~~**Body shape:** `## Decision` + `## Rationale` + `## Alternatives Considered` + (optional) `## Consequences`. No `## Press Release`. No `## Acceptance Criteria`.~~
- ~~**Lives in `intents/decisions/`** with two sub-states encoded by directory: `decisions/active/` and `decisions/superseded/`. No `drafts/ → planned/ → shipped/` lifecycle.~~
- ~~**Never gets a flow-next spec.** Like disciplines, decisions register by living in their directory and are referenced by other intents/specs.~~

### ~~Capture and classification~~ (superseded by the classifier-verdict + ADR-route design above)

- ~~`/abcd:intent plan <itd-N>` accepts `decision` as a valid kind selection; routes to the decision-specific lifecycle (no flow-next call, file moves to `decisions/active/`).~~
- ~~`/abcd:intent reclassify <itd-N> --kind decision` works the same way it does for the other kinds; supersession via `--kind superseded --by <itd-M>`.~~

### ~~Discoverability at plan time~~ (superseded — deferred to itd-39)

- ~~**Active decisions are pulled forward into the planner's context** when planning any new feature.~~
- ~~A `decisions/index.md` (auto-generated) lists active decisions.~~

### ~~Audit / fidelity~~ (superseded — no decision-kind intent files exist to audit)

- ~~The `intent-fidelity-reviewer` agent (spc-12, predecessor store) gets a Role 1 variant for decisions, with per-decision verdicts (`IMPLEMENTED` / `IMPLEMENTED_WITH_DRIFT` / `NOT_IMPLEMENTED` / `INCONCLUSIVE`).~~

### ~~Original full-store Acceptance Criteria~~ (superseded-by-decision)

- ~~*Given* the product thinker has an infrastructure choice to record, *when* they run `/abcd:intent "we use Postgres for the audit trail"`, *then* the classifier writes `suggested_kind: decision` to the new file's frontmatter and the system asks them for rationale and alternatives at plan time.~~
- ~~*Given* an intent with `kind: decision` is in `decisions/active/`, *when* the facilitator plans a spec that touches the audit layer, *then* the active decision is included in the planner's context.~~
- ~~*Given* an active decision is being replaced, *when* the user runs `/abcd:intent reclassify <itd-N> --kind superseded --by <itd-M>`, *then* the file moves to `decisions/superseded/`.~~
- ~~*Given* a shipped spec referenced an active decision, *when* `intent-fidelity-reviewer` runs, *then* per-decision verdicts are written into the decision's `## Audit Notes` section.~~
- ~~*Given* a product thinker is browsing `/abcd:intent` bare output, *when* the corpus contains active decisions, *then* decisions appear as a distinct group alongside drafts / planned / shipped / disciplines / superseded.~~
