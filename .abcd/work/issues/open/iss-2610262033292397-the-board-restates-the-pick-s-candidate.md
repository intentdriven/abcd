---
schema_version: 1
id: "iss-2610262033292397"
slug: "the-board-restates-the-pick-s-candidate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 411 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md:349"
remedy: "none (filed automatically)"
finding_ref: cfk-e69b1431dcdec54a
---

The board restates the pick's candidate rule to choose its next-up head, but leaves out the pick's footprint exclusion.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/08-abcd.md:349` — "so the head is always the intent the pick would choose"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:172` — "less one whose spec's `## Footprint` names no packages to reserve it with (the check `footprint`)"

08-abcd chooses the status block's head from the readiness gate, the build's record-only start checks, the lanes and the peers check, and concludes that the head is always the intent the pick would choose. 34-build's pick also excludes, in a checkout with an `origin` remote, any planned intent whose spec's Footprint names no packages, and the planning verb mints every spec with that section empty. Both pages cover how the head is chosen, and they disagree. An intent with an empty footprint is shown as next up on the board while `abcd build next` passes over it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 411.
