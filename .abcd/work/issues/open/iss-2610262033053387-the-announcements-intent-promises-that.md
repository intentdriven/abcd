---
schema_version: 1
id: "iss-2610262033053387"
slug: "the-announcements-intent-promises-that"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 357 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:24"
remedy: "In the announce intent's press release and quote say a release hold stops every session's land step from arming a merge while the cut is live, and that a pull request already armed or queued lands on its own checks; criterion C7 is already worded that way."
related_intents: [itd-2609151838312703]
finding_ref: cfk-7edc531cd739ffc4
---

The announcements intent promises that nobody's session merges into a release being cut; the launch page says the release hold does not stop a merge armed or queued before the reservation.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:24` — "and when I cut a release, nobody's session merges into it."
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:770` — "The hold stops a merge from being armed; it does not stop a merge armed, or queued, before the release was reserved"

itd-2609151838312703 promises that while a release is cut, nobody's session merges into it, because every session's merge step waits on the release reservation. The launch page describes the same hold (rel-vX.Y.Z, read by the land step, citing this intent's C7 and C8) and says it only stops a merge being armed. A pull request with auto-merge already on, or already in the merge queue, lands when its checks pass. The runbook has to check for such merges by hand. The intent's guarantee is stronger than the mechanism the launch page says it delivers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 357.
