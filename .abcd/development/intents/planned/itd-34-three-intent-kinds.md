---
id: itd-34
slug: three-intent-kinds
kind: standalone
suggested_kind: standalone
bundle: null
spec_id: null
reclassification_history: []
builds_on: [itd-1]
severity: major
---

# Intents Promote In Three Flavours — Standalone, Bundle-Member, Discipline

## Press Release

> **abcd ships three intent kinds, each with its own lifecycle path through `/abcd:intent plan`.** A standalone intent maps 1:1 to a flow-next `spec` — the default for ~60% of intents, and the only path the prior brief recognised. A bundle-member intent shares a `spec` with its bundle-mates: the persona runs `/abcd:intent plan itd-A itd-B`, abcd creates one shared `spec` with both intents listed, and both files move from `drafts/` to `planned/` together. A discipline intent has no persona moment of its own — it's a cross-cutting rule (every `spec` must carry acceptance criteria, every agent `prompt` must declare a `version`) that lives in `disciplines/` instead of `drafts/`, never gets its own `spec`, but applies to every other `spec` as an inherited acceptance gate. Capture stays format-neutral; the binding `kind` field is set at plan time, with an `oracle` classifier writing an advisory hint at capture time. The discipline shape is `voyage`-agnostic — an application `voyage` (e.g., a macOS app under abcd) produces its own disciplines around privacy-impact review or accessibility passes with the same lifecycle as abcd's own discipline intents.
>
> "I had an intent that didn't fit the standard mould — it was a rule about every future `spec`, not a `feature` being built — and I'd been forcing it through the `press-release` format anyway, ending up with prose that read awkwardly because there was no real persona moment to describe," said Iris, product lead. "The discipline kind let me write `## Rule` instead of `## Press Release` and have the system understand what it was looking at. Two of my intents reclassified from standalone to discipline; the lifecycle paths matched the actual shape of the work for the first time."

## Why This Matters

The original abcd intent system treated every intent as a 1:1 capability that maps to one flow-next spec. An audit-driven corpus sweep surfaced a structural finding: **~40% of intents have non-1:1 relationships with at least one other intent.** Three patterns recurred:

1. **Coupled intents** — two or more intents that capture distinct user moments but only make sense delivered together. The original canonical example was itd-31 (cross-document fidelity review) and itd-32 (audit-role taxonomy) — but that bundle was dissolved on 2026-05-07 when the round-2 command-structure review split the three review roles into three distinct verbs under `/abcd:intent` (review/consistency/shape), making itd-32's unified-`/abcd:audit`-surface taxonomy premise no longer load-bearing. itd-31 promoted to standalone; itd-32 superseded. The pattern itself is still real (coupled intents that ship together as one shared spec exist in the corpus), but it has no live example in the current corpus.
2. **Subsuming intents** — one intent whose value is fully covered by another, larger intent. itd-5 (prompt-quality additions) and itd-14 (prompt registry) are the canonical example: itd-5 carves off a cheaper subset of itd-14. Shipping both means writing prompt-version handling twice.
3. **Cross-cutting rules** — itd-1 (acceptance gates) and itd-5 are not features at all. They are rules that apply to every other spec. Forcing them into the standalone-spec mould produces a "spec" whose only deliverable is "every other spec inherits this gate" — which is not a spec.

The 1:1 model fits cleanly only for the standalone ~60%. The other 40% calcifies into wrong shapes — bundles ship as separately-planned epics that retrofit onto each other; disciplines get press-release shaped despite having no user moment; supersession is implicit and lossy.

This intent introduces the three kinds as first-class lifecycle paths. **Most intents stay standalone** — the default doesn't change. Bundle-member and discipline are escape hatches for the shapes the standalone path doesn't fit. Capture stays cheap and format-neutral; the kind decision happens at plan time, when the user is committing to *build* the thing — that's where the shape decision has to be true.

