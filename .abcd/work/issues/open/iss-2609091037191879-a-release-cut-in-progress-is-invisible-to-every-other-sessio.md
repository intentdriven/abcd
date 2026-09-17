---
schema_version: 1
id: "iss-2609091037191879"
slug: "a-release-cut-in-progress-is-invisible-to-every-other-sessio"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
---

A release cut in progress is invisible to every other session, so a change landing on main during a cut is discovered in the changelog rather than chosen. abcd should let launch declare a cut lease that peer sessions, merges and record closes can see: a session about to merge or close a record during a lease is told which cut it would join and who is cutting, and the cutting session rules on the ordering. The convention is in AGENTS.md (concurrent sessions); this seeds the mechanical form, alongside session presence (iss-2608220750029993) and itd-33.
