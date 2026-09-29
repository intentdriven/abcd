---
id: itd-2609291923559186
slug: the-consistency-pass-runs-where-records-are-written-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-48, itd-34]
related_issues: [iss-2609260926323349]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The consistency pass runs where records are written and names each finding the same way every time

Typed links: `builds_on` [itd-48](../shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md) (the consistency pass these five follow-ups extend, and the intent they were routed to) and [itd-34](../shipped/itd-34-three-intent-kinds.md) (the kinds lint the scheduling piece sits beside); `related_issues` [iss-2609260926323349](../../../work/issues/open/iss-2609260926323349-itd-48-shipped-the-consistency-pass-role-2-and-nothing.md) (the tracking record of the five, which this draft carries once planned).

## Press Release

> **The consistency pass runs where records are written and names each finding the same way every time.** The pass that finds two records contradicting each other, `abcd intent consistency`, gains five things together: cross-document lint codes for the contradictions a rule can see without a model, so record-lint names them on every change; a pre-commit hook that runs the pass on the records a commit touches, advisory or blocking as the repository chooses; the pass scheduled beside the kinds lint, so a shape finding and a contradiction are reported together; a corpus too large for one review read in chunks, one at a time, instead of skipped; and a stable key for each finding, so a contradiction found on Monday is the same finding on Friday and a person answers it once.
>
> "I used to rediscover the same contradiction every time I ran the pass and could never tell whether it was new," said Iris, a product thinker. "Now the obvious ones fail the commit that introduces them, the subtle ones come back under the same id, and the big record is read in pieces instead of being passed over."

## Why This Matters

itd-48 shipped the consistency pass (Role 2 of the intent auditor) and none of
the five follow-ups routed to it from spc-33 (A1, A2, A3, A4, G1); since then
iss-2609260926323349 has been their only home. Without them the pass is a
report a person remembers to run: a contradiction a rule could catch waits for
the next model run, nothing runs it when a record changes, and a finding
re-hashes differently on each run, so the same contradiction reads as new
every time and is answered twice. The product thinker ruled on 2026-09-29
(ruling BT2) that the five are planned together as one intent rather than
one at a time or closed.

## Mechanism

We expect the pass to catch contradictions earlier and to be answered once
because the cheap cases move into a gate that runs on every change and the
expensive cases keep one identity across runs; shown wrong if, after it
ships, the same contradiction is still filed twice across two runs, or a
contradiction a lint code covers still reaches the default branch.

## What's In Scope

- **A1, cross-document lint codes**: the mechanical half of Role 2 (schema
  and state contradictions, reference rot, acknowledgement gaps) as
  record-lint codes that need no model.
- **A2, a pre-commit hook for the pass**, with the repository's choice of
  advisory or blocking.
- **A3, scheduling beside the kinds lint** (itd-34): the pass's pre-commit
  mode and its per-commit cost policy.
- **A4, chunked review of a corpus too large for one request**, dormant
  until a report shows `bundle_overflow: true`.
- **G1, a stable key for each finding** that converges across re-runs, with
  an `addressed` state on the report.

## What's Out of Scope

- Role 1 (the per-intent fidelity audit) and Role 3 beyond its scheduling.
- Any change to the judgement itself: the classes and the rubric stay
  itd-48's.

## Scope Conditions

- Holds for a repository whose record carries a brief and intents the pass
  can read; a repository with neither has nothing to compare.
- Holds where the host can run the pass headless for the hook; where it
  cannot, the hook is advisory and says so.

## Acceptance Criteria

- **Given** two records whose frontmatter states contradict in a way a rule
  can see, **when** record-lint runs, **then** a cross-document code names
  both files, with no model run.
- **Given** a repository that chose the blocking hook, **when** a commit
  touches a record the pass finds contradicted, **then** the commit is
  refused naming the finding; **given** the advisory choice, **then** the
  finding is printed and the commit proceeds.
- **Given** a commit the kinds lint and the pass both judge, **when** the
  pre-commit mode runs, **then** both report in one run, within the cost
  policy the repository set.
- **Given** a corpus larger than one request, **when** the pass runs,
  **then** it reviews the corpus in chunks and the report covers every
  chunk; **given** a corpus that fits, **then** nothing is chunked.
- **Given** the same contradiction found on two runs, **when** the second
  report is ingested, **then** the finding carries the same key and links
  the existing record instead of filing a second one.

## Open Questions

- **The decomposition's routing** (the itd-84 table, run by the lane that
  drafted this on 2026-09-29, verdict FILE-AS-IS as ruled): whether A3 stays
  here or joins the kinds lint's own record, and whether A4 stays dormant
  inside this intent or is split out until a report shows an overflow. The
  product thinker confirms at the planning interview.
- **Which piece lands first**: the stable key (G1) is the one every other
  piece's reports rely on.
- **The hook's default** in a managed repository: advisory or blocking.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
