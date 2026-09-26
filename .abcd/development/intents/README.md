# abcd Intents

Intent specifications for the abcd plugin.

---

## What's an Intent?

An **intent** captures *what user-facing capability exists once shipped* — written as an Amazon working-backwards press release, not an engineering feature spec.

**Why press releases instead of feature specs:** Feature specs are engineering-shaped from the start (Problem → Design → Tasks). Press releases are user-experience-shaped (what *exists for the user* once shipped). Forcing intent capture in user-facing language disciplines product clarity before engineering scope.

**Plumbing work doesn't get intents.** Adapters, agents, harness, scaffolding — these have no user moment, and forcing press-release format on them produces strained or mis-targeted prose. Plumbing lives in the brief at `.abcd/development/brief/README.md`. See the brief for the full three-layer model (brief = whole picture, intents = user-facing why, specs = how).

This is a **codified abcd principle**: intent capture is press-release-shaped. abcd ships projects with this convention pre-baked.

---

## Intent IDs

Intent IDs follow the pattern `itd-N` (unpadded, mirrors the native spec `spc-N` convention). Filenames: `itd-N-<slug>.md`.

`itd` reads as "intent" and pairs visually with the native spec `spc-N`.

**IDs are capture-stable.** An intent keeps its `itd-N` for life — IDs are assigned in capture order and never renumbered. Sequencing is *not* encoded in the ID; it lives in the phase docs at [`../roadmap/phases/`](../roadmap/phases), whose `## Scope` sections are the single source of truth for which intents a phase bundles (see [adr-9](../decisions/adrs/0009-phase-as-product-layer.md)).

**Why unpadded:** abcd anticipates intent counts that would exceed any practical padding budget. Unpadded matches `spc-N` visually, avoids the future migration, and reads naturally in prose ("itd-7 spawned itd-19"). Lexical-vs-numeric sort is handled at tool layer (the record lint, registries, dashboards) rather than via filename padding.

---

## Three Intent Kinds

