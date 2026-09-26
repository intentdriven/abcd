---
schema_version: 1
id: "iss-2609261737325810"
slug: "review-of-integ4-reported-that-abcd-capture-defer-prints"
severity: "nitpick"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-integ4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

Review of integ4 reported that abcd capture defer prints 'ledger of <absolute cwd>' on stdout unscrubbed. Reproduced: the line is captureLedgerRoot's checkout banner (iss-2609202053570475), on stderr, printed by every capture verb and by the record dispatcher; --json carries the same identity as its ledger member. It is home-redacted through fsutil.RedactHome, so under the real HOME it reads ~/..., and it is absolute only for a checkout outside HOME, which is how the review's temporary-HOME scratch run saw it.
