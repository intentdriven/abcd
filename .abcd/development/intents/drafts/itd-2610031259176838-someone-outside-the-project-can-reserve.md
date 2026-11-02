---
id: itd-2610031259176838
slug: someone-outside-the-project-can-reserve
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Outside contributors reserve work through a draft pull request that abcd honours

## Press Release

> Someone outside the project can reserve a piece of work too. They cannot push the claim branch abcd lists, which the project's own people use, so their agent opens a draft pull request named after the item instead, and everyone sees it on the website. abcd treats that draft as taken, so no agent of yours starts the same item, and a check refuses a second draft for the same item. When the draft is merged or closed, the item is free again.

## Why This Matters

On 2026-10-03, with a second person joining the project and the repository public, the product thinker chose a claim branch abcd lists, on the shared remote, as the way the project's own people reserve work (itd-2609150819440345, decision 4, narrowed to a branch by itd-2609091034175565 decision 10). People outside the project cannot place that marker, so without this their agents would start items the project's agents already hold, the duplicate work iss-2609020716570699 records. The product thinker asked for this follow-up the same day, marked not urgent.

State of the art (2026-10-03, [`2026-10-03-multi-agent-coordination-sota-and-run-lessons.md`](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md)): forge-native claiming, such as the Rust project's bot that assigns an issue on a "claim" comment, or a draft pull request keyed to the item with a check refusing a second; projects taking agent contributions increasingly accept them only for issues accepted beforehand. Text an outside contributor writes (a title, a description) is untrusted input, read by its shape and never followed as instructions.

Typed links: Builds on itd-2609091034175565 (reserving an item with a claim branch abcd lists; its decision 10); the wider register is itd-2609150819440345; refines iss-2609020716570699 (duplicate work and duplicate pull requests).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

1. 2026-10-03, the product thinker: filed as a follow-up to the claim branch abcd lists (itd-2609091034175565, decision 10), not urgent.

## SOTA

Surveyed on 2026-10-03 in the [multi-agent coordination note](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md)
(its item 6 is this intent's case, S3: a writer without push access); the
forge-side mechanics were re-checked on 2026-10-23.

- **A claim is a compare-and-swap with a lease, re-checked at land**
  (Kubernetes Lease fields; Kleppmann on fencing tokens). A draft pull request
  is the outside writer's version of the claim branch. It carries the same
  four fields (record id, holder, footprint, expiry) once the expiry is set by
  rule, because the forge sets none.
- **The forge-native precedent is a bot turning a comment into an
  assignment.** On Rust's repositories `@rustbot claim` assigns the issue to
  the commenter. A user without write access cannot be assigned, so the bot
  assigns itself and edits the issue's top comment to say who claimed it.
  "Only the current assignee or a team member can release an assignment"
  ([triagebot issue assignment](https://forge.rust-lang.org/triagebot/issue-assignment.html)).
  This is the comment-plus-bot option in the open questions. It needs a
  running bot, where a draft pull request needs only a CI check.
- **Neither precedent expires on its own.** The triagebot page states no
  inactivity release. The forge's stale action defaults to 60 idle days before
  marking and 7 more before closing. It can set pull requests apart
  (`days-before-pr-stale`, `days-before-pr-close`), and it exempts drafts only
  when told to (`exempt-draft-pr`, default `false`)
  ([actions/stale](https://github.com/actions/stale)). An outside reservation's
  expiry is a number this project chooses, and the stale action is a ready
  enforcer of it.
- **Overlapping footprints are common.** Xu et al. measured a 41.7% conflict
  rate between different agents' concurrent pull requests (2026-10-03 note), so
  the draft's duplicate check should key on the record id and also compare the
  draft's changed paths with live reservations.
- **A draft's title and body are untrusted input** (the 2026 incidents in the
  2026-10-03 note). The check reads only the record id from a fixed position,
  never free text.

**Path 2**: native, the draft pull request mapped onto the existing claim
record (id from the branch name or title prefix, holder from the pull
request's author, expiry by rule), with the duplicate check as one CI job, the
stale action as the expiry enforcer, and a comment bot as a later backend
behind the same record. No dependency. The independent fit-challenge runs at
plan time.

## Open Questions

- Whether a draft pull request, a claim comment that a bot turns into an assignment, or both reserve an item.
- Whether outside contributors may reserve any item, or only items the project has marked as open to them.
- How long an outside reservation lasts with no activity before the item is free again.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
