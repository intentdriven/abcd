---
schema_version: 1
id: "iss-2609261056373310"
slug: "scribe-assemble-reads-the-ledger-as-it-stands-in-the-working"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/assemble.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (run A 2026-09-29, lane drainDrift3; the 2026-09-26 DECISIONS entry leaves it undecided; rulings-owed BG): scribe assemble reads the ledger as it stands in the working tree, uncommitted records included, while itd-2609020625402599's scope condition cond-2609020626046719 says committed ledger content. Should the condition be amended to the working tree, or should the assembler read committed content only and refuse or report uncommitted records?"
---

scribe assemble reads the ledger as it stands in the working tree, uncommitted records included, while itd-2609020625402599's scope condition (cond-2609020626046719) says the context is assembled from committed ledger content; which one moves is a ruling owed: amend the condition to the working tree, or have the assembler read committed content (and refuse or report uncommitted records)
