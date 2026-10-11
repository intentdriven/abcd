---
id: itd-2609291923559186
slug: the-consistency-pass-runs-where-records
spec_id: spc-2610021456200324
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-48, itd-34]
related_issues: [iss-2609260926323349]
related_intents: [itd-62]
related_adrs: [adr-2609300821558671]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# The consistency pass runs where records are written and names each finding the same way every time

Typed links: `builds_on` [itd-48](../shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md) (the consistency pass these five follow-ups extend, and the intent they were routed to) and [itd-34](../shipped/itd-34-three-intent-kinds.md) (the kinds lint the scheduling piece sits beside); `related_issues` [iss-2609260926323349](../../../work/issues/open/iss-2609260926323349-itd-48-shipped-the-consistency-pass-role.md) (the tracking record of the five, which this draft carries once planned); `related_intents` [itd-62](../drafts/itd-62-pluggable-safety-gate.md) (the managed pre-commit gates of ruling J23, beside which A2's hook sits); `related_adrs` [adr-2609300821558671](../../decisions/adrs/2609300821558671-a-reading-finding-is-matched-against-the.md) (the filing-time match G1's key is distinct from).

## Press Release

> **The consistency pass runs where records are written and names each finding the same way every time.** The pass that finds two records contradicting each other, `abcd intent consistency`, gains five things together: cross-document lint codes for the contradictions a rule can see without a model, so record-lint names them on every change; a pre-commit hook that runs the rule checks on every commit and the model pass on the records a commit touches wherever the owner has a model route, and that warns, stops, or fixes the obvious and files the rest, as the repository chose at setup; the pass run beside the kinds lint, so a shape finding and a contradiction are reported together; a corpus too large for one review measured, reported and read in chunks instead of skipped; and a stable key for each finding, so a contradiction found on Monday is the same finding on Friday and a person answers it once.
>
> "I used to rediscover the same contradiction every time I ran the pass and could never tell whether it was new," said Iris, a product thinker. "Now the obvious ones fail the commit that introduces them, the subtle ones come back under the same id, and the big record is read in pieces instead of being passed over."

## Why This Matters

itd-48 shipped the consistency pass (Role 2 of the intent auditor) and none of
the five follow-ups routed to it from the predecessor store's spc-33 (A1, A2, A3, A4, G1; itd-48, Routed Deferrals); since then
iss-2609260926323349 has been their only home. Without them the pass is a
report a person remembers to run: a contradiction a rule could catch waits for
the next model run, and nothing runs it when a record changes. A finding is
already linked rather than filed twice when its open record is found
(itd-48's second criterion, and the filing-time duplicate check every route
shares), but a report row carries only its number, so nothing says a row is
the same contradiction as last week's, and a contradiction whose record was
resolved comes back as new. The product thinker ruled on 2026-09-29
(ruling BT2) that the five are planned together as one intent rather than
one at a time or closed.

## Mechanism

We expect the pass to catch contradictions earlier and to be answered once
because the cheap cases move into a gate that runs on every change and the
expensive cases keep one identity across runs; shown wrong if, after it
ships, the same contradiction is still filed twice across two runs, or a
contradiction a lint code covers still reaches the default branch.

## What's In Scope

- **A1, cross-document lint codes**: the mechanical half of Role 2 as
  record-lint codes that need no model, for the contradictions
  `record_schema` does not already refuse: a `builds_on` or `routed_from`
  edge with no reciprocal mention, and a `related_issues` entry the issue
  does not link back. A cross-reference naming no record and one-way supersession are
  already `record_schema` findings and stay there. The new codes are warnings, so a record
  that passes record-lint today keeps passing.
- **A2, a pre-commit hook for the pass.** The A1 codes run on the whole
  corpus on every commit, with no model. The model pass runs on the staged
  records plus the records they link to or that link to them, where the
  owner's personal settings provide a model route; where none does, the
  commit proceeds and the status carries "deep check owed" naming the
  records. The hook never refuses a commit on a judgement it could not
  obtain. On a finding it does what the repository chose at setup (the
  `ahoy` setup interview asks): **warn** (print the finding by its key and
  proceed), **stop** (refuse the commit naming the finding), or
  **automatic** (fix a mechanical contradiction, such as a missing
  back-link, in the same commit with a note of the change, and file a
  judgement contradiction as an issue carrying its key, then proceed). The
  hook is a gate in the core-owned managed pre-commit gates (iss-84 with
  itd-62, ruling J23), and its build waits for them.
