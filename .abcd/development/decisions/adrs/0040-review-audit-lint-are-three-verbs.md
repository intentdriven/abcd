---
id: adr-40
slug: review-audit-lint-are-three-verbs
status: accepted
date: 2026-08-16
supersedes: null
superseded_by: null
related_intents: [itd-16, itd-48, itd-53, itd-83, itd-85, itd-94, itd-97, itd-99, itd-109]
related_rfcs: []
related_adrs: [adr-25, adr-9, adr-5]
---

# ADR-40: Review, audit, lint, and gate are four buckets, separated by what each compares

## Context

abcd performs many assessment-shaped acts, spread across its three roles. A
product thinker wants to know whether what shipped is what they asked for. A
facilitator wants to know whether a record is ready to move, and fires the checks
that answer it. The implementation team wants to know whether a change is any
good. Today these all reach for the word *review*, and the surface offers no way
to tell from a verb name whose question it answers.

**The distinction is already recorded.**
[`05-internals/01-agents.md`](../../brief/05-internals/01-agents.md)
§ Verdict-tag protocol declares two verdict families and rules them disjoint:

> **1. Review verdicts** … `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` …
> **2. Per-criterion intent acceptance verdicts** … `{MET, MET_WITH_CONCERNS, NOT_MET, INCONCLUSIVE}` …
> Different category from review verdicts — review verdicts assess a *change*,
> criterion verdicts assess a *promise vs reality*. These two enums are
> deliberately disjoint and never mixed. **Reviews emit family 1; auditors emit
> family 2.**

The rule is sound. Three surfaces do not follow it:

| Surface | Emits | Named | Should be |
|---|---|---|---|
| `abcd intent review` (`internal/core/intent/review.go`) | family 2 — `MET` / `NOT_MET` / … | review | **audit** |
| `abcd audit` (itd-85) | rule ids + severities; no model | audit | **lint** |
| `disembark oracle` | family 1 — `SHIP` / `NEEDS_WORK` / … | "the oracle audit" in prose | **review** |

The first two crossed with each other. What
[`01-product/03-mental-model.md`](../../brief/01-product/03-mental-model.md)
itself calls *"the intent audit"* — one of *"three audit grains of one shape:
brief audit, phase audit, intent audit"* — is the one surface named `review`.
Meanwhile [`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md)
**reserves** `/abcd:audit` for *"formal verification surface — hash-chain /
Merkle audit trails, fidelity checks"* (itd-16), and itd-85 took that reserved
name for deterministic repo-conformance checking.

**What is *not* wrong.** The brief is deliberately aspirational — it describes
the project as though built, per the working-backwards discipline, while
[adr-5](0005-brief-is-current-state.md) keeps it current rather than versioned.
The mechanism reconciling those two things already exists and works:
[`04-surfaces/README.md`](../../brief/04-surfaces/README.md) carries a
machine-checked `Status` column (`shipped` / `staged`) enforced by the
`surface_coverage` record-lint rule, and the brief carries some one hundred
inline *design target, not yet shipped* markers. This ADR does not replace that
discipline; it extends it to the granularity where it is currently blind.

**Where it is blind: sub-verbs.** `surface_coverage` checks that a `shipped` row
has a backing `commands/<name>.md`. It cannot see inside a row. So six of twenty
rows read `shipped` and then qualify themselves in prose — `/abcd:embark`
("the full embark chapter … remains a design target"), `/abcd:intent` ("refine /
grill / ship / consistency / shape / reclassify … remain design targets"),
`/abcd:capture` ("`promote` is a design target"), and three more. `shipped` at
surface grain means *the top-level verb exists*, and every unbuilt sub-verb hides
inside a row that says otherwise. Documents outside the registry then cite those
sub-verbs as live — `02-constraints/04-naming.md` calls `capture promote` "live as
of spc-30/itd-46"; `05-internals/01-agents.md` says "spc-29 ships" `intent
consistency` — because nothing checks them. Both cite **predecessor-store** spec
ids: the capability shipped in an older spec store and the prose survived the
migration to the native one.

## Decision

**Four buckets, separated by _what is compared to what_.** The comparison — not
the implementation, and not the trigger — is what identifies whose question a
surface answers, so the comparison is what names it.

| Bucket | Compares | Question | Verdict |
|---|---|---|---|
| **lint** | an artefact against *a rule about its form* | "Is this well-formed?" | rule id + severity + exit code |
| **review** | a *change* against *judgement* | "Is this any good?" | `SHIP` / `NEEDS_WORK` / `MAJOR_RETHINK` |
| **audit** | *reality* against *a recorded commitment* | "Did we do what we said?" | `MET` / `MET_WITH_CONCERNS` / `NOT_MET` / `INCONCLUSIVE` |
| **gate** | — *consumes* findings and decides one action | "May I proceed?" | pass/fail + remedy |

### 1. One surface performs exactly one act

A surface that would perform two is split at design time, not documented as a
bundle. This is what makes the bucket a *guarantee* rather than a statement of
intent: run an `audit` and every finding is promise-versus-reality, by
construction.

The cost is borne almost entirely by surfaces that do not yet exist. Every
assessment surface abcd has actually built is already single-act — three are
merely misnamed. The two genuinely multi-act designs, `/abcd:intent consistency`
(five finding categories spanning at least two buckets) and itd-109's
verification suite, are both unbuilt, so they are *specified* correctly rather
than refactored.

The alternative — classify acts, and let a surface be named for what it
predominantly does — was rejected because it breaks the role guarantee below: a
product thinker running an `audit` would receive naming-register collisions and
glossary gaps mixed in among their own findings.

### 2. The buckets are a closed list, extended by PR

Membership is enumerated in
[`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md) under
*Reserved vocabulary*, the same mechanism `task_classes`, `kind`, and `capture
verdict` already use. A surface producing findings is `lint`, `review`, or
`audit`. A surface consuming findings to decide one action is a `gate`.

