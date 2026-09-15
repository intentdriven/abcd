---
id: adr-5
slug: brief-is-current-state
status: accepted
date: 2026-05-08
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-1, adr-3, adr-9, adr-35]
---

# ADR-5: Brief is the current state; no version label, no archive directory

## Context

The brief had accumulated three contradictory framings of its own lifecycle:

1. **Brief = stable, one-time bootstrap.** `00-meta.md` framed the brief as "stable conventions...one-time bootstrap document for the v1 plugin"; future work goes through intents.
2. **Brief = living, gets new versions on restructure.** The same `00-meta.md` then defined a versioning scheme: "v3", "v5", "future iterations become `archive/05/`, `archive/06/`". The brief was on its fifth iteration, with `archive/01/`, `archive/02/`, `archive/03/`, `archive/04/` preserved as frozen historical snapshots.
3. **Brief = stable in shape, mutable in some sections.** In practice, every shipped spec with a new term mutates `02-constraints/04-naming.md` (vocabulary register, hard-enforced via VR001 lint). So the brief was never actually stable — it grew monotonically with every spec.

`brief/01-product/04-scope.md` already said: "The brief does not get re-versioned for v2. v2 is defined by which intents have shipped." `roadmap/README.md` reinforced: "the intent registry remains the canonical 'what abcd does' record post-v1." But the archive policy and "v5" framing pulled the other way.

Embark surface (`04-surfaces/03-embark.md`) clarified the contract: embark seeds a target repo's *v1 brief* from a lifeboat's amended press release; subsequent ahoy/work iterates on it like any other abcd-managed project's brief. The brief is per-project, evolving, *never re-versioned* in normal operation.

The version-label scheme was duplicating `git log brief/`. The `archive/<NN>/` directories were storing what `git log` already knows. The "v5 audit landed 40 fixes" changelog inside `brief/README.md` was redoing commit messages.

## Decision

**The brief is the current state of the project.** One canonical version. Always reflects today.

- **No version label** on the brief. No "v5", no "fifth iteration", no `version:` field.
- **No `archive/` directory.** Historical brief content is recovered via `git log brief/` and `git show <commit>:brief/...`.
- **No changelog blobs inside `brief/README.md`.** Pure delta narration is git's job. Inflection-point rationale (real architectural shifts) lives in ADRs (this directory). File-scoped rationale (why this section reads this way) stays inline in the brief.
- **History snapshots, when needed, come from disembark.** `voyage/disembark/history.jsonl` (per adr-35) is the audit chain.
- **`/abcd:embark` seeds a new repo's v1 brief** from a lifeboat's amended press release. This is the *only* time a "v1 brief" framing is meaningful — at the moment a new project starts.

Generalises ADR-3 (directory-as-truth-for-lifecycle) to the brief itself: the live `brief/` directory IS the brief. No parallel field, no parallel archive, no parallel iteration counter.

## Alternatives Considered

1. **Keep version labels + `archive/<NN>/` directories.** Rejected: duplicates `git log`. Every iteration's "What changed since vN" section inside `README.md` is recomputable from `git log brief/`. Storage cost is real (archive 03 alone is 1527 lines verbatim).
2. **Version the brief but drop the `archive/` directories** (use git tags instead — `brief-v3`, `brief-v5`). Rejected: tags help when the brief has discrete release cuts, but the brief evolves continuously per shipped spec (vocabulary register grows on every ship). A version label that ticks every commit is a worse version label.
3. **Split the brief into "stable canvas" and "living register".** Rejected (initially considered): would require renaming half the brief's files and confusing readers. The current single-namespace brief works once we admit it's *all* living state — every section evolves, just at different rates. ADRs catch the inflections; git catches the deltas; brief catches the present.
4. **Move all historical changelog content to ADRs.** Partially adopted: ADR-1 through ADR-4 capture genuine inflections that were buried in `brief/README.md` changelog blobs. Pure delta narration ("v5 backfilled acceptance criteria for 18 intents") is *not* moved to ADRs — it's deleted, since git covers it.

## Consequences

**Gains:**
- One brief, always current. New readers don't have to ask "is this the latest?"
- `brief/00-meta.md` shrinks dramatically — loses archive policy, version-deltas section, "fifth iteration" framing. Keeps numbered folders, padding, kebab-case, brief↔lifeboat shape contract.
- `brief/README.md` shrinks — loses ~120 lines of changelog blobs ("What changed since v3", "v5 continuation"). Keeps navigation + reading guide.
- Disembark and embark contracts (per adr-35) become simpler: they read the brief as-is, no version negotiation.
- `.abcd/development/decisions/adrs/` becomes the canonical home for inflection-point rationale across the project.

**Costs / obligations:**
- The `archive/01/`, `archive/02/`, `archive/03/`, `archive/04/` directories must be removed. Pre-removal, content audited: any rationale not already preserved (in current brief files, in ADR-1..4, or in `git log`) gets extracted to a new ADR or inline brief section before deletion.
- `brief/00-meta.md` rewritten without archive policy.
- `brief/README.md` purged of changelog blobs.
- `.abcd/development/roadmap/rfcs/README.md` updated — its "abcd doesn't currently use ADRs" claim no longer holds.
- New reserved vocabulary (registered in `02-constraints/04-naming.md`): "decision class" (ADR / RFC / intent), and the principle that `archive/` is not used inside `brief/`.

**Downstream consequences:**
- Future intent-driven work that previously would have triggered "v6 changelog" entries instead lands as commits + (when inflection-shaped) a new ADR.
- The "What is the brief's purpose post-v1?" question now has a single answer: the brief IS the project's current state, always. There is no post-v1.

**Refined 2026-09-01** (decision log, that date; sequenced as [Phase 8](../../roadmap/phases/phase-8-brief-currency.md)): brief, record and release stay in sync, with one legitimate lead — a brief edited after a release was cut is ahead of that release until the next cut.