- **A3, one run beside the kinds lint** (itd-34): the hook's single
  invocation runs record-lint's kind rules, the A1 codes and, where it
  runs, the model pass, and reports them together, with no model call when
  no route is configured.
- **A4, chunked review of a corpus too large for one request.** The pass
  measures the assembled corpus against the request limit and reports an
  overflow with its size; on an overflow it reviews the corpus in chunks,
  one after another, and the report covers every chunk.
- **G1, a stable key for each finding**: a deterministic key on every report
  row, derived from the class, both end paths and the collapsed quotes,
  written into the record a finding files, so the same contradiction carries
  the same key on every run; distinct from the filing-time duplicate check,
  which matches prose. A finding whose record is already resolved or won't
  fix is marked `addressed` on the report, naming the record, and files
  nothing. G1 lands first: the hook and the chunked review name findings by
  it.

## What's Out of Scope

- Role 1 (the per-intent fidelity audit) and Role 3 beyond its scheduling.
- Any change to the judgement itself: the classes and the rubric stay
  itd-48's.

## Scope Conditions

- Holds for a repository whose record carries a brief and intents the pass <!-- cond: cond-2610021456201670 -->
  can read; a repository with neither has nothing to compare.
- The hook's model pass holds only where the owner's personal settings <!-- cond: cond-2610021456205269 -->
  provide a model route; elsewhere the pass is marked owed, never assumed
  run.

## Acceptance Criteria

- **Given** two records whose frontmatter states contradict in a way a rule
  can see (a one-way `builds_on` or a
  `related_issues` entry with no back-link), **when** record-lint runs,
  **then** a cross-document code names the rule and both files, with no
  model run.
- **Given** a repository that chose **stop**, **when** a commit introduces a
  contradiction, **then** the commit is refused naming the finding by its
  key; **given** **warn**, **then** the finding is printed by its key and the
  commit proceeds.
- **Given** a repository that chose **automatic**, **when** a commit
  introduces a missing back-link and a judgement contradiction, **then** the
  back-link is added in the same commit with a note naming the change, an
  issue carrying the second finding's key is filed, and the commit proceeds.
- **Given** the `ahoy` setup interview, **when** it runs in a repository,
  **then** it asks which of warn, stop or automatic the hook uses, and
  records the answer.
- **Given** no model route in the owner's personal settings, **when** the
  hook runs, **then** the A1 codes run, no model is called, the commit is
  not refused for the model pass, and the status names the records whose
  deep check is owed.
- **Given** a route is configured and a commit stages one record, **when**
  the hook runs, **then** the model pass reviews that record and the
  records linked to or from it, and the A1 codes cover the whole corpus.
- **Given** a commit the kind rules and the pass both judge, **when** the
  hook runs, **then** both report in one run.
- **Given** a corpus larger than one request, **when** the pass runs,
  **then** the report states the overflow and its size, the corpus is
  reviewed in chunks, and the report covers every chunk; **given** a corpus
  that fits, **then** no overflow is reported and nothing is chunked.
- **Given** the same contradiction found on two runs, **when** the second
  report is ingested, **then** the row carries the same key as the first and
  the key is written in the filed record.
- **Given** a finding whose record is resolved or won't fix, **when** it is
  found again, **then** the report marks it `addressed` naming the record and
  files nothing.

## Decisions

Ruled at this draft's planning interview, 2026-10-02 (the product thinker and technical facilitator, one question at a time):

1. **A3 stays in this intent** (iss-2609260926323349's body routed it to itd-34, which is shipped; its deferral and ruling BT2 route it here).
2. **A4 stays, and builds its own trigger**: the pass measures the corpus and reports an overflow before it chunks.
3. **The hook runs the rule checks always and the model pass only where a route exists**, marking it owed otherwise; it never refuses a commit on a judgement it could not obtain.
4. **What the hook does on a finding is asked at setup**: warn, stop, or automatic (fix the obvious, file the rest).
5. **Scope:** the rule checks cover the whole corpus; the model pass covers the staged records and their links.
6. **The hook waits for the core-owned managed pre-commit gates** (iss-84 with itd-62, ruling J23); A1, A4 and G1 do not.
7. **G1 lands first.**
8. **Impact is additive**: the A1 codes start as warnings.

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: contradictions get into the records unnoticed and are answered twice, and unattended runs write records with no one checking them for contradictions; wrong if, after it ships, the same contradiction is still filed twice, a rule-catchable one still gets in, or an unattended run still lands a contradiction the check covers.