This is enumeration rather than criterion **because an autonomous run given a
criterion will interpret it, and a table lookup either matches or fails the
gate**. The judgement moves from implementation time to review time, where a
human is present.

`abcd intent ready` is why the list needs a fourth bucket. Its four checks
(`bucket`, acceptance criteria, spec link, spec body) are lint-shaped, but itd-94
shipped `ready` deliberately with an exit-code contract autonomous runs depend on
as their step-0 skip filter. It is not a synonym competing with lint/review/audit
— it names a *decision*, not a comparison. Gates are named for the decision they
guard.

### 3. Determinism is orthogonal and never names a bucket

Whether a model is involved is an implementation property. itd-16's hash-chain
verification compares reality against a recorded commitment and needs no model —
it is an **audit**. itd-85's conformance check applies rules about
well-formedness — it is a **lint** however built. Sorting by determinism is what
produced the current crossing.

A lint's exit code being consumed by CI does not make it a gate; producing
findings is what names it.

### 4. Automatic versus manual never enters the name

`intent review` already shows the correct shape: its emit fires automatically
from `spec close`, its ingest is a deliberate act, one verb. Per
[itd-97](../../intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md),
deciding *when* a check fires is the facilitator's work. Encoding the trigger
would double the namespace for no semantic gain and push a facilitator concern
into a name the other two roles must read.

### 5. The oracle is a seam beneath the buckets, never a bucket

Per [adr-25](0025-host-delegated-llm-default.md), `review` and `audit` reach a
model through the host-delegated oracle seam; `lint` never does. The seam is
transport, not vocabulary. `disembark oracle` is correctly named for the seam it
invokes; what needs correcting is the prose calling its family-1 output an
"audit".

> **Amendment (2026-08-16).** The keep-the-verb ruling above rested on a claim
> the planning investigation disproved: the binary verb never invokes the
> oracle seam. `disembark oracle` is a compute-or-ingest verdict endpoint —
> its deterministic mode is a mechanical mapping over the manifest seal and
> coverage summary, and its delegated mode validates a verdict the host's
> agent produced elsewhere; the model access happens host-side, outside this
> verb. Naming the verb for the seam claims a seam it does not touch and
> collides with the `/abcd:oracle ask` design target, which is the seam's
> real surface. The maintainer therefore reversed this clause: the verb
> renames to `disembark review` (itd-125), and the seam itself — adr-25, and
> `oracle` as the seam's name — stands unchanged. The rest of this section
> (the seam is transport, not vocabulary; lint never reaches a model) is
> unaffected.

### 6. Status and bucket are recorded together, per sub-verb

Each surface file under `04-surfaces/` carries a sub-verb table with two facts
per verb — **what it compares** (the bucket, where the verb is an assessment) and
**whether it exists** (`shipped` / `staged`). `surface_coverage` extends to check
each row against registered cobra sub-commands, in both directions: a sub-verb
that ships without a row fails, and a row claiming a sub-verb that is not
registered fails.

