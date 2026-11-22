---
schema_version: 1
id: "iss-2611221728043111"
slug: "abcd-build-of-an-issue-that-this"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "In ledgerNote give the level checkout the same instruction: when origin holds the record neither open nor finished and this checkout holds it open, say the record has not reached origin and to land it there first, whether or not the checkout is behind."
---

abcd build of an issue that this checkout holds open and origin holds neither open nor finished says to land the record on origin first only when the checkout is also behind origin; a checkout level with origin gets the disagreement with no instruction
