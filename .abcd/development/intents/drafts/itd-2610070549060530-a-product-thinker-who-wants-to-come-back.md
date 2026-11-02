---
id: itd-2610070549060530
slug: a-product-thinker-who-wants-to-come-back
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A word in a new idea marks it to revisit, and the board puts it first

## Press Release

> A product thinker who wants to come back to an idea marks it to revisit: a word such as urgent at the start of the text, or one option, when they file it, or at any time later. From then on abcd's board shows it in a short list of ideas to revisit, above everything else, and it is built before ideas that carry no mark. It leaves the list when it is built or when they take the mark off; an idea never marked is shown and built as before.

_Worded by the facilitator from the product thinker's request of 2026-10-07 ("all it takes is to add a word urgent, important, etc. to the PT's initial intent text … ranked higher and/or added to a list of 'should be revisited'"); revised the same day from the product thinker's answers to the six open questions below; the revised wording is not yet confirmed._

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [traceability-governance SOTA note](../../research/notes/2026-10-23-traceability-governance-sota.md). The six rulings below stand; the research fits them.

- **A marker at the start of the text is an established convention, read only there.** org-mode's priority cookie sits in the headline "right after the TODO keyword" (`*** TODO [#A] ...`); Todoist reads `p1`, `p2` or `p3` typed into the task name and sets the priority when the task is added. Ruling 1 matches.
- **One level is a shipped design, rendered as a short list.** Jira's flag "has only one value: Impediment", shows the card in yellow and takes a comment on adding or removing it; GitHub pins "up to three important issues" "above the issues list". Ruling 2 matches; the cap of three is the field's one guard against a revisit list becoming a second backlog.
- **Most trackers later grow levels** (org-mode A/B/C, Todoist four), and in each the level "affects only sorting" (org-mode). Noted as the direction of travel, not a recommendation.
- **The mark's place in an order is a precedence rule, not a weight.** Todoist puts high-priority tasks "near the top of the list, and below tasks with a date and time". The analogue for ruling 3: a hard tier ahead of every unmarked idea, named as the placing rule in the pick's computed reason; where the product thinker's own order (itd-2610070550516046) is present it governs, and the mark places an idea within or ahead of it as the one ordering function states (itd-78's open question on precedence).
- **When a mark ends is contested.** Linear's snooze returns an issue "at a time of your choosing, or when there's new activity on that issue: whichever comes first"; Shape Up argues against any automatic return ("brought back with a context, by a person, with a purpose"). Ruling 4 (built or removed) is Shape Up's side.
- **Path 2**: native, a one-bit `revisit` mark set by a small fixed set of leading words (`urgent:`, `important:`, `revisit:`) or the flag, settable and removable later with an optional one-line reason, as Jira allows, a short list at the top of the board, a hard tier in the pick (ahead of every unmarked candidate; within or ahead of the product thinker's order where one is present, as the one ordering function states), and a soft cap in GitHub's spirit (the board says when more than three are marked and does nothing else). The seam is the one ordering function shared with itd-78 and itd-2610070550516046; a leading word outside the fixed set marks nothing and the filing says so.

The independent fit-challenge runs at plan time.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

Answered by the product thinker on 2026-10-07, in reply to the six open questions this draft was filed with ("1. marker and/or --urgent; 2. one level (revisit / not revisit); 3. revisit list, but also prioritise for building; 4. when build or marker removed; 5. abcd:abcd for now, dashboard later; 6. can be marked later (incl. in dashboard)").

1. **How the mark is given.** A marker word at the start of the filing text (such as `urgent:`), and/or a flag on the filing command (`--urgent`). A word elsewhere in the text does not mark the idea, so "this is important because…" marks nothing.
2. **Levels.** One: an idea is marked to revisit or it is not. "Urgent", "important" and their kin are the same mark.
3. **Effect.** Both: the idea is listed among the ideas to revisit, and it is raised in what is picked to build. The second changes the pick of itd-2609211116005482, which today scores only facts the intent's record holds; the mark is the product thinker's recorded fact, and the spec must say where in the pick's order it sits (ahead of every unmarked idea, or as a weight).
4. **When the mark ends.** When the idea is built, or when the product thinker removes the mark. Assumed by the facilitator: "built" means shipped, not merely started; to be confirmed at planning.
5. **Where it shows.** On the board (`/abcd:abcd`, bare `abcd`) now; on the dashboard (itd-2610032150577708) later.
6. **Marking later.** An idea already filed can be marked or unmarked at any time, including from the dashboard once it carries the list.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
