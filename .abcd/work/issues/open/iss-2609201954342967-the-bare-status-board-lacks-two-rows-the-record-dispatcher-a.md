---
schema_version: 1
id: "iss-2609201954342967"
slug: "the-bare-status-board-lacks-two-rows-the-record-dispatcher-a"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "overtaken-intent review with the product thinker, 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

The bare status board lacks two rows the record dispatcher already answers for a single record: suggested next actions for the repository, and the planned intents with their spec and whether it is in flight. Ruled by the product thinker on 2026-09-20 while superseding itd-20 (the Python-era board built on [redacted-user]-sync and the logbook) by itd-121: those two pieces of itd-20 are still wanted and are captured here so the supersession loses nothing. Wanted: (1) the board ends with a short next-actions list derived the way abcd <record-id> derives one record's next move (an owed fidelity audit, an unplanned draft with all decisions recorded, a stale claim); (2) the board lists each planned intent with its spec id and an in-flight marker where the spec is open and its branch exists. Both are read-only rows on the existing board; no new verb.
