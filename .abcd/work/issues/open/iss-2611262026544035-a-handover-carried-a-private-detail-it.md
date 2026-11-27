---
schema_version: 1
id: "iss-2611262026544035"
slug: "a-handover-carried-a-private-detail-it"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "away-mode handover before a compaction"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Add a rule to the SCRATCH domain (internal/core/rules/defaults/rules.json): a handover, and anything else written for a later session, holds only what that session needs to do the work; it never records where the person keeps their own files, their personal details, or anything else of theirs the work does not use."
---

A handover carried a private detail it did not need. On 2026-11-26 a session wrote, into its handover file (.abcd/.work.local/NEXT.md) and so into the summary a compaction carries forward, where the person had copied some private files on their own computer. Nothing in the later work needed it, and the person asked why it was recorded at all. No rule says what a handover may hold: the PII domain keeps absolute paths out of what leaves the machine, and a handover never leaves it, so nothing applied.
