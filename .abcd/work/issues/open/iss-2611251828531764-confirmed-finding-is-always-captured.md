---
schema_version: 1
id: "iss-2611251828531764"
slug: "confirmed-finding-is-always-captured"
severity: "major"
category: "process"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Build itd-2611251831138387, whose spec is spc-2611252115253843: the list of confirmed findings, the block a sub-agent ends with, the hold at a stop and the refusal at a landing and at the end of a run. This issue resolves with its close."
related_intents: [itd-2611251831138387]
---

Nothing in abcd holds a session or a sub-agent to the rule that a confirmed finding is captured, whether or not it can be fixed. The product thinker stated it on 2026-11-25 as a basic rule with no exceptions, and that abcd must enforce it: meet an issue, test it, capture it once confirmed, never let it slip or put it off. The deferral discipline (itd-2610141956319695) covers findings a review confirms, and the safe-fix discipline (itd-2610271130017437) a finding fixed where it is found; a finding met while doing other work, or returned in the prose of a sub-agent, is in no check. Met the same day: one session kept several confirmed or confirmable findings in a local log, some called "known" on the strength of a resolved record, and filed them only when told to. The ISSUES rules say to capture and ask for nothing at a stop.
