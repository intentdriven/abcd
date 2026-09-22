---
id: adr-9
slug: phase-as-product-layer
status: superseded
date: 2026-05-16
supersedes: null
superseded_by: adr-2609212115255771
related_intents: []
related_rfcs: []
related_adrs: [adr-1, adr-5, adr-10]
---

# ADR-9: Phase as a product-reflection layer between brief and intent

> **Terminology note.** The *how* layer is named the **spec**. This ADR's prose
> was updated by the spec-terminology-rename ADR
> ([adr-11](0011-spec-terminology-rename.md)).

## Context

abcd organised development work in three layers (adr-1): brief (*what is the
project*), intent (*why does this user-facing change matter*), spec (*how do we
build it*). Sequencing — *which intents and plumbing phases ship together, in
what order* — had no home of its own. It was carried two ways, both unsatisfying:

1. **Plugin versions** (`v1`, `v2`, `v3`). The roadmap's "Versioning Convention"
   section, the `roadmap/milestones/` directory, and a `target_release: vN`
   field on every intent all encoded a version axis. But adr-5 had already
   established that the brief is *current state*, not a versioned artefact —
   and a version axis on intents fought that. "v2" is a label with no content;
   it answers neither "what order" nor "what do we expect to be true."
2. **The dependency DAG** in `brief/06-delivery/01-build-sequence.md`. This
   gives dependency edges but not bundling — it cannot say "these five intents
   form a coherent stretch of work that ends with ahoy working end-to-end."

