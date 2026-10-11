---
schema_version: 1
id: "iss-2609260100396332"
slug: "the-badge-and-the-mode-notice-name-the"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/badge.go"
resolution: "mode.State.WaitingOn composes the ruled phrase once (Addressee names the technical facilitator in full); the badge's role labels and the set form's notice both read it."
impact: fix
resolved_by:
  commit: "6a6ebe35"
---

The badge and the mode notice name the facilitator role in two different words: the status-line badge hard-codes 'waiting on the technical facilitator' while mode.State.Addressee(), which claims to be the one vocabulary seam for both, returns 'facilitator', so abcd mode facilitator prints 'waiting on the facilitator' where the badge reads 'waiting on the technical facilitator'.

## Grounds

- pursued: abcd mode facilitator's notice and the badge both read 'waiting on the technical facilitator'; shown wrong if TestBadgeAndNoticeNameTheSamePerson finds the notice's words absent from the rendered badge