One table serves both disciplines. It is the detector this ADR needs, and it is
the fix for `shipped` meaning six different things — a `staged` row for `capture
promote` cannot then be called "live" elsewhere without failing the build.

### The buckets map onto the three roles

| Role | Their question | Their bucket |
|---|---|---|
| **Product thinker** | "Did I get what I asked for?" | **audit**, and essentially nothing else |
| **Facilitator** | "Is this ready to move, and what fires when?" | all four, plus ownership of every trigger |
| **AI implementation team** | "Is this change any good?" | **review** and **lint** |

This mapping is the operational reason the split is worth its cost.
[itd-99](../../intents/drafts/itd-99-a-team-of-product-thinkers-decides-as.md)
commits abcd to never asking a product thinker to make a technical decision —
only to decide on a clear proposal. A product thinker handed a lint failure has
been handed a technical decision. The four-bucket split with one act per surface
makes that promise **mechanically checkable**: point at the surface, and it names
whose question it is.

### Implied renames, and how they ship

- **`abcd intent review` → `abcd intent audit`.** It emits family 2; the brief
  already calls it the intent audit. The `intent-fidelity-reviewer` agent and the
  `intent_review` task-class token move with it.
- **`abcd audit` (itd-85) → a lint-shaped name**, returning `/abcd:audit` to
  itd-16 as `02-constraints/04-naming.md` reserved it. The replacement name is an
  open question for the planning interview.
- **`disembark oracle`** keeps its verb; the prose describing its output as an
  "audit" is corrected to "review".

**Clean break, no aliases.** abcd is pre-1.0.0, so a declared-breaking change is
the sanctioned path — `--impact breaking` is already in live use (iss-171) and
drives version derivation automatically. An alias would additionally collide with
the documentation discipline, which forbids change-narration: the old name could
not be documented in present tense, leaving an undocumented silent forward, which
serves a user worse than a clean break plus a CHANGELOG line. Users re-download.

Each rename is a separate intent carrying its own acceptance criteria. This ADR
fixes the vocabulary; it does not authorise a breaking change.

## Alternatives Considered

**One verb (`review`) polymorphic on its target.** `review itd-85`, `review
spc-9`, `review <diff>`. Rejected: it merges two verdict families the record
rules disjoint, and erases the role signal — the one thing the product thinker
most needs from a name.

**Classify acts rather than surfaces.** Rejected under decision 1: it makes the
bucket a statement of intent rather than a guarantee, and breaks itd-99.

**Three buckets, no `gate`.** Rejected: it forces `intent ready` to become
`intent lint`, discarding a shipped name with a documented exit-code contract.

**A criterion instead of a closed list** ("no synonym may name an assessment
act"). Rejected: it requires a judgement call at exactly the moment no human is
present.

**Four or more assessment buckets, adding `verify` / `validate`.** Rejected: no
candidate names a comparison the four do not cover. itd-109's verification suite
is a *packaging* of lints and audits, not a fifth kind.

**Leave the vocabulary and document the exceptions.** Rejected: the exceptions
are the two most-used assessment surfaces, so the documentation would be read
more often than the rule.

## Consequences

- Every new assessment surface faces one naming test: *what is compared to
  what?* itd-109 is the first to face it and should be planned only after this
  ADR settles.
- Three shipped surfaces are known-misnamed until their rename intents land. The
  vocabulary is correct from this ADR; the surface trails it, and the gap is
  visible rather than silent.
- `/abcd:audit` is re-reserved for itd-16. Any surface wanting the word must
  compare reality against a recorded commitment.
- The sub-verb tables are real work: twenty surface files, and some bucketings
  will be arguable (`identity` proposes corrections, `changelog` carries a
  guardrail). That population pass lands in the same change as the extended
  `surface_coverage`, so nothing ships unbucketed.
- Once armed, every new verb costs a table row. That friction is intended.
- The `task_classes` closed enum carries `oracle_review`, `intent_review`,
  `audit`, `lint`, and `cross_document_audit` — five tokens spanning three
  buckets. They are re-checked against this taxonomy when the rename intents
  land; the enum is PR-to-extend and no shipped check reads it.
- The product thinker's surface narrows to one bucket, which is the point.
