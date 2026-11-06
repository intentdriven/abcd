---
schema_version: 1
id: "iss-2611060810026397"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Correct the passage to say what the code does."
---

`.abcd/development/brief/04-surfaces/17-guard.md:1044`: The chapter lists same-line carried values as unseen and calls the matter an open ruling, but they block and the ruling is made. All block when the value is assigned on the line (`X='rm -rf /'; eval "$X"`, `P=pkill; $P make`). The ruling is made: internal/core/guard/carried.go cites it, and iss-2609281134544802 is in .abcd/work/issues/resolved/.
