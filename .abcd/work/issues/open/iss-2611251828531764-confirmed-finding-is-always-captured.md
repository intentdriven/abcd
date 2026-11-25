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
remedy: "Enforce it at the places abcd already stands. State the rule in the ISSUES domain and in what a sub-agent is told at its start. Have a sub-agent return its findings in a fixed block, each with its state: fixed in a named commit, refuted with the test, captured under a named record, or held in the private store. The stop hooks keep a per-session list of what is in none of the four and hold the stop once, naming each; landing, a run ending, and a handover refuse while the list is not empty. A finding untested is listed as owed a test, never dropped. Grounds: the deferral discipline has these states already, and the release receipts enforce them."
---

Nothing in abcd holds a session or a sub-agent to the rule that a confirmed finding is captured, whether or not it can be fixed. The product thinker stated it on 2026-11-25 as a basic rule with no exceptions, and that abcd must enforce it: meet an issue, test it, capture it once confirmed, never let it slip or put it off. The deferral discipline (itd-2610141956319695) covers findings a review confirms, and the safe-fix discipline (itd-2610271130017437) a finding fixed where it is found; a finding met while doing other work, or returned in the prose of a sub-agent, is in no check. Met the same day: one session kept several confirmed or confirmable findings in a local log, some called "known" on the strength of a resolved record, and filed them only when told to. The ISSUES rules say to capture and ask for nothing at a stop.
