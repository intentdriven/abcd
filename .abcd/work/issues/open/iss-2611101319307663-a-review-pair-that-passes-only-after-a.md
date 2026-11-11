---
schema_version: 1
id: "iss-2611101319307663"
slug: "a-review-pair-that-passes-only-after-a"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/delta.go"
remedy: "Waits on a ruling first: then accept a bounded delta review receipt for a small code change, scoped to the files it changed, citing the prior pair's receipts and taken from the same reviewer roles."
---

A review pair that passes only after a fix round cannot rest on the earlier full pair when the fix changes code: --rests-on takes a wording delta only, so each fix round's small code change needs a fresh full pair and a fresh preflight, about 1.5 to 2 hours a round. On 7 Nov at 22:20 the second fix round of #1452 needed a full pair rather than a delta; on 9 Nov at 04:20 a head needed a third full pair; on 9 Nov two lanes went through four and three fix rounds this way. A person must choose how large a code delta may rest on a prior pair and what the delta reviewers must be shown.
