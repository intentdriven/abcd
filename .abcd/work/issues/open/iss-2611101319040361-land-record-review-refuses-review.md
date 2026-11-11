---
schema_version: 1
id: "iss-2611101319040361"
slug: "land-record-review-refuses-review"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/review.go"
remedy: "Have abcd hand out the review brief with the exact return shape it reads (head line, verdict line naming the head), count only headings as sections, and list every fault of a return in one refusal; a test records a return with two faults and finds both named."
---

land --record-review refuses review returns in forms its own brief allows, counts a bold bullet label that begins with Verdict as a second Verdict section, and names one missing part per refusal, so each fault costs another reviewer round trip. On 8 Nov at 03:39 a verdict line that opened with APPROVE and named the head was refused as not one a validator writes; on 9 Nov at 18:18 a SHIP return was refused because a bold bullet label began with the word Verdict; on 9 Nov at 20:04 round-two returns were refused twice, first for a missing Reviewed head line and then for a verdict line that did not name the head, because hand-written fix-round briefs had dropped the return shape. iss-2611040712384537 fixed one such case on 4 Nov.
