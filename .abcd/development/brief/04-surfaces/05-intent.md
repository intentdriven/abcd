# `/abcd:intent` — Press-Release Intent Capture

> **Delivery state**: the `intent` binary verb ships — bare invocation is read-only status, plus the sub-verbs this chapter's generated appendix lists, and the quoted-text create path `abcd intent "<text>"` (itd-46, itd-80, itd-94). The implement-readiness gate runs seven checks on one intent (bucket, acceptance criteria, mechanism claim, scope conditions, spec link, spec body, recorded grounds); exit 0 ready / 1 not / 2 fault — and recorded grounds are the one thing it writes: the conjecture behind the gate decision, appended to the intent's `## Grounds` section. The `/abcd:intent` plugin command surface exists (`commands/intent.md`, resolving iss-105) and carries the planning interview an unready intent is routed to. Remaining backing intents sit in `intents/planned/` (itd-27 grill, itd-34 kinds, itd-48 reviewer roles 2–3, itd-50 audit loop) and `intents/drafts/` (itd-16, itd-35 — the `/abcd:audit` sub-verbs); delivery state is the intent lifecycle's, not this page's (see the [brief README's provenance note](../README.md)).

abcd uses **intents** in press-release format (Amazon working-backwards) as the unit of forward-looking *user-facing* planning. Intents capture *what user-facing capability exists once shipped*, written in present tense as if already delivered. This is engineered to discipline product clarity before scope creep — a reader of an intent thinks like a product person first, an engineer second.

Plumbing work (adapters, agents, harness, scaffolding) lives in this brief, not in intents — see [`01-product/03-mental-model.md`](../01-product/03-mental-model.md) for the rationale.

Intents live at `.abcd/development/intents/{drafts,planned,shipped,disciplines,superseded}/`. Five directories encode lifecycle position. Three are for press-release-shaped intents (the standalone + bundle-member kinds); one is for disciplines (no press release, no spec); one is for intents killed by reclassification.

- **`drafts/`** — press-release-shaped intent captured but no native spec yet. Bench of ideas / forward-looking work. Cheap to draft and discard.
- **`planned/`** — a committed capability, scoped into a roadmap phase and awaiting its Go build. Its `spec_id` is `null` (unscheduled) or points at a `spc-N` once the spec layer schedules it (Phase 4). The native spec store ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)) is the scheduling home. Bundle-member intents in `planned/` share a `spec_id` with their bundle-mates.
- **`shipped/`** — a capability built in Go, moved here on the close after which no open spec names it. An intent owns one or more specs, so the transition is the LAST close, not any close: while a remainder spec is open the intent stays in `planned/`, and a partial delivery is announced by nothing. The intent's "Audit Notes" section holds drift findings (per-criterion verdicts: `MET`, `MET_WITH_CONCERNS`, `NOT_MET`, `INCONCLUSIVE`) once `intent-auditor`'s Role 1 has run on it through the audit sub-verb; that review is owed once per intent and its request names every spec that realised it.
- **`disciplines/`** — discipline-kind intents (cross-cutting rules with no user moment). They never get a native spec of their own; instead they impose acceptance gates that every *other* spec inherits and is checked against. Disciplines have no `status` frontmatter — presence in this directory IS the active state. Superseded disciplines move to `superseded/`.
- **`superseded/`** — intents killed by reclassification or absorption (e.g., when a smaller intent is folded into a larger one, or a discipline is replaced by a stricter successor). The file records `superseded_by: <handle>` (the record that formally supersedes this intent — an intent, `itd-N`, or an ADR, `adr-N`, when a decision redecided the question) AND `kind_at_supersession: <original-kind>` (what shape the intent had when retired — standalone vs bundle-member vs discipline). Preserved as historical record; never deleted.

