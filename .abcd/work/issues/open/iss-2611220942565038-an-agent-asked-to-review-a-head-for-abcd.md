---
schema_version: 1
id: "iss-2611220942565038"
slug: "an-agent-asked-to-review-a-head-for-abcd"
severity: "nitpick"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing"
remedy: "Add abcd land --review-brief <head> that prints the return shape a reviewer must write for that head, ready to paste into the reviewer's prompt, from the same parser --record-review uses."
---

An agent asked to review a head for abcd land --record-review has to be told the return's exact shape (the first line naming the head, one Verdict heading, the verdict line naming the head in full) in every prompt, and a shape mistake is found only when the receipt is refused.
