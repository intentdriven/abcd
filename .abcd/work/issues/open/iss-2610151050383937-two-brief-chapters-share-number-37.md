---
schema_version: 1
id: "iss-2610151050383937"
slug: "two-brief-chapters-share-number-37"
severity: "minor"
category: "inconsistency"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Renumber one of 37-embargo.md and 37-reserve.md (the later to land takes the next free number), update its README row, the release-gate manifest's briefDocs and every link to it, and have the brief-register test refuse two chapters with one number."
---

Two brief surface chapters both carry the number 37: 04-surfaces/37-embargo.md and 04-surfaces/37-reserve.md, each minted on its own branch, and the chapter table in 04-surfaces/README.md lists two rows numbered 37. Nothing refused it, because each branch numbered its chapter after the highest on its own base.