The change is also **project-agnostic.** abcd is the loud first user of disciplines because abcd is a methodology project (its outputs include rules), but the pattern generalises. An idelphiDev intent like "every recording feature must include a privacy-impact review" is structurally identical to itd-1 — a rule that applies to every other spec — and naturally fits the discipline kind. The three kinds are a property of the intent framework, not of abcd's particular subject matter.

## What's In Scope

### Three intent kinds

- **`kind: standalone`** — one press-release-shaped user moment, ships as one spec. Default; ~60% of the corpus. Lives in `drafts/` → `planned/` → `shipped/`. No change from the original 1:1 behaviour.
- **`kind: bundle-member`** — multiple intents that share underlying delivery. Each member has its own press release (each captures a distinct user moment); they share an `epic_id` and a `bundle: <bundle-id>` frontmatter field. `/abcd:intent plan <itd-A> <itd-B>` is the multi-arg promotion path. Members move together (`drafts/` → `planned/` → `shipped/`); audit runs per-member against the same delivered reality.
- **`kind: discipline`** — cross-cutting rule with no user moment. Lives in `disciplines/` instead of `drafts/`. Never gets a flow-next spec; instead imposes acceptance gates that every *other* spec inherits and is checked against. Uses `## Rule` + `## Why` template instead of `## Press Release`. `kind_notes` free-text field describes what kind of rule it is (subtype taxonomy deferred — see "Discipline subtypes are deferred" below).

### Lifecycle paths

`/abcd:intent plan` proposes a `kind` based on the intent body, cross-references, and any `suggested_kind` hint from capture time. The user confirms or overrides; the chosen value is *binding* once written to frontmatter.

- **Standalone**: today's path. `/flow-next:plan` + `plan-review`; bidirectional link; `drafts/` → `planned/`.
- **Bundle-member**: multi-arg `/abcd:intent plan itd-A itd-B`. `/flow-next:plan` once with all intents as joint input; `epic.intent: [itd-A, itd-B]`; each intent's `epic_id: spc-N` and `bundle: <bundle-id>`; all members `drafts/` → `planned/` together.
- **Discipline**: no `/flow-next:plan` call. Registers acceptance gates in `.abcd/disciplines/<itd-N>.json`. `/flow-next:plan-review` runs on the discipline's `## Rule` for sanity (catches malformed rules). `drafts/` → `disciplines/` (active state encoded by directory location; no `status:` field per the 2026-05-08 directive — see brief 04-surfaces/05-intent.md § 5). No `ship` step — disciplines are continuously active once in `disciplines/`.

### `/abcd:intent reclassify` subcommand

For late kind changes (a standalone intent realised to be a bundle-member; a draft realised to be a discipline; a shipped intent superseded by a later one):

- `/abcd:intent reclassify <itd-N> --kind <new-kind> [--reason <text>]`
- Records `reclassification_history` entry (date + from-kind + to-kind + reason) in the intent's frontmatter
- Moves the file between directories as the new kind dictates
- `--kind superseded --by <itd-M>` is the supersession path: file moves to `superseded/`; frontmatter records `superseded_by: itd-M`; preserved as historical record

### Frontmatter additions

All intent files gain four new fields:

```yaml
kind: null              # set by /abcd:intent plan: standalone | bundle-member | discipline
suggested_kind: null    # advisory, written by capture-time LLM classifier; can be ignored
bundle: null            # for kind: bundle-member, the bundle ID
reclassification_history: []   # appended to by /abcd:intent reclassify
```

Discipline-kind intents additionally use `kind_notes: "<free-text>"` describing what kind of rule they are. Disciplines have **no `status` field** — the directory IS the state (`disciplines/` = active; `superseded/` = retired). Across all kinds, directory location is the single source of truth for lifecycle state; status fields would duplicate (and risk drifting from) what the directory already says.

### Supersession captures original kind

When `/abcd:intent reclassify <itd-N> --kind superseded --by <itd-M>` runs, the intent's frontmatter records two fields:

```yaml
superseded_by: itd-M               # the successor intent
kind_at_supersession: <original>   # what the intent was when retired:
                                   #   standalone | bundle-member | discipline
```

Both are required (lint hard-blocks if either is missing). The reason: "superseded" means different things depending on what the intent *was*. A superseded standalone is a retired capability. A superseded bundle-member is a retired half of a coupled pair. A superseded discipline is a retired rule that was inherited by every other spec. Without `kind_at_supersession`, future archaeology has to reconstruct the original shape from `reclassification_history` (which exists but is harder to query).

### `intent-fidelity-reviewer` shape-classification role (third role)

The same agent that performs single-document fidelity audits (per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md)) and cross-document fidelity audits (per [itd-48](itd-48-intent-fidelity-reviewer-roles-2-3.md), which superseded [itd-31](../superseded/itd-31-cross-document-fidelity-reviewer.md)) gains a third role: **shape classification.** It runs continuously via the pre-commit hook (writing findings to the latest report) and on-demand via `/abcd:intent shape`. Bare `/abcd:intent` (status+help) surfaces the latest cached shape suggestions in its summary output without itself running a fresh scan — bare invocation never mutates the report. Findings live at `.abcd/logbook/audit/shape-<ts>/report.{json,md}`. Specific suggestions:

- **Bundle candidate:** "intents X and Y reference each other in scope/references and target the same release; consider `kind: bundle-member` with shared bundle ID."
- **Supersession candidate:** "intent X's scope is fully covered by intent Y; consider `kind: superseded --by Y`."
- **Discipline candidate:** "intent X has no user moment in its press release (the customer quote describes a process, not a feature); consider `kind: discipline`."

The user accepts a suggestion via `/abcd:intent reclassify`. Declined suggestions are logged so the reviewer doesn't re-surface the same one every run.

The agent count in the catalog stays at **15** — these three roles share one agent's prompt scaffolding, oracle backend resolution, and receipts.

### Discipline subtypes are deferred

The brief introduces the discipline kind itself; it deliberately does *not* commit to a closed enum of discipline subtypes (e.g., `methodology` / `documentation` / `audit` / `convention`). Each discipline declares a free-text `kind_notes` field describing what kind of rule it is. The subtype taxonomy moves from free-text to formal enum when ANY of the following:

1. Three or more disciplines exist with `kind_notes` describing similar shapes.
2. A user is confused about which existing discipline a finding belongs to.
3. `intent-fidelity-reviewer`'s shape-classification role cannot describe a proposed discipline without more constraint than `kind_notes` provides.
4. Cross-project usage — once abcd is in three or more projects, comparing disciplines across them needs a shared subtype vocabulary.

This mirrors the lint-code namespace pattern in [`05-internals/06-lint.md`](../../brief/05-internals/06-lint.md) — categories added empirically as intents introduce them, not declared up front.

### New directories

- `intents/disciplines/` — active discipline-kind intents (created alongside this intent)
- `intents/superseded/` — intents killed by reclassification or absorption (currently empty)

### Reclassification of two existing intents

[itd-1](../disciplines/itd-1-acceptance-gates.md) (acceptance gates) and [itd-5](../disciplines/itd-5-prompt-quality-additions.md) (prompt-quality additions) are reclassified from `kind: standalone` to `kind: discipline`. Their press-release sections are replaced with `## Rule` + `## Why` (the press-release format requires a user moment, which disciplines do not have). Files move from `drafts/` to `disciplines/`. Each gains a `reclassification_history` entry recording the reclassification.

### Brief edits

- [`01-product/03-mental-model.md`](../../brief/01-product/03-mental-model.md) — three-layer model gains the three intent kinds under "Intents."
- [`04-surfaces/05-intent.md`](../../brief/04-surfaces/05-intent.md) — § 1 grows kinds + discipline format + subtype-deferred revisit triggers; § 2 subcommands table adds `reclassify` and multi-arg `plan`; § 5 lint rules extend to verify `kind`, `bundle:`, and directory matching; § 6 documents the reviewer's three roles.
- [`05-internals/01-agents.md`](../../brief/05-internals/01-agents.md) — `intent-fidelity-reviewer` row condensed; new "three roles" sub-section.

