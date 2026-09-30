---
id: adr-2609291342092738
slug: a-drain-takes-an-issue-alone-only-when-its-fields-say-it
status: accepted
date: 2026-09-29
supersedes: null
superseded_by: null
related_intents: [itd-82, itd-2609201916151817]
related_rfcs: []
related_adrs: [adr-25, adr-27]
drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]
drain_severities: [nitpick, minor]
drain_security: handback
drain_remedy: required
---

# ADR-2609291342092738: A drain takes an issue alone only when its fields say it needs no decision

## Context

`abcd drain` ([itd-82](../../intents/planned/itd-82-drain-ledger-triage.md))
works the open issue ledger unattended: it fixes what needs no decision and
hands the rest back to a person. The load-bearing part is the sorting, which
issue a machine may take alone. Its failure runs one way: a machine that
decides a thing needs no decision, and then makes one.

The evidence gathered on 2026-09-21 is recorded in the intent: models detect
their own ambiguity badly, triage passes that dispatched a coding agent were
reverted after unwanted pull requests, and clean-ups and tests merge far more
often than features and performance work. The product thinker ruled on
2026-09-21 (itd-82 decision 4) that the rule for "needs no decision" decides
what a machine may touch alone, so it is a recorded rule: a decision record and
a brief invariant, minted by the spec's first delivery, and until it exists an
unattended drain refuses to start. Decisions 6 and 7 of the same interview
settle the `remedy:` field and the order.

## Decision

We will let a drain take an open issue alone only when its fields say it needs
no decision, and route every other open issue by the rule that excluded it.

1. **The rule is the fields.** An open issue is eligible when all of these
   hold: no record in its `blocked_by` is still open; its category is in the
   fixable set, `bug`, `documentation`, `drift`, `inconsistency`, `tech-debt`
   or `ux`; its severity is `nitpick` or `minor`; and it carries a non-blank
   `remedy:` (a record carrying only the older `suggested_fix:` reads that
   value as its remedy).
2. **What each exclusion means.** The rules are asked in this order and the
   first that excludes decides the disposition: blocked (skipped, the blocker
   named); category `security` (always a person's); a category outside the
   fixable set, which is `process`, `observation`, `architectural-insight`,
   `future-work-seed` and `lapse` (handed back, the category named); severity
   `major` or `critical` (handed back, the severity named); no remedy
   (ineligible, the field named, until someone adds one). A record the ledger
   reader refuses is named as unreadable. Every open issue receives exactly one
   disposition.
3. **A judgement can only narrow.** A host-delegated judgement
   ([adr-25](0025-host-delegated-llm-default.md)) over an eligible issue's
   remedy may hand it back as a user-visible or trust-boundary change. It
   never makes an excluded issue eligible, and no estimate an agent makes of
   its own confidence lets an issue through.
4. **The order is declared.** Eligible issues are taken by category in the
   order `tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux`,
   then `nitpick` before `minor`, then oldest first.
5. **The classification is re-derived every run** and recorded with each
   disposition in the run's summary; nothing is written onto the issue for it
   (itd-82 decision 8).
6. **The drain refuses to start without this record.** The rule is read from
   the drained repository's own decision store, never from the binary: the
   four `drain_` fields in this record's frontmatter state it
   (`drain_categories`, `drain_severities`, `drain_security`,
   `drain_remedy`), and a repository whose store holds no accepted record
   carrying them is refused by the dry run and the start alike, naming how to
   add one (the product thinker's ruling BX2 of 2026-09-29, verbatim: "the
   PROJECT MUST HOLD the eligibility decision in its own record (e.g. added
   at setup); drain refuses there until it does"). This repository states
   abcd's strict baseline, which the binary bundles as the measure a
   loosening is named against, and a test holds this record to it.
7. **Another project may loosen the floors, loudly.** A project's own record
   may list `major` or `critical` among its severities, or set
   `drain_security: take`, and every floor it loosens is named by the dry run
   and at the start (ruling H11 of 2026-09-29, verbatim: "MAY LOOSEN abcd's
   floors (a project may let drain take major/critical and security issues).
   NOTE for the lane: make a loosened floor loud (drain --dry-run and the
   drain start name every floor the project loosened), and keep abcd's own
   repository at the stricter default."). It may narrow the fixable set but
   never widen it, and it cannot drop the remedy. A partial or malformed
   record refuses rather than falling back to either rule.
8. **A record waiting on a person is a person's.** Whatever the record says,
   an issue whose remedy opens "Waits on" (a fix that waits on an unanswered
   ruling) and an issue whose deferral past the current anchor tag is live are
   handed back, each naming its rule.

## Alternatives Considered

- **A model classifies each issue.** Rejected: ambiguity self-detection is
  unreliable, and a classifier that can let an issue through is the failure
  this rule exists to prevent. The judgement stays, and it can only hand back.
- **Severity alone.** Rejected: severity says how much an issue matters, not
  whether fixing it needs a decision. A `minor` `process` issue is a rule
  change, and a `minor` `security` issue is a trust question.
- **Everything with a remedy.** Rejected: a remedy says what someone proposed,
  not that the proposal needs no decision. The category and severity carry
  that.
- **The fields, read in a fixed order (chosen).** Every field is on the
  capture already, apart from `remedy:`, and each exclusion names the rule a
  person reads to act on it.

## Consequences

- The first drains over today's ledger are small: most records carry no
  `remedy:`, and they wait until someone adds one.
- A change to the fixable set, the severities or the order is a change to this
  record, never merely a code path.
- The issue-keyed lane of `itd-2609201916151817` (decision 10) reads the same
  rule as the check before it starts, so the two cannot disagree about which
  issue is eligible.
- The eligibility record is a file the drained repository authors, deciding
  what an unattended agent may do there. A contributor's pull request that
  loosens it is a trust change: it is committed history, reviewed like code,
  and a loosened floor is named on every dry run and start.
- Owed and out of this record: where a hand-back flag lives.
