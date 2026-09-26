---
schema_version: 1
id: "iss-2609260100396332"
slug: "the-badge-and-the-mode-notice-name-the-facilitator-role-in"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/badge.go"
---

The badge and the mode notice name the facilitator role in two different words: the status-line badge hard-codes 'waiting on the technical facilitator' while mode.State.Addressee(), which claims to be the one vocabulary seam for both, returns 'facilitator', so abcd mode facilitator prints 'waiting on the facilitator' where the badge reads 'waiting on the technical facilitator'.
