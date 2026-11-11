---
schema_version: 1
id: "iss-2611101316403474"
slug: "abcd-build-of-a-named-issue-applies-the"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/eligible.go"
remedy: "Waits on a ruling first: then give build an explicit opt-past for a named issue that sets aside the category and severity rules only, keeping the security, waits-on, decision-needed and deferral rules, and record in the run that it was used."
---

abcd build of a named issue applies the drain's eligibility rule in full, so a coordinator who names one issue to build is refused for its category or its severity. eligibility (internal/core/capture/eligible.go) hands back a category outside the fixable set and a severity above the drain's, and build's issue check uses that verdict unchanged. On 7 Nov at 14:25 build refused a major issue, at 14:40 a process-category issue whose remedy was a code change, on 8 Nov at 04:35 a major issue again, and on 9 Nov at 06:45 another process-category issue; each lane was reserved by hand outside the build. A person must choose whether a named build may pass the drain's category and severity bars, and whether that takes an explicit flag.
