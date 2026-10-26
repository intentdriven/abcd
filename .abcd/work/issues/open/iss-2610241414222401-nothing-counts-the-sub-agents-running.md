---
schema_version: 1
id: "iss-2610241414222401"
slug: "nothing-counts-the-sub-agents-running"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24 autonomous test run"
origin: researcher-authored
production_mode: hand-written
found_at: "sub-agents across concurrent sessions"
remedy: "Keep a machine-wide count of live sub-agents per session under ~/.abcd.noindex/ (written when an agent starts and stops, through the subagent hooks), with the person's cap in ~/.abcd.noindex/config.json, and have a session check it before starting an agent; abcd peers shows the count."
---

Nothing counts the sub-agents running across concurrent sessions on one machine. On 2026-10-24 the person set a combined cap (8, then 10, then 12), and the sessions kept within it only by messaging each other their counts by hand: about twenty cross-session messages, guesses at each other's numbers, and one session asking permission before each start. Any session that did not hear a message could exceed the cap unseen.
