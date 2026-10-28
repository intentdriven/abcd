---
schema_version: 1
id: "iss-2610262044300840"
slug: "peers-means-records-held-by-sibling"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1211 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703, itd-2609091416295622]
finding_ref: cfk-76da725ce477f4fa
---

`peers` means records held by sibling worktrees in the shipped reader, but the planned announcements ruling reserves it for sessions running now

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "the product thinker ruled that `peers` keeps its name for its future meaning, the sessions running now, local first and later over a private network"
- End B: `.abcd/development/intents/shipped/itd-2609091416295622-a-session-sees-the-records-its-sibling.md:55` — "Presence or liveness: A peer holding a record says the record exists, not that a session does."

itd-2609091416295622 defines `peers` as a read of records that sibling worktrees and branches hold, and puts presence out of scope: a peer holding a record does not mean a session exists. itd-2609151838312703 rules that `peers` keeps its name for the sessions running now. One verb name is given two different concepts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1211.
