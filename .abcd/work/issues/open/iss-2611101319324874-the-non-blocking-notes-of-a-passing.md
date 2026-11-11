---
schema_version: 1
id: "iss-2611101319324874"
slug: "the-non-blocking-notes-of-a-passing"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Have the receipt file each non-blocking note of a passing round as a capture, or offer one fix round for them before land; a test passes a round carrying one note and finds it captured."
---

The non-blocking notes of a passing review reach neither a fix round nor a capture. Since 904e2a8f5 the loop records each reviewer's located findings and hands a failing reviewer's findings to the fix brief, but a SHIP verdict ends validation and land refuses a head moved past the reviewed one, so a cheap defect that a passing reviewer names rides only in its return file. On 8 Nov at 11:38 a coordinator captured such a note by hand from another worktree.