Every intent has a `kind` declared in frontmatter, set at `/abcd:intent plan` time. Three kinds exist (see [`brief/04-surfaces/05-intent.md § 1`](../brief/04-surfaces/05-intent.md#1-intent-ids-kinds-and-lifecycle) for the canonical reference):

| `kind` | Has press release? | Lives in | Maps to | Examples |
|---|---|---|---|---|
| `standalone` | Yes | `drafts/` → `planned/` → `shipped/` | One spec (1:1) | itd-3, itd-4, itd-7, most of the corpus |
| `bundle-member` | Yes | Same as standalone, with `bundle: <id>` linking members | Shared spec with bundle-mates (N:1) | itd-20, itd-24, itd-63, itd-69 (bundle `spc-83-operator-surfaces`, in `planned/` — unscheduled); see [Bundles](#bundles) |
| `discipline` | **No** — uses `## Rule` instead | `disciplines/` | No spec; imposes acceptance gates on every other spec | itd-1 (AC gate), itd-5 (prompt-quality) |

Standalone is the default (~60% of the corpus). Bundle-members ship together as one spec. Disciplines are cross-cutting rules with no user moment of their own; they apply to every other spec as inherited acceptance gates.

The kind is **project-agnostic** — application projects (e.g., a macOS app under abcd) produce their own disciplines (privacy-impact review, accessibility passes, code-style conventions). The three kinds are a property of the intent framework, not of abcd's particular subject matter.

**The persisted `kind` enum stays three-valued.** The capture-time classifier has a *fourth* verdict, `decision` (a standing infrastructure choice — "we use Postgres"), but `decision` is **never a persisted `kind`** and never enters this lifecycle: a confirmed `decision` routes to the existing ADR store (`../decisions/adrs/`, `adr-N-<slug>.md`), not to a draft. There is deliberately **no `intents/decisions/` directory**. See [itd-44](drafts/itd-44-fourth-intent-kind-decision.md) (spc-56 thin adoption) and `brief/04-surfaces/05-intent.md § 1`.

---

## Lifecycle Directories

| Directory | Status / role | Meaning |
|---|---|---|
| `drafts/` | 📝 Draft | Press-release-shaped intent captured but no native spec yet. Bench of ideas / forward-looking work. Cheap to draft and discard. |
| `planned/` | 📅 Planned | A committed capability awaiting its Go build — scheduled into a roadmap phase, or committed-but-unscheduled awaiting sequencing (the two axes are orthogonal, per [adr-34](../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md)). `spec_id` is `null` until the native spec layer schedules it (Phase 4), then points at a `spc-N`; bundle-member intents share a `spec_id` with their bundle-mates. |
| `shipped/` | ✅ Shipped | Linked spec closed. The "Audit Notes" section carries an OWED review receipt until `intent-auditor` runs and its ingest replaces the stub with per-criterion verdicts (per the itd-1 discipline) plus a three-bucket prose audit. |
| `disciplines/` | 📐 Active rule | Discipline-kind intents. Never get a native spec of their own; instead they impose acceptance gates that every *other* spec inherits and is checked against. **No `status` frontmatter field — the directory IS the state.** |
| `superseded/` | 🗄️ Superseded | Intents killed by reclassification or absorption. The file records `superseded_by: <handle>` (the successor — an intent, or the ADR that redecided the question the intent rested on) AND `kind_at_supersession: <original-kind>` (what shape the intent had when retired). The successor names the intent back in its own `supersedes`. Preserved as historical record. |

There is no `active/` state — "active" is implicit (a planned intent's linked spec is currently in flight in the native spec store; an active discipline is any intent in `disciplines/`).

**Directory location is the single source of truth for lifecycle state across all kinds.** Standalone and bundle-member intents derive state from `drafts/` / `planned/` / `shipped/`; disciplines derive state from `disciplines/` / `superseded/`. No intent has a `status` field that could disagree with its directory; the record lint enforces the contract.

**Delivered capability is represented by the intent leaving `drafts/` — nothing else says it.** An intent in `drafts/` is a captured idea nobody has committed to build, so no other document may report it as delivered: the CHANGELOG credits an intent only where the tree can back the claim. Two consequences follow, and the `delivery_state` record-lint rule holds both. An intent whose capability ships whole leaves `drafts/` on the ordinary path (`plan` mints or links its spec, `ship` moves it), and until it does, the shipment is only half-recorded. An intent whose capability ships in *part* stays in `drafts/` and is not cited at all — the entry describes what shipped without naming it, because an intent is delivered whole or not at all ([Split-the-intent doctrine](#bidirectional-link-convention)) and a partial citation reads as a promotion that never happened.

---

## Lifecycle (three transition verbs, with deliberate manual steps)

Each step is marked `[shipped]` or `[design target]` on the same convention the
brief's [surface registry](../brief/04-surfaces/README.md) uses. A design target
is aspiration the binary does not yet reach.

```
1. /abcd:intent "<free-text>"                                        [shipped]
   ├─ Assigns next itd-N ID (capture-stable — never renumbered)
   ├─ Writes intents/drafts/itd-N-<slug>.md, seeded from the text
   ├─ Seeds a PLACEHOLDER `## Acceptance Criteria` section — the itd-1
   │  refusal fires at plan time, not here
   ├─ Optional --impact <additive|breaking|fix>, validated and stamped
   └─ The press-release interview is host-run (commands/intent.md), not
      performed by the binary
      └─ LLM classifier writing advisory `suggested_kind`      [design target]

2. /abcd:intent plan <itd-N>            (the maintainer's sign-off act)
   │
   ├─ Refuses a HELD record before anything moves — `held: "<reason>"`,
   │  written by /abcd:intent hold <itd-N> --reason "<one line>" and removed
   │  by /abcd:intent unhold <itd-N>; the refusal names the reason and
   │  the lift (drafts/ and planned/ only)                            [shipped]
   │
   ├─ standalone (exactly one intent ID):                            [shipped]
   │     ├─ Refuses if `## Acceptance Criteria` is missing or empty (itd-1)
   │     ├─ Writes binding `kind:` — defaults to standalone
   │     ├─ Mints the spec, injects the bidirectional link
   │     │  (spec.intent = itd-N; intent.spec_id = spc-N)
   │     └─ drafts/ → planned/
   │
   ├─ Kind proposal + user confirmation, plan-review        [design target]
   │
   ├─ bundle-member (multiple intent IDs in one call)       [design target]
   │
   └─ discipline (registers gates in .abcd/disciplines/)    [design target]

3. Implementation                                                    [manual]
   ├─ `intent ready <itd-N>` gates it (exit 0 proceed / 1 SKIP / 2 fault)
   └─ There is NO `intent ship` verb; the work is done in a session
      └─ /abcd:intent ship                                   [design target]

4. /abcd:spec close <spc-N>                                          [shipped]
   ├─ Closes the spec (open/ → closed/) and, in the same synchronous call,
   │  reconciles the linked intent planned/ → shipped/
   │  (intent.Reconcile — there is no background hook)
   └─ Emits an OWED review receipt into the intent's `## Audit Notes`
      plus an ephemeral request under .abcd/.work.local/reviews/
      (report-only: a failure here never blocks the ship)

5. Fidelity verdict — three deliberate steps                         [shipped]
   ├─ /abcd:intent audit <itd-N>        re-emits the request
   ├─ host runs intent-auditor (single-document role)
   └─ /abcd:intent audit ingest --verdict-json <f>
      ├─ Validates FAIL-CLOSED against the schema and the parked receipt
      ├─ INGESTED: replaces the OWED stub with per-criterion verdicts
      │  (MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE) plus the
      │  three-bucket prose audit (honoured / diverged / missing)
      └─ DEAD_LETTER: quarantines the payload — never a partial apply

      The receipt digest is sha256 over the `## Acceptance Criteria` body
      alone, so writing the verdict cannot change it and re-ingest stays
      idempotent. Editing the criteria after shipping invalidates the
      receipt — a stale verdict is detectable, by design.

6. /abcd:intent reclassify <itd-N> --kind <new-kind>          [design target]
   └─ Including the --kind superseded --by <handle> supersession path.
      The intents in superseded/ were moved by hand.
```

**Shipped verb set:** `intent "<text>"`, `intent plan`, `intent hold`,
`intent unhold`, `intent ready`, `intent link`, `intent audit`,
`intent audit ingest`, and `spec close`.

**Manual overrides:**

- `/abcd:intent link <itd-N> <spc-N>` for retroactive linking of pre-existing specs
- `/abcd:intent audit <itd-N>` to re-emit the fidelity request at any time (Role 1 — single-doc fidelity)

**No `/abcd:intent move`** — file location follows verb side-effects, not user intervention.

---

## Bidirectional Link Convention

| File | Frontmatter field |
|---|---|
| `intents/{drafts,planned,shipped}/itd-N-<slug>.md` | `spec_id: spc-N` (scalar, or `null` when in drafts/ — **never a list**) |
| the native spec `spc-N-<slug>` | `intent: itd-N` (or list — one spec may consume several intents; that is the bundle direction) |

Both directions present once `/abcd:intent plan` runs. The record lint (pre-commit + CI) verifies they agree, and rejects a list-valued `spec_id`.

**Split-the-intent doctrine.** An intent is the unit of consumption: it is implemented by exactly one spec. Work too big for one spec decomposes into *tasks inside* that spec; an intent containing two separately verifiable promises is two intents — split it (precedent: the launch PRD's Tier A/B split into itd-67 and itd-72). This keeps the close hook singular, coverage computable, and doneness unambiguous — an intent can never be half-consumed and called done.

---

## Press Release Format

Every intent file uses this template (spc-3 fields shown; all new fields are optional — pre-existing intents without them remain valid):

```markdown
---
id: itd-N
slug: <kebab-case-slug>
# NOTE: no `status:` field — directory location is the canonical lifecycle state.
#   See brief/04-surfaces/05-intent.md § 6 for the lint rule and rationale.
kind: null               # set by /abcd:intent plan: "standalone" | "bundle-member" | "discipline"
spec_id: null
# held: "<reason>"       # written by /abcd:intent hold, removed by /abcd:intent unhold (drafts/ and
#                        #   planned/ only); one non-empty line. /abcd:intent plan and /abcd:spec close refuse while it stands
# spc-3 fields (optional; additive — pre-existing intents valid without them):
contexts: null           # [list] of bounded-context IDs; required when term has cross-context collision
glossary_terms_used: null  # [list] of qualified <context>/<term> IDs; auto-populated by grill skill
warrants_assumed: null   # [list] of Toulmin warrants assumed (not made explicit in AC)
grilled_at: null         # ISO8601 UTC; set by grill skill at Phase 1 completion
grill_session_id: null   # UUIDv4; set by grill skill
grilled_intent_hash: null  # SHA-256 of intent at grill time (intent_source_hash recipe)
prd_path: null           # relative path to PRD (e.g. .abcd/intents/itd-N/prd.md); set by grill Phase 2
prd_grandfathered: null  # true = pre-spc-3 planned intent; GR002+GL005 suppressed-as-info
---

# <Headline — what user-facing capability exists>

## Press Release

> **abcd ships with <capability>.** <2-4 sentences describing what users can now do, in present tense as if shipped.>
>
> "<Customer quote — picked from personas.json>," said <persona> <role>.

## Why This Matters

<1-2 paragraphs explaining the underlying user need.>

## What's In Scope

- <Bullet>

## What's Out of Scope

- <Bullet — preventing scope creep>

## Acceptance Criteria

> _Required (per itd-1). At least one Given-When-Then bullet. Hard-blocked at /abcd:intent plan time if missing or malformed._

- **Given** <preconditions>, **when** <user/system action>, **then** <observable outcome>.

## Prior Art

> _Required. Positions the intent against the existing corpus: what it builds on, what almost covers it, why it is nonetheless its own intent. At least one resolvable reference (sibling intent, brief section, principle, ADR, or external source); "none found — searched <where>" is a valid entry, an empty section is not._

- <Reference + one line on the relation>

## SOTA

> _Required once planned (per [sota-per-intent](../principles/sota-per-intent.md); `intent_sota` warns on a planned intent without it). The existing alternatives, each one's rough maturity, and the path taken: 1 adopt the alternative, 2 a native floor with a seam for it, 3 bespoke with no swap possible._

- <Alternative — maturity>

**Path:** <1, 2 or 3, and why>

## Open Questions

- <Bullet — anything not yet decided>

## Audit Notes

<Empty until intent moves to shipped/. intent-auditor populates this with per-criterion verdicts plus a three-bucket prose audit comparing delivered reality to the press release above.>
```

---

## PRD Freeze Contract (spc-3)

When `/abcd:intent plan <itd-N>` runs, the PRD at `prd_path` is **frozen**:

1. `frozen_content_hash` is computed from the PRD body + stable frontmatter fields (provenance fields INCLUDED; `frozen_at`, `frozen_content_hash`, `spec`, `planning_attempt_id` EXCLUDED).
2. `frozen_at`, `frozen_content_hash`, `planning_attempt_id` are written atomically to the PRD.
3. Mutating the frozen PRD after promotion triggers `GR003` (blocker lint).

The freeze is **non-self-referential**: re-computing the hash on the frozen PRD (excluding the freeze fields) yields the same value. Mutating `body_markdown` or any included frontmatter field changes the hash; mutating `frozen_at`, `frozen_content_hash`, `spec`, or `planning_attempt_id` does NOT change the hash.

**Hash recipes** — two distinct recipes documented in `prd.schema.json`:
- `intent_source_hash` recipe: used at grill time and plan-time provenance check (hashes the parent intent).
- `frozen_content_hash` recipe: used at freeze time (hashes the PRD body + stable provenance fields).

---

## Customer Quotes — Persona Convention

Customer quotes use placeholder personas from `.abcd/development/personas.json` (Alice, Bob, Carol, ... — a fixed alphabetical sequence). Selection is **by role, never by name**: the intent's audience picks the role; the role's registered name is used. Every persona is they/them.

This is a discipline ([`disciplines/itd-79-persona-registry.md`](disciplines/itd-79-persona-registry.md), enforced by the `persona_registry` record-lint rule): never use real names in press releases (PII), but never use generic "a hypothetical user" language (loses voice). Named personas keep quotes grounded without leaking real-world identifiers.

---

## Sequencing — see `phases/`

Which intents a phase bundles, and in what order phases ship, is **not recorded here.** Sequencing lives in the phase docs at [`../roadmap/phases/`](../roadmap/phases) — each phase doc's `## Scope` section is the single source of truth (per [adr-9](../decisions/adrs/0009-phase-as-product-layer.md)). An intent listed in no phase doc's `## Scope` is implicitly **unscheduled** — valid for `drafts/` and `planned/` alike: a draft is uncommitted, an unscheduled planned intent is committed but awaiting sequencing. The invariant runs one way only: any intent a phase `## Scope` names is committed by definition and lives in `planned/` (or `disciplines/`) — see [adr-34](../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md).

This README describes the intent corpus by *lifecycle state* — what each lifecycle directory means and how an intent moves between them. It transcribes neither the phase→intent mapping nor the directories' contents: both have a home that cannot go stale, and a copy kept here would (adr-5, derive don't store).

---

## Bundles

Active bundles (sets of intents that ship as one shared spec via multi-arg `/abcd:intent plan`):

| Bundle ID | Members | Why a bundle |
|---|---|---|
| `spc-83-operator-surfaces` | itd-20, itd-24, itd-63, itd-69 (all `planned/`, `spec_id: null`) | The operator-facing surface set that couples through the plugin manifest/metadata lockstep — top-level dispatcher (itd-20), reflect (itd-24), setup-wizard (itd-63), and plugin-metadata lockstep (itd-69) share one spec. Committed but unscheduled: named in no phase doc's `## Scope`, and `spec_id` is still `null` (the shared spec is not yet minted). |
| ~~`tier-0-audit-substrate`~~ (dissolved 2026-05-07) | ~~itd-31 + itd-32~~ | The bundle premise (unified `/abcd:audit` surface bundling all review/audit roles into one verb's subverbs) was dissolved when the round-2 command-structure review split the three intent-fidelity-reviewer roles into three distinct verbs under `/abcd:intent` (review/consistency/shape). itd-31 promoted to standalone; itd-32 superseded by itd-31. |

Bundles are declared in member intents' frontmatter (`bundle: <bundle-id>`); membership is bidirectional (verified by the record lint). When a bundle's shared spec closes, all member intents move from `planned/` to `shipped/` together.

**Note on cross-phase bundle attempts:** the `intent-capture-discipline` bundle (itd-27 + itd-30) was retired. The bundle invariant requires *one shared spec shipped together* — and per [adr-9](../decisions/adrs/0009-phase-as-product-layer.md) all bundle members must belong to the same phase. itd-27 has a plan-reviewed spec (`spc-3`); itd-30 is unscheduled. Both intents were reclassified to `standalone`; if itd-30 is later picked up, its spec can depend on or extend `spc-3` for shared interview/lint/persona-registry plumbing without needing the bundle declaration.

---

## Drafts

Captured intents that haven't been promoted to native specs yet. Each standalone or bundle-member intent moves to `planned/` once the user runs `/abcd:intent plan <itd-N>`; discipline-kind intents move to `disciplines/`. For the sequencing view — which phase bundles which intents — see [`../roadmap/phases/`](../roadmap/phases).

The bench itself is [`drafts/`](drafts/) — the directory is the listing, and a transcription of it here is a second copy that drifts on the next capture or promotion (adr-5, derive don't store).

---

## Disciplines

Active discipline-kind intents (cross-cutting rules with no user moment of their own; impose acceptance gates that every other spec inherits). They never get a native spec — they ARE the rule, not a feature being built. **No `status` frontmatter field — presence in `disciplines/` IS the active state; supersession moves to `superseded/`.**

The active rule set is [`disciplines/`](disciplines/); each file states its own rule, which is the description a listing here could only paraphrase.

See [`brief/04-surfaces/05-intent.md § 1`](../brief/04-surfaces/05-intent.md#1-intent-ids-kinds-and-lifecycle) "Discipline format" for the template (no press release; uses `## Rule` + `## Why` + `## Acceptance Criteria` instead; no `status` field).

**Discipline subtypes** (e.g. methodology / documentation / audit) are deferred — see the revisit triggers in the brief. For now each discipline declares a free-text `kind_notes` field describing what kind of rule it is.

---

## Planned

`planned/` holds the committed capabilities awaiting their Go build — some scheduled into a roadmap phase, others committed but not yet sequenced (per [adr-34](../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md)). Their `spec_id` is `null` until the spec layer schedules them (Phase 4).

The commitment set is [`planned/`](planned/); which of those a phase has sequenced is the phase docs' `## Scope` sections, not a list here.

---

## Superseded

The retired set is [`superseded/`](superseded/); each file names its own successor and the shape it had when it was retired, so the directory answers "what happened to itd-N" without a roll-call here.

Intents move here when they are killed by reclassification or absorption — e.g., when a smaller intent is folded into a larger one, or when a discipline is replaced by a stricter successor. The move is made by hand; `/abcd:intent reclassify <itd-N> --kind superseded --by <itd-M>` is a design target. Each superseded intent records two fields:

- **`superseded_by: <handle>`** — the successor. Usually a later intent (`itd-M`); an ADR (`adr-N`) when the decision the intent rested on is redecided rather than the capability re-scoped. The successor carries the other half of the pair, `supersedes: [<itd-N>]`, and `record_schema` blocks a supersession declared from one side only.
- **`kind_at_supersession: <original-kind>`** — what shape the intent had when retired (`standalone`, `bundle-member`, or `discipline`)

The original-kind preservation matters: "superseded" means different things depending on what the intent *was*. A superseded standalone is a retired capability. A superseded bundle-member is a retired half of a coupled pair. A superseded discipline is a retired rule that was inherited by every other spec. Without `kind_at_supersession`, future archaeology has to reconstruct the original shape from `reclassification_history` — which exists but is harder to query.

Files in `superseded/` are preserved as historical record; never deleted.

---

## Shipped

[`shipped/`](shipped/) holds the capabilities built in Go. An intent moves here automatically when its linked spec closes; the close hook emits an OWED review receipt into its "Audit Notes" section, which `intent-auditor` later replaces with the per-criterion verdicts.
