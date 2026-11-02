---
id: itd-2610070550516046
slug: on-abcd-s-dashboard-a-product-thinker
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# On the dashboard, a product thinker sees the brief and their ideas together, and puts the ideas in order

## Press Release

> On abcd's dashboard a product thinker keeps an overview of the two things they own: the brief and the ideas. Each section of the brief sits beside the ideas that realise it, showing which are built, planned or still drafts, so a section nothing realises yet stands out at a glance. From the same page they review any idea and put the ideas in the order they want them built.

_Worded by the facilitator from the product thinker's request of 2026-10-07 ("a way for the PT to review and/or prioritise intents. Those are the two things the PT owns: brief and intents; the dashboard should also have a clear data visualisation to relate brief elements and intents … which intents are already planned, which ones are not"); filed as one goal on the product thinker's answer, wording not yet confirmed._

Proposed links, for the product thinker to confirm at planning: builds on the dashboard (itd-2610032150577708, planned) and on the section-to-intent mapping of itd-2610040817105016 (draft), which this page draws; the revisit mark of itd-2610070549060530 (draft) refines this intent's ordering, as one narrow way to raise an idea.

Extension proposed on 2026-11-01, for the product thinker to confirm at planning: a **goal view**. The product thinker picks an outcome from those abcd offers, drawn from the brief's unrealised sections (a new direction is a brief edit first, the product thinker's ruling of 2026-11-01), and the same page then shows only the brief sections, ideas and open issues that serve it, with what is still missing shown as proposed new ideas to choose from. Nothing about the goal is stored: the view is computed from the brief and the ideas each time, as the Now/Next/Later block is (adr-2609292012006845).

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

Answered by the product thinker on 2026-10-07 ("1. PT order decides, but can be 'resetted'/reverted; 2. suggest changes to it (e.g. notes, etc. as discussed in dashboard intent); 3. depends on abcdesign.app -- if a session can 'pull' it straight away, the anser is meediate; if not, the PT will have to export and import into a session; 4. which picture?; 6. yes").

1. **The product thinker's order decides what is built next**, and can be reset, returning to the scored pick. This reverses the design of itd-2609211116005482 (planned), whose pick scores only facts each intent's record holds; that intent must be reconciled with this one before either is built, and the revisit mark of itd-2610070549060530 is one way into the same order.
2. **Reviewing an idea is suggesting changes to it**: notes and marks, as the dashboard already provides for the brief (itd-2610032150577708), brought to a session for the technical facilitator to propose and the product thinker to confirm.
3. **When a reorder takes effect depends on whether a session can pull it.** If a session can pull the change from the dashboard straight away, it takes effect at once; if not, the product thinker exports it and imports it into a session. The answer names abcdesign.app, the page for editing a public project's brief filed on 2026-10-05 (itd-2610052000411162 and itd-2610052000424067, drafts); how a session pulls from it is settled with those intents.
4. **Ownership.** Read by the facilitator as the answer to question 5 (the answer is numbered 6): "the product thinker owns two things, the brief and the intents" goes into the glossary entry for the product thinker.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [traceability-governance SOTA note](../../research/notes/2026-10-23-traceability-governance-sota.md). The four rulings below stand; the mockups owed under question 1 remain the product thinker's pick.

- **Coverage views in the requirements tools are a filtered table plus an on-demand graph with a cap.** sphinx-needs' `needtable` filters and sorts needs with their link columns; `needflow` draws needs and links with a `max_items` cap, and a truncated view "says so in the page and emits a `needs.max_items` warning". A network graph is a filtered neighbourhood, never everything; this agrees with the facilitator's pass of 2026-10-07 recorded above.
- **The coverage report's first row is the uncovered set.** Spec Kit's `analyze` reports "requirements with no tasks" and a coverage percentage; a section nothing realises is the "at a glance" item.
- **A position per item is how tools built for concurrent editing store manual order.** Figma's fractional indexing ("Every object has a real number as an index"; insert at the average of the neighbours; concurrent inserts can interleave) says so explicitly; Jira's LexoRank (a lexicographic string field with a background balancer and collation failure modes) and GitLab's manual sort ("Each issue is assigned a relative order value"; "everyone who visits the same list sees the updated issue order") store the same shape without stating the reason. With one writer and a few hundred ideas, one ordered list of ids is enough and "reset" (ruling 1) is deleting it.
- **Two modes, shown.** GitHub Projects lets a row be dragged when no sort is applied and is silent on what a later sort does to that order; the honest design says which mode the view is in: the product thinker's order, or the computed pick.
- **The frame is already rendered:** Now/Next/Later is abcd's computed status (adr-2609212115255771), and its inventors' rule is that the columns carry confidence, not dates.
- **Path 2**: native, a coverage strip and section list from itd-2610040817105016's trace index, the ordered ideas beside it, a filtered neighbourhood graph on demand; the product thinker's order as one ordered list of ids kept where the dashboard's notes already live, read by the one ordering function (shared with itd-78 and itd-2610070549060530) as its highest-precedence input when present (ruling 1), with the revisit mark placing an idea within or ahead of it; the precedence between the two is stated once, in the answer to itd-78's open question, not in each spec. The seam is that function; the storage moves to a rank per record only if a second editor appears. No new dependency.

The independent fit-challenge runs at plan time.

## Open Questions

1. **The picture.** The product thinker asked which picture is meant: the "clear data visualisation to relate brief elements and intents" of their request. Put to them again on 2026-10-07 as a grid, a tree or a map, they answered: "Conduct SOTA and create mockups for each first; incl. a network graph, too." Open until a state-of-the-art pass and a mockup of each of the four forms (grid, tree, map, network graph) are in front of them; whether shipped ideas still show is asked with the mockups. Both were prepared by the facilitator on 2026-10-07 in the local tier (`scratch/reports/sota-brief-intent-visualisation-2026-10-07.md`, `scratch/mockups/brief-ideas-views.html`, built from the record with each chapter linked to the ideas that cite it): the research recommends a hybrid, a coverage strip of one tile per section above the section list, with the ordered list beside it, and rules out a treemap or sunburst sized by count and a force-directed network of everything. The pick is the product thinker's.
2. (SOTA 2026-10-23) Where the product thinker's order lives so it survives: a linked worktree's local tier dies with the worktree, and the order is theirs, not a worktree's.
3. (SOTA 2026-10-23) How the view shows which mode it is in (their order, or the computed pick after a reset), given that GitHub Projects leaves the question unanswered and the two are easy to confuse.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