## What's Out of Scope

- **Mandating bundle-member or discipline kinds for any specific intent.** The kinds are *available*; the user (with the LLM classifier's advice) chooses. abcd never auto-reclassifies without user confirmation.
- **A closed enum of discipline subtypes.** Deferred per the revisit triggers above.
- **Cross-bundle dependency tracking** ("bundle X must ship before bundle Y"). Out of scope; bundles are a delivery shape, not a dependency graph.
- **Auto-merging coupled intents at capture time.** The LLM classifier proposes; the user binds at plan time. The framework never silently merges intents.
- **A fourth kind** for some hypothetical case the three kinds don't cover. If a fourth kind is needed, it lands as a future intent, not this one. **Lineage update (itd-44, spc-56):** the standing-infrastructure-choice case the three kinds don't fit landed as [itd-44](../drafts/itd-44-fourth-intent-kind-decision.md) — but *not* as a fourth persisted `kind`. itd-44 adds a fourth capture-time *verdict*, `decision`, that routes a confirmed standing choice into the existing ADR store (`adr-N`); the persisted `kind` enum here stays **three-valued**, and there is deliberately no `intents/decisions/` lifecycle.
- **Migrating disciplines into the brief itself.** Disciplines live in `intents/disciplines/` so they have the same lifecycle, lint, and frontmatter discipline as other intents. Folding them into the brief would lose the lifecycle handling. The brief *describes* the discipline kind in its mental-model and intent-surface sections; the discipline files themselves stay under `intents/`.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md). These gates are checked by `intent-fidelity-reviewer`'s single-document role when this intent moves to `shipped/`._

