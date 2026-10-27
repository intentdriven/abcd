---
schema_version: 1
id: "iss-2610262037376558"
slug: "the-cloud-session-intent-admits"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 736 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610120738075547-a-person-hands-an-item-from-their.md:129"
remedy: "none (filed automatically)"
related_intents: [itd-2610120738075547]
finding_ref: cfk-8e2a30d209681b6d
---

The cloud-session intent admits work-in-progress branches without a receipt as an entry in the reservation build's list; the reserve brief admits receipt-less pushes only when every commit has the empty tree

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610120738075547-a-person-hands-an-item-from-their.md:129` — "the push hook admits such a branch without a preflight receipt as one entry of its declared list of receipt-less prefixes (the list the reservation build introduces)"
- End B: `.abcd/development/brief/04-surfaces/37-reserve.md:111` — "The committed pre-push hook lets a push under the prefix through without a preflight receipt only when every commit it moves has the empty tree."

The cloud intent lets a cloud session's work-in-progress branch past the push hook without a preflight receipt by adding it to the receipt-less prefix list that the reservation build introduced. That branch carries the session's actual changes. The reserve chapter says what that admission requires: a push under the prefix passes without a receipt only when every commit it moves has the empty tree. A branch of real changes fails that condition, so the dependency as written cannot admit it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 736.