Neither carrier captured the thing a product thinker actually wants at this
grain: a **re-statement, in user terms, of what a stretch of work is expected
to make true** — coarser than a single intent's press release, finer than the
whole-project brief. The brief's own mental model gestured at a "phase audit"
(reality vs. a phase's acceptance) but never gave it a target to review
against, because no phase artefact existed.

## Decision

abcd adds a fourth layer: the **phase**. A phase is an ordered stretch of work
that ends in a **milestone** (a concrete, checkable end condition). Phases
replace plugin-version language as the project's sequencing axis.

- Phases live in `.abcd/development/roadmap/phases/` — one `phase-N-<slug>.md`
  per phase, plus a `README.md` index. The `roadmap/milestones/` directory is
  removed; a milestone is now a section *inside* a phase doc, not a separate
  file.
- Each phase doc is **Expectation-led**. Its sections are: `## Expectation`
  (working-backwards: what is true for the user at phase end — press-release
  voice, at phase granularity), `## Milestone` (the concrete checkable cut),
  `## Scope` (which intents and which brief plumbing-phases this phase bundles),
  `## Maps against` (traceability — which brief sections this phase realises),
  `## Dependency rationale`, `## Open questions`.
- **Specs gain a `phase:` frontmatter anchor.** A spec already traces to an
  intent or a brief plumbing-phase; it now also names the phase it belongs to.
  This makes the phase audit real: when every spec in a phase closes, delivered
  reality can be reviewed against the phase's `## Expectation` — the same
  Given-When-Then fidelity shape `intent-fidelity-reviewer` applies one grain
  down.
- **The `target_release` field is removed from intents entirely.** An intent
  carries no release or sequencing field. The intent → phase mapping is
  *editorial* (a sequencing decision), and per the project's own principle —
  per-item mapping lives in links, coarse-grain mapping is editorial — it lives
  in the phase doc's `## Scope`, not in intent frontmatter. `target_release`
  was unanswerable at capture time anyway (a release number is an *output* of
  deciding what each phase contains, not an input a product thinker can stamp
  on a fresh draft — "they want everything"). An intent not yet listed in any
  phase doc is implicitly unscheduled.
- **The bundle invariant is rephrased in phase terms.** It was "all bundle
  members share `target_release`"; it becomes "all bundle members belong to the
  same phase" (a bundle ships as one spec; one spec belongs to one phase).
  `IL011` (the multi-arg-plan guard) becomes a plan-time check rather than a
  frontmatter-field lint, since phase membership lives in phase docs.

The audit hierarchy is now three grains of one shape: brief audit (reality vs.
brief-phase acceptance), **phase audit (reality vs. phase Expectation)**, intent
audit (reality vs. intent acceptance).

## Alternatives Considered

1. **Keep plugin versions; treat phases as informal grouping.** Rejected: keeps
   the `v1/v2` label fighting adr-5, keeps `target_release` as a field a product
   thinker cannot honestly fill in at capture time, and never gives the phase
   audit a target. The version axis is a label, not an artefact — it cannot be
   reviewed against.
2. **Phase as a frontmatter field on intents (`phase: phase-2-ahoy`), no phase
   docs.** Rejected: a bare field records membership but not *expectation*. The
   load-bearing value of a phase is the product thinker's re-statement of what
   the phase should make true — that needs prose, which needs a document.
3. **One flat `phases.md` file instead of a directory.** Rejected: a phase
   carries real per-phase content (Expectation, Milestone, Maps-against,
   rationale). The brief itself adopted numbered folders over a single file for
   exactly this reason (`brief/00-meta.md`); phases follow the same pattern as
   `intents/` and the brief sections.
4. **Migrate `target_release` → `phase` on all intent files.** Rejected:
   replacing one frontmatter sequencing field with another keeps the
   anti-pattern — it still asks the intent file to record its own sequencing,
   which then drifts from the phase docs. The editorial intent→phase mapping in
   phase docs is the single source; intents carry no sequencing field at all.
   (`target_release` was removed outright in this same change — schema,
   `internal/core/lint`/`_prd_writer.py` hash recipes, fixtures, and all 40 intent
   files — rather than renamed.)

## Consequences

**Gains:**
- The roadmap stops fighting adr-5: no version labels on a current-state
  project. Sequencing has a real artefact instead of a label.
- The product thinker gets a genuine reflection point per phase — a place to
  re-map expectation against brief and intents partway through the work.
- The "phase audit" the mental model already named becomes buildable: the
  phase `## Expectation` is what it reviews against.
- Specs gain a coarse anchor (`phase:`), enabling roll-up status ("Phase 2: 3/5
  specs done") and phase-grain fidelity review.

**Costs / obligations:**
- A fourth layer is one more place to look — discoverability cost. Mitigated:
  the layer is thin (a handful of phase docs) and the `roadmap/README.md`
  dashboard renders phase status in one view.
- `brief/01-product/03-mental-model.md` becomes a four-layer model; its diagram
  and prose are updated.
- The `phase:` spec anchor needs eventual lint coverage (verify the named
  phase exists) — deferred until the phase-audit reviewer is built. **Specified
  (spc-66, predecessor store):** the `PA001` verify-exists anchor lint and the
  phase-audit reviewer are both specified there; neither is in the Go binary
  yet. The *corpus anchor backfill* onto the unanchored specs stays deferred
  (see the amendment's delivery note).
- Removing `target_release` changes every intent's `intent_source_hash` (the
  field was in the hash recipe). Harmless here — no intent in the corpus is
  grilled yet, so no stored `grilled_intent_hash` is invalidated — but the two
  hash allow-lists (`internal/core/lint`, `_prd_writer.py`) and `prd.schema.json`'s
  recipe documentation were updated together to keep the "one canonical recipe"
  invariant intact.
- Phase IDs must be stable once specs anchor to them; renaming a phase means
  re-anchoring its specs.

**Downstream decisions enabled:**
- A future phase-fidelity-reviewer role (sibling of `intent-fidelity-reviewer`)
  reviewing delivered reality against the phase's `## Phase Acceptance` (see the
  amendment below — the original "against `## Expectation`" was imprecise; prose
  is not reviewable, structured acceptance is).
- A future implementation of `IL011` as a plan-time cross-phase bundle check
  (it can no longer be a frontmatter-field lint).

---

## Amendment — 2026-05-16: phase acceptance is structured, not prose

The decision above said the phase audit reviews delivered reality against the
phase's `## Expectation`, "the same Given-When-Then fidelity shape one grain up
from the intent audit." That claim was **aspirational, not true as written**:
`## Expectation` is prose (press-release voice). `intent-fidelity-reviewer`
compares reality against an intent's structured `## Acceptance Criteria`, *not*
against its prose press release. A phase audit needs the same — a structured
target — or it has nothing checkable to review against. This amendment closes
that gap. It completes adr-9; it does not reverse it.

### A phase doc carries two parts, mirroring an intent

An intent has a prose press release **followed by** a structured
`## Acceptance Criteria` block. A phase doc takes the same shape, one grain up:

- **`## Expectation`** — prose, working-backwards, the product thinker's
  re-statement of what the phase makes true. The reflection point. Unchanged.
- **`## Phase Acceptance`** — **new, required.** Given/When/Then bullets, the
  same format as intent acceptance. This is the audit target: what the
  phase-fidelity-reviewer compares delivered reality against. The product
  thinker authors it — it is the same press-release-voice skill they already
  use for intents.

`## Milestone` stays as written: the engineering done-cut ("ahoy install +
uninstall + doctor pass on a fresh repo"). `## Phase Acceptance` is the
*user-truth* cut. The two answer different questions — "is the work finished?"
vs. "is the expectation met?" — and both belong.

### The roll-up rule — phase acceptance asserts emergent truth, never copies

This is the load-bearing constraint. A `## Phase Acceptance` bullet MUST be a
**roll-up**, not a re-litigation of intent-level acceptance. It must assert
either:

- **(a) an emergent, cross-intent truth no single intent owns** — e.g. "the
  oracle cascade is whole end-to-end: RP → Codex → in-session" is true of
  Phase 1 but owned by no one intent in it; or
- **(b) a user-journey that spans the phase's intents** — a path a user can
  walk that only exists because several intents shipped together.

It MUST NOT copy a bullet that already lives in an intent's `## Acceptance
Criteria`. Copying duplicates, and duplication drifts — the exact failure the
brief's single-source-of-truth rule and the surface-drift guards exist to
prevent. The phase artefact earns its place precisely by holding the truths no
individual intent can hold.

### Consequences of this amendment

- The four existing phase docs gain a `## Phase Acceptance` section (added in
  the same change as this amendment).
- The phase-doc section list in `roadmap/phases/README.md` is updated to
  include `## Phase Acceptance` between `## Milestone` and `## Scope`.
- The mermaid diagram in `brief/01-product/03-mental-model.md` had the phase
  audit arrow pointing at `Brief`; it is re-pointed so the audit target is the
  phase, agreeing with this amendment.
- The deferred phase-fidelity-reviewer now has a concrete spec to point at:
  it reviews reality against `## Phase Acceptance`, per-bullet, in the
  `intent-fidelity-reviewer` verdict shape.
- A future lint (`IL` or a new prefix) can enforce the roll-up rule — flag a
  `## Phase Acceptance` bullet that duplicates an intent-level bullet. Deferred
  with the reviewer.

---

## Amendment — the `phase:` spec anchor is deferred, not a standing convention

The decision above said "**Specs gain a `phase:` frontmatter anchor**" and that a
spec "now also names the phase it belongs to." Read as a live convention, that
left the corpus in a **half-implemented state**: four specs carried the anchor,
~37 did not, no tooling read it, and its lint was already deferred (Consequences:
"deferred until the phase-audit reviewer is built"). A half-implemented
convention is the worst state — every spec without the anchor reads as drift
against a rule nothing enforces.

This amendment **retires the anchor as a standing convention** and re-scopes it
as **deferred, bundled with the phase-audit tooling that reads it**: the anchor,
its valid-phase lint, and the phase-fidelity-reviewer land together, in one
later spec, or not at all. Until then:

- **Phase membership is editorial**, reconstructed from each phase doc's
  `## Scope` — the live mechanism today. This is unchanged from the original
  decision (the intent→phase mapping was always editorial); the amendment only
  makes the *spec*→phase mapping editorial too, for now.
- **No spec is expected to carry `phase:`.** Its absence is not drift. Specs are
  neither backfilled (rejected: 37 frontmatter edits is destructive-adjacent and
  risks the task-JSON `updated_at` leak class) nor required to add it going
  forward.
- When the phase-audit tooling is specced, it owns reintroducing the anchor —
  with backfill and lint as part of *that* spec's scope, so the convention goes
  from inert to fully-implemented in one move, never half.

This completes adr-9's deferral posture (the lint was already deferred); it does
not reverse the phase layer itself. The editorial `## Scope` mapping, the phase
docs, `## Phase Acceptance`, and the phase audit's *target* all stand. Only the
per-spec frontmatter *anchor* is deferred. Brief and roadmap docs that described
the anchor as live ("authored going forward") were aligned to this posture in the
same change (`roadmap/phases/README.md`, `brief/01-product/03-mental-model.md`,
`brief/04-surfaces/04-launch.md`).

**Delivery note (spc-66, predecessor store).** Two of the three bundled
artefacts are specified there: the `PA001` valid-phase anchor lint
(verify-exists, line-precise on the `phase:` key) and the phase-audit reviewer
(a sibling of `intent-fidelity-reviewer`; specified there, not yet in the Go
binary — reviewing delivered reality against a phase's `## Phase Acceptance` via
the editorial `## Scope` membership chain, receipt-only). The **third** artefact
— the corpus anchor backfill onto the unanchored specs — remains **deferred**,
gated on the reviewer and lint reaching the Go binary: a future spec can then do
the editorial spec→phase mapping with the verification machinery in place.
spc-66 specifies what makes the backfill *safe*; it deliberately does not
perform it (a spec in no `## Scope` stays correctly unscheduled and carries no
anchor).
