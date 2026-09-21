---
id: itd-2609212103568351
slug: the-bare-abcd-status-board-and-the-site-s-status-page-show
spec_id: spc-2609212138241908
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609211913453478, itd-2609211116005482, itd-2609201916151817]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-100, itd-2609061543533170]
related_adrs: [adr-2609212115255771]
---

# The status board shows Now, Next and Later, computed from the record

## Press Release

> **`abcd` and the site's Status page show Now, Next and Later, computed from the record and maintained by nobody.**
>
> "The phase documents told me what someone once thought would happen next; this tells me what is next," said a product thinker reading the board after retiring phases. "Now is what is in a lane and what the run would pick; Next is what is ready; Later is the rest. Nobody edits it, so it cannot lie."

## Why This Matters

Phases and milestones were retired on 2026-09-21 (adr-2609212115255771) on the evidence that a hand-maintained sequencing document drifts to editorial membership nobody anchors to. The field's replacement for a dated roadmap is the Now / Next / Later view; abcd already holds its inputs in the lifecycle shelves, the readiness gate and the build's state file, so the view is a render, not a record, and it lands where a reader already looks: the bare `abcd` status board and the site.

## Mechanism

We expect a view computed from state to stay true where the hand-maintained phase documents drifted, because nothing on it can be edited into a lie; shown wrong if a reader finds Now naming an intent that is neither in a lane nor the pick order's head.

## Scope Conditions

None stated.

## What's In Scope

- **Now**: every intent whose lane the build's state file shows as started, with its lane state, plus the head of `build next`'s pick order marked "next up", so Now is never empty while anything is READY.
- **Next**: every planned intent the readiness gate reports READY, in pick order.
- **Later**: planned intents not READY (with the failing check named), then drafts.
- **Two surfaces, one read**: the bare `abcd` board gains the block; the site's Status page renders it from the same function; `--json` carries the three lists.
- **Nothing stored**: no folder, no field; removing the state file empties Now's lane rows and leaves the head.
- **No roadmap**: the word appears on neither surface; the glossary points `roadmap` at this block.

## What's Out of Scope

- A started state as a stored field (it is rendered from the state file).
- The pick itself (itd-2609211116005482).
- The site's other pages (itd-2609061543533170).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Now / Next / Later is rendered status, never stored (adr-2609212115255771, decision 2).
2. When nothing is being built, Now shows the pick order's head marked "next up".
3. The word roadmap is retired; this block is called status.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** an abcd-managed repository, **when** `abcd` runs bare, **then** the board carries a Now / Next / Later block: Now lists the intents the build's state file shows in a lane, with their lane state, and the pick order's head marked "next up"; Next the planned intents the gate reports READY in pick order; Later the planned intents not READY with the failing check named, then the drafts; each row its id and title.
- **Given** the site is built, **when** its Status page is read, **then** it renders the same block from the same read.
- **Given** the build's state file is absent, **when** the board renders, **then** Now holds only the pick order's head, and nothing else on the board changes.
- **Given** `--json`, **when** the board renders, **then** the payload carries the three lists with ids, titles, lane states and the failing checks.
- **Given** either surface, **when** it is searched for the word roadmap, **then** it is absent, and the glossary's `roadmap` entry names this block as its successor.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: phases are retired today and the record needs a place a person looks to see what is next; we expect the computed block to be read where the phase documents were not; shown wrong if Now is found naming an intent neither in a lane nor next
