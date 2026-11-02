---
id: itd-2610111655362931
slug: on-abcd-s-dashboard-the-product-thinker
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610091918433290, itd-2610032150577708]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# On the dashboard, the product thinker and facilitator review memory notes and promote the ones worth keeping

## Press Release

> On abcd's dashboard, the product thinker and the technical facilitator see every memory note agents have written since they last looked, with who wrote it, when and in which session, and keep, discard or promote each one with a click. A promoted note joins the project's record, so every agent and teammate gets it. Only a person promotes, and in a repository with more than two maintainers another maintainer agrees before a promotion lands.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

- **Given** notes written since the last review, **when** the person opens the review, **then** they see every one of those notes with who wrote it, when and in which session, and they keep, discard or promote each.
- **Given** a note the person promotes, **when** the promotion lands, **then** the note joins the project's record, so every agent and teammate gets it; only a person promotes, and in a repository with more than two maintainers at least one other maintainer must agree before it lands.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [memory-knowledge SOTA note](../../research/notes/2026-10-23-memory-knowledge-sota.md).

- **CODEOWNERS counts path owners, not maintainers.** Owners "must have write permissions", the last matching pattern wins, and owners are requested only for the paths they own (GitHub docs); one person can own every path in a four-maintainer repository, so option 1(a) does not measure the thing.
- **The forge already implements "another maintainer agrees".** A ruleset can require a number of approving reviews, code-owner review, and "an approval from someone other than the last person to push" (GitHub rulesets docs); abcd's own mirror `.abcd/work/rulesets/main-review.json` carries `required_approving_review_count: 0` with `require_code_owner_review: true`, so the mirror shows whether the forge requires a second approval at all. That count is a merge-policy threshold, not a count of people (a four-maintainer repository can require one approval, a solo one none), so option 1(d) fails the same test as CODEOWNERS; in this repository it is zero.
- **The maintainer count is one authenticated call, for organisation repositories.** `GET /repos/{owner}/{repo}/collaborators` returns each person's `role_name`, filtered on each row's permissions or `role_name` (the page documents `permission` as a single filter value, not a combined one), "the highest role assigned ... after considering all sources of grants" (GitHub REST docs); the Maintain role is defined for organisation-owned repositories, and the roles page does not cover user-owned ones.
- **Harness memory review is edit-after-the-fact, and enforcement belongs in a hook.** Claude Code's auto memory has no approval gate (the person edits files from `/memory`), and its docs say memory files are "context, not enforced configuration. To block an action ... use a PreToolUse hook instead". The guard entry and terminal check the earlier spec drafted for the person's-act rule are the right shape.
- **One editor had an approve-before-save pop-up for generated memories and reportedly withdrew it** (Cursor Memories; search snippets only, unverified).
- **Path 1**, with the two jobs split: the forge's required approval on the pull request that carries the promotion is the agreement (2a), and the mirror shows only whether one is required at all; the count of people comes from the collaborators endpoint filtered to maintain or admin (push, for a personal repository), called once at the person's click (1c), with a declared list only for a repository with no forge (1b). A ruleset requiring zero approvals, as abcd's own does, means the more-than-two rule is enforced by nothing, which is open question 3. abcd's own part is refusing the promotion to an agent and stamping who promoted. The independent fit-challenge runs at plan time.

## Open Questions

1. **Where does abcd learn how many maintainers a repository has?** (technical facilitator) The more-than-two rule needs the count, and abcd records none.
   - (a) The distinct owners in the CODEOWNERS file the forge reads.
   - (b) A `maintainers` list in `.abcd/config.json`, which `ahoy install` asks for and a person edits.
   - (c) The forge's own list of people with maintain or admin rights, asked at promotion (a network call).
   - (d) The required approving-review count in the branch ruleset mirrored under `.abcd/work/rulesets/`.
2. **How is another maintainer's agreement recorded and checked?** (technical facilitator)
   - (a) As the forge's approval of the pull request that carries the promotion, required by the repository's ruleset; abcd checks nothing itself.
   - (b) As an `agreed_by` stamp the other maintainer adds by running `abcd memory agree <mem-id>` in their own checkout, which a `memory_promotion_agreed` record-lint rule requires before the change lands.
   - (c) Both.
3. (SOTA 2026-10-23) **What does a promotion say in a repository whose ruleset requires no approving review?** The more-than-two rule is then enforced by nothing; the options are to refuse the promotion until the ruleset requires one, to land it and say in the note's stamp that no second maintainer was required, or to fall back to the `agreed_by` stamp of option 2(b) for that repository only.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-11, product thinker: reviewing memory notes and promoting them "will be part of a later PT / TF dashboard where the PT can click on those they want to promote", and then "Move them to the dashboard". This draft takes the review and promotion criteria, verbatim, from the memory notes intent (itd-2610091918433290), where they were its second and third criteria, accepted on 2026-10-11, and the two open questions on the maintainer count and the agreement from the memory-recall spec (spc-2610111627213138), where they were its fourth and fifth. It builds on the memory notes intent, whose store it reviews, and on the dashboard intent (itd-2610032150577708), where the review happens.
- 2026-10-11, carried from the same intent: only a person promotes, and above two maintainers another must agree (product thinker, recorded in adr-2610091918443997). Before the move, the memory-recall spec had designed the person's-act rule as a guard entry refusing an agent's `abcd memory promote` and a terminal check, after the source ledger's flip (adr-41), and the agreement as a record-lint rule over an `agreed_by` stamp; that design is a starting point for this intent's plan, not a ruling.
