---
schema_version: 1
id: "iss-2610150733180751"
slug: "embark-carries-neither-a-source"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "itd-2609151138388536 build review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat"
remedy: "Have embark carry .abcd/work/decisions/ and a frozen .abcd/work/DECISIONS.md the way it carries the other record stores, with a test that the graveyard re-derived after embark matches the source's."
---

embark carries neither a source repository's decision records nor its frozen decision log, so re-deriving the lifeboat graveyard after embarking can differ from the source for a repository that has either; seen while moving the lifeboat to the decisions folder (step 4 of the decisions build).
