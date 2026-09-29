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
remedy: "hand the pick's candidate set (loop.Candidates) to statusblock.Read from both front doors and take the head from it, or record that the head applies only the readiness gate, the hold and the lane"
---

The status block's next-up head can name a READY intent abcd build next would not pick. The head is read in the pick order and passes over a held intent and one in a lane, but build next's candidates also pass every other pre-start check (open questions, the claim sections, blocked_by, the spec's steps, the peers), which the block does not run: a READY intent with an open question or an unshipped blocker, and the highest score, is marked next up on the board and the site while build next excludes it. statusblock cannot import the loop (loop imports peers, peers capture, capture site), so the fix hands the pick's candidate set in from the front doors the way the lane reader is, at the cost of a peers read on every bare board.
