---
schema_version: 1
id: "iss-2609291803334904"
slug: "the-status-block-s-next-up-head-can-name-a-ready-intent-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusblock/statusblock.go"
remedy: "split in two: the four record-only checks (open questions, the claim sections, blocked_by, the spec's steps) are a plain defect, fixed by lifting them into one shared statement (intent.StartChecksIn) that build next and statusblock.Read both call; the peers check alone needs a peers read on every bare board, which waits on ruling CC1 (the board pays the peers read, or it skips the check and says so)"
resolution: "The four record-only pre-start checks (open questions, the claim sections, blocked_by, the spec's steps) are lifted into intent.StartChecksIn, which build next's checks and the status block's head both call, so the head never names an intent build next excludes for them. The residual is the peers check, which the head does not run: whether the bare board pays a peers read is ruling CC1, owed to the technical facilitator, and the surface chapter and command page state that the head does not consult other checkouts."
impact: fix
resolved_by:
  commit: "fc4644eef"
---

The status block's next-up head can name a READY intent abcd build next would not pick. The head is read in the pick order and passes over a held intent and one in a lane, but build next's candidates also pass every other pre-start check (open questions, the claim sections, blocked_by, the spec's steps, the peers), which the block does not run: a READY intent with an open question or an unshipped blocker, and the highest score, is marked next up on the board and the site while build next excludes it. statusblock cannot import the loop (loop imports peers, peers capture, capture site), so the fix hands the pick's candidate set in from the front doors the way the lane reader is, at the cost of a peers read on every bare board.

The remedy is split. Of build next's extra checks, open questions, the claim sections, blocked_by and the spec's steps are record-only reads and need no peers read: they are fixed by one shared statement of the record-only checks both front doors call, so the head excludes exactly what the pick excludes for them. Only the peers check needs a peers read, and whether the bare board pays one is ruling CC1, owed to the technical facilitator; until it is ruled the block's documentation states that the head does not consult other checkouts.

## Grounds

- pursued: we expect the head to equal build next's pick whenever no peer holds anything; shown wrong if a READY intent with an open question, an unanswered claim section, an unshipped blocker or no step left is marked next up