There is no `active/` state — "active" is implicit (a planned intent's linked spec is currently in flight in the native spec store; an active discipline is any intent in `disciplines/`).

**The store every verb addresses is the checkout's, from anywhere in the tree.**
The front door resolves the checkout root before it reads or writes, through
[`gitutil.CheckoutRoot`](../../../../internal/gitutil/repo.go) — the same
resolution `capture` addresses its ledger through and `decide` its decision
store. It is git's toplevel where git will name one, and a refusal in the two
remaining states rather than a guess: a repo-shaped tree git will not answer for,
and no repository above at all. Outside a checkout every `intent` verb therefore
exits **2** and does nothing, because there is no intent store to address and
laying one where the caller stood produces a draft no gate, no release cut and no
reader ever sees — and one whose spec can never be closed against it, because the
reconcile step looks in the checkout. It deliberately does not fall through to a
marker walk, which would accept any directory carrying the name
(iss-2609090947359464). Resolving is a question and not a write, so bare
invocation stays read-only.

An intent store found **below** the checkout root, on the chain between the
caller and it, is named on stderr and left untouched — the deposit an unresolved
front door leaves behind, reported to the person standing over it rather than
stepped over in silence. The note states what is there; moving a record is a
judgement no verb makes.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `hold` | — | shipped |
| `link` | — | shipped |
| `new` | — | shipped |
| `plan` | — | shipped |
| `unhold` | — | shipped |
| `ready` | gate | shipped |
| `audit` | audit | shipped |
| `audit ingest` | audit | shipped |


## 1. Intent IDs, kinds, and lifecycle

### Intent IDs

- **ID format:** `itd-N` (unpadded — e.g., `itd-1`, `itd-15`). Mirrors the spec store's `spc-N` format. Filenames: `itd-N-<slug>.md`. Lexical-vs-numeric sort handled at the tool layer (`internal/core/lint`, registries) rather than via filename padding.
- **The low IDs (itd-1..itd-7) reflect an early one-time rebase to ordering signal.** Intents created since are capture-stable, picking up at itd-27+. ID number is *not* an execution-order guarantee — the canonical build order is the phase plan at [`roadmap/phases/`](../../roadmap/phases/README.md).
- **An intent carries no release or sequencing field.** Per [adr-9](../../decisions/adrs/0009-phase-as-product-layer.md), an intent's sequencing is its *phase membership*, recorded editorially in the owning phase doc's `## Scope` — not in intent frontmatter. An intent not yet listed in any phase doc is implicitly unscheduled (a `drafts/` bench item). "Which release" is an output of completing phases, never an input stamped on a draft; the former `target_release` field was removed for this reason.

### Intent kinds (per [`01-product/03-mental-model.md`](../01-product/03-mental-model.md))

Every intent has a `kind` declared in frontmatter, set at planning time. Three kinds:

| `kind` | Has press release? | Lives in | Maps to | Examples |
|---|---|---|---|---|
| `standalone` | Yes | `drafts/` → `planned/` → `shipped/` | One spec (1:1) | itd-3, itd-4, itd-7, most of the corpus |
| `bundle-member` | Yes | Same as standalone, with `bundle: <id>` linking members | Shared spec with bundle-mates (N:1) | itd-20, itd-24, itd-63, itd-69 (bundle `spc-83-operator-surfaces`, in `planned/` — committed but unscheduled, named in no phase doc); see [`intents/README.md`](../../intents/README.md#bundles) |
| `discipline` | **No** — uses `## Rule` instead | `disciplines/` | No spec; imposes acceptance gates on every other spec | itd-1 (AC gate), itd-5 (prompt-quality) |

**`kind` is binding once set at plan time.** Late changes go through a reclassify step — a later phase (no reclassify sub-verb ships; the generated appendix lists what the binary exposes) — which records the change in the intent's frontmatter `reclassification_history` and surfaces it for reviewer review.

**Two distinct history fields, two distinct concerns:** `reclassification_history` records *kind* transitions (standalone ↔ bundle-member ↔ discipline ↔ superseded). `surface_history` records *surface-shape* transitions where the kind is unchanged but the user-facing surface form changes (e.g., skill → sub-verb, top-level command → sub-verb of another command, command → flag). Both are append-only; both have the same `{ date, from, to, reason }` shape. Worked example: itd-27 was always `kind: standalone`, but its surface shifted from a top-level skill (`/abcd:grill`) to a sub-verb of `/abcd:intent` on 2026-05-07 — that's a `surface_history` entry, not a `reclassification_history` entry, because the kind is unchanged. The two fields together preserve a complete audit trail of an intent's evolution without overloading either.

**`suggested_kind` is advisory.** The quoted-text create seeds `suggested_kind: null`; the field is an optional soft hint a human may set later, informational only — the binding decision is at plan time.

**A fourth capture verdict, `decision`, routes to the ADR store (itd-44 — a later phase).** A later-phase capture-time classifier can also emit `decision` — a *standing infrastructure choice* (no user moment, not a per-artefact rule, e.g. "we use Postgres"). `decision` is a **capture verdict only, never a persisted `kind`**: the `kind` / `kind_at_supersession` enums stay three-valued (`standalone` / `bundle-member` / `discipline`). A confirmed `decision` DIVERTS capture to the existing ADR store (`.abcd/development/decisions/adrs/`, `NNNN-<slug>.md`, zero-padded) instead of writing an intent draft — no spec, no lifecycle directory, no `intents/decisions/`. The verdict is advisory: capture confirms "capture as an ADR?" or overrides to a normal draft carrying a *plannable* `suggested_kind` (`null`/`standalone`, never `decision`). In that design `suggested_kind: decision` is a legal value that the plan and reclassify paths refuse ("decisions are not plannable"). Nothing enforces it today, and there is nothing yet to enforce it against: the tree holds no intent schema file, the create seed writes `suggested_kind: null`, and no shipped code reads the field at all. Planning never consults it, defaulting a null `kind` to `standalone`. See [itd-44](../../intents/drafts/itd-44-fourth-intent-kind-decision.md) and the [ADR store README](../../decisions/adrs/README.md).

**Bundle invariant: all members of a bundle MUST belong to the same phase.** A bundle ships as one shared spec; one spec belongs to one phase. Cross-phase bundles are structurally impossible — the shared spec cannot live in two phases at once. Planning several intents at once (multi-arg, kind=bundle-member) hard-blocks promotion when the proposed members are scoped to different phases. The only resolutions are: (a) re-scope all members into the same phase before re-running plan, or (b) downgrade one or more members to `kind: standalone` so they ship independently. Worked example: the `intent-capture-discipline` bundle (itd-27 + itd-30) was retired on 2026-05-07 precisely because the two members were scoped to different phases — both intents reclassified to standalone (see their `reclassification_history` entries for the full reasoning). Lint code: `IL011` per [`05-internals/06-lint.md`](../05-internals/06-lint.md) (plan-time tooling, a later phase — the shipped record lint has no bundle check).

### Discipline format

Discipline-kind intents skip the press release entirely (no user moment to describe; no customer quote to attribute). They also have no `status` field — the directory IS the state (`disciplines/` = active; `superseded/` = retired). The format is:

```markdown
---
id: itd-N
slug: <kebab-case>
kind: discipline
kind_notes: "<free-text describing what kind of discipline this is — e.g.,
              'cross-cutting acceptance-criteria gate, applied via lint and auditor'>"
suggested_kind: <null, or the advisory hint a human set>
spec_id: null          # required: a discipline never gets a spec
reclassification_history: []
severity: <minor|major|critical>
---

# <Headline — what rule this imposes on every spec>

## Rule

<One-paragraph statement of the rule, in present tense.>

## Why

<2-3 paragraphs: the failure mode this prevents; the cost of not having the rule;
the prior art (other projects' versions of this discipline).>

## What's In Scope

<Bullets — what every other spec must do/include because of this rule.>

## What's Out of Scope

<Bullets — what this discipline does NOT cover.>

## Acceptance Criteria

> _Asked of disciplines too (per the itd-1 discipline itself): at least one
> Given-When-Then bullet describing how the rule is checked. Nothing checks it
> here — a discipline never passes the plan step, which is where the acceptance
> refusal lives, so this section is held by hand and several disciplines carry
> none (§ 6)._

- **Given** <preconditions>, **when** <spec event>, **then** <gate behaviour>.

## Audit Notes

<Empty until first spec ships under this discipline. The shape-classification
role of intent-auditor populates findings here.>

## References

<Citations to brief sections, related intents, prior art.>
```

**Two body idioms are in the tree.** The template above is what the first seven
disciplines follow. The seven written since (itd-190 onwards) head the rule
`## The rule` and carry `## The gate`, `## Fit` and `## Staging` in place of the
scope, acceptance and reference sections: a shorter shape that says what the rule
is, what holds it, where it fits and how far it is armed. Nothing arbitrates
between them, because no gate reads a discipline's body at all. Settling on one
is open work.

**Discipline subtypes come later.** The `kind_notes` field is free-text deliberately. This is a **deferred** capability — Discipline subtype taxonomy is NOT a live Role 3 `suggestion_type` (the three live types are `kind_change`, `bundle`, `supersession`). The subtype taxonomy (e.g., a closed enum of methodology, documentation, audit and convention) moves from free-text to formal enum when ANY of the following:

1. **Three or more disciplines exist** with `kind_notes` describing similar shapes.
2. **A user is confused** about which existing discipline a finding belongs to.
3. **`intent-auditor`'s shape-classification role cannot describe a proposed discipline** without more constraint than `kind_notes` provides.
4. **Cross-project usage** — once abcd is in three or more projects, comparing disciplines across them needs a shared subtype vocabulary.

Until then, `kind_notes` is the free-text descriptor.

**The first trigger has fired, and the enum has not been drawn.** Fourteen
disciplines are in the tree and every one of them carries a `kind_notes` value,
against a threshold of three. What is outstanding is the judgement the trigger
asks for: whether those descriptors fall into shapes a closed enum could name.
Drawing it is open work, so this section describes a deferral that has outlived
its own condition rather than one still waiting on it.

### Lifecycle

- **The low IDs (itd-1..itd-7) reflect an early one-time rebase to ordering signal.** Intents created since are capture-stable, picking up at itd-27+. ID number is *not* an execution-order guarantee — the canonical build order is the phase plan at [`roadmap/phases/`](../../roadmap/phases/README.md).
- **An intent carries no release or sequencing field.** Per [adr-9](../../decisions/adrs/0009-phase-as-product-layer.md), an intent's sequencing is its *phase membership*, recorded editorially in the owning phase doc's `## Scope` — not in intent frontmatter. An intent not yet listed in any phase doc is implicitly unscheduled (a `drafts/` bench item). "Which release" is an output of completing phases, never an input stamped on a draft; the former `target_release` field was removed for this reason.
- **Lifecycle (automated, not user-managed):**

```
1. Create: the quoted free text is the whole operand   (canonical bare quoted create)
   ├─ Mints a timestamp-numeric itd-N id through the shared record-id seam
   │   (adr-45: no maximum consulted, capture-stable once assigned)
   ├─ Seeds the draft skeleton: frontmatter (kind: null, suggested_kind: null,
   │   spec_id: null, reclassification_history: [], builds_on: [], severity: minor)
   │   and a body whose `## Press Release` IS the quoted text, as prose, under an
   │   H1 that is the text's first sentence (or a title the caller gives), with
   │   `## Why This Matters` and `## Acceptance Criteria` (per itd-1) seeded as
   │   prompts for the human to fill before planning
   └─ Writes intents/drafts/itd-N-<slug>.md (no spec created yet)

2. Plan: one intent id, optionally with its impact judgement    (when ready to commit to work)
   ├─ Refuses a HELD record before anything moves, naming the reason and
   │  the unhold that lifts it (the identity-only re-run on a held planned record refuses too)
   ├─ Refuses to promote if `## Acceptance Criteria` is missing/malformed (the intent package's own hasAcceptanceCriteria check, not internal/core/lint)
   ├─ Settles the `impact` judgement when the caller gives one — validated at the create path's bar, refused
   │  when it disagrees with a judgement the record already holds, a no-op when it agrees — before any write
   ├─ Mints (or reuses) the intent's native spec; kind defaults to standalone
   ├─ Stamps kind (and the impact, when supplied) onto the draft, then injects the bidirectional link (spec.intent: itd-N; intent.spec_id: spc-N)
   └─ Moves intents/drafts/itd-N-*.md → intents/planned/itd-N-*.md

   A hold is the one state a draft or planned record carries beside its bucket:
   holding a record with a one-line reason writes `held: "<reason>"`
   (required, single-line, redacted through the store's scanner) and
   unholding it removes the line; a record already held is
   refused naming the standing reason, a record not held is refused by unhold,
   and both refuse shipped/, superseded/ and disciplines/. A hold blocks every
   lifecycle move until unhold: closing a spec realising a held
   planned record refuses too, before anything moves, and never strips the
   key. `abcd <itd-N>` reports the hold as the first next move. The verb writes the value; a hand-typed legal
   line is byte-identical to that write and stops plan the same way, and the
   record_provenance lint rule reports only a `held` value in a shape no verb
   writes (blank, null, a list, a map, a block scalar, or a legal value in a
   bucket the verbs refuse), which plan refuses too — fail closed.

   Later phase — plan grows a PRD-freeze front end and multi-kind dispatch:
     a prd_path read + provenance freeze sequence (§ 5); a suggested_kind-driven
     kind proposal the user confirms or overrides (binding); a plan-review of the
     stub; a multi-arg bundle-member branch (plan <itd-A> <itd-B> …, one shared
     spec across members); and a discipline branch (no spec — registers the rule
     in a disciplines gate store, moves drafts/ → disciplines/). None of these
     ship today; plan schedules a single standalone intent.

3. Spec marked done in the native spec store   (standalone + bundle: work complete)
   ├─ a MANUAL step, run in the same change that lands the work: closing the spec with the `spec` verb (CLI-only; no hook
   │  runs it; a change that declares `Delivers: itd-N` is refused at merge by RS005 in
   │  scripts/check-issue-resolution.sh until the intent enters shipped/ in that change, naming every open
   │  spec that names it; a change declaring nothing is refused nothing, and a planned intent whose code is
   │  on main is invisible to the launch cut, which composes
   │  only from terminal folders — the cut exits 0 without it; the intent's `impact` is required, supplied by the
   │  record or by the caller on the close, and a close with neither is refused; the verb
   │  resolves the checkout root before it reads the store, so it addresses the checkout's spec store from any
   │  directory in the tree and exits 2 outside a repository, where there is no spec store to address)
   ├─ native spec-store close-hook (spc-36, predecessor store) → intent lifecycle reconcile (spc-28, predecessor store)
   └─ Moves intents/planned/itd-N-*.md → intents/shipped/itd-N-*.md (+ enqueues a review)
       (For bundles, all member intents move together when the shared spec closes.)

   Then, as a separate MANUAL step — run the audit on the intent:
   └─ intent-auditor agent (single-document role / Role 1 itd-1 pass)
       └─ Compares as-shipped reality against original press release + acceptance criteria
           └─ Per-criterion verdicts (MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE)
               written to the intent file's "Audit Notes" section (verdict of record),
               with the review request staged under .abcd/.work.local/reviews/.
               For bundles, review runs per-intent (each member's acceptance criteria
               checked separately against the same delivered reality).
   (spc-12 (predecessor store) ships only this MANUAL review surface. spc-28 (predecessor store) ships the on-close hook
    that moves the intent planned → shipped and QUEUES a review on that transition.
    Auto-running the reviewer off that queue is still deferred (no epic currently owns
    it; spc-6 (predecessor store) disowned auto-firing). Until then, the `## Audit Notes` of a freshly
    shipped intent stays empty until the audit is run by hand.)

4. Reclassify: one intent id and its new kind     (late reclassification — a later phase; no reclassify sub-verb ships yet)
   ├─ Records the change in intent.reclassification_history (date + from-kind + to-kind + reason)
   ├─ Moves the file between directories (e.g., drafts/ → disciplines/) as needed
   ├─ For supersession: the new kind superseded, naming its successor, moves the file to superseded/,
   │   writes superseded_by: <handle> (an intent, itd-M, or an ADR, adr-M, when a
   │   decision redecided the question), AND captures the original kind in
   │   kind_at_supersession: <original-kind> (so future readers know what shape the
   │   intent had when it was retired — standalone vs bundle-member vs discipline
   │   change the meaning of "superseded")
   └─ Triggers intent-auditor (shape-classification role) to verify the new kind fits

Later phase — intent-auditor (shape-classification role) scans the corpus
              when the user runs /abcd:intent shape (spc-29, predecessor store; an on-demand surface, not
              yet a binary sub-verb). The user accepts a suggestion via
              the reclassify step; declined suggestions become entries in the
              intent's Audit Notes for future review. (Deferred follow-up: scheduled /
              pre-commit shape scanning — the shape(...) function's mode="pre_commit"
              parameter is a preserved seam that no hook invokes.)
```

## 2. Subcommands

| Subcommand | Purpose | File movement |
|---|---|---|
| `/abcd:intent` (no args) | Read-only status: bucket counts (drafts / planned / shipped / disciplines / superseded), open/closed spec counts, the itd↔spc links, a ledger-routing hint (`abcd capture "…"` for an observation, `abcd intent "…"` for a user-facing change), and an ideate-routing line (a big, unproven idea? `abcd ideate` runs the optional admission gauntlet and records the verdict either way) | — |
| `/abcd:intent "<free-text>"` | **Canonical create** (spc-30 (predecessor store)/itd-46): a leading quoted seed is the canonical create entry. Seeds a draft skeleton whose `## Press Release` is the quoted text as prose, under an H1 derived from the text's first sentence (cut on a word boundary at the slug cap) or given as a title — one line, non-empty, redacted like the text — with Why This Matters and Acceptance Criteria seeded as prompts for the human to fill; assigns `itd-N` and derives the slug from the text; writes `suggested_kind: null`. An optional impact (additive, breaking or fix) stamps the draft's product impact at create time, and an optional production mode (hand-written, dictated-and-formatted or scribe-transcribed) stamps how its text was produced (itd-178); the draft's `origin` carries no flag and is derived from the verb that ran. A leading quote always creates — never falls through to bare render | writes to `drafts/itd-N-<slug>.md` (no spec created) |
| Deprecated create alias | Deprecated alias for the quoted-text create (`abcd intent "<text>"`); files a draft from the text | writes to `drafts/itd-N-<slug>.md` (no spec created) |
| The grill step, on one intent id | Socratic adversarial interview that stress-tests an intent for vagueness, missing acceptance, hidden assumptions before planning. Glossary-aware once `terminology/` exists. A brief-section mode would stress-test a brief section instead. (per itd-27, `intents/planned/` — a later phase; no grill sub-verb ships yet) | (stays in current state) |
| Plan (one intent id) | Plans a draft: mints its native spec, injects the bidirectional link (intent `spec_id` ↔ spec `intent`), stamps an identity onto every unmarked scope condition, and moves the file `drafts/` → `planned/`. An impact given at planning stamps the INTENT's product-impact judgement, because the planning interview is where that judgement is made: validated at the create path's bar (never `internal`), written as the bare scalar the create path writes, refused before anything moves when it disagrees with a judgement the record already carries, and a no-op when it agrees; without one the field is left as found and the judgement stays owed to the close (iss-2609170726457256). A production mode given at planning stamps the MINTED SPEC's disclosure pair; the intent's own stamp was written at create time and is never rewritten. On an intent already in `planned/` it does the identity step alone (no spec, no move), takes an impact under the same rules, and refuses when nothing is unmarked and no judgement is added. Single intent ID. | `drafts/` → `planned/` (stamp step: no move) |
| Readiness gate (one intent id, optionally with grounds) | **Implement-readiness gate**: reports whether an intent is ready to implement — seven checks, four of which gate: in `planned/`, with acceptance criteria, a bidirectional spec link, and a written spec body. The two claim rows (mechanism prompted-and-nullable, scope conditions with each condition identified) and the grounds row (a discipline record is exempt: it carries no conjecture of its own) are reported as advisory and never withhold readiness, their refusals parked by iss-2609091009111294 until the rethink of the reading work. Exit 0 ready / 1 not ready / 2 fault. Recording grounds, in the form `<pursued\|deferred\|declined>: <conjecture>`, is the gate's one write: it appends the conjecture behind this decision — what is expected, and what would show it wrong — to the intent's `## Grounds` section, append-only ([adr-57](../../decisions/adrs/0057-grounds-accumulate-as-an-append-only-section.md)), and then reports; a shipped or superseded record is never backfilled. | (no move; recorded grounds append to `## Grounds`) |
| Audit (one intent id) | **Role 1 — single-document fidelity.** Takes a **shipped** intent and nothing else: a record still in `drafts/`, `planned/`, `disciplines/` or `superseded/` is refused by name, because only a shipped intent has a delivered reality to be judged against. Compares the intent's press release + acceptance criteria against delivered reality (code, configs, docs, tests). Per-criterion verdicts (`MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE`) appended to the intent's `## Audit Notes`. Aligns with the spec store's `plan-review` / `impl-review` / `completion-review` vocabulary — same operation shape (adversarial second opinion), different opponent (press release vs engineering spec). spc-12 (predecessor store) ships this **manual** verb; spc-28 (predecessor store) ships the on-close hook (move `planned → shipped` + queue a review), but auto-running the reviewer off that queue is still deferred (no spec currently owns it; spc-6 (predecessor store) disowned auto-firing). | (stays) |
| Audit ingest (a verdict JSON path) | Ingests a host-delegated intent-fidelity verdict JSON, validated fail-closed against the schema and the parked review request, and writes its per-criterion verdict into the shipped intent's `## Audit Notes` (or quarantines a bad payload). | (no move; updates `## Audit Notes`) |
| `/abcd:intent consistency [<itd-N>]` | **Role 2 — cross-document fidelity.** Surfaces five judgement categories (terminology drift, premise contradictions, scope leakage, sequencing impossibilities, naming conflicts) across briefs + intents. **Bare** scans the whole corpus; **with `<itd-N>`** narrows to one intent's relationship with the rest. Findings land in `.abcd/.work.local/logs/audit/consistency-<ts>/report.{json,md}`. The judgement half + on-demand verb are the predecessor's spc-29 (a later phase); mechanical-half categories and pre-commit hook are deferred follow-ups. | (stays) |
| `/abcd:intent shape [<itd-N>]` | **Role 3 — kind classification.** Examines whether an intent's declared `kind` (the noun) still fits the corpus. Surfaces *suggested* reclassifications across three live types: `kind_change`, `bundle`, `supersession`. **Bare** scans the corpus; **with `<itd-N>`** checks one intent. Pairs with the reclassify step (action verb that commits a `shape` finding). On-demand only per spc-29 (predecessor store; a later phase); findings land in `.abcd/.work.local/logs/audit/shape-<ts>/report.{json,md}`. Concurrency via `flock(2)` on `.abcd/coordination/shape.lock` (see § 7). Scheduled / continuous invocation is a deferred follow-up. | (stays) |
| The reclassify step, on one intent id | **A later phase — no reclassify sub-verb ships yet.** Late reclassification (e.g., a standalone intent realised to be a bundle-member; a draft realised to be a discipline; a shipped intent superseded by a later one). Records `reclassification_history` entry; moves the file between directories as the new kind dictates. Reclassifying to superseded, naming the successor handle, is the supersession path: the file moves to `superseded/`, frontmatter records `superseded_by: <handle>` — the record that formally supersedes this intent, either an intent (`itd-M`) or an ADR (`adr-M`) when a decision redecided the question — AND `kind_at_supersession: <original-kind>` so future readers know what shape the intent had when retired. | varies by destination kind |
| Hold (one intent id and a reason) | Holds a draft or planned intent: writes `held: "<reason>"` — the reason is required, single-line and redacted through the store's scanner before the write, and the JSON reports `redacted` like the other write verbs. Planning and closing a spec refuse a held record before anything moves, naming the reason and the unhold that lifts it; `abcd <itd-N>` reports the hold as the next move. Refused on a record already held (naming the standing reason — an updated reason is an unhold then a hold) and on a shipped, superseded or discipline record. The `record_provenance` lint rule reports a `held` value in a shape the verb never writes; a legal hand-typed line is byte-identical to the write and is not reported. | (no move; writes `held`) |
| Unhold (one intent id) | Lifts a hold: removes the `held:` line the hold wrote and reports the reason that stood. Refused on a record not held, on a terminal record, and on a `held` value in a shape the verb never writes (a hand repair record-lint names). | (no move; removes `held`) |
| Link (one intent id and one spec id) | Manual completion of a half-made link: used if the auto-link missed (rare) or for retroactive linking of pre-existing specs. It writes ONE side, the intent's `spec_id`, and refuses unless the spec already declares this intent, so it completes a link from the spec side rather than forging one. A spec that realises a different intent is a mismatch and fails closed. The intent must be in `planned/` | (no move; writes the intent's `spec_id`) |

**No aggregator verb.** A check subverb that runs the audit, consistency and shape passes together is *not* provided — the three primitives have very different runtime costs (the audit is code+oracle expensive; consistency is corpus-wide expensive; shape is cheap on demand). Bundling them produces a slow verb users avoid. Release-readiness is `/abcd:launch`'s pre-flight job. (Note: a scheduled / pre-commit shape leg is a **deferred follow-up**; the predecessor's spc-29 shape surface is on demand only.)

**Bare-command-as-help is a common abcd convention, not a universal one** — many commands in the surface set (the enumeration lives in the [surfaces README](README.md)) show a status board when invoked without args, while others print a plain block instead: `version` renders a version/install/vintage/staleness block, and the `docs`/`history`/`disembark` cobra parents print help or usage with no board. Suggested-next-actions is a design target rather than a shipped guarantee on every bare invocation. It provides discoverability without forcing the user to remember subcommand names.

## 3. Press-release format (standalone + bundle-member kinds)

Standalone and bundle-member intents use this template:

```markdown
---
id: itd-N
slug: <kebab-case>
# NOTE: no `status:` field. Lifecycle state is encoded by directory location only
#   (drafts/ | planned/ | shipped/ | disciplines/ | superseded/) — uniform across all kinds.
#   Per the 2026-05-08 directive: "directory IS the state, no cached mirror."
# The ten keys below are the canonical seed skeleton the quoted-text create writes:
spec_id: null            # or spc-N (set at planning)
kind: null               # set at planning: "standalone" | "bundle-member"
suggested_kind: null     # advisory, written by a capture-time classifier; can be ignored
reclassification_history: []   # appended to by the reclassify step (kind changes only)
builds_on: []            # itd-N ids this intent builds on
severity: minor          # seeded capture-grain severity of the draft
origin: researcher-authored   # arrival path (itd-178), DERIVED from which command ran and carried by no flag:
                              #   researcher-authored | extracted-from-record (an issue's promotion) |
                              #   contributed-by-reading <rdg-N>/<rdi-N> (the reading-ingest verb only).
                              #   Stamped at mint and never rewritten.
production_mode: hand-written # how the text was produced: hand-written | dictated-and-formatted |
                              #   scribe-transcribed. Closed choice carried by a production-mode flag; an absent
                              #   flag takes the repo's declared default from .abcd/config/identity.json.
# Written at mint too, but only by the path that has a value for it:
#   promoted_from: iss-N | rdi-N  — the spc-24 promote back-edge, written by the capture/reading
#                                   promote path and by a later promoted_from stamp on an existing
#                                   intent. Absent on a quoted-text draft, which graduated from
#                                   nothing. The shipped record_provenance rule holds it against
#                                   origin: extracted-from-record / contributed-by-reading
# Added later, not part of the seed skeleton:
#   held: "<reason>"              — the hold the intent verb writes and its unhold removes, on a
#                                   drafts/ or planned/ record only: one non-empty line, redacted before the
#                                   write. Planning and closing a spec refuse while it stands. The record_provenance rule
#                                   reports a value in a shape no verb writes; a legal hand-typed line is
#                                   byte-identical to the verb's and is not reported
#   bundle: <id>                  — for kind: bundle-member, the bundle ID
#   impact: additive|breaking|fix — the compatibility judgement the derived version is computed from. Never "internal" (a press-release-first intent is user-facing by definition), and required before the intent may move to shipped/. Three verbs stamp it: the create at create time, planning at the planning interview (the moment the judgement is made), and the close of its last spec at the move. A verb refuses to overwrite a judgement the record already holds, so a judgement that changed is revised by editing the record — the refusal names the path
#   surface_history: []           — appended when an intent's user-facing surface shape changes (e.g., skill → sub-verb, top-level command → sub-verb, command → flag) WITHOUT changing kind. Distinct from reclassification_history. Schema: { date, from, to, reason }
---

# <Headline — what user-facing capability exists>

## Press Release

> **abcd ships with <capability>.** <2-4 sentences in present tense.>
>
> "<Customer quote>," said <persona> <role>.

## Why This Matters
## What's In Scope
## What's Out of Scope

## Mechanism                # Prompted, nullable (claim-recording-gradient discipline, per adr-51):
                            #   a falsifiable "we expect X because Y", not the outcome restated.
                            #   A blank section passes the readiness gate with the nullity recorded —
                            #   an absent field and a recorded nullity are never collapsed
## Scope Conditions         # Reported at the readiness gate on an advisory row (claim-recording-gradient discipline):
                            #   the population/platform/scale/assumptions the claim holds under, each
                            #   condition carrying a persistent identity that survives edits to its
                            #   text — or the explicit nullity "none stated". An absent section (no
                            #   conditions AND no nullity) exits the gate non-zero, naming the field

## Grounds                  # Required at the readiness gate for a press-release intent (a discipline
                            #   record is exempt). One bullet per gate decision, appended by
                            #   the readiness gate's grounds; the section is append-only (adr-57), so a
                            #   later decision adds a bullet beside the earlier one. Each reads
                            #   `- <pursued|deferred|declined>: <what is expected, and what would
                            #   show it wrong>`

## Acceptance Criteria      # Required (per the itd-1 discipline); Given-When-Then bullets

## Open Questions
## Audit Notes               # populated by the audit (manual Role 1 run)
```

The two sections follow the claim recording gradient — criteria mandatory
(itd-1), mechanism prompted-and-nullable, scope conditions mandatory with an
explicit nullity — enforced by the claim-recording-gradient discipline, the
staged gate [adr-51](../../decisions/adrs/0051-intents-declare-mechanism-and-scope-conditions.md)
anticipated ("its own record on the itd-84/itd-1 pattern"). The distinction
the gate preserves: an absent field is a claim not carried; a recorded
nullity is a claim considered and declined. They are never collapsed.

Discipline-kind intents use a different template — see § 1 "Discipline format" above.

## 4. Persona registry

`.abcd/development/personas.json` is the machine-checked persona roster (SSOT): the `persona_registry` lint rule resolves every press-release quote attribution against the names in that JSON file (its path is set by `.abcd/record-lint.json`). The prose page [`01-product/05-personas.md`](../01-product/05-personas.md) describes the roster and carries the codified abcd principle (no real names, no "hypothetical user"), but defers to `personas.json` as the roster's home — the lint checks the JSON, not the page.

## 5. Frontmatter fields (spc-3 (predecessor store) additions — a later phase)

spc-3 (predecessor store) adds the following optional frontmatter fields to intent files. All are additive — pre-existing intents without them remain valid (schema: `intent.schema.json`, part of the plan-time intent-lint design per [`05-internals/06-lint.md`](../05-internals/06-lint.md) — a later phase). **Not shipped:** the grill sub-verb that writes these fields and the PRD at `.abcd/intents/<itd-N>/prd.md` are a design target (backing intent itd-27, `intents/planned/`); the shipped quoted-text create seeds none of them (see § 1's seed skeleton).

| Field | Type | When set | Purpose |
|-------|------|----------|---------|
| `contexts` | `[list]` | optional; required when a cited term has cross-context collision | Bounded contexts this intent references; used by GL003 to resolve cross-context ambiguity |
| `glossary_terms_used` | `[list]` | auto-populated by grill sub-verb | Qualified `<context>/<term>` IDs cited in the intent body. Machine-readable only — body prose uses canonical display names, not qualified IDs |
| `warrants_assumed` | `[list]` | optional; populated by grill sub-verb | Toulmin warrants surfaced during grill that the author chose to assume rather than make explicit in acceptance criteria |
| `grilled_at` | ISO8601 | set by grill sub-verb | UTC timestamp of Phase 1 grill completion |
| `grill_session_id` | UUID | set by grill sub-verb | UUIDv4 of the Phase 1 grill session that produced the latest grill report |
| `grilled_intent_hash` | SHA-256 | set by grill sub-verb | Hash of the intent at grill time (intent_source_hash recipe). Copied to PRD as `source_intent_hash`. Used at planning to detect intent-edited-after-grill |
| `prd_path` | string or null | set by grill sub-verb Phase 2 | Relative path to the PRD at `.abcd/intents/<itd-N>/prd.md`. Null until grilled |
| `prd_grandfathered` | bool or null | set by one-shot migration | True for pre-spc-3 (predecessor store) planned intents. Suppresses GR002 and GL005 as info-only (not blocker). Cleared when intent is regrilled |

### Term ID semantics — machine vs body prose

**Machine-readable fields** (qualified `<context>/<term>` form REQUIRED):
- `glossary_terms_used` frontmatter field
- PRD frontmatter `glossary_terms_used`
- Grill report `glossary_candidates`
- Lint output and `internal/core/lint` JSON output
- Schema enforcement

**Body prose** (canonical display name only): intent body markdown uses the term's canonical display name (e.g., `persona`, not `core/persona`). Lint extractor for GL005 knows both shapes. Optional explicit citation in body: `[persona](glossary:core/persona)` is supported; lint treats the link target as authoritative when present.

### The grill step as the PRD-producing sub-verb

The grill step, on one intent id, runs two phases over a single session context:

1. **Phase 1 (interactive)**: Socratic adversarial interview. Produces a grill-report at `.abcd/.work.local/logs/grill/<ts>-<itd-N>/grill-report.json`. Writes `grill_session_id`, `grilled_at`, `grilled_intent_hash`, `glossary_terms_used` back to intent frontmatter.
2. **Phase 2 (silent synthesis)**: Consumes the sharpened intent + glossary citations + grill findings. Produces the PRD at `.abcd/intents/<itd-N>/prd.md` with all required frontmatter including `source_intent_hash`, `grill_report_path`, `grill_report_hash`. Sets `prd_path` on the intent.

The PRD is a **frozen contract** artefact (not a session log). It lives at a per-intent path, not under the ephemeral logs tier.

### Planning: the later-phase PRD-validating + freezing front end

The later-phase PRD-freeze front end for planning runs this ordered sequence (the shipped plan step mints the spec, links both sides, and moves drafts → planned — see § 1):

1. Reads `prd_path` from intent frontmatter; refuses if null (no PRD yet).
2. Validates PRD file exists, non-empty, passes section and frontmatter validators.
3. **Provenance verification**: computes `current_intent_hash` (intent_source_hash recipe); verifies it matches PRD's `source_intent_hash`; verifies PRD's `grill_report_hash` against on-disk report. Refuses on any mismatch.
4. **Draft+deprecated term stabilisation**: surfaces any cited term with `status: draft` or `status: deprecated`; user resolves before promotion continues.
5. Computes `frozen_content_hash` (frozen_content_hash recipe; provenance fields INCLUDED to prevent tampering).
6. Writes `frozen_at`, `frozen_content_hash`, `planning_attempt_id` to PRD (atomic).
7. Writes durable attempt journal at `.abcd/intents/<itd-N>/.planning-attempt.json`.
8. Passes intent + frozen PRD to the native plan step as primary context.
9. Writes `## Links` block to the new spec (atomic; idempotent).
10. Writes `spec: spc-N` back to PRD frontmatter.

Both the press-release intent and the frozen PRD are immutable input artefacts post-promotion. The press release is the elevator pitch; the PRD is the AI-consumption contract.

## 6. Acceptance gates and bidirectional link verification

`internal/core/lint` (cross-cutting; its shipped wiring is the docs currency lint and the `cmd/record-lint` gate) is the record-lint over the committed intent tree — it does not run inside planning; the acceptance-criteria refusal at plan time is the intent package's own `hasAcceptanceCriteria` check (`internal/core/intent`). The armed record-lint rules that bear on the intent tree are `intent_lifecycle` (the directory/kind/`spec_id` invariants and the `status:`-key ban below), `intent_impact_valid` (the `impact:` field's legal value set), `persona_registry` (press-release quote attributions resolve to the persona roster), `record_schema` (the `itd` store's filename↔id agreement, and `superseded_by` handle validity with two-way agreement across stores), `record_provenance` (the `origin`/`production_mode` disclosure pair and the `promoted_from` back-edge), `spec_lifecycle` and `spec_id_unique` (the itd↔spc bidirectional agreement below), and `delivery_state` (no CHANGELOG delivery entry, under `Added` or `Changed`, cites an intent still sitting in `drafts/`). The `IL0xx` codes per [`05-internals/06-lint.md`](../05-internals/06-lint.md) are plan-time design, a later phase.

The invariants below are the contract the tree is held to, and each names what holds it. A bullet marked **(convention)** is practice the corpus follows by hand, with no shipped check behind it:

- **Acceptance criteria present and well-formed** (per the itd-1 discipline): an intent cannot be planned without a `## Acceptance Criteria` section carrying at least one Given-When-Then bullet. The block is at plan time, not in the record-lint: the refusal is the intent package's own `hasAcceptanceCriteria` check on a draft, plus the `acceptance_criteria` row of the readiness gate. Everything in `planned/` and `shipped/` has therefore passed it. The two buckets the plan step never crosses are held by hand and are **(convention)**: a draft still on the bench may carry none, and so may a discipline, whose route into `disciplines/` does not run through planning at all. Both are true of this corpus today — four bench drafts and seven of the fourteen disciplines carry no section. No record-lint rule reads it, so a committed intent that lost one still passes the gate.
- **`kind` is set on intents in `planned/`, `shipped/`, `disciplines/`, and `superseded/`.** Intents in `drafts/` may have `kind: null`. The shipped plan step neither infers a kind nor asks for one: it writes `standalone` wherever the draft left the field null, so `standalone` is what an unstated kind becomes. **A later phase** replaces that default with the proposal the user confirms or overrides (§ 1, "Later phase — plan grows a PRD-freeze front end and multi-kind dispatch"). What the record lint holds meanwhile is the value set per bucket: a draft's kind must be null, `standalone` or `bundle-member`, and a planned or shipped record's must be one of the latter two, non-null (`intent_lifecycle`).
- **`kind: bundle-member` requires a `bundle:` field** pointing to a bundle ID; *all* members of a bundle reference the same bundle ID, and bundles are bidirectional in their members' frontmatter. **(convention)** No shipped lint reads `bundle`: `intent_lifecycle` knows `bundle-member` only as a legal `kind` value. **Exception for superseded bundle-members:** intents in `superseded/` with `kind_at_supersession: bundle-member` carry `bundle: null` AND `bundle_at_supersession: <bundle-id>` (preserves the bundle the intent was part of when retired, while signalling the bundle is no longer active). **(convention)** `bundle_at_supersession` appears in no shipped code either.
- **Bundle invariant: all members belong to the same phase.** Planning several intents at once (multi-arg, kind=bundle-member) hard-blocks promotion when the proposed members are scoped to different phases. Lint code `IL011`. Resolution: re-scope into one phase or downgrade one member to `kind: standalone`. See § 1 "Bundle invariant" for the canonical statement and the worked example (`intent-capture-discipline` retirement on 2026-05-07).
- **`surface_history` entries are well-formed.** Every entry must include `date` (ISO YYYY-MM-DD), `from` (free-form surface descriptor), `to`, and `reason` (non-empty). Lint code `IL012` (severity: warn — it's an audit trail, not a gate). See itd-27's `surface_history` (skill → sub-verb on 2026-05-07) for a worked example.
- **`kind: discipline` lives only in `disciplines/` or `superseded/`.** A discipline-kind record in `drafts/` is an error, caught by the record lint over the committed tree rather than at plan time: the `intent_lifecycle` drafts rule admits only a null, `standalone` or `bundle-member` kind, and the disciplines rule demands `discipline`. The gate is the commit, not the promotion.
- **No intent has a `status` field — across any kind.** Lifecycle state is encoded by directory location only (`drafts/` / `planned/` / `shipped/` / `disciplines/` / `superseded/`). The 2026-05-08 directive removed the cached-mirror option: directory IS the state, no exceptions. Lint hard-blocks any frontmatter containing a `status:` key (shipped lint rule: `intent_lifecycle`, severity: blocker; templates and existing files were stripped in the 2026-05-08 sweep). The historical `status: draft | planned | shipped` field on standalone/bundle-member intents has been retired; uniform "directory is canonical" applies to all kinds.
- Every intent in `drafts/` has `spec_id: null` (drafts have no plan yet).
- Every intent in `planned/` has `spec_id: null` (unscheduled) or a `spc-N` id; a non-null `spec_id` points to an existing native-spec-store `<spec_id>-*.md` whose frontmatter `intent` field matches the intent's `id` (or contains the intent's `id` as one of a list, for bundle-member intents).
- **An intent owns one or more specs, and it ships when its last spec closes.** The intent↔spec relation is 1:n (invariant 17 in [`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md), per [adr-2609151513118583](../../decisions/adrs/2609151513118583-an-intent-owns-one-or-more-specs-and-it-ships-when-its-last.md)). The spec's own `intent:` field is the source of truth for the link: the intent's scalar `spec_id` names the spec it was planned with, and the set of specs realising an intent is derived from the back-links (`spec.Store.SpecsForIntent`, `lint.SpecLinkIndex.SpecsForIntent`) — no field carries a list. The bidirectional check is therefore membership, not equality: a spec naming an intent is clean when that intent's `spec_id` names *some* spec realising it (`spec_lifecycle`), so a remainder spec is not drift. Closing a spec ships the intent only when no open spec is left naming it; a remainder slug given on the close mints the follow-on spec in the same operation, and the impact is demanded at the close that ships and refused at any earlier one. The release cut's stale-intent refusal asks whether a planned intent has any OPEN spec, never whether its spec has closed — a planned intent with one closed and one open spec is the correct steady state of a partial delivery.
- **A bundle is the opposite relation and is untouched.** `kind: bundle-member` with a `bundle:` link is N:1 — several intents sharing one spec — and the bundle invariant above (all members in one phase) still holds. 1:n and N:1 are different relations, not two names for one thing; composing them into N:M is not authorised by anything in the record. An intent's own specs may sit in different phases, because the reason a second spec exists is that the work did not fit the cycle that carried the first.
- Every intent in `shipped/` has `kind` set (`standalone` or `bundle-member`) and a non-null `spec_id`. (The stronger invariant — the linked spec exists and is closed, or `spec_id: null` + a `manual_ship_reason` for the no-spec case — is a later-phase gate; the shipped rule checks only that `spec_id` is non-null.)
- Discipline-kind intents have `spec_id: null` always (disciplines never get a spec; this is structurally enforced).
- **Every intent in `superseded/` has both `superseded_by: <handle>` AND `kind_at_supersession: <original-kind>`.** The first names the record that formally supersedes this intent — either a later intent (`itd-M`) or the ADR (`adr-M`) that redecided the question; the second preserves what shape the intent had when it was retired (standalone vs bundle-member vs discipline change the meaning of "superseded"). Both are required, and the two are held differently: `intent_lifecycle` blocks a `superseded/` record whose `superseded_by` is absent or malformed or names an intent no bucket holds, and `record_schema` resolves the handle across stores, while `kind_at_supersession` is **(convention)** — nothing reads it, though every record in `superseded/` carries it. If `kind_at_supersession: bundle-member`, the intent ALSO carries `bundle_at_supersession: <bundle-id>`, preserving the bundle membership at retirement time even though the active `bundle:` field is `null`; that field is convention too.
- No intent ID collisions; no spec referencing a non-existent intent ID.
- File location matches `kind` frontmatter (drift between dir and field flagged).
- For intents promoted from issues (per itd-4): bidirectional `related_issues` ↔ `related_intents` linkage holds. Per spc-23 (predecessor store; intent-auditor's issue-drift role — a later phase).

Drift triggers a warning, not a block (since spec-store state may legitimately lag intent state during work in progress). A kind/directory mismatch is a hard block in `intent_lifecycle`, because the kind/directory contract is what makes the lifecycle navigable. Acceptance-criteria absence blocks too, at plan time and at the readiness gate rather than in the record-lint: the whole point of the itd-1 discipline is to force the AC discipline before an intent is committed to build.

## 7. The `intent-auditor` agent (three roles, three verbs)

`intent-auditor` is a single agent in the catalog (per `05-internals/01-agents.md`) that owns three roles. Roles share the agent's prompt scaffolding, oracle backend resolution, and receipts; they differ in what they review, when they run, where findings land, and **which subverb users invoke them through**. Each role has its own dedicated verb — no role-by-kind dispatch, no hidden-state forking.

The audit verb names Role 1, per adr-40: it emits family-2 promise-vs-reality verdicts, and the four-bucket vocabulary reserves *review* for family-1 change-judgement (`plan-review`, `impl-review` stay family-1 nouns). The top-level conformance check is `abcd lint`, with `/abcd:audit` reserved for itd-16's hash-chain fidelity checks. Each verb means one thing.

### Role 1 — single-document fidelity → the audit

The audit is the **manual** Role 1 surface. The bucket is what
it gates on, not the kind: only an intent in `shipped/` is accepted, and any
other is refused naming the bucket it is in. That reaches the same records the
kind framing describes — `shipped/` holds `standalone` and `bundle-member`
intents and nothing else — but it also means a standalone intent still awaiting
its build is turned away, as it should be. Compares:
- **Intent press release + acceptance criteria** ("what user-facing capability exists, plus the verifiable bar")
- **Delivered reality** (current state of the source repo — code, configs, docs, tests)

This is product-tier review. The opponent is the codebase. Distinct from the spec store's `completion-review` (engineering-tier — code vs spec) — both can pass or fail independently, and disagreement between them is signal: the spec may have mistranslated the press release.

**Two passes, two destinations.** Role 1 judges two document kinds and writes each verdict to its own destination:
- the **itd-1 acceptance pass** (a shipped intent) writes per-criterion verdicts into that intent's `## Audit Notes` (the verdict of record), with the review request staged under `.abcd/.work.local/reviews/`;
- the **itd-37 `MG004` pass** (a native spec's `## Modification Grammar`) writes its `PASS` / `FAIL` verdict to an `audit/spec-mg-<ts>/` receipt under the local ephemeral logs tier — native specs have no `## Audit Notes` section, so the verdict cannot land in-file (a later phase).

**What the predecessor's spc-12 ships.** The predecessor's spc-12 ships the **discipline-judgement subset** of the audit — the itd-1 per-criterion acceptance verdicts, with their writer and receipt. The itd-37 `MG004` boilerplate check is a later phase, as the two-passes note above records: no `MG004` check, writer or receipt exists in this tree. The broader **press-release prose review** (the `honoured` / `diverged` / `missing` buckets below) and other prose/terminology/PRD-fidelity outputs are **deferred** to a later spec. It also ships the **manual** review surface; spc-28 (predecessor store) ships the on-close hook (move `planned → shipped` + queue a review on that transition). Auto-running the reviewer off that queue is still deferred — no spec currently owns it; spc-6 (predecessor store) disowned auto-firing.

**The spc-23 (predecessor store) issue-drift role (a later phase).** The predecessor's spc-23 adds an issue-drift role — a corpus-wide bidirectional cross-reference walk between shipped intents and the `iss-N` ledger (per itd-4), with receipts under `.abcd/.work.local/logs/audit/issue-drift-<ts>/`, a default exit 0 with warnings to stderr, and a strict exit-1 mode for CI gates. See the predecessor's `spc-23-intent-auditor-extension`.

Outputs findings with two layers:

**Per-criterion verdicts** (per the itd-1 discipline) — for every Given-When-Then bullet in the acceptance section:
- `[MET]` — verified
- `[MET_WITH_CONCERNS]` — partially observed
- `[NOT_MET]` — divergence (with explicit "what was delivered vs what was promised")
- `[INCONCLUSIVE]` — could not verify

**Press-release prose review** (three buckets — `honoured` / `diverged` / `missing`):
- **honoured** — capabilities the press release promised that exist as described
- **diverged** — capabilities present but materially different from the press release
- **missing** — capabilities the press release promised that aren't observable in delivered reality

Findings appended to the intent's `## Audit Notes` section. Manual re-run of the audit available at any time. **Overall verdict rollup:** any `NOT_MET` → overall `NOT_MET`; any `INCONCLUSIVE` without `NOT_MET` → overall `INCONCLUSIVE`; any `MET_WITH_CONCERNS` without `NOT_MET`/`INCONCLUSIVE` → overall `MET_WITH_CONCERNS`; else `MET`. (The verdict tags and the verb now share the audit register, per adr-40.)

For bundle-member intents, this role runs *per intent* against the same delivered reality (each member's acceptance criteria checked separately).

#### The audit loop — record-only vs loop-to-acceptance (itd-50 / spc-52, predecessor store)

Role 1 records per-criterion verdicts; **itd-50 adds the POLICY that decides what happens to a recorded verdict.** The policy rides the review-queue drainer on the run seam (adr-27) — it is NEVER in the pure on-close lifecycle hook (the lifecycle close stays a pure data function; the mode logic lives in the drainer/policy layer).

**Three audit-loop modes, facilitator-elected per intent** via the `audit_mode` frontmatter key:

- **`record-only`** (the default, and today's behaviour) — a `NOT_MET` is written to `## Audit Notes`; no re-work is triggered. An **absent** `audit_mode` key resolves to `record-only` (additive, spc-28/spc-43-compatible — predecessor store).
- **`loop-to-acceptance`** — a `NOT_MET` re-opens the linked work and iterates against the same acceptance criteria until they read `MET`, bounded by `audit_budget` (the spec-grain SHIP/NEEDS_WORK fix-loop lifted to the intent grain). See `05-internals/03-configuration.md` for the `audit_mode` / `audit_budget` keys, the default budget (`3`), and the fail-closed rule for a malformed/zero/negative budget.

**Full state-coverage table** (every Family-2 rollup maps to a defined action — no dead-ends; the loop trigger is `NOT_MET` only):

| Reviewer rollup | Loop action (`loop-to-acceptance`) |
|---|---|
| all `MET` | Succeed → manual-verification gate. `audit_outcome=MET`. |
| `MET_WITH_CONCERNS` (no `NOT_MET`) | Proceed to the gate (concerns are advisory; the product thinker sees them). Does NOT consume budget. `audit_outcome=MET`. |
| `NOT_MET` (budget remaining) | Re-open linked work (re-enqueue at the queue layer), `audit_iterations_used += 1`. |
| `NOT_MET` (budget exhausted) | Terminate `UNACHIEVABLE` (budget-exhausted) → replan invitation. |
| `NOT_MET` + reviewer judges criteria unmeetable | Terminate `UNACHIEVABLE` (reviewer-impossible) **early**, before budget exhaustion → replan invitation. |
| `INCONCLUSIVE` | Fail-closed: recorded as today, no iterate, no summons, no replan; never flips to `UNACHIEVABLE`. |

`UNACHIEVABLE` is an intent-level **rollup** terminal the policy layer writes — it is **NOT** a per-criterion verdict (`ACCEPTANCE_VERDICTS = {MET, MET_WITH_CONCERNS, NOT_MET, INCONCLUSIVE}` is unchanged) and is recognised at the `Overall:` / rollup parse layer only.

**The `UNACHIEVABLE` replan surface (no rollback).** A terminal `UNACHIEVABLE` writes a `why-unachievable` explanation + a **replan invitation** block (naming both the product thinker and the facilitator) into the intent's `## Audit Notes`. The intent **stays in `shipped/`** — its `spec_id` / `kind` / directory + delivered artifacts are byte-untouched (a `drafts/` move would break the lifecycle invariant + spc-48 (predecessor store) lint and read as a partial un-ship). The invitation seeds the grill step; no machine authors a replan, nothing is auto-rolled-back.

**Gated manual verification + verification receipt (R5).** The manual-verification invitation renders **only** when the machine rollup is acceptance-eligible (all `MET`, or `MET_WITH_CONCERNS` with no `NOT_MET`) — if any criterion is not `MET`/concerns, the loop or replan invitation runs first; the product thinker is never asked to hand-test something the audit already knows is broken. The sign-off is recorded as a **verification receipt distinct from the machine verdict of record** — a separate JSON artifact under `.abcd/.work.local/logs/audit/verify-<ts>/receipt.json`, never merged into `## Audit Notes`:

```json
{ "intent_id": "itd-N", "machine_rollup": "MET", "state": "offered",
  "justification": "...optional...", "recorded_by_role": "product thinker",
  "ts": "20260614T…Z" }
```

Receipt **states**: `offered` (the gate opened — the drainer stamps this on a `MET` loop outcome), `accepted` (the product thinker confirms the intention is delivered), `rejected_wrong_criteria` (every criterion passes but the *why* is not delivered).

**`rejected_wrong_criteria` → replan, NOT a synthetic `NOT_MET`.** When the machine says `MET` but the product thinker judges the criteria themselves were wrong, the defect is the *criteria*, not the code — so the rejection routes to the **same replan surface** as `UNACHIEVABLE` (one writer, two entry points), carrying the rejection justification into the seeded grill. It does **not** write a `NOT_MET` (which would re-loop the implementation against criteria that already pass) and does **not** move the intent.

### Role 2 — cross-document fidelity → `/abcd:intent consistency [<itd-N>]`

Introduced by itd-48 (which superseded itd-31). The opponent is *other documents*: compares the brief and every intent against each other (and against the brief itself), surfacing the five live judgement categories — **terminology drift, premise contradictions, scope leakage, sequencing impossibilities, naming conflicts**. No spec-store analogue — the spec store reviews one artefact at a time; corpus-wide consistency is pure abcd ground.

The judgement half runs on demand via `/abcd:intent consistency` (Carmack-level oracle review) — a later phase (spc-29, predecessor store), not yet a binary sub-verb.

**Deferred follow-up**: the mechanical-half lint categories — schema/state contradictions, reference rot, acknowledgement gaps — were originally planned as `internal/core/lint` cross-doc codes `XD002`/`XD006`/`XD007` per `05-internals/06-lint.md`; the lint-code half is deferred to a follow-up intent. Pre-commit hook wiring that would let `/abcd:intent consistency` findings block commits is also deferred.

**Polymorphic on arg presence (same operation, narrowed scope):** bare = scan the whole corpus; with `<itd-N>` = scan one intent's relationship with the rest. This is *not* the forbidden hidden-state dispatch — the operation is identical; the arg just narrows scope (like `git log` vs `git log <path>`).

Findings land in `.abcd/.work.local/logs/audit/consistency-<ts>/report.{json,md}`.

### Role 3 — kind classification → `/abcd:intent shape [<itd-N>]`

Introduced alongside the three intent kinds (per itd-34). The opponent is the *kind taxonomy*: examines whether each intent's declared `kind` (the noun in frontmatter) still fits the corpus. The verb `shape` matches the taxonomy noun and pairs cleanly with the reclassify step (the action verb that commits a `shape` finding).

The on-demand surface is a later phase (spc-29, predecessor store). **Bare** scans the corpus; **with `<itd-N>`** checks one intent. Findings land in a report under `.abcd/.work.local/logs/audit/shape-<ts>/report.{json,md}`. The user accepts a suggestion via the reclassify step; declined suggestions are logged for future review (so the reviewer doesn't re-surface the same suggestion every run).

**Concurrency contract** (between any future scheduled invocation and on-demand `shape`):

```
.abcd/coordination/shape.lock  (file lock via flock(2))
```

- On-demand `/abcd:intent shape` acquires **blocking with 60s timeout**; on timeout, reports "background run in progress; try again or wait for it".
- The intent's `## Audit Notes` section is updated atomically (read full file → modify in memory → write via `.tmp` + `rename(2)`) by the on-demand path.

Mirrors the file-claim pattern itd-33 will introduce in a later phase but is far simpler — single lock per audit-log subdirectory, no agent identity, no heartbeat.

The three live `suggestion_type` values this role produces:

- **`kind_change`** — a 1:1 reclassification between `standalone` and `discipline`. Example: "intent X has no user moment in its press release (the customer quote describes a process, not a feature); consider `kind: discipline`."
- **`bundle`** — "intents X and Y reference each other in scope/references and target the same release; consider `kind: bundle-member` with shared bundle ID."
- **`supersession`** — "intent X's scope is fully covered by intent Y; consider reclassifying it as superseded by Y."

> **Deferred follow-up**: pre-commit hook wiring for continuous shape scanning, and the `shape(...)` function's `mode="pre_commit"` parameter is preserved as a seam but no hook invokes it. Discipline subtype clustering ("once enough disciplines exist, surfaces 'three disciplines have similar `kind_notes`; consider formalising a subtype'") was named in earlier itd-34 drafts and is *not* shipped by spc-29 (predecessor store) — it is not a live suggestion type.

### Review and audit trail layout

The shipped audit keeps its record in the intent file itself: its ingest of a verdict JSON writes the per-criterion verdict into the shipped intent's `## Audit Notes` section (the verdict of record), and the emit path stages an ephemeral review request under `.abcd/.work.local/reviews/` (gitignored, report-only). Idempotency and review state live in that one committed place — directory/file-as-truth, no side database.

The later-phase review/audit verbs write their per-run receipts under the local ephemeral logs tier, `.abcd/.work.local/logs/audit/<sub-tier>-<ts>/`, where the `audit/` name reflects "this is the on-disk audit trail" regardless of which verb produced it and the sub-tier prefix names the verb:

- The audit (MG004 pass) → `audit/spec-mg-<ts>/` (Role 1 itd-37 `MG004` check on a native spec's `## Modification Grammar`; one per-run batch receipt, one `results[]` entry per spec — native specs have no `## Audit Notes` section, so the verdict lands here, per itd-37 — a later phase)
- `/abcd:intent consistency` → `audit/consistency-<ts>/` (Role 2, cross-document fidelity per itd-48, which superseded itd-31)
- `/abcd:intent shape` → `audit/shape-<ts>/` (Role 3, shape classification per itd-34)
- `/abcd:audit chain` → `audit/chain-<ts>/` (conversation/edit-history Merkle, default application per itd-16 — a later phase)
- `/abcd:audit lifeboat <path>` → `audit/lifeboat-<ts>/` (lifeboat-artefact integrity per itd-35 — a later phase)

`chain` and `lifeboat` are later-phase sub-verbs of the reserved `/abcd:audit` (their backing intents itd-16 and itd-35 sit in `intents/drafts/`); the read-only working-conventions conformance check is `abcd lint`. The audit is a shipped sub-verb of `/abcd:intent`;
 `consistency` and `shape` are later phases. Bare `/abcd:intent` is status+help per the common (not universal) bare-command-as-help convention.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd intent`

Sub-verbs: `abcd intent audit`, `abcd intent hold`, `abcd intent link`, `abcd intent new`, `abcd intent plan`, `abcd intent ready`, `abcd intent unhold`.

| Flag | Type |
|---|---|
| `--impact` | string |
| `--production-mode` | string |
| `--title` | string |

### `abcd intent audit`

Sub-verbs: `abcd intent audit ingest`.

Flags: none.

### `abcd intent audit ingest`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--verdict-json` | string |

### `abcd intent hold`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--reason` | string |

### `abcd intent link`

Sub-verbs: none.

Flags: none.

### `abcd intent new`

Sub-verbs: none.

Flags: none.

### `abcd intent plan`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--impact` | string |
| `--production-mode` | string |

### `abcd intent ready`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--grounds` | string |

### `abcd intent unhold`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