- **Given** a draft intent at `drafts/itd-N-foo.md` with `kind: null`, **when** the user runs `/abcd:intent plan itd-N`, **then** abcd proposes a kind based on the intent body and cross-references, the user confirms or overrides via interactive prompt, and the chosen kind is written to the intent's frontmatter as binding.
- **Given** two draft intents at `drafts/itd-A-foo.md` and `drafts/itd-B-bar.md` whose press releases reference each other in their References sections, **when** the user runs `/abcd:intent plan itd-A itd-B`, **then** `/flow-next:plan` is called once with both intents as joint input; the resulting spec (`spc-N`) in the native spec store has `intent: [itd-A, itd-B]`; each intent's frontmatter has `kind: bundle-member`, `bundle: <auto-generated-or-user-supplied-id>`, and `epic_id: spc-N`; both files move from `drafts/` to `planned/`.
- **Given** two draft intents `itd-A` and `itd-B` scoped to different phases, **when** the user runs `/abcd:intent plan itd-A itd-B` (multi-arg, kind=bundle-member), **then** abcd hard-blocks promotion with lint code `IL011`, cites the bundle invariant in [`brief/04-surfaces/05-intent.md § 1`](../../brief/04-surfaces/05-intent.md#1-intent-ids-kinds-and-lifecycle), and offers the user two resolutions: (a) re-scope all proposed members into the same phase, or (b) downgrade one or more members to `kind: standalone` and re-run plan. Worked example in the wild: the `intent-capture-discipline` bundle (itd-27 + itd-30) was retired on 2026-05-07 for exactly this reason — both intents now ship as `kind: standalone`.
- **Given** a bundle's shared spec closes in the native spec store, **when** the lifecycle hook fires, **then** all bundle-member intents (those with the matching `bundle:` field) move from `planned/` to `shipped/` together AND `intent-fidelity-reviewer`'s single-document role runs once per member against the same delivered reality, producing per-member Audit Notes.
- **Given** a draft intent that the user wants to promote as a discipline, **when** the user runs `/abcd:intent plan itd-N` and selects `kind: discipline` at the prompt, **then** NO `/flow-next:plan` call is made; the intent's acceptance gates are registered in `.abcd/disciplines/<itd-N>.json`; the file moves from `drafts/` to `disciplines/`. The intent's frontmatter has NO `status` field — the directory IS the state.
- **Given** a discipline-kind intent in `disciplines/`, **when** any subsequent flow-next spec is plan-reviewed, **then** the plan-review verifies the spec's acceptance criteria are compatible with all active disciplines' rules — drift between spec acceptance and discipline gate is flagged.
- **Given** a shipped intent that the user later realises is fully superseded by a newer intent, **when** the user runs `/abcd:intent reclassify <itd-N> --kind superseded --by <itd-M> --reason "fully covered by itd-M"`, **then** the file moves from `shipped/` to `superseded/`; frontmatter gains both `superseded_by: itd-M` AND `kind_at_supersession: standalone` (preserving what the intent was when retired); a `reclassification_history` entry records the date + reason.
- **Given** an active discipline that is being replaced by a stricter successor, **when** the user runs `/abcd:intent reclassify <itd-N> --kind superseded --by <itd-M>`, **then** the file moves from `disciplines/` to `superseded/`; frontmatter gains both `superseded_by: itd-M` AND `kind_at_supersession: discipline` so future readers can tell the retired intent was a rule (not a capability).
- **Given** an intent in `superseded/` lacks either `superseded_by` or `kind_at_supersession`, **when** `internal/core/lint` runs, **then** the lint hard-blocks with a clear error — both fields are required for every superseded intent regardless of original kind.
- **Given** the corpus contains two intents whose press releases reference each other and target the same release, **when** `intent-fidelity-reviewer`'s shape-classification role runs (pre-commit or on `/abcd:intent` invocation), **then** a "bundle candidate" suggestion appears in `/abcd:intent` status output AND in `.abcd/logbook/audit/shape-<ts>/report.{json,md}`.
- **Given** the user declines a shape-classification suggestion, **when** the reviewer runs again on the same corpus state, **then** the declined suggestion is NOT re-surfaced (logged in the reviewer's "declined-suggestions" cache) — the reviewer doesn't nag.
- **Given** a project shipping a non-abcd application (e.g., idelphiDev) under the abcd intent framework, **when** the user captures a project-specific discipline (e.g., "every recording feature includes a privacy-impact review"), **then** the same lifecycle applies — the discipline lands in `disciplines/`, never gets its own spec, and is enforced against every other spec via plan-review. The framework treats it identically to abcd's own disciplines.
- **Given** the brief and intent corpus are updated, **when** `internal/core/lint` runs, **then** it verifies (a) every intent in `planned/` has `kind` set and matching the directory; (b) `kind: bundle-member` intents have `bundle:` set bidirectionally with their bundle-mates; (c) `kind: discipline` intents live only in `disciplines/` or `superseded/`; (d) `kind: discipline` intents have `epic_id: null`; (e) discipline-kind intents have NO `status` field (the directory is the state); (f) every intent in `superseded/` has both `superseded_by` and `kind_at_supersession`. Violations are hard-block lint failures.

## Dependencies

- **Coordinated with:** the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md) — itd-1 is the first intent reclassified to `kind: discipline` under this framework. itd-1's content rewrite and itd-34's lifecycle changes ship in the same brief revision.
- **Coordinated with:** the [itd-5 discipline](../disciplines/itd-5-prompt-quality-additions.md) — second discipline reclassified.
- **Coordinated with:** [itd-48](itd-48-intent-fidelity-reviewer-roles-2-3.md) — itd-48 owns the cross-document role (Role 2) and the shape-classification role (Role 3) on `intent-fidelity-reviewer`, superseding [itd-31](../superseded/itd-31-cross-document-fidelity-reviewer.md) which originally introduced the cross-document concept. The agent's three-role architecture is documented uniformly across the brief and intent surfaces.
- **Coordinated with:** [itd-48](itd-48-intent-fidelity-reviewer-roles-2-3.md) (cross-document role) and [itd-34's own shape-classification role] — these are the second and third roles on `intent-fidelity-reviewer` (the first being single-document fidelity per itd-1). Each role has its own user-facing verb under `/abcd:intent` (consistency, shape, review respectively). The earlier `tier-0-audit-substrate` bundle ([itd-31](../superseded/itd-31-cross-document-fidelity-reviewer.md) + itd-32) was dissolved on 2026-05-07 when the unified-`/abcd:audit`-surface premise no longer held; itd-31 promoted to standalone (later superseded by itd-48 on 2026-05-27), itd-32 superseded. An even earlier attempted bundle (`intent-capture-discipline`, itd-27 + itd-30) was retired on the same day because itd-27 and itd-30 are scoped to different phases — bundles cannot span phases (one shared spec shipped together is the invariant). Both intent-capture intents reclassified to standalone.

## Open Questions

- **Bundle ID assignment.** Multi-arg `/abcd:intent plan itd-A itd-B` needs to generate or accept a bundle ID. Options: (a) auto-generate a slug from the joint title (terse), (b) prompt for an explicit ID at plan time (deliberate), (c) require pre-declaration in one of the member intents' frontmatter before plan runs (strict). Recommend (b) with default suggestion derived from joint title.
- ~~**What happens if a bundle's intents are scoped to different phases?**~~ **Resolved 2026-05-07** — bundle invariant codified: all members belong to the same phase; multi-arg plan hard-blocks via `IL011` if they disagree. See [`brief/04-surfaces/05-intent.md § 1`](../../brief/04-surfaces/05-intent.md#1-intent-ids-kinds-and-lifecycle) "Bundle invariant" and AC bullet above. Worked example: `intent-capture-discipline` retirement (itd-27 + itd-30 scoped to different phases).
- **Discipline reclassification of a *shipped* intent.** Hypothetical: a standalone intent ships, then later we realise it was actually a discipline all along (rule applied to every subsequent spec anyway). Can `/abcd:intent reclassify <itd-N> --kind discipline` work post-ship? Probably yes, with the historical fidelity audit preserved as a `pre-reclassification-audit` field. Edge case; defer until first occurrence.
- **Bundle dissolution.** If two bundle-members ship as a bundle, then later one of them is superseded, what happens to the bundle? Recommend: the surviving member stays at `kind: bundle-member` with `bundle: <id>` but a new `bundle_dissolved: true` field; the superseded member moves to `superseded/` as usual.
- **Multi-bundle membership.** Can an intent belong to two bundles? Recommend no — bundle membership is exclusive. If two bundles overlap on an intent, that's a sign one of the bundles is mis-scoped.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer's single-document role when this intent moves to shipped/._

## References

- [`brief/01-product/03-mental-model.md`](../../brief/01-product/03-mental-model.md) — three intent kinds in the three-layer model.
- [`brief/04-surfaces/05-intent.md`](../../brief/04-surfaces/05-intent.md) — § 1 (kinds + discipline format + revisit triggers); § 2 (subcommands including `reclassify` and multi-arg `plan`); § 5 (lint rules); § 6 (reviewer's three roles).
- [`brief/05-internals/01-agents.md`](../../brief/05-internals/01-agents.md) — `intent-fidelity-reviewer`'s three roles documented; agent count stays 15.
- [`README.md`](../../../../README.md) — top-level "What is intent-driven development?" section gains a paragraph on the three kinds; `/abcd:intent` planned-commands table gains `reclassify` and multi-arg `plan` rows.
- 2026-05-07 audit conversation that surfaced the structural finding: ~40% of intents have non-1:1 relationships with at least one other intent.
- The discipline-kind framing is consistent across project types (framework projects produce more disciplines than application projects, but both produce some). Cross-project taxonomy revisit: see "Discipline subtypes are deferred" above.
