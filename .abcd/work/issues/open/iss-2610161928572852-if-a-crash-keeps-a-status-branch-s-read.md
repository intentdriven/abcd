---
schema_version: 1
id: "iss-2610161928572852"
slug: "if-a-crash-keeps-a-status-branch-s-read"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "messaging step 3 fix round 4 (feat/announce-step3, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Write each branch's read mark only after its seen files, and on reading a mark check that the commit just below it has a seen file that parses, walking from the last one that does."
---

If a crash keeps a status branch's read mark but loses one of the seen files written beside it (they are written without a flush), the commit that file stood for is never read again, because the next walk starts past the mark.
