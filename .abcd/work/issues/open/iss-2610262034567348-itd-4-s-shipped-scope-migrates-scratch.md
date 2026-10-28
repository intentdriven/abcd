---
schema_version: 1
id: "iss-2610262034567348"
slug: "itd-4-s-shipped-scope-migrates-scratch"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 535 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-4-issue-capture.md:46"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-4fd3b29ba3ef5db8
---

itd-4's shipped scope migrates scratch issues through dev-sync; the capture surface page says that migration is not built and rides a dev-sync surface that is only a design target.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-4-issue-capture.md:46` — "`dev-sync` promotes existing `.abcd/.work.local/issues.md` entries to the structured ledger on first run after install"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:521` — "**Not built yet.** The migration rides the `dev-sync work` surface"

itd-4 places in scope a migration path in which `dev-sync` promotes the entries of .abcd/.work.local/issues.md into the ledger on the first run after install. 06-capture.md says the legacy scratch migration is not built, that it rides the `dev-sync work` surface, and that this surface is itself a design target. A shipped intent's scope rests on a verb that, as the surface page states it, does not exist, so that part of the intent cannot have been delivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 535.
